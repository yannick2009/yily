package repository

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/yannick2009/yily/internal/crypto"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/id"
	"gorm.io/gorm"
)

// newTOTPTestDB opens a private in-memory SQLite database with the tables the
// user repository touches. A single connection keeps :memory: shared between
// statements.
func newTOTPTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("database handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&model.User{}, &model.Session{}, &model.PersonalAccessToken{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// randomValue returns an unpredictable filler for hash columns.
func randomValue(t *testing.T) string {
	t.Helper()
	value, err := crypto.GenerateCSRFToken()
	if err != nil {
		t.Fatalf("generate value: %v", err)
	}
	return value
}

func TestUserRepositoryTOTPState(t *testing.T) {
	db := newTOTPTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Name:             "Admin",
		Email:            "admin@example.com",
		PasswordHash:     randomValue(t),
		RecoveryCodeHash: randomValue(t),
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// A pending enrollment is stored but stays inert.
	if err := repo.SetTOTPSecret(ctx, user.ID, "encrypted-seed"); err != nil {
		t.Fatalf("SetTOTPSecret: %v", err)
	}
	got := mustGetUser(t, repo, user.ID)
	if got.TOTPEnabled || got.TOTPSecretEncrypted != "encrypted-seed" {
		t.Fatalf("pending TOTP state = %+v", got)
	}

	// Enabling records the first consumed step.
	if err := repo.EnableTOTP(ctx, user.ID, 100); err != nil {
		t.Fatalf("EnableTOTP: %v", err)
	}
	got = mustGetUser(t, repo, user.ID)
	if !got.TOTPEnabled || got.TOTPLastStep != 100 {
		t.Fatalf("enabled TOTP state = %+v", got)
	}

	// A step can be consumed once; an equal or lower step is a replay.
	if accepted, err := repo.ConsumeTOTPStep(ctx, user.ID, 100); err != nil || accepted {
		t.Fatalf("ConsumeTOTPStep(100) = %v, %v; want false, nil", accepted, err)
	}
	if accepted, err := repo.ConsumeTOTPStep(ctx, user.ID, 101); err != nil || !accepted {
		t.Fatalf("ConsumeTOTPStep(101) = %v, %v; want true, nil", accepted, err)
	}

	// Five failures reset the counter and set the lockout deadline.
	lockUntil := time.Now().UTC().Add(5 * time.Minute).Truncate(time.Second)
	for i := 0; i < 5; i++ {
		if err := repo.RecordTOTPFailure(ctx, user.ID, 5, lockUntil); err != nil {
			t.Fatalf("RecordTOTPFailure %d: %v", i+1, err)
		}
	}
	got = mustGetUser(t, repo, user.ID)
	if got.TOTPFailedAttempts != 0 {
		t.Errorf("failed attempts = %d, want 0 after reaching the threshold", got.TOTPFailedAttempts)
	}
	if got.TOTPLockedUntil == nil || !got.TOTPLockedUntil.UTC().Equal(lockUntil) {
		t.Errorf("locked until = %v, want %v", got.TOTPLockedUntil, lockUntil)
	}

	// Disabling clears the seed and every counter.
	if err := repo.DisableTOTP(ctx, user.ID); err != nil {
		t.Fatalf("DisableTOTP: %v", err)
	}
	got = mustGetUser(t, repo, user.ID)
	if got.TOTPEnabled || got.TOTPSecretEncrypted != "" || got.TOTPLastStep != 0 || got.TOTPLockedUntil != nil {
		t.Fatalf("TOTP state after disable = %+v", got)
	}

	// Credential reset clears TOTP as well.
	if err := repo.SetTOTPSecret(ctx, user.ID, "second-seed"); err != nil {
		t.Fatalf("SetTOTPSecret again: %v", err)
	}
	if err := repo.EnableTOTP(ctx, user.ID, 200); err != nil {
		t.Fatalf("EnableTOTP again: %v", err)
	}
	if err := repo.ResetCredentials(ctx, user.ID, randomValue(t), randomValue(t), time.Now().UTC()); err != nil {
		t.Fatalf("ResetCredentials: %v", err)
	}
	got = mustGetUser(t, repo, user.ID)
	if got.TOTPEnabled || got.TOTPSecretEncrypted != "" || got.TOTPLastStep != 0 || got.TOTPFailedAttempts != 0 || got.TOTPLockedUntil != nil {
		t.Fatalf("TOTP state after credential reset = %+v", got)
	}
}

func mustGetUser(t *testing.T, repo UserRepository, userID id.ID) *model.User {
	t.Helper()
	user, err := repo.GetByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	return user
}
