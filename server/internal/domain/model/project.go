package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Project represents a project in the system.
type Project struct {
	ID           uuid.UUID     `gorm:"primaryKey" json:"id"`                                              // Unique identifier for the project
	Name         string        `gorm:"type:varchar(255);not null" json:"name"`                            // Name of the project
	CreatedAt    time.Time     `gorm:"autoCreateTime" json:"created_at"`                                  // Timestamp when the project was created
	UpdatedAt    time.Time     `gorm:"autoUpdateTime" json:"updated_at"`                                  // Timestamp when the project was last updated
	Environments []Environment `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"environments"` // List of environments associated with the project
	Tags         []Tag         `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"tags"`         // List of tags associated with the project
}

// BeforeCreate is a GORM hook that is triggered before creating a new Project record.
func (p *Project) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
