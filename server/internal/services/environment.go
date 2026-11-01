package services

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/repository"
)

// EnvironmentDTO is the response shape for an environment in list endpoints, including its variable count.
type EnvironmentDTO struct {
	ID            uuid.UUID `json:"id"`             // Unique identifier of the environment.
	Name          string    `json:"name"`           // Name of the environment.
	Description   string    `json:"description"`    // Description of the environment.
	ProjectID     uuid.UUID `json:"project_id"`     // Foreign key referencing the associated project.
	VariableCount int       `json:"variable_count"` // Number of variables in the environment.
	CreatedAt     time.Time `json:"created_at"`     // Timestamp when the environment was created.
	UpdatedAt     time.Time `json:"updated_at"`     // Timestamp when the environment was last updated.
}

// MaxEnvironmentDescriptionLength is the maximum number of characters allowed
// in an environment description.
const MaxEnvironmentDescriptionLength = 50

// ErrEnvironmentDescriptionTooLong is returned when a description exceeds the
// maximum allowed length.
var ErrEnvironmentDescriptionTooLong = errors.New("environment description must be at most 50 characters")

// EnvironmentService defines the interface for environment business logic.
type EnvironmentService interface {
	Create(ctx context.Context, projectID uuid.UUID, name, description string) (*model.Environment, error) // Create creates a new environment.
	GetByID(ctx context.Context, environmentID uuid.UUID) (*model.Environment, error)                      // GetByID retrieves an environment by its ID.
	GetAll(ctx context.Context, projectID uuid.UUID) ([]EnvironmentDTO, error)                             // GetAll retrieves all environments of a project with their variable counts.
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

	_ = s.audit.Record(ctx, types.ActionCreated, types.ResourceEnvironment, environment.ID, environment.Name)

	return environment, nil
}

// GetByID retrieves an environment by its ID.
func (s *environmentService) GetByID(ctx context.Context, environmentID uuid.UUID) (*model.Environment, error) {
	return s.environmentRepository.GetByID(ctx, environmentID)
}

// GetAll retrieves all environments of a project with their variable counts.
func (s *environmentService) GetAll(ctx context.Context, projectID uuid.UUID) ([]EnvironmentDTO, error) {
	environments, err := s.environmentRepository.GetAll(ctx, projectID)
	if err != nil {
		return nil, err
	}

	dtos := make([]EnvironmentDTO, 0, len(environments))
	for _, environment := range environments {
		variableCount, err := s.variableRepository.CountByEnvironmentID(ctx, environment.ID)
		if err != nil {
			return nil, err
		}

		dtos = append(dtos, EnvironmentDTO{
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
	if err := s.environmentRepository.SetName(ctx, environmentID, name); err != nil {
		return err
	}
	_ = s.audit.Record(ctx, types.ActionUpdated, types.ResourceEnvironment, environmentID, name)
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
		_ = s.audit.Record(ctx, types.ActionUpdated, types.ResourceEnvironment, environmentID, environment.Name)
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
		_ = s.audit.Record(ctx, types.ActionDeleted, types.ResourceEnvironment, environmentID, name)
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
