package crypto

import (
	"bytes"
	"encoding/base64"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yannick2009/yily/internal/id"
)

func Test_Encrypt(t *testing.T) {
	tests := []struct {
		name       string
		plaintext  string
		masterKey  []byte
		variableID id.ID
		version    int
		Err        error
	}{
		{
			name:       "empty master key",
			variableID: "test-variable-id",
			plaintext:  "Hello, World!",
			masterKey:  []byte(""),
			version:    1,
			Err:        ErrInvalidKeyLength,
		},
		{
			name:       "short master key",
			variableID: "test-variable-id",
			plaintext:  "Hello, World!",
			masterKey:  []byte("short-key"),
			version:    1,
			Err:        ErrInvalidKeyLength,
		},
		{
			name:       "nil master key",
			variableID: "test-variable-id",
			plaintext:  "Hello, World!",
			masterKey:  nil,
			version:    1,
			Err:        ErrInvalidKeyLength,
		},
		{
			name:       "31 byte master key",
			variableID: "test-variable-id",
			plaintext:  "Hello, World!",
			masterKey:  make([]byte, 31),
			version:    1,
			Err:        ErrInvalidKeyLength,
		},
		{
			name:       "33 byte master key",
			variableID: "test-variable-id",
			plaintext:  "Hello, World!",
			masterKey:  make([]byte, 33),
			version:    1,
			Err:        ErrInvalidKeyLength,
		},
		{
			name:       "encrypts successfully",
			variableID: "test-variable-id",
			plaintext:  "Hello, World!",
			masterKey:  []byte("12345678901234567890123456789012"),
			version:    1,
			Err:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := Encrypt(tt.plaintext, tt.masterKey, tt.variableID, tt.version)

			if tt.Err != nil {
				assert.ErrorIs(t, err, tt.Err)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, encrypted)

			parts := strings.Split(encrypted, ":")
			assert.Equal(t, 3, len(parts), "Encrypted payload should have three parts separated by ':'")
			assert.Equal(t, versionPrefix, parts[0], "First part of encrypted payload should be the version prefix")
			assert.NotEmpty(t, parts[1], "Second part of encrypted payload (nonce) should not be empty")
			assert.NotEmpty(t, parts[2], "Third part of encrypted payload (ciphertext) should not be empty")
		})
	}
}

func Test_Decrypt(t *testing.T) {
	successID := id.ID("test-variable-id")
	successMasterKey := []byte("12345678901234567890123456789012")
	successPayload, err := Encrypt("Hello, World!", successMasterKey, successID, 1)
	assert.NoError(t, err, "Setup for Test_Decrypt failed: Encrypt returned an error")

	validNonceB64 := base64.StdEncoding.EncodeToString(make([]byte, nonceSize))
	wrongLengthNonceB64 := base64.StdEncoding.EncodeToString(make([]byte, nonceSize+4))
	shortCiphertextB64 := base64.StdEncoding.EncodeToString(make([]byte, 5))
	validCiphertextB64 := base64.StdEncoding.EncodeToString(make([]byte, 32))

	// Derive tampered payloads from a valid envelope so the tests below
	// exercise the GCM authentication failure path, not the structure checks.
	successParts := strings.Split(successPayload, ":")
	successNonce, err := base64.StdEncoding.DecodeString(successParts[1])
	assert.NoError(t, err, "setup: decode nonce: %v", err)

	successCiphertext, err := base64.StdEncoding.DecodeString(successParts[2])
	assert.NoError(t, err, "setup: decode ciphertext: %v", err)

	encode := func(nonce, ciphertext []byte) string {
		return versionPrefix + ":" + base64.StdEncoding.EncodeToString(nonce) + ":" + base64.StdEncoding.EncodeToString(ciphertext)
	}

	// Each tampered payload changes exactly one thing in a valid envelope.
	tamperedNonce := flipBit(successNonce)                                    // flip one bit of the nonce
	flippedCiphertext := flipBit(successCiphertext)                           // flip one bit of the ciphertext
	truncatedTag := bytes.Clone(successCiphertext[:len(successCiphertext)-1]) // drop the last byte of the tag
	appendedBytes := append(bytes.Clone(successCiphertext), 0x00)             // add an extra trailing byte

	v2Payload := "v2" + strings.TrimPrefix(successPayload, versionPrefix) // same envelope, unsupported version

	tests := []struct {
		name       string
		payload    string
		masterKey  []byte
		variableID id.ID
		version    int
		Err        error
	}{
		{
			name:       "empty master key",
			payload:    "v1:nonce:ciphertext",
			masterKey:  []byte(""),
			variableID: "test-variable-id",
			version:    1,
			Err:        ErrInvalidKeyLength,
		},
		{
			name:       "short master key",
			payload:    "v1:nonce:ciphertext",
			masterKey:  []byte("short-key"),
			variableID: "test-variable-id",
			version:    1,
			Err:        ErrInvalidKeyLength,
		},
		{
			name:       "nil master key",
			payload:    "v1:nonce:ciphertext",
			masterKey:  nil,
			variableID: "test-variable-id",
			version:    1,
			Err:        ErrInvalidKeyLength,
		},
		{
			name:       "31 byte master key",
			payload:    "v1:nonce:ciphertext",
			masterKey:  make([]byte, 31),
			variableID: "test-variable-id",
			version:    1,
			Err:        ErrInvalidKeyLength,
		},
		{
			name:       "33 byte master key",
			payload:    "v1:nonce:ciphertext",
			masterKey:  make([]byte, 33),
			variableID: "test-variable-id",
			version:    1,
			Err:        ErrInvalidKeyLength,
		},
		{
			name:       "empty string payload",
			payload:    "",
			masterKey:  successMasterKey,
			variableID: successID,
			version:    1,
			Err:        ErrMalformedPayload,
		},
		{
			name:       "one part payload",
			payload:    "v1",
			masterKey:  successMasterKey,
			variableID: successID,
			version:    1,
			Err:        ErrMalformedPayload,
		},
		{
			name:       "two parts payload",
			payload:    "v1:nonce",
			masterKey:  successMasterKey,
			variableID: successID,
			version:    1,
			Err:        ErrMalformedPayload,
		},
		{
			name:       "four parts payload",
			payload:    "v1:a:b:c",
			masterKey:  successMasterKey,
			variableID: successID,
			version:    1,
			Err:        ErrMalformedPayload,
		},
		{
			name:       "unsupported envelope version",
			payload:    v2Payload,
			masterKey:  successMasterKey,
			variableID: successID,
			version:    1,
			Err:        ErrMalformedPayload,
		},
		{
			name:       "invalid nonce base64",
			payload:    "v1:!!!:" + validCiphertextB64,
			masterKey:  successMasterKey,
			variableID: successID,
			version:    1,
			Err:        ErrMalformedPayload,
		},
		{
			name:       "nonce with wrong length",
			payload:    "v1:" + wrongLengthNonceB64 + ":" + validCiphertextB64,
			masterKey:  successMasterKey,
			variableID: successID,
			version:    1,
			Err:        ErrMalformedPayload,
		},
		{
			name:       "invalid ciphertext base64",
			payload:    "v1:" + validNonceB64 + ":!!!",
			masterKey:  successMasterKey,
			variableID: successID,
			version:    1,
			Err:        ErrMalformedPayload,
		},
		{
			name:       "ciphertext shorter than GCM overhead",
			payload:    "v1:" + validNonceB64 + ":" + shortCiphertextB64,
			masterKey:  successMasterKey,
			variableID: successID,
			version:    1,
			Err:        ErrMalformedPayload,
		},
		{
			name:       "decryption fails with wrong key",
			payload:    successPayload,
			masterKey:  []byte("abcdefghijklmnopqrstuvwxyz123456"),
			variableID: successID,
			version:    1,
			Err:        ErrDecryptionFailed,
		},
		{
			name:       "decryption fails with wrong AAD version",
			payload:    successPayload,
			masterKey:  successMasterKey,
			variableID: successID,
			version:    2,
			Err:        ErrDecryptionFailed,
		},
		{
			name:       "decryption fails with wrong AAD variable ID",
			payload:    successPayload,
			masterKey:  successMasterKey,
			variableID: "different-variable-id",
			version:    1,
			Err:        ErrDecryptionFailed,
		},
		{
			name:       "tampered nonce",
			payload:    encode(tamperedNonce, successCiphertext),
			masterKey:  successMasterKey,
			variableID: successID,
			version:    1,
			Err:        ErrDecryptionFailed,
		},
		{
			name:       "flipped ciphertext bit",
			payload:    encode(successNonce, flippedCiphertext),
			masterKey:  successMasterKey,
			variableID: successID,
			version:    1,
			Err:        ErrDecryptionFailed,
		},
		{
			name:       "truncated tag",
			payload:    encode(successNonce, truncatedTag),
			masterKey:  successMasterKey,
			variableID: successID,
			version:    1,
			Err:        ErrDecryptionFailed,
		},
		{
			name:       "appended bytes",
			payload:    encode(successNonce, appendedBytes),
			masterKey:  successMasterKey,
			variableID: successID,
			version:    1,
			Err:        ErrDecryptionFailed,
		},
		{
			name:       "valid payload",
			payload:    successPayload,
			masterKey:  successMasterKey,
			variableID: successID,
			version:    1,
			Err:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decrypted, err := Decrypt(tt.payload, tt.masterKey, tt.variableID, tt.version)

			if tt.Err != nil {
				assert.ErrorIs(t, err, tt.Err)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, decrypted)
			assert.Equal(t, "Hello, World!", decrypted, "Decrypted text does not match the original plaintext")
		})
	}
}

func Test_EncryptDecrypt_RoundTrip(t *testing.T) {
	masterKey := []byte("12345678901234567890123456789012")
	variableID := id.ID("roundtrip-variable-id")
	version := 3

	tests := []struct {
		name      string
		plaintext string
	}{
		{name: "empty string", plaintext: ""},
		{name: "ASCII", plaintext: "Hello, World!"},
		{name: "UTF-8 with emoji", plaintext: "sécurité 🔐 秘密 🤫"},
		{name: "multi-line", plaintext: "line one\nline two\nline three"},
		{name: "contains separators and base64 chars", plaintext: "a:b/c+d==?"},
		{name: "large value", plaintext: strings.Repeat("a", 100*1024)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := Encrypt(tt.plaintext, masterKey, variableID, version)
			assert.NoError(t, err)

			// Envelope format: v1:<base64 nonce>:<base64 ciphertext>.
			parts := strings.Split(payload, ":")
			assert.Equal(t, 3, len(parts), "envelope should have three parts separated by ':'")
			assert.Equal(t, versionPrefix, parts[0], "first part should be the version prefix")

			nonce, err := base64.StdEncoding.DecodeString(parts[1])
			assert.NoError(t, err)
			assert.Equal(t, nonceSize, len(nonce), "nonce should decode to exactly 12 bytes")

			ciphertext, err := base64.StdEncoding.DecodeString(parts[2])
			assert.NoError(t, err)
			assert.NotEmpty(t, ciphertext)

			// The plaintext must never leak into the envelope or the ciphertext.
			if tt.plaintext != "" {
				assert.NotContains(t, payload, tt.plaintext)
				assert.NotContains(t, string(ciphertext), tt.plaintext)
			}

			// Round-trip: decrypting yields the original value back.
			decrypted, err := Decrypt(payload, masterKey, variableID, version)
			assert.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func Test_Encrypt_RandomNonce(t *testing.T) {
	masterKey := []byte("12345678901234567890123456789012")
	variableID := id.ID("random-nonce-id")

	first, err := Encrypt("same plaintext", masterKey, variableID, 1)
	assert.NoError(t, err)
	second, err := Encrypt("same plaintext", masterKey, variableID, 1)
	assert.NoError(t, err)

	assert.NotEqual(t, first, second, "two encryptions of the same value should produce different envelopes")

	firstParts := strings.Split(first, ":")
	secondParts := strings.Split(second, ":")
	assert.NotEqual(t, firstParts[1], secondParts[1], "nonces should differ between encryptions")
	assert.NotEqual(t, firstParts[2], secondParts[2], "ciphertexts should differ between encryptions")
}

func Test_buildAAD(t *testing.T) {
	tests := []struct {
		name       string
		variableID id.ID
		version    int
		want       []byte
	}{
		{
			name:       "builds AAD correctly",
			variableID: "test-variable-id",
			version:    1,
			want:       []byte("yily:test-variable-id:1"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAAD(tt.variableID, tt.version)

			assert.Equal(t, tt.want, got, "AAD does not match expected value")

			parts := strings.Split(string(got), ":")

			assert.Equal(t, 3, len(parts), "AAD should have three parts separated by ':'")
			assert.Equal(t, "yily", parts[0], "First part of AAD should be 'yily'")
			assert.Equal(t, string(tt.variableID), parts[1], "Second part of AAD should be the variable ID")
			assert.Equal(t, strconv.Itoa(tt.version), parts[2], "Third part of AAD should be the version number")
		})
	}
}

// flipBit returns a copy of b with one bit of the first byte flipped. It is
// used to simulate a tampered payload while keeping the same length, so the
// Decrypt tests can assert that GCM authentication rejects the change.
func flipBit(b []byte) []byte {
	out := bytes.Clone(b)
	out[0] ^= 0x01
	return out
}
