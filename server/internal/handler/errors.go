package handler

import (
	"errors"

	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/env"
	"github.com/yannick2009/yily/internal/repository"
	"github.com/yannick2009/yily/internal/services"
)

// ErrorCode is a stable machine-readable identifier for a known error. It
// travels alongside the human-readable English message so UI clients can
// localise the error, while non-browser callers (CLI, scripts) still get a
// meaningful message by default.
type ErrorCode string

// Generic codes. These don't map to a sentinel error; handlers emit them
// directly when a specific validation or I/O step fails outside the services
// layer (parsing a UUID, reading a multipart file, …).
const (
	CodeInternal           ErrorCode = "INTERNAL_ERROR"
	CodeBadRequest         ErrorCode = "BAD_REQUEST"
	CodeInvalidUUID        ErrorCode = "INVALID_UUID"
	CodeValidationFailed   ErrorCode = "VALIDATION_FAILED"
	CodeInvalidRequestBody ErrorCode = "INVALID_REQUEST_BODY"
	CodeFileParseFailed    ErrorCode = "FILE_PARSE_FAILED"
	CodeFileOpenFailed     ErrorCode = "FILE_OPEN_FAILED"
	CodeFileReadFailed     ErrorCode = "FILE_READ_FAILED"
)

// Codes tied to sentinel errors from the services / repository / env packages.
const (
	// Auth
	CodeAuthInvalidCredentials ErrorCode = "AUTH_INVALID_CREDENTIALS"
	CodeAuthUnauthorized       ErrorCode = "AUTH_UNAUTHORIZED"
	CodeAuthSessionExpired     ErrorCode = "AUTH_SESSION_EXPIRED"
	CodeAuthSessionRevoked     ErrorCode = "AUTH_SESSION_REVOKED"
	CodeAuthWeakPassword       ErrorCode = "AUTH_WEAK_PASSWORD"
	CodeAuthUserExists         ErrorCode = "AUTH_USER_ALREADY_EXISTS"
	CodeAuthUserNotFound       ErrorCode = "AUTH_USER_NOT_FOUND"

	// Personal Access Tokens
	CodeTokenExpired           ErrorCode = "TOKEN_EXPIRED"
	CodeTokenRevoked           ErrorCode = "TOKEN_REVOKED"
	CodeTokenNotFound          ErrorCode = "TOKEN_NOT_FOUND"
	CodeTokenInvalidName       ErrorCode = "TOKEN_INVALID_NAME"
	CodeTokenInvalidExpiration ErrorCode = "TOKEN_INVALID_EXPIRATION"
	CodeTokenInvalidScope      ErrorCode = "TOKEN_INVALID_SCOPE"
	CodeTokenWriteForbidden    ErrorCode = "TOKEN_WRITE_FORBIDDEN"

	// Resources
	CodeProjectNotFound         ErrorCode = "PROJECT_NOT_FOUND"
	CodeEnvironmentNotFound     ErrorCode = "ENVIRONMENT_NOT_FOUND"
	CodeEnvironmentDescTooLong  ErrorCode = "ENVIRONMENT_DESCRIPTION_TOO_LONG"
	CodeVariableNotFound        ErrorCode = "VARIABLE_NOT_FOUND"
	CodeVariableVersionNotFound ErrorCode = "VARIABLE_VERSION_NOT_FOUND"
	CodeVariableNameExists      ErrorCode = "VARIABLE_NAME_ALREADY_EXISTS"
	CodeTagNotFound             ErrorCode = "TAG_NOT_FOUND"
	CodeTagNameExists           ErrorCode = "TAG_NAME_ALREADY_EXISTS"
	CodeTagNotInProject         ErrorCode = "TAG_NOT_IN_PROJECT"

	// Import
	CodeFileUnsupportedType   ErrorCode = "FILE_UNSUPPORTED_TYPE"
	CodeFileInvalidJSON       ErrorCode = "FILE_INVALID_JSON"
	CodeFileInvalidYAML       ErrorCode = "FILE_INVALID_YAML"
	CodeFileInvalidEnvFormat  ErrorCode = "FILE_INVALID_ENV_FORMAT"
	CodeFileDuplicateVariable ErrorCode = "FILE_DUPLICATE_VARIABLE"
)

// sentinelCodes pairs every known sentinel error with its stable code.
// The lookup is done with errors.Is, so wrapped errors still match.
var sentinelCodes = []struct {
	err  error
	code ErrorCode
}{
	// Auth
	{services.ErrInvalidCredentials, CodeAuthInvalidCredentials},
	{services.ErrUnauthorized, CodeAuthUnauthorized},
	{services.ErrSessionExpired, CodeAuthSessionExpired},
	{services.ErrSessionRevoked, CodeAuthSessionRevoked},
	{services.ErrWeakPassword, CodeAuthWeakPassword},
	{repository.ErrUserAlreadyExists, CodeAuthUserExists},
	{repository.ErrUserNotFound, CodeAuthUserNotFound},

	// Tokens
	{services.ErrPATExpired, CodeTokenExpired},
	{services.ErrPATRevoked, CodeTokenRevoked},
	{repository.ErrPATNotFound, CodeTokenNotFound},
	{services.ErrInvalidPATName, CodeTokenInvalidName},
	{services.ErrInvalidPATExpiration, CodeTokenInvalidExpiration},
	{services.ErrInvalidPATScope, CodeTokenInvalidScope},
	{services.ErrPATWriteForbidden, CodeTokenWriteForbidden},

	// Resources
	{repository.ErrProjectNotFound, CodeProjectNotFound},
	{repository.ErrEnvironmentNotFound, CodeEnvironmentNotFound},
	{services.ErrEnvironmentDescriptionTooLong, CodeEnvironmentDescTooLong},
	{repository.ErrVariableNotFound, CodeVariableNotFound},
	{repository.ErrVariableVersionNotFound, CodeVariableVersionNotFound},
	{repository.ErrVariableNameAlreadyExists, CodeVariableNameExists},
	{repository.ErrTagNotFound, CodeTagNotFound},
	{repository.ErrTagNameAlreadyExists, CodeTagNameExists},
	{repository.ErrTagNotInProject, CodeTagNotInProject},

	// Import
	{types.ErrUnsupportedFileType, CodeFileUnsupportedType},
	{env.ErrInvalidJSONFormat, CodeFileInvalidJSON},
	{env.ErrInvalidYAMLFormat, CodeFileInvalidYAML},
	{env.ErrInvalidEnvFileFormat, CodeFileInvalidEnvFormat},
	{env.ErrDuplicateVariableName, CodeFileDuplicateVariable},
}

// codeFor returns the stable code for a known sentinel error, or an empty
// code if no mapping exists (the message still carries the English wording).
func codeFor(err error) ErrorCode {
	if err == nil {
		return ""
	}
	for _, entry := range sentinelCodes {
		if errors.Is(err, entry.err) {
			return entry.code
		}
	}
	return CodeInternal
}
