package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yannick2009/yily/internal/id"
)

const (
	// TOTPPeriod is the time step length in seconds. 30 seconds is the de-facto
	// default understood by every authenticator app.
	TOTPPeriod = 30
	// TOTPDigits is the number of digits in a generated code.
	TOTPDigits = 6
	// TOTPSecretSize is the seed length in bytes. RFC 4226 recommends at least
	// 128 bits; 160 bits matches the HMAC-SHA1 block-derived size used by the
	// RFC 6238 test vectors.
	TOTPSecretSize = 20
	// TOTPSkew is the number of adjacent time steps accepted on each side of the
	// current one, covering small clock differences between server and phone.
	TOTPSkew = 1
	// TOTPIssuer is the issuer label written into provisioning URIs.
	TOTPIssuer = "Yily"
)

var (
	// ErrInvalidTOTPSecret is returned when a TOTP secret is not valid Base32.
	ErrInvalidTOTPSecret = errors.New("invalid TOTP secret")
	// ErrInvalidTOTPCode is returned when a code is malformed (not exactly six digits).
	ErrInvalidTOTPCode = errors.New("invalid TOTP code")
)

// totpEncoding is unpadded uppercase Base32, the representation used by
// authenticator apps and otpauth:// URIs.
var totpEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// GenerateTOTPSecret returns a fresh random Base32-encoded TOTP seed.
func GenerateTOTPSecret() (string, error) {
	buf := make([]byte, TOTPSecretSize)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return totpEncoding.EncodeToString(buf), nil
}

// TOTPCode computes the code for a Base32 secret at the given instant.
func TOTPCode(secret string, at time.Time) (string, error) {
	key, err := decodeTOTPSecret(secret)
	if err != nil {
		return "", err
	}
	return hotp(key, uint64(at.Unix()/TOTPPeriod)), nil
}

// ValidateTOTP checks a user-supplied code against the current time step and
// its immediate neighbours. It returns the matched step and true on success,
// or zero and false otherwise. Comparing every candidate with a constant-time
// comparison keeps the check independent of how many digits matched.
func ValidateTOTP(secret, code string, at time.Time) (int64, bool) {
	normalized := NormalizeTOTPCode(code)
	if len(normalized) != TOTPDigits {
		return 0, false
	}

	key, err := decodeTOTPSecret(secret)
	if err != nil {
		return 0, false
	}

	current := at.Unix() / TOTPPeriod
	for offset := int64(TOTPSkew); offset >= -int64(TOTPSkew); offset-- {
		step := current + offset
		if step < 0 {
			continue
		}
		expected := hotp(key, uint64(step))
		if subtle.ConstantTimeCompare([]byte(expected), []byte(normalized)) == 1 {
			return step, true
		}
	}

	return 0, false
}

// NormalizeTOTPCode strips spaces and dashes from a code and rejects anything
// that is not a digit. An empty result means the input is malformed.
func NormalizeTOTPCode(code string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(code) {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-':
			// Separators are cosmetic: "123 456" and "123-456" are both accepted.
		default:
			return ""
		}
	}
	return b.String()
}

// TOTPURI builds the otpauth:// provisioning URI consumed by authenticator
// apps. SHA1/6 digits/30 seconds are the defaults every app supports.
func TOTPURI(issuer, account, secret string) string {
	label := url.PathEscape(issuer + ":" + account)
	params := url.Values{}
	params.Set("secret", secret)
	params.Set("issuer", issuer)
	params.Set("algorithm", "SHA1")
	params.Set("digits", strconv.Itoa(TOTPDigits))
	params.Set("period", strconv.Itoa(TOTPPeriod))
	return "otpauth://totp/" + label + "?" + params.Encode()
}

// EncryptTOTPSecret encrypts a TOTP seed with the master key, binding the
// ciphertext to the owning user so it cannot be transplanted between accounts.
func EncryptTOTPSecret(secret string, masterKey []byte, userID id.ID) (string, error) {
	return EncryptWithAAD(secret, masterKey, buildTOTPAAD(userID))
}

// DecryptTOTPSecret authenticates and decrypts a TOTP seed belonging to userID.
func DecryptTOTPSecret(payload string, masterKey []byte, userID id.ID) (string, error) {
	return DecryptWithAAD(payload, masterKey, buildTOTPAAD(userID))
}

// buildTOTPAAD constructs the additional authenticated data for a TOTP seed.
func buildTOTPAAD(userID id.ID) []byte {
	return fmt.Appendf(nil, "yily:totp:%s", userID)
}

// decodeTOTPSecret normalizes and decodes a Base32 TOTP seed.
func decodeTOTPSecret(secret string) ([]byte, error) {
	normalized := strings.ToUpper(strings.TrimSpace(secret))
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.TrimRight(normalized, "=")
	if normalized == "" {
		return nil, ErrInvalidTOTPSecret
	}

	key, err := totpEncoding.DecodeString(normalized)
	if err != nil || len(key) == 0 {
		return nil, ErrInvalidTOTPSecret
	}
	return key, nil
}

// hotp implements RFC 4226 dynamic truncation for one counter value.
func hotp(key []byte, counter uint64) string {
	var counterBytes [8]byte
	binary.BigEndian.PutUint64(counterBytes[:], counter)

	// HMAC-SHA1 is the default TOTP profile in RFC 6238 and the only one every
	// authenticator app implements. HMAC-SHA1 is not affected by the collision
	// attacks on plain SHA-1 and remains safe for this MAC construction.
	mac := hmac.New(sha1.New, key)
	mac.Write(counterBytes[:])
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])

	return fmt.Sprintf("%0*d", TOTPDigits, value%1_000_000)
}
