package services

import (
	"context"
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
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")                    // ErrInvalidCredentials is returned when login email or password verification fails.
	ErrWeakPassword       = errors.New("password must be at least 12 characters long") // ErrWeakPassword is returned when the provided password is shorter than MinPasswordLength.
	ErrUnauthorized       = errors.New("unauthorized")                                 // ErrUnauthorized is returned when a request lacks a valid session or PAT.
	ErrSessionExpired     = errors.New("session expired")                              // ErrSessionExpired is returned when a session has exceeded its TTL or idle timeout.
	ErrSessionRevoked     = errors.New("session revoked")                              // ErrSessionRevoked is returned when a session has been explicitly logged out or rotated.
)

type authUserContextKey struct{}
type authSessionContextKey struct{}

// WithAuthenticatedIdentity stores the authenticated User and Session in the request context.
func WithAuthenticatedIdentity(ctx context.Context, user *model.User, session *model.Session) context.Context {
	ctx = context.WithValue(ctx, authUserContextKey{}, user)
	ctx = context.WithValue(ctx, authSessionContextKey{}, session)
	return ctx
}

// AuthenticatedUserFromContext retrieves the authenticated User injected by the auth middleware from the context.
func AuthenticatedUserFromContext(ctx context.Context) (*model.User, bool) {
	user, ok := ctx.Value(authUserContextKey{}).(*model.User)
	return user, ok && user != nil
}

// AuthenticatedSessionFromContext retrieves the authenticated Session injected by the auth middleware from the context.
func AuthenticatedSessionFromContext(ctx context.Context) (*model.Session, bool) {
	session, ok := ctx.Value(authSessionContextKey{}).(*model.Session)
	return session, ok && session != nil
}

// AuthService defines the interface for Super Admin authentication and server-side session management.
type AuthService interface {
	Setup(ctx context.Context, name, email, password string) (*model.User, error)                                     // Setup creates the initial Super Admin account.
	Login(ctx context.Context, email, password, existingRawToken string) (string, *model.Session, *model.User, error) // Login verifies credentials, rotates any prior session, and issues a new session.
	ValidateSession(ctx context.Context, rawToken string) (*model.User, *model.Session, error)                        // ValidateSession verifies a raw session token against its stored SHA-256 hash and loads the owning user.
	Logout(ctx context.Context, rawToken string) error                                                                // Logout revokes a server-side session by hashing its raw token.
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

	dummyOnce sync.Once // dummyOnce ensures dummyHash is computed at most once.
	dummyHash string    // dummyHash is a pre-computed Argon2id PHC hash used when login email is not found to equalize response time and prevent timing-based email enumeration.
}

// NewAuthService creates a new AuthService with default session TTL and inactivity settings.
func NewAuthService(userRepository repository.UserRepository, sessionRepository repository.SessionRepository) AuthService {
	return NewAuthServiceWithConfig(userRepository, sessionRepository, AuthConfig{})
}

// NewAuthServiceWithConfig creates a new AuthService with custom session TTL, idle timeout, and clock function.
func NewAuthServiceWithConfig(userRepository repository.UserRepository, sessionRepository repository.SessionRepository, cfg AuthConfig) AuthService {
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

// Setup creates the single Super Admin account with a normalized lowercase email and an Argon2id password hash.
func (s *authService) Setup(ctx context.Context, name, email, password string) (*model.User, error) {
	if len(password) < MinPasswordLength {
		return nil, ErrWeakPassword
	}

	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Name:         strings.TrimSpace(name),
		Email:        normalizeEmail(email),
		PasswordHash: passwordHash,
	}

	if err := s.userRepository.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
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

	session := &model.Session{
		UserID:         user.ID,
		TokenHash:      tokenHash,
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
