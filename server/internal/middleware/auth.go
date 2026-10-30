package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/handler"
	"github.com/yannick2009/yily/internal/services"
)

const (
	AuthorizationHeader = "Authorization" // AuthorizationHeader is the HTTP header carrying the Bearer PAT.
	BearerScheme        = "Bearer"        // BearerScheme is the authentication scheme prefix for PATs.
)

// RequireAuth validates an incoming request using either a Bearer PAT or a server-side session cookie, and injects the authenticated identity into context.
func RequireAuth(authService services.AuthService, patService services.PATService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// If an Authorization header is provided, authenticate via Bearer PAT.
			if authHeader := strings.TrimSpace(c.Request().Header.Get(AuthorizationHeader)); authHeader != "" {
				rawToken, ok := extractBearerToken(authHeader)
				if !ok {
					return handler.NewHTTPResponse[any](c, http.StatusUnauthorized, services.ErrUnauthorized.Error(), nil)
				}

				user, pat, err := patService.Validate(c.Request().Context(), rawToken)
				if err != nil {
					return handler.NewHTTPResponse[any](c, http.StatusUnauthorized, services.ErrUnauthorized.Error(), nil)
				}

				ctx := services.WithAuthenticatedPAT(c.Request().Context(), user, pat)
				c.SetRequest(c.Request().WithContext(ctx))
				return next(c)
			}

			// Otherwise, authenticate via server-side session cookie.
			cookie, err := c.Cookie(handler.SessionCookieName)
			if err != nil || cookie == nil || strings.TrimSpace(cookie.Value) == "" {
				return handler.NewHTTPResponse[any](c, http.StatusUnauthorized, services.ErrUnauthorized.Error(), nil)
			}

			user, session, err := authService.ValidateSession(c.Request().Context(), cookie.Value)
			if err != nil {
				return handler.NewHTTPResponse[any](c, http.StatusUnauthorized, services.ErrUnauthorized.Error(), nil)
			}

			ctx := services.WithAuthenticatedIdentity(c.Request().Context(), user, session)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

// extractBearerToken parses an "Authorization: Bearer <token>" header and returns the raw token string.
func extractBearerToken(authHeader string) (string, bool) {
	authHeader = strings.TrimSpace(authHeader)
	if authHeader == "" {
		return "", false
	}

	scheme, token, found := strings.Cut(authHeader, " ")
	if !found || !strings.EqualFold(scheme, BearerScheme) {
		return "", false
	}

	token = strings.TrimSpace(token)
	if token == "" || strings.Contains(token, " ") {
		return "", false
	}

	return token, true
}
