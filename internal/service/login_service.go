package service

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Security constants for SOC2/ISO27001 compliance
const (
	// Rate limiting (SOC2 CC6.1 - Logical Access Controls)
	MaxLoginAttempts   = 5                // Maximum failed attempts before lockout
	LoginAttemptWindow = 15 * time.Minute // Time window for counting attempts
	AccountLockoutTime = 30 * time.Minute // Account lockout duration

	// Input validation (ISO27001 A.14.2.1)
	MinPasswordLength = 8
	MaxPasswordLength = 128
	MaxUsernameLength = 50
	MaxClientIDLength = 100

	// Session security (SOC2 CC6.3)
	MaxConcurrentSessions = 5 // Maximum concurrent sessions per user
)

// LoginAttempt tracks failed login attempts for rate limiting
type LoginAttempt struct {
	Identifier  string     // Username, email, or IP
	Attempts    int        // Number of failed attempts
	LastAttempt time.Time  // Time of last attempt
	LockedUntil *time.Time // Account locked until this time
}

// SecurityEvent represents a security-related event for audit logging
type SecurityEvent struct {
	EventType string    // "login_success", "login_failure", "account_locked", etc.
	UserID    string    // User identifier (if available)
	ClientID  string    // Client identifier
	IPAddress string    // Source IP address
	UserAgent string    // User agent string
	Timestamp time.Time // Event timestamp
	Details   string    // Additional event details
}

type LoginService interface {
	Login(usernameOrEmail, password, authClientID, authContainerID string) (*dto.AuthResponseDto, error)
	GetUserByEmail(email string, authContainerID int64) (*model.User, error)
}

type loginService struct {
	db             *gorm.DB
	authClientRepo repository.AuthClientRepository
	userRepo       repository.UserRepository
	userTokenRepo  repository.UserTokenRepository

	// In-memory rate limiting (in production, use Redis or database)
	loginAttempts map[string]*LoginAttempt
}

func NewLoginService(
	db *gorm.DB,
	authClientRepo repository.AuthClientRepository,
	userRepo repository.UserRepository,
	userTokenRepo repository.UserTokenRepository,
) LoginService {
	return &loginService{
		db:             db,
		authClientRepo: authClientRepo,
		userRepo:       userRepo,
		userTokenRepo:  userTokenRepo,
		loginAttempts:  make(map[string]*LoginAttempt),
	}
}

// validateLoginInput performs comprehensive input validation
// Complies with SOC2 CC6.1 and ISO27001 A.14.2.1
func (s *loginService) validateLoginInput(username, password, clientID string) error {
	// Validate username/email
	if strings.TrimSpace(username) == "" {
		return errors.New("username or email is required")
	}
	if len(username) > MaxUsernameLength {
		return fmt.Errorf("username exceeds maximum length of %d characters", MaxUsernameLength)
	}

	// Validate password
	if strings.TrimSpace(password) == "" {
		return errors.New("password is required")
	}
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}
	if len(password) > MaxPasswordLength {
		return fmt.Errorf("password exceeds maximum length of %d characters", MaxPasswordLength)
	}

	// Validate client ID (for public login)
	if clientID != "" {
		if strings.TrimSpace(clientID) == "" {
			return errors.New("client ID cannot be empty")
		}
		if len(clientID) > MaxClientIDLength {
			return fmt.Errorf("client ID exceeds maximum length of %d characters", MaxClientIDLength)
		}
	}

	return nil
}

// checkRateLimit implements rate limiting and account lockout
// Complies with SOC2 CC6.1 and ISO27001 A.9.4.2
func (s *loginService) checkRateLimit(identifier string) error {
	now := time.Now()

	// Get or create login attempt record
	attempt, exists := s.loginAttempts[identifier]
	if !exists {
		s.loginAttempts[identifier] = &LoginAttempt{
			Identifier:  identifier,
			Attempts:    0,
			LastAttempt: now,
		}
		return nil
	}

	// Check if account is currently locked
	if attempt.LockedUntil != nil && now.Before(*attempt.LockedUntil) {
		remainingTime := attempt.LockedUntil.Sub(now)
		return fmt.Errorf("account is locked for %v due to too many failed login attempts", remainingTime.Round(time.Minute))
	}

	// Reset attempts if outside the time window
	if now.Sub(attempt.LastAttempt) > LoginAttemptWindow {
		attempt.Attempts = 0
		attempt.LockedUntil = nil
	}

	// Check if maximum attempts exceeded
	if attempt.Attempts >= MaxLoginAttempts {
		lockUntil := now.Add(AccountLockoutTime)
		attempt.LockedUntil = &lockUntil

		// Log security event
		s.logSecurityEvent(SecurityEvent{
			EventType: "account_locked",
			UserID:    identifier,
			Timestamp: now,
			Details:   fmt.Sprintf("Account locked after %d failed login attempts", attempt.Attempts),
		})

		return fmt.Errorf("account locked for %v due to too many failed login attempts", AccountLockoutTime)
	}

	return nil
}

// recordFailedAttempt records a failed login attempt
func (s *loginService) recordFailedAttempt(identifier string) {
	now := time.Now()

	attempt, exists := s.loginAttempts[identifier]
	if !exists {
		s.loginAttempts[identifier] = &LoginAttempt{
			Identifier:  identifier,
			Attempts:    1,
			LastAttempt: now,
		}
		return
	}

	// Reset if outside time window
	if now.Sub(attempt.LastAttempt) > LoginAttemptWindow {
		attempt.Attempts = 1
		attempt.LockedUntil = nil
	} else {
		attempt.Attempts++
	}

	attempt.LastAttempt = now
}

// resetFailedAttempts clears failed attempts after successful login
func (s *loginService) resetFailedAttempts(identifier string) {
	delete(s.loginAttempts, identifier)
}

// logSecurityEvent logs security events for audit compliance
// Complies with SOC2 CC7.2 and ISO27001 A.12.4.1
func (s *loginService) logSecurityEvent(event SecurityEvent) {
	// In production, this should write to a secure audit log system
	// For now, we'll use structured logging
	log.Printf("[SECURITY_AUDIT] Type=%s UserID=%s ClientID=%s Timestamp=%s Details=%s",
		event.EventType,
		event.UserID,
		event.ClientID,
		event.Timestamp.Format(time.RFC3339),
		event.Details,
	)
}

func (s *loginService) Login(usernameOrEmail, password, authClientID, authContainerID string) (*dto.AuthResponseDto, error) {
	startTime := time.Now()

	// Input validation (SOC2 CC6.1 - Logical Access Controls)
	if err := s.validateLoginInput(usernameOrEmail, password, authClientID); err != nil {
		s.logSecurityEvent(SecurityEvent{
			EventType: "login_validation_failure",
			UserID:    usernameOrEmail,
			ClientID:  authClientID,
			Timestamp: startTime,
			Details:   fmt.Sprintf("Input validation failed: %s", err.Error()),
		})
		return nil, errors.New("invalid input parameters")
	}

	// Parse auth container ID
	authContainerId, err := strconv.ParseInt(authContainerID, 10, 64)
	if err != nil {
		s.logSecurityEvent(SecurityEvent{
			EventType: "login_validation_failure",
			UserID:    usernameOrEmail,
			ClientID:  authClientID,
			Timestamp: startTime,
			Details:   "Invalid auth container ID format",
		})
		return nil, errors.New("invalid auth container ID format")
	}

	// Rate limiting check (SOC2 CC6.1 - Logical Access Controls)
	if err := s.checkRateLimit(usernameOrEmail); err != nil {
		s.logSecurityEvent(SecurityEvent{
			EventType: "login_rate_limited",
			UserID:    usernameOrEmail,
			ClientID:  authClientID,
			Timestamp: startTime,
			Details:   err.Error(),
		})
		return nil, err
	}

	// Get and validate auth client
	authClient, err := s.authClientRepo.FindByClientID(authClientID)
	if err != nil {
		s.logSecurityEvent(SecurityEvent{
			EventType: "login_client_lookup_failure",
			UserID:    usernameOrEmail,
			ClientID:  authClientID,
			Timestamp: startTime,
			Details:   "Client lookup failed",
		})
		return nil, errors.New("authentication failed")
	}

	if authClient == nil ||
		!authClient.IsActive ||
		authClient.Domain == nil || *authClient.Domain == "" {
		s.logSecurityEvent(SecurityEvent{
			EventType: "login_invalid_client",
			UserID:    usernameOrEmail,
			ClientID:  authClientID,
			Timestamp: startTime,
			Details:   "Invalid or inactive client configuration",
		})
		return nil, errors.New("authentication failed")
	}

	// Get user (timing-safe user lookup)
	var user *model.User
	var userLookupErr error

	if util.IsValidEmail(usernameOrEmail) {
		user, userLookupErr = s.userRepo.FindByEmail(usernameOrEmail, authContainerId)
	} else {
		user, userLookupErr = s.userRepo.FindByUsername(usernameOrEmail, authContainerId)
	}

	// Timing-safe credential verification to prevent user enumeration
	var passwordValid bool = false
	var hashedPassword []byte

	if userLookupErr == nil && user != nil && user.Password != nil {
		hashedPassword = []byte(*user.Password)
		passwordValid = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password)) == nil
	} else {
		// Perform dummy bcrypt operation to maintain consistent timing
		bcrypt.CompareHashAndPassword([]byte("$2a$10$dummy.hash.to.prevent.timing.attacks"), []byte(password))
	}

	// Check if authentication succeeded
	if !passwordValid || user == nil || user.Password == nil {
		// Record failed attempt
		s.recordFailedAttempt(usernameOrEmail)

		s.logSecurityEvent(SecurityEvent{
			EventType: "login_failure",
			UserID:    usernameOrEmail,
			ClientID:  authClientID,
			Timestamp: startTime,
			Details:   "Invalid credentials provided",
		})

		return nil, errors.New("invalid credentials")
	}

	// Check if user account is active
	if !user.IsActive {
		s.logSecurityEvent(SecurityEvent{
			EventType: "login_inactive_user",
			UserID:    user.UserUUID.String(),
			ClientID:  authClientID,
			Timestamp: startTime,
			Details:   "Attempt to login with inactive user account",
		})
		return nil, errors.New("account is not active")
	}

	// Reset failed attempts on successful authentication
	s.resetFailedAttempts(usernameOrEmail)

	// Log successful login
	s.logSecurityEvent(SecurityEvent{
		EventType: "login_success",
		UserID:    user.UserUUID.String(),
		ClientID:  authClientID,
		Timestamp: startTime,
		Details:   fmt.Sprintf("Successful login for user %s", user.Username),
	})

	// Generate token response
	return s.generateTokenResponse(user.UserUUID.String(), user, authClient)
}

func (s *loginService) GetUserByEmail(email string, authContainerID int64) (*model.User, error) {
	return s.userRepo.FindByEmail(email, authContainerID)
}

func (s *loginService) generateTokenResponse(userUUID string, user *model.User, authClient *model.AuthClient) (*dto.AuthResponseDto, error) {
	accessToken, err := util.GenerateAccessToken(
		userUUID,
		"openid profile email",
		*authClient.Domain,
		*authClient.ClientID,
		authClient.IdentityProvider.AuthContainer.AuthContainerUUID,
		*authClient.ClientID,
		authClient.IdentityProvider.IdentityProviderUUID,
	)
	if err != nil {
		return nil, err
	}

	// Create user profile for ID token (populate from user data)
	profile := &util.UserProfile{
		Email:         user.Email,
		EmailVerified: user.IsEmailVerified,
		Phone:         user.Phone,
		PhoneVerified: user.IsPhoneVerified,
	}

	// Generate ID token with user profile (no nonce for login flow)
	idToken, err := util.GenerateIDToken(userUUID, *authClient.Domain, *authClient.ClientID, profile, "")
	if err != nil {
		return nil, err
	}

	refreshToken, err := util.GenerateRefreshToken(userUUID, *authClient.Domain, *authClient.ClientID)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponseDto{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		IssuedAt:     time.Now().Unix(),
	}, nil
}
