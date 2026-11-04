package repository

import (
	"context"
	"time"

	"github.com/yannick2009/yily/internal/domain/model"
	"gorm.io/gorm"
)

// AuditListFilter filters the audit log listing.
type AuditListFilter struct {
	Search    string     // Search matches resource and actor names case-insensitively (empty for no filter).
	ActorType string     // ActorType filters by actor type (empty for no filter).
	Action    string     // Action filters by action (empty for no filter).
	From      *time.Time // From is an inclusive lower bound on created_at (nil for no lower bound).
	To        *time.Time // To is an exclusive upper bound on created_at (nil for no upper bound).
	Limit     int        // Limit caps the number of returned events.
	Offset    int        // Offset skips this many events before returning results.
}

// AuditRepository defines the interface for persisting and querying audit events.
type AuditRepository interface {
	Create(ctx context.Context, event *model.AuditEvent) error                           // Create appends a new audit event.
	List(ctx context.Context, filter AuditListFilter) ([]model.AuditEvent, int64, error) // List returns a page of events, newest first, and the total count matching the filter.
	ListAll(ctx context.Context, filter AuditListFilter) ([]model.AuditEvent, error)     // ListAll returns every event matching the filter, oldest first (no pagination).
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
	query := r.applyFilters(ctx, filter)

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

// ListAll returns every audit event matching the filter, oldest first (no pagination).
// It is used by the CSV export, where a chronological order reads more naturally.
func (r *auditRepository) ListAll(ctx context.Context, filter AuditListFilter) ([]model.AuditEvent, error) {
	var events []model.AuditEvent
	if err := r.applyFilters(ctx, filter).Order("created_at ASC").Order("id ASC").Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// applyFilters returns a query scoped to the given filters, shared by List and ListAll.
func (r *auditRepository) applyFilters(ctx context.Context, filter AuditListFilter) *gorm.DB {
	query := r.DB.WithContext(ctx).Model(&model.AuditEvent{})

	if filter.Search != "" {
		query = query.Where("(resource_name LIKE ? OR actor_name LIKE ?)", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}
	if filter.ActorType != "" {
		query = query.Where("actor_type = ?", filter.ActorType)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.From != nil {
		query = query.Where("created_at >= ?", filter.From.UTC())
	}
	if filter.To != nil {
		query = query.Where("created_at < ?", filter.To.UTC())
	}

	return query
}
