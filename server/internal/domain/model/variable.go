package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Variable represents a variable in the system.
type Variable struct {
	ID            uuid.UUID         `gorm:"primaryKey" json:"id"`                                                              // Unique identifier for the variable
	Name          string            `gorm:"type:varchar(255);not null;uniqueIndex:idx_variable_environment_name" json:"name"`  // Name of the variable
	CreatedAt     time.Time         `gorm:"autoCreateTime" json:"created_at"`                                                  // Timestamp when the variable was created
	UpdatedAt     time.Time         `gorm:"autoUpdateTime" json:"updated_at"`                                                  // Timestamp when the variable was last updated
	EnvironmentID uuid.UUID         `gorm:"not null;index;uniqueIndex:idx_variable_environment_name" json:"environment_id"`    // Foreign key referencing the associated environment
	Versions      []VariableVersion `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"versions"`                     // List of variable versions associated with the variable
	Tags          []Tag             `gorm:"many2many:variable_tags;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"tags"` // List of tags associated with the variable
}

// BeforeCreate is a GORM hook that is triggered before creating a new Variable record.
func (v *Variable) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}
