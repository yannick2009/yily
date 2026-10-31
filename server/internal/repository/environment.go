package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
	"gorm.io/gorm"
)

var (
	ErrEnvironmentNotFound = errors.New("environment not found") // ErrNotFound is returned when a record is not found in the database.
)

// EnvironmentRepository defines the interface for interacting with environment data in the database.
type EnvironmentRepository interface {
	Create(ctx context.Context, environment *model.Environment) error                      // Create creates a new environment in the database.
	SetName(ctx context.Context, environmentID uuid.UUID, name string) error               // SetName updates the name of an existing environment in the database.
	SetDescription(ctx context.Context, environmentID uuid.UUID, description string) error // SetDescription updates the description of an existing environment in the database.
	Delete(ctx context.Context, environmentID uuid.UUID) error                             // Delete deletes an environment from the database by its ID.
	GetByID(ctx context.Context, environmentID uuid.UUID) (*model.Environment, error)      // GetByID retrieves an environment from the database by its ID.
	GetAll(ctx context.Context, projectID uuid.UUID) ([]model.Environment, error)          // GetAll retrieves all environments from the database.
}

// environmentRepository is a struct that implements the EnvironmentRepository interface.
type environmentRepository struct {
	DB *gorm.DB
}

// NewEnvironmentRepository creates a new instance of environmentRepository with the provided GORM database connection.
func NewEnvironmentRepository(db *gorm.DB) EnvironmentRepository {
	return &environmentRepository{DB: db}
}

// Create creates a new environment in the database.
func (r *environmentRepository) Create(ctx context.Context, environment *model.Environment) error {
	return gorm.G[model.Environment](r.DB).Create(ctx, environment)
}

// SetName updates the name of an existing environment in the database.
func (r *environmentRepository) SetName(ctx context.Context, environmentID uuid.UUID, name string) error {
	_, err := gorm.G[model.Environment](r.DB).Where("id = ?", environmentID).Update(ctx, "name", name)
	return err
}

// SetDescription updates the description of an existing environment in the database.
func (r *environmentRepository) SetDescription(ctx context.Context, environmentID uuid.UUID, description string) error {
	_, err := gorm.G[model.Environment](r.DB).Where("id = ?", environmentID).Update(ctx, "description", description)
	return err
}

// Delete deletes an environment from the database by its ID.
func (r *environmentRepository) Delete(ctx context.Context, environmentID uuid.UUID) error {
	_, err := gorm.G[model.Environment](r.DB).Where("id = ?", environmentID).Delete(ctx)
	return err
}

// GetByID retrieves an environment from the database by its ID.
func (r *environmentRepository) GetByID(ctx context.Context, environmentID uuid.UUID) (*model.Environment, error) {
	environment := new(model.Environment)
	if err := r.DB.WithContext(ctx).Where("id = ?", environmentID).First(environment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEnvironmentNotFound
		}
		return nil, err
	}
	return environment, nil
}

// GetAll retrieves all environments from the database.
func (r *environmentRepository) GetAll(ctx context.Context, projectID uuid.UUID) ([]model.Environment, error) {
	environments, err := gorm.G[model.Environment](r.DB).Where("project_id = ?", projectID).Find(ctx)
	if err != nil {
		return nil, err
	}
	return environments, nil
}
