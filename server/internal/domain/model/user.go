package model

import (
	"time"

	"github.com/yannick2009/yily/internal/id"
	"gorm.io/gorm"
)

// User represents the Super Admin account in the system.
type User struct {
	ID                   id.ID                 `gorm:"primaryKey;type:text" json:"id"`                         // Unique identifier for the user
	Name                 string                `gorm:"type:varchar(255);not null" json:"name"`                 // Display name of the user
	Email                string                `gorm:"type:varchar(255);not null;uniqueIndex" json:"email"`    // Normalized (lowercase) email address
	PasswordHash         string                `gorm:"type:text;not null" json:"-"`                            // Argon2id password hash (never serialized in JSON)
	RecoveryCodeHash     string                `gorm:"type:text" json:"-"`                                     // Argon2id hash of the recovery code (never serialized); empty when recovery is unavailable
	TOTPSecretEncrypted  string                `gorm:"type:text" json:"-"`                                     // AES-256-GCM encrypted TOTP seed bound to the user ID (never serialized)
	TOTPEnabled          bool                  `gorm:"not null;default:false" json:"totp_enabled"`             // TOTPEnabled reports whether login requires a code from the authenticator app
	TOTPLastStep         int64                 `gorm:"not null;default:0" json:"-"`                            // TOTPLastStep is the last accepted time step, used to reject replayed codes
	TOTPFailedAttempts   int                   `gorm:"not null;default:0" json:"-"`                            // TOTPFailedAttempts counts consecutive invalid codes for lockout
	TOTPLockedUntil      *time.Time            `gorm:"type:datetime" json:"-"`                                 // TOTPLockedUntil refuses TOTP verification before this instant, when set
	CreatedAt            time.Time             `gorm:"autoCreateTime" json:"created_at"`                       // Timestamp when the user was created
	UpdatedAt            time.Time             `gorm:"autoUpdateTime" json:"updated_at"`                       // Timestamp when the user was last updated
	Sessions             []Session             `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"` // List of sessions associated with the user
	PersonalAccessTokens []PersonalAccessToken `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"` // List of PATs associated with the user
}

// BeforeCreate is a GORM hook that is triggered before creating a new User record.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = id.New(id.PrefixUser)
	}
	return nil
}
