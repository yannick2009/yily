Oui. Vu la nouvelle décision de Yannick, il faut aussi nettoyer le document : **pas de rôles, pas de multi-utilisateur, pas de permissions complexes**. On garde l’authentification et une autorisation minimale cohérente avec une instance YILY mono-utilisateur.

```md
# Authentication and Security Architecture for YILY

## 1. Objective

YILY should provide a fully internal authentication mechanism without relying on an external identity provider, SMTP, or any other external service.

The solution should support the main YILY usage modes:

- Web interface
- HTTP API
- CLI

The objective is to keep the authentication system lightweight while maintaining a clear separation between authentication, authorization, business logic, persistence, and cryptography.

## 2. Proposed Architecture

```text
                 Web / API / CLI
                       |
                       v
                Authentication
                       |
                       v
                  Middleware
                       |
                       v
                 Authorization
                       |
                       v
                    Service
                  /         \
                 v           v
            Repository     Crypto
                              |
                              v
                         Master Key
```

Each component has a specific responsibility.

## 3. Authentication

YILY manages authentication internally.

A single user is supported at this stage: the default Super Admin created during the initial setup.

The user's password is protected using Argon2id with a randomly generated salt.

The password is never stored in plaintext and is not encrypted as application data.

Authentication remains independent from the Master Key used to encrypt stored secrets.

## 4. Session Management

The Web interface uses server-side sessions.

After successful authentication, YILY generates a cryptographically secure session identifier and sends it to the browser through a secure cookie.

The cookie should use:

- `HttpOnly`
- `Secure` when HTTPS is enabled
- `SameSite`

Sessions should have an expiration mechanism and support revocation.

A new session should be created after authentication to prevent session fixation.

## 5. Authorization

Since YILY currently supports only one user, no role or permission management is required at this stage.

The authenticated user is the Super Admin of the local instance and has access to the resources of that instance.

Authorization therefore remains lightweight and focuses on verifying that:

- the request is authenticated;
- the requested resource belongs to the current YILY instance;
- sensitive operations are explicitly allowed in the current authentication context.

In particular, authentication must be validated before accessing or decrypting sensitive values.

A more advanced authorization model can be introduced later if YILY becomes multi-user.

## 6. Middleware

The HTTP middleware validates the authentication mechanism for protected routes.

It retrieves the session and makes the authenticated identity available to the request context.

The middleware should remain independent from business logic.

It is responsible for establishing a trusted authentication context before the request reaches the application services.

## 7. Service

The Service layer orchestrates business operations.

It coordinates authentication-related checks, data persistence, and calls to the Crypto package when required.

The same business services should be reusable by:

- the Web interface;
- the HTTP API;
- the CLI.

The Repository should not be responsible for authentication, authorization, or encryption logic.

## 8. Repository

The Repository layer manages persistence.

It provides access to authentication-related data such as:

- the Super Admin account;
- sessions;
- API tokens, if required.

It also continues to manage the existing YILY business data.

The Repository should not handle browser cookies, password verification, or cryptographic operations.

## 9. Crypto

The existing Crypto package remains responsible for encryption and decryption of sensitive application values.

It remains independent from authentication.

The Master Key must not be derived from the user's password and must remain separate from the authentication mechanism.

## 10. Web Interface

The user authenticates directly against the local YILY instance.

After successful authentication:

```text
User
  |
  v
Login
  |
  v
Authentication
  |
  v
Session created
  |
  v
Secure cookie
  |
  v
Protected requests
```

Protected requests are validated by the middleware before reaching the relevant service.

CSRF protection should be applied where required by the cookie-based authentication model.

## 11. HTTP API

The API follows the same authentication and authorization rules as the Web interface.

For requests originating from the Web interface, the existing session mechanism can be reused.

If a client such as the CLI needs to communicate with a remote YILY instance, an API token can be used instead.

API tokens should be:

- cryptographically random;
- associated with the local Super Admin;
- revocable;
- optionally expirables;
- stored as hashes when possible.

## 12. CLI

The CLI should be able to operate against a local YILY instance without depending on a browser or an external authentication service.

For local access, authentication can rely on the local instance and its security boundaries.

When the CLI communicates with a remote YILY instance, it should authenticate using a dedicated API token.

The CLI should reuse the same Service layer rather than implementing a separate business logic.

The CLI can be distributed as pre-built binaries through native package managers using GoReleaser, avoiding the need for users to compile YILY themselves.

## 13. Password Reset

Password reset must be handled entirely by YILY without SMTP or an external service.

Since YILY currently has a single Super Admin, the recovery mechanism should be designed around local administrative access to the instance.

The reset mechanism should use a temporary, cryptographically random value with:

- limited validity;
- single use;
- explicit invalidation after use;
- session revocation after a successful password reset.

The temporary reset value should not be stored in plaintext when storing its hash is sufficient for validation.

## 14. Authentication Data

The initial data model can remain minimal:

```text
USER
- ID
- username / email
- password_hash
- status
- created_at
- updated_at

SESSION
- ID
- user_id
- token_hash
- expires_at
- revoked_at

API_TOKEN
- ID
- user_id
- token_hash
- expires_at
- revoked_at
```

No role field is required at this stage because the installation has only one user: the Super Admin.

## 15. Master Key Separation

The Master Key remains independent from authentication.

The user's password is used to authenticate the user, not to encrypt or derive the Master Key.

The authentication mechanism therefore does not directly provide access to the Master Key.

This maintains a clear separation:

```text
User Password
      |
      v
Authentication
      |
      v
Access to YILY
```

while:

```text
Master Key
      |
      v
Crypto
      |
      v
Encrypted Secrets
```

## 16. Main Security Decisions

The initial implementation should:

1. Use Argon2id for password hashing.
2. Use server-side sessions for the Web interface.
3. Use secure cookie attributes.
4. Keep authentication and cryptography independent.
5. Validate authentication before sensitive operations.
6. Decrypt secrets only when their value is explicitly requested.
7. Use API tokens for remote CLI/API access when required.
8. Make sessions and tokens revocable.
9. Apply login-attempt protection.
10. Prevent passwords, tokens, Master Keys, and decrypted secrets from appearing in logs.
11. Use HTTPS for network communication.
12. Test invalid authentication, expired sessions, revoked sessions, invalid tokens, and unauthorized access.

## 17. Implementation Scope

This document defines the proposed architecture only.

Once the proposal is validated, the concrete implementation can define:

- Go packages and interfaces;
- authentication services;
- session management;
- HTTP middleware;
- repository methods;
- authentication routes;
- CLI authentication flow;
- password reset flow;
- security tests.

The exact interfaces and method names should be decided during implementation rather than being imposed by this proposal.

## Conclusion

The proposed approach keeps YILY lightweight and fully self-managed.

The current model is intentionally limited to one user: the default Super Admin. No role or multi-user authorization system is introduced at this stage.

Web authentication uses server-side sessions, while remote CLI/API access can use dedicated API tokens.

Authentication remains separated from the existing Crypto package and Master Key, allowing each security mechanism to have a focused responsibility.

The architecture can later evolve toward multi-user authorization without requiring the authentication and cryptographic layers to be redesigned.
```