package model

import (
	"time"

	"github.com/yannick2009/yily/internal/id"
	"gorm.io/gorm"
)

// Variable represents a variable in the system.
type Variable struct {
	ID            id.ID             `gorm:"primaryKey;type:text" json:"id"`                                                           // Unique identifier for the variable
	Name          string            `gorm:"type:varchar(255);not null;uniqueIndex:idx_variable_environment_name" json:"name"`         // Name of the variable
	CreatedAt     time.Time         `gorm:"autoCreateTime" json:"created_at"`                                                         // Timestamp when the variable was created
	UpdatedAt     time.Time         `gorm:"autoUpdateTime" json:"updated_at"`                                                         // Timestamp when the variable was last updated
	EnvironmentID id.ID             `gorm:"not null;index;uniqueIndex:idx_variable_environment_name;type:text" json:"environment_id"` // Foreign key referencing the associated environment
	Versions      []VariableVersion `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"versions"`                            // List of variable versions associated with the variable
	Tags          []Tag             `gorm:"many2many:variable_tags;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"tags"`        // List of tags associated with the variable
	Active        bool              `gorm:"-" json:"active"`                                                                          // Whether the variable has a current (active) value; computed on read, never persisted
}

// BeforeCreate is a GORM hook that is triggered before creating a new Variable record.
func (v *Variable) BeforeCreate(tx *gorm.DB) error {
	if v.ID == "" {
		v.ID = id.New(id.PrefixVariable)
	}
	return nil
}
