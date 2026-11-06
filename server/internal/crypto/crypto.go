package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"uuid"
)

const (
	versionPrefix = "v1"
	keySize       = 32
	nonceSize     = 12
)

var (
	ErrInvalidKeyLength = errors.New("master key must be exactly 32 bytes")
	ErrMalformedPayload = errors.New("malformed encrypted payload")
	ErrDecryptionFailed = errors.New("decryption failed: invalid ciphertext or authentication tag")
)

// Encrypt encrypts plaintext using AES-256-GCM and returns a versioned envelope
func Encrypt(plaintext string, masterKey []byte, variableID uuid.UUID, version int) (string, error) {
	if len(masterKey) != keySize {
		return "", ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	aad := buildAAD(variableID, version)
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), aad)

	nonceB64 := base64.StdEncoding.EncodeToString(nonce)
	ciphertextB64 := base64.StdEncoding.EncodeToString(ciphertext)

	return fmt.Sprintf("%s:%s:%s", versionPrefix, nonceB64, ciphertextB64), nil
}

// Decrypt authenticates and decrypts an envelope using AES-256-GCM.
func Decrypt(payload string, masterKey []byte, variableID uuid.UUID, version int) (string, error) {
	if len(masterKey) != keySize {
		return "", ErrInvalidKeyLength
	}

	parts := strings.Split(payload, ":")
	if len(parts) != 3 {
		return "", ErrMalformedPayload
	}

	nonce, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil || len(nonce) != nonceSize {
		return "", ErrMalformedPayload
	}

	ciphertext, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return "", ErrMalformedPayload
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < gcm.Overhead() {
		return "", ErrMalformedPayload
	}

	aad := buildAAD(variableID, version)
	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// buildAAD constructs the authenticated data bound to a variable and its version.
func buildAAD(variableID uuid.UUID, version int) []byte {
	return []byte(fmt.Sprintf("yily:%s:%d", variableID, version))
}
