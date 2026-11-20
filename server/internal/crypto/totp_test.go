package crypto

import (
	"encoding/base32"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/yannick2009/yily/internal/id"
)

// TestTOTPMatchesRFC6238Vectors checks the HMAC-SHA1/6-digit profile against
// the Appendix B test vectors from RFC 6238 (truncated to six digits).
func TestTOTPMatchesRFC6238Vectors(t *testing.T) {
	// The RFC uses the ASCII string "12345678901234567890" as the seed.
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))

	tests := []struct {
		unix int64
		want string
	}{
		{59, "287082"},
		{1111111109, "081804"},
		{1111111111, "050471"},
		{1234567890, "005924"},
		{2000000000, "279037"},
		{20000000000, "353130"},
	}

	for _, tt := range tests {
		got, err := TOTPCode(secret, time.Unix(tt.unix, 0))
		if err != nil {
			t.Fatalf("TOTPCode at %d: %v", tt.unix, err)
		}
		if got != tt.want {
			t.Errorf("TOTPCode at %d = %q, want %q", tt.unix, got, tt.want)
		}
	}
}

func TestValidateTOTPAcceptsSkewAndNormalizes(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("GenerateTOTPSecret: %v", err)
	}

	now := time.Unix(1_700_000_001, 0) // one second past a time-step boundary
	current, err := TOTPCode(secret, now)
	if err != nil {
		t.Fatalf("TOTPCode: %v", err)
	}

	step, ok := ValidateTOTP(secret, current, now)
	if !ok {
		t.Fatal("current code was rejected")
	}
	if want := now.Unix() / TOTPPeriod; step != want {
		t.Errorf("step = %d, want %d", step, want)
	}

	if _, ok := ValidateTOTP(secret, current[:3]+" "+current[3:], now); !ok {
		t.Error("code with a space separator was rejected")
	}
	if _, ok := ValidateTOTP(secret, current[:3]+"-"+current[3:], now); !ok {
		t.Error("code with a dash separator was rejected")
	}

	previous, err := TOTPCode(secret, now.Add(-TOTPPeriod*time.Second))
	if err != nil {
		t.Fatalf("TOTPCode previous: %v", err)
	}
	if _, ok := ValidateTOTP(secret, previous, now); !ok {
		t.Error("code from the previous step was rejected despite skew")
	}

	tooOld, err := TOTPCode(secret, now.Add(-2*TOTPPeriod*time.Second))
	if err != nil {
		t.Fatalf("TOTPCode too old: %v", err)
	}
	if _, ok := ValidateTOTP(secret, tooOld, now); ok {
		t.Error("code from two steps back was accepted")
	}

	for _, malformed := range []string{"", "12345", "1234567", "abcdef", "12a456"} {
		if _, ok := ValidateTOTP(secret, malformed, now); ok {
			t.Errorf("malformed code %q was accepted", malformed)
		}
	}
}

func TestTOTPURI(t *testing.T) {
	const secret = "JBSWY3DPEHPK3PXP"
	raw := TOTPURI("Yily", "admin@example.com", secret)

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse URI: %v", err)
	}
	if parsed.Scheme != "otpauth" || parsed.Host != "totp" {
		t.Fatalf("unexpected URI scheme/host: %s", raw)
	}
	if !strings.Contains(parsed.Path, "Yily:admin@example.com") {
		t.Errorf("path = %q, want it to contain the issuer and account", parsed.Path)
	}

	query := parsed.Query()
	for key, want := range map[string]string{
		"secret":    secret,
		"issuer":    "Yily",
		"algorithm": "SHA1",
		"digits":    "6",
		"period":    "30",
	} {
		if got := query.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestTOTPSecretEncryptionIsBoundToUser(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = 0x2a
	}

	userID := id.New(id.PrefixUser)
	otherID := id.New(id.PrefixUser)
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("GenerateTOTPSecret: %v", err)
	}

	payload, err := EncryptTOTPSecret(secret, key, userID)
	if err != nil {
		t.Fatalf("EncryptTOTPSecret: %v", err)
	}

	got, err := DecryptTOTPSecret(payload, key, userID)
	if err != nil {
		t.Fatalf("DecryptTOTPSecret: %v", err)
	}
	if got != secret {
		t.Errorf("round trip = %q, want %q", got, secret)
	}

	if _, err := DecryptTOTPSecret(payload, key, otherID); !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("decrypting under another user ID returned %v, want ErrDecryptionFailed", err)
	}
}
