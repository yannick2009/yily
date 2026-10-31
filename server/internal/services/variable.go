package services

import (
	"bytes"
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/crypto"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/env"
	"github.com/yannick2009/yily/internal/repository"
)

var (
	ErrDecryptionFailed = errors.New("failed to decrypt secret value")
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
	Import(ctx context.Context, environmentID uuid.UUID, data []byte, fileType types.FileType) error      // Import imports variables from a file (JSON, YAML or .env), creating new ones and updating existing ones whose value changed.

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
	audit              AuditService
	masterKey          []byte
}

// NewVariableService creates a new variableService with the provided repositories, audit service and master key.
func NewVariableService(variableRepository repository.VariableRepository, versionRepository repository.VariableVersionRepository, audit AuditService, masterKey []byte) VariableService {
	return &variableService{
		variableRepository: variableRepository,
		versionRepository:  versionRepository,
		audit:              audit,
		masterKey:          bytes.Clone(masterKey),
	}
}

// Create creates a new variable with its first version.
// If the version creation fails, the variable is removed again to keep the data consistent.
func (s *variableService) Create(ctx context.Context, environmentID uuid.UUID, name, value string) (*model.Variable, error) {
	variable := &model.Variable{
		Name:          normalizeVariableName(name),
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

	_ = s.audit.Record(ctx, types.ActionCreated, types.ResourceVariable, variable.ID, variable.Name)

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
	normalized := normalizeVariableName(name)
	if err := s.variableRepository.SetName(ctx, variableID, normalized); err != nil {
		return err
	}
	_ = s.audit.Record(ctx, types.ActionUpdated, types.ResourceVariable, variableID, normalized)
	return nil
}

// Delete deletes a variable (and its versions via cascade).
func (s *variableService) Delete(ctx context.Context, variableID uuid.UUID) error {
	name := ""
	if variable, err := s.variableRepository.GetByID(ctx, variableID); err == nil {
		name = variable.Name
	}
	if err := s.variableRepository.Delete(ctx, variableID); err != nil {
		return err
	}
	if name != "" {
		_ = s.audit.Record(ctx, types.ActionDeleted, types.ResourceVariable, variableID, name)
	}
	return nil
}

// SetTags replaces the set of tags associated with a variable.
func (s *variableService) SetTags(ctx context.Context, variableID uuid.UUID, tagIDs []uuid.UUID) error {
	if err := s.variableRepository.SetTags(ctx, variableID, tagIDs); err != nil {
		return err
	}
	s.auditVariable(ctx, types.ActionUpdated, variableID)
	return nil
}

// SetVersionActive activates or deactivates a version.
func (s *variableService) SetVersionActive(ctx context.Context, variableVersionID uuid.UUID, active bool) error {
	return s.versionRepository.SetActive(ctx, variableVersionID, active)
}

// GetCurrentVersion returns the current active version of a variable with its decrypted value.
func (s *variableService) GetCurrentVersion(ctx context.Context, variableID uuid.UUID) (*model.VariableVersion, error) {
	version, err := s.versionRepository.GetCurrentActive(ctx, variableID)
	if err != nil {
		return nil, err
	}

	decrypted, err := crypto.Decrypt(version.Value, s.masterKey, version.VariableID, version.Version)
	if err != nil {
		return nil, ErrDecryptionFailed
	}
	version.Value = decrypted

	s.auditVariable(ctx, types.ActionRevealed, variableID)

	return version, nil
}

// GetAllVersions returns all versions of a variable with their decrypted values, newest first.
func (s *variableService) GetAllVersions(ctx context.Context, variableID uuid.UUID) ([]model.VariableVersion, error) {
	versions, err := s.versionRepository.GetAll(ctx, variableID)
	if err != nil {
		return nil, err
	}

	for i := range versions {
		decrypted, err := crypto.Decrypt(versions[i].Value, s.masterKey, versions[i].VariableID, versions[i].Version)
		if err != nil {
			return nil, ErrDecryptionFailed
		}
		versions[i].Value = decrypted
	}

	return versions, nil
}

// AddVersion adds a new active version to a variable and returns it.
func (s *variableService) AddVersion(ctx context.Context, variableID uuid.UUID, value string) (*model.VariableVersion, error) {
	version, err := s.addVersion(ctx, variableID, value)
	if err != nil {
		return nil, err
	}
	s.auditVariable(ctx, types.ActionUpdated, variableID)
	return version, nil
}

// ClearValue clears the variable's current value by deactivating all its active versions.
func (s *variableService) ClearValue(ctx context.Context, variableID uuid.UUID) error {
	if err := s.versionRepository.DeactivateAll(ctx, variableID); err != nil {
		return err
	}
	s.auditVariable(ctx, types.ActionUpdated, variableID)
	return nil
}

// addVersion creates a new active encrypted version for the given variable.
func (s *variableService) addVersion(ctx context.Context, variableID uuid.UUID, value string) (*model.VariableVersion, error) {
	nextVersion, err := s.versionRepository.GetNextVersion(ctx, variableID)
	if err != nil {
		return nil, err
	}

	encryptedValue, err := crypto.Encrypt(value, s.masterKey, variableID, nextVersion)
	if err != nil {
		return nil, err
	}

	version := &model.VariableVersion{
		VariableID: variableID,
		Version:    nextVersion,
		Value:      encryptedValue,
		Active:     true,
	}

	if err := s.versionRepository.Create(ctx, version); err != nil {
		return nil, err
	}

	// Keep plaintext value in memory to avoid leaking ciphertext in API responses.
	version.Value = value
	return version, nil
}

// Import imports variables from a file (JSON, YAML or .env) into the given environment.
// New variables are created; existing ones get a new version only if their value changed.
func (s *variableService) Import(ctx context.Context, environmentID uuid.UUID, data []byte, fileType types.FileType) error {
	var raw map[string]string
	var err error

	switch fileType {
	case types.FileTypeJSON:
		raw, err = env.ParseJSONVariables(data)
	case types.FileTypeYAML:
		raw, err = env.ParseYAMLVariables(data)
	case types.FileTypeEnv:
		raw, err = env.ParseEnvFileVariables(data)
	default:
		return types.ErrUnsupportedFileType
	}
	if err != nil {
		return err
	}

	// Normalize names (trim + uppercase) and detect name collisions within the file.
	variables := make(map[string]string, len(raw))
	for name, value := range raw {
		normalized := normalizeVariableName(name)
		if _, exists := variables[normalized]; exists {
			return env.ErrDuplicateVariableName
		}
		variables[normalized] = strings.TrimSpace(value)
	}

	return s.variableRepository.Import(ctx, environmentID, variables)
}

// normalizeVariableName returns a variable name trimmed and uppercased.
func normalizeVariableName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}

// auditVariable records an audit event for a variable, resolving its name best-effort.
func (s *variableService) auditVariable(ctx context.Context, action types.Action, variableID uuid.UUID) {
	name := ""
	if variable, err := s.variableRepository.GetByID(ctx, variableID); err == nil {
		name = variable.Name
	}
	_ = s.audit.Record(ctx, action, types.ResourceVariable, variableID, name)
}
