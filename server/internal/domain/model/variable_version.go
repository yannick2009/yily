package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VariableVersion represents a version of a variable in the system.
type VariableVersion struct {
	ID         uuid.UUID `gorm:"primaryKey" json:"id"`                // Unique identifier for the variable version
	Version    int       `gorm:"not null" json:"version"`             // Version of the variable
	Value      string    `gorm:"type:text;not null" json:"value"`     // Value of the variable version
	Active     bool      `gorm:"not null;default:true" json:"active"` // Indicates if this version is the active version
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`    // Timestamp when the variable version was last updated
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`    // Timestamp when the variable version was created
	VariableID uuid.UUID `gorm:"not null;index" json:"variable_id"`   // Foreign key referencing the associated variable
}

// BeforeCreate is a GORM hook that is triggered before creating a new VariableVersion record.
func (vv *VariableVersion) BeforeCreate(tx *gorm.DB) error {
	if vv.ID == uuid.Nil {
		vv.ID = uuid.New()
	}
	return nil
}
