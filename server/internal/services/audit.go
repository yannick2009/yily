package services

import (
	"context"

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

// ActorFromContext resolves the authenticated actor from the request context.
// A Bearer PAT takes precedence over a session user, so token activity is attributed to the token.
func ActorFromContext(ctx context.Context) AuditActor {
	if pat, ok := AuthenticatedPATFromContext(ctx); ok {
		return AuditActor{Type: types.ActorTypeToken, ID: pat.ID, Name: pat.Name}
	}
	if user, ok := AuthenticatedUserFromContext(ctx); ok {
		return AuditActor{Type: types.ActorTypeUser, ID: user.ID, Name: user.Name}
	}
	return AuditActor{Type: types.ActorTypeSystem}
}

// AuditFilter filters the audit log listing.
type AuditFilter struct {
	Search    string // Search matches resource and actor names (empty for no filter).
	ActorType string // ActorType filters by actor type (empty for no filter).
	Action    string // Action filters by action (empty for no filter).
	Limit     int    // Limit caps the number of returned events.
	Offset    int    // Offset skips this many events before returning results.
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
}

// auditService implements AuditService.
type auditService struct {
	auditRepository repository.AuditRepository
}

// NewAuditService creates a new auditService with the provided AuditRepository.
func NewAuditService(auditRepository repository.AuditRepository) AuditService {
	return &auditService{auditRepository: auditRepository}
}

// Record appends an audit event attributed to the actor resolved from the request context.
func (s *auditService) Record(ctx context.Context, action types.Action, resourceType types.ResourceType, resourceID uuid.UUID, resourceName string) error {
	return s.RecordAs(ctx, ActorFromContext(ctx), action, resourceType, resourceID, resourceName)
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
		Limit:     filter.Limit,
		Offset:    filter.Offset,
	})
	if err != nil {
		return nil, err
	}
	return &AuditList{Events: events, Total: total}, nil
}
