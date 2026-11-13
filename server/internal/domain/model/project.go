package model

import (
	"time"

	"github.com/yannick2009/yily/internal/id"
	"gorm.io/gorm"
)

// Project represents a project in the system.
type Project struct {
	ID           id.ID         `gorm:"primaryKey;type:text" json:"id"`                                    // Unique identifier for the project
	Name         string        `gorm:"type:varchar(255);not null" json:"name"`                            // Name of the project
	CreatedAt    time.Time     `gorm:"autoCreateTime" json:"created_at"`                                  // Timestamp when the project was created
	UpdatedAt    time.Time     `gorm:"autoUpdateTime" json:"updated_at"`                                  // Timestamp when the project was last updated
	Environments []Environment `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"environments"` // List of environments associated with the project
	Tags         []Tag         `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"tags"`         // List of tags associated with the project
}

// BeforeCreate is a GORM hook that is triggered before creating a new Project record.
func (p *Project) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = id.New(id.PrefixProject)
	}
	return nil
}
