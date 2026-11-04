// Package httputil holds transport-layer primitives shared by the handler and middleware packages: the standard JSON response envelope and the session cookie name.
package httputil

import "github.com/labstack/echo/v5"

// SessionCookieName is the HTTP cookie name storing the raw session token.
const SessionCookieName = "yily_session"

// HTTPResponse is the standard JSON envelope used for every API response.
// On errors, Code carries a stable machine-readable identifier so clients can localise the message; it stays empty on successful responses.
type HTTPResponse[T any] struct {
	Status  int    `json:"status"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
}

// NewHTTPResponse sends a standardized JSON response.
func NewHTTPResponse[T any](c *echo.Context, status int, message string, data T) error {
	return c.JSON(status, HTTPResponse[T]{Status: status, Message: message, Data: data})
}
