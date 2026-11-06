package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/api"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/httputil"
	"github.com/yannick2009/yily/internal/service"
)

// EnvironmentHandler defines the interface for handling HTTP requests related to environments.
type EnvironmentHandler interface {
	Create(c *echo.Context) error
	Clone(c *echo.Context) error
	GetAll(c *echo.Context) error
	GetByID(c *echo.Context) error
	SetName(c *echo.Context) error
	SetDescription(c *echo.Context) error
	Delete(c *echo.Context) error
	Import(c *echo.Context) error
	Export(c *echo.Context) error
}

// environmentHandler handles HTTP requests for environments.
type environmentHandler struct {
	environmentService service.EnvironmentService
	variableService    service.VariableService
}

// NewEnvironmentHandler creates a new EnvironmentHandler.
func NewEnvironmentHandler(environmentService service.EnvironmentService, variableService service.VariableService) EnvironmentHandler {
	return &environmentHandler{environmentService, variableService}
}

// Clone handles POST /environments/:id/clone. The new environment lives in the same project; variables are duplicated by name only (no values).
func (h *environmentHandler) Clone(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	var req api.CloneEnvironmentRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	if err := h.environmentService.Clone(c.Request().Context(), id, req.Name, req.CopyTags); err != nil {
		return httputil.NewHTTPResponse[any](c, httpStatus(err), err.Error(), nil)
	}

	return httputil.NewHTTPResponse[any](c, http.StatusCreated, "Environment cloned successfully", nil)
}

// Create handles POST /projects/:projectID/environments.
func (h *environmentHandler) Create(c *echo.Context) error {
	projectID, ok := uuidParam(c, "projectID")
	if !ok {
		return nil
	}

	var req api.CreateEnvironmentRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	environment, err := h.environmentService.Create(c.Request().Context(), projectID, req.Name, req.Description)
	if err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse(c, http.StatusCreated, "Environment created successfully", environment)
}

// GetAll handles GET /projects/:projectID/environments.
func (h *environmentHandler) GetAll(c *echo.Context) error {
	projectID, ok := uuidParam(c, "projectID")
	if !ok {
		return nil
	}

	environments, err := h.environmentService.GetAll(c.Request().Context(), projectID)
	if err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse(c, http.StatusOK, "Environments retrieved successfully", environments)
}

// GetByID handles GET /environments/:id.
func (h *environmentHandler) GetByID(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	environment, err := h.environmentService.GetByID(c.Request().Context(), id)
	if err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse(c, http.StatusOK, "Environment retrieved successfully", environment)
}

// SetName handles PATCH /environments/:id.
func (h *environmentHandler) SetName(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	var req api.UpdateEnvironmentNameRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	if err := h.environmentService.SetName(c.Request().Context(), id, req.Name); err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse[any](c, http.StatusOK, "Environment name updated successfully", nil)
}

// SetDescription handles PATCH /environments/:id/description.
func (h *environmentHandler) SetDescription(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	var req api.UpdateEnvironmentDescriptionRequest
	if !bindAndValidate(c, &req) {
		return nil
	}

	if err := h.environmentService.SetDescription(c.Request().Context(), id, req.Description); err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse[any](c, http.StatusOK, "Environment description updated successfully", nil)
}

// Delete handles DELETE /environments/:id.
func (h *environmentHandler) Delete(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	if err := h.environmentService.Delete(c.Request().Context(), id); err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse[any](c, http.StatusOK, "Environment deleted successfully", nil)
}

// Import handles POST /environments/:id/import.
func (h *environmentHandler) Import(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	// Parse the multipart form data
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return respondErrorCode(c, http.StatusBadRequest, CodeFileParseFailed, "Failed to parse multipart form data")
	}

	// Check the file extension
	fileType, err := types.FromFilename(fileHeader.Filename)
	if err != nil {
		return respondError(c, err)
	}

	// Get the file from the form data
	file, err := fileHeader.Open()
	if err != nil {
		return respondErrorCode(c, http.StatusInternalServerError, CodeFileOpenFailed, "Failed to open uploaded file")
	}
	defer func() { _ = file.Close() }()

	contentByte, err := io.ReadAll(file)
	if err != nil {
		return respondErrorCode(c, http.StatusInternalServerError, CodeFileReadFailed, "Failed to read uploaded file")
	}

	// Call the variableService to import variables from the uploaded file
	if err := h.variableService.Import(c.Request().Context(), id, contentByte, fileType); err != nil {
		return respondError(c, err)
	}

	return httputil.NewHTTPResponse[any](c, http.StatusOK, "Variables imported successfully", nil)
}

// Export handles GET /environments/:id/export?format=json|yaml|env.
func (h *environmentHandler) Export(c *echo.Context) error {
	id, ok := uuidParam(c, "id")
	if !ok {
		return nil
	}

	fileType, err := types.FromString(c.QueryParam("format"))
	if err != nil {
		return respondErrorCode(c, http.StatusBadRequest, CodeFileUnsupportedType, "unsupported or missing format")
	}

	environment, err := h.environmentService.GetByID(c.Request().Context(), id)
	if err != nil {
		return respondError(c, err)
	}

	content, err := h.variableService.Export(c.Request().Context(), id, fileType)
	if err != nil {
		return respondError(c, err)
	}

	filename := exportFilename(environment.Name, fileType)
	c.Response().Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	return c.Blob(http.StatusOK, exportContentType(fileType), content)
}

// exportFilename builds a safe download filename from an environment name and file type.
func exportFilename(name string, fileType types.FileType) string {
	name = strings.NewReplacer(`"`, "", "\r", "", "\n", "").Replace(name)
	return "export_" + name + fileType.Extension()
}

// exportContentType returns the HTTP content type for an exported file.
func exportContentType(fileType types.FileType) string {
	switch fileType {
	case types.FileTypeJSON:
		return "application/json"
	case types.FileTypeYAML:
		return "application/x-yaml"
	case types.FileTypeEnv:
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}
