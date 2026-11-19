package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/yannick2009/yily/internal/crypto"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/id"
	"github.com/yannick2009/yily/internal/repository"
)

const (
	DefaultPATTTL = 30 * 24 * time.Hour // DefaultPATTTL is the default Personal Access Token lifetime.
	MaxPATTTL     = 90 * 24 * time.Hour // MaxPATTTL caps the lifetime of a Personal Access Token to limit exposure in case of leak.
)

// AllowedPATTTLDays enumerates the fixed set of PAT lifetimes accepted at creation and rotation.
var AllowedPATTTLDays = []int{7, 30, 60, 90}

var (
	ErrPATExpired                 = errors.New("personal access token expired")                             // ErrPATExpired is returned when a PAT has reached or passed its expiration date.
	ErrPATRevoked                 = errors.New("personal access token revoked")                             // ErrPATRevoked is returned when a PAT has been explicitly revoked.
	ErrInvalidPATName             = errors.New("invalid personal access token name")                        // ErrInvalidPATName is returned when a PAT name is empty or whitespace-only.
	ErrInvalidPATExpiration       = errors.New("invalid personal access token expiration")                  // ErrInvalidPATExpiration is returned when a PAT expiration duration is negative, zero, or above the 90-day cap, or when a write-scoped PAT is created without an expiration.
	ErrInvalidPATScope            = errors.New("invalid personal access token scope")                       // ErrInvalidPATScope is returned when a PAT scope is not one of the accepted values (read, write).
	ErrPATWriteForbidden          = errors.New("personal access token lacks write scope")                   // ErrPATWriteForbidden is returned when a read-only PAT attempts a write operation.
	ErrPATResourceForbidden       = errors.New("personal access token is not authorized for this resource") // ErrPATResourceForbidden is returned when a scoped PAT accesses a project or environment outside its restriction.
	ErrPATEnvironmentNotInProject = errors.New("environment does not belong to the specified project")      // ErrPATEnvironmentNotInProject is returned when creating a PAT with mismatched project_id and environment_id.
)

// PATService defines the interface for Personal Access Token creation, validation, listing, rotation, and revocation.
type PATService interface {
	Create(ctx context.Context, userID id.ID, name string, scope types.PATScope, ttl time.Duration, projectID, environmentID *id.ID) (string, *model.PersonalAccessToken, error) // Create generates a new PAT with the given scope, stores only its SHA-256 hash, and returns the raw token once.
	Rotate(ctx context.Context, userID id.ID, patID id.ID, ttl time.Duration) (string, *model.PersonalAccessToken, error)                                                        // Rotate revokes an existing PAT and issues a new one under the same name and scope, returning the raw replacement token once.
	Validate(ctx context.Context, rawToken string) (*model.User, *model.PersonalAccessToken, error)                                                                              // Validate verifies a raw PAT against its stored SHA-256 hash and loads the owning user.
	GetAll(ctx context.Context, userID id.ID) ([]model.PersonalAccessToken, error)                                                                                               // GetAll lists all PATs belonging to a user without exposing raw tokens or hashes.
	Revoke(ctx context.Context, userID id.ID, patID id.ID) error                                                                                                                 // Revoke marks a user's PAT as revoked by its ID.
}

// patService implements PATService.
type patService struct {
	userRepository        repository.UserRepository        // userRepository verifies and loads the User owning a PAT.
	patRepository         repository.PATRepository         // patRepository manages Personal Access Token persistence.
	projectRepository     repository.ProjectRepository     // projectRepository verifies that a PAT's project_id exists.
	environmentRepository repository.EnvironmentRepository // environmentRepository verifies that a PAT's environment_id exists and belongs to the project.
	defaultTTL            time.Duration                    // defaultTTL is the default token expiration duration.
	nowFunc               func() time.Time                 // nowFunc returns the current UTC timestamp.
	audit                 AuditService                     // audit records audit events for token actions.
}

// NewPATService creates a new PATService with the provided repositories and audit service.
func NewPATService(userRepository repository.UserRepository, patRepository repository.PATRepository, projectRepository repository.ProjectRepository, environmentRepository repository.EnvironmentRepository, audit AuditService) PATService {
	return &patService{
		userRepository:        userRepository,
		patRepository:         patRepository,
		projectRepository:     projectRepository,
		environmentRepository: environmentRepository,
		audit:                 audit,
		defaultTTL:            DefaultPATTTL,
		nowFunc:               time.Now,
	}
}

// patMetadata builds the audit metadata for a PAT: its scope and, when it expires, its absolute expiration date.
func patMetadata(scope types.PATScope, expiresAt *time.Time, projectID, environmentID *id.ID) map[string]string {
	metadata := map[string]string{"scope": string(scope)}
	if expiresAt != nil {
		metadata["expires_at"] = expiresAt.Format("2006-01-02 15:04")
	}

	if projectID != nil {
		metadata["project_id"] = projectID.String()
	}

	if environmentID != nil {
		metadata["environment_id"] = environmentID.String()
	}

	return metadata
}

// Create generates a 192-bit PAT with the "yily_" prefix, persists only its SHA-256 hash in SQLite, and returns the raw token once.
// A write-scoped PAT must have a positive TTL within MaxPATTTL. A read-scoped PAT may omit the TTL (ttl == 0) for a non-expiring token.
func (s *patService) Create(ctx context.Context, userID id.ID, name string, scope types.PATScope, ttl time.Duration, projectID, environmentID *id.ID) (string, *model.PersonalAccessToken, error) {
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

	if ttl == 0 && scope == types.PATScopeWrite {
		ttl = s.defaultTTL
	}

	if _, err := s.userRepository.GetByID(ctx, userID); err != nil {
		return "", nil, err
	}

	resolvedProjectID, projectName, environmentName, err := s.resolveCreateScope(ctx, projectID, environmentID)
	if err != nil {
		return "", nil, err
	}

	rawToken, tokenHash, err := crypto.GeneratePAT()
	if err != nil {
		return "", nil, err
	}

	now := s.nowFunc().UTC()
	pat := &model.PersonalAccessToken{
		UserID:          userID,
		Name:            trimmedName,
		Scope:           scope,
		ProjectID:       resolvedProjectID,
		EnvironmentID:   environmentID,
		ProjectName:     projectName,
		EnvironmentName: environmentName,
		TokenHash:       tokenHash,
		CreatedAt:       now,
	}

	if ttl > 0 {
		expiresAt := now.Add(ttl)
		pat.ExpiresAt = &expiresAt
	}

	if err := s.patRepository.Create(ctx, pat); err != nil {
		return "", nil, err
	}

	_ = s.audit.Record(ctx, types.ActionCreated, types.ResourceToken, pat.ID, pat.Name, patMetadata(pat.Scope, pat.ExpiresAt, pat.ProjectID, pat.EnvironmentID))
	return rawToken, pat, nil
}

// resolveCreateScope verifies that the provided projectID and environmentID exist and are consistent, returning the resolved projectID and names for audit logging.
func (s *patService) resolveCreateScope(ctx context.Context, projectID, environmentID *id.ID) (*id.ID, string, string, error) {
	if environmentID != nil {
		env, err := s.environmentRepository.GetByID(ctx, *environmentID)
		if err != nil {
			return nil, "", "", err
		}

		if projectID != nil && *projectID != env.ProjectID {
			return nil, "", "", ErrPATEnvironmentNotInProject
		}

		proj, err := s.projectRepository.GetByID(ctx, env.ProjectID)
		if err != nil {
			return nil, "", "", err
		}

		pid := env.ProjectID
		return &pid, proj.Name, env.Name, nil
	}

	if projectID != nil {
		proj, err := s.projectRepository.GetByID(ctx, *projectID)
		if err != nil {
			return nil, "", "", err
		}

		pid := proj.ID
		return &pid, proj.Name, "", nil
	}

	return nil, "", "", nil
}

// Rotate revokes an existing PAT belonging to the user and issues a replacement under the same name, scope, and resource restriction with a fresh TTL.
func (s *patService) Rotate(ctx context.Context, userID id.ID, patID id.ID, ttl time.Duration) (string, *model.PersonalAccessToken, error) {
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

	if existing.IsRevoked() {
		return "", nil, ErrPATRevoked
	}

	if ttl == 0 && existing.Scope == types.PATScopeWrite {
		ttl = s.defaultTTL
	}

	rawToken, tokenHash, err := crypto.GeneratePAT()
	if err != nil {
		return "", nil, err
	}

	now := s.nowFunc().UTC()
	replacement := &model.PersonalAccessToken{
		UserID:        userID,
		Name:          existing.Name,
		Scope:         existing.Scope,
		ProjectID:     existing.ProjectID,
		EnvironmentID: existing.EnvironmentID,
		TokenHash:     tokenHash,
		CreatedAt:     now,
	}

	if ttl > 0 {
		expiresAt := now.Add(ttl)
		replacement.ExpiresAt = &expiresAt
	}

	if err := s.patRepository.Create(ctx, replacement); err != nil {
		return "", nil, err
	}

	if !existing.IsRevoked() {
		if err := s.patRepository.RevokeByID(ctx, existing.ID, userID, now); err != nil {
			return "", nil, err
		}
	}

	s.populateScopeNames(ctx, replacement)
	_ = s.audit.Record(ctx, types.ActionRotated, types.ResourceToken, existing.ID, existing.Name, patMetadata(existing.Scope, replacement.ExpiresAt, replacement.ProjectID, replacement.EnvironmentID))
	_ = s.audit.Record(ctx, types.ActionCreated, types.ResourceToken, replacement.ID, replacement.Name, patMetadata(replacement.Scope, replacement.ExpiresAt, replacement.ProjectID, replacement.EnvironmentID))

	return rawToken, replacement, nil
}

// Validate checks the "yily_" prefix, looks up the SHA-256 hash in SQLite, verifies the PAT is neither revoked nor expired, and loads its owning User.
func (s *patService) Validate(ctx context.Context, rawToken string) (*model.User, *model.PersonalAccessToken, error) {
	rawToken = strings.TrimSpace(rawToken)
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

	user, err := s.userRepository.GetByID(ctx, pat.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, err
	}

	if pat.EnvironmentID != nil {
		env, err := s.environmentRepository.GetByID(ctx, *pat.EnvironmentID)
		if err != nil {
			if errors.Is(err, repository.ErrEnvironmentNotFound) {
				return nil, nil, ErrUnauthorized
			}
			return nil, nil, err
		}
		pat.EnvironmentName = env.Name
	}

	if pat.ProjectID != nil {
		proj, err := s.projectRepository.GetByID(ctx, *pat.ProjectID)
		if err != nil {
			if errors.Is(err, repository.ErrProjectNotFound) {
				return nil, nil, ErrUnauthorized
			}
			return nil, nil, err
		}
		pat.ProjectName = proj.Name
	}

	if err := s.patRepository.TouchLastUsed(ctx, pat.ID, now); err == nil {
		pat.LastUsedAt = &now
	}
	return user, pat, nil
}

// GetAll retrieves all PAT metadata belonging to a user without exposing raw tokens or SHA-256 hashes.
// Because project_name and environment_name are transient display fields (gorm:"-"), this method resolves
// them in memory with per-call lookup caches so repeated scoped tokens avoid redundant database queries.
func (s *patService) GetAll(ctx context.Context, userID id.ID) ([]model.PersonalAccessToken, error) {
	pats, err := s.patRepository.GetAllByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	projectNames := make(map[id.ID]string)
	envNames := make(map[id.ID]string)

	for i := range pats {
		pats[i].ProjectName = resolveCachedName(pats[i].ProjectID, projectNames, func(pid id.ID) (string, error) {
			proj, err := s.projectRepository.GetByID(ctx, pid)
			if err != nil {
				return "", err
			}
			return proj.Name, nil
		})

		pats[i].EnvironmentName = resolveCachedName(pats[i].EnvironmentID, envNames, func(eid id.ID) (string, error) {
			env, err := s.environmentRepository.GetByID(ctx, eid)
			if err != nil {
				return "", err
			}
			return env.Name, nil
		})
	}

	return pats, nil
}

// resolveCachedName returns the resource name for targetID using cache to avoid duplicate lookups within a call.
func resolveCachedName(targetID *id.ID, cache map[id.ID]string, fetch func(id.ID) (string, error)) string {
	if targetID == nil {
		return ""
	}

	if name, cached := cache[*targetID]; cached {
		return name
	}

	name := ""
	if fetched, err := fetch(*targetID); err == nil {
		name = fetched
	}

	cache[*targetID] = name
	return name
}

// populateScopeNames resolves the transient project and environment names of a scoped PAT for API responses.
func (s *patService) populateScopeNames(ctx context.Context, pat *model.PersonalAccessToken) {
	if pat.ProjectID != nil {
		if proj, err := s.projectRepository.GetByID(ctx, *pat.ProjectID); err == nil {
			pat.ProjectName = proj.Name
		}
	}

	if pat.EnvironmentID != nil {
		if env, err := s.environmentRepository.GetByID(ctx, *pat.EnvironmentID); err == nil {
			pat.EnvironmentName = env.Name
		}
	}
}

// Revoke marks a user's active PAT as revoked in SQLite so it can no longer authenticate API requests.
func (s *patService) Revoke(ctx context.Context, userID id.ID, patID id.ID) error {
	name := ""
	if pat, err := s.patRepository.GetByID(ctx, patID); err == nil {
		name = pat.Name
	}

	now := s.nowFunc().UTC()
	if err := s.patRepository.RevokeByID(ctx, patID, userID, now); err != nil {
		return err
	}

	if name != "" {
		_ = s.audit.Record(ctx, types.ActionRevoked, types.ResourceToken, patID, name, nil)
	}

	return nil
}
