package model

import (
	"time"

	"github.com/yannick2009/yily/internal/id"
	"gorm.io/gorm"
)

// Session represents a server-side authenticated web session.
type Session struct {
	ID             id.ID      `gorm:"primaryKey;type:text" json:"id"`                 // Unique identifier for the session
	UserID         id.ID      `gorm:"not null;index;type:text" json:"user_id"`        // Foreign key referencing the associated user
	TokenHash      string     `gorm:"type:varchar(64);not null;uniqueIndex" json:"-"` // SHA-256 hex digest of the session token (never serialized)
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`               // Timestamp when the session was created
	LastActivityAt time.Time  `gorm:"not null" json:"last_activity_at"`               // Timestamp of the last authenticated activity
	ExpiresAt      time.Time  `gorm:"not null;index" json:"expires_at"`               // Timestamp when the session expires
	RevokedAt      *time.Time `gorm:"index" json:"revoked_at,omitempty"`              // Timestamp when the session was revoked (nil if active)
}

// BeforeCreate is a GORM hook that is triggered before creating a new Session record.
func (s *Session) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = id.New(id.PrefixSession)
	}
	return nil
}

// IsRevoked reports whether the session has been explicitly revoked.
func (s *Session) IsRevoked() bool {
	return s.RevokedAt != nil
}

// IsExpired reports whether the session has exceeded its absolute expiration or inactivity timeout.
func (s *Session) IsExpired(now time.Time, idleTimeout time.Duration) bool {
	if !now.Before(s.ExpiresAt) {
		return true
	}
	if idleTimeout > 0 && !s.LastActivityAt.IsZero() && now.Sub(s.LastActivityAt) >= idleTimeout {
		return true
	}
	return false
}
