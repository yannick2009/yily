package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
	"gorm.io/gorm"
)

var (
	ErrPATNotFound = errors.New("personal access token not found") // ErrPATNotFound is returned when no PAT matches the given ID or SHA-256 token hash.
)

// PATRepository defines the interface for Personal Access Token persistence.
type PATRepository interface {
	Create(ctx context.Context, pat *model.PersonalAccessToken) error                          // Create persists a new PAT storing only its SHA-256 hash.
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.PersonalAccessToken, error)  // GetByTokenHash retrieves a PAT by its SHA-256 hash during Bearer authentication.
	GetAllByUserID(ctx context.Context, userID uuid.UUID) ([]model.PersonalAccessToken, error) // GetAllByUserID retrieves all PAT metadata belonging to a user.
	RevokeByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, revokedAt time.Time) error // RevokeByID marks a user's PAT as revoked by its ID.
}

// patRepository implements PATRepository.
type patRepository struct {
	DB *gorm.DB
}

// NewPATRepository creates a new PATRepository.
func NewPATRepository(db *gorm.DB) PATRepository {
	return &patRepository{DB: db}
}

// Create persists a new PAT record in the database.
func (r *patRepository) Create(ctx context.Context, pat *model.PersonalAccessToken) error {
	return gorm.G[model.PersonalAccessToken](r.DB).Create(ctx, pat)
}

// GetByTokenHash retrieves a PAT by its SHA-256 token hash to verify incoming Bearer tokens.
func (r *patRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.PersonalAccessToken, error) {
	pat, err := gorm.G[model.PersonalAccessToken](r.DB).Where("token_hash = ?", tokenHash).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPATNotFound
		}
		return nil, err
	}
	return &pat, nil
}

// GetAllByUserID retrieves all PATs belonging to a user ordered by creation date descending.
func (r *patRepository) GetAllByUserID(ctx context.Context, userID uuid.UUID) ([]model.PersonalAccessToken, error) {
	return gorm.G[model.PersonalAccessToken](r.DB).Where("user_id = ?", userID).Order("created_at DESC").Find(ctx)
}

// RevokeByID sets revoked_at on an active PAT owned by the user, or returns ErrPATNotFound if the ID does not belong to them.
func (r *patRepository) RevokeByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, revokedAt time.Time) error {
	res := r.DB.WithContext(ctx).
		Model(&model.PersonalAccessToken{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", id, userID).
		Update("revoked_at", revokedAt)
	if res.Error != nil {
		return res.Error
	}
	// If no active row was updated, check whether the PAT does not exist for this user or was already revoked (idempotent).
	if res.RowsAffected == 0 {
		var count int64
		if err := r.DB.WithContext(ctx).Model(&model.PersonalAccessToken{}).Where("id = ? AND user_id = ?", id, userID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return ErrPATNotFound
		}
	}
	return nil
}
