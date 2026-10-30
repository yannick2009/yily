# YILY — Authentication and Security Implementation Proposal

## 1. Objective

Implement authentication fully managed by YILY, without relying on an identity provider, SMTP, or any external service.

The current scope covers the Web interface with a single account: the Super Admin.

The objective is to secure access to YILY and its secrets while limiting the impact of a compromise of any individual layer.

## 2. Authentication

The Super Admin authenticates using:

- email address;
- strong password.

The strong password is protected using **Argon2id**, with a salt generated using `crypto/rand`.

After **5 consecutive failed attempts**, further attempts are temporarily blocked. The account is never automatically deleted, preventing an attacker from permanently locking the Super Admin out.

Login errors remain generic to avoid revealing whether an account exists.

## 3. Web Sessions

YILY uses **server-side sessions** instead of JWTs for the Web interface.

The session token is generated using `crypto/rand`. Only its **SHA-256 hash** is stored in SQLite.

Each session has:

- an expiration date;
- a revocation state;
- a maximum inactivity period.

A new session is created after each successful authentication to prevent session fixation.

Logging out revokes the session on the server side.

## 4. Cookie and CSRF

The session token is transmitted through a cookie configured with:

- `HttpOnly`;
- `Secure` when HTTPS is used;
- `SameSite`.

Requests that modify YILY state also use a **CSRF token generated with `crypto/rand`**.

When YILY is behind a reverse proxy, `X-Forwarded-Proto: https` must be correctly handled to determine the original protocol used by the client.

## 5. Route Protection

Sensitive routes require a valid session.

This includes:

- reading, creating, modifying, and deleting secrets;
- changing the strong password;
- managing PATs;
- recovery operations.

The middleware validates the session before passing the request to the services.

The identity used by the services comes from the validated authentication context and not from a value arbitrarily provided by the client.

## 6. Recovery Code

Account recovery does not depend on an external service.

A **random Recovery Code** is generated using `crypto/rand`.

The code is never stored in plaintext; only its **SHA-256 hash** is stored.

When it is used:

- a new strong password is set;
- the Recovery Code is invalidated;
- all existing sessions are revoked;
- existing PATs are revoked.

The Recovery Code can only be used once.

If the Super Admin loses both the strong password and the Recovery Code, no automatic recovery mechanism can bypass these protections.

## 7. Personal Access Token (PAT)

PATs are intended for clients that need to communicate with a remote YILY instance.

They are generated using `crypto/rand` and provide **256 bits of entropy**.

Format:

```text
yily_<token>
```

The complete token is never stored in plaintext. Only its **SHA-256 hash** is stored.

API authentication uses:

```text
Authorization: Bearer yily_<token>
```

PATs can have an expiration date and can be revoked.

## 8. Master Key Protection

The Master Key is independent from the strong password and authentication mechanisms.

It is never stored in SQLite, the repository, or logs.

Secrets remain encrypted using **AES-256-GCM**, with a random nonce generated using `crypto/rand`.

The Master Key must be stored separately from SQLite. A key storage mechanism protected by the execution environment, such as a TPM or system keystore, may be used to strengthen this protection.

## 9. YILY Isolation

YILY should not run as `root` when it is not required.

The process uses a dedicated system user, for example `yily`, with only the permissions required for its operation.

The container:

- must not use `privileged`;
- must not expose the Docker socket;
- should drop unnecessary Linux capabilities;
- should limit mounted volumes to those strictly required.

The objective is to ensure that compromising YILY does not automatically grant `root` privileges.

## 10. SQLite and Sensitive Data

SQLite is protected by system-level permissions and is accessible only to the YILY process.

Sensitive data is not stored in plaintext when cryptographic protection is required.

Authentication tokens (sessions, PATs, and Recovery Codes) are stored as **SHA-256 hashes**.

The database alone must therefore not be sufficient to directly recover usable authentication tokens or the Master Key.

## 11. Logs and Errors

Secrets must never appear in logs or error messages:

- strong password;
- session token;
- PAT;
- Recovery Code;
- Master Key;
- decrypted secrets.

Security events may be logged without their sensitive values.

## 12. Implementation Order

Implementation can be carried out progressively:

1. Super Admin + strong password + Argon2id
2. Authentication + 5-attempt limit
3. Sessions + expiration + revocation
4. Cookie + CSRF
5. Middleware + route protection
6. Recovery Code
7. PAT
8. Master Key protection
9. Non-root execution + container hardening
10. Security testing and verification of compromise scenarios

The final objective is a complete, lightweight Web authentication system that is independent of external services, with a clear separation between authentication, sessions, secrets, and encryption.