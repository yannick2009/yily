package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"uuid"

	"github.com/yannick2009/yily/internal/crypto"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/repository"
)

const (
	DefaultPATTTL = 30 * 24 * time.Hour // DefaultPATTTL is the default Personal Access Token lifetime.
	MaxPATTTL     = 90 * 24 * time.Hour // MaxPATTTL caps the lifetime of a Personal Access Token to limit exposure in case of leak.
)

// AllowedPATTTLDays enumerates the fixed set of PAT lifetimes accepted at creation and rotation.
var AllowedPATTTLDays = []int{7, 30, 60, 90}

var (
	ErrPATExpired           = errors.New("personal access token expired")            // ErrPATExpired is returned when a PAT has reached or passed its expiration date.
	ErrPATRevoked           = errors.New("personal access token revoked")            // ErrPATRevoked is returned when a PAT has been explicitly revoked.
	ErrInvalidPATName       = errors.New("invalid personal access token name")       // ErrInvalidPATName is returned when a PAT name is empty or whitespace-only.
	ErrInvalidPATExpiration = errors.New("invalid personal access token expiration") // ErrInvalidPATExpiration is returned when a PAT expiration duration is negative, zero, or above the 90-day cap, or when a write-scoped PAT is created without an expiration.
	ErrInvalidPATScope      = errors.New("invalid personal access token scope")      // ErrInvalidPATScope is returned when a PAT scope is not one of the accepted values (read, write).
	ErrPATWriteForbidden    = errors.New("personal access token lacks write scope")  // ErrPATWriteForbidden is returned when a read-only PAT attempts a write operation.
)

// PATService defines the interface for Personal Access Token creation, validation, listing, rotation, and revocation.
type PATService interface {
	Create(ctx context.Context, userID uuid.UUID, name string, scope types.PATScope, ttl time.Duration) (string, *model.PersonalAccessToken, error) // Create generates a new PAT with the given scope, stores only its SHA-256 hash, and returns the raw token once.
	Rotate(ctx context.Context, userID uuid.UUID, patID uuid.UUID, ttl time.Duration) (string, *model.PersonalAccessToken, error)                   // Rotate revokes an existing PAT and issues a new one under the same name and scope, returning the raw replacement token once.
	Validate(ctx context.Context, rawToken string) (*model.User, *model.PersonalAccessToken, error)                                                 // Validate verifies a raw PAT against its stored SHA-256 hash and loads the owning user.
	GetAll(ctx context.Context, userID uuid.UUID) ([]model.PersonalAccessToken, error)                                                              // GetAll lists all PATs belonging to a user without exposing raw tokens or hashes.
	Revoke(ctx context.Context, userID uuid.UUID, patID uuid.UUID) error                                                                            // Revoke marks a user's PAT as revoked by its ID.
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
// A write-scoped PAT must have a positive TTL within MaxPATTTL. A read-scoped PAT may omit the TTL (ttl == 0) for a non-expiring token.
func (s *patService) Create(ctx context.Context, userID uuid.UUID, name string, scope types.PATScope, ttl time.Duration) (string, *model.PersonalAccessToken, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return "", nil, ErrInvalidPATName
	}

	if !scope.IsValid() {
		return "", nil, ErrInvalidPATScope
	}

	if ttl < 0 || ttl > MaxPATTTL {
		return "", nil, ErrInvalidPATExpiration
	}
	// Write-scoped tokens must expire to limit exposure if leaked; read-only tokens may live forever.
	if ttl == 0 && scope == types.PATScopeWrite {
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
		Scope:     scope,
		TokenHash: tokenHash,
		CreatedAt: now,
	}
	if ttl > 0 {
		expiresAt := now.Add(ttl)
		pat.ExpiresAt = &expiresAt
	}

	if err := s.patRepository.Create(ctx, pat); err != nil {
		return "", nil, err
	}

	_ = s.audit.Record(ctx, types.ActionCreated, types.ResourceToken, pat.ID, pat.Name)

	return rawToken, pat, nil
}

// Rotate revokes an existing PAT belonging to the user and issues a replacement under the same name and scope with a fresh TTL. The raw token is returned exactly once.
func (s *patService) Rotate(ctx context.Context, userID uuid.UUID, patID uuid.UUID, ttl time.Duration) (string, *model.PersonalAccessToken, error) {
	if ttl < 0 || ttl > MaxPATTTL {
		return "", nil, ErrInvalidPATExpiration
	}

	existing, err := s.patRepository.GetByID(ctx, patID)
	if err != nil {
		return "", nil, err
	}
	if existing.UserID != userID {
		return "", nil, repository.ErrPATNotFound
	}

	// Write-scoped tokens must expire to limit exposure if leaked; read-only tokens may live forever.
	if ttl == 0 && existing.Scope == types.PATScopeWrite {
		ttl = s.defaultTTL
	}

	rawToken, tokenHash, err := crypto.GeneratePAT()
	if err != nil {
		return "", nil, err
	}

	now := s.nowFunc().UTC()
	replacement := &model.PersonalAccessToken{
		UserID:    userID,
		Name:      existing.Name,
		Scope:     existing.Scope,
		TokenHash: tokenHash,
		CreatedAt: now,
	}
	if ttl > 0 {
		expiresAt := now.Add(ttl)
		replacement.ExpiresAt = &expiresAt
	}
	if err := s.patRepository.Create(ctx, replacement); err != nil {
		return "", nil, err
	}

	// Revoke the old PAT immediately so the leaked secret stops working.
	if !existing.IsRevoked() {
		if err := s.patRepository.RevokeByID(ctx, existing.ID, userID, now); err != nil {
			return "", nil, err
		}
	}

	_ = s.audit.Record(ctx, types.ActionRotated, types.ResourceToken, existing.ID, existing.Name)
	_ = s.audit.Record(ctx, types.ActionCreated, types.ResourceToken, replacement.ID, replacement.Name)

	return rawToken, replacement, nil
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

	// Record this successful authentication. A failure here must not block the request.
	if err := s.patRepository.TouchLastUsed(ctx, pat.ID, now); err == nil {
		pat.LastUsedAt = &now
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
