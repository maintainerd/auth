package service

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RegisterService interface {
	Register(username, password string, email, phone *string, clientID, providerID string) (*dto.RegisterResponseDto, error)
	RegisterInvite(username, password, clientID, providerID, inviteToken string) (*dto.RegisterResponseDto, error)
}

type registerService struct {
	db                   *gorm.DB
	authClientRepo       repository.AuthClientRepository
	userRepo             repository.UserRepository
	userRoleRepo         repository.UserRoleRepository
	userTokenRepo        repository.UserTokenRepository
	roleRepo             repository.RoleRepository
	inviteRepo           repository.InviteRepository
	identityProviderRepo repository.IdentityProviderRepository
}

func NewRegistrationService(
	db *gorm.DB,
	clientRepo repository.AuthClientRepository,
	userRepo repository.UserRepository,
	userRoleRepo repository.UserRoleRepository,
	userTokenRepo repository.UserTokenRepository,
	roleRepo repository.RoleRepository,
	inviteRepo repository.InviteRepository,
	identityProviderRepo repository.IdentityProviderRepository,
) RegisterService {
	return &registerService{
		db:                   db,
		authClientRepo:       clientRepo,
		userRepo:             userRepo,
		userRoleRepo:         userRoleRepo,
		userTokenRepo:        userTokenRepo,
		roleRepo:             roleRepo,
		inviteRepo:           inviteRepo,
		identityProviderRepo: identityProviderRepo,
	}
}

func (s *registerService) Register(
	username,
	password string,
	email,
	phone *string,
	clientID,
	providerID string,
) (*dto.RegisterResponseDto, error) {
	// Rate limiting check to prevent registration abuse
	if err := util.CheckRateLimit(username); err != nil {
		return nil, err
	}

	var createdUser *model.User
	var authClient *model.AuthClient

	// All database operations in transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txIdentityProviderRepo := s.identityProviderRepo.WithTx(tx)

		// Get and validate auth client with proper relationship preloading
		var txErr error
		authClient, txErr = txAuthClientRepo.FindByClientIDAndIdentityProvider(clientID, providerID)
		if txErr != nil {
			return txErr
		}
		if authClient == nil ||
			!authClient.IsActive ||
			authClient.Domain == nil || *authClient.Domain == "" {
			return errors.New("invalid or inactive auth client")
		}

		// Look up identity provider by identifier to get auth container
		identityProvider, txErr := txIdentityProviderRepo.FindByIdentifier(providerID)
		if txErr != nil {
			return errors.New("identity provider lookup failed")
		}

		if identityProvider == nil {
			return errors.New("identity provider not found")
		}

		tenantId := identityProvider.TenantID

		// Check if username already exists
		existingUser, txErr := txUserRepo.FindByUsername(username, tenantId)
		if txErr != nil {
			return txErr
		}
		if existingUser != nil {
			return errors.New("username already taken")
		}

		// Check if email already exists (if provided)
		if email != nil && *email != "" {
			existingEmailUser, txErr := txUserRepo.FindByEmail(*email, tenantId)
			if txErr != nil {
				return txErr
			}
			if existingEmailUser != nil {
				return errors.New("email already registered")
			}
		}

		// Check if phone already exists (if provided)
		if phone != nil && *phone != "" {
			existingPhoneUser, txErr := txUserRepo.FindByPhone(*phone, tenantId)
			if txErr != nil {
				return txErr
			}
			if existingPhoneUser != nil {
				return errors.New("phone number already registered")
			}
		}

		// Hash password
		hashed, txErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if txErr != nil {
			return txErr
		}

		// Create user
		newUser := &model.User{
			Username: username,
			Password: util.Ptr(string(hashed)),
			TenantID: tenantId,
			IsActive: true,
		}

		// Set email if provided
		if email != nil && *email != "" {
			newUser.Email = *email
		}

		// Set phone if provided
		if phone != nil && *phone != "" {
			newUser.Phone = *phone
		}

		createdUser, txErr = txUserRepo.Create(newUser)
		if txErr != nil {
			return txErr
		}

		// Generate OTP
		otp, txErr := util.GenerateOTP(6)
		if txErr != nil {
			return txErr
		}

		// Create user token
		userToken := &model.UserToken{
			UserID:    createdUser.UserID,
			TokenType: "user:email:verification",
			Token:     otp,
		}
		_, txErr = txUserTokenRepo.Create(userToken)
		if txErr != nil {
			return txErr
		}

		return nil // commit transaction
	})

	if err != nil {
		return nil, err
	}

	// Return token response
	return s.generateTokenResponse(createdUser.UserUUID.String(), createdUser, authClient)
}

func (s *registerService) RegisterInvite(
	username,
	password,
	clientID,
	providerID,
	inviteToken string,
) (*dto.RegisterResponseDto, error) {
	var createdUser *model.User
	var authClient *model.AuthClient

	// All database operations in transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)
		txInviteRepo := s.inviteRepo.WithTx(tx)
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txIdentityProviderRepo := s.identityProviderRepo.WithTx(tx)

		// Get and validate auth client with proper relationship preloading
		var txErr error
		authClient, txErr = txAuthClientRepo.FindByClientIDAndIdentityProvider(clientID, providerID)
		if txErr != nil {
			return txErr
		}
		if authClient == nil ||
			!authClient.IsActive ||
			authClient.Domain == nil || *authClient.Domain == "" {
			return errors.New("invalid or inactive auth client")
		}

		// Look up identity provider by identifier to get tenant
		identityProvider, txErr := txIdentityProviderRepo.FindByIdentifier(providerID)
		if txErr != nil {
			return errors.New("identity provider lookup failed")
		}

		if identityProvider == nil {
			return errors.New("identity provider not found")
		}

		tenantId := identityProvider.TenantID

		// Validate invite token
		invite, txErr := txInviteRepo.FindByToken(inviteToken)
		if txErr != nil {
			return errors.New("invalid invite token")
		}
		if invite == nil {
			return errors.New("invite not found")
		}

		// Check invite status and expiration
		if invite.Status != "pending" {
			return errors.New("invite has already been used or is no longer valid")
		}
		if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
			return errors.New("invite has expired")
		}

		// Check if username already exists
		existingUser, txErr := txUserRepo.FindByUsername(username, tenantId)
		if txErr != nil {
			return txErr
		}
		if existingUser != nil {
			return errors.New("username already taken")
		}

		// Check if invited email already exists
		existingEmailUser, txErr := txUserRepo.FindByEmail(invite.InvitedEmail, tenantId)
		if txErr != nil {
			return txErr
		}
		if existingEmailUser != nil {
			return errors.New("invited email already registered")
		}

		// Hash password
		hashed, txErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if txErr != nil {
			return txErr
		}

		// Create user
		newUser := &model.User{
			Username:        username,
			Email:           invite.InvitedEmail, // Always use the invited email
			Password:        util.Ptr(string(hashed)),
			TenantID:        tenantId,
			IsActive:        true,
			IsEmailVerified: true, // Auto-verify email for invited users
		}

		createdUser, txErr = txUserRepo.Create(newUser)
		if txErr != nil {
			return txErr
		}

		// Assign all roles from the invite to the user
		for _, role := range invite.Roles {
			// Check if user already has this role (shouldn't happen for new user, but safety check)
			existingUserRole, txErr := txUserRoleRepo.FindByUserIDAndRoleID(createdUser.UserID, role.RoleID)
			if txErr != nil && !errors.Is(txErr, gorm.ErrRecordNotFound) {
				return txErr
			}
			if existingUserRole != nil {
				continue // Skip if already assigned
			}

			// Create user-role association
			userRole := &model.UserRole{
				UserID: createdUser.UserID,
				RoleID: role.RoleID,
			}

			_, txErr = txUserRoleRepo.Create(userRole)
			if txErr != nil {
				return txErr
			}
		}

		// Mark invite as used (using repository method)
		txErr = txInviteRepo.MarkAsUsed(invite.InviteUUID)
		if txErr != nil {
			return txErr
		}

		return nil // commit transaction
	})

	if err != nil {
		return nil, err
	}

	// Return token response
	return s.generateTokenResponse(createdUser.UserUUID.String(), createdUser, authClient)
}

func (s *registerService) generateTokenResponse(userUUID string, user *model.User, authClient *model.AuthClient) (*dto.RegisterResponseDto, error) {
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

	// Generate ID token with user profile (no nonce for registration flow)
	idToken, err := util.GenerateIDToken(userUUID, *authClient.Domain, *authClient.ClientID, profile, "")
	if err != nil {
		return nil, err
	}

	refreshToken, err := util.GenerateRefreshToken(userUUID, *authClient.Domain, *authClient.ClientID)
	if err != nil {
		return nil, err
	}

	return &dto.RegisterResponseDto{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		IssuedAt:     time.Now().Unix(),
	}, nil
}
