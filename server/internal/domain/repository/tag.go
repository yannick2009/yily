package repository

import (
	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
)

type TagRepository interface {
	Create(tag *model.Tag) error              // Create inserts a new tag
	GetByID(id uuid.UUID) (*model.Tag, error) // GetByID retrieves a tag by its UUID
	List() ([]*model.Tag, error)              // List retrieves all tags
	Update(tag *model.Tag) error              // Update modifies an existing tag
	Delete(id uuid.UUID) error                // Delete deletes a tag by its UUID
}
