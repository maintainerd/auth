package authn

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var secValidatePasswordPolicy = security.ValidatePasswordPolicy
var secHashPassword = security.HashPassword
var secHashPasswordWithPolicy = security.HashPasswordWithPolicy

type RegisterService interface {
	RegisterPublic(ctx context.Context, username, fullname, password string, email, phone *string, clientID, providerID *string) (*RegisterResponseDTO, error)
	RegisterInvitePublic(ctx context.Context, username, password, clientID, providerID, inviteToken string) (*RegisterResponseDTO, error)
	Register(ctx context.Context, username, fullname, password string, email, phone *string, clientID, providerID *string) (*RegisterResponseDTO, error)
}

type registerService struct {
	db                   *gorm.DB
	clientRepo           ClientRepository
	userRepo             UserRepository
	userRoleRepo         UserRoleRepository
	userTokenRepo        UserTokenRepository
	userIdentityRepo     UserIdentityRepository
	roleRepo             RoleRepository
	inviteRepo           InviteRepository
	identityProviderRepo IdentityProviderRepository
	securitySettingRepo  secpolicy.SecuritySettingRepository // nil → use defaults
	passwordHistoryRepo  UserPasswordHistoryRepository       // nil → skip history
	authFlowRoleRepo     AuthFlowRoleRepository
}

func NewRegistrationService(
	db *gorm.DB,
	clientRepo ClientRepository,
	userRepo UserRepository,
	userRoleRepo UserRoleRepository,
	userTokenRepo UserTokenRepository,
	userIdentityRepo UserIdentityRepository,
	roleRepo RoleRepository,
	inviteRepo InviteRepository,
	identityProviderRepo IdentityProviderRepository,
	securitySettingRepo secpolicy.SecuritySettingRepository,
	passwordHistoryRepo UserPasswordHistoryRepository,
	authFlowRoleRepo AuthFlowRoleRepository,
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
		securitySettingRepo:  securitySettingRepo,
		passwordHistoryRepo:  passwordHistoryRepo,
		authFlowRoleRepo:     authFlowRoleRepo,
	}
}

// Helper function to find the default role for a tenant
func (s *registerService) findDefaultRole(roleRepo RoleRepository, tenantID int64) (*Role, error) {
	// First try to find a role marked as default
	filter := RoleRepositoryGetFilter{
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
	role, err := roleRepo.FindByNameAndTenantID(shared.RoleRegistered, tenantID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, apperror.NewValidation("no default role found for tenant")
	}

	return role, nil
}

// resolveRegistrationRole prefers the tenant registration policy's configured
// default_role when it names an existing role for the tenant; otherwise it
// falls back to the tenant's default/registered role. This keeps a misconfigured
// or unseeded default_role from blocking registration.
func (s *registerService) resolveRegistrationRole(roleRepo RoleRepository, tenantID int64, configuredRole string) (*Role, error) {
	if name := strings.TrimSpace(configuredRole); name != "" {
		if role, err := roleRepo.FindByNameAndTenantID(name, tenantID); err == nil && role != nil {
			return role, nil
		}
	}
	return s.findDefaultRole(roleRepo, tenantID)
}

func enforceRegistrationAbuseControls(ctx context.Context, tenantID int64, regPolicy *secpolicy.RegistrationPolicy) error {
	if regPolicy == nil {
		return nil
	}
	if regPolicy.CaptchaOnSignup {
		captchaToken := registrationCaptchaTokenFromContext(ctx)
		clientIP := middleware.ClientIPFromContext(ctx)
		if err := security.VerifyCaptcha(ctx, captchaToken, clientIP); err != nil {
			return apperror.NewValidation("captcha verification failed")
		}
	}
	if regPolicy.RegistrationRateLimitPerIPPerHour > 0 {
		clientIP := middleware.ClientIPFromContext(ctx)
		if err := security.CheckRegistrationRateLimit(ctx, tenantID, clientIP, regPolicy.RegistrationRateLimitPerIPPerHour); err != nil {
			return apperror.NewValidation("registration rate limit exceeded")
		}
	}
	return nil
}

// RegisterPublic registers new users for public-facing applications.
// clientID and providerID are optional. When absent, registers under the
// default client of the system tenant (mirrors internal Register() fallback).
// Used by external applications on port 8081.
func (s *registerService) RegisterPublic(
	ctx context.Context,
	username,
	fullname,
	password string,
	email,
	phone *string,
	clientID,
	providerID *string,
) (*RegisterResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "register.public")
	defer span.End()
	if clientID != nil {
		span.SetAttributes(attribute.String("client.id", *clientID))
	}
	if providerID != nil {
		span.SetAttributes(attribute.String("provider.id", *providerID))
	}

	// Rate limiting check to prevent registration abuse
	if err := security.CheckRateLimit(username); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "register public failed")
		return nil, err
	}

	isExplicitClient := clientID != nil && providerID != nil

	var createdUser *User
	var Client *Client
	var userIdentitySub string

	// All database operations in transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txIdentityProviderRepo := s.identityProviderRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		var txErr error
		var tenantId int64

		if isExplicitClient {
			Client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(*clientID, *providerID)
			if txErr != nil {
				return txErr
			}
			if Client == nil ||
				Client.Status != shared.StatusActive ||
				Client.Domain == nil || *Client.Domain == "" {
				return apperror.NewValidation("invalid or inactive auth client")
			}

			identityProvider, txErr := txIdentityProviderRepo.FindByIdentifier(*providerID)
			if txErr != nil {
				return apperror.NewInternal("identity provider lookup failed", txErr)
			}
			if identityProvider == nil {
				return apperror.NewNotFoundWithReason("identity provider not found")
			}
			tenantId = identityProvider.TenantID
		} else {
			Client, txErr = txClientRepo.FindSystem()
			if txErr != nil {
				return txErr
			}
			if Client == nil ||
				Client.Status != shared.StatusActive ||
				Client.Domain == nil || *Client.Domain == "" {
				return apperror.NewNotFoundWithReason("auth client not found or inactive")
			}

			tenantId = Client.IdentityProvider.Tenant.TenantID
		}

		// Enforce tenant registration policy (self-service path).
		regPolicy := secpolicy.LoadRegistrationPolicy(s.securitySettingRepo, tenantId)
		if !regPolicy.SelfRegistrationEnabled {
			return apperror.NewForbidden("self-registration is disabled for this tenant")
		}
		if err := enforceRegistrationAbuseControls(ctx, tenantId, regPolicy); err != nil {
			return err
		}
		if email != nil && *email != "" && !regPolicy.EmailDomainAllowed(*email) {
			return apperror.NewValidation("email domain is not permitted for registration")
		}

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

		// Validate password against tenant policy
		policy := secpolicy.LoadPasswordPolicy(s.securitySettingRepo, tenantId)
		if txErr = secValidatePasswordPolicy(password, policy); txErr != nil {
			return apperror.NewValidation(txErr.Error())
		}

		// Hash password
		hashed, txErr := secHashPasswordWithPolicy(ctx, []byte(password), policy)
		if txErr != nil {
			return txErr
		}

		now := time.Now()
		// Create user. Email-verified state is governed by the tenant
		// registration policy (auto-confirm or verification-not-required).
		newUser := &User{
			Username:          username,
			Fullname:          fullname,
			Password:          ptr.Ptr(string(hashed)),
			Status:            registrationInitialStatus(regPolicy, email),
			IsEmailVerified:   regPolicy.EmailVerified(),
			PasswordChangedAt: &now,
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

		// Record password history
		secpolicy.RecordPasswordHistory(s.passwordHistoryRepo, createdUser.UserID, policy.HistoryCount, string(hashed))

		// Create user identity
		userIdentity := &UserIdentity{
			TenantID:           tenantId,
			UserID:             createdUser.UserID,
			ClientID:           Client.ClientID,
			IdentityProviderID: &Client.IdentityProviderID,
			Provider:           shared.ProviderDefault,
			Sub:                uuid.New().String(),
			Metadata:           datatypes.JSON([]byte(`{}`)),
		}

		_, txErr = txUserIdentityRepo.Create(userIdentity)
		if txErr != nil {
			return txErr
		}
		userIdentitySub = userIdentity.Sub

		// Resolve the role to assign: prefer the tenant policy's default_role
		// when it names an existing role, otherwise fall back to the tenant's
		// default/registered role.
		defaultRole, txErr := s.resolveRegistrationRole(txRoleRepo, tenantId, regPolicy.DefaultRole)
		if txErr != nil {
			return txErr
		}

		// Assign default role to user
		userRole := &UserRole{
			UserID: createdUser.UserID,
			RoleID: defaultRole.RoleID,
		}
		_, txErr = txUserRoleRepo.Create(userRole)
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
	return s.generateTokenResponse(ctx, userIdentitySub, createdUser, Client)
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
) (*RegisterResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "register.internal")
	defer span.End()

	// Rate limiting check to prevent registration abuse
	if err := security.CheckRateLimit(username); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "register failed")
		return nil, err
	}

	var createdUser *User
	var Client *Client
	var userIdentitySub string

	// All database operations in transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
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
			Client.Status != shared.StatusActive ||
			Client.Domain == nil || *Client.Domain == "" {
			return apperror.NewNotFoundWithReason("auth client not found or inactive")
		}

		// Get tenant ID from the default client's identity provider
		tenantId := Client.IdentityProvider.Tenant.TenantID

		// Enforce tenant registration policy.
		regPolicy := secpolicy.LoadRegistrationPolicy(s.securitySettingRepo, tenantId)
		if !regPolicy.SelfRegistrationEnabled {
			return apperror.NewForbidden("self-registration is disabled for this tenant")
		}
		if err := enforceRegistrationAbuseControls(ctx, tenantId, regPolicy); err != nil {
			return err
		}
		if email != nil && *email != "" && !regPolicy.EmailDomainAllowed(*email) {
			return apperror.NewValidation("email domain is not permitted for registration")
		}

		// Check if user already exists
		existingUser, txErr := txUserRepo.FindByUsername(username)
		if txErr != nil && txErr.Error() != "record not found" {
			return txErr
		}
		if existingUser != nil {
			return apperror.NewConflict("user already exists")
		}

		// Validate password against tenant policy
		policy := secpolicy.LoadPasswordPolicy(s.securitySettingRepo, tenantId)
		if txErr = secValidatePasswordPolicy(password, policy); txErr != nil {
			return apperror.NewValidation(txErr.Error())
		}

		// Hash password
		hashed, txErr := secHashPasswordWithPolicy(ctx, []byte(password), policy)
		if txErr != nil {
			return txErr
		}

		now := time.Now()
		// Create user. Email-verified state follows the tenant registration policy.
		newUser := &User{
			Username:          username,
			Fullname:          fullname,
			Password:          ptr.Ptr(string(hashed)),
			Status:            registrationInitialStatus(regPolicy, email),
			IsEmailVerified:   regPolicy.EmailVerified(),
			PasswordChangedAt: &now,
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

		// Record password history
		secpolicy.RecordPasswordHistory(s.passwordHistoryRepo, createdUser.UserID, policy.HistoryCount, string(hashed))

		// Create user identity
		userIdentity := &UserIdentity{
			TenantID:           tenantId,
			UserID:             createdUser.UserID,
			ClientID:           Client.ClientID,
			IdentityProviderID: &Client.IdentityProviderID,
			Provider:           shared.ProviderDefault,
			Sub:                uuid.New().String(),
			Metadata:           datatypes.JSON([]byte(`{}`)),
		}

		_, txErr = txUserIdentityRepo.Create(userIdentity)
		if txErr != nil {
			return txErr
		}
		userIdentitySub = userIdentity.Sub

		// Resolve role from policy default_role with safe fallback.
		defaultRole, txErr := s.resolveRegistrationRole(txRoleRepo, tenantId, regPolicy.DefaultRole)
		if txErr != nil {
			return txErr
		}

		// Assign default role to user
		userRole := &UserRole{
			UserID: createdUser.UserID,
			RoleID: defaultRole.RoleID,
		}
		_, txErr = txUserRoleRepo.Create(userRole)
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
	return s.generateTokenResponse(ctx, userIdentitySub, createdUser, Client)
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
) (*RegisterResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "register.invitePublic")
	defer span.End()
	span.SetAttributes(attribute.String("client.id", clientID), attribute.String("provider.id", providerID))

	var createdUser *User
	var Client *Client
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
			Client.Status != shared.StatusActive ||
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
		if invite.Status != shared.StatusPending {
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

		// Validate password against tenant policy
		policy := secpolicy.LoadPasswordPolicy(s.securitySettingRepo, tenantId)
		if txErr = secValidatePasswordPolicy(password, policy); txErr != nil {
			return apperror.NewValidation(txErr.Error())
		}

		// Hash password
		hashed, txErr := secHashPasswordWithPolicy(ctx, []byte(password), policy)
		if txErr != nil {
			return txErr
		}

		now := time.Now()
		// Create user
		newUser := &User{
			Username:           username,
			Email:              invite.InvitedEmail, // Always use the invited email
			Password:           ptr.Ptr(string(hashed)),
			Status:             shared.StatusActive,
			IsEmailVerified:    true,  // Auto-verify email for invited users
			IsProfileCompleted: false, // Require profile completion
			IsAccountCompleted: false, // Require account completion
			PasswordChangedAt:  &now,
		}

		createdUser, txErr = txUserRepo.Create(newUser)
		if txErr != nil {
			return txErr
		}

		// Record password history
		secpolicy.RecordPasswordHistory(s.passwordHistoryRepo, createdUser.UserID, policy.HistoryCount, string(hashed))

		// Create user identity
		userIdentity := &UserIdentity{
			TenantID:           tenantId,
			UserID:             createdUser.UserID,
			ClientID:           Client.ClientID,
			IdentityProviderID: &Client.IdentityProviderID,
			Provider:           shared.ProviderDefault,
			Sub:                uuid.New().String(),
			Metadata:           datatypes.JSON([]byte(`{}`)),
		}

		_, txErr = txUserIdentityRepo.Create(userIdentity)
		if txErr != nil {
			return txErr
		}
		userIdentitySub = userIdentity.Sub

		// Get default role and assign it first
		defaultRole, txErr := s.findDefaultRole(txRoleRepo, tenantId)
		if txErr != nil {
			return txErr
		}

		// Assign default role to user
		defaultUserRole := &UserRole{
			UserID: createdUser.UserID,
			RoleID: defaultRole.RoleID,
		}
		_, txErr = txUserRoleRepo.Create(defaultUserRole)
		if txErr != nil {
			return txErr
		}

		// If invite references an auth_flow, grant its roles
		if invite.AuthFlowID != nil && s.authFlowRoleRepo != nil {
			txAuthFlowRoleRepo := s.authFlowRoleRepo.WithTx(tx)
			roleIDs, txErr := txAuthFlowRoleRepo.FindRoleIDsByAuthFlowID(*invite.AuthFlowID)
			if txErr != nil {
				return txErr
			}
			for _, roleID := range roleIDs {
				if roleID == defaultRole.RoleID {
					continue // already assigned
				}
				existingRole, txErr := txUserRoleRepo.FindByUserIDAndRoleID(createdUser.UserID, roleID)
				if txErr != nil {
					return txErr
				}
				if existingRole != nil {
					continue
				}
				_, txErr = txUserRoleRepo.Create(&UserRole{
					UserID: createdUser.UserID,
					RoleID: roleID,
				})
				if txErr != nil {
					return txErr
				}
			}
		}

		// Mark invite as used
		txErr = txInviteRepo.MarkAsUsed(invite.InviteUUID)
		if txErr != nil {
			return txErr
		}

		return nil // commit
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "register invite public failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	// Return token response
	return s.generateTokenResponse(ctx, userIdentitySub, createdUser, Client)
}

func (s *registerService) generateTokenResponse(ctx context.Context, sub string, user *User, client *Client) (*RegisterResponseDTO, error) {
	policy := resolveEffectiveSessionPolicy(s.securitySettingRepo, client)
	accessToken, idToken, refreshToken, err := generateTokenSetWithAuthContext(ctx, sub, user, client, tokenAuthContextWithPolicy([]string{jwt.AMRPassword}, jwt.ACRLevel1, "", policy))
	if err != nil {
		return nil, err
	}
	resp := buildRegisterTokenResponse(accessToken, idToken, refreshToken, time.Now().Unix())
	applyRegisterCookiePolicy(resp, policy)
	if policy.AccessTokenTTLSeconds > 0 {
		resp.ExpiresIn = int64(policy.AccessTokenTTLSeconds)
	}
	return resp, nil
}

func registrationInitialStatus(policy *secpolicy.RegistrationPolicy, email *string) string {
	emailValue := ""
	if email != nil {
		emailValue = *email
	}
	if policy != nil && policy.InitialUserStatus(emailValue) == shared.StatusPending {
		return shared.StatusPending
	}
	return shared.StatusActive
}
