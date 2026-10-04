package repository

import (
	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
)

type EnvironmentRepository interface {
	Create(environment *model.Environment) error                       // Create inserts a new environment
	GetByID(id uuid.UUID) (*model.Environment, error)                  // GetByID retrieves an environment by its UUID
	ListByProjectID(projectID uuid.UUID) ([]*model.Environment, error) // ListByProjectID retrieves all environments associated with a specific project
	Update(environment *model.Environment) error                       // Update modifies an existing environment
	Delete(id uuid.UUID) error                                         // Delete deletes an environment by its UUID
}
