package repository

import (
	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
)

// ProjectRepository defines the interface for project storage operations.
type ProjectRepository interface {
	Create(project *model.Project) error          // Create inserts a new project
	GetByID(id uuid.UUID) (*model.Project, error) // GetByID retrieves a project by its UUID
	List() ([]*model.Project, error)              // List retrieves all projects
	Update(project *model.Project) error          // Update modifies an existing project
	Delete(id uuid.UUID) error                    // Delete deletes a project by its UUID
}
