# Authentication and Session Specification

## 1. Overview

Yily implements a self-contained, zero-external-dependency authentication and authorization system. It does not rely on third-party identity providers, OAuth, or external SMTP servers.

Access is restricted to a single account: the **Super Admin**. That account is reached through two access models:

1. **Web Interface**: server-side sessions carried by an `HttpOnly` cookie, with a per-session CSRF token for state-changing requests.
2. **Programmatic / API Access**: Personal Access Tokens (PAT) sent as `Authorization: Bearer yily_<token>`, each scoped to `read` or `write` and optionally restricted to a single project or environment.

Two out-of-band secrets complete the model:

* **Setup token** (`YILY_SETUP_TOKEN`): required once, to create the Super Admin on a fresh instance.
* **Recovery code**: issued at setup, used to reset a forgotten password. It is single-use and replaced on every reset.

---

## 2. Technical Specifications

| Parameter | Specification | Rationale |
| :--- | :--- | :--- |
| **Password Hashing** | **Argon2id** (RFC 9106) | Memory-hard, side-channel resistant password hashing algorithm. |
| **Argon2id Cost** | `t=3, m=64MiB, p=4, keyLen=32` | OWASP baseline balancing verification speed and brute-force resistance. |
| **Salt** | 16 bytes (128 bits) | Cryptographically random, generated uniquely per hash. |
| **Password Policy** | 12–128 characters, with at least one uppercase letter, one lowercase letter, one digit and one special character | Enforced on setup and recovery. |
| **Session Token** | 32 bytes (256 bits), Base64URL | Cryptographically random. Plaintext never stored in database. |
| **Session Storage** | SHA-256 hex digest (64 chars) | Compromising the database does not yield usable session tokens. |
| **Session Lifetime** | 24-hour absolute TTL / 2-hour idle timeout | Prevents stale sessions; activity is refreshed upon each authenticated request. |
| **Session Cookie** | `yily_session`, `HttpOnly`, `SameSite=Strict`, `Secure` on HTTPS, `Path=/` | Not readable by scripts and never sent on cross-site requests, including top-level navigations. |
| **CSRF Token** | 32 bytes (256 bits), Base64URL, one per session | Required in the `X-CSRF-Token` header on state-changing session requests. |
| **PAT Format** | `yily_<base64url_192bit_token>` | Prefix identifies Yily credentials; 192 bits of cryptographically random entropy. |
| **PAT Storage** | SHA-256 hex digest (64 chars) | The complete PAT is shown only once; only its SHA-256 hash is persisted. |
| **PAT Scopes** | `read` or `write` | `write` implies `read`. No PAT can manage tokens or reach session-only operations. |
| **PAT Resource Scope** | Instance-wide (`all projects`), single project (`project_id`), or single environment (`project_id` + `environment_id`) | Enforces least privilege so CI/CD pipelines and team scripts can only access their assigned project or environment. |
| **PAT Lifetime** | 7, 30, 60 or 90 days; `write` defaults to 30 days and always expires; `read` may never expire | Limits exposure of a leaked token, especially one that can modify secrets. |
| **Recovery Code** | 16 bytes (128 bits), uppercase hex in groups of four (`XXXX-XXXX-…`) | Stored only as an Argon2id hash; single-use. |
| **API Auth Header** | `Authorization: Bearer yily_<token>` | Standard RFC 6750 Bearer authentication scheme. |

---

## 3. Storage & Cryptographic Representations

### 3.1. Password and Recovery Code PHC Strings
Passwords and recovery codes are stored as Argon2id hashes in the standard PHC string format:

```text
$argon2id$v=19$m=65536,t=3,p=4$<base64_salt>$<base64_hash>
```

Recovery codes are normalized before hashing and before verification (separators and whitespace removed, letters uppercased), so the Super Admin can type them with or without dashes and in any case.

### 3.2. Session & PAT Storage
Neither session tokens nor Personal Access Tokens are ever persisted in plaintext. Only their SHA-256 digests are stored:

```text
token_hash = hex(sha256(raw_token))
```

The CSRF token is the exception: it is stored in plaintext alongside its session, because the server must send it back to the web UI after a page reload. It is useless without the matching session cookie.

Password hashes, recovery code hashes, token hashes and CSRF tokens are never serialized in API responses or logs. The CSRF token is returned on purpose, and only, by the login response and `GET /api/v1/auth/csrf`.

---

## 4. Security Guarantees & Lifecycle

### 4.1. Initial Setup
* `POST /api/v1/auth/setup` creates the Super Admin. It requires the `setup_token` field to match `YILY_SETUP_TOKEN` (constant-time comparison).
* If `YILY_SETUP_TOKEN` is unset or blank, setup is refused; the instance never falls back to open registration.
* Setup is accepted only while no account exists. The check is repeated inside the account-creation transaction, so concurrent setup requests cannot create two accounts.
* On success the response is the one-time recovery code, served as a downloadable `yily-recovery-code.txt` file.

### 4.2. Timing Attack Mitigation (Email Enumeration)
Password verification with Argon2id takes approximately 50–100ms. If an invalid email returned immediately while an existing email took ~80ms, attackers could enumerate whether the Super Admin email exists.
* **Mitigation**: When the email is unknown, login still runs an Argon2id verification against a dummy hash. Every login attempt takes a uniform amount of time regardless of whether the email exists.

### 4.3. Session Fixation Prevention
When a login succeeds and the request already carries a session cookie:
1. The prior session is revoked on the server (only after the credentials have been verified).
2. A brand-new session token and CSRF token are generated.
3. The new token is set in the response cookie.

### 4.4. CSRF Protection
* Each session gets its own CSRF token at login. The web UI receives it in the login response, keeps it in memory, and re-fetches it with `GET /api/v1/auth/csrf` after a page reload.
* Any `POST`, `PUT`, `PATCH` or `DELETE` request authenticated by a session cookie is rejected with `403` unless its `X-CSRF-Token` header matches the session's token (constant-time comparison).
* PAT-authenticated requests are exempt: they carry an explicit `Authorization` header that a browser never attaches on its own.
* `GET`, `HEAD` and `OPTIONS` requests are exempt and must never change state.
* The server sends no CORS headers, so a cross-origin page cannot read any API response.

### 4.5. HTTPS & Reverse Proxy Handling
The `Secure` cookie flag is set when the request is detected as HTTPS:
1. A direct TLS connection is always HTTPS.
2. Forwarding headers (`X-Forwarded-Proto`, then `X-Forwarded-Protocol`, `X-Forwarded-Ssl`, `X-Url-Scheme`) are honored **only** when the direct peer is a loopback, link-local or private address, or a Unix socket. Headers sent by any other client are ignored.
3. When `X-Forwarded-Proto` is present, its **last** value is used. A reverse proxy must therefore set (overwrite) the header rather than append to the value sent by the client.

### 4.6. Personal Access Token (PAT) Lifecycle
Token management is restricted to the web session: a PAT can never create, list, rotate or revoke tokens.

* **Creation**: `POST /api/v1/tokens` with `name`, `scope` (`read` or `write`), optional `expires_in_days` (`7`, `30`, `60` or `90`), and optional resource restriction fields `project_id` and `environment_id`. If both `project_id` and `environment_id` are supplied, the environment must belong to that project (`400` `ENVIRONMENT_NOT_IN_PROJECT`); if only `environment_id` is supplied, its parent `project_id` is resolved automatically. Without `expires_in_days`, a `write` token expires after 30 days and a `read` token never expires. The raw token (`yily_...`) is returned in the response **only once**.
* **Listing**: `GET /api/v1/tokens` returns metadata only (`id`, `name`, `scope`, `project_id`, `environment_id`, `project_name`, `environment_name`, `created_at`, `expires_at`, `revoked_at`, `last_used_at`), never the raw token or its hash.
* **Rotation**: `POST /api/v1/tokens/:id/rotate` issues a replacement with the same name, permission `scope`, and resource restriction (`project_id` / `environment_id`) — plus an optional new `expires_in_days` — returns it once, and revokes the old token immediately. A revoked token cannot be rotated.
* **Revocation**: `DELETE /api/v1/tokens/:id` revokes a token. Revoked tokens are rejected on the next request.
* **Introspection**: `GET /api/v1/tokens/me` returns the metadata of the PAT used for the request, including its resolved `project_name` and `environment_name` when restricted (a session gets `403` `TOKEN_REQUIRED`).
* **Resource enforcement**:
  * **Instance-wide** (`project_id` and `environment_id` omitted): allowed on all non-session-only read and write endpoints across all projects.
  * **Project-scoped** (`project_id` set, `environment_id` omitted): restricted to that project and its environments, tags, and variables. Instance-wide operations (`GET /api/v1/dashboard`, `GET /api/v1/audit`, `GET /api/v1/audit/export`, `POST /api/v1/projects`, `PATCH /api/v1/projects/:id`, `DELETE /api/v1/projects/:id`) and resources in other projects return `403` (`TOKEN_RESOURCE_FORBIDDEN`). `GET /api/v1/projects` filters its response to the permitted project only.
  * **Environment-scoped** (`environment_id` set, with its parent `project_id`): restricted to reading the parent project and its tags, and reading or writing that specific environment and its variables, versions, and imports/exports. Project-level mutations (creating, cloning, or deleting environments; creating or deleting tags) and operations on any other environment return `403` (`TOKEN_RESOURCE_FORBIDDEN`). `GET /api/v1/projects` and `GET /api/v1/projects/:projectID/environments` filter their lists to the permitted project and environment.
  * If the target project or environment referenced by a restricted PAT is later deleted, token validation fails with `401 Unauthorized`.
* **Usage tracking**: every successful authentication updates `last_used_at`.

### 4.7. Password Recovery
`POST /api/v1/auth/recover` resets a forgotten password with the recovery code (`recovery_code`, `password`, `confirm`).

* The new password must satisfy the password policy, and the code is verified against its Argon2id hash after normalization.
* On success, a single database transaction:
  1. replaces the password hash,
  2. replaces the recovery code hash with a newly generated code (the old code stops working),
  3. revokes every active session,
  4. revokes every unexpired PAT (already-expired tokens keep their expired status).
* If any step fails, nothing changes: the old password and recovery code stay valid and the request returns an error.
* The new recovery code is returned once, as a downloadable `yily-recovery-code.txt` file.

Nothing obtained under the old password (a stolen session cookie, or a token created with it) survives a reset. Because every PAT is revoked, CLI and CI integrations need new tokens afterwards.

Recovery is currently the only way to change the Super Admin password.

---

## 5. Request Authentication

Every protected request goes through a single authentication step. An `Authorization` header always selects PAT authentication: there is no fallback to the cookie, so a request carrying an invalid or malformed header is rejected even if it also has a valid session cookie.

```text
Incoming Request
      │
      ▼
Authorization header present?
     ├── YES ──► Parse "Bearer <token>" (scheme case-insensitive)
     │                │
     │                ├── Malformed ──► 401 Unauthorized
     │                ▼
     │           Token starts with "yily_"? ── NO ──► 401 Unauthorized
     │                │
     │                ▼
     │           SHA-256 hash ──► Lookup PAT
     │                │
     │                ├── Not Found / Revoked / Expired ──► 401 Unauthorized
     │                └── Valid ──► Update last_used_at ──► Authenticated as PAT
     │
     └── NO  ──► Read "yily_session" cookie
                      │
                      ├── Missing / Empty ──► 401 Unauthorized
                      ▼
                 SHA-256 hash ──► Lookup Session
                      │
                      ├── Not Found / Revoked / Expired (TTL or Idle) ──► 401 Unauthorized
                      └── Valid ──► Update last_activity_at ──► Authenticated as Session
```

Once authenticated, each operation requires one of these access levels (plus any PAT resource restriction enforced by the service layer):

| Access level | Allowed callers | Rejected callers |
| :--- | :--- | :--- |
| **Read** | Session, `read` PAT, `write` PAT (within PAT resource scope) | Out-of-scope PAT (`403` `TOKEN_RESOURCE_FORBIDDEN`) |
| **Write** | Session (with valid CSRF token), `write` PAT (within PAT resource scope) | `read` PAT (`403`), out-of-scope PAT (`403` `TOKEN_RESOURCE_FORBIDDEN`) |
| **PAT only** | `read` PAT, `write` PAT | Session (`403` `TOKEN_REQUIRED`) |
| **Session only** | Session (with valid CSRF token on state-changing methods) | Any PAT (`403`) |

### 5.1. Route Access Classification

| Group | Access requirement | Routes (`/api/v1/...`) |
| :--- | :--- | :--- |
| **Public** | Unauthenticated | `GET /auth/status`, `POST /auth/setup`, `POST /auth/login`, `POST /auth/recover` |
| **Session only** | Web session (+ CSRF on mutating requests) | `POST /auth/logout`, `GET /auth/me`, `GET /auth/csrf`, `GET /version`, `GET /version/update`, `POST /tokens`, `GET /tokens`, `POST /tokens/:id/rotate`, `DELETE /tokens/:id`, `DELETE /projects` |
| **PAT only** | Any valid PAT (`read` or `write`) | `GET /tokens/me` |
| **Global Read** | Session or global PAT | `GET /dashboard`, `GET /audit` |
| **Global Write** | Session or global `write` PAT | `POST /projects`, `PATCH /projects/:id`, `DELETE /projects/:id`, `GET /audit/export` |
| **Project / Env Read** | Session or matching-scope PAT (lists filtered for scoped PATs) | `GET /projects`, `GET /projects/:id`, `GET /projects/:projectID/tags`, `GET /projects/:projectID/environments`, `GET /environments/:id`, `GET /environments/:id/export`, `GET /environments/:environmentID/variables`, `POST /environments/:environmentID/variables/values`, `GET /variables/:id`, `GET /variables/:id/value`, `GET /variables/:id/versions`, `GET /variables/:id/versions/current` |
| **Project Write** | Session, global `write` PAT, or project-scoped `write` PAT | `POST /projects/:projectID/environments`, `POST /environments/:id/clone`, `DELETE /environments/:id`, `POST /projects/:projectID/tags`, `DELETE /projects/:projectID/tags/:tagID` |
| **Environment Write** | Session, global `write` PAT, project-scoped `write` PAT, or matching env-scoped `write` PAT | `PATCH /environments/:id/name`, `PATCH /environments/:id/description`, `POST /environments/:id/imports`, `POST /environments/:environmentID/variables`, `PATCH /variables/:id`, `DELETE /variables/:id`, `PUT /variables/:id/tags`, `POST /variables/:id/versions`, `PATCH /variables/:id/versions/:versionID`, `DELETE /variables/:id/value` |

---

## 6. Out of Scope (Future Phases)

* Changing the password from an authenticated session (today, recovery is the only way to change it).
* Lockout after consecutive failed login or recovery attempts. The only protection today is the global per-address rate limiter; behind a reverse proxy, every client shares the proxy's address.
* Multi-user accounts and role-based access control (RBAC).
