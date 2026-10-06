# Variable Encryption Specification

## 1. Overview

Yily stores sensitive secrets (API keys, tokens, credentials) in the `VariableVersion.Value` column. To ensure zero plaintext secrets at rest, all values must be encrypted using the application's `YILY_MASTER_KEY` before being written to SQLite.

This specification defines the encryption scheme used by Yily to protect secrets at rest.

---

## 2. Technical Specifications

| Parameter | Specification | Rationale |
| :--- | :--- | :--- |
| **Algorithm** | **AES-256-GCM** (AEAD) | Industry-standard authenticated encryption. Hardware-accelerated (AES-NI) in Go's `crypto/cipher`. |
| **Master Key** | `YILY_MASTER_KEY` (32 bytes) | 256-bit cryptographically secure key loaded from the environment. |
| **Nonce** | 12 bytes (96 bits) | Cryptographically random (`crypto/rand`), generated independently for each encryption. Nonce reuse with the same key must never occur. |
| **AAD** | `yily:<variable_id>:<version>` | Additional Authenticated Data binding ciphertext to its variable and version. |
| **Storage Format** | `v1:<nonce_b64>:<ciphertext_b64>` | Self-contained envelope with a scheme version prefix for seamless future migrations. |

---

## 3. Storage Format

Encrypted secrets are stored as a single serialized string in `VariableVersion.Value`:

```text
v1:<base64_nonce>:<base64_ciphertext_and_tag>
```

### Components:
* **`v1`**: The encryption envelope schema version.
* **`<base64_nonce>`**: Standard Base64 representation of the 12-byte random nonce.
* **`<base64_ciphertext_and_tag>`**: Standard Base64 representation of the encrypted payload and its 16-byte authentication tag.

---

## 4. Security Guarantees

1. **Confidentiality**: The plaintext cannot be decrypted without the `YILY_MASTER_KEY`.
2. **Integrity**: Any modification to the ciphertext, nonce, or authentication tag causes decryption to fail.
3. **Context Binding**: The ciphertext is cryptographically bound to its variable ID and version via AAD. Swapping ciphertext between different variables or versions in SQLite causes decryption to fail.

---

## 5. Encryption Lifecycle & Timing

To construct the deterministic AAD, `variable_id` and `version` must be determined **before** encryption occurs:

```text
1. Determine VariableID (UUID) and Version (int)
       ↓
2. Build AAD: "yily:<variable_id>:<version>"
       ↓
3. Generate 12-byte random Nonce
       ↓
4. AES-256-GCM Seal(nonce, plaintext, AAD)
       ↓
5. Format envelope: "v1:<nonce_b64>:<ciphertext_b64>"
       ↓
6. Persist to VariableVersion.Value in SQLite
```

---

## 6. Database Schema Adjustment

In `server/internal/domain/model/variable_version.go`:

```go
// Current:
Value string `gorm:"type:varchar(255)" json:"value"`

// Proposed:
Value string `gorm:"type:text;not null" json:"value"`
```

**Reason:** Long secrets (private keys, TLS certificates, long tokens) combined with encryption overhead (nonce, tag, Base64 encoding) easily exceed 255 characters. Changing to `type:text` avoids imposing an application-level 255-character storage limit.

---

## 7. Proposed Go Package API

A dedicated package `server/internal/crypto` using standard library primitives (`crypto/aes`, `crypto/cipher`, `crypto/rand`):

```go
package crypto

import "github.com/google/uuid"

// Encrypt encrypts a secret value using AES-256-GCM with contextual AAD.
// masterKey must be exactly 32 bytes.
func Encrypt(plaintext string, masterKey []byte, variableID uuid.UUID, version int) (string, error)

// Decrypt authenticates and decrypts an encrypted payload using AES-256-GCM.
// masterKey must be exactly 32 bytes.
func Decrypt(payload string, masterKey []byte, variableID uuid.UUID, version int) (string, error)
```

---

## 8. Next Steps

1. Update `VariableVersion.Value` GORM tag to `type:text;not null`.
2. Implement `server/internal/crypto` with unit tests covering:
   - Successful encryption and decryption round-trip.
   - Key length validation (must be exactly 32 bytes).
   - Corrupted ciphertext or modified nonce detection.
   - AAD mismatch detection (wrong variable ID or wrong version).
   - Malformed envelope handling.
3. Integrate encryption and decryption into `VariableService`.
4. Ensure plaintext secrets are never persisted, cached, or logged.
