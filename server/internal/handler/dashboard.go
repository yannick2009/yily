package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/httputil"
	"github.com/yannick2009/yily/internal/middleware"
	"github.com/yannick2009/yily/internal/service"
)

// DashboardHandler defines the interface for handling HTTP requests related to the dashboard.
type DashboardHandler interface {
	Get(c *echo.Context) error // Get handles GET /dashboard to return the aggregated dashboard data.
}

// dashboardHandler handles HTTP requests for the dashboard.
type dashboardHandler struct {
	dashboardService service.DashboardService
}

// NewDashboardHandler creates a new dashboardHandler.
func NewDashboardHandler(dashboardService service.DashboardService) DashboardHandler {
	return &dashboardHandler{dashboardService: dashboardService}
}

// Get handles GET /dashboard.
func (h *dashboardHandler) Get(c *echo.Context) error {
	user, ok := middleware.AuthenticatedUserFromContext(c.Request().Context())
	if !ok {
		return respondError(c, service.ErrUnauthorized)
	}

	dashboard, err := h.dashboardService.GetDashboard(c.Request().Context(), user.ID)
	if err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse(c, http.StatusOK, "Dashboard retrieved successfully", dashboard)
}
