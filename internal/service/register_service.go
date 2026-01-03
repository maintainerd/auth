package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type RegisterService interface {
	RegisterPublic(username, fullname, password string, email, phone *string, clientID, providerID string) (*dto.RegisterResponseDto, error)
	RegisterInvitePublic(username, password, clientID, providerID, inviteToken string) (*dto.RegisterResponseDto, error)
	Register(username, fullname, password string, email, phone *string, clientID, providerID *string) (*dto.RegisterResponseDto, error)
	RegisterInvite(username, password, inviteToken string, clientID, providerID *string) (*dto.RegisterResponseDto, error)
}

type registerService struct {
	db                   *gorm.DB
	authClientRepo       repository.AuthClientRepository
	userRepo             repository.UserRepository
	userRoleRepo         repository.UserRoleRepository
	userTokenRepo        repository.UserTokenRepository
	userIdentityRepo     repository.UserIdentityRepository
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
	userIdentityRepo repository.UserIdentityRepository,
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
		userIdentityRepo:     userIdentityRepo,
		roleRepo:             roleRepo,
		inviteRepo:           inviteRepo,
		identityProviderRepo: identityProviderRepo,
	}
}

// Helper function to find the default role for a tenant
func (s *registerService) findDefaultRole(roleRepo repository.RoleRepository, tenantID int64) (*model.Role, error) {
	// First try to find a role marked as default
	filter := repository.RoleRepositoryGetFilter{
		IsDefault: &[]bool{true}[0],
		TenantID:  tenantID,
		Page:      1,
		Limit:     1,
	}

	result, err := roleRepo.FindPaginated(filter)
	if err != nil {
		return nil, err
	}

	if len(result.Data) > 0 {
		return &result.Data[0], nil
	}

	// Fallback: if no default role found, try to find "registered" role
	role, err := roleRepo.FindByNameAndTenantID("registered", tenantID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("no default role found for tenant")
	}

	return role, nil
}

// RegisterPublic registers new users for public-facing applications.
// Requires clientID and providerID to identify the auth client.
// Used by external applications on port 8081.
func (s *registerService) RegisterPublic(
	username,
	fullname,
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
	var userIdentitySub string

	// All database operations in transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txIdentityProviderRepo := s.identityProviderRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		// Get and validate auth client with proper relationship preloading
		var txErr error
		authClient, txErr = txAuthClientRepo.FindByClientIDAndIdentityProvider(clientID, providerID)
		if txErr != nil {
			return txErr
		}
		if authClient == nil ||
			authClient.Status != "active" ||
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
		existingUser, txErr := txUserRepo.FindByUsername(username)
		if txErr != nil {
			return txErr
		}
		if existingUser != nil {
			return errors.New("username already taken")
		}

		// Check if email already exists (if provided)
		if email != nil && *email != "" {
			existingEmailUser, txErr := txUserRepo.FindByEmail(*email)
			if txErr != nil {
				return txErr
			}
			if existingEmailUser != nil {
				return errors.New("email already registered")
			}
		}

		// Check if phone already exists (if provided)
		if phone != nil && *phone != "" {
			existingPhoneUser, txErr := txUserRepo.FindByPhone(*phone)
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
			Fullname: fullname,
			Password: util.Ptr(string(hashed)),
			Status:   "active",
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

		// Create user identity
		userIdentity := &model.UserIdentity{
			UserID:       createdUser.UserID,
			AuthClientID: authClient.AuthClientID,
			Provider:     "default",
			Sub:          uuid.New().String(),
			Metadata:     datatypes.JSON([]byte(`{}`)),
		}

		_, txErr = txUserIdentityRepo.Create(userIdentity)
		if txErr != nil {
			return txErr
		}

		// Get default role
		defaultRole, txErr := s.findDefaultRole(txRoleRepo, tenantId)
		if txErr != nil {
			return txErr
		}

		// Assign default role to user
		userRole := &model.UserRole{
			UserID: createdUser.UserID,
			RoleID: defaultRole.RoleID,
		}
		_, txErr = txUserRoleRepo.Create(userRole)
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
	return s.generateTokenResponse(userIdentitySub, createdUser, authClient)
}

// Register registers new users for internal applications.
// If clientID and providerID are provided, uses the specified auth client.
// If not provided, uses the default auth client.
// Used by internal applications on port 8080.
func (s *registerService) Register(
	username,
	fullname,
	password string,
	email,
	phone *string,
	clientID,
	providerID *string,
) (*dto.RegisterResponseDto, error) {
	// Rate limiting check to prevent registration abuse
	if err := util.CheckRateLimit(username); err != nil {
		return nil, err
	}

	var createdUser *model.User
	var authClient *model.AuthClient
	var userIdentitySub string

	// All database operations in transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		// Get auth client - either by client_id and provider_id or default
		var txErr error
		if clientID != nil && providerID != nil {
			// Get auth client by client_id and identity provider identifier
			authClient, txErr = txAuthClientRepo.FindByClientIDAndIdentityProvider(*clientID, *providerID)
			if txErr != nil {
				return errors.New("auth client lookup by client_id and provider_id failed")
			}
		} else {
			// Get default auth client for internal authentication
			authClient, txErr = txAuthClientRepo.FindDefault()
			if txErr != nil {
				return txErr
			}
		}

		if authClient == nil ||
			authClient.Status != "active" ||
			authClient.Domain == nil || *authClient.Domain == "" {
			return errors.New("auth client not found or inactive")
		}

		// Get tenant ID from the default client's identity provider
		tenantId := authClient.IdentityProvider.Tenant.TenantID

		// Check if user already exists
		existingUser, txErr := txUserRepo.FindByUsername(username)
		if txErr != nil && txErr.Error() != "record not found" {
			return txErr
		}
		if existingUser != nil {
			return errors.New("user already exists")
		}

		// Hash password
		hashed, txErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if txErr != nil {
			return txErr
		}

		// Create user
		newUser := &model.User{
			Username: username,
			Fullname: fullname,
			Password: util.Ptr(string(hashed)),
			Status:   "active",
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

		// Create user identity
		userIdentity := &model.UserIdentity{
			TenantID:     tenantId,
			UserID:       createdUser.UserID,
			AuthClientID: authClient.AuthClientID,
			Provider:     "default",
			Sub:          uuid.New().String(),
			Metadata:     datatypes.JSON([]byte(`{}`)),
		}

		_, txErr = txUserIdentityRepo.Create(userIdentity)
		if txErr != nil {
			return txErr
		}

		// Get default role
		defaultRole, txErr := s.findDefaultRole(txRoleRepo, tenantId)
		if txErr != nil {
			return txErr
		}

		// Assign default role to user
		userRole := &model.UserRole{
			UserID: createdUser.UserID,
			RoleID: defaultRole.RoleID,
		}
		_, txErr = txUserRoleRepo.Create(userRole)
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
	return s.generateTokenResponse(userIdentitySub, createdUser, authClient)
}

// RegisterInvite registers new users via invite token for internal applications.
// If clientID and providerID are provided, uses the specified auth client.
// If not provided, uses the default auth client.
// Used by internal applications on port 8080.
func (s *registerService) RegisterInvite(
	username,
	password,
	inviteToken string,
	clientID,
	providerID *string,
) (*dto.RegisterResponseDto, error) {
	var createdUser *model.User
	var authClient *model.AuthClient
	var userIdentitySub string

	// All database operations in transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txInviteRepo := s.inviteRepo.WithTx(tx)
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)

		// Get auth client - either by client_id and provider_id or default
		var txErr error
		if clientID != nil && providerID != nil {
			// Get auth client by client_id and identity provider identifier
			authClient, txErr = txAuthClientRepo.FindByClientIDAndIdentityProvider(*clientID, *providerID)
			if txErr != nil {
				return errors.New("auth client lookup by client_id and provider_id failed")
			}
		} else {
			// Get default auth client for internal authentication
			authClient, txErr = txAuthClientRepo.FindDefault()
			if txErr != nil {
				return txErr
			}
		}

		if authClient == nil ||
			authClient.Status != "active" ||
			authClient.Domain == nil || *authClient.Domain == "" {
			return errors.New("auth client not found or inactive")
		}

		// Get tenant ID from the default client's identity provider
		tenantId := authClient.IdentityProvider.Tenant.TenantID

		// Validate invite token
		invite, txErr := txInviteRepo.FindByToken(inviteToken)
		if txErr != nil {
			return errors.New("invalid invite token")
		}
		if invite == nil || invite.Status != "pending" || (invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now())) {
			return errors.New("invite token is invalid or expired")
		}

		// Check if user already exists
		existingUser, txErr := txUserRepo.FindByUsername(username)
		if txErr != nil && txErr.Error() != "record not found" {
			return txErr
		}
		if existingUser != nil {
			return errors.New("user already exists")
		}

		// Hash password
		hashed, txErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if txErr != nil {
			return txErr
		}

		// Create user
		newUser := &model.User{
			Username: username,
			Fullname: username, // Use username as fullname since invite doesn't have fullname
			Password: util.Ptr(string(hashed)),
			Email:    invite.InvitedEmail,
			Status:   "active",
		}

		createdUser, txErr = txUserRepo.Create(newUser)
		if txErr != nil {
			return txErr
		}

		// Create user identity
		userIdentity := &model.UserIdentity{
			TenantID:     tenantId,
			UserID:       createdUser.UserID,
			AuthClientID: authClient.AuthClientID,
			Provider:     "default",
			Sub:          uuid.New().String(),
			Metadata:     datatypes.JSON([]byte(`{}`)),
		}

		_, txErr = txUserIdentityRepo.Create(userIdentity)
		if txErr != nil {
			return txErr
		}

		// Get default role and assign it first
		defaultRole, txErr := s.findDefaultRole(txRoleRepo, tenantId)
		if txErr != nil {
			return txErr
		}

		// Assign default role to user
		defaultUserRole := &model.UserRole{
			UserID: createdUser.UserID,
			RoleID: defaultRole.RoleID,
		}
		_, txErr = txUserRoleRepo.Create(defaultUserRole)
		if txErr != nil {
			return txErr
		}

		// Assign additional roles from invite
		for _, role := range invite.Roles {
			userRole := &model.UserRole{
				UserID: createdUser.UserID,
				RoleID: role.RoleID,
			}
			_, txErr = txUserRoleRepo.Create(userRole)
			if txErr != nil {
				return txErr
			}
		}

		// Mark invite as used
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
	return s.generateTokenResponse(userIdentitySub, createdUser, authClient)
}

// RegisterInvitePublic registers new users via invite token for public-facing applications.
// Requires clientID and providerID to identify the auth client.
// Used by external applications on port 8081.
func (s *registerService) RegisterInvitePublic(
	username,
	password,
	clientID,
	providerID,
	inviteToken string,
) (*dto.RegisterResponseDto, error) {
	var createdUser *model.User
	var authClient *model.AuthClient
	var userIdentitySub string

	// All database operations in transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txInviteRepo := s.inviteRepo.WithTx(tx)
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txIdentityProviderRepo := s.identityProviderRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)

		// Get and validate auth client with proper relationship preloading
		var txErr error
		authClient, txErr = txAuthClientRepo.FindByClientIDAndIdentityProvider(clientID, providerID)
		if txErr != nil {
			return txErr
		}
		if authClient == nil ||
			authClient.Status != "active" ||
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
		existingUser, txErr := txUserRepo.FindByUsername(username)
		if txErr != nil {
			return txErr
		}
		if existingUser != nil {
			return errors.New("username already taken")
		}

		// Check if invited email already exists
		existingEmailUser, txErr := txUserRepo.FindByEmail(invite.InvitedEmail)
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
			Status:          "active",
			IsEmailVerified: true, // Auto-verify email for invited users
		}

		createdUser, txErr = txUserRepo.Create(newUser)
		if txErr != nil {
			return txErr
		}

		// Create user identity
		userIdentity := &model.UserIdentity{
			TenantID:     tenantId,
			UserID:       createdUser.UserID,
			AuthClientID: authClient.AuthClientID,
			Provider:     "default",
			Sub:          uuid.New().String(),
			Metadata:     datatypes.JSON([]byte(`{}`)),
		}

		_, txErr = txUserIdentityRepo.Create(userIdentity)
		if txErr != nil {
			return txErr
		}

		// Get default role and assign it first
		defaultRole, txErr := s.findDefaultRole(txRoleRepo, tenantId)
		if txErr != nil {
			return txErr
		}

		// Assign default role to user
		defaultUserRole := &model.UserRole{
			UserID: createdUser.UserID,
			RoleID: defaultRole.RoleID,
		}
		_, txErr = txUserRoleRepo.Create(defaultUserRole)
		if txErr != nil {
			return txErr
		}

		// Assign all additional roles from the invite to the user
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
	return s.generateTokenResponse(userIdentitySub, createdUser, authClient)
}

func (s *registerService) generateTokenResponse(sub string, user *model.User, authClient *model.AuthClient) (*dto.RegisterResponseDto, error) {
	accessToken, err := util.GenerateAccessToken(
		sub,
		"openid profile email",
		*authClient.Domain,
		*authClient.ClientID,
		*authClient.ClientID,
		authClient.IdentityProvider.Identifier,
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
	idToken, err := util.GenerateIDToken(sub, *authClient.Domain, *authClient.ClientID, authClient.IdentityProvider.Identifier, profile, "")
	if err != nil {
		return nil, err
	}

	refreshToken, err := util.GenerateRefreshToken(sub, *authClient.Domain, *authClient.ClientID, authClient.IdentityProvider.Identifier)
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
