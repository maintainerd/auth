package authn

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"sync"
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

// Context-aware: the HIBP breach check inside makes an outbound call, so it must
// join the request trace and honour the request's cancellation. The non-context
// wrapper hardcodes context.Background(), which detached the span and made the
// fail-open warning unattributable to a request.
var secValidatePasswordPolicy = security.ValidatePasswordPolicyWithContext
var secHashPassword = security.HashPassword
var secHashPasswordWithPolicy = security.HashPasswordWithPolicy

// A seam, so a test can prove the verifier is never reached when no provider is
// configured. security.VerifyCaptcha currently returns nil for an unset
// CAPTCHA_SECRET; that fail-open is scheduled for removal, and registration must
// not be the thing that discovers it. Only captchaProviderConfigured decides
// "no provider" here, so the removal is a no-op for this path.
var secVerifyCaptcha = security.VerifyCaptcha

// RegisterService is the registration surface actually mounted by the router
// (RegisterPublicRoute). The former internal variants — Register and
// RegisterInvite, mounted only by a RegisterRoute nothing ever called — are
// gone: they were unreachable line-for-line duplicates of these two that still
// had to be kept in sync, so a fix applied to the live path could silently miss
// the copy.
type RegisterService interface {
	RegisterPublic(ctx context.Context, username, fullname, password string, email, phone *string, clientID, tenantID *string, registrationFlowName string) (*RegisterResponseDTO, error)
	RegisterInvitePublic(ctx context.Context, username, password, clientID, tenantID, inviteToken string) (*RegisterResponseDTO, error)
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
	consentRecorder          UserConsentRecorder
	// sessionService creates the user_sessions row for a newly registered user.
	// Nil only in tests that never mint tokens.
	sessionService SessionService
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
	opts ...RegisterServiceOption,
) RegisterService {
	s := &registerService{
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
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type RegisterServiceOption func(*registerService)

func WithEmailVerificationService(svc EmailVerificationService) RegisterServiceOption {
	return func(s *registerService) { s.emailVerificationSvc = svc }
}

func WithConsentRecorder(r UserConsentRecorder) RegisterServiceOption {
	return func(s *registerService) { s.consentRecorder = r }
}

// WithRegisterSessionService gives registration the same session handling as
// login. Registration signs the user in, so it must create a session like every
// other authentication path — see generateTokenResponse.
func WithRegisterSessionService(svc SessionService) RegisterServiceOption {
	return func(s *registerService) { s.sessionService = svc }
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

// captchaProviderConfigured reports whether this deployment has a CAPTCHA
// verifier wired up. It repeats the platform verifier's own environment lookup
// because internal/platform/security exposes no predicate for it — a
// security.CaptchaConfigured() helper there is the DRY home for this, and this
// copy should collapse into it. It is a variable so a test can pin the
// "no provider" shape regardless of what the ambient environment holds.
var captchaProviderConfigured = func() bool {
	return strings.TrimSpace(os.Getenv("CAPTCHA_SECRET")) != ""
}

// warnedCaptchaUnenforceableTenants keeps the "policy asks for CAPTCHA,
// deployment has no provider" notice to one line per tenant per process. It is a
// standing misconfiguration, not a per-request event, so logging it on every
// signup would bury real signal under duplicates — but a single process-wide
// sync.Once would hide every tenant after the first, and this warning is the
// only signal an operator gets that a tenant's CAPTCHA policy is not being
// applied.
var warnedCaptchaUnenforceableTenants sync.Map

func warnCaptchaUnenforceable(tenantID int64) {
	if _, alreadyWarned := warnedCaptchaUnenforceableTenants.LoadOrStore(tenantID, struct{}{}); alreadyWarned {
		return
	}
	slog.Warn("captcha_on_signup is enabled but no CAPTCHA provider is configured; the signup CAPTCHA check is NOT enforced",
		"tenant_id", tenantID)
}

func enforceRegistrationAbuseControls(ctx context.Context, tenantID int64, regPolicy *secpolicy.RegistrationPolicy) error {
	if regPolicy == nil {
		return nil
	}
	if regPolicy.CaptchaOnSignup {
		// The tenant flag alone does not enforce: a CAPTCHA provider must also be
		// configured for this deployment. CAPTCHA is deferred and has no
		// client-side half — no first-party signup form emits a captcha_token —
		// so the flag can be on for a tenant whose deployment has no verifier at
		// all. Denying there would reject 100% of self-service registration on a
		// control that could never have been satisfied, and tenants already carry
		// a persisted captcha_on_signup=true that lowering the seeded default does
		// not clear (migrations are create-only, so nothing backfills the rows).
		// This is the single deliberate non-denial in this function, and it is
		// logged per tenant rather than swallowed; the moment a verifier exists
		// the branch below denies on anything short of a provider pass.
		if !captchaProviderConfigured() {
			warnCaptchaUnenforceable(tenantID)
		} else {
			// With a provider configured the check is strict: a missing token, a
			// provider rejection, and a provider/network error all mean "not
			// proven human" and all reject.
			captchaToken := registrationCaptchaTokenFromContext(ctx)
			clientIP := middleware.ClientIPFromContext(ctx)
			if err := secVerifyCaptcha(ctx, captchaToken, clientIP); err != nil {
				slog.Warn("registration rejected: signup CAPTCHA not verified", "tenant_id", tenantID, "err", err)
				return apperror.NewValidation("captcha verification failed")
			}
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

// registrationFlowByName resolves the flow named by a public registration link.
// This is the highest-privilege unauthenticated path in the service — the
// resolved flow decides which roles the new user is granted — so it is scoped by
// client AND tenant, and system flows are refused outright.
func (s *registerService) registrationFlowByName(tx *gorm.DB, clientID, tenantID int64, name string) (*RegistrationFlow, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return nil, nil
	}
	if s.registrationFlowRoleRepo == nil {
		return nil, apperror.NewInternal("registration flow repository is unavailable", nil)
	}
	txRepo := s.registrationFlowRoleRepo.WithTx(tx)
	// Tenant is part of the predicate, not an afterthought: matching the client
	// alone proves the flow exists, not that it belongs to the tenant this
	// request resolved to.
	flow, err := txRepo.FindByNameAndClientTenant(name, clientID, tenantID)
	if err != nil {
		return nil, apperror.NewInternal("failed to load registration flow", err)
	}
	if flow == nil || flow.TenantID != tenantID {
		return nil, apperror.NewNotFoundWithReason("registration flow not found for this client")
	}
	// System flows (e.g. owner onboarding, which grants super-admin) are
	// invite-only by construction: an invite binds its flow by internal id, so a
	// self-service link must never be able to redeem one. Reported as not-found
	// so the endpoint does not confirm that a system flow exists.
	if flow.IsSystem {
		return nil, apperror.NewNotFoundWithReason("registration flow not found for this client")
	}
	// Inactive is reported as not-found, identically to unknown/wrong-tenant/
	// system. Status is the operator's kill switch for a published link, so a
	// distinguishable "exists but disabled" response would let whoever holds a
	// leaked link poll until the switch is lifted — and would confirm which flow
	// names exist. /oauth/authorize already collapses these for the same reason.
	if flow.Status != shared.StatusActive {
		return nil, apperror.NewNotFoundWithReason("registration flow not found for this client")
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
	// Tenant comes from the flow itself, which the caller already verified against
	// the resolved client's tenant. Roles that are cross-tenant, inactive, system
	// or administrative are filtered out here — the authoritative grant cap.
	roleIDs, err := s.registrationFlowRoleRepo.WithTx(tx).
		FindGrantableRoleIDsByRegistrationFlowID(flow.RegistrationFlowID, flow.TenantID)
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
	registrationFlowName string,
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
		Client, txErr = resolvePublicClient(ctx, txClientRepo, clientID, tenantID)
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
		registrationFlow, txErr = s.registrationFlowByName(tx, Client.ClientID, tenantId, registrationFlowName)
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
		if txErr = secValidatePasswordPolicy(ctx, password, policy); txErr != nil {
			return apperror.NewValidation(txErr.Error())
		}

		hashed, txErr := secHashPasswordWithPolicy(ctx, []byte(password), policy)
		if txErr != nil {
			return txErr
		}

		now := time.Now()
		newUser := &User{
			TenantID: tenantId,
			Username: username,
			Fullname: fullname,
			Password: ptr.Ptr(string(hashed)),
			Status:   registrationInitialStatus(regPolicy, email),
			// email_verified reflects PROVEN control, never policy. A tenant that
			// auto-confirms or does not require verification still has an
			// unproven address, so this stays false until the address is actually
			// confirmed via the verification flow (OIDC Core §5.1). Account
			// activation is handled separately by registrationInitialStatus above.
			IsEmailVerified:   false,
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

		// Use the transaction-scoped repo: the base repo writes on a separate
		// connection where the user row is not yet committed, so the
		// user_password_history FK fails and the entry is lost. Returning the
		// error rolls the whole registration back rather than creating a user
		// whose first password is not in their reuse history.
		if s.passwordHistoryRepo != nil {
			if txErr = secpolicy.RecordPasswordHistory(s.passwordHistoryRepo.WithTx(tx), createdUser.UserID, policy.HistoryCount, string(hashed)); txErr != nil {
				return apperror.NewInternal("failed to record password history", txErr)
			}
		}

		identityProviderID, idpErr := clientIdentityProviderID(Client)
		if idpErr != nil {
			return idpErr
		}
		userIdentity := &UserIdentity{
			TenantID:           tenantId,
			UserID:             createdUser.UserID,
			IdentityProviderID: identityProviderID,
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
		if s.consentRecorder != nil {
			_ = s.consentRecorder.Record(ctx, tx, createdUser.UserID, tenantId, "terms_of_service", "1.0", middleware.ClientIPFromContext(ctx), "")
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
		Client, txErr = resolvePublicClient(ctx, txClientRepo, stringPtrOrNil(clientID), stringPtrOrNil(tenantID))
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

		// blocked_email_domains is a hard org policy and must hold even for
		// invites — an invite is an explicit grant that legitimately overrides the
		// self-signup allowlist and self_registration_enabled, but it must not be
		// able to provision an identity at a domain the tenant has hard-blocked.
		if secpolicy.LoadRegistrationPolicy(s.securitySettingRepo, tenantId).EmailDomainBlocked(invite.InvitedEmail) {
			return apperror.NewValidation("email domain is not permitted")
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
		if txErr = secValidatePasswordPolicy(ctx, password, policy); txErr != nil {
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
			TenantID:          tenantId,
			Username:          username,
			Email:             invite.InvitedEmail, // Always use the invited email
			Password:          ptr.Ptr(string(hashed)),
			Status:            shared.StatusActive,
			IsEmailVerified:   true, // Auto-verify email for invited users
			PasswordChangedAt: &now,
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
		// Use the transaction-scoped repo: the base repo writes on a separate
		// connection where the user row is not yet committed, so the
		// user_password_history FK fails and the entry is lost. Returning the
		// error rolls the whole registration back rather than creating a user
		// whose first password is not in their reuse history.
		if s.passwordHistoryRepo != nil {
			if txErr = secpolicy.RecordPasswordHistory(s.passwordHistoryRepo.WithTx(tx), createdUser.UserID, policy.HistoryCount, string(hashed)); txErr != nil {
				return apperror.NewInternal("failed to record password history", txErr)
			}
		}

		// Create user identity
		identityProviderID, idpErr := clientIdentityProviderID(Client)
		if idpErr != nil {
			return idpErr
		}
		userIdentity := &UserIdentity{
			TenantID:           tenantId,
			UserID:             createdUser.UserID,
			IdentityProviderID: identityProviderID,
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
		if s.consentRecorder != nil {
			_ = s.consentRecorder.Record(ctx, tx, createdUser.UserID, tenantId, "terms_of_service", "1.0", middleware.ClientIPFromContext(ctx), "")
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
	tokenPolicy := resolveEffectiveTokenPolicy(s.securitySettingRepo, client)

	// Registration signs the user in, so it must create a session exactly like
	// login and SMS login do. It previously passed an empty session id and
	// created no user_sessions row at all, which quietly exempted every
	// registered user from the entire session layer: they never appeared in
	// /account/sessions, SessionValidationMiddleware skipped them (so no idle
	// timeout and no absolute lifetime), the concurrent-session limit did not
	// apply, and logout had no session to revoke. Their refresh token was also
	// unbound, so it survived every revocation control.
	var sessionID string
	if s.sessionService != nil {
		if err := enforceConcurrentLimitWithPolicy(ctx, s.sessionService, user.UserUUID, user.UserID, policy); err != nil {
			return nil, err
		}
		attrs := SessionAttributes{
			AMR:                []string{jwt.AMRPassword},
			ACR:                jwt.ACRLevel1,
			IdentityProviderID: connectedSystemIdentityProviderID(client),
		}
		if client != nil && client.ClientID > 0 {
			cid := client.ClientID
			attrs.ClientID = &cid
		}
		sess, err := createSessionWithPolicy(ctx, s.sessionService, user.UserID, clientTenantID(client), middleware.ClientIPFromContext(ctx), middleware.UserAgentFromContext(ctx), policy, attrs)
		if err != nil {
			return nil, err
		}
		sessionID = sess.UserSessionUUID.String()
	}

	accessToken, idToken, refreshToken, err := generateTokenSetWithAuthContext(ctx, sub, user, client, tokenAuthContextWithPolicy([]string{jwt.AMRPassword}, jwt.ACRLevel1, sessionID, policy, tokenPolicy))
	if err != nil {
		return nil, err
	}
	resp := buildRegisterTokenResponse(accessToken, idToken, refreshToken, time.Now().Unix())
	applyRegisterCookiePolicy(resp, policy)
	if policy.AccessTokenTTLSeconds > 0 {
		resp.ExpiresIn = int64(policy.AccessTokenTTLSeconds)
	}
	// Mirror login: the client needs the session id it was just given.
	if sessionID != "" {
		resp.SessionID = &sessionID
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
