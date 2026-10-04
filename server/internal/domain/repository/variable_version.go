package repository

import (
	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
)

type VariableVersionRepository interface {
	Create(version *model.VariableVersion) error                             // Create inserts a new variable version
	GetByID(id uuid.UUID) (*model.VariableVersion, error)                    // GetByID retrieves a variable version by its UUID
	ListByVariableID(variableID uuid.UUID) ([]*model.VariableVersion, error) // ListByVariableID retrieves all versions of a specific variable
	Update(version *model.VariableVersion) error                             // Update modifies an existing variable version
	Delete(id uuid.UUID) error                                               // Delete deletes a variable version by its UUID
}
