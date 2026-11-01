package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditEvent represents a point-in-time record of an action: who did what to which resource.
// It is append-only and never updated or deleted, keeping a durable history even after the
// referenced resource is gone.
type AuditEvent struct {
	ID           uuid.UUID `gorm:"primaryKey" json:"id"`                                 // Unique identifier for the audit event
	CreatedAt    time.Time `gorm:"autoCreateTime;index" json:"created_at"`               // Timestamp when the event occurred
	ActorType    string    `gorm:"type:varchar(16);not null;index" json:"actor_type"`    // Who acted: "user", "token" or "system"
	ActorID      uuid.UUID `gorm:"index" json:"actor_id"`                                // ID of the actor (uuid.Nil for "system")
	ActorName    string    `gorm:"type:varchar(255);not null" json:"actor_name"`         // Human-readable name of the actor at the time of the event (empty for "system")
	Action       string    `gorm:"type:varchar(32);not null;index" json:"action"`        // What happened, e.g. "created", "updated", "deleted", "revealed"
	ResourceType string    `gorm:"type:varchar(32);not null;index" json:"resource_type"` // Kind of resource, e.g. "project", "environment", "variable" (empty for auth events)
	ResourceID   uuid.UUID `gorm:"index" json:"resource_id"`                             // ID of the resource (uuid.Nil when not applicable)
	ResourceName string    `gorm:"type:varchar(255);not null" json:"resource_name"`      // Human-readable name of the resource (empty for auth events)
	Metadata     string    `gorm:"type:text" json:"metadata"`                            // Optional JSON string with extra context (environment, IP, …)
}

// BeforeCreate is a GORM hook that is triggered before creating a new AuditEvent record.
func (e *AuditEvent) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}
