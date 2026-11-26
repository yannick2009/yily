package repository

import (
	"context"
	"errors"
	"time"

	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/id"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound      = errors.New("user not found")                  // ErrUserNotFound is returned when no user matches the given ID or email.
	ErrUserAlreadyExists = errors.New("super admin user already exists") // ErrUserAlreadyExists is returned when attempting to create a second Super Admin account.
)

// UserRepository defines the interface for user persistence.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error                                                                    // Create persists the single Super Admin user if none exists yet.
	GetByEmail(ctx context.Context, email string) (*model.User, error)                                                     // GetByEmail retrieves the Super Admin by normalized lowercase email during login.
	GetByID(ctx context.Context, userID id.ID) (*model.User, error)                                                        // GetByID loads the User entity owning a validated session or PAT to attach to the request context.
	SetName(ctx context.Context, userID id.ID, name string) error                                                          // SetName updates the Super Admin display name.
	SetEmail(ctx context.Context, userID id.ID, email string) error                                                        // SetEmail updates the Super Admin email; callers must pass a normalized lowercase address.
	Count(ctx context.Context) (int64, error)                                                                              // Count returns how many Super Admin accounts exist (0 or 1 by design).
	GetAdmin(ctx context.Context) (*model.User, error)                                                                     // GetAdmin returns the single Super Admin account.
	ResetCredentials(ctx context.Context, userID id.ID, passwordHash, recoveryCodeHash string, revokedAt time.Time) error  // ResetCredentials atomically replaces the password and recovery code hashes, disables TOTP, and revokes every active session and PAT.
	ChangePassword(ctx context.Context, userID id.ID, passwordHash string, keepSessionID id.ID, changedAt time.Time) error // ChangePassword replaces the password hash and revokes every session except keepSessionID in one transaction.
	SetTOTPSecret(ctx context.Context, userID id.ID, encryptedSecret string) error                                         // SetTOTPSecret stores a pending, encrypted TOTP seed and leaves TOTP disabled until it is confirmed.
	EnableTOTP(ctx context.Context, userID id.ID, step int64) error                                                        // EnableTOTP activates TOTP and records the first accepted time step.
	DisableTOTP(ctx context.Context, userID id.ID) error                                                                   // DisableTOTP clears the seed, the replay marker, and every failure counter.
	ConsumeTOTPStep(ctx context.Context, userID id.ID, step int64) (bool, error)                                           // ConsumeTOTPStep atomically accepts a time step once, rejecting replayed codes.
	RecordTOTPFailure(ctx context.Context, userID id.ID, threshold int, lockedUntil time.Time) error                       // RecordTOTPFailure increments the failure counter and locks verification once the threshold is reached.
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
func (r *userRepository) GetByID(ctx context.Context, userID id.ID) (*model.User, error) {
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

// GetAdmin returns the single Super Admin account (there is at most one by design).
func (r *userRepository) GetAdmin(ctx context.Context) (*model.User, error) {
	user := new(model.User)
	if err := r.DB.WithContext(ctx).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// ResetCredentials replaces the password and recovery code hashes of the Super
// Admin and revokes every active session of the account and every unexpired
// Personal Access Token, in a single transaction: either the whole reset
// applies or nothing changes. Nothing obtained under the old password (a stolen
// session cookie, or a token minted with it) can outlive the reset.
func (r *userRepository) ResetCredentials(ctx context.Context, userID id.ID, passwordHash, recoveryCodeHash string, revokedAt time.Time) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).
			Where("id = ?", userID).
			Updates(map[string]any{
				"password_hash":         passwordHash,
				"recovery_code_hash":    recoveryCodeHash,
				"totp_secret_encrypted": "",
				"totp_enabled":          false,
				"totp_last_step":        0,
				"totp_failed_attempts":  0,
				"totp_locked_until":     nil,
			}).Error; err != nil {
			return err
		}

		if err := revokeSessions(tx, userID, "", revokedAt); err != nil {
			return err
		}

		return tx.Model(&model.PersonalAccessToken{}).
			Where("user_id = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", userID, revokedAt).
			Update("revoked_at", revokedAt).Error
	})
}

// ChangePassword replaces the Super Admin password hash and revokes every
// session except keepSessionID in a single transaction, so a stolen session
// cannot outlive the password change while the caller stays signed in.
func (r *userRepository) ChangePassword(ctx context.Context, userID id.ID, passwordHash string, keepSessionID id.ID, changedAt time.Time) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).
			Where("id = ?", userID).
			Update("password_hash", passwordHash).Error; err != nil {
			return err
		}

		return revokeSessions(tx, userID, keepSessionID, changedAt)
	})
}

// SetTOTPSecret stores a new pending TOTP seed and resets every TOTP state
// field. TOTP stays disabled until EnableTOTP confirms the seed with a valid
// code, so an abandoned enrollment never locks the account out.
func (r *userRepository) SetTOTPSecret(ctx context.Context, userID id.ID, encryptedSecret string) error {
	return r.DB.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"totp_secret_encrypted": encryptedSecret,
			"totp_enabled":          false,
			"totp_last_step":        0,
			"totp_failed_attempts":  0,
			"totp_locked_until":     nil,
		}).Error
}

// EnableTOTP activates TOTP for the user and records the step of the code that
// completed enrollment, so that same code cannot be replayed at login.
func (r *userRepository) EnableTOTP(ctx context.Context, userID id.ID, step int64) error {
	return r.DB.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"totp_enabled":         true,
			"totp_last_step":       step,
			"totp_failed_attempts": 0,
			"totp_locked_until":    nil,
		}).Error
}

// DisableTOTP removes the seed and every TOTP-related counter.
func (r *userRepository) DisableTOTP(ctx context.Context, userID id.ID) error {
	return r.DB.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"totp_secret_encrypted": "",
			"totp_enabled":          false,
			"totp_last_step":        0,
			"totp_failed_attempts":  0,
			"totp_locked_until":     nil,
		}).Error
}

// ConsumeTOTPStep accepts a code's time step exactly once. The conditional
// update makes concurrent requests with the same code race safely: only the
// first one observes a higher step and wins, the rest are treated as replays.
func (r *userRepository) ConsumeTOTPStep(ctx context.Context, userID id.ID, step int64) (bool, error) {
	res := r.DB.WithContext(ctx).Model(&model.User{}).
		Where("id = ? AND totp_last_step < ?", userID, step).
		UpdateColumns(map[string]any{
			"totp_last_step":       step,
			"totp_failed_attempts": 0,
			"totp_locked_until":    nil,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// RecordTOTPFailure increments the consecutive failure counter. When the
// threshold is reached, the counter resets and verification is locked until
// lockedUntil. Both CASE expressions read the same pre-update row, so the
// decision is consistent within one statement.
func (r *userRepository) RecordTOTPFailure(ctx context.Context, userID id.ID, threshold int, lockedUntil time.Time) error {
	return r.DB.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		UpdateColumns(map[string]any{
			"totp_failed_attempts": gorm.Expr("CASE WHEN totp_failed_attempts + 1 >= ? THEN 0 ELSE totp_failed_attempts + 1 END", threshold),
			"totp_locked_until":    gorm.Expr("CASE WHEN totp_failed_attempts + 1 >= ? THEN ? ELSE totp_locked_until END", threshold, lockedUntil),
		}).Error
}

// SetName updates the display name of the Super Admin account.
func (r *userRepository) SetName(ctx context.Context, userID id.ID, name string) error {
	return r.DB.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Update("name", name).Error
}

// SetEmail updates the email address of the Super Admin account. Callers must
// pass a normalized lowercase address so login lookups keep matching.
func (r *userRepository) SetEmail(ctx context.Context, userID id.ID, email string) error {
	return r.DB.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Update("email", email).Error
}
