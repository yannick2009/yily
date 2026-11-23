package crypto

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GenerateSessionToken(t *testing.T) {
	raw, hash, err := GenerateSessionToken()
	assert.NoError(t, err)

	assert.Len(t, raw, 43, "raw session token should be 43 chars (32 random bytes, raw Base64URL, no padding)")

	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	assert.NoError(t, err)
	assert.Equal(t, sessionTokenBytesSize, len(decoded), "raw token should decode to 32 bytes")

	assert.Regexp(t, "^[0-9a-f]{64}$", hash, "hash should be 64 lowercase hex chars")
	assert.Equal(t, hash, HashToken(raw), "HashToken(raw) should equal the returned hash")
}

func Test_GeneratePAT(t *testing.T) {
	raw, hash, err := GeneratePAT()
	assert.NoError(t, err)

	assert.True(t, strings.HasPrefix(raw, PATPrefix), "PAT should start with 'yily_'")

	suffix := strings.TrimPrefix(raw, PATPrefix)
	assert.Len(t, suffix, 32, "PAT suffix should be 32 chars (24 random bytes, raw Base64URL)")

	decoded, err := base64.RawURLEncoding.DecodeString(suffix)
	assert.NoError(t, err)
	assert.Equal(t, patTokenBytesSize, len(decoded), "PAT suffix should decode to 24 bytes")

	assert.Regexp(t, "^[0-9a-f]{64}$", hash, "hash should be 64 lowercase hex chars")
	assert.Equal(t, hash, HashToken(raw), "HashToken(raw) should equal the returned hash")
}

func Test_GenerateTokens_Unique(t *testing.T) {
	const count = 2000

	sessions := make(map[string]struct{}, count)
	for range count {
		raw, _, err := GenerateSessionToken()
		assert.NoError(t, err)
		if _, exists := sessions[raw]; exists {
			t.Fatalf("duplicate session token generated: %s", raw)
		}
		sessions[raw] = struct{}{}
	}

	pats := make(map[string]struct{}, count)
	for range count {
		raw, _, err := GeneratePAT()
		assert.NoError(t, err)
		if _, exists := pats[raw]; exists {
			t.Fatalf("duplicate PAT generated: %s", raw)
		}
		pats[raw] = struct{}{}
	}
}

func Test_HashToken_KnownVector(t *testing.T) {
	const want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	assert.Equal(t, want, HashToken("hello"), "SHA-256 of \"hello\" should match the known digest")
}

func Test_GenerateTokens_URLSafe(t *testing.T) {
	const count = 100

	for range count {
		session, _, err := GenerateSessionToken()
		assert.NoError(t, err)
		assert.True(t, isBase64URL(session), "session token %q contains non-URL-safe characters", session)

		pat, _, err := GeneratePAT()
		assert.NoError(t, err)
		assert.True(t, isBase64URL(pat), "PAT %q contains non-URL-safe characters", pat)
	}
}

func Test_GenerateCSRFToken(t *testing.T) {
	token, err := GenerateCSRFToken()
	assert.NoError(t, err)

	assert.Len(t, token, 43, "CSRF token should be 43 chars (32 random bytes, raw Base64URL, no padding)")

	decoded, err := base64.RawURLEncoding.DecodeString(token)
	assert.NoError(t, err)
	assert.Equal(t, csrfTokenBytesSize, len(decoded), "CSRF token should decode to 32 bytes")

	assert.True(t, isBase64URL(token), "CSRF token %q contains non-URL-safe characters", token)
}

// isBase64URL reports whether s contains only the Base64URL alphabet
// (A-Z, a-z, 0-9, '-' and '_').
func isBase64URL(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return false
		}
	}
	return true
}
