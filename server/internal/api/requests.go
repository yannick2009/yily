package api

import "uuid"

// CreateProjectRequest is the JSON body for creating a project.
type CreateProjectRequest struct {
	Name string `json:"name" validate:"required"`
}

// UpdateProjectRequest is the JSON body for renaming a project.
// It has the same shape as CreateProjectRequest.
type UpdateProjectRequest = CreateProjectRequest

// CreateEnvironmentRequest is the JSON body for creating an environment.
type CreateEnvironmentRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

// UpdateEnvironmentNameRequest is the JSON body for renaming an environment.
type UpdateEnvironmentNameRequest struct {
	Name string `json:"name" validate:"required"`
}

// UpdateEnvironmentDescriptionRequest is the JSON body for updating an environment's description.
// An empty description clears it.
type UpdateEnvironmentDescriptionRequest struct {
	Description string `json:"description"`
}

// CloneEnvironmentRequest is the JSON body for cloning an environment.
type CloneEnvironmentRequest struct {
	Name     string `json:"name" validate:"required"`
	CopyTags bool   `json:"copy_tags"`
}

// CreateVariableRequest is the JSON body for creating a variable.
type CreateVariableRequest struct {
	Name  string `json:"name" validate:"required"`
	Value string `json:"value"`
}

// UpdateVariableRequest is the JSON body for renaming a variable.
type UpdateVariableRequest struct {
	Name string `json:"name" validate:"required"`
}

// AddVariableVersionRequest is the JSON body for adding a variable version.
type AddVariableVersionRequest struct {
	Value string `json:"value" validate:"required"`
}

// SetVariableVersionActiveRequest is the JSON body for (de)activating a variable version.
type SetVariableVersionActiveRequest struct {
	Active bool `json:"active"`
}

// SetVariableTagsRequest is the JSON body for replacing a variable's tags.
type SetVariableTagsRequest struct {
	TagIDs []uuid.UUID `json:"tag_ids"`
}

// CreateTagRequest is the JSON body for creating one or more tags.
// Name is kept for backward compatibility; new callers should use Names.
type CreateTagRequest struct {
	Name  string   `json:"name,omitempty"`
	Names []string `json:"names,omitempty"`
}

// GetVariableValuesRequest is the JSON body for batch variable value retrieval.
// Each variable may select its own version; an omitted version selects the
// variable's active version.
type GetVariableValuesRequest struct {
	Variables []VariableSelection `json:"variables" validate:"required,min=1,max=1000,dive"`
}

// VariableSelection selects one variable and, optionally, a specific version.
type VariableSelection struct {
	Name    string `json:"name" validate:"required"`
	Version int    `json:"version" validate:"gte=0"`
}
