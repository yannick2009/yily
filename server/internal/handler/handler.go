package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/repository"
)

// HTTPResponse is the standard JSON envelope used for every API response.
type HTTPResponse[T any] struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
}

// NewHTTPResponse sends a standardized JSON response.
func NewHTTPResponse[T any](c *echo.Context, status int, message string, data T) error {
	return c.JSON(status, HTTPResponse[T]{Status: status, Message: message, Data: data})
}

// bindAndValidate binds the request body into req, then validates it using the
// registered validator. On failure it replies 400 and returns false.
func bindAndValidate(c *echo.Context, req any) bool {
	if err := c.Bind(req); err != nil {
		_ = NewHTTPResponse[any](c, http.StatusBadRequest, "invalid request body", nil)
		return false
	}
	if err := c.Validate(req); err != nil {
		_ = NewHTTPResponse[any](c, http.StatusBadRequest, err.Error(), nil)
		return false
	}
	return true
}

// httpStatus maps a service error to an HTTP status code.
// Sentinel "not found" errors become 404, anything else becomes 500.
func httpStatus(err error) int {
	switch {
	case errors.Is(err, repository.ErrVariableNameAlreadyExists),
		errors.Is(err, repository.ErrTagNameAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, repository.ErrTagNotInProject):
		return http.StatusBadRequest
	case errors.Is(err, repository.ErrProjectNotFound),
		errors.Is(err, repository.ErrEnvironmentNotFound),
		errors.Is(err, repository.ErrVariableNotFound),
		errors.Is(err, repository.ErrVariableVersionNotFound),
		errors.Is(err, repository.ErrTagNotFound):
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
		_ = NewHTTPResponse[any](c, http.StatusBadRequest, "invalid "+name, nil)
		return uuid.Nil, false
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
