// Package id generates and validates the public identifiers used by Yily.
// An identifier is a three-letter type trigram followed by twelve base62
// characters, with no separator: PRJx9boixbYI9by. The trigram identifies the
// resource kind, and the random suffix carries about 71 bits of entropy.
package id

import (
	"crypto/rand"
	"errors"
)

// ID is the public identifier of a Yily resource.
type ID string

// String returns the identifier as a plain string.
func (i ID) String() string { return string(i) }

// Type trigrams. Every identifier starts with exactly one of these.
const (
	PrefixProject     = "PRJ" // PrefixProject identifies a project.
	PrefixEnvironment = "ENV" // PrefixEnvironment identifies an environment.
	PrefixVariable    = "VAR" // PrefixVariable identifies a variable.
	PrefixVersion     = "VER" // PrefixVersion identifies a variable version.
	PrefixTag         = "TAG" // PrefixTag identifies a tag.
	PrefixUser        = "USR" // PrefixUser identifies a user.
	PrefixSession     = "SES" // PrefixSession identifies a session.
	PrefixPAT         = "PAT" // PrefixPAT identifies a personal access token.
	PrefixAudit       = "AUD" // PrefixAudit identifies an audit event.
)

const (
	prefixLength = 3
	randomLength = 12
	totalLength  = prefixLength + randomLength
	alphabet     = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

// ErrInvalidID is returned when a value is not a well-formed identifier or
// does not carry the expected type trigram.
var ErrInvalidID = errors.New("invalid identifier")

// New returns a new identifier carrying the given type trigram.
func New(prefix string) ID {
	if len(prefix) != prefixLength {
		panic("id: prefix must be exactly three characters")
	}
	for i := range prefixLength {
		if prefix[i] < 'A' || prefix[i] > 'Z' {
			panic("id: prefix must contain only upper-case ASCII letters")
		}
	}

	buf := make([]byte, prefixLength, totalLength)
	copy(buf, prefix)
	buf = append(buf, randomSuffix()...)
	return ID(buf)
}

// Parse validates raw as an identifier with the expected type trigram.
func Parse(prefix, raw string) (ID, error) {
	if !Match(prefix, raw) {
		return "", ErrInvalidID
	}
	return ID(raw), nil
}

// Match reports whether raw is a well-formed identifier with the given prefix.
func Match(prefix, raw string) bool {
	return len(prefix) == prefixLength && valid(raw) && raw[:prefixLength] == prefix
}

// valid reports whether raw is a well-formed identifier of any type.
func valid(raw string) bool {
	if len(raw) != totalLength {
		return false
	}
	for i := range prefixLength {
		if raw[i] < 'A' || raw[i] > 'Z' {
			return false
		}
	}
	for i := prefixLength; i < totalLength; i++ {
		if !isBase62(raw[i]) {
			return false
		}
	}
	return true
}

// randomSuffix returns twelve uniformly random base62 characters.
func randomSuffix() []byte {
	suffix := make([]byte, 0, randomLength)
	var buf [32]byte
	for len(suffix) < randomLength {
		if _, err := rand.Read(buf[:]); err != nil {
			panic("id: cannot read random bytes: " + err.Error())
		}
		for _, b := range buf {
			// Masking a uniform byte with 63 yields a uniform value in [0, 64);
			// 62 and 63 are rejected to keep the 62-character alphabet uniform.
			v := b & 63
			if int(v) >= len(alphabet) {
				continue
			}
			suffix = append(suffix, alphabet[v])
			if len(suffix) == randomLength {
				break
			}
		}
	}
	return suffix
}

// isBase62 reports whether c is an alphanumeric ASCII character.
func isBase62(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
}
