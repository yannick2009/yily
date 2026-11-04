package handler

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/services"
)

// Default pagination bounds for the audit log listing.
const (
	defaultAuditLimit = 50  // defaultAuditLimit is the page size used when no limit is provided.
	maxAuditLimit     = 100 // maxAuditLimit caps the page size to keep responses bounded.
)

// AuditHandler defines the interface for handling HTTP requests related to the audit log.
type AuditHandler interface {
	List(c *echo.Context) error   // List handles GET /audit to list audit events.
	Export(c *echo.Context) error // Export handles GET /audit/export to download audit events as CSV.
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

	from, err := parseAuditBoundary(c.QueryParam("from"), false)
	if err != nil {
		return NewHTTPResponse[any](c, http.StatusBadRequest, "invalid from", nil)
	}
	to, err := parseAuditBoundary(c.QueryParam("to"), true)
	if err != nil {
		return NewHTTPResponse[any](c, http.StatusBadRequest, "invalid to", nil)
	}

	filter := services.AuditFilter{
		Search:    c.QueryParam("search"),
		ActorType: c.QueryParam("actor_type"),
		Action:    c.QueryParam("action"),
		From:      from,
		To:        to,
		Limit:     limit,
		Offset:    offset,
	}

	list, err := h.auditService.List(c.Request().Context(), filter)
	if err != nil {
		return respondError(c, err)
	}

	return NewHTTPResponse(c, http.StatusOK, "Audit log retrieved successfully", list)
}

// Export handles GET /audit/export, streaming every event matching the same
// filters as List as a CSV attachment. The export is not paginated.
func (h *auditHandler) Export(c *echo.Context) error {
	from, err := parseAuditBoundary(c.QueryParam("from"), false)
	if err != nil {
		return NewHTTPResponse[any](c, http.StatusBadRequest, "invalid from", nil)
	}
	to, err := parseAuditBoundary(c.QueryParam("to"), true)
	if err != nil {
		return NewHTTPResponse[any](c, http.StatusBadRequest, "invalid to", nil)
	}

	filter := services.AuditFilter{
		Search:    c.QueryParam("search"),
		ActorType: c.QueryParam("actor_type"),
		Action:    c.QueryParam("action"),
		From:      from,
		To:        to,
	}

	events, err := h.auditService.Export(c.Request().Context(), filter)
	if err != nil {
		return respondError(c, err)
	}

	content, err := auditCSV(events)
	if err != nil {
		return respondError(c, err)
	}

	c.Response().Header().Set("Content-Disposition", `attachment; filename="audit-log.csv"`)
	return c.Blob(http.StatusOK, "text/csv", content)
}

// auditCSV serializes audit events into a CSV document with a header row.
func auditCSV(events []model.AuditEvent) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if err := w.Write([]string{"created_at", "actor_type", "actor_name", "action", "resource_type", "resource_name", "metadata"}); err != nil {
		return nil, err
	}
	for _, e := range events {
		if err := w.Write([]string{
			e.CreatedAt.UTC().Format(time.RFC3339),
			e.ActorType,
			e.ActorName,
			e.Action,
			e.ResourceType,
			e.ResourceName,
			e.Metadata,
		}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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

// parseAuditBoundary parses an audit log date boundary. Accepts a date-only
// value (YYYY-MM-DD) in UTC or a full RFC3339 timestamp. An empty value returns
// (nil, nil). For a date-only upper bound, inclusiveDay shifts the result to
// the next midnight so the whole day is covered by the exclusive upper bound.
func parseAuditBoundary(raw string, inclusiveDay bool) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", raw, time.UTC); err == nil {
		if inclusiveDay {
			t = t.Add(24 * time.Hour)
		}
		return &t, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
