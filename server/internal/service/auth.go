package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/yannick2009/yily/internal/crypto"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/repository"
)

const (
	DefaultSessionTTL         = 24 * time.Hour // DefaultSessionTTL is the default maximum session lifetime.
	DefaultSessionIdleTimeout = 2 * time.Hour  // DefaultSessionIdleTimeout is the default maximum session inactivity duration.
	MinPasswordLength         = 12             // MinPasswordLength is the minimum Super Admin password length.
	MaxPasswordLength         = 128            // MaxPasswordLength is the maximum Super Admin password length.
)

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")                                                                                      // ErrInvalidCredentials is returned when login email or password verification fails.
	ErrWeakPassword        = errors.New("password must be at least 12 characters long")                                                                   // ErrWeakPassword is returned when the provided password is shorter than MinPasswordLength.
	ErrPasswordTooLong     = errors.New("password must be at most 128 characters long")                                                                   // ErrPasswordTooLong is returned when the provided password is longer than MaxPasswordLength.
	ErrPasswordNotComplex  = errors.New("password must contain at least one uppercase letter, one lowercase letter, one digit and one special character") // ErrPasswordNotComplex is returned when the password fails the character-class complexity policy.
	ErrUnauthorized        = errors.New("unauthorized")                                                                                                   // ErrUnauthorized is returned when a request lacks a valid session or PAT.
	ErrSessionExpired      = errors.New("session expired")                                                                                                // ErrSessionExpired is returned when a session has exceeded its TTL or idle timeout.
	ErrSessionRevoked      = errors.New("session revoked")                                                                                                // ErrSessionRevoked is returned when a session has been explicitly logged out or rotated.
	ErrInvalidSetupToken   = errors.New("invalid setup token")                                                                                            // ErrInvalidSetupToken is returned when the first-run setup token is missing or wrong.
	ErrSetupAlreadyDone    = errors.New("setup has already been completed")                                                                               // ErrSetupAlreadyDone is returned when setup is attempted after the Super Admin already exists.
	ErrSetupTokenNotSet    = errors.New("setup token is not configured; set YILY_SETUP_TOKEN")                                                            // ErrSetupTokenNotSet is returned when setup is attempted on an uninitialized instance without a configured setup token.
	ErrInvalidRecoveryCode = errors.New("invalid recovery code")                                                                                          // ErrInvalidRecoveryCode is returned when the provided recovery code does not match the stored hash.
	ErrRecoveryUnavailable = errors.New("password recovery is not available")                                                                             // ErrRecoveryUnavailable is returned when the Super Admin has no recovery code configured.
)

// AuthService defines the interface for Super Admin authentication and server-side session management.
type AuthService interface {
	Setup(ctx context.Context, name, email, password, setupToken string) (*model.User, string, error)                 // Setup creates the initial Super Admin account, gated by the out-of-band setup token, and returns the one-time recovery code.
	Login(ctx context.Context, email, password, existingRawToken string) (string, *model.Session, *model.User, error) // Login verifies credentials, rotates any prior session, and issues a new session.
	ValidateSession(ctx context.Context, rawToken string) (*model.User, *model.Session, error)                        // ValidateSession verifies a raw session token against its stored SHA-256 hash and loads the owning user.
	Logout(ctx context.Context, rawToken string) error                                                                // Logout revokes a server-side session by hashing its raw token.
	IsInitialized(ctx context.Context) (bool, error)                                                                  // IsInitialized reports whether the Super Admin account has been created.
	Recover(ctx context.Context, recoveryCode, newPassword string) (string, error)                                    // Recover resets the Super Admin password using the recovery code and returns a new recovery code.
}

// AuthConfig holds optional timing parameters for AuthService.
type AuthConfig struct {
	SessionTTL  time.Duration    // SessionTTL is the maximum lifetime of a session before forced expiration.
	IdleTimeout time.Duration    // IdleTimeout is the maximum inactivity period before a session expires.
	NowFunc     func() time.Time // NowFunc returns the current time (injectable for deterministic testing).
}

// authService implements AuthService.
type authService struct {
	userRepository    repository.UserRepository    // userRepository manages Super Admin persistence.
	sessionRepository repository.SessionRepository // sessionRepository manages server-side session persistence.
	sessionTTL        time.Duration                // sessionTTL is the absolute session expiration duration.
	idleTimeout       time.Duration                // idleTimeout is the session inactivity timeout duration.
	nowFunc           func() time.Time             // nowFunc returns the current UTC timestamp.
	setupToken        string                       // setupToken is the out-of-band secret required to create the initial Super Admin.

	dummyOnce sync.Once // dummyOnce ensures dummyHash is computed at most once.
	dummyHash string    // dummyHash is a pre-computed Argon2id PHC hash used when login email is not found to equalize response time and prevent timing-based email enumeration.
}

// NewAuthService creates a new AuthService with default session TTL and inactivity settings.
func NewAuthService(userRepository repository.UserRepository, sessionRepository repository.SessionRepository, setupToken string) AuthService {
	return NewAuthServiceWithConfig(userRepository, sessionRepository, setupToken, AuthConfig{})
}

// NewAuthServiceWithConfig creates a new AuthService with custom session TTL, idle timeout, and clock function.
func NewAuthServiceWithConfig(userRepository repository.UserRepository, sessionRepository repository.SessionRepository, setupToken string, cfg AuthConfig) AuthService {
	ttl := cfg.SessionTTL
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}
	idle := cfg.IdleTimeout
	if idle <= 0 {
		idle = DefaultSessionIdleTimeout
	}
	nowFn := cfg.NowFunc
	if nowFn == nil {
		nowFn = time.Now
	}

	return &authService{
		userRepository:    userRepository,
		sessionRepository: sessionRepository,
		sessionTTL:        ttl,
		idleTimeout:       idle,
		nowFunc:           nowFn,
		setupToken:        setupToken,
	}
}

// getDummyHash lazily computes a dummy Argon2id hash once so unknown-email logins still run Argon2id and take the same time as valid-email logins, preventing timing-based email enumeration.
func (s *authService) getDummyHash() string {
	s.dummyOnce.Do(func() {
		h, err := crypto.HashPassword("dummy-password-for-constant-time-mitigation")
		if err == nil {
			s.dummyHash = h
		}
	})
	return s.dummyHash
}

// Setup creates the single Super Admin account with a normalized lowercase email, an Argon2id password hash and a one-time recovery code. It is gated by the out-of-band setup token and only allowed while the instance is uninitialized.
func (s *authService) Setup(ctx context.Context, name, email, password, setupToken string) (*model.User, string, error) {
	initialized, err := s.IsInitialized(ctx)
	if err != nil {
		return nil, "", err
	}
	if initialized {
		return nil, "", ErrSetupAlreadyDone
	}

	if s.setupToken == "" {
		return nil, "", ErrSetupTokenNotSet
	}

	if subtle.ConstantTimeCompare([]byte(setupToken), []byte(s.setupToken)) != 1 {
		return nil, "", ErrInvalidSetupToken
	}

	if len(password) < MinPasswordLength {
		return nil, "", ErrWeakPassword
	}
	if len(password) > MaxPasswordLength {
		return nil, "", ErrPasswordTooLong
	}
	if !crypto.IsStrongPassword(password) {
		return nil, "", ErrPasswordNotComplex
	}

	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, "", err
	}

	recoveryCode, err := crypto.GenerateRecoveryCode()
	if err != nil {
		return nil, "", err
	}
	recoveryCodeHash, err := crypto.HashRecoveryCode(recoveryCode)
	if err != nil {
		return nil, "", err
	}

	user := &model.User{
		Name:             strings.TrimSpace(name),
		Email:            normalizeEmail(email),
		PasswordHash:     passwordHash,
		RecoveryCodeHash: recoveryCodeHash,
	}

	if err := s.userRepository.Create(ctx, user); err != nil {
		return nil, "", err
	}

	return user, recoveryCode, nil
}

// Login verifies the Super Admin's Argon2id password, revokes any previous session to prevent session fixation, and creates a new SHA-256 hashed session.
func (s *authService) Login(ctx context.Context, email, password, existingRawToken string) (string, *model.Session, *model.User, error) {
	normalizedEmail := normalizeEmail(email)
	if normalizedEmail == "" || password == "" {
		return "", nil, nil, ErrInvalidCredentials
	}

	user, err := s.userRepository.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			// Verify against the dummy Argon2id hash so an unknown email takes the same ~80ms as a wrong password, preventing timing attacks.
			if dummy := s.getDummyHash(); dummy != "" {
				_, _ = crypto.VerifyPassword(password, dummy)
			}
			return "", nil, nil, ErrInvalidCredentials
		}
		return "", nil, nil, err
	}

	valid, err := crypto.VerifyPassword(password, user.PasswordHash)
	if err != nil || !valid {
		return "", nil, nil, ErrInvalidCredentials
	}

	now := s.nowFunc().UTC()

	// Revoke any existing session sent with the login request to prevent session fixation.
	if trimmedToken := strings.TrimSpace(existingRawToken); trimmedToken != "" {
		oldHash := crypto.HashToken(trimmedToken)
		_ = s.sessionRepository.RevokeByTokenHash(ctx, oldHash, now)
	}

	rawToken, tokenHash, err := crypto.GenerateSessionToken()
	if err != nil {
		return "", nil, nil, err
	}

	csrfToken, err := crypto.GenerateCSRFToken()
	if err != nil {
		return "", nil, nil, err
	}

	session := &model.Session{
		UserID:         user.ID,
		TokenHash:      tokenHash,
		CSRFToken:      csrfToken,
		CreatedAt:      now,
		LastActivityAt: now,
		ExpiresAt:      now.Add(s.sessionTTL),
	}

	if err := s.sessionRepository.Create(ctx, session); err != nil {
		return "", nil, nil, err
	}

	return rawToken, session, user, nil
}

// ValidateSession hashes the raw session token with SHA-256, checks revocation and expiration in SQLite, refreshes activity, and loads the owning User.
func (s *authService) ValidateSession(ctx context.Context, rawToken string) (*model.User, *model.Session, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, nil, ErrUnauthorized
	}

	tokenHash := crypto.HashToken(rawToken)
	session, err := s.sessionRepository.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, err
	}

	if session.IsRevoked() {
		return nil, nil, ErrSessionRevoked
	}

	now := s.nowFunc().UTC()
	if session.IsExpired(now, s.idleTimeout) {
		return nil, nil, ErrSessionExpired
	}

	// Load the User owning this session so handlers can read the authenticated identity from the request context.
	user, err := s.userRepository.GetByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, err
	}

	session.LastActivityAt = now
	_ = s.sessionRepository.TouchActivity(ctx, session.ID, now)

	return user, session, nil
}

// Logout hashes the raw session token with SHA-256 and marks the matching session as revoked in SQLite.
func (s *authService) Logout(ctx context.Context, rawToken string) error {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return ErrUnauthorized
	}

	now := s.nowFunc().UTC()
	tokenHash := crypto.HashToken(rawToken)

	if err := s.sessionRepository.RevokeByTokenHash(ctx, tokenHash, now); err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			return ErrUnauthorized
		}
		return err
	}
	return nil
}

// normalizeEmail trims leading/trailing whitespace and converts an email address to lowercase.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Recover resets the Super Admin password using the one-time recovery code. On
// success every active session is revoked and a fresh recovery code is issued
func (s *authService) Recover(ctx context.Context, recoveryCode, newPassword string) (string, error) {
	if len(newPassword) < MinPasswordLength {
		return "", ErrWeakPassword
	}
	if len(newPassword) > MaxPasswordLength {
		return "", ErrPasswordTooLong
	}
	if !crypto.IsStrongPassword(newPassword) {
		return "", ErrPasswordNotComplex
	}

	user, err := s.userRepository.GetAdmin(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", ErrUnauthorized
		}
		return "", err
	}

	if user.RecoveryCodeHash == "" {
		return "", ErrRecoveryUnavailable
	}

	valid, err := crypto.VerifyRecoveryCode(recoveryCode, user.RecoveryCodeHash)
	if err != nil || !valid {
		return "", ErrInvalidRecoveryCode
	}

	newPasswordHash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return "", err
	}

	newRecoveryCode, err := crypto.GenerateRecoveryCode()
	if err != nil {
		return "", err
	}
	newRecoveryCodeHash, err := crypto.HashRecoveryCode(newRecoveryCode)
	if err != nil {
		return "", err
	}

	if err := s.userRepository.ResetCredentials(ctx, user.ID, newPasswordHash, newRecoveryCodeHash, s.nowFunc().UTC()); err != nil {
		return "", err
	}

	return newRecoveryCode, nil
}

// IsInitialized reports whether the Super Admin account has been created.
func (s *authService) IsInitialized(ctx context.Context) (bool, error) {
	count, err := s.userRepository.Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
