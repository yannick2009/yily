package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/services"
)

// Default pagination bounds for the audit log listing.
const (
	defaultAuditLimit = 50  // defaultAuditLimit is the page size used when no limit is provided.
	maxAuditLimit     = 100 // maxAuditLimit caps the page size to keep responses bounded.
)

// AuditHandler defines the interface for handling HTTP requests related to the audit log.
type AuditHandler interface {
	List(c *echo.Context) error // List handles GET /audit to list audit events.
}

// auditHandler handles HTTP requests for the audit log.
type auditHandler struct {
	auditService services.AuditService
}

// NewAuditHandler creates a new auditHandler.
func NewAuditHandler(auditService services.AuditService) AuditHandler {
	return &auditHandler{auditService: auditService}
}

// List handles GET /audit.
// Optional query parameters: search, actor_type, action, limit, offset.
func (h *auditHandler) List(c *echo.Context) error {
	limit := parseIntParam(c, "limit", defaultAuditLimit)
	if limit <= 0 {
		limit = defaultAuditLimit
	}
	if limit > maxAuditLimit {
		limit = maxAuditLimit
	}
	offset := max(parseIntParam(c, "offset", 0), 0)

	filter := services.AuditFilter{
		Search:    c.QueryParam("search"),
		ActorType: c.QueryParam("actor_type"),
		Action:    c.QueryParam("action"),
		Limit:     limit,
		Offset:    offset,
	}

	list, err := h.auditService.List(c.Request().Context(), filter)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse(c, http.StatusOK, "Audit log retrieved successfully", list)
}

// parseIntParam parses an integer query parameter, falling back to def on any error.
func parseIntParam(c *echo.Context, name string, def int) int {
	raw := c.QueryParam(name)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}
