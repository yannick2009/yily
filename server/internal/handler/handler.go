package handler

import (
	"errors"
	"net/http"
	"strings"
	"uuid"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/env"
	"github.com/yannick2009/yily/internal/httputil"
	"github.com/yannick2009/yily/internal/repository"
	"github.com/yannick2009/yily/internal/service"
)

// respondError sends a standardized error response, pairing a stable code
// (looked up from the sentinel error) with the original English message.
func respondError(c *echo.Context, err error) error {
	status := httpStatus(err)
	return c.JSON(status, httputil.HTTPResponse[any]{
		Status:  status,
		Code:    string(codeFor(err)),
		Message: err.Error(),
	})
}

// respondErrorCode is for non-sentinel error paths where we want a stable code
// anyway (parsing a multipart form, parsing a UUID path param, …).
func respondErrorCode(c *echo.Context, status int, code ErrorCode, message string) error {
	return c.JSON(status, httputil.HTTPResponse[any]{
		Status:  status,
		Code:    string(code),
		Message: message,
	})
}

// bindAndValidate binds the request body into req, then validates it using the
// registered validator. On failure it replies 400 and returns false.
func bindAndValidate(c *echo.Context, req any) bool {
	if err := c.Bind(req); err != nil {
		_ = respondErrorCode(c, http.StatusBadRequest, CodeInvalidRequestBody, "invalid request body")
		return false
	}
	if err := c.Validate(req); err != nil {
		_ = respondErrorCode(c, http.StatusBadRequest, CodeValidationFailed, err.Error())
		return false
	}
	return true
}

// httpStatus maps a service error to an HTTP status code.
// Sentinel "not found" errors become 404, anything else becomes 500.
func httpStatus(err error) int {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials),
		errors.Is(err, service.ErrUnauthorized),
		errors.Is(err, service.ErrSessionExpired),
		errors.Is(err, service.ErrSessionRevoked),
		errors.Is(err, service.ErrPATExpired),
		errors.Is(err, service.ErrPATRevoked):
		return http.StatusUnauthorized
	case errors.Is(err, repository.ErrVariableNameAlreadyExists),
		errors.Is(err, repository.ErrTagNameAlreadyExists),
		errors.Is(err, repository.ErrUserAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, service.ErrPATWriteForbidden):
		return http.StatusForbidden
	case errors.Is(err, repository.ErrTagNotInProject),
		errors.Is(err, env.ErrInvalidJSONFormat),
		errors.Is(err, env.ErrInvalidYAMLFormat),
		errors.Is(err, env.ErrInvalidEnvFileFormat),
		errors.Is(err, types.ErrUnsupportedFileType),
		errors.Is(err, env.ErrDuplicateVariableName),
		errors.Is(err, service.ErrWeakPassword),
		errors.Is(err, service.ErrInvalidPATName),
		errors.Is(err, service.ErrInvalidPATExpiration),
		errors.Is(err, service.ErrInvalidPATScope),
		errors.Is(err, service.ErrEnvironmentDescriptionTooLong),
		errors.Is(err, service.ErrInvalidVariableVersion),
		errors.Is(err, service.ErrDuplicateVariableSelection):
		return http.StatusBadRequest
	case errors.Is(err, repository.ErrProjectNotFound),
		errors.Is(err, repository.ErrEnvironmentNotFound),
		errors.Is(err, repository.ErrVariableNotFound),
		errors.Is(err, repository.ErrVariableVersionNotFound),
		errors.Is(err, repository.ErrTagNotFound),
		errors.Is(err, repository.ErrUserNotFound),
		errors.Is(err, repository.ErrSessionNotFound),
		errors.Is(err, repository.ErrPATNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// uuidParam parses the named path parameter as a UUID.
// On failure it replies 400 and returns false.
func uuidParam(c *echo.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		_ = respondErrorCode(c, http.StatusBadRequest, CodeInvalidUUID, "invalid "+name)
		return uuid.Nil(), false
	}
	return id, true
}

// parseTagIDs parses a comma-separated list of UUIDs from the "tag_ids" query
// parameter. It returns an empty slice when the parameter is absent or empty.
func parseTagIDs(c *echo.Context) ([]uuid.UUID, error) {
	raw := strings.TrimSpace(c.QueryParam("tag_ids"))
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	ids := make([]uuid.UUID, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := uuid.Parse(part)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
