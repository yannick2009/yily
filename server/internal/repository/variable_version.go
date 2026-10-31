package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
	"gorm.io/gorm"
)

var (
	ErrVariableVersionNotFound = errors.New("variable latest version not found")
	ErrDuplicateVersion        = errors.New("duplicate variable version")
)

// VariableVersionRepository defines the interface for interacting with variable version data.
type VariableVersionRepository interface {
	GetCurrentActive(ctx context.Context, variableID uuid.UUID) (*model.VariableVersion, error) // GetCurrentActive retrieves the current active version.
	GetNextVersion(ctx context.Context, variableID uuid.UUID) (int, error)                      // GetNextVersion computes the next version number for a variable.
	Create(ctx context.Context, variableVersion *model.VariableVersion) error                   // Create persists a variable version in the database.
	SetActive(ctx context.Context, variableVersionID uuid.UUID, active bool) error              // SetActive updates the active status.
	DeactivateAll(ctx context.Context, variableID uuid.UUID) error                              // DeactivateAll deactivates all versions.
	GetAll(ctx context.Context, variableID uuid.UUID) ([]model.VariableVersion, error)          // GetAll retrieves all versions.
}

// variableVersionRepository is a struct that implements the VariableVersionRepository interface.
type variableVersionRepository struct {
	DB *gorm.DB
}

// NewVariableVersionRepository creates a new instance of variableVersionRepository.
func NewVariableVersionRepository(db *gorm.DB) VariableVersionRepository {
	return &variableVersionRepository{DB: db}
}

// GetNextVersion computes the next version number for a variable.
func (r *variableVersionRepository) GetNextVersion(ctx context.Context, variableID uuid.UUID) (int, error) {
	var maxVersion int
	err := r.DB.WithContext(ctx).Model(&model.VariableVersion{}).
		Where("variable_id = ?", variableID).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion).Error
	if err != nil {
		return 0, err
	}
	return maxVersion + 1, nil
}

// Create persists a new variable version in the database.
func (r *variableVersionRepository) Create(ctx context.Context, variableVersion *model.VariableVersion) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if variableVersion.Version == 0 {
			var maxVersion int
			err := tx.Model(&model.VariableVersion{}).
				Where("variable_id = ?", variableVersion.VariableID).
				Select("COALESCE(MAX(version), 0)").
				Scan(&maxVersion).Error
			if err != nil {
				return err
			}
			variableVersion.Version = maxVersion + 1
		}

		if err := gorm.G[model.VariableVersion](tx).Create(ctx, variableVersion); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrDuplicateVersion
			}
			return err
		}
		return nil
	})
}

// SetActive updates the active status of a variable version in the database.
func (r *variableVersionRepository) SetActive(ctx context.Context, variableVersionID uuid.UUID, active bool) error {
	_, err := gorm.G[model.VariableVersion](r.DB).Where("id = ?", variableVersionID).Update(ctx, "active", active)
	return err
}

// DeactivateAll deactivates all active versions of a variable.
func (r *variableVersionRepository) DeactivateAll(ctx context.Context, variableID uuid.UUID) error {
	_, err := gorm.G[model.VariableVersion](r.DB).Where("variable_id = ? AND active = ?", variableID, true).Update(ctx, "active", false)
	return err
}

// GetAll retrieves all variable versions from the database.
func (r *variableVersionRepository) GetAll(ctx context.Context, variableID uuid.UUID) ([]model.VariableVersion, error) {
	variableVersions, err := gorm.G[model.VariableVersion](r.DB).Where("variable_id = ?", variableID).Order("version DESC").Find(ctx)
	if err != nil {
		return nil, err
	}
	return variableVersions, nil
}

// GetCurrentActive retrieves the current active variable version for a given variable ID.
func (r *variableVersionRepository) GetCurrentActive(ctx context.Context, variableID uuid.UUID) (*model.VariableVersion, error) {
	variableVersion := new(model.VariableVersion)
	if err := r.DB.WithContext(ctx).Where("variable_id = ? AND active = ?", variableID, true).Order("version DESC").First(variableVersion).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVariableVersionNotFound
		}
		return nil, err
	}
	return variableVersion, nil
}
