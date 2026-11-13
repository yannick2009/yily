package model

import (
	"time"

	"github.com/yannick2009/yily/internal/id"
	"gorm.io/gorm"
)

// Environment represents an environment in the system.
type Environment struct {
	ID          id.ID      `gorm:"primaryKey;type:text" json:"id"`                                 // Unique identifier for the environment
	Name        string     `gorm:"type:varchar(255);not null" json:"name"`                         // Name of the environment
	Description string     `gorm:"type:varchar(255)" json:"description"`                           // Description of the environment
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`                               // Timestamp when the environment was created
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`                               // Timestamp when the environment was last updated
	ProjectID   id.ID      `gorm:"not null;index;type:text" json:"project_id"`                     // Foreign key referencing the associated project
	Variables   []Variable `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"variables"` // List of variables associated with the environment
}

// BeforeCreate is a GORM hook that is triggered before creating a new Environment record.
func (e *Environment) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = id.New(id.PrefixEnvironment)
	}
	return nil
}
