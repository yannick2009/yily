package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/crypto"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/repository"
)

const (
	DefaultPATTTL = 30 * 24 * time.Hour // DefaultPATTTL is the default Personal Access Token lifetime.
)

var (
	ErrPATExpired           = errors.New("personal access token expired")            // ErrPATExpired is returned when a PAT has reached or passed its expiration date.
	ErrPATRevoked           = errors.New("personal access token revoked")            // ErrPATRevoked is returned when a PAT has been explicitly revoked.
	ErrInvalidPATName       = errors.New("invalid personal access token name")       // ErrInvalidPATName is returned when a PAT name is empty or whitespace-only.
	ErrInvalidPATExpiration = errors.New("invalid personal access token expiration") // ErrInvalidPATExpiration is returned when a PAT expiration duration is negative or invalid.
)

type authPATContextKey struct{}

// WithAuthenticatedPAT stores the authenticated User and PersonalAccessToken in the request context.
func WithAuthenticatedPAT(ctx context.Context, user *model.User, pat *model.PersonalAccessToken) context.Context {
	ctx = context.WithValue(ctx, authUserContextKey{}, user)
	ctx = context.WithValue(ctx, authPATContextKey{}, pat)
	return ctx
}

// AuthenticatedPATFromContext retrieves the authenticated PersonalAccessToken injected by the auth middleware from the context.
func AuthenticatedPATFromContext(ctx context.Context) (*model.PersonalAccessToken, bool) {
	pat, ok := ctx.Value(authPATContextKey{}).(*model.PersonalAccessToken)
	return pat, ok && pat != nil
}

// PATService defines the interface for Personal Access Token creation, validation, listing, and revocation.
type PATService interface {
	Create(ctx context.Context, userID uuid.UUID, name string, ttl time.Duration) (string, *model.PersonalAccessToken, error) // Create generates a new PAT, stores only its SHA-256 hash, and returns the raw token once.
	Validate(ctx context.Context, rawToken string) (*model.User, *model.PersonalAccessToken, error)                           // Validate verifies a raw PAT against its stored SHA-256 hash and loads the owning user.
	GetAll(ctx context.Context, userID uuid.UUID) ([]model.PersonalAccessToken, error)                                        // GetAll lists all PATs belonging to a user without exposing raw tokens or hashes.
	Revoke(ctx context.Context, userID uuid.UUID, patID uuid.UUID) error                                                      // Revoke marks a user's PAT as revoked by its ID.
}

// patService implements PATService.
type patService struct {
	userRepository repository.UserRepository // userRepository verifies and loads the User owning a PAT.
	patRepository  repository.PATRepository  // patRepository manages Personal Access Token persistence.
	defaultTTL     time.Duration             // defaultTTL is the default token expiration duration.
	nowFunc        func() time.Time          // nowFunc returns the current UTC timestamp.
	audit          AuditService              // audit records audit events for token actions.
}

// NewPATService creates a new PATService with the provided repositories and audit service.
func NewPATService(userRepository repository.UserRepository, patRepository repository.PATRepository, audit AuditService) PATService {
	return &patService{
		userRepository: userRepository,
		patRepository:  patRepository,
		audit:          audit,
		defaultTTL:     DefaultPATTTL,
		nowFunc:        time.Now,
	}
}

// Create generates a 256-bit PAT with the "yily_" prefix, persists only its SHA-256 hash in SQLite, and returns the raw token once.
func (s *patService) Create(ctx context.Context, userID uuid.UUID, name string, ttl time.Duration) (string, *model.PersonalAccessToken, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return "", nil, ErrInvalidPATName
	}

	if ttl < 0 {
		return "", nil, ErrInvalidPATExpiration
	}
	if ttl == 0 {
		ttl = s.defaultTTL
	}

	if _, err := s.userRepository.GetByID(ctx, userID); err != nil {
		return "", nil, err
	}

	rawToken, tokenHash, err := crypto.GeneratePAT()
	if err != nil {
		return "", nil, err
	}

	now := s.nowFunc().UTC()
	pat := &model.PersonalAccessToken{
		UserID:    userID,
		Name:      trimmedName,
		TokenHash: tokenHash,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}

	if err := s.patRepository.Create(ctx, pat); err != nil {
		return "", nil, err
	}

	_ = s.audit.Record(ctx, types.ActionCreated, types.ResourceToken, pat.ID, pat.Name)

	return rawToken, pat, nil
}

// Validate checks the "yily_" prefix, looks up the SHA-256 hash in SQLite, verifies the PAT is neither revoked nor expired, and loads its owning User.
func (s *patService) Validate(ctx context.Context, rawToken string) (*model.User, *model.PersonalAccessToken, error) {
	rawToken = strings.TrimSpace(rawToken)
	// Require the mandatory "yily_" prefix so arbitrary strings or raw DB SHA-256 hashes are rejected immediately.
	if !strings.HasPrefix(rawToken, crypto.PATPrefix) || len(rawToken) <= len(crypto.PATPrefix) {
		return nil, nil, ErrUnauthorized
	}

	tokenHash := crypto.HashToken(rawToken)
	pat, err := s.patRepository.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrPATNotFound) {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, err
	}

	if pat.IsRevoked() {
		return nil, nil, ErrPATRevoked
	}

	now := s.nowFunc().UTC()
	if pat.IsExpired(now) {
		return nil, nil, ErrPATExpired
	}

	// Load the User owning this PAT so downstream handlers can access the authenticated identity from context.
	user, err := s.userRepository.GetByID(ctx, pat.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, err
	}

	return user, pat, nil
}

// GetAll retrieves all PAT metadata belonging to a user without exposing raw tokens or SHA-256 hashes.
func (s *patService) GetAll(ctx context.Context, userID uuid.UUID) ([]model.PersonalAccessToken, error) {
	return s.patRepository.GetAllByUserID(ctx, userID)
}

// Revoke marks a user's active PAT as revoked in SQLite so it can no longer authenticate API requests.
func (s *patService) Revoke(ctx context.Context, userID uuid.UUID, patID uuid.UUID) error {
	name := ""
	if pat, err := s.patRepository.GetByID(ctx, patID); err == nil {
		name = pat.Name
	}

	now := s.nowFunc().UTC()
	if err := s.patRepository.RevokeByID(ctx, patID, userID, now); err != nil {
		return err
	}

	if name != "" {
		_ = s.audit.Record(ctx, types.ActionRevoked, types.ResourceToken, patID, name)
	}
	return nil
}
