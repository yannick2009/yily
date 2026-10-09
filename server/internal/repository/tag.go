package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
	"gorm.io/gorm"
)

var (
	ErrTagNotFound          = errors.New("tag not found")                                 // ErrTagNotFound is returned when a tag is not found in the database.
	ErrTagNameAlreadyExists = errors.New("tag name already exists in this project")       // ErrTagNameAlreadyExists is returned when a tag name already exists in the project.
	ErrTagNotInProject      = errors.New("tag does not belong to the variable's project") // ErrTagNotInProject is returned when a tag is not part of the variable's project.
)

// TagRepository defines the interface for interacting with tag data in the database.
type TagRepository interface {
	Create(ctx context.Context, tag *model.Tag) error                     // Create creates a new tag in the database.
	Delete(ctx context.Context, projectID, tagID uuid.UUID) error         // Delete deletes a tag from the database by its project and ID.
	GetAll(ctx context.Context, projectID uuid.UUID) ([]model.Tag, error) // GetAll retrieves all tags of a project from the database.
}

// tagRepository is a struct that implements the TagRepository interface.
type tagRepository struct {
	DB *gorm.DB
}

// NewTagRepository creates a new instance of tagRepository with the provided GORM database connection.
func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{DB: db}
}

// Create creates a new tag in the database.
func (r *tagRepository) Create(ctx context.Context, tag *model.Tag) error {
	if err := gorm.G[model.Tag](r.DB).Create(ctx, tag); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrTagNameAlreadyExists
		}
		return err
	}
	return nil
}

// Delete deletes a tag from the database by its project and ID.
func (r *tagRepository) Delete(ctx context.Context, projectID, tagID uuid.UUID) error {
	rows, err := gorm.G[model.Tag](r.DB).Where("id = ? AND project_id = ?", tagID, projectID).Delete(ctx)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTagNotFound
	}
	return nil
}

// GetAll retrieves all tags of a project from the database.
func (r *tagRepository) GetAll(ctx context.Context, projectID uuid.UUID) ([]model.Tag, error) {
	tags, err := gorm.G[model.Tag](r.DB).Where("project_id = ?", projectID).Find(ctx)
	if err != nil {
		return nil, err
	}
	return tags, nil
}
