package handler

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/authctx"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/httputil"
	"github.com/yannick2009/yily/internal/id"
	"github.com/yannick2009/yily/internal/service"
)

// PATHandler defines the interface for Personal Access Token HTTP handlers.
type PATHandler interface {
	Create(c *echo.Context) error // Create handles POST /tokens to issue a new PAT.
	GetAll(c *echo.Context) error // GetAll handles GET /tokens to list PAT metadata.
	Me(c *echo.Context) error     // Me handles GET /tokens/me to return the PAT used to authenticate the request.
	Revoke(c *echo.Context) error // Revoke handles DELETE /tokens/:id to revoke a PAT.
	Rotate(c *echo.Context) error // Rotate handles POST /tokens/:id/rotate to replace an existing PAT.
}

// patHandler handles HTTP requests for Personal Access Tokens.
type patHandler struct {
	patService service.PATService
}

// NewPATHandler creates a new PATHandler.
func NewPATHandler(patService service.PATService) PATHandler {
	return &patHandler{patService: patService}
}

// createPATRequest represents the JSON body for creating a new PAT.
type createPATRequest struct {
	Name          string         `json:"name" validate:"required"`
	Scope         types.PATScope `json:"scope" validate:"required,oneof=read write"`
	ExpiresInDays *int           `json:"expires_in_days,omitempty"`
	ProjectID     *string        `json:"project_id,omitempty"`
	EnvironmentID *string        `json:"environment_id,omitempty"`
}

// rotatePATRequest represents the JSON body for rotating an existing PAT.
type rotatePATRequest struct {
	ExpiresInDays *int `json:"expires_in_days,omitempty"`
}

// CreatePATResponse represents the one-time creation response that includes the raw "yily_<token>" value.
type CreatePATResponse struct {
	ID              id.ID          `json:"id"`                         // Unique identifier of the created PAT
	UserID          id.ID          `json:"user_id"`                    // ID of the User owning the PAT
	Name            string         `json:"name"`                       // Human-readable label of the PAT
	Scope           types.PATScope `json:"scope"`                      // Permission scope granted to the PAT
	ProjectID       *id.ID         `json:"project_id,omitempty"`       // Optional project restriction (nil means all projects)
	EnvironmentID   *id.ID         `json:"environment_id,omitempty"`   // Optional environment restriction (nil means all environments in scope)
	ProjectName     string         `json:"project_name,omitempty"`     // Resolved project name (computed on read, never persisted)
	EnvironmentName string         `json:"environment_name,omitempty"` // Resolved environment name (computed on read, never persisted)
	Token           string         `json:"token"`                      // Raw "yily_<token>" returned only once at creation time
	CreatedAt       time.Time      `json:"created_at"`                 // Creation timestamp of the PAT
	ExpiresAt       *time.Time     `json:"expires_at,omitempty"`       // Expiration timestamp of the PAT (nil means never expires)
}

// newCreatePATResponse constructs a CreatePATResponse from a model.PersonalAccessToken and its raw token.
func newCreatePATResponse(rawToken string, pat *model.PersonalAccessToken) CreatePATResponse {
	return CreatePATResponse{
		ID:              pat.ID,
		UserID:          pat.UserID,
		Name:            pat.Name,
		Scope:           pat.Scope,
		ProjectID:       pat.ProjectID,
		EnvironmentID:   pat.EnvironmentID,
		ProjectName:     pat.ProjectName,
		EnvironmentName: pat.EnvironmentName,
		Token:           rawToken,
		CreatedAt:       pat.CreatedAt,
		ExpiresAt:       pat.ExpiresAt,
	}
}

// parseOptionalID parses an optional string pointer into an *id.ID, returning nil for nil or empty strings, and validating the prefix if present.
func parseOptionalID(raw *string, prefix string) (*id.ID, error) {
	if raw == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := id.Parse(prefix, trimmed)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

// Create handles POST /tokens to generate a new PAT for the authenticated user and return its raw token once.
func (h *patHandler) Create(c *echo.Context) error {
	user, ok := authctx.AuthenticatedUserFromContext(c.Request().Context())
	if !ok {
		return respondError(c, service.ErrUnauthorized)
	}

	var req createPATRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	ttl, err := resolvePATTTL(req.ExpiresInDays)
	if err != nil {
		return respondError(c, err)
	}

	projectID, err := parseOptionalID(req.ProjectID, id.PrefixProject)
	if err != nil {
		return respondErrorCode(c, http.StatusBadRequest, CodeInvalidID, "invalid project_id")
	}

	environmentID, err := parseOptionalID(req.EnvironmentID, id.PrefixEnvironment)
	if err != nil {
		return respondErrorCode(c, http.StatusBadRequest, CodeInvalidID, "invalid environment_id")
	}

	rawToken, pat, err := h.patService.Create(c.Request().Context(), user.ID, req.Name, req.Scope, ttl, projectID, environmentID)
	if err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse(c, http.StatusCreated, "Personal access token created successfully", newCreatePATResponse(rawToken, pat))
}

// GetAll handles GET /tokens to list the authenticated user's PAT metadata without exposing raw tokens or hashes.
func (h *patHandler) GetAll(c *echo.Context) error {
	user, ok := authctx.AuthenticatedUserFromContext(c.Request().Context())
	if !ok {
		return respondError(c, service.ErrUnauthorized)
	}

	pats, err := h.patService.GetAll(c.Request().Context(), user.ID)
	if err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse(c, http.StatusOK, "Personal access tokens retrieved successfully", pats)
}

// Me handles GET /tokens/me to return metadata about the PAT used to authenticate the request.
// It is reachable by any valid PAT (read or write scope) and never returns the raw token or its hash.
func (h *patHandler) Me(c *echo.Context) error {
	pat, ok := authctx.AuthenticatedPATFromContext(c.Request().Context())
	if !ok {
		return respondErrorCode(c, http.StatusForbidden, CodeTokenRequired, "this endpoint requires a personal access token")
	}

	return httputil.NewHTTPResponse(c, http.StatusOK, "Personal access token retrieved successfully", pat)
}

// Revoke handles DELETE /tokens/:id to revoke a PAT belonging to the authenticated user.
func (h *patHandler) Revoke(c *echo.Context) error {
	user, ok := authctx.AuthenticatedUserFromContext(c.Request().Context())
	if !ok {
		return respondError(c, service.ErrUnauthorized)
	}

	id, ok := idParam(c, "id", id.PrefixPAT)
	if !ok {
		return nil
	}

	if err := h.patService.Revoke(c.Request().Context(), user.ID, id); err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse[any](c, http.StatusOK, "Personal access token revoked successfully", nil)
}

// Revoke handles DELETE /tokens/:id to revoke a PAT belonging to the authenticated user.
func (h *patHandler) Rotate(c *echo.Context) error {
	user, ok := authctx.AuthenticatedUserFromContext(c.Request().Context())
	if !ok {
		return httputil.NewHTTPResponse[any](c, http.StatusUnauthorized, service.ErrUnauthorized.Error(), nil)
	}

	id, ok := idParam(c, "id", id.PrefixPAT)
	if !ok {
		return nil
	}

	var req rotatePATRequest
	if c.Request().ContentLength > 0 && !bindAndValidate(c, &req) {
		return nil
	}

	ttl, err := resolvePATTTL(req.ExpiresInDays)
	if err != nil {
		return httputil.NewHTTPResponse[any](c, http.StatusBadRequest, err.Error(), nil)
	}

	rawToken, pat, err := h.patService.Rotate(c.Request().Context(), user.ID, id, ttl)
	if err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse(c, http.StatusCreated, "Personal access token rotated successfully", newCreatePATResponse(rawToken, pat))
}
func resolvePATTTL(expiresInDays *int) (time.Duration, error) {
	if expiresInDays == nil {
		return 0, nil
	}

	days := *expiresInDays
	if slices.Contains(service.AllowedPATTTLDays, days) {
		return time.Duration(days) * 24 * time.Hour, nil
	}

	return 0, service.ErrInvalidPATExpiration
}
