package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/services"
)

// DashboardHandler defines the interface for handling HTTP requests related to the dashboard.
type DashboardHandler interface {
	Get(c *echo.Context) error // Get handles GET /dashboard to return the aggregated dashboard data.
}

// dashboardHandler handles HTTP requests for the dashboard.
type dashboardHandler struct {
	dashboardService services.DashboardService
}

// NewDashboardHandler creates a new dashboardHandler.
func NewDashboardHandler(dashboardService services.DashboardService) DashboardHandler {
	return &dashboardHandler{dashboardService: dashboardService}
}

// Get handles GET /dashboard.
func (h *dashboardHandler) Get(c *echo.Context) error {
	user, ok := services.AuthenticatedUserFromContext(c.Request().Context())
	if !ok {
		return NewHTTPResponse[any](c, http.StatusUnauthorized, services.ErrUnauthorized.Error(), nil)
	}

	dashboard, err := h.dashboardService.GetDashboard(c.Request().Context(), user.ID)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse(c, http.StatusOK, "Dashboard retrieved successfully", dashboard)
}
