package middleware

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/authctx"
	"github.com/yannick2009/yily/internal/httputil"
	"github.com/yannick2009/yily/internal/service"
)

// RequireWriteScope rejects authenticated requests whose effective scope does not grant write access. Must run after RequireAuth.
func RequireWriteScope() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			scope, ok := authctx.AuthenticatedScopeFromContext(c.Request().Context())
			if !ok {
				return httputil.NewHTTPResponse[any](c, http.StatusUnauthorized, service.ErrUnauthorized.Error(), nil)
			}
			if !scope.AllowsWrite() {
				return httputil.NewHTTPResponse[any](c, http.StatusForbidden, service.ErrPATWriteForbidden.Error(), nil)
			}
			return next(c)
		}
	}
}

// RequireSession rejects authenticated requests that were authenticated via a PAT. Use to gate super-admin-only routes (token management, system config) to the web UI session. Must run after RequireAuth.
func RequireSession() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if _, ok := authctx.AuthenticatedPATFromContext(c.Request().Context()); ok {
				return httputil.NewHTTPResponse[any](c, http.StatusForbidden, "this endpoint is only accessible through the web interface", nil)
			}
			if _, ok := authctx.AuthenticatedSessionFromContext(c.Request().Context()); !ok {
				return httputil.NewHTTPResponse[any](c, http.StatusUnauthorized, service.ErrUnauthorized.Error(), nil)
			}
			return next(c)
		}
	}
}
