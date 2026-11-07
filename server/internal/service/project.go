package service

import (
	"context"
	"uuid"

	"github.com/yannick2009/yily/internal/api"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/repository"
)

// defaultEnvironmentName is the name of the environment created alongside every new project.
const defaultEnvironmentName = "default"

// ProjectService defines the interface for project business logic.
type ProjectService interface {
	Create(ctx context.Context, name string) (*model.Project, error)          // Create creates a new project.
	GetByID(ctx context.Context, projectID uuid.UUID) (*model.Project, error) // GetByID retrieves a project by its ID.
	GetAll(ctx context.Context) ([]api.ProjectSummaryResponse, error)         // GetAll retrieves all projects with their environment and tag counts.
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

	_ = s.audit.Record(ctx, types.ActionCreated, types.ResourceProject, project.ID, project.Name, nil)

	return project, nil
}

// GetByID retrieves a project by its ID.
func (s *projectService) GetByID(ctx context.Context, projectID uuid.UUID) (*model.Project, error) {
	return s.projectRepository.GetByID(ctx, projectID)
}

// GetAll retrieves all projects with their environment and tag counts.
func (s *projectService) GetAll(ctx context.Context) ([]api.ProjectSummaryResponse, error) {
	projects, err := s.projectRepository.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make([]api.ProjectSummaryResponse, 0, len(projects))
	for _, project := range projects {
		environmentCount, err := s.environmentRepository.CountByProjectID(ctx, project.ID)
		if err != nil {
			return nil, err
		}
		tagCount, err := s.tagRepository.CountByProjectID(ctx, project.ID)
		if err != nil {
			return nil, err
		}

		dtos = append(dtos, api.ProjectSummaryResponse{
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
	oldName := ""
	if project, err := s.projectRepository.GetByID(ctx, projectID); err == nil {
		oldName = project.Name
	}
	if err := s.projectRepository.SetName(ctx, projectID, name); err != nil {
		return err
	}
	var metadata map[string]string
	if oldName != "" && oldName != name {
		metadata = map[string]string{"old_name": oldName, "new_name": name}
	}
	_ = s.audit.Record(ctx, types.ActionUpdated, types.ResourceProject, projectID, name, metadata)
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
		_ = s.audit.Record(ctx, types.ActionDeleted, types.ResourceProject, projectID, name, nil)
	}
	return nil
}

// Clear deletes all projects.
func (s *projectService) Clear(ctx context.Context) error {
	if err := s.projectRepository.DeleteAll(ctx); err != nil {
		return err
	}
	_ = s.audit.Record(ctx, types.ActionDeleted, types.ResourceProject, uuid.Nil(), "all projects", nil)
	return nil
}
