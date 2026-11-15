package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"strings"
)

// recoveryCodeBytes is the entropy size in bytes (128 bits) of a recovery code.
const recoveryCodeBytes = 16

// GenerateRecoveryCode generates a 128-bit cryptographically random recovery
// code, formatted in uppercase hex groups of four for readability
// (e.g. "ABCD-EF01-2345-6789-ABCD-EF01-2345-6789"). The raw code is returned
// only once (to be handed to the Super Admin); only its hash is stored.
func GenerateRecoveryCode() (string, error) {
	buf := make([]byte, recoveryCodeBytes)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return formatRecoveryCode(strings.ToUpper(hex.EncodeToString(buf))), nil
}

// NormalizeRecoveryCode uppercases a recovery code and strips separators and
// whitespace, so a user can type it with or without dashes and in any case.
func NormalizeRecoveryCode(code string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '-', ' ', '\t', '\n', '\r', '\v', '\f':
			return -1
		}
		if r >= 'a' && r <= 'z' {
			return r - ('a' - 'A')
		}
		return r
	}, code)
}

// HashRecoveryCode hashes a normalized recovery code with Argon2id for storage.
func HashRecoveryCode(code string) (string, error) {
	return HashPassword(NormalizeRecoveryCode(code))
}

// VerifyRecoveryCode verifies a recovery code against its stored Argon2id hash.
func VerifyRecoveryCode(code, encodedHash string) (bool, error) {
	return VerifyPassword(NormalizeRecoveryCode(code), encodedHash)
}

// formatRecoveryCode inserts a dash every four characters.
func formatRecoveryCode(normalized string) string {
	var b strings.Builder
	b.Grow(len(normalized) + len(normalized)/4)
	for i, r := range normalized {
		if i > 0 && i%4 == 0 {
			b.WriteByte('-')
		}
		b.WriteRune(r)
	}
	return b.String()
}
