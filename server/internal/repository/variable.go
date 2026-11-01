package repository

import (
	"context"
	"errors"
	"strings"

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
	Create(ctx context.Context, variable *model.Variable) error                                             // Create creates a new variable in the database.
	SetName(ctx context.Context, variableID uuid.UUID, name string) error                                   // SetName updates the name of an existing variable in the database.
	Delete(ctx context.Context, variableID uuid.UUID) error                                                 // Delete deletes a variable from the database by its ID.
	GetByID(ctx context.Context, variableID uuid.UUID) (*model.Variable, error)                             // GetByID retrieves a variable from the database by its ID.
	List(ctx context.Context, environmentID uuid.UUID, filter VariableListFilter) ([]model.Variable, error) // List retrieves all variables of an environment matching the given filter.
	CountByProjectID(ctx context.Context, projectID uuid.UUID) (int64, error)                               // CountByProjectID counts the variables across all environments of a project.
	CountByEnvironmentID(ctx context.Context, environmentID uuid.UUID) (int64, error)                       // CountByEnvironmentID counts the variables of an environment.
	SetTags(ctx context.Context, variableID uuid.UUID, tagIDs []uuid.UUID) error                            // SetTags replaces the set of tags associated with a variable.
}

// VariableListFilter filters the variable listing of an environment.
type VariableListFilter struct {
	Search string      // Search matches variable names case-insensitively (empty for no filter).
	TagIDs []uuid.UUID // TagIDs filters to variables that have at least one of these tags (empty for no filter).
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

// GetByID retrieves a variable from the database by its ID, including its tags.
func (r *variableRepository) GetByID(ctx context.Context, variableID uuid.UUID) (*model.Variable, error) {
	variable := new(model.Variable)
	if err := r.DB.WithContext(ctx).Preload("Tags").Where("id = ?", variableID).First(variable).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVariableNotFound
		}
		return nil, err
	}
	return variable, nil
}

// CountByProjectID counts the variables across all environments of a project.
func (r *variableRepository) CountByProjectID(ctx context.Context, projectID uuid.UUID) (int64, error) {
	var count int64
	if err := r.DB.WithContext(ctx).
		Model(&model.Variable{}).
		Joins("JOIN environments ON environments.id = variables.environment_id").
		Where("environments.project_id = ?", projectID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByEnvironmentID counts the variables of an environment.
func (r *variableRepository) CountByEnvironmentID(ctx context.Context, environmentID uuid.UUID) (int64, error) {
	var count int64
	if err := r.DB.WithContext(ctx).Model(&model.Variable{}).Where("environment_id = ?", environmentID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// List retrieves all variables of the given environment matching the filter,
// including their tags and active-value status. Variable names are stored
// uppercase, so the search term is normalized to match them.
func (r *variableRepository) List(ctx context.Context, environmentID uuid.UUID, filter VariableListFilter) ([]model.Variable, error) {
	query := r.DB.WithContext(ctx).Preload("Tags").Where("environment_id = ?", environmentID)

	if search := strings.ToUpper(strings.TrimSpace(filter.Search)); search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}
	if len(filter.TagIDs) > 0 {
		query = query.Where("EXISTS (SELECT 1 FROM variable_tags WHERE variable_tags.variable_id = variables.id AND variable_tags.tag_id IN ?)", filter.TagIDs)
	}

	var variables []model.Variable
	if err := query.Find(&variables).Error; err != nil {
		return nil, err
	}

	if err := r.markActive(ctx, variables); err != nil {
		return nil, err
	}

	return variables, nil
}

// markActive sets each variable's Active flag to whether it has an active
// version, using a single query for the whole batch (without loading the
// encrypted version values).
func (r *variableRepository) markActive(ctx context.Context, variables []model.Variable) error {
	if len(variables) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(variables))
	for _, v := range variables {
		ids = append(ids, v.ID)
	}

	var activeIDs []uuid.UUID
	if err := r.DB.WithContext(ctx).
		Model(&model.VariableVersion{}).
		Where("variable_id IN ? AND active = ?", ids, true).
		Distinct("variable_id").
		Pluck("variable_id", &activeIDs).Error; err != nil {
		return err
	}

	active := make(map[uuid.UUID]bool, len(activeIDs))
	for _, id := range activeIDs {
		active[id] = true
	}
	for i := range variables {
		variables[i].Active = active[variables[i].ID]
	}

	return nil
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
