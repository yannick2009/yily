package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/services"
)

// TagHandler defines the interface for handling HTTP requests related to tags.
type TagHandler interface {
	Create(c *echo.Context) error
	GetAll(c *echo.Context) error
	Delete(c *echo.Context) error
}

// tagHandler handles HTTP requests for tags.
type tagHandler struct {
	tagService services.TagService
}

// NewTagHandler creates a new TagHandler.
func NewTagHandler(tagService services.TagService) TagHandler {
	return &tagHandler{tagService}
}

// createTagRequest represents the expected JSON body for creating a tag.
type createTagRequest struct {
	Name string `json:"name" validate:"required"`
}

// Create handles POST /projects/:projectID/tags.
func (h *tagHandler) Create(c *echo.Context) error {
	projectID, ok := uuidParam(c, "projectID")
	if !ok {
		return nil
	}

	var req createTagRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	tag, err := h.tagService.Create(c.Request().Context(), projectID, req.Name)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}
	return NewHTTPResponse(c, http.StatusCreated, "Created", tag)
}

// GetAll handles GET /projects/:projectID/tags.
func (h *tagHandler) GetAll(c *echo.Context) error {
	projectID, ok := uuidParam(c, "projectID")
	if !ok {
		return nil
	}

	tags, err := h.tagService.GetAll(c.Request().Context(), projectID)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}
	return NewHTTPResponse(c, http.StatusOK, "Tags retrieved successfully", tags)
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
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}
	return NewHTTPResponse[any](c, http.StatusOK, "Tag deleted successfully", nil)
}
