package service

import (
	"context"
	"fmt"
	"time"

	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/jwt"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/security"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type LoginService interface {
	LoginPublic(ctx context.Context, usernameOrEmail, password, clientID, providerID string) (*dto.LoginResponseDTO, error)
	Login(ctx context.Context, usernameOrEmail, password string, clientID, providerID *string) (*dto.LoginResponseDTO, error)
	GetUserByEmail(ctx context.Context, email string, tenantID int64) (*model.User, error)
}

type loginService struct {
	db                   *gorm.DB
	clientRepo           repository.ClientRepository
	userRepo             repository.UserRepository
	userTokenRepo        repository.UserTokenRepository
	userIdentityRepo     repository.UserIdentityRepository
	identityProviderRepo repository.IdentityProviderRepository
}

func NewLoginService(
	db *gorm.DB,
	clientRepo repository.ClientRepository,
	userRepo repository.UserRepository,
	userTokenRepo repository.UserTokenRepository,
	userIdentityRepo repository.UserIdentityRepository,
	identityProviderRepo repository.IdentityProviderRepository,
) LoginService {
	return &loginService{
		db:                   db,
		clientRepo:           clientRepo,
		userRepo:             userRepo,
		userTokenRepo:        userTokenRepo,
		userIdentityRepo:     userIdentityRepo,
		identityProviderRepo: identityProviderRepo,
	}
}

// Function variables to allow stubbing in tests.
var generateIDTokenFn = jwt.GenerateIDToken
var generateRefreshTokenFn = jwt.GenerateRefreshToken

// LoginPublic authenticates users for public-facing applications.
// Requires clientID and providerID to identify the auth client.
// Used by external applications on port 8081.
func (s *loginService) LoginPublic(ctx context.Context, usernameOrEmail, password, clientID, providerID string) (result *dto.LoginResponseDTO, err error) {
	_, span := otel.Tracer("service").Start(ctx, "login.public")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "login failed")
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
	}()
	startTime := time.Now()

	// Input validation is now handled at the DTO/handler level

	// Rate limiting check (SOC2 CC6.1 - Logical Access Controls)
	if err := security.CheckRateLimit(usernameOrEmail); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_rate_limited",
			UserID:    usernameOrEmail,
			ClientID:  clientID,
			Timestamp: startTime,
			Details:   err.Error(),
		})
		return nil, err
	}

	var user *model.User
	var client *model.Client
	var userLookupErr error
	var userIdentitySub string

	// All database operations in transaction (read-only for consistency)
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txIdentityProviderRepo := s.identityProviderRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)

		// Look up identity provider by identifier to get auth container
		identityProvider, txErr := txIdentityProviderRepo.FindByIdentifier(providerID)
		if txErr != nil {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "login_validation_failure",
				UserID:    usernameOrEmail,
				ClientID:  clientID,
				Timestamp: startTime,
				Details:   "Identity provider lookup failed",
			})
			return apperror.NewUnauthorized("authentication failed")
		}

		if identityProvider == nil {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "login_validation_failure",
				UserID:    usernameOrEmail,
				ClientID:  clientID,
				Timestamp: startTime,
				Details:   "Identity provider not found",
			})
			return apperror.NewUnauthorized("authentication failed")
		}

		// Get and validate auth client with proper relationship preloading
		client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(clientID, providerID)
		if txErr != nil {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "login_client_lookup_failure",
				UserID:    usernameOrEmail,
				ClientID:  clientID,
				Timestamp: startTime,
				Details:   "Client lookup failed",
			})
			return apperror.NewUnauthorized("authentication failed")
		}

		if client == nil ||
			client.Status != model.StatusActive ||
			client.Domain == nil || *client.Domain == "" {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "login_invalid_client",
				UserID:    usernameOrEmail,
				ClientID:  clientID,
				Timestamp: startTime,
				Details:   "Invalid or inactive client configuration",
			})
			return apperror.NewUnauthorized("authentication failed")
		}

		// Get user by username (timing-safe user lookup)
		user, userLookupErr = txUserRepo.FindByUsername(usernameOrEmail)

		// Fetch user identity to get the Sub claim
		if userLookupErr == nil && user != nil {
			userIdentity, txErr := txUserIdentityRepo.FindByUserIDAndClientID(user.UserID, client.ClientID)
			if txErr == nil && userIdentity != nil {
				userIdentitySub = userIdentity.Sub
			}
		}

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
		bcrypt.CompareHashAndPassword(security.GetDummyBcryptHash(), []byte(password)) //nolint:errcheck // intentional timing dummy; error is irrelevant
	}

	// Check if authentication succeeded
	if !passwordValid || user == nil || user.Password == nil {
		// Record failed attempt
		security.RecordFailedAttempt(usernameOrEmail)

		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_failure",
			UserID:    usernameOrEmail,
			ClientID:  clientID,
			Timestamp: startTime,
			Details:   "Invalid credentials provided",
		})

		return nil, apperror.NewUnauthorized("invalid credentials")
	}

	// Check if user account is active
	if user.Status != model.StatusActive {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_inactive_user",
			UserID:    user.UserUUID.String(),
			ClientID:  clientID,
			Timestamp: startTime,
			Details:   "Attempt to login with inactive user account",
		})
		return nil, apperror.NewUnauthorized("account is not active")
	}

	// Reset failed attempts on successful authentication
	security.ResetFailedAttempts(usernameOrEmail)

	// Log successful login
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "login_success",
		UserID:    user.UserUUID.String(),
		ClientID:  clientID,
		Timestamp: startTime,
		Details:   fmt.Sprintf("Successful login for user %s", user.Username),
	})

	// Generate token response
	return s.generateTokenResponse(userIdentitySub, user, client)
}

// Login authenticates users for internal applications.
// If clientID and providerID are provided, uses the specified auth client.
// If not provided, uses the default auth client.
// Used by internal applications on port 8080.
func (s *loginService) Login(ctx context.Context, usernameOrEmail, password string, clientID, providerID *string) (result *dto.LoginResponseDTO, err error) {
	_, span := otel.Tracer("service").Start(ctx, "login.internal")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "login failed")
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
	}()
	startTime := time.Now()

	// Rate limiting check (SOC2 CC6.1 - Logical Access Controls)
	if err := security.CheckRateLimit(usernameOrEmail); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_rate_limited",
			UserID:    usernameOrEmail,
			ClientID:  "internal",
			Timestamp: startTime,
			Details:   err.Error(),
		})
		return nil, err
	}

	var user *model.User
	var client *model.Client
	var userIdentitySub string

	// All database operations in transaction for consistency
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)

		// Get auth client - either by client_id and provider_id or default
		var txErr error
		if clientID != nil && providerID != nil {
			// Get auth client by client_id and identity provider identifier
			client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(*clientID, *providerID)
			if txErr != nil {
				security.LogSecurityEvent(security.SecurityEvent{
					EventType: "login_client_lookup_failure",
					UserID:    usernameOrEmail,
					ClientID:  *clientID,
					Timestamp: startTime,
					Details:   "Client lookup by client_id and provider_id failed",
				})
				return apperror.NewUnauthorized("authentication failed")
			}
		} else {
			// Get system client for no-client_id authentication (always the system tenant)
			client, txErr = txClientRepo.FindSystem()
			if txErr != nil {
				security.LogSecurityEvent(security.SecurityEvent{
					EventType: "login_client_lookup_failure",
					UserID:    usernameOrEmail,
					ClientID:  "internal",
					Timestamp: startTime,
					Details:   "Default client lookup failed",
				})
				return apperror.NewUnauthorized("authentication failed")
			}
		}

		if client == nil ||
			client.Status != model.StatusActive ||
			client.Domain == nil || *client.Domain == "" {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "login_invalid_client",
				UserID:    usernameOrEmail,
				ClientID:  "internal",
				Timestamp: startTime,
				Details:   "Invalid or inactive default client configuration",
			})
			return apperror.NewUnauthorized("authentication failed")
		}

		// Get user by username (timing-safe user lookup)
		user, _ = txUserRepo.FindByUsername(usernameOrEmail)
		// Note: We don't return error here to maintain timing-safe behavior

		// Fetch user identity to get the Sub claim
		if user != nil {
			userIdentity, txErr := txUserIdentityRepo.FindByUserIDAndClientID(user.UserID, client.ClientID)
			if txErr == nil && userIdentity != nil {
				userIdentitySub = userIdentity.Sub
			}
		}

		return nil // No error, continue with authentication logic outside transaction
	})

	if err != nil {
		return nil, err
	}

	// Timing-safe password comparison (always compare even if user not found)
	var passwordValid bool
	if user != nil && user.Password != nil {
		err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password))
		passwordValid = (err == nil)
	} else {
		// Use a properly pre-computed dummy hash so the timing profile is
		// identical whether the user exists or not. A literal/invalid hash
		// string would return immediately without doing real bcrypt work.
		bcrypt.CompareHashAndPassword(security.GetDummyBcryptHash(), []byte(password)) //nolint:errcheck
		passwordValid = false
	}

	// Check if authentication succeeded
	if !passwordValid || user == nil || user.Password == nil {
		// Record failed attempt
		security.RecordFailedAttempt(usernameOrEmail)

		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_failure",
			UserID:    usernameOrEmail,
			ClientID:  "internal",
			Timestamp: startTime,
			Details:   "Invalid credentials provided",
		})

		return nil, apperror.NewUnauthorized("invalid credentials")
	}

	// Check if user account is active
	if user.Status != model.StatusActive {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_inactive_user",
			UserID:    user.UserUUID.String(),
			ClientID:  "internal",
			Timestamp: startTime,
			Details:   "Attempt to login with inactive user account",
		})
		return nil, apperror.NewUnauthorized("account is not active")
	}

	// Reset failed attempts on successful authentication
	security.ResetFailedAttempts(usernameOrEmail)

	// Log successful login
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "login_success",
		UserID:    user.UserUUID.String(),
		ClientID:  "internal",
		Timestamp: startTime,
		Details:   fmt.Sprintf("Successful internal login for user %s", user.Username),
	})

	// Generate token response
	return s.generateTokenResponse(userIdentitySub, user, client)
}

// GetUserByEmail looks up a user by email, scoped to the given tenant when
// tenantID > 0. Falls back to a global lookup only when no tenant is specified.
func (s *loginService) GetUserByEmail(ctx context.Context, email string, tenantID int64) (*model.User, error) {
	_, span := otel.Tracer("service").Start(ctx, "login.getUserByEmail")
	defer span.End()
	var user *model.User
	var err error
	if tenantID > 0 {
		user, err = s.userRepo.FindByEmailAndTenantID(email, tenantID)
	} else {
		user, err = s.userRepo.FindByEmail(email)
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get user by email failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return user, nil
}

func (s *loginService) generateTokenResponse(sub string, user *model.User, Client *model.Client) (*dto.LoginResponseDTO, error) {
	accessToken, err := jwt.GenerateAccessToken(
		sub,
		"openid profile email",
		*Client.Domain,
		*Client.Identifier,
		*Client.Identifier,
		Client.IdentityProvider.Identifier,
	)
	if err != nil {
		return nil, err
	}

	// Create user profile for ID token (populate from user data)
	profile := &jwt.UserProfile{
		Email:         user.Email,
		EmailVerified: user.IsEmailVerified,
		Phone:         user.Phone,
		PhoneVerified: user.IsPhoneVerified,
	}

	// Generate ID token with user profile (no nonce for login flow)
	idToken, err := generateIDTokenFn(sub, *Client.Domain, *Client.Identifier, Client.IdentityProvider.Identifier, profile, "")
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateRefreshTokenFn(sub, *Client.Domain, *Client.Identifier, Client.IdentityProvider.Identifier)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponseDTO{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		IssuedAt:     time.Now().Unix(),
	}, nil
}
