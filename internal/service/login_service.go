package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type LoginService interface {
	Login(usernameOrEmail, password, clientID, providerID string) (*dto.LoginResponseDto, error)
	GetUserByEmail(email string, tenantID int64) (*model.User, error)
}

type loginService struct {
	db                   *gorm.DB
	authClientRepo       repository.AuthClientRepository
	userRepo             repository.UserRepository
	userTokenRepo        repository.UserTokenRepository
	identityProviderRepo repository.IdentityProviderRepository
}

func NewLoginService(
	db *gorm.DB,
	authClientRepo repository.AuthClientRepository,
	userRepo repository.UserRepository,
	userTokenRepo repository.UserTokenRepository,
	identityProviderRepo repository.IdentityProviderRepository,
) LoginService {
	return &loginService{
		db:                   db,
		authClientRepo:       authClientRepo,
		userRepo:             userRepo,
		userTokenRepo:        userTokenRepo,
		identityProviderRepo: identityProviderRepo,
	}
}

func (s *loginService) Login(usernameOrEmail, password, clientID, providerID string) (*dto.LoginResponseDto, error) {
	startTime := time.Now()

	// Input validation is now handled at the DTO/handler level

	// Rate limiting check (SOC2 CC6.1 - Logical Access Controls)
	if err := util.CheckRateLimit(usernameOrEmail); err != nil {
		util.LogSecurityEvent(util.SecurityEvent{
			EventType: "login_rate_limited",
			UserID:    usernameOrEmail,
			ClientID:  clientID,
			Timestamp: startTime,
			Details:   err.Error(),
		})
		return nil, err
	}

	var user *model.User
	var authClient *model.AuthClient
	var userLookupErr error

	// All database operations in transaction (read-only for consistency)
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txIdentityProviderRepo := s.identityProviderRepo.WithTx(tx)

		// Look up identity provider by identifier to get auth container
		identityProvider, txErr := txIdentityProviderRepo.FindByIdentifier(providerID)
		if txErr != nil {
			util.LogSecurityEvent(util.SecurityEvent{
				EventType: "login_validation_failure",
				UserID:    usernameOrEmail,
				ClientID:  clientID,
				Timestamp: startTime,
				Details:   "Identity provider lookup failed",
			})
			return errors.New("authentication failed")
		}

		if identityProvider == nil {
			util.LogSecurityEvent(util.SecurityEvent{
				EventType: "login_validation_failure",
				UserID:    usernameOrEmail,
				ClientID:  clientID,
				Timestamp: startTime,
				Details:   "Identity provider not found",
			})
			return errors.New("authentication failed")
		}

		tenantId := identityProvider.TenantID

		// Get and validate auth client with proper relationship preloading
		authClient, txErr = txAuthClientRepo.FindByClientIDAndIdentityProvider(clientID, providerID)
		if txErr != nil {
			util.LogSecurityEvent(util.SecurityEvent{
				EventType: "login_client_lookup_failure",
				UserID:    usernameOrEmail,
				ClientID:  clientID,
				Timestamp: startTime,
				Details:   "Client lookup failed",
			})
			return errors.New("authentication failed")
		}

		if authClient == nil ||
			!authClient.IsActive ||
			authClient.Domain == nil || *authClient.Domain == "" {
			util.LogSecurityEvent(util.SecurityEvent{
				EventType: "login_invalid_client",
				UserID:    usernameOrEmail,
				ClientID:  clientID,
				Timestamp: startTime,
				Details:   "Invalid or inactive client configuration",
			})
			return errors.New("authentication failed")
		}

		// Get user by username (timing-safe user lookup)
		user, userLookupErr = txUserRepo.FindByUsername(usernameOrEmail, tenantId)

		return nil // No error, continue with authentication logic outside transaction
	})

	if err != nil {
		return nil, err
	}

	// Timing-safe credential verification to prevent user enumeration
	var passwordValid bool = false
	var hashedPassword []byte

	if userLookupErr == nil && user != nil && user.Password != nil {
		hashedPassword = []byte(*user.Password)
		passwordValid = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password)) == nil
	} else {
		// Perform dummy bcrypt operation to maintain consistent timing
		bcrypt.CompareHashAndPassword(util.GenerateDummyBcryptHash(), []byte(password))
	}

	// Check if authentication succeeded
	if !passwordValid || user == nil || user.Password == nil {
		// Record failed attempt
		util.RecordFailedAttempt(usernameOrEmail)

		util.LogSecurityEvent(util.SecurityEvent{
			EventType: "login_failure",
			UserID:    usernameOrEmail,
			ClientID:  clientID,
			Timestamp: startTime,
			Details:   "Invalid credentials provided",
		})

		return nil, errors.New("invalid credentials")
	}

	// Check if user account is active
	if !user.IsActive {
		util.LogSecurityEvent(util.SecurityEvent{
			EventType: "login_inactive_user",
			UserID:    user.UserUUID.String(),
			ClientID:  clientID,
			Timestamp: startTime,
			Details:   "Attempt to login with inactive user account",
		})
		return nil, errors.New("account is not active")
	}

	// Reset failed attempts on successful authentication
	util.ResetFailedAttempts(usernameOrEmail)

	// Log successful login
	util.LogSecurityEvent(util.SecurityEvent{
		EventType: "login_success",
		UserID:    user.UserUUID.String(),
		ClientID:  clientID,
		Timestamp: startTime,
		Details:   fmt.Sprintf("Successful login for user %s", user.Username),
	})

	// Generate token response
	return s.generateTokenResponse(user.UserUUID.String(), user, authClient)
}

func (s *loginService) GetUserByEmail(email string, tenantID int64) (*model.User, error) {
	return s.userRepo.FindByEmail(email, tenantID)
}

func (s *loginService) generateTokenResponse(userUUID string, user *model.User, authClient *model.AuthClient) (*dto.LoginResponseDto, error) {
	accessToken, err := util.GenerateAccessToken(
		userUUID,
		"openid profile email",
		*authClient.Domain,
		*authClient.ClientID,
		authClient.IdentityProvider.Tenant.TenantUUID,
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

	return &dto.LoginResponseDto{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		IssuedAt:     time.Now().Unix(),
	}, nil
}
