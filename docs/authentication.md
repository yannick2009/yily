# Authentication and Session Specification

## 1. Overview

Yily implements a self-contained, zero-external-dependency authentication and authorization system. It does not rely on third-party identity providers, OAuth, or external SMTP servers.

Access is restricted to a single account: the **Super Admin**. That account is reached through two access models:

1. **Web Interface**: server-side sessions carried by an `HttpOnly` cookie, with a per-session CSRF token for state-changing requests.
2. **Programmatic / API Access**: Personal Access Tokens (PAT) sent as `Authorization: Bearer yily_<token>`, each scoped to `read` or `write`.

Interactive sign-in can optionally be protected by a **TOTP authenticator app** (RFC 6238). TOTP acts either as a second factor after the password or as the sole credential for passwordless sign-in. It never replaces a Personal Access Token: automated clients still authenticate with PATs.

Two out-of-band secrets complete the model:

* **Setup token** (`YILY_SETUP_TOKEN`): required once, to create the Super Admin on a fresh instance.
* **Recovery code**: issued at setup, used to reset a forgotten password. It is single-use and replaced on every reset.

The TOTP seed is a third secret, but the server generates it during enrollment instead of receiving it out of band.

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
| **PAT Lifetime** | 7, 30, 60 or 90 days; `write` defaults to 30 days and always expires; `read` may never expire | Limits exposure of a leaked token, especially one that can modify secrets. |
| **TOTP Algorithm** | RFC 6238, HMAC-SHA1, 6 digits, 30-second period | Widely supported default profile for authenticator apps. |
| **TOTP Secret** | 20 bytes (160 bits), Base32 | RFC 4226 recommends at least 128 bits of entropy. |
| **TOTP Secret Storage** | AES-256-GCM, AAD bound to the user ID | The server must recover the seed to verify codes, so it cannot be hashed; the AAD prevents moving the ciphertext between accounts. |
| **TOTP Skew** | ±1 time step | Covers small clock differences without widening the guessing window. |
| **TOTP Replay Protection** | Last accepted time step persisted per account | A code can be used only once per step, including the code that completed enrollment. |
| **TOTP Lockout** | 5 consecutive invalid codes → 5-minute lock | A 6-digit code has only 10⁶ possibilities; a per-account lock is required on top of the global rate limiter. |
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

### 3.3. TOTP Seed Storage
A TOTP seed cannot be hashed: the server must recover it on every login to compute the expected code. It is therefore the only reversible authentication secret in the database, and it is protected accordingly:

```text
totp_secret_encrypted = AES-256-GCM(YILY_MASTER_KEY, seed, AAD = "yily:totp:<user_id>")
```

Binding the ciphertext to the user ID with the AAD means a seed copied to another account (or a database row swapped between users) fails authentication. The seed is never returned after enrollment, never serialized in `User` JSON, and never written to logs. Only the `totp_enabled` flag is exposed to clients.

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

* **Creation**: `POST /api/v1/tokens` with `name`, `scope` (`read` or `write`) and an optional `expires_in_days` (`7`, `30`, `60` or `90`). Without it, a `write` token expires after 30 days and a `read` token never expires. The raw token (`yily_...`) is returned in the response **only once**.
* **Listing**: `GET /api/v1/tokens` returns metadata only (`id`, `name`, `scope`, `created_at`, `expires_at`, `revoked_at`, `last_used_at`), never the raw token or its hash.
* **Rotation**: `POST /api/v1/tokens/:id/rotate` issues a replacement with the same name and scope (and an optional new `expires_in_days`), returns it once, and revokes the old token immediately. A revoked token cannot be rotated.
* **Revocation**: `DELETE /api/v1/tokens/:id` revokes a token. Revoked tokens are rejected on the next request.
* **Introspection**: `GET /api/v1/tokens/me` returns the metadata of the PAT used for the request (a session gets `403`).
* **Usage tracking**: every successful authentication updates `last_used_at`.

### 4.7. Password Recovery
`POST /api/v1/auth/recover` resets a forgotten password with the recovery code (`recovery_code`, `password`, `confirm`).

* The new password must satisfy the password policy, and the code is verified against its Argon2id hash after normalization.
* On success, a single database transaction:
  1. replaces the password hash,
  2. replaces the recovery code hash with a newly generated code (the old code stops working),
  3. disables TOTP and clears its seed, replay marker and failure counters,
  4. revokes every active session,
  5. revokes every unexpired PAT (already-expired tokens keep their expired status).
* If any step fails, nothing changes: the old password and recovery code stay valid and the request returns an error.
* The new recovery code is returned once, as a downloadable `yily-recovery-code.txt` file.

Nothing obtained under the old password (a stolen session cookie, or a token created with it) survives a reset. Because every PAT is revoked, CLI and CI integrations need new tokens afterwards. Disabling TOTP here deliberately trades the second factor for account recovery: the recovery code is the only way back in when the authenticator app is lost.

Recovery is currently the only way to change the Super Admin password.

### 4.8. TOTP Two-Factor and Passwordless Login
TOTP is optional per account and is enabled from the profile page after signing in with the password.

* **Enrollment**: `POST /api/v1/auth/totp/setup` (session + CSRF) generates a fresh 160-bit seed, encrypts it with `YILY_MASTER_KEY` and binds the ciphertext to the user ID with AES-GCM AAD. The response contains the Base32 `secret`, the `otpauth://` provisioning URI and a PNG QR code as a data URI. TOTP stays disabled at this point.
* **Confirmation**: `POST /api/v1/auth/totp/confirm` (session + CSRF) verifies a code against the pending seed and enables TOTP. The accepted time step is stored, so the enrollment code cannot be replayed at login. Restarting `setup` before confirming replaces the pending seed; an abandoned enrollment never locks the account.
* **Password + code**: `POST /api/v1/auth/login` accepts an optional `totp_code`. When the password is correct but TOTP is enabled and the code is missing, the server returns `401` with code `AUTH_TOTP_REQUIRED`; the web UI then reveals the second-factor field and resubmits both values. Sessions are created only after both factors succeed.
* **Passwordless login**: `POST /api/v1/auth/login/totp` accepts `email` and `code` and issues the same session cookie. Unknown emails, accounts without TOTP, wrong codes and locked accounts all return the generic `AUTH_INVALID_CREDENTIALS`, so the endpoint cannot be used to enumerate accounts or probe lockout state.
* **Replay and lockout**: a code is accepted once per time step, with ±1 step of clock skew. Each invalid code increments a per-account counter; five consecutive failures lock TOTP verification for five minutes (`429` `AUTH_TOTP_LOCKED`). A missing code is not a failure. Because passwordless login is public, an attacker who knows the admin email can deliberately trigger this lockout; the trade-off is bounded (five minutes) and is what makes 6-digit online guessing impractical.
* **Disabling**: `POST /api/v1/auth/totp/disable` (session + CSRF) requires the account password. A PAT cannot reach it.
* **Recovery**: using the recovery code disables TOTP as part of the credential reset, so a lost authenticator app is never a permanent lockout.
* **Audit**: enabling and disabling TOTP are recorded as `totp_enabled` / `totp_disabled` events with the `user` resource.

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

Once authenticated, each operation requires one of these access levels:

| Access level | Allowed callers | Rejected callers |
| :--- | :--- | :--- |
| **Read** | Session, `read` PAT, `write` PAT | — |
| **Write** | Session (with valid CSRF token), `write` PAT | `read` PAT (`403`) |
| **Session only** | Session (with valid CSRF token on state-changing methods) | Any PAT (`403`) |

Token management, the Super Admin profile and all `/auth/totp/*` management endpoints are **session only**. Setup, login, passwordless login and recovery are public.

---

## 6. Out of Scope (Future Phases)

* Changing the password from an authenticated session (today, recovery is the only way to change it).
* Lockout after consecutive failed password or recovery attempts. TOTP codes already have a per-account lockout (section 4.8); password and recovery attempts are protected only by the global per-address rate limiter, and behind a reverse proxy every client shares the proxy's address.
* Multi-user accounts and role-based access control (RBAC).
