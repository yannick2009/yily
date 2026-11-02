package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tag represents a tag in the system.
type Tag struct {
	ID        uuid.UUID  `gorm:"primaryKey" json:"id"`                                                                   // Unique identifier for the tag
	Name      string     `gorm:"type:varchar(255);not null;uniqueIndex:idx_tag_project_name" json:"name"`                // Name of the tag
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`                                                       // Timestamp when the tag was created
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"`                                                       // Timestamp when the tag was last updated
	ProjectID uuid.UUID  `gorm:"not null;index;uniqueIndex:idx_tag_project_name" json:"project_id"`                      // Foreign key referencing the associated project
	Variables []Variable `gorm:"many2many:variable_tags;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"variables"` // List of variables associated with the tag
}

// BeforeCreate is a GORM hook that is triggered before creating a new Tag record.
func (t *Tag) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
