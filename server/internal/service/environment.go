package service

import (
	"context"
	"errors"
	"unicode/utf8"
	"uuid"

	"github.com/yannick2009/yily/internal/api"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/repository"
)

// MaxEnvironmentDescriptionLength is the maximum number of characters allowed
// in an environment description.
const MaxEnvironmentDescriptionLength = 50

// ErrEnvironmentDescriptionTooLong is returned when a description exceeds the
// maximum allowed length.
var ErrEnvironmentDescriptionTooLong = errors.New("environment description must be at most 50 characters")

// EnvironmentService defines the interface for environment business logic.
type EnvironmentService interface {
	Create(ctx context.Context, projectID uuid.UUID, name, description string) (*model.Environment, error) // Create creates a new environment.
	Clone(ctx context.Context, sourceEnvironmentID uuid.UUID, newName string, copyTags bool) error         // Clone creates a new environment in the same project with the source's variable names (and optionally tags) but no values; audit entries are recorded for the new environment and each copied variable.
	GetByID(ctx context.Context, environmentID uuid.UUID) (*model.Environment, error)                      // GetByID retrieves an environment by its ID.
	GetAll(ctx context.Context, projectID uuid.UUID) ([]api.EnvironmentSummaryResponse, error)             // GetAll retrieves all environments of a project with their variable counts.
	SetName(ctx context.Context, environmentID uuid.UUID, name string) error                               // SetName updates the name of an environment.
	SetDescription(ctx context.Context, environmentID uuid.UUID, description string) error                 // SetDescription updates the description of an environment.
	Delete(ctx context.Context, environmentID uuid.UUID) error                                             // Delete deletes an environment.
}

// environmentService implements EnvironmentService.
type environmentService struct {
	environmentRepository repository.EnvironmentRepository
	variableRepository    repository.VariableRepository
	audit                 AuditService
}

// NewEnvironmentService creates a new environmentService with the provided repositories and audit service.
func NewEnvironmentService(environmentRepository repository.EnvironmentRepository, variableRepository repository.VariableRepository, audit AuditService) EnvironmentService {
	return &environmentService{environmentRepository, variableRepository, audit}
}

// Create creates a new environment.
func (s *environmentService) Create(ctx context.Context, projectID uuid.UUID, name, description string) (*model.Environment, error) {
	if err := validateDescription(description); err != nil {
		return nil, err
	}

	environment := &model.Environment{
		ProjectID:   projectID,
		Name:        name,
		Description: description,
	}
	if err := s.environmentRepository.Create(ctx, environment); err != nil {
		return nil, err
	}

	_ = s.audit.Record(ctx, types.ActionCreated, types.ResourceEnvironment, environment.ID, environment.Name, nil)

	return environment, nil
}

// Clone creates a new environment in the same project as the source and copies every variable structure (name, and optionally tags). Values are not copied; cloned variables have no versions and therefore show Active=false until a value is set.
func (s *environmentService) Clone(ctx context.Context, sourceEnvironmentID uuid.UUID, newName string, copyTags bool) error {
	source, err := s.environmentRepository.GetByID(ctx, sourceEnvironmentID)
	if err != nil {
		return err
	}

	target, err := s.Create(ctx, source.ProjectID, newName, source.Description)
	if err != nil {
		return err
	}

	variables, err := s.variableRepository.List(ctx, source.ID, repository.VariableListFilter{})
	if err != nil {
		return err
	}

	for _, src := range variables {
		clone := &model.Variable{
			Name:          src.Name,
			EnvironmentID: target.ID,
		}
		if err := s.variableRepository.Create(ctx, clone); err != nil {
			return err
		}

		if copyTags && len(src.Tags) > 0 {
			tagIDs := make([]uuid.UUID, 0, len(src.Tags))
			for _, tag := range src.Tags {
				tagIDs = append(tagIDs, tag.ID)
			}
			if err := s.variableRepository.SetTags(ctx, clone.ID, tagIDs); err != nil {
				return err
			}
		}

		_ = s.audit.Record(ctx, types.ActionCreated, types.ResourceVariable, clone.ID, clone.Name, nil)
	}

	return nil
}

// GetByID retrieves an environment by its ID.
func (s *environmentService) GetByID(ctx context.Context, environmentID uuid.UUID) (*model.Environment, error) {
	return s.environmentRepository.GetByID(ctx, environmentID)
}

// GetAll retrieves all environments of a project with their variable counts.
func (s *environmentService) GetAll(ctx context.Context, projectID uuid.UUID) ([]api.EnvironmentSummaryResponse, error) {
	environments, err := s.environmentRepository.GetAll(ctx, projectID)
	if err != nil {
		return nil, err
	}

	dtos := make([]api.EnvironmentSummaryResponse, 0, len(environments))
	for _, environment := range environments {
		variableCount, err := s.variableRepository.CountByEnvironmentID(ctx, environment.ID)
		if err != nil {
			return nil, err
		}

		dtos = append(dtos, api.EnvironmentSummaryResponse{
			ID:            environment.ID,
			Name:          environment.Name,
			Description:   environment.Description,
			ProjectID:     environment.ProjectID,
			VariableCount: int(variableCount),
			CreatedAt:     environment.CreatedAt,
			UpdatedAt:     environment.UpdatedAt,
		})
	}

	return dtos, nil
}

// SetName updates the name of an environment.
func (s *environmentService) SetName(ctx context.Context, environmentID uuid.UUID, name string) error {
	oldName := ""
	if environment, err := s.environmentRepository.GetByID(ctx, environmentID); err == nil {
		oldName = environment.Name
	}
	if err := s.environmentRepository.SetName(ctx, environmentID, name); err != nil {
		return err
	}
	var metadata map[string]string
	if oldName != "" && oldName != name {
		metadata = map[string]string{"old_name": oldName, "new_name": name}
	}
	_ = s.audit.Record(ctx, types.ActionUpdated, types.ResourceEnvironment, environmentID, name, metadata)
	return nil
}

// SetDescription updates the description of an environment.
func (s *environmentService) SetDescription(ctx context.Context, environmentID uuid.UUID, description string) error {
	if err := validateDescription(description); err != nil {
		return err
	}

	if err := s.environmentRepository.SetDescription(ctx, environmentID, description); err != nil {
		return err
	}
	if environment, err := s.environmentRepository.GetByID(ctx, environmentID); err == nil {
		_ = s.audit.Record(ctx, types.ActionUpdated, types.ResourceEnvironment, environmentID, environment.Name, nil)
	}
	return nil
}

// Delete deletes an environment.
func (s *environmentService) Delete(ctx context.Context, environmentID uuid.UUID) error {
	name := ""
	if environment, err := s.environmentRepository.GetByID(ctx, environmentID); err == nil {
		name = environment.Name
	}
	if err := s.environmentRepository.Delete(ctx, environmentID); err != nil {
		return err
	}
	if name != "" {
		_ = s.audit.Record(ctx, types.ActionDeleted, types.ResourceEnvironment, environmentID, name, nil)
	}
	return nil
}

// validateDescription rejects a description longer than the maximum length.
func validateDescription(description string) error {
	if utf8.RuneCountInString(description) > MaxEnvironmentDescriptionLength {
		return ErrEnvironmentDescriptionTooLong
	}
	return nil
}
