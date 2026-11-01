package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound      = errors.New("user not found")                  // ErrUserNotFound is returned when no user matches the given ID or email.
	ErrUserAlreadyExists = errors.New("super admin user already exists") // ErrUserAlreadyExists is returned when attempting to create a second Super Admin account.
)

// UserRepository defines the interface for user persistence.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error                 // Create persists the single Super Admin user if none exists yet.
	GetByEmail(ctx context.Context, email string) (*model.User, error)  // GetByEmail retrieves the Super Admin by normalized lowercase email during login.
	GetByID(ctx context.Context, userID uuid.UUID) (*model.User, error) // GetByID loads the User entity owning a validated session or PAT to attach to the request context.
	Count(ctx context.Context) (int64, error)                           // Count returns how many Super Admin accounts exist (0 or 1 by design).
}

// userRepository implements UserRepository.
type userRepository struct {
	DB *gorm.DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{DB: db}
}

// Create persists the Super Admin account inside a transaction, rejecting creation if an account already exists.
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Enforce the single Super Admin invariant before inserting.
		var count int64
		if err := tx.Model(&model.User{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrUserAlreadyExists
		}

		if err := gorm.G[model.User](tx).Create(ctx, user); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrUserAlreadyExists
			}
			return err
		}
		return nil
	})
}

// GetByEmail retrieves a user by normalized lowercase email address.
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user := new(model.User)
	if err := r.DB.WithContext(ctx).Where("email = ?", email).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetByID loads the User entity owning a validated session or PAT so it can be injected into the request context.
func (r *userRepository) GetByID(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	user := new(model.User)
	if err := r.DB.WithContext(ctx).Where("id = ?", userID).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// Count returns how many Super Admin accounts exist (0 or 1 by design).
func (r *userRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.DB.WithContext(ctx).Model(&model.User{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
