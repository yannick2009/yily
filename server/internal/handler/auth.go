package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/services"
)

const (
	SessionCookieName = "yily_session" // SessionCookieName is the HTTP cookie name storing the raw session token.
)

// AuthHandler defines the interface for Super Admin authentication and session HTTP handlers.
type AuthHandler interface {
	Setup(c *echo.Context) error
	Login(c *echo.Context) error
	Logout(c *echo.Context) error
	Me(c *echo.Context) error
	Status(c *echo.Context) error
}

// authHandler handles HTTP requests for authentication and server-side sessions.
type authHandler struct {
	authService services.AuthService
	audit       services.AuditService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService services.AuthService, audit services.AuditService) AuthHandler {
	return &authHandler{authService: authService, audit: audit}
}

// setupRequest represents the JSON body for creating the initial Super Admin account.
type setupRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// loginRequest represents the JSON body for authenticating the Super Admin.
type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// Setup handles POST /auth/setup to create the initial Super Admin account with name, email, and strong password.
func (h *authHandler) Setup(c *echo.Context) error {
	var req setupRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	user, err := h.authService.Setup(c.Request().Context(), req.Name, req.Email, req.Password)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse(c, http.StatusCreated, "Super Admin created successfully", user)
}

// Login handles POST /auth/login to authenticate the Super Admin, rotate any prior session, and set the HttpOnly session cookie.
func (h *authHandler) Login(c *echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return NewHTTPResponse[any](c, http.StatusBadRequest, "invalid request body", nil)
	}
	if strings.TrimSpace(req.Email) == "" || req.Password == "" {
		return NewHTTPResponse[any](c, http.StatusUnauthorized, services.ErrInvalidCredentials.Error(), nil)
	}

	var existingToken string
	if cookie, err := c.Cookie(SessionCookieName); err == nil && cookie != nil {
		existingToken = cookie.Value
	}

	rawToken, session, user, err := h.authService.Login(c.Request().Context(), req.Email, req.Password, existingToken)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}
	if user == nil {
		return NewHTTPResponse[any](c, http.StatusInternalServerError, "login returned no user", nil)
	}

	maxAge := int(session.ExpiresAt.Sub(session.CreatedAt).Seconds())
	if maxAge <= 0 {
		maxAge = int(services.DefaultSessionTTL.Seconds())
	}

	http.SetCookie(c.Response(), &http.Cookie{
		Name:     SessionCookieName,
		Value:    rawToken,
		Path:     "/",
		Expires:  session.ExpiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   isHTTPSRequest(c),
		SameSite: http.SameSiteLaxMode,
	})

	_ = h.audit.RecordAs(c.Request().Context(), services.AuditActor{Type: types.ActorTypeUser, ID: user.ID, Name: user.Name}, types.ActionSignedIn, "", uuid.Nil, "")

	return NewHTTPResponse(c, http.StatusOK, "Login successful", user)
}

// Logout handles POST /auth/logout to revoke the current server-side session and expire the client cookie.
func (h *authHandler) Logout(c *echo.Context) error {
	var rawToken string
	if cookie, err := c.Cookie(SessionCookieName); err == nil && cookie != nil {
		rawToken = cookie.Value
	}

	err := h.authService.Logout(c.Request().Context(), rawToken)

	http.SetCookie(c.Response(), &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isHTTPSRequest(c),
		SameSite: http.SameSiteLaxMode,
	})

	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	_ = h.audit.Record(c.Request().Context(), types.ActionSignedOut, "", uuid.Nil, "")

	return NewHTTPResponse[any](c, http.StatusOK, "Logout successful", nil)
}

// Me handles GET /auth/me to return the authenticated Super Admin from the request context.
func (h *authHandler) Me(c *echo.Context) error {
	user, ok := services.AuthenticatedUserFromContext(c.Request().Context())
	if !ok {
		return NewHTTPResponse[any](c, http.StatusUnauthorized, services.ErrUnauthorized.Error(), nil)
	}

	return NewHTTPResponse(c, http.StatusOK, "Authenticated user retrieved successfully", user)
}

// Status handles GET /auth/status to report whether the instance has been initialized.
func (h *authHandler) Status(c *echo.Context) error {
	initialized, err := h.authService.IsInitialized(c.Request().Context())
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse(c, http.StatusOK, "Setup status retrieved successfully", struct {
		Initialized bool `json:"initialized"`
	}{Initialized: initialized})
}

// isHTTPSRequest reports whether the request uses HTTPS directly via TLS or through a reverse proxy setting X-Forwarded-Proto: https.
func isHTTPSRequest(c *echo.Context) bool {
	if c.Request().TLS != nil {
		return true
	}

	if forwardedProto := c.Request().Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		firstProto, _, _ := strings.Cut(forwardedProto, ",")
		if strings.EqualFold(strings.TrimSpace(firstProto), "https") {
			return true
		}
	}

	return strings.EqualFold(c.Scheme(), "https")
}
