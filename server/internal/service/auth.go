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
	"github.com/yannick2009/yily/internal/id"
	"github.com/yannick2009/yily/internal/qr"
	"github.com/yannick2009/yily/internal/repository"
)

const (
	DefaultSessionTTL         = 24 * time.Hour  // DefaultSessionTTL is the default maximum session lifetime.
	DefaultSessionIdleTimeout = 2 * time.Hour   // DefaultSessionIdleTimeout is the default maximum session inactivity duration.
	MinPasswordLength         = 12              // MinPasswordLength is the minimum Super Admin password length.
	MaxPasswordLength         = 128             // MaxPasswordLength is the maximum Super Admin password length.
	TOTPMaxAttempts           = 5               // TOTPMaxAttempts is the number of consecutive invalid codes that triggers a lockout.
	TOTPLockoutDuration       = 5 * time.Minute // TOTPLockoutDuration is how long TOTP verification stays locked after too many failures.
)

var (
	ErrInvalidCredentials      = errors.New("invalid email or password")                                                                                      // ErrInvalidCredentials is returned when login email or password verification fails.
	ErrWeakPassword            = errors.New("password must be at least 12 characters long")                                                                   // ErrWeakPassword is returned when the provided password is shorter than MinPasswordLength.
	ErrPasswordTooLong         = errors.New("password must be at most 128 characters long")                                                                   // ErrPasswordTooLong is returned when the provided password is longer than MaxPasswordLength.
	ErrPasswordNotComplex      = errors.New("password must contain at least one uppercase letter, one lowercase letter, one digit and one special character") // ErrPasswordNotComplex is returned when the password fails the character-class complexity policy.
	ErrUnauthorized            = errors.New("unauthorized")                                                                                                   // ErrUnauthorized is returned when a request lacks a valid session or PAT.
	ErrSessionExpired          = errors.New("session expired")                                                                                                // ErrSessionExpired is returned when a session has exceeded its TTL or idle timeout.
	ErrSessionRevoked          = errors.New("session revoked")                                                                                                // ErrSessionRevoked is returned when a session has been explicitly logged out or rotated.
	ErrInvalidSetupToken       = errors.New("invalid setup token")                                                                                            // ErrInvalidSetupToken is returned when the first-run setup token is missing or wrong.
	ErrSetupAlreadyDone        = errors.New("setup has already been completed")                                                                               // ErrSetupAlreadyDone is returned when setup is attempted after the Super Admin already exists.
	ErrSetupTokenNotSet        = errors.New("setup token is not configured; set YILY_SETUP_TOKEN")                                                            // ErrSetupTokenNotSet is returned when setup is attempted on an uninitialized instance without a configured setup token.
	ErrInvalidRecoveryCode     = errors.New("invalid recovery code")                                                                                          // ErrInvalidRecoveryCode is returned when the provided recovery code does not match the stored hash.
	ErrRecoveryUnavailable     = errors.New("password recovery is not available")                                                                             // ErrRecoveryUnavailable is returned when the Super Admin has no recovery code configured.
	ErrTOTPRequired            = errors.New("a TOTP code is required")                                                                                        // ErrTOTPRequired is returned when the password is valid but TOTP is enabled and no code was supplied.
	ErrInvalidTOTPCode         = errors.New("invalid TOTP code")                                                                                              // ErrInvalidTOTPCode is returned when a code does not match the current or adjacent time steps, or was already used.
	ErrTOTPAlreadyEnabled      = errors.New("TOTP is already enabled")                                                                                        // ErrTOTPAlreadyEnabled is returned when enrollment starts for an account that already requires TOTP.
	ErrTOTPNotEnabled          = errors.New("TOTP is not enabled")                                                                                            // ErrTOTPNotEnabled is returned when TOTP is confirmed or disabled without an active or pending enrollment.
	ErrTOTPLocked              = errors.New("too many invalid TOTP codes; try again later")                                                                   // ErrTOTPLocked is returned while the account is locked after repeated invalid codes.
	ErrNoProfileUpdate         = errors.New("no profile fields to update")                                                                                    // ErrNoProfileUpdate is returned when a profile update carries neither a name nor an email.
	ErrCurrentPasswordRequired = errors.New("current password is required to change the email address")                                                       // ErrCurrentPasswordRequired is returned when an email change is attempted without the account password.
	ErrInvalidCurrentPassword  = errors.New("current password is incorrect")                                                                                  // ErrInvalidCurrentPassword is returned when the password supplied with an email change does not match.
)

// AuthService defines the interface for Super Admin authentication and server-side session management.
type AuthService interface {
	Setup(ctx context.Context, name, email, password, setupToken string) (*model.User, string, error)                           // Setup creates the initial Super Admin account, gated by the out-of-band setup token, and returns the one-time recovery code.
	Login(ctx context.Context, email, password, totpCode, existingRawToken string) (string, *model.Session, *model.User, error) // Login verifies the password and, when enabled, a TOTP code, rotates any prior session, and issues a new session.
	LoginWithTOTP(ctx context.Context, email, code, existingRawToken string) (string, *model.Session, *model.User, error)       // LoginWithTOTP authenticates with a TOTP code alone, for passwordless sign-in.
	ValidateSession(ctx context.Context, rawToken string) (*model.User, *model.Session, error)                                  // ValidateSession verifies a raw session token against its stored SHA-256 hash and loads the owning user.
	Logout(ctx context.Context, rawToken string) error                                                                          // Logout revokes a server-side session by hashing its raw token.
	UpdateProfile(ctx context.Context, userID id.ID, name, email, currentPassword *string) (*model.User, error)                 // UpdateProfile updates the Super Admin's display name and/or email and returns the refreshed user.
	IsInitialized(ctx context.Context) (bool, error)                                                                            // IsInitialized reports whether the Super Admin account has been created.
	Recover(ctx context.Context, recoveryCode, newPassword string) (string, error)                                              // Recover resets the Super Admin password using the recovery code, disables TOTP, and returns a new recovery code.
	SetupTOTP(ctx context.Context, userID id.ID) (*TOTPSetup, error)                                                            // SetupTOTP starts enrollment: it stores a new encrypted seed and returns it with a provisioning URI and QR image.
	ConfirmTOTP(ctx context.Context, userID id.ID, code string) error                                                           // ConfirmTOTP verifies a code against the pending seed and enables TOTP.
	DisableTOTP(ctx context.Context, userID id.ID, password string) error                                                       // DisableTOTP removes TOTP after re-verifying the account password.
}

// TOTPSetup is the one-time payload returned when TOTP enrollment starts.
type TOTPSetup struct {
	Secret string `json:"secret"`      // Secret is the Base32 seed for manual entry.
	URI    string `json:"otpauth_uri"` // URI is the otpauth:// provisioning URI.
	QR     string `json:"qr_png"`      // QR is a PNG data URI of the provisioning URI; empty when rendering failed.
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
	masterKey         []byte                       // masterKey encrypts the TOTP seed at rest.

	dummyOnce sync.Once // dummyOnce ensures dummyHash is computed at most once.
	dummyHash string    // dummyHash is a pre-computed Argon2id PHC hash used when login email is not found to equalize response time and prevent timing-based email enumeration.
}

// NewAuthService creates a new AuthService with default session TTL and inactivity settings.
func NewAuthService(userRepository repository.UserRepository, sessionRepository repository.SessionRepository, setupToken string, masterKey []byte) AuthService {
	return NewAuthServiceWithConfig(userRepository, sessionRepository, setupToken, masterKey, AuthConfig{})
}

// NewAuthServiceWithConfig creates a new AuthService with custom session TTL, idle timeout, and clock function.
func NewAuthServiceWithConfig(userRepository repository.UserRepository, sessionRepository repository.SessionRepository, setupToken string, masterKey []byte, cfg AuthConfig) AuthService {
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
		masterKey:         masterKey,
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

// Login verifies the Super Admin's Argon2id password and, when TOTP is
// enabled, a code from the authenticator app. It then revokes any previous
// session to prevent session fixation and creates a new SHA-256 hashed session.
func (s *authService) Login(ctx context.Context, email, password, totpCode, existingRawToken string) (string, *model.Session, *model.User, error) {
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

	if user.TOTPEnabled {
		if err := s.verifyTOTP(ctx, user, totpCode); err != nil {
			return "", nil, nil, err
		}
	}

	return s.issueSession(ctx, user, existingRawToken)
}

// LoginWithTOTP authenticates the Super Admin with a TOTP code alone, for
// passwordless sign-in. Unknown emails, accounts without TOTP, and wrong codes
// all fail with ErrInvalidCredentials so the endpoint cannot be used to
// enumerate accounts.
func (s *authService) LoginWithTOTP(ctx context.Context, email, code, existingRawToken string) (string, *model.Session, *model.User, error) {
	normalizedEmail := normalizeEmail(email)
	if normalizedEmail == "" || crypto.NormalizeTOTPCode(code) == "" {
		return "", nil, nil, ErrInvalidCredentials
	}

	user, err := s.userRepository.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			s.runDummyTOTP(code)
			return "", nil, nil, ErrInvalidCredentials
		}
		return "", nil, nil, err
	}

	if !user.TOTPEnabled || user.TOTPSecretEncrypted == "" {
		s.runDummyTOTP(code)
		return "", nil, nil, ErrInvalidCredentials
	}

	if err := s.verifyTOTP(ctx, user, code); err != nil {
		// Keep the public endpoint non-enumerable: a wrong code and a locked
		// account must be indistinguishable from an unknown email.
		if errors.Is(err, ErrInvalidTOTPCode) || errors.Is(err, ErrTOTPLocked) {
			return "", nil, nil, ErrInvalidCredentials
		}
		return "", nil, nil, err
	}

	return s.issueSession(ctx, user, existingRawToken)
}

// SetupTOTP starts enrollment for an account that does not have TOTP yet. The
// seed is stored encrypted and inert until ConfirmTOTP proves the authenticator
// app produces valid codes, so an abandoned setup cannot lock the account out.
func (s *authService) SetupTOTP(ctx context.Context, userID id.ID) (*TOTPSetup, error) {
	user, err := s.userRepository.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.TOTPEnabled {
		return nil, ErrTOTPAlreadyEnabled
	}

	secret, err := crypto.GenerateTOTPSecret()
	if err != nil {
		return nil, err
	}

	encrypted, err := crypto.EncryptTOTPSecret(secret, s.masterKey, user.ID)
	if err != nil {
		return nil, err
	}

	if err := s.userRepository.SetTOTPSecret(ctx, user.ID, encrypted); err != nil {
		return nil, err
	}

	uri := crypto.TOTPURI(crypto.TOTPIssuer, user.Email, secret)

	// The QR image is a convenience: enrollment still works without it because
	// the response always carries the secret and URI for manual entry.
	qrPNG, _ := qr.DataURI(uri, 6)

	return &TOTPSetup{Secret: secret, URI: uri, QR: qrPNG}, nil
}

// ConfirmTOTP verifies a code against the pending seed and enables TOTP. The
// accepted time step is recorded so the enrollment code cannot be replayed.
func (s *authService) ConfirmTOTP(ctx context.Context, userID id.ID, code string) error {
	user, err := s.userRepository.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.TOTPEnabled {
		return ErrTOTPAlreadyEnabled
	}
	if user.TOTPSecretEncrypted == "" {
		return ErrTOTPNotEnabled
	}

	secret, err := crypto.DecryptTOTPSecret(user.TOTPSecretEncrypted, s.masterKey, user.ID)
	if err != nil {
		return err
	}

	step, ok := crypto.ValidateTOTP(secret, code, s.nowFunc().UTC())
	if !ok {
		return ErrInvalidTOTPCode
	}

	return s.userRepository.EnableTOTP(ctx, user.ID, step)
}

// DisableTOTP turns TOTP off after re-verifying the account password. The
// recovery code path disables TOTP as well, so a lost authenticator app is
// never a permanent lockout.
func (s *authService) DisableTOTP(ctx context.Context, userID id.ID, password string) error {
	user, err := s.userRepository.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !user.TOTPEnabled {
		return ErrTOTPNotEnabled
	}

	valid, err := crypto.VerifyPassword(password, user.PasswordHash)
	if err != nil || !valid {
		return ErrInvalidCredentials
	}

	return s.userRepository.DisableTOTP(ctx, user.ID)
}

// verifyTOTP checks a code against the user's seed, enforces the lockout
// window, and atomically consumes the matched time step to reject replays.
func (s *authService) verifyTOTP(ctx context.Context, user *model.User, code string) error {
	now := s.nowFunc().UTC()
	if user.TOTPLockedUntil != nil && now.Before(*user.TOTPLockedUntil) {
		return ErrTOTPLocked
	}

	normalized := crypto.NormalizeTOTPCode(code)
	if normalized == "" {
		// The password was valid: ask the UI for the missing second factor. A
		// missing code is not a failed attempt and must not count toward lockout.
		return ErrTOTPRequired
	}

	secret, err := crypto.DecryptTOTPSecret(user.TOTPSecretEncrypted, s.masterKey, user.ID)
	if err != nil {
		return err
	}

	step, ok := crypto.ValidateTOTP(secret, normalized, now)
	if !ok {
		if err := s.userRepository.RecordTOTPFailure(ctx, user.ID, TOTPMaxAttempts, now.Add(TOTPLockoutDuration)); err != nil {
			return err
		}
		return ErrInvalidTOTPCode
	}

	consumed, err := s.userRepository.ConsumeTOTPStep(ctx, user.ID, step)
	if err != nil {
		return err
	}
	if !consumed {
		return ErrInvalidTOTPCode
	}

	return nil
}

// dummyTOTPSecret is a random per-process seed used to spend the same HMAC work
// when an email has no TOTP-enabled account, keeping passwordless login timing
// uniform. It is never accepted for any account.
var dummyTOTPSecret, _ = crypto.GenerateTOTPSecret()

// runDummyTOTP performs a throwaway validation to equalize passwordless login timing.
func (s *authService) runDummyTOTP(code string) {
	_, _ = crypto.ValidateTOTP(dummyTOTPSecret, code, s.nowFunc().UTC())
}

// issueSession revokes any session supplied with the login request (to prevent
// session fixation) and creates a new session with a fresh CSRF token.
func (s *authService) issueSession(ctx context.Context, user *model.User, existingRawToken string) (string, *model.Session, *model.User, error) {
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

// UpdateProfile updates the Super Admin's display name and/or email. Nil
// fields are left unchanged. The name is trimmed and validated; the email is
// normalized to lowercase and validated. Changing the email address requires
// the account's current password. It returns the refreshed user.
func (s *authService) UpdateProfile(ctx context.Context, userID id.ID, name, email, currentPassword *string) (*model.User, error) {
	if name == nil && email == nil {
		return nil, ErrNoProfileUpdate
	}

	var newName, newEmail string
	if name != nil {
		newName = strings.TrimSpace(*name)
		if newName == "" {
			return nil, ErrNameRequired
		}
		if err := validateName(newName); err != nil {
			return nil, err
		}
	}
	if email != nil {
		newEmail = normalizeEmail(*email)
		if err := validateEmail(newEmail); err != nil {
			return nil, err
		}
	}

	user, err := s.userRepository.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// The email address is the login identifier: re-authenticate before
	// changing it so a stolen session cannot silently lock the owner out.
	if email != nil && newEmail != user.Email {
		password := ""
		if currentPassword != nil {
			password = *currentPassword
		}
		if password == "" {
			return nil, ErrCurrentPasswordRequired
		}
		valid, err := crypto.VerifyPassword(password, user.PasswordHash)
		if err != nil || !valid {
			return nil, ErrInvalidCurrentPassword
		}
	}

	if name != nil {
		if err := s.userRepository.SetName(ctx, userID, newName); err != nil {
			return nil, err
		}
	}
	if email != nil {
		if err := s.userRepository.SetEmail(ctx, userID, newEmail); err != nil {
			return nil, err
		}
	}

	return s.userRepository.GetByID(ctx, userID)
}

// normalizeEmail trims leading/trailing whitespace and converts an email address to lowercase.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Recover resets the Super Admin password using the one-time recovery code. On
// success, in a single transaction, the password and recovery code are replaced,
// TOTP is disabled, and every active session and unexpired PAT is revoked; the
// fresh recovery code is returned. On failure nothing changes and the old
// recovery code stays valid.
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
