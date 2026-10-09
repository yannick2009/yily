package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/services"
)

// ProjectHandler defines the interface for handling HTTP requests related to projects.
type ProjectHandler interface {
	Create(c *echo.Context) error
	GetAll(c *echo.Context) error
	GetByID(c *echo.Context) error
	SetName(c *echo.Context) error
	Delete(c *echo.Context) error
}

// projectHandler handles HTTP requests for projects.
type projectHandler struct {
	projectService services.ProjectService
}

// NewProjectHandler creates a new projectHandler.
func NewProjectHandler(projectService services.ProjectService) ProjectHandler {
	return &projectHandler{projectService}
}

// projectRequest represents the expected JSON body for creating or updating a project.
type projectRequest struct {
	Name string `json:"name" validate:"required"`
}

// Create handles POST /projects.
func (h *projectHandler) Create(c *echo.Context) error {
	var req projectRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	project, err := h.projectService.Create(c.Request().Context(), req.Name)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse(c, http.StatusCreated, "Project created successfully", project)
}

// GetAll handles GET /projects.
func (h *projectHandler) GetAll(c *echo.Context) error {
	projects, err := h.projectService.GetAll(c.Request().Context())
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse(c, http.StatusOK, "Projects retrieved successfully", projects)
}

// GetByID handles GET /projects/:id.
func (h *projectHandler) GetByID(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	project, err := h.projectService.GetByID(c.Request().Context(), id)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse(c, http.StatusOK, "Project retrieved successfully", project)
}

// SetName handles PATCH /projects/:id.
func (h *projectHandler) SetName(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	var req projectRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	if err := h.projectService.SetName(c.Request().Context(), id, req.Name); err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse[any](c, http.StatusOK, "Project name updated successfully", nil)
}

// Delete handles DELETE /projects/:id.
func (h *projectHandler) Delete(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	if err := h.projectService.Delete(c.Request().Context(), id); err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse[any](c, http.StatusOK, "Project deleted successfully", nil)
}
