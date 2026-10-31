package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/repository"
)

// ProjectDTO is the response shape for a project in list endpoints, including its related counts.
type ProjectDTO struct {
	ID               uuid.UUID `json:"id"`                // Unique identifier of the project.
	Name             string    `json:"name"`              // Name of the project.
	EnvironmentCount int       `json:"environment_count"` // Number of environments in the project.
	TagCount         int       `json:"tag_count"`         // Number of tags in the project.
	CreatedAt        time.Time `json:"created_at"`        // Timestamp when the project was created.
	UpdatedAt        time.Time `json:"updated_at"`        // Timestamp when the project was last updated.
}

// defaultEnvironmentName is the name of the environment created alongside every new project.
const defaultEnvironmentName = "default"

// ProjectService defines the interface for project business logic.
type ProjectService interface {
	Create(ctx context.Context, name string) (*model.Project, error)          // Create creates a new project.
	GetByID(ctx context.Context, projectID uuid.UUID) (*model.Project, error) // GetByID retrieves a project by its ID.
	GetAll(ctx context.Context) ([]ProjectDTO, error)                         // GetAll retrieves all projects with their environment and tag counts.
	SetName(ctx context.Context, projectID uuid.UUID, name string) error      // SetName updates the name of a project.
	Delete(ctx context.Context, projectID uuid.UUID) error                    // Delete deletes a project.
	Clear(ctx context.Context) error                                          // Clear deletes all projects.
}

// projectService implements ProjectService.
type projectService struct {
	projectRepository     repository.ProjectRepository
	environmentRepository repository.EnvironmentRepository
	tagRepository         repository.TagRepository
	audit                 AuditService
}

// NewProjectService creates a new projectService with the provided repositories and audit service.
func NewProjectService(projectRepository repository.ProjectRepository, environmentRepository repository.EnvironmentRepository, tagRepository repository.TagRepository, audit AuditService) ProjectService {
	return &projectService{projectRepository, environmentRepository, tagRepository, audit}
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

	_ = s.audit.Record(ctx, types.ActionCreated, types.ResourceProject, project.ID, project.Name)

	return project, nil
}

// GetByID retrieves a project by its ID.
func (s *projectService) GetByID(ctx context.Context, projectID uuid.UUID) (*model.Project, error) {
	return s.projectRepository.GetByID(ctx, projectID)
}

// GetAll retrieves all projects with their environment and tag counts.
func (s *projectService) GetAll(ctx context.Context) ([]ProjectDTO, error) {
	projects, err := s.projectRepository.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make([]ProjectDTO, 0, len(projects))
	for _, project := range projects {
		environmentCount, err := s.environmentRepository.CountByProjectID(ctx, project.ID)
		if err != nil {
			return nil, err
		}
		tagCount, err := s.tagRepository.CountByProjectID(ctx, project.ID)
		if err != nil {
			return nil, err
		}

		dtos = append(dtos, ProjectDTO{
			ID:               project.ID,
			Name:             project.Name,
			EnvironmentCount: int(environmentCount),
			TagCount:         int(tagCount),
			CreatedAt:        project.CreatedAt,
			UpdatedAt:        project.UpdatedAt,
		})
	}

	return dtos, nil
}

// SetName updates the name of a project.
func (s *projectService) SetName(ctx context.Context, projectID uuid.UUID, name string) error {
	if err := s.projectRepository.SetName(ctx, projectID, name); err != nil {
		return err
	}
	_ = s.audit.Record(ctx, types.ActionUpdated, types.ResourceProject, projectID, name)
	return nil
}

// Delete deletes a project.
func (s *projectService) Delete(ctx context.Context, projectID uuid.UUID) error {
	name := ""
	if project, err := s.projectRepository.GetByID(ctx, projectID); err == nil {
		name = project.Name
	}
	if err := s.projectRepository.Delete(ctx, projectID); err != nil {
		return err
	}
	if name != "" {
		_ = s.audit.Record(ctx, types.ActionDeleted, types.ResourceProject, projectID, name)
	}
	return nil
}

// Clear deletes all projects.
func (s *projectService) Clear(ctx context.Context) error {
	return s.projectRepository.DeleteAll(ctx)
}
