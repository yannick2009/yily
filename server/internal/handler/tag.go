package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/api"
	"github.com/yannick2009/yily/internal/httputil"
	"github.com/yannick2009/yily/internal/service"
)

// TagHandler defines the interface for handling HTTP requests related to tags.
type TagHandler interface {
	Create(c *echo.Context) error
	GetAll(c *echo.Context) error
	Delete(c *echo.Context) error
}

// tagHandler handles HTTP requests for tags.
type tagHandler struct {
	tagService service.TagService
}

// NewTagHandler creates a new TagHandler.
func NewTagHandler(tagService service.TagService) TagHandler {
	return &tagHandler{tagService}
}

// Create handles POST /projects/:projectID/tags. The body may carry either a single "name" (legacy) or a "names" array; empty values and duplicates are ignored. Existing names are reported in `skipped` and never surface as errors.
func (h *tagHandler) Create(c *echo.Context) error {
	projectID, ok := uuidParam(c, "projectID")
	if !ok {
		return nil
	}

	var req api.CreateTagRequest
	if err := c.Bind(&req); err != nil {
		return httputil.NewHTTPResponse[any](c, http.StatusBadRequest, "invalid request body", nil)
	}

	names := req.Names
	if len(names) == 0 && req.Name != "" {
		names = []string{req.Name}
	}
	if len(names) == 0 {
		return httputil.NewHTTPResponse[any](c, http.StatusBadRequest, "name or names is required", nil)
	}

	result, err := h.tagService.CreateBatch(c.Request().Context(), projectID, names)
	if err != nil {
		return respondError(c, err)
	}
	return httputil.NewHTTPResponse(c, http.StatusCreated, "Created", api.CreateTagResponse{
		Created: result.Created,
		Skipped: result.Skipped,
	})
}

// GetAll handles GET /projects/:projectID/tags.
func (h *tagHandler) GetAll(c *echo.Context) error {
	projectID, ok := uuidParam(c, "projectID")
	if !ok {
		return nil
	}

	tags, err := h.tagService.GetAll(c.Request().Context(), projectID)
	if err != nil {
		return respondError(c, err)
	}
	return httputil.NewHTTPResponse(c, http.StatusOK, "Tags retrieved successfully", tags)
}

// Delete handles DELETE /projects/:projectID/tags/:tagID.
func (h *tagHandler) Delete(c *echo.Context) error {
	projectID, ok := uuidParam(c, "projectID")
	if !ok {
		return nil
	}
	tagID, ok := uuidParam(c, "tagID")
	if !ok {
		return nil
	}

	if err := h.tagService.Delete(c.Request().Context(), projectID, tagID); err != nil {
		return respondError(c, err)
	}
	return httputil.NewHTTPResponse[any](c, http.StatusOK, "Tag deleted successfully", nil)
}
