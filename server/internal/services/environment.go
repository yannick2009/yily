package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/repository"
)

// EnvironmentService defines the interface for environment business logic.
type EnvironmentService interface {
	Create(ctx context.Context, projectID uuid.UUID, name, description string) (*model.Environment, error) // Create creates a new environment.
	GetByID(ctx context.Context, environmentID uuid.UUID) (*model.Environment, error)                      // GetByID retrieves an environment by its ID.
	GetAll(ctx context.Context, projectID uuid.UUID) ([]model.Environment, error)                          // GetAll retrieves all environments of a project.
	SetName(ctx context.Context, environmentID uuid.UUID, name string) error                               // SetName updates the name of an environment.
	SetDescription(ctx context.Context, environmentID uuid.UUID, description string) error                 // SetDescription updates the description of an environment.
	Delete(ctx context.Context, environmentID uuid.UUID) error                                             // Delete deletes an environment.
}

// environmentService implements EnvironmentService.
type environmentService struct {
	environmentRepository repository.EnvironmentRepository
	audit                 AuditService
}

// NewEnvironmentService creates a new environmentService with the provided EnvironmentRepository and audit service.
func NewEnvironmentService(environmentRepository repository.EnvironmentRepository, audit AuditService) EnvironmentService {
	return &environmentService{environmentRepository, audit}
}

// Create creates a new environment.
func (s *environmentService) Create(ctx context.Context, projectID uuid.UUID, name, description string) (*model.Environment, error) {
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

// GetAll retrieves all environments of a project.
func (s *environmentService) GetAll(ctx context.Context, projectID uuid.UUID) ([]model.Environment, error) {
	return s.environmentRepository.GetAll(ctx, projectID)
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
