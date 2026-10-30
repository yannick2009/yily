package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
)

const (
	sessionTokenBytesSize = 32      // sessionTokenBytesSize is the entropy size in bytes (256 bits) for session tokens.
	patTokenBytesSize     = 32      // patTokenBytesSize is the entropy size in bytes (256 bits) for Personal Access Tokens.
	PATPrefix             = "yily_" // PATPrefix is the mandatory prefix identifying YILY Personal Access Tokens.
)

var (
	ErrEmptyToken = errors.New("token cannot be empty") // ErrEmptyToken is returned when an empty token is provided.
)

// GenerateSessionToken generates a 256-bit cryptographically random session token and its SHA-256 hex digest for DB storage.
func GenerateSessionToken() (rawToken string, tokenHash string, err error) {
	buf := make([]byte, sessionTokenBytesSize)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", "", err
	}

	rawToken = base64.RawURLEncoding.EncodeToString(buf)
	tokenHash = HashToken(rawToken)
	return rawToken, tokenHash, nil
}

// GeneratePAT generates a 256-bit cryptographically random PAT prefixed with "yily_" and its SHA-256 hex digest for DB storage.
func GeneratePAT() (rawToken string, tokenHash string, err error) {
	buf := make([]byte, patTokenBytesSize)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", "", err
	}

	rawToken = PATPrefix + base64.RawURLEncoding.EncodeToString(buf)
	tokenHash = HashToken(rawToken)
	return rawToken, tokenHash, nil
}

// HashToken computes the SHA-256 hex digest of a raw token so plaintext tokens are never stored in the database.
func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
