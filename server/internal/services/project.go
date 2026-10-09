package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/repository"
)

// defaultEnvironmentName is the name of the environment created alongside every new project.
const defaultEnvironmentName = "default"

// ProjectService defines the interface for project business logic.
type ProjectService interface {
	Create(ctx context.Context, name string) (*model.Project, error)          // Create creates a new project.
	GetByID(ctx context.Context, projectID uuid.UUID) (*model.Project, error) // GetByID retrieves a project by its ID.
	GetAll(ctx context.Context) ([]model.Project, error)                      // GetAll retrieves all projects.
	SetName(ctx context.Context, projectID uuid.UUID, name string) error      // SetName updates the name of a project.
	Delete(ctx context.Context, projectID uuid.UUID) error                    // Delete deletes a project.
}

// projectService implements ProjectService.
type projectService struct {
	projectRepository     repository.ProjectRepository
	environmentRepository repository.EnvironmentRepository
}

// NewProjectService creates a new projectService with the provided repositories.
func NewProjectService(projectRepository repository.ProjectRepository, environmentRepository repository.EnvironmentRepository) ProjectService {
	return &projectService{projectRepository, environmentRepository}
}

// Create creates a new project along with its "default" environment.
func (s *projectService) Create(ctx context.Context, name string) (*model.Project, error) {
	project := &model.Project{Name: name}
	if err := s.projectRepository.Create(ctx, project); err != nil {
		return nil, err
	}

	defaultEnvironment := &model.Environment{
		ProjectID: project.ID,
		Name:      defaultEnvironmentName,
	}
	if err := s.environmentRepository.Create(ctx, defaultEnvironment); err != nil {
		_ = s.projectRepository.Delete(ctx, project.ID) // Best-effort compensating cleanup: a project without its default environment is unusable.
		return nil, err
	}

	return project, nil
}

// GetByID retrieves a project by its ID.
func (s *projectService) GetByID(ctx context.Context, projectID uuid.UUID) (*model.Project, error) {
	return s.projectRepository.GetByID(ctx, projectID)
}

// GetAll retrieves all projects.
func (s *projectService) GetAll(ctx context.Context) ([]model.Project, error) {
	return s.projectRepository.GetAll(ctx)
}

// SetName updates the name of a project.
func (s *projectService) SetName(ctx context.Context, projectID uuid.UUID, name string) error {
	return s.projectRepository.SetName(ctx, projectID, name)
}

// Delete deletes a project.
func (s *projectService) Delete(ctx context.Context, projectID uuid.UUID) error {
	return s.projectRepository.Delete(ctx, projectID)
}
