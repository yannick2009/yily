package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/services"
)

// EnvironmentHandler defines the interface for handling HTTP requests related to environments.
type EnvironmentHandler interface {
	Create(c *echo.Context) error
	GetAll(c *echo.Context) error
	GetByID(c *echo.Context) error
	SetName(c *echo.Context) error
	SetDescription(c *echo.Context) error
	Delete(c *echo.Context) error
}

// environmentHandler handles HTTP requests for environments.
type environmentHandler struct {
	environmentService services.EnvironmentService
}

// NewEnvironmentHandler creates a new EnvironmentHandler.
func NewEnvironmentHandler(environmentService services.EnvironmentService) EnvironmentHandler {
	return &environmentHandler{environmentService}
}

// createEnvironmentRequest represents the expected JSON body for creating an environment.
type createEnvironmentRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

// updateEnvironmentNameRequest represents the expected JSON body for updating an environment's name.
type updateEnvironmentNameRequest struct {
	Name string `json:"name" validate:"required"`
}

// updateEnvironmentDescriptionRequest represents the expected JSON body for updating an environment's description.
// The description may be empty, which clears it.
type updateEnvironmentDescriptionRequest struct {
	Description string `json:"description"`
}

// Create handles POST /projects/:projectID/environments.
func (h *environmentHandler) Create(c *echo.Context) error {
	projectID, ok := uuidParam(c, "projectID")
	if !ok {
		return nil
	}

	var req createEnvironmentRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	environment, err := h.environmentService.Create(c.Request().Context(), projectID, req.Name, req.Description)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse(c, http.StatusCreated, "Environment created successfully", environment)
}

// GetAll handles GET /projects/:projectID/environments.
func (h *environmentHandler) GetAll(c *echo.Context) error {
	projectID, ok := uuidParam(c, "projectID")
	if !ok {
		return nil
	}

	environments, err := h.environmentService.GetAll(c.Request().Context(), projectID)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse(c, http.StatusOK, "Environments retrieved successfully", environments)
}

// GetByID handles GET /environments/:id.
func (h *environmentHandler) GetByID(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	environment, err := h.environmentService.GetByID(c.Request().Context(), id)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse(c, http.StatusOK, "Environment retrieved successfully", environment)
}

// SetName handles PATCH /environments/:id.
func (h *environmentHandler) SetName(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	var req updateEnvironmentNameRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	if err := h.environmentService.SetName(c.Request().Context(), id, req.Name); err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse[any](c, http.StatusOK, "Environment name updated successfully", nil)
}

// SetDescription handles PATCH /environments/:id/description.
func (h *environmentHandler) SetDescription(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	var req updateEnvironmentDescriptionRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	if err := h.environmentService.SetDescription(c.Request().Context(), id, req.Description); err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse[any](c, http.StatusOK, "Environment description updated successfully", nil)
}

// Delete handles DELETE /environments/:id.
func (h *environmentHandler) Delete(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	if err := h.environmentService.Delete(c.Request().Context(), id); err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse[any](c, http.StatusOK, "Environment deleted successfully", nil)
}
