package service

import (
	"context"
	"encoding/json"
	"time"
	"uuid"

	"github.com/yannick2009/yily/internal/authctx"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/repository"
)

// AuditFilter filters the audit log listing.
type AuditFilter struct {
	Search    string     // Search matches resource and actor names (empty for no filter).
	ActorType string     // ActorType filters by actor type (empty for no filter).
	Action    string     // Action filters by action (empty for no filter).
	From      *time.Time // From is an inclusive lower bound on created_at (nil for no lower bound).
	To        *time.Time // To is an exclusive upper bound on created_at (nil for no upper bound).
	Limit     int        // Limit caps the number of returned events.
	Offset    int        // Offset skips this many events before returning results.
}

// AuditList is a page of audit events with a total count.
type AuditList struct {
	Events []model.AuditEvent `json:"events"` // Events is the page of events, newest first.
	Total  int64              `json:"total"`  // Total is the total number of events matching the filter.
}

// AuditService defines the interface for recording and listing audit events.
type AuditService interface {
	Record(ctx context.Context, action types.Action, resourceType types.ResourceType, resourceID uuid.UUID, resourceName string, metadata map[string]string) error // Record appends an event attributed to the actor resolved from ctx; metadata is nil when there is nothing extra to record.
	List(ctx context.Context, filter AuditFilter) (*AuditList, error)                                                                                              // List returns a page of audit events.
	Export(ctx context.Context, filter AuditFilter) ([]model.AuditEvent, error)                                                                                    // Export returns every audit event matching the filter, oldest first.
}

// auditService implements AuditService.
type auditService struct {
	auditRepository repository.AuditRepository
}

// NewAuditService creates a new AuditService with the provided AuditRepository.
func NewAuditService(auditRepository repository.AuditRepository) AuditService {
	return &auditService{auditRepository: auditRepository}
}

// Record appends an audit event attributed to the actor resolved from the request context via the injected resolver. metadata is optional (nil for none) and is serialized as JSON into the event.
func (s *auditService) Record(ctx context.Context, action types.Action, resourceType types.ResourceType, resourceID uuid.UUID, resourceName string, metadata map[string]string) error {
	var metadataJSON string
	if len(metadata) > 0 {
		if raw, err := json.Marshal(metadata); err == nil {
			metadataJSON = string(raw)
		}
	}

	actor := authctx.ActorFromContext(ctx)

	return s.auditRepository.Create(ctx, &model.AuditEvent{
		ActorType:    string(actor.Type),
		ActorID:      actor.ID,
		ActorName:    actor.Name,
		Action:       string(action),
		ResourceType: string(resourceType),
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Metadata:     metadataJSON,
	})
}

// List returns a page of audit events, newest first.
func (s *auditService) List(ctx context.Context, filter AuditFilter) (*AuditList, error) {
	events, total, err := s.auditRepository.List(ctx, repository.AuditListFilter{
		Search:    filter.Search,
		ActorType: filter.ActorType,
		Action:    filter.Action,
		From:      filter.From,
		To:        filter.To,
		Limit:     filter.Limit,
		Offset:    filter.Offset,
	})
	if err != nil {
		return nil, err
	}
	return &AuditList{Events: events, Total: total}, nil
}

// Export returns every audit event matching the filter, oldest first.
func (s *auditService) Export(ctx context.Context, filter AuditFilter) ([]model.AuditEvent, error) {
	return s.auditRepository.ListAll(ctx, repository.AuditListFilter{
		Search:    filter.Search,
		ActorType: filter.ActorType,
		Action:    filter.Action,
		From:      filter.From,
		To:        filter.To,
	})
}
