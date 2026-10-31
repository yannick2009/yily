package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PersonalAccessToken represents a programmatic API token owned by the Super Admin.
type PersonalAccessToken struct {
	ID        uuid.UUID  `gorm:"primaryKey" json:"id"`                           // Unique identifier for the PAT
	UserID    uuid.UUID  `gorm:"not null;index" json:"user_id"`                  // Foreign key referencing the associated user
	Name      string     `gorm:"type:varchar(255);not null" json:"name"`         // Human-readable name for the PAT
	TokenHash string     `gorm:"type:varchar(64);not null;uniqueIndex" json:"-"` // SHA-256 hex digest of the full yily_<token> (never serialized)
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`               // Timestamp when the PAT was created
	ExpiresAt time.Time  `gorm:"not null;index" json:"expires_at"`               // Timestamp when the PAT expires
	RevokedAt *time.Time `gorm:"index" json:"revoked_at,omitempty"`              // Timestamp when the PAT was revoked (nil if active)
}

// BeforeCreate is a GORM hook that assigns a UUID before creating a new PersonalAccessToken record.
func (p *PersonalAccessToken) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// IsRevoked reports whether the PAT has been explicitly revoked.
func (p *PersonalAccessToken) IsRevoked() bool {
	return p.RevokedAt != nil
}

// IsExpired reports whether the PAT has reached or passed its expiration timestamp.
func (p *PersonalAccessToken) IsExpired(now time.Time) bool {
	return !p.ExpiresAt.IsZero() && !now.Before(p.ExpiresAt)
}
