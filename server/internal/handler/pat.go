package handler

import (
	"net/http"
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
		return NewHTTPResponse[any](c, http.StatusUnauthorized, services.ErrUnauthorized.Error(), nil)
	}

	var req createPATRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	var ttl time.Duration
	if req.ExpiresInDays != nil {
		if *req.ExpiresInDays <= 0 {
			return NewHTTPResponse[any](c, http.StatusBadRequest, services.ErrInvalidPATExpiration.Error(), nil)
		}
		ttl = time.Duration(*req.ExpiresInDays) * 24 * time.Hour
	}

	rawToken, pat, err := h.patService.Create(c.Request().Context(), user.ID, req.Name, ttl)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	resp := CreatePATResponse{
		ID:        pat.ID,
		UserID:    pat.UserID,
		Name:      pat.Name,
		Token:     rawToken,
		CreatedAt: pat.CreatedAt,
		ExpiresAt: pat.ExpiresAt,
	}

	return NewHTTPResponse(c, http.StatusCreated, "Personal access token created successfully", resp)
}

// GetAll handles GET /tokens to list the authenticated user's PAT metadata without exposing raw tokens or hashes.
func (h *patHandler) GetAll(c *echo.Context) error {
	user, ok := services.AuthenticatedUserFromContext(c.Request().Context())
	if !ok {
		return NewHTTPResponse[any](c, http.StatusUnauthorized, services.ErrUnauthorized.Error(), nil)
	}

	pats, err := h.patService.GetAll(c.Request().Context(), user.ID)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse(c, http.StatusOK, "Personal access tokens retrieved successfully", pats)
}

// Revoke handles DELETE /tokens/:id to revoke a PAT belonging to the authenticated user.
func (h *patHandler) Revoke(c *echo.Context) error {
	user, ok := services.AuthenticatedUserFromContext(c.Request().Context())
	if !ok {
		return NewHTTPResponse[any](c, http.StatusUnauthorized, services.ErrUnauthorized.Error(), nil)
	}

	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	if err := h.patService.Revoke(c.Request().Context(), user.ID, id); err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse[any](c, http.StatusOK, "Personal access token revoked successfully", nil)
}
