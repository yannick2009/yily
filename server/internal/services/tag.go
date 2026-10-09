package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/repository"
)

// TagService defines the interface for tag business logic.
type TagService interface {
	Create(ctx context.Context, projectID uuid.UUID, name string) (*model.Tag, error) // Create creates a new tag for a project.
	Delete(ctx context.Context, projectID, tagID uuid.UUID) error                     // Delete deletes a tag of a project.
	GetAll(ctx context.Context, projectID uuid.UUID) ([]model.Tag, error)             // GetAll retrieves all tags of a project.
}

// tagService implements TagService.
type tagService struct {
	tagRepository repository.TagRepository
}

// NewTagService creates a new tagService with the provided TagRepository.
func NewTagService(tagRepository repository.TagRepository) TagService {
	return &tagService{tagRepository}
}

// Create creates a new tag for a project.
func (s *tagService) Create(ctx context.Context, projectID uuid.UUID, name string) (*model.Tag, error) {
	tag := &model.Tag{Name: name, ProjectID: projectID}
	if err := s.tagRepository.Create(ctx, tag); err != nil {
		return nil, err
	}

	return tag, nil
}

// Delete deletes a tag of a project.
func (s *tagService) Delete(ctx context.Context, projectID, tagID uuid.UUID) error {
	return s.tagRepository.Delete(ctx, projectID, tagID)
}

// GetAll retrieves all tags of a project.
func (s *tagService) GetAll(ctx context.Context, projectID uuid.UUID) ([]model.Tag, error) {
	return s.tagRepository.GetAll(ctx, projectID)
}
