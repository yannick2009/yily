package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GenerateRecoveryCode(t *testing.T) {
	code, err := GenerateRecoveryCode()
	assert.NoError(t, err)

	assert.Regexp(t, `^[0-9A-F]{4}(-[0-9A-F]{4}){7}$`, code,
		"recovery code should be 8 groups of 4 uppercase hex chars separated by dashes")

	normalized := NormalizeRecoveryCode(code)
	assert.Regexp(t, `^[0-9A-F]{32}$`, normalized, "normalized code should be 32 uppercase hex chars")
}

func Test_GenerateRecoveryCode_Random(t *testing.T) {
	first, err := GenerateRecoveryCode()
	assert.NoError(t, err)
	second, err := GenerateRecoveryCode()
	assert.NoError(t, err)

	assert.NotEqual(t, first, second, "two recovery codes should differ")
}

func Test_NormalizeRecoveryCode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "already normalized", in: "ABCDEF0123456789ABCDEF0123456789", want: "ABCDEF0123456789ABCDEF0123456789"},
		{name: "lowercase", in: "abcdef0123456789abcdef0123456789", want: "ABCDEF0123456789ABCDEF0123456789"},
		{name: "with dashes", in: "ABCD-EF01-2345-6789-ABCD-EF01-2345-6789", want: "ABCDEF0123456789ABCDEF0123456789"},
		{name: "with whitespace", in: "ABCD EF01\n2345\t6789", want: "ABCDEF0123456789"},
		{name: "mixed case and separators", in: " ab-cd\nEF-01 ", want: "ABCDEF01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeRecoveryCode(tt.in))
		})
	}
}

func Test_formatRecoveryCode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "shorter than a group", in: "ABCD", want: "ABCD"},
		{name: "one char past a group", in: "ABCDE", want: "ABCD-E"},
		{name: "full code", in: "ABCDEF0123456789ABCDEF0123456789", want: "ABCD-EF01-2345-6789-ABCD-EF01-2345-6789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatRecoveryCode(tt.in))
		})
	}
}

func Test_HashRecoveryCode_Empty(t *testing.T) {
	_, err := HashRecoveryCode("")
	assert.ErrorIs(t, err, ErrEmptyPassword)
}

func Test_VerifyRecoveryCode(t *testing.T) {
	code, err := GenerateRecoveryCode()
	assert.NoError(t, err)

	hash, err := HashRecoveryCode(code)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	t.Run("correct code verifies", func(t *testing.T) {
		ok, err := VerifyRecoveryCode(code, hash)
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("case and separators are tolerated", func(t *testing.T) {
		ok, err := VerifyRecoveryCode(strings.ToLower(code), hash)
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("wrong code is rejected", func(t *testing.T) {
		ok, err := VerifyRecoveryCode("FFFF-FFFF-FFFF-FFFF-FFFF-FFFF-FFFF-FFFF", hash)
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("empty code returns false with no error", func(t *testing.T) {
		ok, err := VerifyRecoveryCode("", hash)
		assert.NoError(t, err)
		assert.False(t, ok)
	})
}
