package handler

import (
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/services"
)

// PATHandler defines the interface for Personal Access Token HTTP handlers.
type PATHandler interface {
	Create(c *echo.Context) error // Create handles POST /tokens to issue a new PAT.
	GetAll(c *echo.Context) error // GetAll handles GET /tokens to list PAT metadata.
	Revoke(c *echo.Context) error // Revoke handles DELETE /tokens/:id to revoke a PAT.
	Rotate(c *echo.Context) error // Rotate handles POST /tokens/:id/rotate to replace an existing PAT.
}

// patHandler handles HTTP requests for Personal Access Tokens.
type patHandler struct {
	patService services.PATService
}

// NewPATHandler creates a new PATHandler.
func NewPATHandler(patService services.PATService) PATHandler {
	return &patHandler{patService: patService}
}

// createPATRequest represents the JSON body for creating a new PAT.
type createPATRequest struct {
	Name          string `json:"name" validate:"required"`
	ExpiresInDays *int   `json:"expires_in_days,omitempty"`
}

// rotatePATRequest represents the JSON body for rotating an existing PAT.
type rotatePATRequest struct {
	ExpiresInDays *int `json:"expires_in_days,omitempty"`
}

// CreatePATResponse represents the one-time creation response that includes the raw "yily_<token>" value.
type CreatePATResponse struct {
	ID        uuid.UUID `json:"id"`         // Unique identifier of the created PAT
	UserID    uuid.UUID `json:"user_id"`    // ID of the User owning the PAT
	Name      string    `json:"name"`       // Human-readable label of the PAT
	Token     string    `json:"token"`      // Raw "yily_<token>" returned only once at creation time
	CreatedAt time.Time `json:"created_at"` // Creation timestamp of the PAT
	ExpiresAt time.Time `json:"expires_at"` // Expiration timestamp of the PAT
}

// Create handles POST /tokens to generate a new PAT for the authenticated user and return its raw token once.
func (h *patHandler) Create(c *echo.Context) error {
	user, ok := services.AuthenticatedUserFromContext(c.Request().Context())
	if !ok {
		return respondError(c, services.ErrUnauthorized)
	}

	var req createPATRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	ttl, err := resolvePATTTL(req.ExpiresInDays)
	if err != nil {
		return respondError(c, err)
	}

	rawToken, pat, err := h.patService.Create(c.Request().Context(), user.ID, req.Name, ttl)
	if err != nil {
		return respondError(c, err)
	}

	resp := CreatePATResponse{
		ID:        pat.ID,
		UserID:    pat.UserID,
		Name:      pat.Name,
		Token:     rawToken,
		CreatedAt: pat.CreatedAt,
		ExpiresAt: pat.ExpiresAt,
	}

	return httputil.NewHTTPResponse(c, http.StatusCreated, "Personal access token created successfully", resp)
}

// GetAll handles GET /tokens to list the authenticated user's PAT metadata without exposing raw tokens or hashes.
func (h *patHandler) GetAll(c *echo.Context) error {
	user, ok := services.AuthenticatedUserFromContext(c.Request().Context())
	if !ok {
		return respondError(c, services.ErrUnauthorized)
	}

	pats, err := h.patService.GetAll(c.Request().Context(), user.ID)
	if err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse(c, http.StatusOK, "Personal access tokens retrieved successfully", pats)
}

// Revoke handles DELETE /tokens/:id to revoke a PAT belonging to the authenticated user.
func (h *patHandler) Revoke(c *echo.Context) error {
	user, ok := services.AuthenticatedUserFromContext(c.Request().Context())
	if !ok {
		return respondError(c, services.ErrUnauthorized)
	}

	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	if err := h.patService.Revoke(c.Request().Context(), user.ID, id); err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse[any](c, http.StatusOK, "Personal access token revoked successfully", nil)
}

// Rotate handles POST /tokens/:id/rotate to revoke an existing PAT and issue a replacement under the same name, returning the raw replacement token once.
func (h *patHandler) Rotate(c *echo.Context) error {
	user, ok := services.AuthenticatedUserFromContext(c.Request().Context())
	if !ok {
		return NewHTTPResponse[any](c, http.StatusUnauthorized, services.ErrUnauthorized.Error(), nil)
	}

	id, ok := uuidParam(c, "id")
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
		return httputil.NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	resp := CreatePATResponse{
		ID:        pat.ID,
		UserID:    pat.UserID,
		Name:      pat.Name,
		Token:     rawToken,
		CreatedAt: pat.CreatedAt,
		ExpiresAt: pat.ExpiresAt,
	}

	return httputil.NewHTTPResponse(c, http.StatusCreated, "Personal access token rotated successfully", resp)
}

// resolvePATTTL converts an optional expires_in_days value to a time.Duration, enforcing the fixed preset list (7, 30, 60, 90). A nil value means "use the service default".
func resolvePATTTL(expiresInDays *int) (time.Duration, error) {
	if expiresInDays == nil {
		return 0, nil
	}
	days := *expiresInDays
	if slices.Contains(services.AllowedPATTTLDays, days) {
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return 0, services.ErrInvalidPATExpiration
}
