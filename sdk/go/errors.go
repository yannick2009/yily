package yily

import (
	"errors"
	"fmt"
)

var (
	ErrMissingBaseURL       = errors.New("yily: base URL is required")                         // ErrMissingBaseURL is returned when the base URL is missing
	ErrMissingToken         = errors.New("yily: token is required")                            // ErrMissingToken is returned when PAT is missing
	ErrMissingEnvironmentID = errors.New("yily: environment ID is required")                   // ErrMissingEnvironmentID is returned when the environment ID is missing
	ErrInvalidTarget        = errors.New("yily: target must be a non-nil pointer to a struct") // ErrInvalidTarget is returned when Load receives anything other than a non-nil pointer to a struct
	ErrVariableNotFound     = errors.New("yily: variable not found")                           // ErrVariableNotFound is returned when requested variable does not exist
	ErrUnsupportedType      = errors.New("yily: unsupported field type")                       // ErrUnsupportedType is returned when struct field type is not supported for decoding
)

var (
	ErrUnauthorized = errors.New("yily: unauthorized") // ErrUnauthorized indicates an authentication failure (HTTP 401)
	ErrNotFound     = errors.New("yily: not found")    // ErrNotFound indicates the requested resource was not found (HTTP 404)
)

// APIError represents an error response returned by the Yily API
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("yily: api error (status %d, code %s): %s", e.StatusCode, e.Code, e.Message)
	}

	return fmt.Sprintf("yily: api error (status %d): %s", e.StatusCode, e.Message)
}
