package repository

import (
	"context"
	"errors"
	"sort"

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
	Import(ctx context.Context, environmentID uuid.UUID, variables map[string]string) error               // Import applies the given variables (name → value), creating new ones and updating existing ones whose value changed.
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
	variable := new(model.Variable)
	if err := r.DB.WithContext(ctx).Where("id = ?", variableID).First(variable).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVariableNotFound
		}
		return nil, err
	}
	return variable, nil
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

// Import applies the given variables (name → value) to the environment in a single transaction.
// New variables are created (with their first version when a value is provided). Existing variables
// get a new version only when their value actually changed, so re-importing the same file is a no-op.
func (r *variableRepository) Import(ctx context.Context, environmentID uuid.UUID, variables map[string]string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		names := make([]string, 0, len(variables))
		for name := range variables {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			value := variables[name]

			var existing model.Variable
			err := tx.Where("name = ? AND environment_id = ?", name, environmentID).First(&existing).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				if err := createVariable(ctx, tx, environmentID, name, value); err != nil {
					return err
				}
			case err != nil:
				return err
			default:
				if err := upsertValue(ctx, tx, existing.ID, value); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// createVariable creates a new variable and, when a value is provided, its first version.
func createVariable(ctx context.Context, tx *gorm.DB, environmentID uuid.UUID, name, value string) error {
	variable := &model.Variable{Name: name, EnvironmentID: environmentID}
	if err := gorm.G[model.Variable](tx).Create(ctx, variable); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrVariableNameAlreadyExists
		}
		return err
	}
	if value == "" {
		return nil
	}
	return gorm.G[model.VariableVersion](tx).Create(ctx, &model.VariableVersion{
		Value:      value,
		Active:     true,
		Version:    1,
		VariableID: variable.ID,
	})
}

// upsertValue adds a new version to an existing variable only when its value changed.
func upsertValue(ctx context.Context, tx *gorm.DB, variableID uuid.UUID, value string) error {
	// An empty value does not represent a change.
	if value == "" {
		return nil
	}

	// Compare against the current active value.
	var current model.VariableVersion
	err := tx.Where("variable_id = ? AND active = ?", variableID, true).Order("version DESC").First(&current).Error
	if err == nil {
		if current.Value == value {
			return nil // Unchanged.
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Value changed (or the variable has no value yet): compute the next version number.
	var maxVersion int
	if err := tx.Model(&model.VariableVersion{}).
		Where("variable_id = ?", variableID).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion).Error; err != nil {
		return err
	}

	return gorm.G[model.VariableVersion](tx).Create(ctx, &model.VariableVersion{
		Value:      value,
		Active:     true,
		Version:    maxVersion + 1,
		VariableID: variableID,
	})
}
