package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents the Super Admin account in the system.
type User struct {
	ID                   uuid.UUID             `gorm:"primaryKey" json:"id"`                                   // Unique identifier for the user
	Name                 string                `gorm:"type:varchar(255);not null" json:"name"`                 // Display name of the user
	Email                string                `gorm:"type:varchar(255);not null;uniqueIndex" json:"email"`    // Normalized (lowercase) email address
	PasswordHash         string                `gorm:"type:text;not null" json:"-"`                            // Argon2id password hash (never serialized in JSON)
	CreatedAt            time.Time             `gorm:"autoCreateTime" json:"created_at"`                       // Timestamp when the user was created
	UpdatedAt            time.Time             `gorm:"autoUpdateTime" json:"updated_at"`                       // Timestamp when the user was last updated
	Sessions             []Session             `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"` // List of sessions associated with the user
	PersonalAccessTokens []PersonalAccessToken `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"` // List of PATs associated with the user
}

// BeforeCreate is a GORM hook that is triggered before creating a new User record.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
