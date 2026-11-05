package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/repository"
)

// AuditActor identifies who performed an audited action.
type AuditActor struct {
	Type types.ActorType // Type is the kind of actor (user, token or system).
	ID   uuid.UUID       // ID is the actor's UUID (uuid.Nil for "system").
	Name string          // Name is the actor's display name (empty for "system").
}

// ActorResolver resolves the authenticated actor from the request context. It is injected into the AuditService so the services layer never depends on HTTP context primitives.
type ActorResolver func(ctx context.Context) AuditActor

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
	Record(ctx context.Context, action types.Action, resourceType types.ResourceType, resourceID uuid.UUID, resourceName string) error                     // Record appends an event attributed to the actor resolved from ctx.
	RecordAs(ctx context.Context, actor AuditActor, action types.Action, resourceType types.ResourceType, resourceID uuid.UUID, resourceName string) error // RecordAs appends an event with an explicit actor.
	List(ctx context.Context, filter AuditFilter) (*AuditList, error)                                                                                      // List returns a page of audit events.
	Export(ctx context.Context, filter AuditFilter) ([]model.AuditEvent, error)                                                                            // Export returns every audit event matching the filter, oldest first.
}

// auditService implements AuditService.
type auditService struct {
	auditRepository repository.AuditRepository
	resolveActor    ActorResolver
}

// NewAuditService creates a new AuditService with the provided AuditRepository and actor resolver. The resolver reads the authenticated actor from the request context; wire it to the context-aware helper at startup.
func NewAuditService(auditRepository repository.AuditRepository, resolveActor ActorResolver) AuditService {
	return &auditService{auditRepository: auditRepository, resolveActor: resolveActor}
}

// Record appends an audit event attributed to the actor resolved from the request context via the injected resolver.
func (s *auditService) Record(ctx context.Context, action types.Action, resourceType types.ResourceType, resourceID uuid.UUID, resourceName string) error {
	return s.RecordAs(ctx, s.resolveActor(ctx), action, resourceType, resourceID, resourceName)
}

// RecordAs appends an audit event with an explicit actor.
func (s *auditService) RecordAs(ctx context.Context, actor AuditActor, action types.Action, resourceType types.ResourceType, resourceID uuid.UUID, resourceName string) error {
	return s.auditRepository.Create(ctx, &model.AuditEvent{
		ActorType:    string(actor.Type),
		ActorID:      actor.ID,
		ActorName:    actor.Name,
		Action:       string(action),
		ResourceType: string(resourceType),
		ResourceID:   resourceID,
		ResourceName: resourceName,
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
