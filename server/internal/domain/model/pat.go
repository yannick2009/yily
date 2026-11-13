package model

import (
	"time"

	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/id"
	"gorm.io/gorm"
)

// PersonalAccessToken represents a programmatic API token owned by the Super Admin.
type PersonalAccessToken struct {
	ID         id.ID          `gorm:"primaryKey;type:text" json:"id"`                 // Unique identifier for the PAT
	UserID     id.ID          `gorm:"not null;index;type:text" json:"user_id"`        // Foreign key referencing the associated user
	Name       string         `gorm:"type:varchar(255);not null" json:"name"`         // Human-readable name for the PAT
	Scope      types.PATScope `gorm:"type:varchar(16);not null;index" json:"scope"`   // Permission level granted to this PAT (read or write)
	TokenHash  string         `gorm:"type:varchar(64);not null;uniqueIndex" json:"-"` // SHA-256 hex digest of the full yily_<token> (never serialized)
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`               // Timestamp when the PAT was created
	ExpiresAt  *time.Time     `gorm:"index" json:"expires_at,omitempty"`              // Timestamp when the PAT expires (nil means never expires; only allowed for read scope)
	RevokedAt  *time.Time     `gorm:"index" json:"revoked_at,omitempty"`              // Timestamp when the PAT was revoked (nil if active)
	LastUsedAt *time.Time     `gorm:"index" json:"last_used_at,omitempty"`            // Timestamp of the most recent successful authentication with this PAT
}

// BeforeCreate is a GORM hook that assigns an identifier before creating a new PersonalAccessToken record.
func (p *PersonalAccessToken) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = id.New(id.PrefixPAT)
	}
	return nil
}

// IsRevoked reports whether the PAT has been explicitly revoked.
func (p *PersonalAccessToken) IsRevoked() bool {
	return p.RevokedAt != nil
}

// IsExpired reports whether the PAT has reached or passed its expiration timestamp. A nil ExpiresAt means the PAT never expires.
func (p *PersonalAccessToken) IsExpired(now time.Time) bool {
	if p.ExpiresAt == nil {
		return false
	}
	return !now.Before(*p.ExpiresAt)
}
