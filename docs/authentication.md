# Authentication and Session Specification

## 1. Overview

Yily implements a self-contained, zero-external-dependency authentication and authorization system. It does not rely on third-party identity providers, OAuth, or external SMTP servers.

The system serves two distinct access models:
1. **Web Interface**: State-managed server-side sessions via secure, `HttpOnly` HTTP cookies.
2. **Programmatic / API Access**: Stateless Personal Access Tokens (PAT) transmitted via `Authorization: Bearer yily_<token>`.

Access is restricted to a single account: the **Super Admin**.

---

## 2. Technical Specifications

| Parameter | Specification | Rationale |
| :--- | :--- | :--- |
| **Password Hashing** | **Argon2id** (RFC 9106) | Memory-hard, side-channel resistant password hashing algorithm. |
| **Argon2id Cost** | `t=3, m=64MB, p=4, keyLen=32` | Recommended OWASP baseline balancing verification speed and brute-force resistance. |
| **Salt** | 16 bytes (128 bits) | Cryptographically random (`crypto/rand`), generated uniquely per password. |
| **Session Token** | 32 bytes (256 bits) Base64URL | Cryptographically random (`crypto/rand`). Plaintext never stored in database. |
| **Session Storage** | SHA-256 hex digest (64 chars) | Compromising the SQLite database does not yield usable session tokens. |
| **Session Lifetime** | 24-hour absolute TTL / 2-hour idle timeout | Prevents stale sessions; activity is refreshed upon each authenticated request. |
| **Session Cookie** | `yily_session`, `HttpOnly`, `SameSite=Lax`, `Secure` | Mitigates XSS token theft; detects HTTPS directly or via `X-Forwarded-Proto: https`. |
| **PAT Format** | `yily_<base64url_256bit_token>` | Scoped token prefix identifying Yily credentials. |
| **PAT Storage** | SHA-256 hex digest (64 chars) | Complete PAT is shown only once at creation time; only the SHA-256 hash is persisted. |
| **API Auth Header** | `Authorization: Bearer yily_<token>` | Standard RFC 6750 Bearer authentication scheme. |

---

## 3. Storage & Cryptographic Representations

### 3.1. Password PHC String
Passwords are saved in the `User.PasswordHash` column formatted according to the standard PHC string format:

```text
$argon2id$v=19$m=65536,t=3,p=4$<base64_salt>$<base64_hash>
```

### 3.2. Session & PAT SQLite Storage
Neither session tokens nor Personal Access Tokens are ever persisted in plaintext. Only their SHA-256 digests are stored:

```text
token_hash = hex(sha256(raw_token))
```

Both `model.User.PasswordHash`, `model.Session.TokenHash`, and `model.PersonalAccessToken.TokenHash` carry the `json:"-"` struct tag to prevent accidental serialization in API responses or logs.

---

## 4. Security Guarantees & Lifecycle

### 4.1. Timing Attack Mitigation (Email Enumeration)
Password verification with Argon2id takes approximately 50–100ms. If an invalid email returns immediately in <1ms while an existing email takes ~80ms, attackers could enumerate whether the Super Admin email exists.
* **Mitigation**: When `GetByEmail` returns `ErrUserNotFound`, `AuthService.Login` executes `crypto.VerifyPassword` against a lazily initialized dummy Argon2id hash. Every login attempt takes a uniform amount of time regardless of whether the email exists.

### 4.2. Session Fixation Prevention
During authentication, if an existing session cookie is presented in the request:
1. The prior session is immediately revoked on the server.
2. A brand-new session token is generated with a new cryptographic ID.
3. The new token is set in the response cookie.

### 4.3. HTTPS & Reverse Proxy Handling
Yily detects HTTPS connections through:
1. Direct TLS connection (`c.Request().TLS != nil`).
2. Reverse proxy header `X-Forwarded-Proto: https` (inspecting the leftmost protocol when chained).
3. Echo URI scheme (`c.Scheme() == "https"`).

When HTTPS is detected, cookies are marked `Secure=true`.

### 4.4. Personal Access Token (PAT) Lifecycle
* **Creation**: The Super Admin creates a PAT via `POST /api/v1/tokens`. The raw token (`yily_...`) is returned in the response payload **only once**.
* **Subsequent Queries**: `GET /api/v1/tokens` returns token metadata (`id`, `name`, `created_at`, `expires_at`, `revoked_at`) without ever exposing the raw token or token hash.
* **Revocation**: A token can be revoked at any time via `DELETE /api/v1/tokens/:id`. Revoked tokens are immediately rejected.

---

## 5. Unified Authentication Middleware

A single middleware in `internal/middleware` handles both access methods transparently:

```text
Incoming Request
      │
      ▼
Has Authorization: Bearer <token> ?
     ├── YES ──► Extract "yily_<token>"
     │                │
     │                ▼
     │           Compute SHA-256 hash
     │                │
     │                ▼
     │           Lookup PAT in SQLite
     │                │
     │                ├── Expired / Revoked / Not Found ──► 401 Unauthorized
     │                └── Valid ──► Attach User & PAT to Context ──► Next()
     │
     └── NO  ──► Extract "yily_session" Cookie
                      │
                      ▼
                 Compute SHA-256 hash
                      │
                      ▼
                 Lookup Session in SQLite
                      │
                      ├── Expired (TTL or Idle) / Revoked / Not Found ──► 401 Unauthorized
                      └── Valid ──► Update LastActivityAt
                                        │
                                        ▼
                                   Attach User & Session to Context ──► Next()
```

---

## 6. API Route Access Map

| Endpoint | Method | Authentication | Description |
| :--- | :--- | :--- | :--- |
| `/api/v1/auth/setup` | `POST` | Public | Creates initial Super Admin account (single-tenant invariant). |
| `/api/v1/auth/login` | `POST` | Public | Authenticates Super Admin and issues `yily_session` cookie. |
| `/api/v1/auth/logout` | `POST` | Session / PAT | Revokes session on server and expires cookie on browser. |
| `/api/v1/auth/me` | `GET` | Session / PAT | Returns authenticated Super Admin user profile. |
| `/api/v1/tokens` | `POST` | Session / PAT | Issues a new PAT (raw token displayed once). |
| `/api/v1/tokens` | `GET` | Session / PAT | Lists PAT metadata for the Super Admin. |
| `/api/v1/tokens/:id` | `DELETE` | Session / PAT | Revokes a PAT by ID. |
| `/api/v1/projects/*` | `*` | Session / PAT | Protected resource management. |
| `/api/v1/environments/*` | `*` | Session / PAT | Protected resource management. |
| `/api/v1/variables/*` | `*` | Session / PAT | Protected secret & version management. |

---

## 7. Package Architecture

```text
server/internal/
├── crypto/
│   ├── password.go       # Argon2id hashing & constant-time verification
│   └── token.go          # crypto/rand session & PAT generator, SHA-256 hasher
├── domain/model/
│   ├── user.go           # Super Admin model (Name, Email, PasswordHash)
│   ├── session.go        # Server-side Session model (TokenHash, Expirations)
│   └── pat.go            # PersonalAccessToken model (TokenHash, RevokedAt)
├── repository/
│   ├── user.go           # Single Super Admin creation & lookups
│   ├── session.go        # Session persistence, lookups & activity updates
│   └── pat.go            # PAT persistence & ID-based revocation
├── services/
│   ├── auth.go           # Setup, Login (anti-fixation & timing mitigation), Session validation, Logout
│   └── pat.go            # PAT generation, verification & revocation
├── handler/
│   ├── auth.go           # Setup, Login, Logout, Me endpoints
│   └── pat.go            # Token creation, listing & revocation endpoints
└── middleware/
    └── auth.go           # Unified RequireAuth middleware (Bearer PAT + Cookie fallback)
```

---

## 8. Out of Scope (Future Phases)

* Account recovery codes (one-time emergency recovery).
* Strong password reset flows.
* Rate limiting on consecutive failed login attempts (lockout threshold).
* Multi-user role-based access control (RBAC).
* CLI-specific local credential storage.

