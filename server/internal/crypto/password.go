package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    uint32 = 3         // argonTime is the number of Argon2id iterations (time cost).
	argonMemory  uint32 = 64 * 1024 // argonMemory is the Argon2id memory cost in KiB (64 MiB).
	argonThreads uint8  = 4         // argonThreads is the Argon2id parallelism degree (lanes).
	argonKeyLen  uint32 = 32        // argonKeyLen is the derived Argon2id key length in bytes (256 bits).
	argonSaltLen int    = 16        // argonSaltLen is the random salt length in bytes (128 bits).
)

var (
	ErrEmptyPassword       = errors.New("password cannot be empty")              // ErrEmptyPassword is returned when an empty password is provided for hashing.
	ErrInvalidPasswordHash = errors.New("invalid argon2id password hash format") // ErrInvalidPasswordHash is returned when a stored PHC hash string is malformed.
)

// HashPassword hashes a plaintext password with Argon2id using a random 16-byte salt and returns its PHC string.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}

	salt := make([]byte, argonSaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64Salt, b64Hash), nil
}

// VerifyPassword parses an Argon2id PHC hash, recomputes the key with its parameters, and compares both in constant time.
func VerifyPassword(password, encodedHash string) (bool, error) {
	if password == "" {
		return false, nil
	}

	// Expected PHC format: $argon2id$v=19$m=<memory>,t=<time>,p=<threads>$<base64-salt>$<base64-hash>
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrInvalidPasswordHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, ErrInvalidPasswordHash
	}

	var memory, timeCost uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeCost, &threads); err != nil {
		return false, ErrInvalidPasswordHash
	}
	if memory == 0 || timeCost == 0 || threads == 0 {
		return false, ErrInvalidPasswordHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return false, ErrInvalidPasswordHash
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expectedHash) == 0 {
		return false, ErrInvalidPasswordHash
	}

	actualHash := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, uint32(len(expectedHash)))

	// Compare hashes in constant time to prevent timing side-channel leaks.
	return subtle.ConstantTimeCompare(expectedHash, actualHash) == 1, nil
}
