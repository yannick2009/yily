package service

import (
	"errors"
	"unicode"
	"unicode/utf8"
)

// MaxNameLength is the maximum number of characters allowed in a name
// (project, environment, variable or tag). It matches the varchar(255) column
// size and bounds what ends up in audit records, LIKE filters and export file
// names.
const MaxNameLength = 255

var (
	// ErrNameTooLong is returned when a name exceeds MaxNameLength.
	ErrNameTooLong = errors.New("name must be at most 255 characters")
	// ErrInvalidNameChars is returned when a name contains characters that are not allowed.
	ErrInvalidNameChars = errors.New("name contains characters that are not allowed")
)

// validateName rejects names longer than MaxNameLength and names that contain
// anything other than printable characters (so control characters, tabs and
// newlines are excluded). It does not reject empty names; callers enforce that
// separately.
func validateName(name string) error {
	if utf8.RuneCountInString(name) > MaxNameLength {
		return ErrNameTooLong
	}
	for _, r := range name {
		if !unicode.IsPrint(r) {
			return ErrInvalidNameChars
		}
	}
	return nil
}

// validateVariableName enforces the stricter charset required for variable
// names, which are normalized to uppercase and exported as .env keys. Limiting
// them to letters, digits and underscores keeps the .env export unambiguous:
// characters such as "=" can never appear in a key.
func validateVariableName(name string) error {
	if utf8.RuneCountInString(name) > MaxNameLength {
		return ErrNameTooLong
	}
	for _, r := range name {
		if r != '_' && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return ErrInvalidNameChars
		}
	}
	return nil
}

// MaxEnvironmentDescriptionLength is the maximum number of characters allowed
// in an environment description.
const MaxEnvironmentDescriptionLength = 50

// ErrEnvironmentDescriptionTooLong is returned when a description exceeds the
// maximum allowed length.
var ErrEnvironmentDescriptionTooLong = errors.New("environment description must be at most 50 characters")

// validateDescription rejects a description longer than the maximum length.
func validateDescription(description string) error {
	if utf8.RuneCountInString(description) > MaxEnvironmentDescriptionLength {
		return ErrEnvironmentDescriptionTooLong
	}
	return nil
}
