package crypto

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/argon2"
)

func Test_HashPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	assert.NoError(t, err)

	parts := strings.Split(hash, "$")
	assert.Equal(t, 6, len(parts), "PHC string should have six parts separated by '$'")
	assert.Equal(t, "", parts[0])
	assert.Equal(t, "argon2id", parts[1])
	assert.Equal(t, "v=19", parts[2])
	assert.Equal(t, "m=65536,t=3,p=4", parts[3])

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	assert.NoError(t, err)
	assert.Equal(t, argonSaltLen, len(salt), "salt should decode to 16 bytes")

	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	assert.NoError(t, err)
	assert.Equal(t, int(argonKeyLen), len(key), "hash should decode to 32 bytes")
}

func Test_HashPassword_RandomSalt(t *testing.T) {
	first, err := HashPassword("same password")
	assert.NoError(t, err)
	second, err := HashPassword("same password")
	assert.NoError(t, err)

	assert.NotEqual(t, first, second, "two hashes of the same password should differ (random salt)")
}

func Test_VerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct password")
	assert.NoError(t, err)

	t.Run("correct password verifies", func(t *testing.T) {
		ok, err := VerifyPassword("correct password", hash)
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("wrong password is rejected", func(t *testing.T) {
		ok, err := VerifyPassword("wrong password", hash)
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("empty password returns false with no error", func(t *testing.T) {
		ok, err := VerifyPassword("", hash)
		assert.NoError(t, err)
		assert.False(t, ok)
	})
}

func Test_VerifyPassword_Malformed(t *testing.T) {
	validSalt := base64.RawStdEncoding.EncodeToString(make([]byte, argonSaltLen))
	validHash := base64.RawStdEncoding.EncodeToString(make([]byte, int(argonKeyLen)))

	tests := []struct {
		name string
		hash string
	}{
		{name: "too few parts", hash: "$argon2id$v=19$m=65536,t=3,p=4$" + validSalt},
		{name: "too many parts", hash: "$argon2id$v=19$m=65536,t=3,p=4$" + validSalt + "$" + validHash + "$extra"},
		{name: "wrong algorithm", hash: "$argon2i$v=19$m=65536,t=3,p=4$" + validSalt + "$" + validHash},
		{name: "bad version", hash: "$argon2id$v=XX$m=65536,t=3,p=4$" + validSalt + "$" + validHash},
		{name: "bad parameters", hash: "$argon2id$v=19$m=abc,t=3,p=4$" + validSalt + "$" + validHash},
		{name: "zero memory", hash: "$argon2id$v=19$m=0,t=3,p=4$" + validSalt + "$" + validHash},
		{name: "zero time", hash: "$argon2id$v=19$m=65536,t=0,p=4$" + validSalt + "$" + validHash},
		{name: "zero threads", hash: "$argon2id$v=19$m=65536,t=3,p=0$" + validSalt + "$" + validHash},
		{name: "invalid salt base64", hash: "$argon2id$v=19$m=65536,t=3,p=4$!!!$" + validHash},
		{name: "invalid hash base64", hash: "$argon2id$v=19$m=65536,t=3,p=4$" + validSalt + "$!!!"},
		{name: "empty salt", hash: "$argon2id$v=19$m=65536,t=3,p=4$$" + validHash},
		{name: "empty hash", hash: "$argon2id$v=19$m=65536,t=3,p=4$" + validSalt + "$"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := VerifyPassword("some password", tt.hash)
			assert.ErrorIs(t, err, ErrInvalidPasswordHash)
			assert.False(t, ok)
		})
	}
}

func Test_VerifyPassword_CustomParameters(t *testing.T) {
	// Parameters that differ from the package defaults (64 MiB, t=3, p=4), to
	// prove the verifier reads m, t and p from the stored PHC string.
	const (
		customMemory  uint32 = 32 * 1024
		customTime    uint32 = 2
		customThreads uint8  = 2
	)

	password := "correct password"
	salt := make([]byte, argonSaltLen)
	for i := range salt {
		salt[i] = byte(i)
	}

	hash := argon2.IDKey([]byte(password), salt, customTime, customMemory, customThreads, argonKeyLen)
	phc := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, customMemory, customTime, customThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash))

	ok, err := VerifyPassword(password, phc)
	assert.NoError(t, err)
	assert.True(t, ok, "the verifier should read m, t and p from the stored PHC string")

	ok, err = VerifyPassword("wrong password", phc)
	assert.NoError(t, err)
	assert.False(t, ok)
}

func Test_IsStrongPassword(t *testing.T) {
	tests := []struct {
		password string
		want     bool
	}{
		{password: "short", want: false},
		{password: "alllowercase", want: false},
		{password: "ALLUPPERCASE", want: false},
		{password: "1234567890", want: false},
		{password: "NoDigitsHere!", want: false},
		{password: "nouppercase1!", want: false},
		{password: "NOLOWERCASE1!", want: false},
		{password: "ValidPassword1!", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			got := IsStrongPassword(tt.password)
			assert.Equal(t, tt.want, got)
		})
	}
}
