package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"uuid"

	"github.com/yannick2009/yily/internal/crypto"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/env"
	"github.com/yannick2009/yily/internal/repository"
)

var (
	ErrDecryptionFailed           = errors.New("failed to decrypt secret value")
	ErrInvalidVariableVersion     = errors.New("variable version must be zero or a positive integer")
	ErrDuplicateVariableSelection = errors.New("a variable can be selected only once per request")
)

// VariableService defines the interface for variable business logic, including its versions.
type VariableService interface {
	// variables
	Create(ctx context.Context, environmentID uuid.UUID, name, value string) (*model.Variable, error)                  // Create creates a new variable with its first version.
	GetByID(ctx context.Context, variableID uuid.UUID) (*model.Variable, error)                                        // GetByID retrieves a variable by its ID.
	List(ctx context.Context, environmentID uuid.UUID, filter VariableListFilter) ([]model.Variable, error)            // List retrieves the variables of an environment matching the given filter.
	GetValues(ctx context.Context, environmentID uuid.UUID, selections []VariableSelection) (map[string]string, error) // GetValues retrieves requested secret values, each at its own version.
	SetName(ctx context.Context, variableID uuid.UUID, name string) error                                              // SetName updates the name of a variable.
	Delete(ctx context.Context, variableID uuid.UUID) error                                                            // Delete deletes a variable (and its versions via cascade).
	SetTags(ctx context.Context, variableID uuid.UUID, tagIDs []uuid.UUID) error                                       // SetTags replaces the set of tags associated with a variable.
	Import(ctx context.Context, environmentID uuid.UUID, data []byte, fileType types.FileType, filename string) error  // Import imports variables from a file (JSON, YAML or .env), creating new ones and updating existing ones whose value changed.
	Export(ctx context.Context, environmentID uuid.UUID, fileType types.FileType) ([]byte, error)                      // Export serializes all variables of an environment into the given file format.

	// versions
	AddVersion(ctx context.Context, variableID uuid.UUID, value string) (*model.VariableVersion, error) // AddVersion adds a new active version to a variable.
	ClearValue(ctx context.Context, variableID uuid.UUID) error                                         // ClearValue clears the variable's current value by deactivating all active versions.
	SetVersionActive(ctx context.Context, variableVersionID uuid.UUID, active bool) error               // SetVersionActive activates or deactivates a version.
	GetCurrentVersion(ctx context.Context, variableID uuid.UUID) (*model.VariableVersion, error)        // GetCurrentVersion returns the current active version of a variable.
	GetValue(ctx context.Context, variableID uuid.UUID) (string, error)                                 // GetValue returns the current decrypted value of a variable.
	GetAllVersions(ctx context.Context, variableID uuid.UUID) ([]model.VariableVersion, error)          // GetAllVersions returns all versions of a variable, newest first.
}

// VariableListFilter filters the variable listing of an environment.
type VariableListFilter struct {
	Search string                       // Search matches variable names case-insensitively (empty for no filter).
	TagIDs []uuid.UUID                  // TagIDs filters to variables that have at least one of these tags (empty for no filter).
	Sort   repository.VariableSortField // Sort column; empty means default (name).
	Order  repository.VariableSortOrder // Order direction; empty means default (ascending).
}

// VariableSelection selects one variable and, optionally, a specific version.
// A zero version selects the variable's active version.
type VariableSelection struct {
	Name    string
	Version int
}

// variableService implements VariableService.
type variableService struct {
	variableRepository    repository.VariableRepository
	versionRepository     repository.VariableVersionRepository
	environmentRepository repository.EnvironmentRepository
	audit                 AuditService
	masterKey             []byte
}

// NewVariableService creates a new variableService with the provided repositories, audit service and master key.
func NewVariableService(variableRepository repository.VariableRepository, versionRepository repository.VariableVersionRepository, environmentRepository repository.EnvironmentRepository, audit AuditService, masterKey []byte) VariableService {
	return &variableService{
		variableRepository:    variableRepository,
		versionRepository:     versionRepository,
		environmentRepository: environmentRepository,
		audit:                 audit,
		masterKey:             bytes.Clone(masterKey),
	}
}

// Create creates a new variable with its first version.
func (s *variableService) Create(ctx context.Context, environmentID uuid.UUID, name, value string) (*model.Variable, error) {
	variable, err := s.create(ctx, environmentID, name, value)
	if err != nil {
		return nil, err
	}

	_ = s.audit.Record(ctx, types.ActionCreated, types.ResourceVariable, variable.ID, variable.Name, nil)

	return variable, nil
}

// create creates a new variable with its first version without recording
// an audit event. Callers that batch many variables (e.g. import) record a single
// aggregate event instead of one per variable.
// If the version creation fails, the variable is removed again to keep the data consistent.
func (s *variableService) create(ctx context.Context, environmentID uuid.UUID, name, value string) (*model.Variable, error) {
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

	return variable, nil
}

// GetByID retrieves a variable by its ID.
func (s *variableService) GetByID(ctx context.Context, variableID uuid.UUID) (*model.Variable, error) {
	return s.variableRepository.GetByID(ctx, variableID)
}

// List retrieves the variables of an environment matching the given filter.
func (s *variableService) List(ctx context.Context, environmentID uuid.UUID, filter VariableListFilter) ([]model.Variable, error) {
	return s.variableRepository.List(ctx, environmentID, repository.VariableListFilter{
		Search: filter.Search,
		TagIDs: filter.TagIDs,
		Sort:   filter.Sort,
		Order:  filter.Order,
	})
}

// SetName updates the name of a variable.
func (s *variableService) SetName(ctx context.Context, variableID uuid.UUID, name string) error {
	normalized := normalizeVariableName(name)
	oldName := ""
	if variable, err := s.variableRepository.GetByID(ctx, variableID); err == nil {
		oldName = variable.Name
	}
	if err := s.variableRepository.SetName(ctx, variableID, normalized); err != nil {
		return err
	}
	var metadata map[string]string
	if oldName != "" && oldName != normalized {
		metadata = map[string]string{"old_name": oldName, "new_name": normalized}
	}
	_ = s.audit.Record(ctx, types.ActionUpdated, types.ResourceVariable, variableID, normalized, metadata)
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
		_ = s.audit.Record(ctx, types.ActionDeleted, types.ResourceVariable, variableID, name, nil)
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

// GetCurrentVersion returns the current active version of a variable, without its value.
func (s *variableService) GetCurrentVersion(ctx context.Context, variableID uuid.UUID) (*model.VariableVersion, error) {
	version, err := s.versionRepository.GetCurrentActive(ctx, variableID)
	if err != nil {
		return nil, err
	}
	if version == nil {
		return nil, repository.ErrVariableVersionNotFound
	}

	// Never expose the value here; use GetValue.
	version.Value = ""

	return version, nil
}

// GetValue returns the current decrypted value of a variable.
func (s *variableService) GetValue(ctx context.Context, variableID uuid.UUID) (string, error) {
	version, err := s.versionRepository.GetCurrentActive(ctx, variableID)
	if err != nil {
		return "", err
	}

	value, err := s.decryptVersion(version)
	if err != nil {
		return "", err
	}

	s.auditVariable(ctx, types.ActionRevealed, variableID)

	return value, nil
}

// GetValues retrieves requested variable values from one environment, each at
// its own selected version. A zero version selects the variable's active
// version. A name may appear only once; duplicate selections are rejected so a
// request cannot silently pick one of two conflicting versions.
func (s *variableService) GetValues(ctx context.Context, environmentID uuid.UUID, selections []VariableSelection) (map[string]string, error) {
	prepared := make([]VariableSelection, 0, len(selections))
	seen := make(map[string]struct{}, len(selections))
	for _, selection := range selections {
		if selection.Version < 0 {
			return nil, ErrInvalidVariableVersion
		}
		name := normalizeVariableName(selection.Name)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateVariableSelection, name)
		}
		seen[name] = struct{}{}
		prepared = append(prepared, VariableSelection{Name: name, Version: selection.Version})
	}

	if _, err := s.environmentRepository.GetByID(ctx, environmentID); err != nil {
		return nil, err
	}
	if len(prepared) == 0 {
		return map[string]string{}, nil
	}

	names := make([]string, 0, len(prepared))
	requestedVersion := make(map[string]int, len(prepared))
	for _, selection := range prepared {
		names = append(names, selection.Name)
		requestedVersion[selection.Name] = selection.Version
	}

	variables, err := s.variableRepository.GetByNames(ctx, environmentID, names)
	if err != nil {
		return nil, err
	}
	if len(variables) == 0 {
		return map[string]string{}, nil
	}

	// Group variable IDs by requested version so each distinct version is
	// fetched in one query; version 0 means the active version.
	versionGroups := make(map[int][]uuid.UUID)
	for _, variable := range variables {
		version := requestedVersion[variable.Name]
		versionGroups[version] = append(versionGroups[version], variable.ID)
	}

	versions := make([]int, 0, len(versionGroups))
	for version := range versionGroups {
		versions = append(versions, version)
	}
	sort.Ints(versions)

	versionByVariable := make(map[uuid.UUID]model.VariableVersion, len(variables))
	for _, version := range versions {
		found, err := s.versionRepository.GetForVariables(ctx, versionGroups[version], version)
		if err != nil {
			return nil, err
		}
		for _, variableVersion := range found {
			if _, exists := versionByVariable[variableVersion.VariableID]; !exists {
				versionByVariable[variableVersion.VariableID] = variableVersion
			}
		}
	}

	values := make(map[string]string, len(variables))
	revealed := make([]model.Variable, 0, len(variables))
	for _, variable := range variables {
		variableVersion, exists := versionByVariable[variable.ID]
		if !exists {
			if requestedVersion[variable.Name] > 0 {
				return nil, repository.ErrVariableVersionNotFound
			}
			// Match export behavior for a variable that has no active version.
			values[variable.Name] = ""
			continue
		}

		value, err := s.decryptVersion(&variableVersion)
		if err != nil {
			return nil, err
		}
		values[variable.Name] = value
		revealed = append(revealed, variable)
	}
	for _, variable := range revealed {
		_ = s.audit.Record(ctx, types.ActionRevealed, types.ResourceVariable, variable.ID, variable.Name, nil)
	}

	return values, nil
}

// GetAllVersions returns all versions of a variable, newest first, without their values.
func (s *variableService) GetAllVersions(ctx context.Context, variableID uuid.UUID) ([]model.VariableVersion, error) {
	versions, err := s.versionRepository.GetAll(ctx, variableID)
	if err != nil {
		return nil, err
	}

	for i := range versions {
		// Never expose values here; use GetValue.
		versions[i].Value = ""
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

	// Never expose the value in API responses; use GetValue.
	version.Value = ""
	return version, nil
}

// Import imports variables from a file (JSON, YAML or .env) into the given environment.
// New variables are created; existing ones get a new version only if their value changed.
func (s *variableService) Import(ctx context.Context, environmentID uuid.UUID, data []byte, fileType types.FileType, filename string) error {
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

	// Index existing variables by name so we can tell which to create and which to update.
	existing, err := s.variableRepository.List(ctx, environmentID, repository.VariableListFilter{})
	if err != nil {
		return err
	}
	byName := make(map[string]*model.Variable, len(existing))
	for i := range existing {
		byName[existing[i].Name] = &existing[i]
	}

	// Deterministic order keeps the result reproducible regardless of map iteration.
	names := make([]string, 0, len(variables))
	for name := range variables {
		names = append(names, name)
	}
	sort.Strings(names)

	createdCount := 0
	updatedCount := 0
	for _, name := range names {
		value := variables[name]
		variable, exists := byName[name]
		if !exists {
			if _, err := s.create(ctx, environmentID, name, value); err != nil {
				return err
			}
			createdCount++
			continue
		}
		changed, err := s.applyImportValue(ctx, variable.ID, value)
		if err != nil {
			return err
		}
		if changed {
			updatedCount++
		}
	}

	// Record a single import event rather than one per created/updated variable.
	environment, err := s.environmentRepository.GetByID(ctx, environmentID)
	if err != nil {
		return err
	}
	_ = s.audit.Record(ctx, types.ActionImport, types.ResourceEnvironment, environmentID, environment.Name, map[string]string{
		"file":          filepath.Base(filename),
		"file_type":     string(fileType),
		"created_count": strconv.Itoa(createdCount),
		"updated_count": strconv.Itoa(updatedCount),
	})

	return nil
}

// applyImportValue adds a new encrypted version to an existing variable when its
// current decrypted value differs from the imported one. An empty value is a
// no-op, so re-importing an unchanged file does not create spurious versions.
// It reports whether a new version was actually added.
func (s *variableService) applyImportValue(ctx context.Context, variableID uuid.UUID, value string) (bool, error) {
	if value == "" {
		return false, nil
	}

	version, err := s.versionRepository.GetCurrentActive(ctx, variableID)
	if err == nil {
		if current, decErr := s.decryptVersion(version); decErr == nil && current == value {
			return false, nil // unchanged
		}
	} else if !errors.Is(err, repository.ErrVariableVersionNotFound) {
		return false, err
	}

	if _, err := s.addVersion(ctx, variableID, value); err != nil {
		return false, err
	}
	return true, nil
}

// Export serializes all variables of an environment (name → current decrypted value) into the given file format.
func (s *variableService) Export(ctx context.Context, environmentID uuid.UUID, fileType types.FileType) ([]byte, error) {
	variables, err := s.variableRepository.List(ctx, environmentID, repository.VariableListFilter{})
	if err != nil {
		return nil, err
	}

	values := make(map[string]string, len(variables))
	for _, variable := range variables {
		value := ""
		version, err := s.versionRepository.GetCurrentActive(ctx, variable.ID)
		switch {
		case err == nil:
			value, err = s.decryptVersion(version)
			if err != nil {
				return nil, err
			}
		case errors.Is(err, repository.ErrVariableVersionNotFound):
			// The variable has no value yet — export it as empty.
		default:
			return nil, err
		}
		values[variable.Name] = value
	}

	var content []byte
	switch fileType {
	case types.FileTypeJSON:
		content, err = env.FormatJSONVariables(values)
	case types.FileTypeYAML:
		content, err = env.FormatYAMLVariables(values)
	case types.FileTypeEnv:
		content, err = env.FormatEnvFileVariables(values)
	default:
		return nil, types.ErrUnsupportedFileType
	}
	if err != nil {
		return nil, err
	}

	// Record a single export event rather than a reveal per variable.
	environment, err := s.environmentRepository.GetByID(ctx, environmentID)
	if err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, types.ActionExport, types.ResourceEnvironment, environmentID, environment.Name, map[string]string{
		"file_type":      string(fileType),
		"variable_count": strconv.Itoa(len(variables)),
	})

	return content, nil
}

// normalizeVariableName uppercases a variable name and replaces whitespace
// runs with a single underscore.
func normalizeVariableName(name string) string {
	return strings.Join(strings.Fields(strings.ToUpper(name)), "_")
}

// decryptVersion returns the decrypted value of a version.
func (s *variableService) decryptVersion(version *model.VariableVersion) (string, error) {
	value, err := crypto.Decrypt(version.Value, s.masterKey, version.VariableID, version.Version)
	if err != nil {
		return "", ErrDecryptionFailed
	}
	return value, nil
}

// auditVariable records an audit event for a variable, resolving its name best-effort.
func (s *variableService) auditVariable(ctx context.Context, action types.Action, variableID uuid.UUID) {
	name := ""
	if variable, err := s.variableRepository.GetByID(ctx, variableID); err == nil {
		name = variable.Name
	}
	_ = s.audit.Record(ctx, action, types.ResourceVariable, variableID, name, nil)
}
