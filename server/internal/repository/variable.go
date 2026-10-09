package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
	"gorm.io/gorm"
)

var (
	ErrVariableNotFound          = errors.New("variable not found")                               // ErrVariableNotFound is returned when a variable is not found in the database.
	ErrVariableNameAlreadyExists = errors.New("variable name already exists in this environment") // ErrVariableNameAlreadyExists is returned when a variable name already exists in the environment.
)

// VariableRepository defines the interface for interacting with variable data in the database.
type VariableRepository interface {
	Create(ctx context.Context, variable *model.Variable) error                                           // Create creates a new variable in the database.
	SetName(ctx context.Context, variableID uuid.UUID, name string) error                                 // SetName updates the name of an existing variable in the database.
	Delete(ctx context.Context, variableID uuid.UUID) error                                               // Delete deletes a variable from the database by its ID.
	GetByID(ctx context.Context, variableID uuid.UUID) (*model.Variable, error)                           // GetByID retrieves a variable from the database by its ID.
	GetAll(ctx context.Context, environmentID uuid.UUID) ([]model.Variable, error)                        // GetAll retrieves all variables from the database.
	GetByTags(ctx context.Context, environmentID uuid.UUID, tagIDs []uuid.UUID) ([]model.Variable, error) // GetByTags retrieves all variables that have at least one of the given tags.
	SetTags(ctx context.Context, variableID uuid.UUID, tagIDs []uuid.UUID) error                          // SetTags replaces the set of tags associated with a variable.
}

// variableRepository is a struct that implements the VariableRepository interface.
type variableRepository struct {
	DB *gorm.DB
}

// NewVariableRepository creates a new instance of variableRepository with the provided GORM database connection.
func NewVariableRepository(db *gorm.DB) VariableRepository {
	return &variableRepository{DB: db}
}

// Create creates a new variable in the database.
func (r *variableRepository) Create(ctx context.Context, variable *model.Variable) error {
	if err := gorm.G[model.Variable](r.DB).Create(ctx, variable); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrVariableNameAlreadyExists
		}
		return err
	}
	return nil
}

// SetName updates the name of an existing variable in the database.
func (r *variableRepository) SetName(ctx context.Context, variableID uuid.UUID, name string) error {
	_, err := gorm.G[model.Variable](r.DB).Where("id = ?", variableID).Update(ctx, "name", name)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrVariableNameAlreadyExists
		}
		return err
	}
	return nil
}

// Delete deletes a variable from the database by its ID.
func (r *variableRepository) Delete(ctx context.Context, variableID uuid.UUID) error {
	_, err := gorm.G[model.Variable](r.DB).Where("id = ?", variableID).Delete(ctx)
	return err
}

// GetByID retrieves a variable from the database by its ID.
func (r *variableRepository) GetByID(ctx context.Context, variableID uuid.UUID) (*model.Variable, error) {
	variable, err := gorm.G[model.Variable](r.DB).Where("id = ?", variableID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVariableNotFound
		}
		return nil, err
	}
	return &variable, nil
}

// GetAll retrieves all variables from the database.
func (r *variableRepository) GetAll(ctx context.Context, environmentID uuid.UUID) ([]model.Variable, error) {
	variables, err := gorm.G[model.Variable](r.DB).Where("environment_id = ?", environmentID).Find(ctx)
	if err != nil {
		return nil, err
	}
	return variables, nil
}

// GetByTags retrieves all variables of the given environment that have at least one of the given tags.
// If tagIDs is empty, all variables of the environment are returned.
func (r *variableRepository) GetByTags(ctx context.Context, environmentID uuid.UUID, tagIDs []uuid.UUID) ([]model.Variable, error) {
	if len(tagIDs) == 0 {
		return r.GetAll(ctx, environmentID)
	}

	variables, err := gorm.G[model.Variable](r.DB).
		Where("environment_id = ?", environmentID).
		Where("EXISTS (SELECT 1 FROM variable_tags WHERE variable_tags.variable_id = variables.id AND variable_tags.tag_id IN ?)", tagIDs).
		Find(ctx)
	if err != nil {
		return nil, err
	}

	return variables, nil
}

// SetTags replaces the set of tags associated with a variable.
func (r *variableRepository) SetTags(ctx context.Context, variableID uuid.UUID, tagIDs []uuid.UUID) error {
	// Resolve the variable's project via its environment to scope the tags.
	v, err := gorm.G[model.Variable](r.DB).Where("id = ?", variableID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVariableNotFound
		}
		return err
	}

	variable := model.Variable{ID: variableID}

	// No tags means "remove them all".
	if len(tagIDs) == 0 {
		return r.DB.Model(&variable).Association("Tags").Clear()
	}

	env, err := gorm.G[model.Environment](r.DB).Where("id = ?", v.EnvironmentID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrEnvironmentNotFound
		}
		return err
	}

	// Fetch the tag models (the generic API does not expose Association, hence plain r.DB below).
	tags, err := gorm.G[model.Tag](r.DB).Where("id IN ?", tagIDs).Find(ctx)
	if err != nil {
		return err
	}
	for _, tag := range tags {
		if tag.ProjectID != env.ProjectID {
			return ErrTagNotInProject
		}
	}

	return r.DB.Model(&variable).Association("Tags").Replace(tags)
}
