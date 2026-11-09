package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/api"
	"github.com/yannick2009/yily/internal/httputil"
	"github.com/yannick2009/yily/internal/version"
)

// VersionHandler defines the interface for serving the server version.
type VersionHandler interface {
	Get(c *echo.Context) error
}

// versionHandler handles HTTP requests for the server version.
type versionHandler struct{}

// NewVersionHandler creates a new VersionHandler.
func NewVersionHandler() VersionHandler {
	return &versionHandler{}
}

// Get handles GET /version.
func (h *versionHandler) Get(c *echo.Context) error {
	return httputil.NewHTTPResponse(c, http.StatusOK, "Version retrieved successfully", api.VersionResponse{
		Version: version.Get().Number,
	})
}
