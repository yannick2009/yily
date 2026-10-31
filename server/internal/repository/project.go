package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/yannick2009/yily/internal/domain/model"
	"gorm.io/gorm"
)

var (
	ErrProjectNotFound = errors.New("project not found") // ErrNotFound is returned when a record is not found in the database.
)

// ProjectRepository defines the interface for interacting with project data in the database.
type ProjectRepository interface {
	Create(ctx context.Context, project *model.Project) error                 // Create creates a new project in the database.
	SetName(ctx context.Context, projectID uuid.UUID, name string) error      // SetName updates the name of an existing project in the database.
	Delete(ctx context.Context, projectID uuid.UUID) error                    // Delete deletes a project from the database by its ID.
	GetByID(ctx context.Context, projectID uuid.UUID) (*model.Project, error) // GetByID retrieves a project from the database by its ID.
	GetAll(ctx context.Context) ([]model.Project, error)                      // GetAll retrieves all projects from the database.
	DeleteAll(ctx context.Context) error                                      // DeleteAll deletes all projects from the database.
}

// projectRepository is a struct that implements the ProjectRepository interface.
type projectRepository struct {
	DB *gorm.DB
}

// NewProjectRepository creates a new instance of projectRepository with the provided GORM database connection.
func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{DB: db}
}

// Create creates a new project in the database.
func (r *projectRepository) Create(ctx context.Context, project *model.Project) error {
	return gorm.G[model.Project](r.DB).Create(ctx, project)
}

// SetName updates the name of an existing project in the database.
func (r *projectRepository) SetName(ctx context.Context, projectID uuid.UUID, name string) error {
	_, err := gorm.G[model.Project](r.DB).Where("id = ?", projectID).Update(ctx, "name", name)
	return err
}

// Delete deletes a project from the database by its ID.
func (r *projectRepository) Delete(ctx context.Context, projectID uuid.UUID) error {
	_, err := gorm.G[model.Project](r.DB).Where("id = ?", projectID).Delete(ctx)
	return err
}

// GetByID retrieves a project from the database by its ID.
func (r *projectRepository) GetByID(ctx context.Context, projectID uuid.UUID) (*model.Project, error) {
	project := new(model.Project)
	if err := r.DB.WithContext(ctx).Where("id = ?", projectID).First(project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	return project, nil
}

// GetAll retrieves all projects from the database.
func (r *projectRepository) GetAll(ctx context.Context) ([]model.Project, error) {
	projects, err := gorm.G[model.Project](r.DB).Find(ctx)
	if err != nil {
		return nil, err
	}

	return projects, nil
}

// DeleteAll deletes all projects from the database.
func (r *projectRepository) DeleteAll(ctx context.Context) error {
	_, err := gorm.G[model.Project](r.DB).Where("1 = 1").Delete(ctx)
	return err
}
