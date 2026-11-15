package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/authctx"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/httputil"
	"github.com/yannick2009/yily/internal/service"
)

// AuthHandler defines the interface for Super Admin authentication and session HTTP handlers.
type AuthHandler interface {
	Setup(c *echo.Context) error
	Login(c *echo.Context) error
	Logout(c *echo.Context) error
	Me(c *echo.Context) error
	Status(c *echo.Context) error
	CSRF(c *echo.Context) error
	Recover(c *echo.Context) error
}

// authHandler handles HTTP requests for authentication and server-side sessions.
type authHandler struct {
	authService service.AuthService
	audit       service.AuditService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService service.AuthService, audit service.AuditService) AuthHandler {
	return &authHandler{authService: authService, audit: audit}
}

// setupRequest represents the JSON body for creating the initial Super Admin account.
type setupRequest struct {
	Name       string `json:"name" validate:"required"`
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=12,max=128"`
	Confirm    string `json:"confirm" validate:"required,eqfield=Password"`
	SetupToken string `json:"setup_token" validate:"required"`
}

// loginRequest represents the JSON body for authenticating the Super Admin.
type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// recoverRequest represents the JSON body for resetting the Super Admin password with the recovery code.
type recoverRequest struct {
	RecoveryCode string `json:"recovery_code" validate:"required"`
	Password     string `json:"password" validate:"required,min=12,max=128"`
	Confirm      string `json:"confirm" validate:"required,eqfield=Password"`
}

// recoveryCodeFilename is the name of the downloaded file carrying the recovery code.
const recoveryCodeFilename = "yily-recovery-code.txt"

// Setup handles POST /auth/setup to create the initial Super Admin account with name, email, and strong password.
func (h *authHandler) Setup(c *echo.Context) error {
	var req setupRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	_, recoveryCode, err := h.authService.Setup(c.Request().Context(), req.Name, req.Email, req.Password, req.SetupToken)
	if err != nil {
		return respondError(c, err)
	}

	return serveRecoveryCode(c, http.StatusCreated, recoveryCode)
}

// Recover handles POST /auth/recover to reset the Super Admin password using the recovery code.
func (h *authHandler) Recover(c *echo.Context) error {
	var req recoverRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	newRecoveryCode, err := h.authService.Recover(c.Request().Context(), req.RecoveryCode, req.Password)
	if err != nil {
		return respondError(c, err)
	}

	return serveRecoveryCode(c, http.StatusOK, newRecoveryCode)
}

// serveRecoveryCode streams the recovery code as a downloadable plain-text file.
func serveRecoveryCode(c *echo.Context, status int, code string) error {
	c.Response().Header().Set("Content-Disposition", `attachment; filename="`+recoveryCodeFilename+`"`)
	return c.String(status, code+"\n")
}

// Login handles POST /auth/login to authenticate the Super Admin, rotate any prior session, and set the HttpOnly session cookie.
func (h *authHandler) Login(c *echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return respondErrorCode(c, http.StatusBadRequest, CodeInvalidRequestBody, "invalid request body")
	}
	if strings.TrimSpace(req.Email) == "" || req.Password == "" {
		return respondError(c, service.ErrInvalidCredentials)
	}

	var existingToken string
	if cookie, err := c.Cookie(httputil.SessionCookieName); err == nil && cookie != nil {
		existingToken = cookie.Value
	}

	rawToken, session, user, err := h.authService.Login(c.Request().Context(), req.Email, req.Password, existingToken)
	if err != nil {
		return respondError(c, err)
	}
	if user == nil {
		return respondErrorCode(c, http.StatusInternalServerError, CodeInternal, "login returned no user")
	}

	maxAge := int(session.ExpiresAt.Sub(session.CreatedAt).Seconds())
	if maxAge <= 0 {
		maxAge = int(service.DefaultSessionTTL.Seconds())
	}

	http.SetCookie(c.Response(), &http.Cookie{
		Name:     httputil.SessionCookieName,
		Value:    rawToken,
		Path:     "/",
		Expires:  session.ExpiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   isHTTPSRequest(c),
		SameSite: http.SameSiteStrictMode,
	})

	ctx := authctx.WithAuthenticatedIdentity(c.Request().Context(), user, session)
	_ = h.audit.Record(ctx, types.ActionSignedIn, "", "", "", nil)

	return httputil.NewHTTPResponse(c, http.StatusOK, "Login successful", struct {
		User      *model.User `json:"user"`
		CSRFToken string      `json:"csrf_token"`
	}{User: user, CSRFToken: session.CSRFToken})
}

// Logout handles POST /auth/logout to revoke the current server-side session and expire the client cookie.
func (h *authHandler) Logout(c *echo.Context) error {
	var rawToken string
	if cookie, err := c.Cookie(httputil.SessionCookieName); err == nil && cookie != nil {
		rawToken = cookie.Value
	}

	err := h.authService.Logout(c.Request().Context(), rawToken)

	http.SetCookie(c.Response(), &http.Cookie{
		Name:     httputil.SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isHTTPSRequest(c),
		SameSite: http.SameSiteStrictMode,
	})

	if err != nil {
		return respondError(c, err)
	}

	_ = h.audit.Record(c.Request().Context(), types.ActionSignedOut, "", "", "", nil)

	return httputil.NewHTTPResponse[any](c, http.StatusOK, "Logout successful", nil)
}

// Me handles GET /auth/me to return the authenticated Super Admin from the request context.
func (h *authHandler) Me(c *echo.Context) error {
	user, ok := authctx.AuthenticatedUserFromContext(c.Request().Context())
	if !ok {
		return respondError(c, service.ErrUnauthorized)
	}

	return httputil.NewHTTPResponse(c, http.StatusOK, "Authenticated user retrieved successfully", user)
}

// Status handles GET /auth/status to report whether the instance has been initialized.
func (h *authHandler) Status(c *echo.Context) error {
	initialized, err := h.authService.IsInitialized(c.Request().Context())
	if err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse(c, http.StatusOK, "Setup status retrieved successfully", struct {
		Initialized bool `json:"initialized"`
	}{Initialized: initialized})
}

// CSRF handles GET /auth/csrf to return the current session's CSRF token so the
// SPA can send it back on mutations after a page reload.
func (h *authHandler) CSRF(c *echo.Context) error {
	session, ok := authctx.AuthenticatedSessionFromContext(c.Request().Context())
	if !ok || session.CSRFToken == "" {
		return respondError(c, service.ErrUnauthorized)
	}

	return httputil.NewHTTPResponse(c, http.StatusOK, "CSRF token retrieved", struct {
		CSRFToken string `json:"csrf_token"`
	}{CSRFToken: session.CSRFToken})
}

// isHTTPSRequest reports whether the request uses HTTPS. It delegates to the
// request scheme resolver, which honors X-Forwarded-Proto only when the direct
// peer is a trusted (loopback/private) address — never for arbitrary clients.
func isHTTPSRequest(c *echo.Context) bool {
	return strings.EqualFold(c.Scheme(), "https")
}
