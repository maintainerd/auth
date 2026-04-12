package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/crypto"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/jwt"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/ptr"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/security"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type RegisterService interface {
	RegisterPublic(ctx context.Context, username, fullname, password string, email, phone *string, clientID, providerID string) (*dto.RegisterResponseDTO, error)
	RegisterInvitePublic(ctx context.Context, username, password, clientID, providerID, inviteToken string) (*dto.RegisterResponseDTO, error)
	Register(ctx context.Context, username, fullname, password string, email, phone *string, clientID, providerID *string) (*dto.RegisterResponseDTO, error)
	RegisterInvite(ctx context.Context, username, password, inviteToken string, clientID, providerID *string) (*dto.RegisterResponseDTO, error)
}

type registerService struct {
	db                   *gorm.DB
	clientRepo           repository.ClientRepository
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
	clientRepo repository.ClientRepository,
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
		clientRepo:           clientRepo,
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
	role, err := roleRepo.FindByNameAndTenantID(model.RoleRegistered, tenantID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, apperror.NewValidation("no default role found for tenant")
	}

	return role, nil
}

// RegisterPublic registers new users for public-facing applications.
// Requires clientID and providerID to identify the auth client.
// Used by external applications on port 8081.
func (s *registerService) RegisterPublic(
	ctx context.Context,
	username,
	fullname,
	password string,
	email,
	phone *string,
	clientID,
	providerID string,
) (*dto.RegisterResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "register.public")
	defer span.End()
	span.SetAttributes(attribute.String("client.id", clientID), attribute.String("provider.id", providerID))

	// Rate limiting check to prevent registration abuse
	if err := security.CheckRateLimit(username); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "register public failed")
		return nil, err
	}

	var createdUser *model.User
	var Client *model.Client
	var userIdentitySub string

	// All database operations in transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txIdentityProviderRepo := s.identityProviderRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		// Get and validate auth client with proper relationship preloading
		var txErr error
		Client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(clientID, providerID)
		if txErr != nil {
			return txErr
		}
		if Client == nil ||
			Client.Status != model.StatusActive ||
			Client.Domain == nil || *Client.Domain == "" {
			return apperror.NewValidation("invalid or inactive auth client")
		}

		// Look up identity provider by identifier to get auth container
		identityProvider, txErr := txIdentityProviderRepo.FindByIdentifier(providerID)
		if txErr != nil {
			return apperror.NewInternal("identity provider lookup failed", txErr)
		}

		if identityProvider == nil {
			return apperror.NewNotFoundWithReason("identity provider not found")
		}

		tenantId := identityProvider.TenantID

		// Check if username already exists
		existingUser, txErr := txUserRepo.FindByUsername(username)
		if txErr != nil {
			return txErr
		}
		if existingUser != nil {
			return apperror.NewConflict("username already taken")
		}

		// Check if email already exists (if provided)
		if email != nil && *email != "" {
			existingEmailUser, txErr := txUserRepo.FindByEmail(*email)
			if txErr != nil {
				return txErr
			}
			if existingEmailUser != nil {
				return apperror.NewConflict("email already registered")
			}
		}

		// Check if phone already exists (if provided)
		if phone != nil && *phone != "" {
			existingPhoneUser, txErr := txUserRepo.FindByPhone(*phone)
			if txErr != nil {
				return txErr
			}
			if existingPhoneUser != nil {
				return apperror.NewConflict("phone number already registered")
			}
		}

		// Hash password
		hashed, txErr := security.HashPassword([]byte(password))
		if txErr != nil {
			return txErr
		}

		// Create user
		newUser := &model.User{
			Username: username,
			Fullname: fullname,
			Password: ptr.Ptr(string(hashed)),
			Status:   model.StatusActive,
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
			UserID:   createdUser.UserID,
			ClientID: Client.ClientID,
			Provider: model.ProviderDefault,
			Sub:      uuid.New().String(),
			Metadata: datatypes.JSON([]byte(`{}`)),
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
		otp, txErr := crypto.GenerateOTP(6)
		if txErr != nil {
			return txErr
		}

		// Create user token
		userToken := &model.UserToken{
			UserID:    createdUser.UserID,
			TokenType: model.TokenTypeEmailVerification,
			Token:     otp,
		}
		_, txErr = txUserTokenRepo.Create(userToken)
		if txErr != nil {
			return txErr
		}

		return nil // commit transaction
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "register public failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	// Return token response
	return s.generateTokenResponse(userIdentitySub, createdUser, Client)
}

// Register registers new users for internal applications.
// If clientID and providerID are provided, uses the specified auth client.
// If not provided, uses the default auth client.
// Used by internal applications on port 8080.
func (s *registerService) Register(
	ctx context.Context,
	username,
	fullname,
	password string,
	email,
	phone *string,
	clientID,
	providerID *string,
) (*dto.RegisterResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "register.internal")
	defer span.End()

	// Rate limiting check to prevent registration abuse
	if err := security.CheckRateLimit(username); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "register failed")
		return nil, err
	}

	var createdUser *model.User
	var Client *model.Client
	var userIdentitySub string

	// All database operations in transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		// Get auth client - either by client_id and provider_id or default
		var txErr error
		if clientID != nil && providerID != nil {
			// Get auth client by client_id and identity provider identifier
			Client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(*clientID, *providerID)
			if txErr != nil {
				return apperror.NewInternal("auth client lookup by client_id and provider_id failed", txErr)
			}
		} else {
			// Get default auth client for internal authentication
			Client, txErr = txClientRepo.FindSystem()
			if txErr != nil {
				return txErr
			}
		}

		if Client == nil ||
			Client.Status != model.StatusActive ||
			Client.Domain == nil || *Client.Domain == "" {
			return apperror.NewNotFoundWithReason("auth client not found or inactive")
		}

		// Get tenant ID from the default client's identity provider
		tenantId := Client.IdentityProvider.Tenant.TenantID

		// Check if user already exists
		existingUser, txErr := txUserRepo.FindByUsername(username)
		if txErr != nil && txErr.Error() != "record not found" {
			return txErr
		}
		if existingUser != nil {
			return apperror.NewConflict("user already exists")
		}

		// Hash password
		hashed, txErr := security.HashPassword([]byte(password))
		if txErr != nil {
			return txErr
		}

		// Create user
		newUser := &model.User{
			Username: username,
			Fullname: fullname,
			Password: ptr.Ptr(string(hashed)),
			Status:   model.StatusActive,
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
			TenantID: tenantId,
			UserID:   createdUser.UserID,
			ClientID: Client.ClientID,
			Provider: model.ProviderDefault,
			Sub:      uuid.New().String(),
			Metadata: datatypes.JSON([]byte(`{}`)),
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
		otp, txErr := crypto.GenerateOTP(6)
		if txErr != nil {
			return txErr
		}

		// Create user token
		userToken := &model.UserToken{
			UserID:    createdUser.UserID,
			TokenType: model.TokenTypeEmailVerification,
			Token:     otp,
		}
		_, txErr = txUserTokenRepo.Create(userToken)
		if txErr != nil {
			return txErr
		}

		return nil // commit transaction
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "register failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	// Return token response
	return s.generateTokenResponse(userIdentitySub, createdUser, Client)
}

// RegisterInvite registers new users via invite token for internal applications.
// If clientID and providerID are provided, uses the specified auth client.
// If not provided, uses the default auth client.
// Used by internal applications on port 8080.
func (s *registerService) RegisterInvite(
	ctx context.Context,
	username,
	password,
	inviteToken string,
	clientID,
	providerID *string,
) (*dto.RegisterResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "register.invite")
	defer span.End()

	var createdUser *model.User
	var Client *model.Client
	var userIdentitySub string

	// All database operations in transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txInviteRepo := s.inviteRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)

		// Get auth client - either by client_id and provider_id or default
		var txErr error
		if clientID != nil && providerID != nil {
			// Get auth client by client_id and identity provider identifier
			Client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(*clientID, *providerID)
			if txErr != nil {
				return apperror.NewInternal("auth client lookup by client_id and provider_id failed", txErr)
			}
		} else {
			// Get default auth client for internal authentication
			Client, txErr = txClientRepo.FindSystem()
			if txErr != nil {
				return txErr
			}
		}

		if Client == nil ||
			Client.Status != model.StatusActive ||
			Client.Domain == nil || *Client.Domain == "" {
			return apperror.NewNotFoundWithReason("auth client not found or inactive")
		}

		// Get tenant ID from the default client's identity provider
		tenantId := Client.IdentityProvider.Tenant.TenantID

		// Validate invite token
		invite, txErr := txInviteRepo.FindByToken(inviteToken)
		if txErr != nil {
			return apperror.NewUnauthorized("invalid invite token")
		}
		if invite == nil || invite.Status != model.StatusPending || (invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now())) {
			return apperror.NewUnauthorized("invite token is invalid or expired")
		}

		// Check if user already exists
		existingUser, txErr := txUserRepo.FindByUsername(username)
		if txErr != nil && txErr.Error() != "record not found" {
			return txErr
		}
		if existingUser != nil {
			return apperror.NewConflict("user already exists")
		}

		// Hash password
		hashed, txErr := security.HashPassword([]byte(password))
		if txErr != nil {
			return txErr
		}

		// Create user
		newUser := &model.User{
			Username: username,
			Fullname: username, // Use username as fullname since invite doesn't have fullname
			Password: ptr.Ptr(string(hashed)),
			Email:    invite.InvitedEmail,
			Status:   model.StatusActive,
		}

		createdUser, txErr = txUserRepo.Create(newUser)
		if txErr != nil {
			return txErr
		}

		// Create user identity
		userIdentity := &model.UserIdentity{
			TenantID: tenantId,
			UserID:   createdUser.UserID,
			ClientID: Client.ClientID,
			Provider: model.ProviderDefault,
			Sub:      uuid.New().String(),
			Metadata: datatypes.JSON([]byte(`{}`)),
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "register invite failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	// Return token response
	return s.generateTokenResponse(userIdentitySub, createdUser, Client)
}

// RegisterInvitePublic registers new users via invite token for public-facing applications.
// Requires clientID and providerID to identify the auth client.
// Used by external applications on port 8081.
func (s *registerService) RegisterInvitePublic(
	ctx context.Context,
	username,
	password,
	clientID,
	providerID,
	inviteToken string,
) (*dto.RegisterResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "register.invitePublic")
	defer span.End()
	span.SetAttributes(attribute.String("client.id", clientID), attribute.String("provider.id", providerID))

	var createdUser *model.User
	var Client *model.Client
	var userIdentitySub string

	// All database operations in transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txInviteRepo := s.inviteRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txIdentityProviderRepo := s.identityProviderRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)

		// Get and validate auth client with proper relationship preloading
		var txErr error
		Client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(clientID, providerID)
		if txErr != nil {
			return txErr
		}
		if Client == nil ||
			Client.Status != model.StatusActive ||
			Client.Domain == nil || *Client.Domain == "" {
			return apperror.NewValidation("invalid or inactive auth client")
		}

		// Look up identity provider by identifier to get tenant
		identityProvider, txErr := txIdentityProviderRepo.FindByIdentifier(providerID)
		if txErr != nil {
			return apperror.NewInternal("identity provider lookup failed", txErr)
		}

		if identityProvider == nil {
			return apperror.NewNotFoundWithReason("identity provider not found")
		}

		tenantId := identityProvider.TenantID

		// Validate invite token
		invite, txErr := txInviteRepo.FindByToken(inviteToken)
		if txErr != nil {
			return apperror.NewUnauthorized("invalid invite token")
		}
		if invite == nil {
			return apperror.NewNotFound("invite not found")
		}

		// Check invite status and expiration
		if invite.Status != model.StatusPending {
			return apperror.NewUnauthorized("invite has already been used or is no longer valid")
		}
		if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
			return apperror.NewUnauthorized("invite has expired")
		}

		// Check if username already exists
		existingUser, txErr := txUserRepo.FindByUsername(username)
		if txErr != nil {
			return txErr
		}
		if existingUser != nil {
			return apperror.NewConflict("username already taken")
		}

		// Check if invited email already exists
		existingEmailUser, txErr := txUserRepo.FindByEmail(invite.InvitedEmail)
		if txErr != nil {
			return txErr
		}
		if existingEmailUser != nil {
			return apperror.NewConflict("invited email already registered")
		}

		// Hash password
		hashed, txErr := security.HashPassword([]byte(password))
		if txErr != nil {
			return txErr
		}

		// Create user
		newUser := &model.User{
			Username:        username,
			Email:           invite.InvitedEmail, // Always use the invited email
			Password:        ptr.Ptr(string(hashed)),
			Status:          model.StatusActive,
			IsEmailVerified: true, // Auto-verify email for invited users
		}

		createdUser, txErr = txUserRepo.Create(newUser)
		if txErr != nil {
			return txErr
		}

		// Create user identity
		userIdentity := &model.UserIdentity{
			TenantID: tenantId,
			UserID:   createdUser.UserID,
			ClientID: Client.ClientID,
			Provider: model.ProviderDefault,
			Sub:      uuid.New().String(),
			Metadata: datatypes.JSON([]byte(`{}`)),
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
			if txErr != nil {
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "register invite public failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	// Return token response
	return s.generateTokenResponse(userIdentitySub, createdUser, Client)
}

func (s *registerService) generateTokenResponse(sub string, user *model.User, Client *model.Client) (*dto.RegisterResponseDTO, error) {
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

	// Generate ID token with user profile (no nonce for registration flow)
	idToken, err := jwt.GenerateIDToken(sub, *Client.Domain, *Client.Identifier, Client.IdentityProvider.Identifier, profile, "")
	if err != nil {
		return nil, err
	}

	refreshToken, err := jwt.GenerateRefreshToken(sub, *Client.Domain, *Client.Identifier, Client.IdentityProvider.Identifier)
	if err != nil {
		return nil, err
	}

	return &dto.RegisterResponseDTO{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		IssuedAt:     time.Now().Unix(),
	}, nil
}
