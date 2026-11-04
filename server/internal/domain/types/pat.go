package types

// PATScope defines the permission level granted to a Personal Access Token.
type PATScope string

const (
	PATScopeRead  PATScope = "read"  // PATScopeRead allows read-only access to resources.
	PATScopeWrite PATScope = "write" // PATScopeWrite allows read and write access to resources (excludes token management, which is UI-only).
)

// IsValid reports whether the scope value is one of the defined PATScope constants.
func (s PATScope) IsValid() bool {
	switch s {
	case PATScopeRead, PATScopeWrite:
		return true
	default:
		return false
	}
}

// AllowsWrite reports whether the scope grants write access. A write scope implicitly grants read.
func (s PATScope) AllowsWrite() bool {
	return s == PATScopeWrite
}
