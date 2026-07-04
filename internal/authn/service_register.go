package authn

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
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
	RegisterPublic(ctx context.Context, username, fullname, password string, email, phone *string, clientID, tenantID *string, registrationFlowIdentifier string) (*RegisterResponseDTO, error)
	RegisterInvitePublic(ctx context.Context, username, password, clientID, tenantID, inviteToken string) (*RegisterResponseDTO, error)
	RegisterInvite(ctx context.Context, username, password string, clientID, tenantID *string, inviteToken string) (*RegisterResponseDTO, error)
	Register(ctx context.Context, username, fullname, password string, email, phone *string, clientID, tenantID *string, registrationFlowIdentifier string) (*RegisterResponseDTO, error)
}

type registerService struct {
	db                       *gorm.DB
	clientRepo               ClientRepository
	userRepo                 UserRepository
	userRoleRepo             UserRoleRepository
	userTokenRepo            UserTokenRepository
	userIdentityRepo         UserIdentityRepository
	roleRepo                 RoleRepository
	inviteRepo               InviteRepository
	identityProviderRepo     IdentityProviderRepository
	securitySettingRepo      secpolicy.SecuritySettingRepository // nil → use defaults
	passwordHistoryRepo      UserPasswordHistoryRepository       // nil → skip history
	registrationFlowRoleRepo RegistrationFlowRoleRepository
	emailVerificationSvc     EmailVerificationService
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
	registrationFlowRoleRepo RegistrationFlowRoleRepository,
	emailVerificationSvc ...EmailVerificationService,
) RegisterService {
	var emailSvc EmailVerificationService
	if len(emailVerificationSvc) > 0 {
		emailSvc = emailVerificationSvc[0]
	}

	return &registerService{
		db:                       db,
		clientRepo:               clientRepo,
		userRepo:                 userRepo,
		userRoleRepo:             userRoleRepo,
		userTokenRepo:            userTokenRepo,
		userIdentityRepo:         userIdentityRepo,
		roleRepo:                 roleRepo,
		inviteRepo:               inviteRepo,
		identityProviderRepo:     identityProviderRepo,
		securitySettingRepo:      securitySettingRepo,
		passwordHistoryRepo:      passwordHistoryRepo,
		registrationFlowRoleRepo: registrationFlowRoleRepo,
		emailVerificationSvc:     emailSvc,
	}
}

// Helper function to find the system default registration role for a tenant.
func (s *registerService) findDefaultRole(roleRepo RoleRepository, tenantID int64) (*Role, error) {
	// Registration always assigns the system "registered" role. The fallback to
	// an is_default role keeps legacy/unseeded tenants from blocking signup.
	role, err := roleRepo.FindByNameAndTenantID(shared.RoleRegistered, tenantID)
	if err != nil {
		return nil, err
	}
	if role != nil {
		return role, nil
	}

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

	return nil, apperror.NewValidation("registered role not found for tenant")
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

func enforceIdentityProviderRegistrationGate(client *Client) error {
	if client == nil || client.ConnectedProviders == nil {
		return nil
	}
	for _, connection := range *client.ConnectedProviders {
		provider := connection.IdentityProvider
		if !connection.Enabled || provider == nil || (!provider.IsSystem && provider.ProviderType != shared.IDPTypeSystem) {
			continue
		}
		if provider.Status != shared.StatusActive || !provider.AllowRegistration {
			return apperror.NewForbidden("registration is disabled for this identity provider")
		}
		return nil
	}
	// A client without an in-house provider has no per-IdP registration gate.
	return nil
}

func (s *registerService) registrationFlowByIdentifier(tx *gorm.DB, clientID int64, identifier string) (*RegistrationFlow, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, nil
	}
	if s.registrationFlowRoleRepo == nil {
		return nil, apperror.NewInternal("registration flow repository is unavailable", nil)
	}
	txRepo := s.registrationFlowRoleRepo.WithTx(tx)
	flow, err := txRepo.FindByIdentifierAndClientID(identifier, clientID)
	if err != nil {
		return nil, apperror.NewInternal("failed to load registration flow", err)
	}
	if flow == nil {
		return nil, apperror.NewNotFoundWithReason("registration flow not found for this client")
	}
	if flow.Status != shared.StatusActive {
		return nil, apperror.NewForbidden("registration flow is inactive")
	}
	return flow, nil
}

func (s *registerService) validateInviteRegistrationFlow(tx *gorm.DB, invite *Invite) (*RegistrationFlow, error) {
	if invite == nil || invite.RegistrationFlowID == nil {
		return nil, nil
	}
	if s.registrationFlowRoleRepo == nil {
		return nil, apperror.NewInternal("registration flow repository is unavailable", nil)
	}
	flow, err := s.registrationFlowRoleRepo.WithTx(tx).FindByID(*invite.RegistrationFlowID)
	if err != nil {
		return nil, apperror.NewInternal("failed to load invite registration flow", err)
	}
	if flow == nil || flow.TenantID != invite.TenantID {
		return nil, apperror.NewUnauthorized("invite registration flow is invalid")
	}
	if flow.Status != shared.StatusActive {
		return nil, apperror.NewUnauthorized("invite registration flow is inactive")
	}
	return flow, nil
}

func parseRequiredRegistrationFields(raw datatypes.JSON) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var fields []string
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, apperror.NewValidation("registration flow required_fields must be a JSON string array")
	}
	return fields, nil
}

func enforceRequiredRegistrationFields(flow *RegistrationFlow, fullname string, email, phone *string) error {
	if flow == nil {
		return nil
	}
	fields, err := parseRequiredRegistrationFields(flow.RequiredFields)
	if err != nil {
		return err
	}
	for _, field := range fields {
		switch strings.ToLower(strings.TrimSpace(field)) {
		case "", "username", "password":
			// Username and password are always required by RegisterRequestDTO.
		case "fullname":
			if strings.TrimSpace(fullname) == "" {
				return apperror.NewValidation("fullname is required by the registration flow")
			}
		case "email":
			if email == nil || strings.TrimSpace(*email) == "" {
				return apperror.NewValidation("email is required by the registration flow")
			}
		case "phone":
			if phone == nil || strings.TrimSpace(*phone) == "" {
				return apperror.NewValidation("phone is required by the registration flow")
			}
		default:
			return apperror.NewValidation("unsupported registration flow required field: " + field)
		}
	}
	return nil
}

func effectiveRegistrationPolicy(base *secpolicy.RegistrationPolicy, flow *RegistrationFlow) *secpolicy.RegistrationPolicy {
	effective := *base
	if flow != nil && flow.VerificationRequired {
		effective.RequireEmailVerification = true
	}
	return &effective
}

func (s *registerService) assignRegistrationFlowRoles(tx *gorm.DB, userRoleRepo UserRoleRepository, userID, defaultRoleID int64, flow *RegistrationFlow) error {
	if flow == nil || s.registrationFlowRoleRepo == nil {
		return nil
	}
	roleIDs, err := s.registrationFlowRoleRepo.WithTx(tx).FindRoleIDsByRegistrationFlowID(flow.RegistrationFlowID)
	if err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		if roleID == defaultRoleID {
			continue
		}
		existing, err := userRoleRepo.FindByUserIDAndRoleID(userID, roleID)
		if err != nil {
			return err
		}
		if existing != nil {
			continue
		}
		if _, err := userRoleRepo.Create(&UserRole{UserID: userID, RoleID: roleID}); err != nil {
			return err
		}
	}
	return nil
}

// RegisterPublic registers new users for public-facing applications.  clientID
// and tenantID are optional (pointer params).  When omitted the system client
// is used.  When clientID is set the tenant is derived from the client record
// (clients.identifier is globally unique); when only tenantID is set the
// is_system client under that tenant is selected.
// Used on port 8081 (public).
func (s *registerService) RegisterPublic(
	ctx context.Context,
	username,
	fullname,
	password string,
	email,
	phone *string,
	clientID,
	tenantID *string,
	registrationFlowIdentifier string,
) (*RegisterResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "register.public")
	defer span.End()
	if clientID != nil {
		span.SetAttributes(attribute.String("client.id", *clientID))
	}
	if tenantID != nil {
		span.SetAttributes(attribute.String("tenant.id", *tenantID))
	}

	// Rate limiting check to prevent registration abuse
	if err := security.CheckRateLimit(username); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "register public failed")
		return nil, err
	}

	var createdUser *User
	var Client *Client
	var userIdentitySub string
	var needEmailVerification bool
	var registrationFlow *RegistrationFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		var txErr error
		Client, txErr = resolvePublicClient(txClientRepo, clientID, tenantID)
		if txErr != nil {
			return txErr
		}
		if Client == nil ||
			Client.Status != shared.StatusActive ||
			Client.Domain == nil || *Client.Domain == "" {
			return apperror.NewValidation("invalid or inactive auth client")
		}

		tenantId := clientTenantID(Client)
		if tenantId == 0 {
			return apperror.NewValidation("auth client tenant could not be resolved")
		}

		// Enforce tenant registration policy (self-service path).
		regPolicy := secpolicy.LoadRegistrationPolicy(s.securitySettingRepo, tenantId)
		if !regPolicy.SelfRegistrationEnabled {
			return apperror.NewForbidden("self-registration is disabled for this tenant")
		}
		if !Client.AllowRegistration {
			return apperror.NewForbidden("self-registration is disabled for this client")
		}
		if err := enforceIdentityProviderRegistrationGate(Client); err != nil {
			return err
		}
		registrationFlow, txErr = s.registrationFlowByIdentifier(tx, Client.ClientID, registrationFlowIdentifier)
		if txErr != nil {
			return txErr
		}
		if txErr = enforceRequiredRegistrationFields(registrationFlow, fullname, email, phone); txErr != nil {
			return txErr
		}
		regPolicy = effectiveRegistrationPolicy(regPolicy, registrationFlow)
		needEmailVerification = regPolicy.RequireEmailVerification
		if registrationFlow != nil && registrationFlow.VerificationRequired && (email == nil || strings.TrimSpace(*email) == "") {
			return apperror.NewValidation("email is required when signup verification is enabled")
		}
		if err := enforceRegistrationAbuseControls(ctx, tenantId, regPolicy); err != nil {
			return err
		}
		if email != nil && *email != "" && !regPolicy.EmailDomainAllowed(*email) {
			return apperror.NewValidation("email domain is not permitted for registration")
		}
		if regPolicy.RequirePhoneVerification && (phone == nil || *phone == "") {
			return apperror.NewValidation("phone number is required for registration")
		}

		existingUser, txErr := txUserRepo.FindByUsernameAndTenantID(username, tenantId)
		if txErr != nil {
			return txErr
		}
		if existingUser != nil {
			return apperror.NewConflict("username already taken")
		}

		// Account-enumeration hardening (H8): on the public self-service surface,
		// do not disclose which PII field (email or phone) already exists. Return a
		// single generic conflict so an attacker cannot confirm a specific email or
		// phone is registered. (Username availability is intentionally still explicit
		// above, since usernames are user-chosen identifiers, not PII.)
		// NOTE: full non-enumeration (identical response for new vs existing) requires
		// a verification-first registration flow and is a separate product decision.
		if email != nil && *email != "" {
			existingEmailUser, txErr := txUserRepo.FindByEmailAndTenantID(*email, tenantId)
			if txErr != nil {
				return txErr
			}
			if existingEmailUser != nil {
				return apperror.NewConflict("registration could not be completed with the provided details")
			}
		}

		if phone != nil && *phone != "" {
			existingPhoneUser, txErr := txUserRepo.FindByPhoneAndTenantID(*phone, tenantId)
			if txErr != nil {
				return txErr
			}
			if existingPhoneUser != nil {
				return apperror.NewConflict("registration could not be completed with the provided details")
			}
		}

		policy := secpolicy.LoadPasswordPolicy(s.securitySettingRepo, tenantId)
		if txErr = secValidatePasswordPolicy(password, policy); txErr != nil {
			return apperror.NewValidation(txErr.Error())
		}

		hashed, txErr := secHashPasswordWithPolicy(ctx, []byte(password), policy)
		if txErr != nil {
			return txErr
		}

		now := time.Now()
		newUser := &User{
			TenantID:          tenantId,
			Username:          username,
			Fullname:          fullname,
			Password:          ptr.Ptr(string(hashed)),
			Status:            registrationInitialStatus(regPolicy, email),
			IsEmailVerified:   regPolicy.EmailVerified(),
			IsPhoneVerified:   false,
			PasswordChangedAt: &now,
		}

		if email != nil && *email != "" {
			newUser.Email = *email
		}

		if phone != nil && *phone != "" {
			newUser.Phone = *phone
		}

		createdUser, txErr = txUserRepo.Create(newUser)
		if txErr != nil {
			return txErr
		}

		secpolicy.RecordPasswordHistory(s.passwordHistoryRepo, createdUser.UserID, policy.HistoryCount, string(hashed))

		userIdentity := &UserIdentity{
			TenantID:           tenantId,
			UserID:             createdUser.UserID,
			ClientID:           Client.ClientID,
			IdentityProviderID: clientIdentityProviderIDPtr(Client),
			Provider:           shared.ProviderMaintainerd,
			Sub:                uuid.New().String(),
			Metadata:           datatypes.JSON([]byte(`{}`)),
		}

		_, txErr = txUserIdentityRepo.Create(userIdentity)
		if txErr != nil {
			return txErr
		}
		userIdentitySub = userIdentity.Sub

		defaultRole, txErr := s.findDefaultRole(txRoleRepo, tenantId)
		if txErr != nil {
			return txErr
		}

		userRole := &UserRole{
			UserID: createdUser.UserID,
			RoleID: defaultRole.RoleID,
		}
		_, txErr = txUserRoleRepo.Create(userRole)
		if txErr != nil {
			return txErr
		}
		if txErr = s.assignRegistrationFlowRoles(tx, txUserRoleRepo, createdUser.UserID, defaultRole.RoleID, registrationFlow); txErr != nil {
			return txErr
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "register public failed")
		return nil, err
	}

	if email != nil && *email != "" && needEmailVerification && s.emailVerificationSvc != nil {
		if _, err := s.emailVerificationSvc.SendVerificationEmail(ctx, *email, clientID, tenantID); err != nil {
			slog.Warn("failed to send verification email during registration", "email", *email, "err", err)
		}
	}

	span.SetStatus(codes.Ok, "")
	return s.generateTokenResponse(ctx, userIdentitySub, createdUser, Client)
}

// Register registers new users for internal applications.
// clientID and tenantID are optional (pointer params).  When omitted the
// system client is used.  Resolution priority: clientID > tenantID > default.
// Used by internal applications on port 8080.
func (s *registerService) Register(
	ctx context.Context,
	username,
	fullname,
	password string,
	email,
	phone *string,
	clientID,
	tenantID *string,
	registrationFlowIdentifier string,
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
	var needEmailVerification bool
	var registrationFlow *RegistrationFlow

	// All database operations in transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		var txErr error
		Client, txErr = resolveClient(txClientRepo, clientID, tenantID)
		if txErr != nil {
			return txErr
		}

		if Client == nil ||
			Client.Status != shared.StatusActive ||
			Client.Domain == nil || *Client.Domain == "" {
			return apperror.NewNotFoundWithReason("auth client not found or inactive")
		}

		tenantId := clientTenantID(Client)
		if tenantId == 0 {
			return apperror.NewValidation("auth client tenant could not be resolved")
		}

		// Enforce tenant registration policy.
		regPolicy := secpolicy.LoadRegistrationPolicy(s.securitySettingRepo, tenantId)
		if !regPolicy.SelfRegistrationEnabled {
			return apperror.NewForbidden("self-registration is disabled for this tenant")
		}
		if !Client.AllowRegistration {
			return apperror.NewForbidden("self-registration is disabled for this client")
		}
		if err := enforceIdentityProviderRegistrationGate(Client); err != nil {
			return err
		}
		registrationFlow, txErr = s.registrationFlowByIdentifier(tx, Client.ClientID, registrationFlowIdentifier)
		if txErr != nil {
			return txErr
		}
		if txErr = enforceRequiredRegistrationFields(registrationFlow, fullname, email, phone); txErr != nil {
			return txErr
		}
		regPolicy = effectiveRegistrationPolicy(regPolicy, registrationFlow)
		needEmailVerification = regPolicy.RequireEmailVerification
		if registrationFlow != nil && registrationFlow.VerificationRequired && (email == nil || strings.TrimSpace(*email) == "") {
			return apperror.NewValidation("email is required when signup verification is enabled")
		}
		if err := enforceRegistrationAbuseControls(ctx, tenantId, regPolicy); err != nil {
			return err
		}
		if email != nil && *email != "" && !regPolicy.EmailDomainAllowed(*email) {
			return apperror.NewValidation("email domain is not permitted for registration")
		}
		if regPolicy.RequirePhoneVerification && (phone == nil || *phone == "") {
			return apperror.NewValidation("phone number is required for registration")
		}

		// Check if user already exists (scoped to this tenant)
		existingUser, txErr := txUserRepo.FindByUsernameAndTenantID(username, tenantId)
		if txErr != nil && txErr.Error() != "record not found" {
			return txErr
		}
		if existingUser != nil {
			return apperror.NewConflict("user already exists")
		}
		if email != nil && strings.TrimSpace(*email) != "" {
			existingEmailUser, lookupErr := txUserRepo.FindByEmailAndTenantID(*email, tenantId)
			if lookupErr != nil {
				return lookupErr
			}
			if existingEmailUser != nil {
				return apperror.NewConflict("email already registered")
			}
		}
		if phone != nil && strings.TrimSpace(*phone) != "" {
			existingPhoneUser, lookupErr := txUserRepo.FindByPhoneAndTenantID(*phone, tenantId)
			if lookupErr != nil {
				return lookupErr
			}
			if existingPhoneUser != nil {
				return apperror.NewConflict("phone already registered")
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
		// Email-verified state follows the tenant registration policy.
		newUser := &User{
			TenantID:          tenantId,
			Username:          username,
			Fullname:          fullname,
			Password:          ptr.Ptr(string(hashed)),
			Status:            registrationInitialStatus(regPolicy, email),
			IsEmailVerified:   regPolicy.EmailVerified(),
			IsPhoneVerified:   false,
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
			IdentityProviderID: clientIdentityProviderIDPtr(Client),
			Provider:           shared.ProviderMaintainerd,
			Sub:                uuid.New().String(),
			Metadata:           datatypes.JSON([]byte(`{}`)),
		}

		_, txErr = txUserIdentityRepo.Create(userIdentity)
		if txErr != nil {
			return txErr
		}
		userIdentitySub = userIdentity.Sub

		// Assign the system default registration role.
		defaultRole, txErr := s.findDefaultRole(txRoleRepo, tenantId)
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
		if txErr = s.assignRegistrationFlowRoles(tx, txUserRoleRepo, createdUser.UserID, defaultRole.RoleID, registrationFlow); txErr != nil {
			return txErr
		}

		return nil // commit transaction
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "register failed")
		return nil, err
	}

	// Trigger email verification if the tenant policy requires it.
	if email != nil && *email != "" && needEmailVerification && s.emailVerificationSvc != nil {
		if _, err := s.emailVerificationSvc.SendVerificationEmail(ctx, *email, clientID, tenantID); err != nil {
			slog.Warn("failed to send verification email during registration", "email", *email, "err", err)
		}
	}

	span.SetStatus(codes.Ok, "")
	// Return token response
	return s.generateTokenResponse(ctx, userIdentitySub, createdUser, Client)
}

// RegisterInvitePublic registers new users via invite token for public-facing applications.
// clientID and tenantID identify the auth client (resolution priority: clientID > tenantID).
// When both are empty the system client is used.
// Used by external applications on port 8081.
func (s *registerService) RegisterInvitePublic(
	ctx context.Context,
	username,
	password,
	clientID,
	tenantID,
	inviteToken string,
) (*RegisterResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "register.invitePublic")
	defer span.End()
	span.SetAttributes(attribute.String("client.id", clientID))

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
		txRoleRepo := s.roleRepo.WithTx(tx)

		var txErr error
		Client, txErr = resolvePublicClient(txClientRepo, stringPtrOrNil(clientID), stringPtrOrNil(tenantID))
		if txErr != nil {
			return txErr
		}
		if Client == nil ||
			Client.Status != shared.StatusActive ||
			Client.Domain == nil || *Client.Domain == "" {
			return apperror.NewValidation("invalid or inactive auth client")
		}

		tenantId := clientTenantID(Client)
		if tenantId == 0 {
			return apperror.NewValidation("auth client tenant could not be resolved")
		}

		// Validate invite token
		invite, txErr := txInviteRepo.FindByTokenForUpdate(inviteToken)
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
		if invite.TenantID == 0 || invite.TenantID != tenantId {
			return apperror.NewUnauthorized("invite does not belong to the auth client tenant")
		}
		inviteFlow, txErr := s.validateInviteRegistrationFlow(tx, invite)
		if txErr != nil {
			return txErr
		}

		// Check if username already exists (scoped to the invite's tenant)
		existingUser, txErr := txUserRepo.FindByUsernameAndTenantID(username, tenantId)
		if txErr != nil {
			return txErr
		}
		if existingUser != nil {
			return apperror.NewConflict("username already taken")
		}

		// Check if invited email already exists (scoped to the invite's tenant)
		existingEmailUser, txErr := txUserRepo.FindByEmailAndTenantID(invite.InvitedEmail, tenantId)
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
			TenantID:           tenantId,
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

		// Auto-enroll email OTP MFA — the invite link already proves email ownership.
		// Best-effort: if it fails (e.g. table not ready), registration still succeeds.
		_ = tx.Exec(
			`INSERT INTO user_mfa_emails (mfa_email_uuid, user_id, email, is_verified, verified_at, created_at, updated_at) VALUES (gen_random_uuid(), ?, ?, true, ?, ?, ?) ON CONFLICT (user_id) DO NOTHING`,
			createdUser.UserID, invite.InvitedEmail, now, now, now,
		).Error

		// Record password history
		secpolicy.RecordPasswordHistory(s.passwordHistoryRepo, createdUser.UserID, policy.HistoryCount, string(hashed))

		// Create user identity
		userIdentity := &UserIdentity{
			TenantID:           tenantId,
			UserID:             createdUser.UserID,
			ClientID:           Client.ClientID,
			IdentityProviderID: clientIdentityProviderIDPtr(Client),
			Provider:           shared.ProviderMaintainerd,
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

		if txErr = s.assignRegistrationFlowRoles(tx, txUserRoleRepo, createdUser.UserID, defaultRole.RoleID, inviteFlow); txErr != nil {
			return txErr
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

// RegisterInvite registers new users via invite token for internal applications.
// Unlike RegisterInvitePublic, clientID and tenantID are optional (pointer params).
// When nil, the system client is resolved from the invite's tenant.
// Used by internal applications on port 8080.
func (s *registerService) RegisterInvite(
	ctx context.Context,
	username,
	password string,
	clientID,
	tenantID *string,
	inviteToken string,
) (*RegisterResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "register.invite")
	defer span.End()

	var createdUser *User
	var Client *Client
	var userIdentitySub string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txInviteRepo := s.inviteRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)

		invite, txErr := txInviteRepo.FindByTokenForUpdate(inviteToken)
		if txErr != nil {
			return apperror.NewUnauthorized("invalid invite token")
		}
		if invite == nil {
			return apperror.NewNotFound("invite not found")
		}

		if invite.Status != shared.StatusPending {
			return apperror.NewUnauthorized("invite has already been used or is no longer valid")
		}
		if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
			return apperror.NewUnauthorized("invite has expired")
		}

		if clientID != nil || tenantID != nil {
			Client, txErr = resolveClient(txClientRepo, clientID, tenantID)
			if txErr != nil {
				return txErr
			}
		} else {
			Client, txErr = txClientRepo.FindSystem()
			if txErr != nil {
				return txErr
			}
		}

		if Client == nil ||
			Client.Status != shared.StatusActive ||
			Client.Domain == nil || *Client.Domain == "" {
			return apperror.NewValidation("invalid or inactive auth client")
		}
		if invite.TenantID == 0 || clientTenantID(Client) != invite.TenantID {
			return apperror.NewUnauthorized("invite does not belong to the auth client tenant")
		}
		inviteFlow, txErr := s.validateInviteRegistrationFlow(tx, invite)
		if txErr != nil {
			return txErr
		}

		tenantId := invite.TenantID

		existingUser, txErr := txUserRepo.FindByUsernameAndTenantID(username, tenantId)
		if txErr != nil {
			return txErr
		}
		if existingUser != nil {
			return apperror.NewConflict("username already taken")
		}

		existingEmailUser, txErr := txUserRepo.FindByEmailAndTenantID(invite.InvitedEmail, tenantId)
		if txErr != nil {
			return txErr
		}
		if existingEmailUser != nil {
			return apperror.NewConflict("invited email already registered")
		}

		policy := secpolicy.LoadPasswordPolicy(s.securitySettingRepo, tenantId)
		if txErr = secValidatePasswordPolicy(password, policy); txErr != nil {
			return apperror.NewValidation(txErr.Error())
		}

		hashed, txErr := secHashPasswordWithPolicy(ctx, []byte(password), policy)
		if txErr != nil {
			return txErr
		}

		now := time.Now()
		newUser := &User{
			TenantID:           tenantId,
			Username:           username,
			Email:              invite.InvitedEmail,
			Password:           ptr.Ptr(string(hashed)),
			Status:             shared.StatusActive,
			IsEmailVerified:    true,
			IsProfileCompleted: false,
			IsAccountCompleted: false,
			PasswordChangedAt:  &now,
		}

		createdUser, txErr = txUserRepo.Create(newUser)
		if txErr != nil {
			return txErr
		}

		secpolicy.RecordPasswordHistory(s.passwordHistoryRepo, createdUser.UserID, policy.HistoryCount, string(hashed))

		userIdentity := &UserIdentity{
			TenantID:           tenantId,
			UserID:             createdUser.UserID,
			ClientID:           Client.ClientID,
			IdentityProviderID: clientIdentityProviderIDPtr(Client),
			Provider:           shared.ProviderMaintainerd,
			Sub:                uuid.New().String(),
			Metadata:           datatypes.JSON([]byte(`{}`)),
		}

		_, txErr = txUserIdentityRepo.Create(userIdentity)
		if txErr != nil {
			return txErr
		}
		userIdentitySub = userIdentity.Sub

		defaultRole, txErr := s.findDefaultRole(txRoleRepo, tenantId)
		if txErr != nil {
			return txErr
		}

		defaultUserRole := &UserRole{
			UserID: createdUser.UserID,
			RoleID: defaultRole.RoleID,
		}
		_, txErr = txUserRoleRepo.Create(defaultUserRole)
		if txErr != nil {
			return txErr
		}

		if txErr = s.assignRegistrationFlowRoles(tx, txUserRoleRepo, createdUser.UserID, defaultRole.RoleID, inviteFlow); txErr != nil {
			return txErr
		}

		txErr = txInviteRepo.MarkAsUsed(invite.InviteUUID)
		if txErr != nil {
			return txErr
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "register invite failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.generateTokenResponse(ctx, userIdentitySub, createdUser, Client)
}

func (s *registerService) generateTokenResponse(ctx context.Context, sub string, user *User, client *Client) (*RegisterResponseDTO, error) {
	policy := resolveEffectiveSessionPolicy(s.securitySettingRepo, client)
	tokenPolicy := resolveEffectiveTokenPolicy(s.securitySettingRepo, client)
	accessToken, idToken, refreshToken, err := generateTokenSetWithAuthContext(ctx, sub, user, client, tokenAuthContextWithPolicy([]string{jwt.AMRPassword}, jwt.ACRLevel1, "", policy, tokenPolicy))
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
