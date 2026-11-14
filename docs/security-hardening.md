# Authentication & Transport Security Hardening

This document explains the authentication, session, token, proxy, and client transport security weaknesses identified in Yily and the targeted fixes implemented under the project's **"Simple and effective"** design principle.

Every change relies strictly on Go's standard library, Echo v5's built-in security primitives, and existing SQLite/GORM transactions—avoiding external services (such as Redis) or unnecessary abstractions.

---

## Summary of Changes

| Block | Area | Problem | Solution |
| :--- | :--- | :--- | :--- |
| **1** | First-run Setup | Anonymous network callers could race `POST /api/v1/auth/setup` before the operator initialized the instance; password checks counted raw UTF-8 bytes and accepted whitespace padding. | Added `YILY_SETUP_TOKEN` (with automatic one-time bootstrap token generation when unset), rune/blank/email password checks, and UI field support when required. |
| **1** | Login Hardening & DoS | `POST /api/v1/auth/login` skipped validator checks, had no payload size bound before Argon2id, had no concurrent Argon2id cap (~64 MiB/call), lacked brute-force rate limiting, and did not log failures. | Enforced `bindAndValidate`, 512-byte password cap, bounded Argon2id semaphore (`4` slots / ~256 MiB max), per-IP rate limiting + exponential cooldown (`429`), and structured `slog` failure logs. |
| **2** | Session & Account Recovery | No endpoint existed to change the Super Admin password or revoke active sessions across devices; `PAT.Rotate` was non-transactional, resurrected revoked PATs, bypassed validation on chunked bodies, and omitted error codes. | Added `POST /api/v1/auth/password` (revokes all sessions), `POST /api/v1/auth/logout-all`, transactional `PAT.Rotate` refusing revoked tokens, chunked-safe validation + `respondError`, `DELETE /api/v1/tokens/me`, and `yily logout` server revocation. |
| **3** | Proxy, CSP, CSRF & CLI | `X-Forwarded-Proto` was trusted unconditionally, `RealIP` wasn't proxy-aware, no CSP was configured (`client/index.html` used inline JS), cookie mutations lacked subdomain `Origin` checks, and the CLI followed redirects / allowed unencrypted HTTP silently. | Added `YILY_TRUSTED_PROXIES` for Echo `IPExtractor`/`SchemeExtractor`, strict `Content-Security-Policy` with external `/theme-init.js`, `RequireSameOrigin` middleware, CLI redirect refusal, and HTTPS enforcement (`--insecure-http` opt-in). |

---

## Block 1 — Initial Setup & Login Hardening

### 1.1 Pre-setup Takeover Prevention (`POST /api/v1/auth/setup`)

- **Problem**: On an uninitialized instance (`userCount == 0`), `POST /api/v1/auth/setup` was open to any caller able to reach the HTTP port, enabling first-come-first-served account takeover.
- **Solution**:
  - When no Super Admin exists yet, `server/cmd/server/main.go` loads `YILY_SETUP_TOKEN` or, if unset, generates a 256-bit cryptographically random one-time bootstrap token via `crypto.GenerateSetupToken()` and prints it to `stderr`.
  - `Setup` verifies `Count > 0` first (returning `409 Conflict` once initialized so an old `YILY_SETUP_TOKEN` never becomes a guessing oracle), then validates `setup_token` (or header `X-Setup-Token`) with `subtle.ConstantTimeCompare` before running Argon2id (`403 AUTH_INVALID_SETUP_TOKEN` on mismatch).
  - Once setup succeeds, the in-memory bootstrap token is wiped and `GET /api/v1/auth/status` reports `setup_token_required: false`. The frontend (`/init`) dynamically prompts for the setup token only when `setup_token_required` is true.

### 1.2 Password Strength Rules

- **Problem**: `len(password) < 12` measured bytes instead of Unicode characters, accepted all-whitespace strings (`"            "`) or single-character padded secrets (`"a           "`), and had no upper bound before passing inputs into Argon2id.
- **Solution** (`service.ValidatePassword` & `crypto.MaxPasswordBytes`):
  - Requires `12` to `128` Unicode runes after trimming outer whitespace (`utf8.RuneCountInString(strings.TrimSpace(password)) >= 12`), allowing multi-word passphrases with internal spaces while rejecting blank or padded passwords.
  - Rejects passwords matching the account email (case-insensitive) and bounds inputs to `512` UTF-8 bytes.
  - `Login` rejects passwords exceeding `512` bytes with `401 AUTH_INVALID_CREDENTIALS` before executing Argon2id.

### 1.3 Argon2id Memory DoS Cap

- **Problem**: Each Argon2id call allocates `64 MiB` (`m = 64 * 1024` KiB, `t = 3`, `p = 4`). Concurrent login/setup bursts multiplied memory linearly without bound.
- **Solution** (`server/internal/crypto/password.go`):
  - Added context-aware `HashPasswordContext` and `VerifyPasswordContext` guarded by a buffered channel semaphore (`maxConcurrentArgon2 = 4`, capping peak working memory to ~`256 MiB`) with a `1.5s` wait timeout (`argon2AcquireWait`).
  - Under saturation or client disconnect, waiting requests abort early instead of queuing allocations and return `503 AUTH_SERVICE_BUSY` with `Retry-After: 2`.

### 1.4 Login Validation, Rate Limiting & Progressive Cooldown

- **Problem**: `Login` used raw `c.Bind` without `bindAndValidate`, shared a single global `50 req/s` limiter across all endpoints (too permissive for brute force), and logged neither failed logins nor lockouts.
- **Solution** (`server/internal/middleware/ratelimit.go`, `server/internal/handler/auth.go`):
  - Switched `Login` to `bindAndValidate` (enforcing valid email syntax and bounds up front).
  - Attached `NewAuthRateLimiter` exclusively to `POST /api/v1/auth/login` and `POST /api/v1/auth/setup`:
    - Per-IP quota: **10 requests / 1 minute**.
    - Failure-based exponential backoff: after **5 consecutive `401`/`403` responses** from the same client IP, cooldown activates at **15s**, doubling per additional failure (`30s`, `60s`, ...) up to **15 minutes** (`429 AUTH_RATE_LIMITED` + `Retry-After` header). Resets immediately on a `2xx` response.
    - Tracked IP map is bounded to `10,000` entries with stale/oldest eviction to prevent memory exhaustion under distributed sprays.
  - Failed logins, setup rejections, and rate-limit lockouts emit structured `slog.Warn` events (without writing unauthenticated brute-force traffic into SQLite's `audit_events` table).

---

## Block 2 — Session Revocation, Password Change & PAT Rotation

### 2.1 Password Rotation & Global Session Revocation

- **Problem**: If the Super Admin password or a session cookie was suspected compromised, there was no API to rotate the password or invalidate sessions on other devices.
- **Solution**:
  - `POST /api/v1/auth/logout-all` (`RequireSession` only): revokes every active session for the user (`SessionRepository.RevokeAllByUserID`), clears the `yily_session` cookie, and records an audit event (`revoked`, `scope: all_sessions`).
  - `POST /api/v1/auth/password` (`RequireSession` only): validates `current_password`, enforces `ValidatePassword` and non-equality (`ErrSamePassword -> 400 AUTH_SAME_PASSWORD`) on `new_password`, updates the Argon2id hash (`UserRepository.UpdatePasswordHash`), revokes all active sessions across all devices, clears the caller's cookie, and records an audit event (`updated`, `event: password_changed`).

### 2.2 Transactional `PAT.Rotate` & Validation Fixes

- **Problem**:
  1. `patService.Rotate` created the replacement token before revoking the old one outside a transaction, allowing race windows and double rotation.
  2. Already-revoked PATs could be rotated to resurrect access.
  3. `patHandler.Rotate` checked `c.Request().ContentLength > 0`, which was bypassed by `Transfer-Encoding: chunked` (`ContentLength == -1`) and omitted error `Code` fields by bypassing `respondError`.
- **Solution**:
  - `PATRepository.Rotate` performs lookup, active verification (`!existing.IsRevoked()`), conditional revocation (`WHERE id = ? AND user_id = ? AND revoked_at IS NULL`), and replacement insertion inside a single SQLite transaction. Revoked or raced PATs return `409 TOKEN_ALREADY_REVOKED`.
  - `patHandler.Rotate` reads optional bodies via `readOptionalBody` (catching both content-length and chunked JSON payloads) and uses `respondError` consistently across all failure paths.

### 2.3 Server-Side CLI Token Revocation

- **Problem**: `yily logout` only deleted `~/.config/yily/config.json`, leaving the PAT valid on the server until expiration.
- **Solution**:
  - Added `DELETE /api/v1/tokens/me` (`readRoutes`) enabling a PAT to revoke **only itself** (`patHandler.RevokeSelf`), returning `403 TOKEN_REQUIRED` if called with a session cookie.
  - Updated `yily logout` to call `DELETE /api/v1/tokens/me` before deleting the local config (with `--local` available to skip the server call when offline).

---

## Block 3 — Proxy Trust, Browser Hardening & CLI Transport Security

### 3.1 Explicit Reverse Proxy Trust (`YILY_TRUSTED_PROXIES`)

- **Problem**: `isHTTPSRequest` trusted `X-Forwarded-Proto` from any peer, while `c.RealIP()` defaulted to direct TCP peers—making rate limits share a single proxy IP behind Nginx/Caddy/Traefik or allowing header spoofing when directly exposed.
- **Solution**:
  - Added `YILY_TRUSTED_PROXIES` (comma-separated IPs, CIDRs, `"loopback"`, or `"private"`).
  - When unset (default): `Echo.IPExtractor = echo.ExtractIPDirect()` and `Echo.SchemeExtractor = echo.ExtractSchemeDirect()`. Untrusted `X-Forwarded-*` headers are completely ignored.
  - When set: `Echo.IPExtractor` and `Echo.SchemeExtractor` use Echo v5's `ExtractIPFromXFFHeader` and `ExtractSchemeFromHeaders` with `TrustLoopback(false)`, `TrustLinkLocal(false)`, `TrustPrivateNet(false)`, and explicit `TrustIPRange` filters for the configured proxies only.
  - Simplified `isHTTPSRequest` to rely on `c.Request().TLS != nil || strings.EqualFold(c.Scheme(), "https")`.

### 3.2 Strict Content-Security-Policy & Security Headers

- **Problem**: Echo's default `Secure()` middleware sent no `Content-Security-Policy`, and `client/index.html` embedded an inline `<script>` for pre-paint theme loading.
- **Solution**:
  - Moved the theme bootstrapper to `client/public/theme-init.js` and updated `client/index.html` to load `<script src="/theme-init.js"></script>`.
  - Configured `echomw.SecureWithConfig` with `XFrameOptions: "DENY"`, `ReferrerPolicy: "strict-origin-when-cross-origin"`, conditional HTTPS HSTS (`HSTSMaxAge: 31536000`), and strict CSP:
    ```text
    default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; object-src 'none'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'
    ```

### 3.3 `Origin` & Fetch Metadata Guard (`RequireSameOrigin`)

- **Problem**: `SameSite=Lax` cookies protect against third-party sites, not compromised same-site subdomains (e.g., `sub.example.com` submitting requests to `yily.example.com`).
- **Solution** (`server/internal/middleware/origin.go`):
  - Applied `RequireSameOrigin` to `/api/v1` for state-changing methods (`POST`, `PUT`, `PATCH`, `DELETE`) that do not carry a valid `Authorization: Bearer` header.
  - Rejects `Origin: null`, cross-origin or subdomain mismatches (`Origin` host vs request `Host`), HTTP-to-HTTPS origin downgrades, and `Sec-Fetch-Site: cross-site` / `same-site` when `Origin` is omitted (`403 AUTH_FORBIDDEN_ORIGIN`).
  - Updated `client/vite.config.ts` dev proxy (`changeOrigin: false`) so Vite local development preserves `Host` matching `Origin`.

### 3.4 CLI Redirect & Cleartext HTTP Protections

- **Problem**: `cli.NewClient` used Go's default redirect policy (which can forward `Authorization: Bearer` on redirects) and allowed cleartext `http://` URLs to remote hosts silently.
- **Solution** (`server/internal/cli/client.go`, `config.go`, `register.go`):
  - Configured `CheckRedirect` on `Client.httpClient` to refuse HTTP redirects explicitly so bearer tokens are never forwarded across redirect hops.
  - Added `ValidateServerURL`: rejects non-loopback `http://` URLs during `yily login` unless `--insecure-http` is explicitly passed (`http://localhost`, `127.0.0.1`, and `::1` remain allowed for local development), rejects embedded URL userinfo credentials, and warns to `stderr` if an older config still points to remote `http://`.

---

## Operator Configuration Reference

| Variable | Default | Purpose |
| :--- | :--- | :--- |
| `YILY_SETUP_TOKEN` | Auto-generated random token (if uninitialized) | Shared secret required via `setup_token` or `X-Setup-Token` during `POST /api/v1/auth/setup`. |
| `YILY_TRUSTED_PROXIES` | Empty (direct peer only) | Comma-separated proxy IPs, CIDRs, or `loopback` / `private` aliases authorized to supply `X-Forwarded-For` and `X-Forwarded-Proto`. Example: `loopback` or `127.0.0.1,172.17.0.0/16`. |

---

## Automated Security Verification

Unit and `httptest` security suites verify these controls under the Go race detector (`cd server && go test -race ./...`):

- `internal/crypto/password_test.go`: password byte limits, Argon2id bounded semaphore, and context cancellation.
- `internal/middleware/ratelimit_test.go`: per-IP request quotas, window resets, progressive failure backoff (`401`/`403` -> `429` + `Retry-After`), and success streak resets.
- `internal/middleware/origin_test.go`: cross-origin, subdomain, `null` origin, HTTP->HTTPS mismatch, and `Sec-Fetch-Site` enforcement.
- `internal/config/config_test.go`: `YILY_SETUP_TOKEN` and `YILY_TRUSTED_PROXIES` parsing/validation.
- `internal/service/auth_test.go` & `pat_test.go`: password rules, setup token validation, concurrent setup/rotate races, revoked PAT rotation rejection, password rotation session invalidation, and `LogoutAll`.
- `internal/handler/auth_pat_test.go`: chunked JSON validation on `/tokens/:id/rotate`, `DELETE /tokens/me`, `/auth/password`, `/auth/logout-all`, CSP/`X-Frame-Options` headers, and proxy spoofing vs trusted proxy cookie/scheme handling.
- `internal/cli/client_test.go`: redirect credential leak prevention and `ValidateServerURL` HTTP/HTTPS rules.
