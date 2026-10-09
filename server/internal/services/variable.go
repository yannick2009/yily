package services

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/repository"
)

// VariableService defines the interface for variable business logic, including its versions.
type VariableService interface {
	// variables
	Create(ctx context.Context, environmentID uuid.UUID, name, value string) (*model.Variable, error)     // Create creates a new variable with its first version.
	GetByID(ctx context.Context, variableID uuid.UUID) (*model.Variable, error)                           // GetByID retrieves a variable by its ID.
	GetAll(ctx context.Context, environmentID uuid.UUID) ([]model.Variable, error)                        // GetAll retrieves all variables of an environment.
	GetByTags(ctx context.Context, environmentID uuid.UUID, tagIDs []uuid.UUID) ([]model.Variable, error) // GetByTags retrieves variables that have at least one of the given tags.
	SetName(ctx context.Context, variableID uuid.UUID, name string) error                                 // SetName updates the name of a variable.
	Delete(ctx context.Context, variableID uuid.UUID) error                                               // Delete deletes a variable (and its versions via cascade).
	SetTags(ctx context.Context, variableID uuid.UUID, tagIDs []uuid.UUID) error                          // SetTags replaces the set of tags associated with a variable.

	// versions
	AddVersion(ctx context.Context, variableID uuid.UUID, value string) (*model.VariableVersion, error) // AddVersion adds a new active version to a variable.
	ClearValue(ctx context.Context, variableID uuid.UUID) error                                         // ClearValue clears the variable's current value by deactivating all active versions.
	SetVersionActive(ctx context.Context, variableVersionID uuid.UUID, active bool) error               // SetVersionActive activates or deactivates a version.
	GetCurrentVersion(ctx context.Context, variableID uuid.UUID) (*model.VariableVersion, error)        // GetCurrentVersion returns the current active version of a variable.
	GetAllVersions(ctx context.Context, variableID uuid.UUID) ([]model.VariableVersion, error)          // GetAllVersions returns all versions of a variable, newest first.
}

// variableService implements VariableService.
type variableService struct {
	variableRepository repository.VariableRepository
	versionRepository  repository.VariableVersionRepository
}

// NewVariableService creates a new variableService with the provided repositories.
func NewVariableService(variableRepository repository.VariableRepository, versionRepository repository.VariableVersionRepository) VariableService {
	return &variableService{variableRepository, versionRepository}
}

// Create creates a new variable with its first version.
// If the version creation fails, the variable is removed again to keep the data consistent.
func (s *variableService) Create(ctx context.Context, environmentID uuid.UUID, name, value string) (*model.Variable, error) {
	variable := &model.Variable{
		Name:          name,
		EnvironmentID: environmentID,
	}
	if err := s.variableRepository.Create(ctx, variable); err != nil {
		return nil, err
	}

	// A first version is created only when a non-empty value is provided (whitespace counts as empty).
	value = strings.TrimSpace(value)
	if value != "" {
		if _, err := s.addVersion(ctx, variable.ID, value); err != nil {
			_ = s.variableRepository.Delete(ctx, variable.ID) // Best-effort compensating cleanup: keep the data consistent if the first version fails.
			return nil, err
		}
	}

	return variable, nil
}

// GetByID retrieves a variable by its ID.
func (s *variableService) GetByID(ctx context.Context, variableID uuid.UUID) (*model.Variable, error) {
	return s.variableRepository.GetByID(ctx, variableID)
}

// GetAll retrieves all variables of an environment.
func (s *variableService) GetAll(ctx context.Context, environmentID uuid.UUID) ([]model.Variable, error) {
	return s.variableRepository.GetAll(ctx, environmentID)
}

// GetByTags retrieves the variables of an environment that have at least one of the given tags.
func (s *variableService) GetByTags(ctx context.Context, environmentID uuid.UUID, tagIDs []uuid.UUID) ([]model.Variable, error) {
	return s.variableRepository.GetByTags(ctx, environmentID, tagIDs)
}

// SetName updates the name of a variable.
func (s *variableService) SetName(ctx context.Context, variableID uuid.UUID, name string) error {
	return s.variableRepository.SetName(ctx, variableID, name)
}

// Delete deletes a variable (and its versions via cascade).
func (s *variableService) Delete(ctx context.Context, variableID uuid.UUID) error {
	return s.variableRepository.Delete(ctx, variableID)
}

// SetTags replaces the set of tags associated with a variable.
func (s *variableService) SetTags(ctx context.Context, variableID uuid.UUID, tagIDs []uuid.UUID) error {
	return s.variableRepository.SetTags(ctx, variableID, tagIDs)
}

// SetVersionActive activates or deactivates a version.
func (s *variableService) SetVersionActive(ctx context.Context, variableVersionID uuid.UUID, active bool) error {
	return s.versionRepository.SetActive(ctx, variableVersionID, active)
}

// GetCurrentVersion returns the current active version of a variable.
func (s *variableService) GetCurrentVersion(ctx context.Context, variableID uuid.UUID) (*model.VariableVersion, error) {
	return s.versionRepository.GetCurrentActive(ctx, variableID)
}

// GetAllVersions returns all versions of a variable, newest first.
func (s *variableService) GetAllVersions(ctx context.Context, variableID uuid.UUID) ([]model.VariableVersion, error) {
	return s.versionRepository.GetAll(ctx, variableID)
}

// AddVersion adds a new active version to a variable and returns it.
func (s *variableService) AddVersion(ctx context.Context, variableID uuid.UUID, value string) (*model.VariableVersion, error) {
	return s.addVersion(ctx, variableID, value)
}

// ClearValue clears the variable's current value by deactivating all its active versions.
func (s *variableService) ClearValue(ctx context.Context, variableID uuid.UUID) error {
	return s.versionRepository.DeactivateAll(ctx, variableID)
}

// addVersion creates a new active version for the given variable.
func (s *variableService) addVersion(ctx context.Context, variableID uuid.UUID, value string) (*model.VariableVersion, error) {
	version := &model.VariableVersion{
		Value:      value,
		Active:     true,
		VariableID: variableID,
	}

	if err := s.versionRepository.Create(ctx, version); err != nil {
		return nil, err
	}

	return version, nil
}
