package repository

import (
	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
)

type VariableRepository interface {
	Create(variable *model.Variable) error                                  // Create inserts a new variable
	GetByID(id uuid.UUID) (*model.Variable, error)                          // GetByID retrieves a variable by its UUID
	ListByEnvironmentID(environmentID uuid.UUID) ([]*model.Variable, error) // ListByEnvironmentID retrieves all variables associated with a specific environment
	Update(variable *model.Variable) error                                  // Update modifies an existing variable
	Delete(id uuid.UUID) error                                              // Delete deletes a variable by its UUID
}
