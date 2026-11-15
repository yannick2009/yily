package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/authctx"
	"github.com/yannick2009/yily/internal/httputil"
)

// CSRFHeader is the HTTP header carrying the CSRF token on state-changing
// requests from the web UI.
const CSRFHeader = "X-CSRF-Token"

// RequireCSRF rejects state-changing requests authenticated via the session
// cookie unless they carry the session's CSRF token. It must run after
// RequireAuth so the authenticated session is available in the context.
//
// PAT-authenticated requests are exempt: they are authenticated by an explicit
// Authorization header (not a cookie), so they are not CSRF-vulnerable, and
// non-browser clients would not have a CSRF token. Read-only methods are also
// exempt since CSRF can only force a state change, not read a response.
func RequireCSRF() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if isReadMethod(c.Request().Method) {
				return next(c)
			}

			session, ok := authctx.AuthenticatedSessionFromContext(c.Request().Context())
			if !ok {
				return next(c)
			}

			provided := strings.TrimSpace(c.Request().Header.Get(CSRFHeader))
			if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(session.CSRFToken)) != 1 {
				return httputil.NewHTTPResponse[any](c, http.StatusForbidden, "invalid or missing CSRF token", nil)
			}
			return next(c)
		}
	}
}

// isReadMethod reports whether the HTTP method is a read-only verb that does not
// require CSRF protection.
func isReadMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}
