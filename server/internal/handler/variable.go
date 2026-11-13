package handler

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/api"
	"github.com/yannick2009/yily/internal/httputil"
	"github.com/yannick2009/yily/internal/repository"
	"github.com/yannick2009/yily/internal/service"
)

// VariableHandler defines the interface for handling HTTP requests related to variables and their versions.
type VariableHandler interface {
	Create(c *echo.Context) error
	List(c *echo.Context) error
	GetByID(c *echo.Context) error
	GetValue(c *echo.Context) error
	GetValues(c *echo.Context) error
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
	variableService service.VariableService
}

// NewVariableHandler creates a new VariableHandler.
func NewVariableHandler(variableService service.VariableService) VariableHandler {
	return &variableHandler{variableService}
}

// Create handles POST /environments/:environmentID/variables.
func (h *variableHandler) Create(c *echo.Context) error {
	environmentID, ok := uuidParam(c, "environmentID")
	if !ok {
		return nil
	}

	var req api.CreateVariableRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	variable, err := h.variableService.Create(c.Request().Context(), environmentID, req.Name, req.Value)
	if err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse(c, http.StatusCreated, "Variable created successfully", variable)
}

// List handles GET /environments/:environmentID/variables.
// Optional "search" (case-insensitive substring on name) and "tag_ids"
// (comma-separated UUIDs) query parameters filter the result.
func (h *variableHandler) List(c *echo.Context) error {
	environmentID, ok := uuidParam(c, "environmentID")
	if !ok {
		return nil
	}

	tagIDs, err := parseTagIDs(c)
	if err != nil {
		return respondErrorCode(c, http.StatusBadRequest, CodeInvalidUUID, "invalid tag_ids")
	}

	variables, err := h.variableService.List(c.Request().Context(), environmentID, service.VariableListFilter{
		Search: strings.TrimSpace(c.QueryParam("search")),
		TagIDs: tagIDs,
		Sort:   parseVariableSort(c.QueryParam("sort")),
		Order:  parseVariableOrder(c.QueryParam("order")),
	})
	if err != nil {
		return respondError(c, err)
	}
	return httputil.NewHTTPResponse(c, http.StatusOK, "Variables retrieved successfully", variables)
}

// GetByID handles GET /variables/:id.
func (h *variableHandler) GetByID(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	variable, err := h.variableService.GetByID(c.Request().Context(), id)
	if err != nil {
		return respondError(c, err)
	}
	return httputil.NewHTTPResponse(c, http.StatusOK, "Variable retrieved successfully", variable)
}

// GetValue handles GET /variables/:id/value.
func (h *variableHandler) GetValue(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	value, err := h.variableService.GetValue(c.Request().Context(), id)
	if err != nil {
		return respondError(c, err)
	}
	return httputil.NewHTTPResponse(c, http.StatusOK, "Variable value retrieved successfully", api.VariableValueResponse{Value: value})
}

// GetValues handles POST /environments/:environmentID/variables/values.
// The JSON body selects each variable individually, so one request can mix
// active and historical versions.
func (h *variableHandler) GetValues(c *echo.Context) error {
	environmentID, ok := uuidParam(c, "environmentID")
	if !ok {
		return nil
	}

	var req api.GetVariableValuesRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	selections := make([]service.VariableSelection, 0, len(req.Variables))
	for _, variable := range req.Variables {
		if strings.TrimSpace(variable.Name) == "" {
			return respondErrorCode(c, http.StatusBadRequest, CodeBadRequest, "variable names cannot be empty")
		}
		selections = append(selections, service.VariableSelection{
			Name:    variable.Name,
			Version: variable.Version,
		})
	}

	values, err := h.variableService.GetValues(c.Request().Context(), environmentID, selections)
	if err != nil {
		return respondError(c, err)
	}
	return httputil.NewHTTPResponse(c, http.StatusOK, "Variable values retrieved successfully", values)
}

// SetName handles PATCH /variables/:id.
func (h *variableHandler) SetName(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	var req api.UpdateVariableRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	if err := h.variableService.SetName(c.Request().Context(), id, req.Name); err != nil {
		return respondError(c, err)
	}
	return httputil.NewHTTPResponse[any](c, http.StatusOK, "Variable name updated successfully", nil)
}

// Delete handles DELETE /variables/:id.
func (h *variableHandler) Delete(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	if err := h.variableService.Delete(c.Request().Context(), id); err != nil {
		return respondError(c, err)
	}
	return httputil.NewHTTPResponse[any](c, http.StatusOK, "Variable deleted successfully", nil)
}

// SetTags handles PUT /variables/:id/tags.
func (h *variableHandler) SetTags(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	var req api.SetVariableTagsRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	if err := h.variableService.SetTags(c.Request().Context(), id, req.TagIDs); err != nil {
		return respondError(c, err)
	}
	return httputil.NewHTTPResponse[any](c, http.StatusOK, "Variable tags updated successfully", nil)
}

// AddVersion handles POST /variables/:id/versions.
func (h *variableHandler) AddVersion(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	var req api.AddVariableVersionRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	version, err := h.variableService.AddVersion(c.Request().Context(), id, req.Value)
	if err != nil {
		return respondError(c, err)
	}
	return httputil.NewHTTPResponse(c, http.StatusCreated, "Created", version)
}

// GetAllVersions handles GET /variables/:id/versions.
func (h *variableHandler) GetAllVersions(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	versions, err := h.variableService.GetAllVersions(c.Request().Context(), id)
	if err != nil {
		return respondError(c, err)
	}
	return httputil.NewHTTPResponse(c, http.StatusOK, "OK", versions)
}

// GetCurrentVersion handles GET /variables/:id/versions/current.
func (h *variableHandler) GetCurrentVersion(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	version, err := h.variableService.GetCurrentVersion(c.Request().Context(), id)
	if err != nil {
		return respondError(c, err)
	}
	return httputil.NewHTTPResponse(c, http.StatusOK, "OK", version)
}

// SetVersionActive handles PATCH /variables/:id/versions/:versionID.
func (h *variableHandler) SetVersionActive(c *echo.Context) error {
	versionID, ok := uuidParam(c, "versionID")
	if !ok {
		return nil
	}

	var req api.SetVariableVersionActiveRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	if err := h.variableService.SetVersionActive(c.Request().Context(), versionID, req.Active); err != nil {
		return respondError(c, err)
	}
	return httputil.NewHTTPResponse[any](c, http.StatusOK, "Variable version active status updated successfully", nil)
}

// ClearValue handles DELETE /variables/:id/value.
func (h *variableHandler) ClearValue(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	if err := h.variableService.ClearValue(c.Request().Context(), id); err != nil {
		return respondError(c, err)
	}
	return httputil.NewHTTPResponse[any](c, http.StatusOK, "Variable value cleared successfully", nil)
}

// parseVariableSort whitelists the accepted sort columns. Unknown or empty values fall back to the default (name).
func parseVariableSort(raw string) repository.VariableSortField {
	switch raw {
	case string(repository.VariableSortByCreatedAt):
		return repository.VariableSortByCreatedAt
	default:
		return repository.VariableSortByName
	}
}

// parseVariableOrder whitelists the accepted sort directions. Unknown or empty values fall back to ascending.
func parseVariableOrder(raw string) repository.VariableSortOrder {
	switch raw {
	case string(repository.VariableSortDesc):
		return repository.VariableSortDesc
	default:
		return repository.VariableSortAsc
	}
}
