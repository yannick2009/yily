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
	ErrSessionNotFound = errors.New("session not found") // ErrSessionNotFound is returned when no session matches the given SHA-256 token hash.
)

// SessionRepository defines the interface for server-side web session persistence.
type SessionRepository interface {
	Create(ctx context.Context, session *model.Session) error                           // Create persists a new session storing only its SHA-256 token hash.
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.Session, error)       // GetByTokenHash retrieves a session by its SHA-256 hash.
	RevokeByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error // RevokeByTokenHash marks a session as revoked by its SHA-256 hash.
	TouchActivity(ctx context.Context, sessionID uuid.UUID, activityAt time.Time) error // TouchActivity updates the session's last activity timestamp.
}

// sessionRepository implements SessionRepository.
type sessionRepository struct {
	DB *gorm.DB
}

// NewSessionRepository creates a new SessionRepository.
func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{DB: db}
}

// Create persists a new session in the database.
func (r *sessionRepository) Create(ctx context.Context, session *model.Session) error {
	return gorm.G[model.Session](r.DB).Create(ctx, session)
}

// GetByTokenHash retrieves a session by its SHA-256 token hash.
func (r *sessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.Session, error) {
	session, err := gorm.G[model.Session](r.DB).Where("token_hash = ?", tokenHash).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	return &session, nil
}

// RevokeByTokenHash marks an active session as revoked by its SHA-256 token hash.
func (r *sessionRepository) RevokeByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	res := r.DB.WithContext(ctx).
		Model(&model.Session{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Update("revoked_at", revokedAt)
	if res.Error != nil {
		return res.Error
	}
	// If no active row was updated, check whether the session does not exist or was already revoked (idempotent).
	if res.RowsAffected == 0 {
		var count int64
		if err := r.DB.WithContext(ctx).Model(&model.Session{}).Where("token_hash = ?", tokenHash).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return ErrSessionNotFound
		}
	}
	return nil
}

// TouchActivity updates the last_activity_at timestamp of a session to track inactivity expiration.
func (r *sessionRepository) TouchActivity(ctx context.Context, sessionID uuid.UUID, activityAt time.Time) error {
	_, err := gorm.G[model.Session](r.DB).Where("id = ?", sessionID).Update(ctx, "last_activity_at", activityAt)
	return err
}
