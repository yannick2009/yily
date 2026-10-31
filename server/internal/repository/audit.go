package repository

import (
	"context"

	"github.com/yannick2009/yily/internal/domain/model"
	"gorm.io/gorm"
)

// AuditListFilter filters the audit log listing.
type AuditListFilter struct {
	Search    string // Search matches resource names case-insensitively (empty for no filter).
	ActorType string // ActorType filters by actor type (empty for no filter).
	Action    string // Action filters by action (empty for no filter).
	Limit     int    // Limit caps the number of returned events.
	Offset    int    // Offset skips this many events before returning results.
}

// AuditRepository defines the interface for persisting and querying audit events.
type AuditRepository interface {
	Create(ctx context.Context, event *model.AuditEvent) error                           // Create appends a new audit event.
	List(ctx context.Context, filter AuditListFilter) ([]model.AuditEvent, int64, error) // List returns a page of events, newest first, and the total count matching the filter.
}

// auditRepository implements AuditRepository.
type auditRepository struct {
	DB *gorm.DB
}

// NewAuditRepository creates a new auditRepository.
func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{DB: db}
}

// Create appends a new audit event.
func (r *auditRepository) Create(ctx context.Context, event *model.AuditEvent) error {
	return gorm.G[model.AuditEvent](r.DB).Create(ctx, event)
}

// List returns a page of audit events, newest first, and the total count matching the filter.
func (r *auditRepository) List(ctx context.Context, filter AuditListFilter) ([]model.AuditEvent, int64, error) {
	query := r.DB.WithContext(ctx).Model(&model.AuditEvent{})

	if filter.Search != "" {
		query = query.Where("resource_name LIKE ?", "%"+filter.Search+"%")
	}
	if filter.ActorType != "" {
		query = query.Where("actor_type = ?", filter.ActorType)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var events []model.AuditEvent
	if err := query.Order("created_at DESC").Order("id DESC").Limit(filter.Limit).Offset(filter.Offset).Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}
