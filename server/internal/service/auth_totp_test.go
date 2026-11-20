package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yannick2009/yily/internal/crypto"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/id"
	"github.com/yannick2009/yily/internal/repository"
)

const testEmail = "admin@example.com"

// fakeUserRepository is an in-memory UserRepository covering the TOTP flow.
type fakeUserRepository struct {
	mu   sync.Mutex
	user *model.User
}

func (r *fakeUserRepository) Create(_ context.Context, user *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.user != nil {
		return repository.ErrUserAlreadyExists
	}
	if user.ID == "" {
		user.ID = id.New(id.PrefixUser)
	}
	r.user = cloneUser(user)
	return nil
}

func (r *fakeUserRepository) GetByEmail(_ context.Context, email string) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.user == nil || r.user.Email != email {
		return nil, repository.ErrUserNotFound
	}
	return cloneUser(r.user), nil
}

func (r *fakeUserRepository) GetByID(_ context.Context, userID id.ID) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.user == nil || r.user.ID != userID {
		return nil, repository.ErrUserNotFound
	}
	return cloneUser(r.user), nil
}

func (r *fakeUserRepository) Count(context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.user == nil {
		return 0, nil
	}
	return 1, nil
}

func (r *fakeUserRepository) GetAdmin(context.Context) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.user == nil {
		return nil, repository.ErrUserNotFound
	}
	return cloneUser(r.user), nil
}

func (r *fakeUserRepository) ResetCredentials(_ context.Context, userID id.ID, passwordHash, recoveryCodeHash string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.user == nil || r.user.ID != userID {
		return repository.ErrUserNotFound
	}
	r.user.PasswordHash = passwordHash
	r.user.RecoveryCodeHash = recoveryCodeHash
	r.user.TOTPSecretEncrypted = ""
	r.user.TOTPEnabled = false
	r.user.TOTPLastStep = 0
	r.user.TOTPFailedAttempts = 0
	r.user.TOTPLockedUntil = nil
	return nil
}

func (r *fakeUserRepository) SetTOTPSecret(_ context.Context, userID id.ID, encryptedSecret string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.user == nil || r.user.ID != userID {
		return repository.ErrUserNotFound
	}
	r.user.TOTPSecretEncrypted = encryptedSecret
	r.user.TOTPEnabled = false
	r.user.TOTPLastStep = 0
	r.user.TOTPFailedAttempts = 0
	r.user.TOTPLockedUntil = nil
	return nil
}

func (r *fakeUserRepository) EnableTOTP(_ context.Context, userID id.ID, step int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.user == nil || r.user.ID != userID {
		return repository.ErrUserNotFound
	}
	r.user.TOTPEnabled = true
	r.user.TOTPLastStep = step
	r.user.TOTPFailedAttempts = 0
	r.user.TOTPLockedUntil = nil
	return nil
}

func (r *fakeUserRepository) DisableTOTP(_ context.Context, userID id.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.user == nil || r.user.ID != userID {
		return repository.ErrUserNotFound
	}
	r.user.TOTPSecretEncrypted = ""
	r.user.TOTPEnabled = false
	r.user.TOTPLastStep = 0
	r.user.TOTPFailedAttempts = 0
	r.user.TOTPLockedUntil = nil
	return nil
}

func (r *fakeUserRepository) ConsumeTOTPStep(_ context.Context, userID id.ID, step int64) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.user == nil || r.user.ID != userID {
		return false, repository.ErrUserNotFound
	}
	if r.user.TOTPLastStep >= step {
		return false, nil
	}
	r.user.TOTPLastStep = step
	r.user.TOTPFailedAttempts = 0
	r.user.TOTPLockedUntil = nil
	return true, nil
}

func (r *fakeUserRepository) RecordTOTPFailure(_ context.Context, userID id.ID, threshold int, lockedUntil time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.user == nil || r.user.ID != userID {
		return repository.ErrUserNotFound
	}
	r.user.TOTPFailedAttempts++
	if r.user.TOTPFailedAttempts >= threshold {
		r.user.TOTPFailedAttempts = 0
		locked := lockedUntil
		r.user.TOTPLockedUntil = &locked
	}
	return nil
}

func (r *fakeUserRepository) snapshot() *model.User {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneUser(r.user)
}

// fakeSessionRepository is an in-memory SessionRepository.
type fakeSessionRepository struct {
	mu       sync.Mutex
	sessions map[string]*model.Session
}

func (r *fakeSessionRepository) Create(_ context.Context, session *model.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if session.ID == "" {
		session.ID = id.New(id.PrefixSession)
	}
	stored := *session
	r.sessions[session.TokenHash] = &stored
	return nil
}

func (r *fakeSessionRepository) GetByTokenHash(_ context.Context, tokenHash string) (*model.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessions[tokenHash]
	if !ok {
		return nil, repository.ErrSessionNotFound
	}
	stored := *session
	return &stored, nil
}

func (r *fakeSessionRepository) RevokeByTokenHash(_ context.Context, tokenHash string, revokedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessions[tokenHash]
	if !ok {
		return repository.ErrSessionNotFound
	}
	session.RevokedAt = &revokedAt
	return nil
}

func (r *fakeSessionRepository) TouchActivity(_ context.Context, sessionID id.ID, activityAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, session := range r.sessions {
		if session.ID == sessionID {
			session.LastActivityAt = activityAt
		}
	}
	return nil
}

func (r *fakeSessionRepository) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.sessions)
}

func cloneUser(user *model.User) *model.User {
	if user == nil {
		return nil
	}
	copied := *user
	return &copied
}

// testEnv bundles an AuthService wired to in-memory repositories with a fixed,
// adjustable clock and credentials generated for the test.
type testEnv struct {
	svc          AuthService
	users        *fakeUserRepository
	sessions     *fakeSessionRepository
	now          *time.Time
	recoveryCode string
	password     string
}

// randomSecret returns a fresh unpredictable string for test credentials.
func randomSecret(t *testing.T) string {
	t.Helper()
	value, err := crypto.GenerateCSRFToken()
	if err != nil {
		t.Fatalf("generate test secret: %v", err)
	}
	return value
}

// newTestEnv builds an AuthService around one Super Admin whose password and
// recovery code are generated per test, never hard-coded.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	password := "Aa1!" + randomSecret(t)
	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	recoveryCode, err := crypto.GenerateRecoveryCode()
	if err != nil {
		t.Fatalf("generate recovery code: %v", err)
	}
	recoveryHash, err := crypto.HashRecoveryCode(recoveryCode)
	if err != nil {
		t.Fatalf("hash recovery code: %v", err)
	}

	user := &model.User{
		ID:               id.New(id.PrefixUser),
		Name:             "Admin",
		Email:            testEmail,
		PasswordHash:     passwordHash,
		RecoveryCodeHash: recoveryHash,
	}
	users := &fakeUserRepository{user: user}
	sessions := &fakeSessionRepository{sessions: map[string]*model.Session{}}
	now := time.Unix(1_700_000_000, 0).UTC()

	svc := NewAuthServiceWithConfig(
		users,
		sessions,
		"",
		bytes.Repeat([]byte{0x2a}, 32),
		AuthConfig{NowFunc: func() time.Time { return now }},
	)
	return &testEnv{svc: svc, users: users, sessions: sessions, now: &now, recoveryCode: recoveryCode, password: password}
}

// enableTOTP runs a full enrollment and returns the Base32 seed.
func enableTOTP(t *testing.T, env *testEnv, at time.Time) string {
	t.Helper()

	setup, err := env.svc.SetupTOTP(context.Background(), env.users.snapshot().ID)
	if err != nil {
		t.Fatalf("SetupTOTP: %v", err)
	}
	if setup.Secret == "" || setup.URI == "" || !strings.HasPrefix(setup.QR, "data:image/png;base64,") {
		t.Fatalf("incomplete TOTP setup payload: %+v", setup)
	}

	code, err := crypto.TOTPCode(setup.Secret, at)
	if err != nil {
		t.Fatalf("TOTPCode: %v", err)
	}
	if err := env.svc.ConfirmTOTP(context.Background(), env.users.snapshot().ID, code); err != nil {
		t.Fatalf("ConfirmTOTP: %v", err)
	}
	if !env.users.snapshot().TOTPEnabled {
		t.Fatal("TOTP was not enabled after confirmation")
	}
	return setup.Secret
}

// wrongTOTPCode returns a six-digit code that matches none of the accepted steps.
func wrongTOTPCode(t *testing.T, secret string, at time.Time) string {
	t.Helper()
	for i := 0; i < 100; i++ {
		candidate := fmt.Sprintf("%06d", i)
		if _, ok := crypto.ValidateTOTP(secret, candidate, at); !ok {
			return candidate
		}
	}
	t.Fatal("could not find an invalid TOTP code")
	return ""
}

func TestTOTPEnrollmentAndPasswordLogin(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	at := *env.now

	secret := enableTOTP(t, env, at)

	if _, _, _, err := env.svc.Login(ctx, testEmail, env.password, "", ""); !errors.Is(err, ErrTOTPRequired) {
		t.Fatalf("login without a code = %v, want ErrTOTPRequired", err)
	}

	// The code used to confirm enrollment is consumed and cannot be replayed.
	enrollmentCode, err := crypto.TOTPCode(secret, at)
	if err != nil {
		t.Fatalf("TOTPCode: %v", err)
	}
	if _, _, _, err := env.svc.Login(ctx, testEmail, env.password, enrollmentCode, ""); !errors.Is(err, ErrInvalidTOTPCode) {
		t.Fatalf("replayed enrollment code = %v, want ErrInvalidTOTPCode", err)
	}

	*env.now = at.Add(crypto.TOTPPeriod * time.Second)
	code, err := crypto.TOTPCode(secret, *env.now)
	if err != nil {
		t.Fatalf("TOTPCode: %v", err)
	}

	rawToken, session, user, err := env.svc.Login(ctx, testEmail, env.password, code, "")
	if err != nil {
		t.Fatalf("login with a valid code: %v", err)
	}
	if rawToken == "" || session == nil || user == nil {
		t.Fatal("login returned an incomplete session")
	}
	if env.sessions.count() != 1 {
		t.Fatalf("stored sessions = %d, want 1", env.sessions.count())
	}

	// A second login with the same code is a replay.
	if _, _, _, err := env.svc.Login(ctx, testEmail, env.password, code, ""); !errors.Is(err, ErrInvalidTOTPCode) {
		t.Fatalf("replayed login code = %v, want ErrInvalidTOTPCode", err)
	}
}

func TestTOTPPasswordlessLogin(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	secret := enableTOTP(t, env, *env.now)
	*env.now = env.now.Add(crypto.TOTPPeriod * time.Second)

	code, err := crypto.TOTPCode(secret, *env.now)
	if err != nil {
		t.Fatalf("TOTPCode: %v", err)
	}

	rawToken, session, user, err := env.svc.LoginWithTOTP(ctx, testEmail, code, "")
	if err != nil {
		t.Fatalf("passwordless login: %v", err)
	}
	if rawToken == "" || session == nil || user == nil {
		t.Fatal("passwordless login returned an incomplete session")
	}

	if _, _, _, err := env.svc.LoginWithTOTP(ctx, "nobody@example.com", code, ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown email = %v, want ErrInvalidCredentials", err)
	}
	if _, _, _, err := env.svc.LoginWithTOTP(ctx, testEmail, wrongTOTPCode(t, secret, *env.now), ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong code = %v, want the generic ErrInvalidCredentials", err)
	}
}

func TestTOTPLockoutAfterRepeatedFailures(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	secret := enableTOTP(t, env, *env.now)
	wrong := wrongTOTPCode(t, secret, *env.now)

	for i := 0; i < TOTPMaxAttempts; i++ {
		if _, _, _, err := env.svc.Login(ctx, testEmail, env.password, wrong, ""); !errors.Is(err, ErrInvalidTOTPCode) {
			t.Fatalf("attempt %d = %v, want ErrInvalidTOTPCode", i+1, err)
		}
	}
	if env.users.snapshot().TOTPLockedUntil == nil {
		t.Fatal("account was not locked after the failure threshold")
	}

	*env.now = env.now.Add(crypto.TOTPPeriod * time.Second)
	code, err := crypto.TOTPCode(secret, *env.now)
	if err != nil {
		t.Fatalf("TOTPCode: %v", err)
	}
	if _, _, _, err := env.svc.Login(ctx, testEmail, env.password, code, ""); !errors.Is(err, ErrTOTPLocked) {
		t.Fatalf("login during lockout = %v, want ErrTOTPLocked", err)
	}

	*env.now = env.now.Add(TOTPLockoutDuration + crypto.TOTPPeriod*time.Second)
	code, err = crypto.TOTPCode(secret, *env.now)
	if err != nil {
		t.Fatalf("TOTPCode: %v", err)
	}
	if _, _, _, err := env.svc.Login(ctx, testEmail, env.password, code, ""); err != nil {
		t.Fatalf("login after lockout expired: %v", err)
	}
}

func TestDisableTOTPRequiresThePassword(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	enableTOTP(t, env, *env.now)
	userID := env.users.snapshot().ID

	if err := env.svc.DisableTOTP(ctx, userID, "definitely-wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("disable with a wrong password = %v, want ErrInvalidCredentials", err)
	}
	if !env.users.snapshot().TOTPEnabled {
		t.Fatal("TOTP was disabled despite the wrong password")
	}

	if err := env.svc.DisableTOTP(ctx, userID, env.password); err != nil {
		t.Fatalf("disable TOTP: %v", err)
	}
	got := env.users.snapshot()
	if got.TOTPEnabled || got.TOTPSecretEncrypted != "" {
		t.Fatalf("TOTP state after disable = %+v", got)
	}
	if err := env.svc.DisableTOTP(ctx, userID, env.password); !errors.Is(err, ErrTOTPNotEnabled) {
		t.Fatalf("second disable = %v, want ErrTOTPNotEnabled", err)
	}
}

func TestRecoverDisablesTOTP(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	enableTOTP(t, env, *env.now)

	if _, err := env.svc.Recover(ctx, env.recoveryCode, "Aa1!"+randomSecret(t)); err != nil {
		t.Fatalf("Recover: %v", err)
	}

	got := env.users.snapshot()
	if got.TOTPEnabled || got.TOTPSecretEncrypted != "" {
		t.Fatal("recovery did not disable TOTP")
	}
}
