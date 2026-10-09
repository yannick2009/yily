package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/services"
)

// VariableHandler defines the interface for handling HTTP requests related to variables and their versions.
type VariableHandler interface {
	Create(c *echo.Context) error
	List(c *echo.Context) error
	GetByID(c *echo.Context) error
	SetName(c *echo.Context) error
	Delete(c *echo.Context) error
	SetTags(c *echo.Context) error

	AddVersion(c *echo.Context) error
	GetAllVersions(c *echo.Context) error
	GetCurrentVersion(c *echo.Context) error
	SetVersionActive(c *echo.Context) error
	ClearValue(c *echo.Context) error
}

// variableHandler handles HTTP requests for variables and their versions.
type variableHandler struct {
	variableService services.VariableService
}

// NewVariableHandler creates a new VariableHandler.
func NewVariableHandler(variableService services.VariableService) VariableHandler {
	return &variableHandler{variableService}
}

// createVariableRequest represents the expected JSON body for creating a variable.
type createVariableRequest struct {
	Name  string `json:"name" validate:"required,uppercase"`
	Value string `json:"value"`
}

// updateVariableRequest represents the expected JSON body for updating a variable's name.
type updateVariableRequest struct {
	Name string `json:"name" validate:"required,uppercase"`
}

// addVersionRequest represents the expected JSON body for adding a new version to a variable.
type addVersionRequest struct {
	Value string `json:"value" validate:"required"`
}

// setVersionActiveRequest represents the expected JSON body for setting a version's active status.
type setVersionActiveRequest struct {
	Active bool `json:"active"`
}

// setTagsRequest represents the expected JSON body for setting tags on a variable.
type setTagsRequest struct {
	TagIDs []uuid.UUID `json:"tag_ids"`
}

// Create handles POST /environments/:environmentID/variables.
func (h *variableHandler) Create(c *echo.Context) error {
	environmentID, ok := uuidParam(c, "environmentID")
	if !ok {
		return nil
	}

	var req createVariableRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	variable, err := h.variableService.Create(c.Request().Context(), environmentID, req.Name, req.Value)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return NewHTTPResponse(c, http.StatusCreated, "Variable created successfully", variable)
}

// List handles GET /environments/:environmentID/variables.
// An optional "tag_ids" query parameter (comma-separated UUIDs) filters by tags.
func (h *variableHandler) List(c *echo.Context) error {
	environmentID, ok := uuidParam(c, "environmentID")
	if !ok {
		return nil
	}

	tagIDs, err := parseTagIDs(c)
	if err != nil {
		return NewHTTPResponse[any](c, http.StatusBadRequest, "invalid tag_ids", nil)
	}

	if len(tagIDs) == 0 {
		variables, err := h.variableService.GetAll(c.Request().Context(), environmentID)
		if err != nil {
			return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
		}
		return NewHTTPResponse(c, http.StatusOK, "Variables retrieved successfully", variables)
	}

	variables, err := h.variableService.GetByTags(c.Request().Context(), environmentID, tagIDs)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}
	return NewHTTPResponse(c, http.StatusOK, "Variables retrieved successfully with tags", variables)
}

// GetByID handles GET /variables/:id.
func (h *variableHandler) GetByID(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	variable, err := h.variableService.GetByID(c.Request().Context(), id)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}
	return NewHTTPResponse(c, http.StatusOK, "Variable retrieved successfully", variable)
}

// SetName handles PATCH /variables/:id.
func (h *variableHandler) SetName(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	var req updateVariableRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	if err := h.variableService.SetName(c.Request().Context(), id, req.Name); err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}
	return NewHTTPResponse[any](c, http.StatusOK, "Variable name updated successfully", nil)
}

// Delete handles DELETE /variables/:id.
func (h *variableHandler) Delete(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	if err := h.variableService.Delete(c.Request().Context(), id); err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}
	return NewHTTPResponse[any](c, http.StatusOK, "Variable deleted successfully", nil)
}

// SetTags handles PUT /variables/:id/tags.
func (h *variableHandler) SetTags(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	var req setTagsRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	if err := h.variableService.SetTags(c.Request().Context(), id, req.TagIDs); err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}
	return NewHTTPResponse[any](c, http.StatusOK, "Variable tags updated successfully", nil)
}

// AddVersion handles POST /variables/:id/versions.
func (h *variableHandler) AddVersion(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	var req addVersionRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	version, err := h.variableService.AddVersion(c.Request().Context(), id, req.Value)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}
	return NewHTTPResponse(c, http.StatusCreated, "Created", version)
}

// GetAllVersions handles GET /variables/:id/versions.
func (h *variableHandler) GetAllVersions(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	versions, err := h.variableService.GetAllVersions(c.Request().Context(), id)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}
	return NewHTTPResponse(c, http.StatusOK, "OK", versions)
}

// GetCurrentVersion handles GET /variables/:id/versions/current.
func (h *variableHandler) GetCurrentVersion(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	version, err := h.variableService.GetCurrentVersion(c.Request().Context(), id)
	if err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}
	return NewHTTPResponse(c, http.StatusOK, "OK", version)
}

// SetVersionActive handles PATCH /variables/:id/versions/:versionID.
func (h *variableHandler) SetVersionActive(c *echo.Context) error {
	versionID, ok := uuidParam(c, "versionID")
	if !ok {
		return nil
	}

	var req setVersionActiveRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	if err := h.variableService.SetVersionActive(c.Request().Context(), versionID, req.Active); err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}
	return NewHTTPResponse[any](c, http.StatusOK, "Variable version active status updated successfully", nil)
}

// ClearValue handles DELETE /variables/:id/value.
func (h *variableHandler) ClearValue(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	if err := h.variableService.ClearValue(c.Request().Context(), id); err != nil {
		return NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}
	return NewHTTPResponse[any](c, http.StatusOK, "Variable value cleared successfully", nil)
}
