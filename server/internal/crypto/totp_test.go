package crypto

import (
	"encoding/base32"
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yannick2009/yily/internal/id"
)

// rfcSecret is the ASCII seed used by the RFC 4226/6238 test vectors, encoded
// the same way TOTP secrets are stored.
func rfcSecret() string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))
}

func Test_GenerateTOTPSecret(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	assert.NoError(t, err)
	assert.Len(t, secret, 32, "20 random bytes should encode to 32 Base32 characters")

	decoded, err := decodeTOTPSecret(secret)
	assert.NoError(t, err)
	assert.Len(t, decoded, TOTPSecretSize)

	other, err := GenerateTOTPSecret()
	assert.NoError(t, err)
	assert.NotEqual(t, secret, other, "two generated secrets should differ")
}

func Test_TOTPCode_RFC6238Vectors(t *testing.T) {
	secret := rfcSecret()

	// Appendix B vectors from RFC 6238 (HMAC-SHA1), truncated to six digits.
	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{name: "59", at: time.Unix(59, 0), want: "287082"},
		{name: "1111111109", at: time.Unix(1111111109, 0), want: "081804"},
		{name: "1111111111", at: time.Unix(1111111111, 0), want: "050471"},
		{name: "1234567890", at: time.Unix(1234567890, 0), want: "005924"},
		{name: "2000000000", at: time.Unix(2000000000, 0), want: "279037"},
		{name: "20000000000", at: time.Unix(20000000000, 0), want: "353130"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TOTPCode(secret, tt.at)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_hotp_RFC4226Vectors(t *testing.T) {
	key := []byte("12345678901234567890")

	// Appendix D vectors from RFC 4226.
	tests := []struct {
		name    string
		counter uint64
		want    string
	}{
		{name: "counter 0", counter: 0, want: "755224"},
		{name: "counter 1", counter: 1, want: "287082"},
		{name: "counter 2", counter: 2, want: "359152"},
		{name: "counter 3", counter: 3, want: "969429"},
		{name: "counter 4", counter: 4, want: "338314"},
		{name: "counter 5", counter: 5, want: "254676"},
		{name: "counter 6", counter: 6, want: "287922"},
		{name: "counter 7", counter: 7, want: "162583"},
		{name: "counter 8", counter: 8, want: "399871"},
		{name: "counter 9", counter: 9, want: "520489"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hotp(key, tt.counter))
		})
	}
}

func Test_NormalizeTOTPCode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain digits", in: "123456", want: "123456"},
		{name: "space separator", in: "123 456", want: "123456"},
		{name: "dash separator", in: "123-456", want: "123456"},
		{name: "surrounding whitespace", in: " 123456 ", want: "123456"},
		{name: "letters rejected", in: "12a456", want: ""},
		{name: "separators and letters", in: "12-34a6", want: ""},
		{name: "empty", in: "", want: ""},
		{name: "short code is not padded", in: "12345", want: "12345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeTOTPCode(tt.in))
		})
	}
}

func Test_ValidateTOTP(t *testing.T) {
	secret := rfcSecret()
	now := time.Unix(1111111111, 0).UTC()
	currentStep := now.Unix() / TOTPPeriod

	current, err := TOTPCode(secret, now)
	assert.NoError(t, err)
	previous, err := TOTPCode(secret, now.Add(-time.Duration(TOTPPeriod)*time.Second))
	assert.NoError(t, err)
	next, err := TOTPCode(secret, now.Add(time.Duration(TOTPPeriod)*time.Second))
	assert.NoError(t, err)
	twoStepsBack, err := TOTPCode(secret, now.Add(-2*time.Duration(TOTPPeriod)*time.Second))
	assert.NoError(t, err)
	twoStepsAhead, err := TOTPCode(secret, now.Add(2*time.Duration(TOTPPeriod)*time.Second))
	assert.NoError(t, err)

	t.Run("current code", func(t *testing.T) {
		step, ok := ValidateTOTP(secret, current, now)
		assert.True(t, ok)
		assert.Equal(t, currentStep, step)
	})

	t.Run("separators are ignored", func(t *testing.T) {
		_, ok := ValidateTOTP(secret, current[:3]+" "+current[3:], now)
		assert.True(t, ok)

		_, ok = ValidateTOTP(secret, current[:3]+"-"+current[3:], now)
		assert.True(t, ok)
	})

	t.Run("previous step within skew", func(t *testing.T) {
		step, ok := ValidateTOTP(secret, previous, now)
		assert.True(t, ok)
		assert.Equal(t, currentStep-1, step)
	})

	t.Run("next step within skew", func(t *testing.T) {
		step, ok := ValidateTOTP(secret, next, now)
		assert.True(t, ok)
		assert.Equal(t, currentStep+1, step)
	})

	t.Run("two steps back is rejected", func(t *testing.T) {
		_, ok := ValidateTOTP(secret, twoStepsBack, now)
		assert.False(t, ok)
	})

	t.Run("two steps ahead is rejected", func(t *testing.T) {
		_, ok := ValidateTOTP(secret, twoStepsAhead, now)
		assert.False(t, ok)
	})

	t.Run("malformed codes are rejected", func(t *testing.T) {
		for _, code := range []string{"", "12345", "1234567", "abcdef", "12a456"} {
			_, ok := ValidateTOTP(secret, code, now)
			assert.False(t, ok, "code %q should be rejected", code)
		}
	})

	t.Run("invalid secret is rejected", func(t *testing.T) {
		_, ok := ValidateTOTP("not-base32!", current, now)
		assert.False(t, ok)
	})
}

func Test_TOTPURI(t *testing.T) {
	const secret = "JBSWY3DPEHPK3PXP"
	raw := TOTPURI("Yily", "admin+test@example.com", secret)

	parsed, err := url.Parse(raw)
	assert.NoError(t, err)
	assert.Equal(t, "otpauth", parsed.Scheme)
	assert.Equal(t, "totp", parsed.Host)
	assert.Contains(t, parsed.Path, "Yily:admin+test@example.com")

	query := parsed.Query()
	assert.Equal(t, secret, query.Get("secret"))
	assert.Equal(t, "Yily", query.Get("issuer"))
	assert.Equal(t, "SHA1", query.Get("algorithm"))
	assert.Equal(t, "6", query.Get("digits"))
	assert.Equal(t, "30", query.Get("period"))
}

func Test_decodeTOTPSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantLen int
		wantErr error
	}{
		{name: "canonical", secret: "JBSWY3DPEHPK3PXP", wantLen: 10},
		{name: "lowercase", secret: "jbswy3dpehpk3pxp", wantLen: 10},
		{name: "spaces", secret: "JBSW Y3DP EHPK 3PXP", wantLen: 10},
		{name: "padding", secret: "JBSWY3DPEHPK3PXP====", wantLen: 10},
		{name: "empty", secret: "", wantErr: ErrInvalidTOTPSecret},
		{name: "invalid characters", secret: "!!!!", wantErr: ErrInvalidTOTPSecret},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := decodeTOTPSecret(tt.secret)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, key, tt.wantLen)
		})
	}
}

func Test_EncryptDecryptTOTPSecret_RoundTrip(t *testing.T) {
	masterKey := []byte("12345678901234567890123456789012")
	userID := id.New(id.PrefixUser)

	tests := []struct {
		name   string
		secret string
	}{
		{name: "generated seed", secret: mustGenerateSecret(t)},
		{name: "fixed seed", secret: "JBSWY3DPEHPK3PXP"},
		{name: "empty string", secret: ""},
		{name: "unicode", secret: "секрет-🔐"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := EncryptTOTPSecret(tt.secret, masterKey, userID)
			assert.NoError(t, err)
			assert.NotEmpty(t, payload)
			if tt.secret != "" {
				assert.NotContains(t, payload, tt.secret)
			}

			decrypted, err := DecryptTOTPSecret(payload, masterKey, userID)
			assert.NoError(t, err)
			assert.Equal(t, tt.secret, decrypted)
		})
	}
}

func Test_EncryptTOTPSecret_Failures(t *testing.T) {
	userID := id.New(id.PrefixUser)

	t.Run("short master key", func(t *testing.T) {
		_, err := EncryptTOTPSecret("JBSWY3DPEHPK3PXP", []byte("short-key"), userID)
		assert.ErrorIs(t, err, ErrInvalidKeyLength)
	})

	t.Run("nil master key", func(t *testing.T) {
		_, err := EncryptTOTPSecret("JBSWY3DPEHPK3PXP", nil, userID)
		assert.ErrorIs(t, err, ErrInvalidKeyLength)
	})
}

func Test_DecryptTOTPSecret_Failures(t *testing.T) {
	masterKey := []byte("12345678901234567890123456789012")
	userID := id.New(id.PrefixUser)
	otherID := id.New(id.PrefixUser)

	payload, err := EncryptTOTPSecret("JBSWY3DPEHPK3PXP", masterKey, userID)
	assert.NoError(t, err)

	t.Run("wrong user ID", func(t *testing.T) {
		_, err := DecryptTOTPSecret(payload, masterKey, otherID)
		assert.ErrorIs(t, err, ErrDecryptionFailed)
	})

	t.Run("wrong master key", func(t *testing.T) {
		_, err := DecryptTOTPSecret(payload, []byte("abcdefghijklmnopqrstuvwxyz123456"), userID)
		assert.ErrorIs(t, err, ErrDecryptionFailed)
	})

	t.Run("short master key", func(t *testing.T) {
		_, err := DecryptTOTPSecret(payload, []byte("short-key"), userID)
		assert.ErrorIs(t, err, ErrInvalidKeyLength)
	})

	t.Run("malformed payload", func(t *testing.T) {
		for _, malformed := range []string{"", "v1", "v1:nonce", "v1:a:b:c"} {
			_, err := DecryptTOTPSecret(malformed, masterKey, userID)
			assert.ErrorIs(t, err, ErrMalformedPayload, "payload %q should be malformed", malformed)
		}
	})

	t.Run("tampered ciphertext", func(t *testing.T) {
		parts := strings.Split(payload, ":")
		assert.Len(t, parts, 3)

		ciphertext, err := base64.StdEncoding.DecodeString(parts[2])
		assert.NoError(t, err)

		tampered := parts[0] + ":" + parts[1] + ":" + base64.StdEncoding.EncodeToString(flipBit(ciphertext))
		_, err = DecryptTOTPSecret(tampered, masterKey, userID)
		assert.ErrorIs(t, err, ErrDecryptionFailed)
	})
}

// mustGenerateSecret returns a fresh seed or fails the test.
func mustGenerateSecret(t *testing.T) string {
	t.Helper()
	secret, err := GenerateTOTPSecret()
	assert.NoError(t, err)
	return secret
}
