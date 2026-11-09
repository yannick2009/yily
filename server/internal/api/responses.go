package api

import (
	"time"
	"uuid"

	"github.com/yannick2009/yily/internal/domain/model"
)

// DashboardStats holds the aggregate counts shown by the recap command.
type DashboardStats struct {
	Projects     int `json:"projects"`     // Projects is the total number of projects.
	Environments int `json:"environments"` // Environments is the total number of environments.
	Variables    int `json:"variables"`    // Variables is the total number of variables (secrets).
	Tokens       int `json:"tokens"`       // Tokens is the total number of the user's personal access tokens.
}

// DashboardRecapResponse is the subset of the dashboard payload used by the recap command.
type DashboardRecapResponse struct {
	Stats DashboardStats `json:"stats"`
}

// ProjectSummaryResponse is a project list item, including its related counts.
type ProjectSummaryResponse struct {
	ID               uuid.UUID `json:"id"`                // Unique identifier of the project.
	Name             string    `json:"name"`              // Name of the project.
	EnvironmentCount int       `json:"environment_count"` // Number of environments in the project.
	TagCount         int       `json:"tag_count"`         // Number of tags in the project.
	CreatedAt        time.Time `json:"created_at"`        // Timestamp when the project was created.
	UpdatedAt        time.Time `json:"updated_at"`        // Timestamp when the project was last updated.
}

// EnvironmentSummaryResponse is an environment list item, including its variable count.
type EnvironmentSummaryResponse struct {
	ID            uuid.UUID `json:"id"`             // Unique identifier of the environment.
	Name          string    `json:"name"`           // Name of the environment.
	Description   string    `json:"description"`    // Description of the environment.
	ProjectID     uuid.UUID `json:"project_id"`     // Foreign key referencing the associated project.
	VariableCount int       `json:"variable_count"` // Number of variables in the environment.
	CreatedAt     time.Time `json:"created_at"`     // Timestamp when the environment was created.
	UpdatedAt     time.Time `json:"updated_at"`     // Timestamp when the environment was last updated.
}

// VariableValueResponse is the response body for revealing a variable's decrypted value.
type VariableValueResponse struct {
	Value string `json:"value"`
}

// CreateTagResponse is the response body of POST /projects/:projectID/tags.
// Created lists the tags persisted during this call; Skipped lists names that already existed.
type CreateTagResponse struct {
	Created []model.Tag `json:"created"`
	Skipped []string    `json:"skipped"`
}

// VersionResponse is the server version exposed by GET /version.
type VersionResponse struct {
	Version string `json:"version"`
}
