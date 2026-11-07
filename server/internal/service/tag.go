package service

import (
	"context"
	"errors"
	"strings"
	"uuid"

	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/repository"
)

// TagBatchResult reports the outcome of a batch tag creation. Both lists are returned so the client can tell apart freshly-created tags from names that already existed.
type TagBatchResult struct {
	Created []model.Tag // Created lists the tags that were persisted during this call.
	Skipped []string    // Skipped lists the input names that were ignored because a tag with that name already existed in the project.
}

// TagService defines the interface for tag business logic.
type TagService interface {
	Create(ctx context.Context, projectID uuid.UUID, name string) (*model.Tag, error)              // Create creates a single tag for a project. Returns ErrTagNameAlreadyExists if the name is taken.
	CreateBatch(ctx context.Context, projectID uuid.UUID, names []string) (*TagBatchResult, error) // CreateBatch creates several tags in one call, skipping names that already exist in the project.
	Delete(ctx context.Context, projectID, tagID uuid.UUID) error                                  // Delete deletes a tag of a project.
	GetAll(ctx context.Context, projectID uuid.UUID) ([]model.Tag, error)                          // GetAll retrieves all tags of a project.
}

// tagService implements TagService.
type tagService struct {
	tagRepository repository.TagRepository
	audit         AuditService
}

// NewTagService creates a new tagService with the provided TagRepository and audit service.
func NewTagService(tagRepository repository.TagRepository, audit AuditService) TagService {
	return &tagService{tagRepository, audit}
}

// Create creates a new tag for a project.
func (s *tagService) Create(ctx context.Context, projectID uuid.UUID, name string) (*model.Tag, error) {
	tag := &model.Tag{Name: name, ProjectID: projectID}
	if err := s.tagRepository.Create(ctx, tag); err != nil {
		return nil, err
	}

	_ = s.audit.Record(ctx, types.ActionCreated, types.ResourceTag, tag.ID, tag.Name, nil)

	return tag, nil
}

// CreateBatch creates several tags for a project in one call. The input is trimmed and deduplicated case-sensitively; names that already exist in the project are reported in Skipped instead of returning an error. Any other persistence error aborts the batch and returns what was created so far along with the error.
func (s *tagService) CreateBatch(ctx context.Context, projectID uuid.UUID, names []string) (*TagBatchResult, error) {
	// Clean up the input: trim whitespace, drop empties, deduplicate while
	// preserving first-seen order so audit entries follow user intent.
	seen := make(map[string]struct{}, len(names))
	cleaned := make([]string, 0, len(names))
	for _, raw := range names {
		n := strings.TrimSpace(raw)
		if n == "" {
			continue
		}
		if _, dup := seen[n]; dup {
			continue
		}
		seen[n] = struct{}{}
		cleaned = append(cleaned, n)
	}

	result := &TagBatchResult{
		Created: make([]model.Tag, 0, len(cleaned)),
		Skipped: make([]string, 0),
	}
	for _, name := range cleaned {
		tag := &model.Tag{Name: name, ProjectID: projectID}
		if err := s.tagRepository.Create(ctx, tag); err != nil {
			if errors.Is(err, repository.ErrTagNameAlreadyExists) {
				result.Skipped = append(result.Skipped, name)
				continue
			}
			return result, err
		}
		_ = s.audit.Record(ctx, types.ActionCreated, types.ResourceTag, tag.ID, tag.Name, nil)
		result.Created = append(result.Created, *tag)
	}
	return result, nil
}

// Delete deletes a tag of a project.
func (s *tagService) Delete(ctx context.Context, projectID, tagID uuid.UUID) error {
	name := ""
	if tag, err := s.tagRepository.GetByID(ctx, tagID); err == nil {
		name = tag.Name
	}
	if err := s.tagRepository.Delete(ctx, projectID, tagID); err != nil {
		return err
	}
	if name != "" {
		_ = s.audit.Record(ctx, types.ActionDeleted, types.ResourceTag, tagID, name, nil)
	}
	return nil
}

// GetAll retrieves all tags of a project.
func (s *tagService) GetAll(ctx context.Context, projectID uuid.UUID) ([]model.Tag, error) {
	return s.tagRepository.GetAll(ctx, projectID)
}
