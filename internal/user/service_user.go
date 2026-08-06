package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/event"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/email"
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

var userHashPasswordWithPolicy = security.HashPasswordWithPolicy

type UserServiceDataResult struct {
	UserUUID        uuid.UUID
	Username        string
	Fullname        string
	Email           string
	Phone           string
	IsEmailVerified bool
	IsPhoneVerified bool
	Status          string
	Metadata        datatypes.JSON
	LastLoginAt     *time.Time
	LoginCount      int
	EmailVerifiedAt *time.Time
	PhoneVerifiedAt *time.Time
	ExternalID      *string
	CreatedBy       *int64
	UpdatedBy       *int64
	Tenant          *TenantServiceDataResult
	UserIdentities  *[]UserIdentityServiceDataResult
	Roles           *[]RoleServiceDataResult
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type UserIdentityServiceDataResult struct {
	UserIdentityUUID uuid.UUID
	Provider         string
	Sub              string
	Metadata         datatypes.JSON
	// The identity provider that issued this identity. Replaces the former
	// Client field: identities belong to a provider, and which applications may
	// use one is a separate relationship (client_identity_providers).
	IdentityProviderUUID *uuid.UUID
	IdentityProviderName string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type UserServiceGetFilter struct {
	Search     *string
	Username   *string
	Email      *string
	Phone      *string
	Fullname   *string
	Status     []string
	TenantID   int64
	RoleUUID   *string
	ClientUUID *string
	Page       int
	Limit      int
	SortBy     string
	SortOrder  string
}

type UserServiceGetResult struct {
	Data       []UserServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type UserService interface {
	Get(ctx context.Context, filter UserServiceGetFilter) (*UserServiceGetResult, error)
	GetByUUID(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	Create(ctx context.Context, username string, email *string, phone *string, password string, status string, metadata datatypes.JSON, tenantUUID string, creatorUserUUID uuid.UUID) (*UserServiceDataResult, error)
	Update(ctx context.Context, userUUID uuid.UUID, tenantID int64, username string, email *string, phone *string, status string, metadata datatypes.JSON, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error)
	SetStatus(ctx context.Context, userUUID uuid.UUID, tenantID int64, status string, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error)
	VerifyEmail(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	VerifyPhone(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	CompleteAccount(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	DeleteByUUID(ctx context.Context, userUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*UserServiceDataResult, error)
	// AnonymizeUser is the canonical GDPR Article 17 erasure implementation. It
	// anonymizes the user's PII in place (rather than hard-deleting the row, so
	// audit/referential integrity is preserved) and cascades to the user's
	// profile, sessions, and consents. Immutable audit tables (auth_events,
	// management_audit_log) are intentionally left untouched: they store only
	// integer user-id references (no PII), and their BEFORE UPDATE triggers
	// forbid mutation.
	AnonymizeUser(ctx context.Context, userID int64) error
	AssignUserRoles(ctx context.Context, userUUID uuid.UUID, roleUUIDs []uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*UserServiceDataResult, error)
	RemoveUserRole(ctx context.Context, userUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	GetUserRoles(ctx context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserRolesFilter) ([]RoleServiceDataResult, int64, error)
	GetUserIdentities(ctx context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserIdentitiesFilter) ([]UserIdentityServiceDataResult, int64, error)
	GetUserSessions(ctx context.Context, userUUID uuid.UUID, tenantID int64) ([]*SessionDataResult, error)
	RevokeUserSession(ctx context.Context, userUUID uuid.UUID, tenantID int64, sessionUUID uuid.UUID) error
	RevokeAllUserSessions(ctx context.Context, userUUID uuid.UUID, tenantID int64) error
	// FindBySubAndClientID resolves a user from a JWT sub claim and client ID.
	// Used by UserContextMiddleware to populate the request context.
	FindBySubAndClientID(ctx context.Context, sub string, clientID string) (*User, error)
	// FindClientByIdentifier resolves the request's OAuth client. The client is
	// a property of the REQUEST, not of the identity — identities belong to an
	// identity provider and are usable from every client connected to it.
	FindClientByIdentifier(ctx context.Context, identifier string) (*Client, error)
	// ListMembershipCandidates returns SYSTEM-tenant users, which are the only
	// users tenant.CreateByUserUUID will accept as members. It is deliberately
	// separate from the general user list: that one is pinned to the caller's own
	// tenant, and widening it with a tenant filter would turn it into a
	// cross-tenant enumeration surface.
	ListMembershipCandidates(ctx context.Context, search *string, page, limit int) ([]MembershipCandidateDTO, int64, error)
	// FindByUserID loads a user by primary key with roles, permissions,
	// identities, and identity tenants preloaded. Used by multi-issuer
	// middleware to build AuthContext for federated principals.
	FindByUserID(ctx context.Context, userID int64) (*User, error)
	// ForcePasswordChange sets or clears the force_password_change flag for a user.
	ForcePasswordChange(ctx context.Context, userUUID uuid.UUID, tenantID int64, force bool) error
	// SetPassword sets a user's password administratively, held to the same
	// tenant policy and reuse history as self-service rotation and always
	// evicting the target's live credentials. temporary=true forces the user to
	// choose their own on next login.
	SetPassword(ctx context.Context, userUUID uuid.UUID, tenantID int64, newPassword string, temporary bool, actorUserUUID uuid.UUID) error
	// AdminLinkIdentity attaches an existing external identity (provider + sub)
	// to a user on behalf of an administrator — the operator remedy for a
	// duplicate account created through a new IdP.
	AdminLinkIdentity(ctx context.Context, userUUID uuid.UUID, tenantID int64, providerUUID uuid.UUID, sub string, actorUserUUID uuid.UUID) (*UserIdentityServiceDataResult, error)
	GetUserMFA(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserMFAResponseDTO, error)
	// EnsureUserInTenant copies the user identified by userUUID into the target
	// tenant if they do not already have a record there. Returns the userID in
	// the target tenant (existing or newly created).
	EnsureUserInTenant(ctx context.Context, userUUID uuid.UUID, targetTenantID int64) (int64, error)
	// GrantRoleByName looks up a role by name within the tenant and assigns it to
	// the user identified by userUUID. Used by tenant member provisioning to
	// grant the super-admin role when a user is added as an owner.
	GrantRoleByName(ctx context.Context, userUUID uuid.UUID, tenantID int64, roleName string) error
}

type userService struct {
	db                   *gorm.DB
	userRepo             UserRepository
	userIdentityRepo     UserIdentityRepository
	userRoleRepo         UserRoleRepository
	roleRepo             RoleRepository
	tenantRepo           TenantRepository
	identityProviderRepo IdentityProviderRepository
	clientRepo           ClientRepository
	cacheInvalidator     cache.Invalidator
	userTokenRepo        UserTokenRepository
	securitySettingRepo  secpolicy.SecuritySettingRepository // nil → use defaults
	passwordHistoryRepo  UserPasswordHistoryRepository       // nil → skip history
	authEventService     authevent.AuthEventService
	eventService         event.EventService // nil → skip integration events
}

func NewUserService(
	db *gorm.DB,
	userRepo UserRepository,
	userIdentityRepo UserIdentityRepository,
	userRoleRepo UserRoleRepository,
	roleRepo RoleRepository,
	tenantRepo TenantRepository,
	identityProviderRepo IdentityProviderRepository,
	clientRepo ClientRepository,
	cacheInvalidator cache.Invalidator,
	userTokenRepo UserTokenRepository,
	securitySettingRepo secpolicy.SecuritySettingRepository,
	passwordHistoryRepo UserPasswordHistoryRepository,
	authEventService authevent.AuthEventService,
	eventService event.EventService,
) UserService {
	return &userService{
		db:                   db,
		userRepo:             userRepo,
		userIdentityRepo:     userIdentityRepo,
		userRoleRepo:         userRoleRepo,
		roleRepo:             roleRepo,
		tenantRepo:           tenantRepo,
		identityProviderRepo: identityProviderRepo,
		clientRepo:           clientRepo,
		cacheInvalidator:     cacheInvalidator,
		userTokenRepo:        userTokenRepo,
		securitySettingRepo:  securitySettingRepo,
		passwordHistoryRepo:  passwordHistoryRepo,
		authEventService:     coalesceAuthEventService(authEventService),
		eventService:         eventService,
	}
}

// invalidateUserCache clears all cached user-context entries for the given
// user's identities. Call this after any mutation that changes data visible
// in the user-context cache (user fields, roles, status, etc.).
func (s *userService) invalidateUserCache(ctx context.Context, identities []UserIdentity) {
	seen := make(map[string]struct{})
	for _, id := range identities {
		if _, ok := seen[id.Sub]; ok {
			continue
		}
		seen[id.Sub] = struct{}{}
		s.cacheInvalidator.InvalidateUserAll(ctx, id.Sub)
	}
}

// Helper function to find the default role for a tenant
func (s *userService) findDefaultRole(roleRepo RoleRepository, tenantID int64) (*Role, error) {
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

func (s *userService) Get(ctx context.Context, filter UserServiceGetFilter) (*UserServiceGetResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.list")
	defer span.End()

	// Convert role UUID to ID if provided
	var roleID *int64
	if filter.RoleUUID != nil {
		roleUUIDParsed, err := uuid.Parse(*filter.RoleUUID)
		if err != nil {
			return nil, apperror.NewValidation("invalid role UUID")
		}

		role, err := s.roleRepo.FindByUUID(roleUUIDParsed)
		if err != nil || role == nil {
			return nil, apperror.NewNotFound("role not found")
		}
		roleID = &role.RoleID
	}

	// Convert client UUID to ID if provided
	var clientID *int64
	if filter.ClientUUID != nil {
		clientUUIDParsed, err := uuid.Parse(*filter.ClientUUID)
		if err != nil {
			return nil, apperror.NewValidation("invalid client UUID")
		}
		client, err := s.clientRepo.FindByUUIDAndTenantID(clientUUIDParsed, filter.TenantID)
		if err != nil || client == nil {
			return nil, apperror.NewNotFound("client not found")
		}
		clientID = &client.ClientID
	}

	// Build query filter
	queryFilter := UserRepositoryGetFilter{
		Search:    filter.Search,
		Username:  filter.Username,
		Email:     filter.Email,
		Phone:     filter.Phone,
		Fullname:  filter.Fullname,
		Status:    filter.Status,
		TenantID:  &filter.TenantID,
		RoleID:    roleID,
		ClientID:  clientID,
		Page:      filter.Page,
		Limit:     filter.Limit,
		SortBy:    filter.SortBy,
		SortOrder: filter.SortOrder,
	}

	result, err := s.userRepo.FindPaginated(queryFilter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list users failed")
		return nil, err
	}

	// Build response data
	resData := make([]UserServiceDataResult, len(result.Data))
	for i, rdata := range result.Data {
		resData[i] = *toUserServiceDataResult(&rdata)
	}

	span.SetStatus(codes.Ok, "")
	return &UserServiceGetResult{
		Data:       resData,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *userService) GetByUUID(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.getByUUID")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID))

	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.Tenant")
	if err != nil || user == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "user not found")
		return nil, apperror.NewNotFound("user not found")
	}

	// Validate tenant ownership - check if user has an identity in this tenant
	if !userHasTenantAccess(user, tenantID) {
		span.SetStatus(codes.Error, "user not found or access denied")
		return nil, apperror.NewNotFoundWithReason("user not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	return toUserServiceDataResult(user), nil
}

func (s *userService) Create(ctx context.Context, username string, email *string, phone *string, password string, status string, metadata datatypes.JSON, tenantUUID string, creatorUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.create")
	defer span.End()
	span.SetAttributes(attribute.String("user.username", username))

	var createdUser *User
	var capturedTenantID, capturedActorID int64

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txTenantRepo := s.tenantRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		// Parse tenant UUID
		tenantUUIDParsed, err := uuid.Parse(tenantUUID)
		if err != nil {
			return apperror.NewValidation("invalid tenant UUID")
		}

		// Validate tenant exists
		targetTenant, err := txTenantRepo.FindByUUID(tenantUUIDParsed)
		if err != nil || targetTenant == nil {
			return apperror.NewNotFound("tenant not found")
		}

		// Get creator user with tenant info
		creatorUser, err := txUserRepo.FindByUUID(creatorUserUUID, "UserIdentities.Tenant")
		if err != nil || creatorUser == nil {
			return apperror.NewNotFoundWithReason("creator user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(creatorUser, targetTenant); err != nil {
			return err
		}
		capturedTenantID = targetTenant.TenantID
		capturedActorID = creatorUser.UserID

		// Uniqueness is scoped to the target tenant — the same username/email may
		// exist in other tenants.
		existingUser, err := txUserRepo.FindByUsernameAndTenantID(username, targetTenant.TenantID)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return apperror.NewConflict("username already exists")
		}

		// Check if user already exists by email (only if email is provided)
		if email != nil && *email != "" {
			existingUser, err = txUserRepo.FindByEmailAndTenantID(*email, targetTenant.TenantID)
			if err != nil {
				return err
			}
			if existingUser != nil {
				return apperror.NewConflict("email already exists")
			}
		}

		// Validate password against tenant policy
		policy := secpolicy.LoadPasswordPolicy(s.securitySettingRepo, targetTenant.TenantID)
		if err = security.ValidatePasswordPolicyWithContext(ctx, password, policy); err != nil {
			return apperror.NewValidation(err.Error())
		}

		// Hash password
		hashedPassword, err := userHashPasswordWithPolicy(ctx, []byte(password), policy)
		if err != nil {
			return err
		}

		// Create user
		hashedPasswordStr := string(hashedPassword)
		now := time.Now()
		temporaryPasswordExpiresAt := now.Add(time.Duration(policy.TempPasswordValidityHours) * time.Hour)

		// Convert optional pointers to strings
		emailStr := ""
		if email != nil {
			emailStr = *email
		}
		phoneStr := ""
		if phone != nil {
			phoneStr = *phone
		}

		newUser := &User{
			TenantID:                   targetTenant.TenantID,
			Username:                   username,
			Email:                      emailStr,
			Phone:                      phoneStr,
			Password:                   &hashedPasswordStr,
			Status:                     status,
			Metadata:                   metadata,
			ForcePasswordChange:        true,
			PasswordChangedAt:          &now,
			TemporaryPasswordExpiresAt: &temporaryPasswordExpiresAt,
		}

		_, err = txUserRepo.Create(newUser)
		if err != nil {
			return err
		}

		// Record password history within the same transaction. Using the base
		// (non-tx) repo here inserts before the user row is committed, so the
		// user_password_history.user_id FK fails and history is silently dropped.
		if s.passwordHistoryRepo != nil {
			if err := secpolicy.RecordPasswordHistory(s.passwordHistoryRepo.WithTx(tx), newUser.UserID, policy.HistoryCount, hashedPasswordStr); err != nil {
				return apperror.NewInternal("failed to record password history", err)
			}
		}

		// Find default auth client for this tenant
		defaultClient, err := txClientRepo.FindDefaultByTenantID(targetTenant.TenantID)
		if err != nil || defaultClient == nil {
			return apperror.NewNotFoundWithReason("default auth client not found for tenant")
		}

		// Resolve the tenant's default identity provider. Clients relate to IdPs
		// via the client_identity_providers join table, so Client.IdentityProviderID
		// is a transient (gorm:"-") field that FindDefaultByTenantID does not
		// populate — using it here would insert identity_provider_id = 0 and violate
		// fk_user_identities_idp. Look the default IdP up directly instead.
		defaultIdP, err := s.identityProviderRepo.WithTx(tx).FindDefaultByTenantID(targetTenant.TenantID)
		if err != nil || defaultIdP == nil {
			return apperror.NewNotFoundWithReason("default identity provider not found for tenant")
		}

		// Create default user identity
		userIdentity := &UserIdentity{
			TenantID:           targetTenant.TenantID,
			UserID:             newUser.UserID,
			IdentityProviderID: defaultIdP.IdentityProviderID,
			Provider:           shared.ProviderMaintainerd,
			Sub:                newUser.UserUUID.String(),
			Metadata:           datatypes.JSON([]byte(`{}`)),
		}

		_, err = txUserIdentityRepo.Create(userIdentity)
		if err != nil {
			return err
		}

		// Assign default registered role to the user
		defaultRole, err := s.findDefaultRole(txRoleRepo, targetTenant.TenantID)
		if err != nil {
			return err
		}

		userRole := &UserRole{
			UserID: newUser.UserID,
			RoleID: defaultRole.RoleID,
		}

		_, err = txUserRoleRepo.Create(userRole)
		if err != nil {
			return err
		}

		// Display name is not stored on the user — it is derived from the user's
		// Profile (see computeFullname). A new user has no profile yet, so the
		// response carries an empty fullname until a profile is created.

		// Fetch created user with relationships
		createdUser, err = txUserRepo.FindByUUID(newUser.UserUUID, "UserIdentities.IdentityProvider", "UserIdentities.Tenant", "Roles", "Profile")
		if err != nil {
			return err
		}

		// Emit integration event inside the transaction
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeUserCreated, 1, targetTenant.TenantID,
			).SetActor(&creatorUser.UserID).SetSubject(&createdUser.UserUUID, "user")); emitErr != nil {
				return emitErr
			}
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create user failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:     capturedTenantID,
		ActorUserID:  &capturedActorID,
		TargetUserID: &createdUser.UserID,
		IPAddress:    middleware.ClientIPFromContext(ctx),
		UserAgent:    ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:     authevent.AuthEventCategoryUser,
		EventType:    authevent.AuthEventTypeUserCreated,
		Severity:     authevent.AuthEventSeverityInfo,
		Result:       authevent.AuthEventResultSuccess,
		Description:  ptr.Ptr(fmt.Sprintf("User created: %s", createdUser.Username)),
	})
	return toUserServiceDataResult(createdUser), nil
}

func (s *userService) Update(ctx context.Context, userUUID uuid.UUID, tenantID int64, username string, email *string, phone *string, status string, metadata datatypes.JSON, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.update")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID))

	var updatedUser *User
	var capturedActorID int64
	// Captured for the out-of-band notice sent AFTER the commit: the old address
	// is the only party who can tell that a takeover just happened.
	var previousEmail, replacementEmail string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)

		// Check if target user exists. Preload the identity's Tenant so the
		// tenant-access check below has a non-nil tenant to validate against.
		user, err := txUserRepo.FindByUUID(userUUID, "UserIdentities.Tenant")
		if err != nil || user == nil {
			return apperror.NewNotFound("user not found")
		}

		// Validate tenant ownership.
		if !userHasTenantAccess(user, tenantID) {
			return apperror.NewNotFoundWithReason("user not found or access denied")
		}

		// Get updater user with tenant info
		updaterUser, err := txUserRepo.FindByUUID(updaterUserUUID, "UserIdentities.Tenant")
		if err != nil || updaterUser == nil {
			return apperror.NewNotFoundWithReason("updater user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(updaterUser, user.UserIdentities[0].Tenant); err != nil {
			return err
		}
		capturedActorID = updaterUser.UserID

		// Check if username is taken by another user in the same tenant
		if username != user.Username {
			existingUser, err := txUserRepo.FindByUsernameAndTenantID(username, user.TenantID)
			if err != nil {
				return err
			}
			if existingUser != nil && existingUser.UserID != user.UserID {
				return apperror.NewConflict("username already exists")
			}
		}

		// Convert optional pointers to strings
		emailStr := ""
		if email != nil {
			emailStr = *email
		}
		phoneStr := ""
		if phone != nil {
			phoneStr = *phone
		}

		// Check if email is taken by another user in the same tenant (only if email is provided and different)
		if email != nil && *email != "" && *email != user.Email {
			existingUser, err := txUserRepo.FindByEmailAndTenantID(*email, user.TenantID)
			if err != nil {
				return err
			}
			if existingUser != nil && existingUser.UserID != user.UserID {
				return apperror.NewConflict("email already exists")
			}
		}

		// Update user - build changed fields list
		//
		// The write is a map, not the User struct it used to be: GORM's
		// Updates(struct) skips zero values, so `is_email_verified = false` and a
		// nil `email_verified_at` — the whole point of the reset below — would have
		// been silently dropped from the statement.
		var changed []string
		updates := map[string]any{"username": username, "status": status}
		if username != user.Username {
			changed = append(changed, "username")
		}
		if status != user.Status {
			changed = append(changed, "status")
		}
		emailChanged := email != nil && emailStr != user.Email
		phoneChanged := phone != nil && phoneStr != user.Phone
		if email != nil {
			updates["email"] = emailStr
		}
		if emailChanged {
			changed = append(changed, "email")
			// An admin rewriting the sign-in address must not inherit the previous
			// address's proof of control. Leaving is_email_verified set turned
			// "user:update" into an account-takeover primitive: point the account at
			// an attacker-controlled inbox and the already-verified flag let the
			// recovery flows (forgot-password, magic link) treat it as proven.
			// Verification is a statement about an address, never about a row.
			updates["is_email_verified"] = false
			updates["email_verified_at"] = nil
			changed = append(changed, "is_email_verified")
			previousEmail, replacementEmail = user.Email, emailStr
		}
		if phone != nil {
			updates["phone"] = phoneStr
		}
		if phoneChanged {
			changed = append(changed, "phone")
			// Same rule for the phone channel, which carries SMS OTP and recovery.
			updates["is_phone_verified"] = false
			updates["phone_verified_at"] = nil
			changed = append(changed, "is_phone_verified")
		}
		if metadata != nil {
			updates["metadata"] = metadata
			changed = append(changed, "metadata")
		}

		_, err = txUserRepo.UpdateByUUID(userUUID, updates)
		if err != nil {
			return err
		}

		// Two reasons to evict, one call:
		//
		//   deactivated  — a status change made through the general update endpoint
		//                  disables the account exactly as PATCH /status does.
		//                  Routing round the dedicated endpoint cannot be a way to
		//                  keep a disabled user signed in.
		//   identity moved — the live sessions and refresh tokens were minted for a
		//                  sign-in identity the account no longer has, so the
		//                  rewrite cannot hand an account over with its existing
		//                  tokens still working.
		deactivated := status != user.Status && status != shared.StatusActive
		if deactivated || emailChanged || username != user.Username {
			if e := revokeLiveCredentials(tx, user.UserID, shared.SessionRevokeAdmin); e != nil {
				return e
			}
		}

		// Fetch updated user with relationships
		updatedUser, err = txUserRepo.FindByUUID(userUUID, "UserIdentities.IdentityProvider", "UserIdentities.Tenant", "Roles", "Profile")
		if err != nil {
			return err
		}

		// Emit integration event inside the transaction
		if s.eventService != nil && len(changed) > 0 {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeUserUpdated, 1, tenantID,
			).SetActor(&updaterUser.UserID).
				SetSubject(&updatedUser.UserUUID, "user").
				SetChangedFields(changed...)); emitErr != nil {
				return emitErr
			}
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update user failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.invalidateUserCache(ctx, updatedUser.UserIdentities)
	// Best-effort and deliberately after the commit: the change has already
	// landed, so a broken SMTP config must not roll it back or fail the call.
	s.notifyEmailReplacedByAdmin(ctx, updatedUser, previousEmail, replacementEmail)
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:     tenantID,
		ActorUserID:  &capturedActorID,
		TargetUserID: &updatedUser.UserID,
		IPAddress:    middleware.ClientIPFromContext(ctx),
		UserAgent:    ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:     authevent.AuthEventCategoryUser,
		EventType:    authevent.AuthEventTypeUserUpdated,
		Severity:     authevent.AuthEventSeverityInfo,
		Result:       authevent.AuthEventResultSuccess,
		Description:  ptr.Ptr(fmt.Sprintf("User updated: %s", updatedUser.Username)),
	})
	return toUserServiceDataResult(updatedUser), nil
}

// revokeLiveCredentials ends every credential a user currently holds, in the
// caller's transaction.
//
// Setting users.status alone never signed anyone out: the access token stayed
// valid until expiry and the refresh token kept minting replacements, so
// "suspend this account" was a label, not an eviction. Offboarding has to reach
// all three stores at once —
//
//	user_sessions        the canonical browser/app session (authn)
//	oauth_refresh_tokens the long-lived credential that re-mints access tokens
//	user_tokens          legacy session rows the admin session console still lists
//
// The tables are written directly rather than through the authn/oauth
// repositories for the reason AnonymizeUser gives above: importing those
// packages here would invert the dependency, and the write must commit or roll
// back WITH the status change. An error aborts the status change entirely — a
// user reported as suspended while still holding live credentials is the exact
// failure this prevents.
func revokeLiveCredentials(tx *gorm.DB, userID int64, reason string) error {
	now := time.Now()

	if e := tx.Table("user_sessions").
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]any{"revoked_at": now, "revoked_reason": reason}).Error; e != nil {
		return apperror.NewInternal("failed to revoke sessions", e)
	}

	if e := tx.Table("oauth_refresh_tokens").
		Where("user_id = ? AND is_revoked = false", userID).
		Updates(map[string]any{"is_revoked": true, "revoked_at": now}).Error; e != nil {
		return apperror.NewInternal("failed to revoke refresh tokens", e)
	}

	if e := tx.Model(&UserToken{}).
		Where("user_id = ? AND token_type = ? AND is_revoked = false", userID, shared.TokenTypeSession).
		Update("is_revoked", true).Error; e != nil {
		return apperror.NewInternal("failed to revoke session tokens", e)
	}

	return nil
}

// notifyEmailReplacedByAdmin warns the address an admin just replaced.
//
// The admin path had no notice at all, which is what made it a usable takeover
// primitive: an operator (or anyone holding a stolen admin session) could move
// the sign-in identity to a mailbox they control and the real owner's first
// clue would be a password reset they never asked for. The out-of-band notice
// to the OLD address is the only channel the attacker does not already own.
// It mirrors accountService.notifyPreviousEmailOfChange, which covers the
// self-service half of the same change.
//
// The body is composed inline for the same reason as the self-service twin:
// there is no seeded "email changed" template, and rendering an unrelated one
// would send the wrong message.
func (s *userService) notifyEmailReplacedByAdmin(ctx context.Context, user *User, previousEmail, newEmail string) {
	if previousEmail == "" || previousEmail == newEmail {
		return
	}
	bodyPlain := fmt.Sprintf(
		"An administrator just changed the email address on your account from %s to %s.\n\n"+
			"If you did not expect this, contact your administrator immediately — "+
			"whoever made the change can now receive your sign-in and password-reset mail.",
		previousEmail, newEmail)
	if err := email.SendEmail(ctx, s.db, email.SendEmailParams{
		TenantID:  user.TenantID,
		To:        previousEmail,
		Subject:   "Your account email address was changed",
		BodyHTML:  fmt.Sprintf("<p>%s</p>", bodyPlain),
		BodyPlain: bodyPlain,
	}); err != nil {
		slog.Error("user: failed to notify previous email of an administrative address change",
			"error", err, "user_id", user.UserID)
	}
}

func (s *userService) SetStatus(ctx context.Context, userUUID uuid.UUID, tenantID int64, status string, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.setStatus")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID), attribute.String("user.status", status))

	// Check if target user exists. Preload the identity's Tenant so the
	// tenant-access check below has a non-nil tenant to validate against.
	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.Tenant")
	if err != nil || user == nil {
		return nil, apperror.NewNotFound("user not found")
	}

	// Validate tenant ownership - check if user has an identity in this tenant
	if !userHasTenantAccess(user, tenantID) {
		return nil, apperror.NewNotFoundWithReason("user not found or access denied")
	}

	// Get updater user with tenant info
	updaterUser, err := s.userRepo.FindByUUID(updaterUserUUID, "UserIdentities.Tenant")
	if err != nil || updaterUser == nil {
		return nil, apperror.NewNotFoundWithReason("updater user not found")
	}

	// Validate tenant access permissions
	if err := ValidateTenantAccess(updaterUser, user.UserIdentities[0].Tenant); err != nil {
		return nil, err
	}

	var updatedUser *User
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		if e := txUserRepo.SetStatus(userUUID, status); e != nil {
			return e
		}
		// Disabling an account has to evict it, not just relabel it.
		if status != shared.StatusActive {
			if e := revokeLiveCredentials(tx, user.UserID, shared.SessionRevokeAdmin); e != nil {
				return e
			}
		}
		u, e := txUserRepo.FindByUUID(userUUID, "UserIdentities.IdentityProvider", "UserIdentities.Tenant", "Roles")
		if e != nil {
			return e
		}
		updatedUser = u
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeUserStatusChanged, 1, tenantID,
			).SetActor(&updaterUser.UserID).
				SetSubject(&updatedUser.UserUUID, "user").
				SetChangedFields("status")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "set user status failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.invalidateUserCache(ctx, updatedUser.UserIdentities)
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:     tenantID,
		ActorUserID:  &updaterUser.UserID,
		TargetUserID: &updatedUser.UserID,
		IPAddress:    middleware.ClientIPFromContext(ctx),
		UserAgent:    ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:     authevent.AuthEventCategoryUser,
		EventType:    authevent.AuthEventTypeUserUpdated,
		Severity:     authevent.AuthEventSeverityInfo,
		Result:       authevent.AuthEventResultSuccess,
		Description:  ptr.Ptr(fmt.Sprintf("User status set to %s: %s", status, user.Username)),
	})
	return toUserServiceDataResult(updatedUser), nil
}

func (s *userService) VerifyEmail(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.verifyEmail")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID))

	// Check if target user exists and preload identities for tenant validation
	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.Tenant")
	if err != nil || user == nil {
		return nil, apperror.NewNotFound("user not found")
	}

	// Validate tenant ownership - check if user has an identity in this tenant
	if !userHasTenantAccess(user, tenantID) {
		return nil, apperror.NewNotFoundWithReason("user not found or access denied")
	}

	// Update is_email_verified
	_, err = s.userRepo.UpdateByUUID(userUUID, map[string]any{
		"is_email_verified": true,
		"email_verified_at": time.Now(),
	})
	if err != nil {
		return nil, err
	}

	// Fetch updated user with relationships
	updatedUser, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.IdentityProvider", "UserIdentities.Tenant", "Roles")
	if err != nil {
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.invalidateUserCache(ctx, updatedUser.UserIdentities)

	return toUserServiceDataResult(updatedUser), nil
}

func (s *userService) VerifyPhone(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.verifyPhone")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID))

	// Check if target user exists and preload identities for tenant validation
	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.Tenant")
	if err != nil || user == nil {
		return nil, apperror.NewNotFound("user not found")
	}

	// Validate tenant ownership - check if user has an identity in this tenant
	if !userHasTenantAccess(user, tenantID) {
		return nil, apperror.NewNotFoundWithReason("user not found or access denied")
	}

	// Update is_phone_verified
	_, err = s.userRepo.UpdateByUUID(userUUID, map[string]any{
		"is_phone_verified": true,
		"phone_verified_at": time.Now(),
	})
	if err != nil {
		return nil, err
	}

	// Fetch updated user with relationships
	updatedUser, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.IdentityProvider", "UserIdentities.Tenant", "Roles")
	if err != nil {
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.invalidateUserCache(ctx, updatedUser.UserIdentities)

	return toUserServiceDataResult(updatedUser), nil
}

func (s *userService) CompleteAccount(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.completeAccount")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID))

	// Check if target user exists and preload identities for tenant validation
	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.Tenant")
	if err != nil || user == nil {
		return nil, apperror.NewNotFound("user not found")
	}

	// Validate tenant ownership - check if user has an identity in this tenant
	if !userHasTenantAccess(user, tenantID) {
		return nil, apperror.NewNotFoundWithReason("user not found or access denied")
	}

	_, err = s.userRepo.UpdateByUUID(userUUID, map[string]any{})
	if err != nil {
		return nil, err
	}

	// Fetch updated user with relationships
	updatedUser, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.IdentityProvider", "UserIdentities.Tenant", "Roles")
	if err != nil {
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.invalidateUserCache(ctx, updatedUser.UserIdentities)

	return toUserServiceDataResult(updatedUser), nil
}

func (s *userService) DeleteByUUID(ctx context.Context, userUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.delete")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID))

	// Check if target user exists
	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.IdentityProvider", "UserIdentities.Tenant", "Roles")
	if err != nil || user == nil {
		return nil, apperror.NewNotFound("user not found")
	}

	// Validate tenant ownership - check if user has an identity in this tenant
	if !userHasTenantAccess(user, tenantID) {
		return nil, apperror.NewNotFoundWithReason("user not found or access denied")
	}

	// Get deleter user with tenant info
	deleterUser, err := s.userRepo.FindByUUID(deleterUserUUID, "UserIdentities.Tenant")
	if err != nil || deleterUser == nil {
		return nil, apperror.NewNotFoundWithReason("deleter user not found")
	}

	// Validate tenant access permissions
	if err := ValidateTenantAccess(deleterUser, user.UserIdentities[0].Tenant); err != nil {
		return nil, err
	}

	// Guard: cannot delete a user who is a tenant owner
	type ownershipCheck struct {
		TenantID int64
		IsSystem bool
	}
	var ownerships []ownershipCheck
	if err := s.db.Table("tenant_members").
		Select("tenant_members.tenant_id, tenants.is_system").
		Joins("JOIN tenants ON tenants.tenant_id = tenant_members.tenant_id").
		Where("tenant_members.user_id = ? AND tenant_members.role = ? AND tenant_members.deleted_at IS NULL",
			user.UserID, shared.TenantRoleOwner).
		Find(&ownerships).Error; err != nil {
		return nil, apperror.NewInternal("check tenant ownership", err)
	}
	for _, o := range ownerships {
		if o.IsSystem {
			return nil, apperror.NewValidation("cannot delete the owner of the system tenant")
		}
	}
	if len(ownerships) > 0 {
		return nil, apperror.NewValidation("cannot delete a user who is a tenant owner — remove their ownership first")
	}

	// Invalidate cache before deletion (identities will be gone after)
	s.invalidateUserCache(ctx, user.UserIdentities)

	// Delete user (cascade will handle related records)
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if e := s.userRepo.WithTx(tx).DeleteByUUID(userUUID); e != nil {
			return e
		}
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeUserDeleted, 1, tenantID,
			).SetActor(&deleterUser.UserID).SetSubject(&user.UserUUID, "user")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete user failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:     tenantID,
		ActorUserID:  &deleterUser.UserID,
		TargetUserID: &user.UserID,
		IPAddress:    middleware.ClientIPFromContext(ctx),
		UserAgent:    ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:     authevent.AuthEventCategoryUser,
		EventType:    authevent.AuthEventTypeUserDeleted,
		Severity:     authevent.AuthEventSeverityWarn,
		Result:       authevent.AuthEventResultSuccess,
		Description:  ptr.Ptr(fmt.Sprintf("User deleted: %s", user.Username)),
	})
	return toUserServiceDataResult(user), nil
}

// AnonymizeUser implements the GDPR Article 17 erasure cascade. Runs in a single
// transaction so the anonymization is all-or-nothing.
//
// Schema-reconciled deviations from the plan's field list:
//   - users.phone is VARCHAR(20) and cannot hold "deleted_{uuid}@erased"; the
//     PII is removed by setting it NULL instead.
//   - users.pending_email no longer exists (removed in section 1.4), so there is
//     nothing to null there.
//   - auth_events and management_audit_log carry only integer user-id FKs (no
//     PII) and are immutable (BEFORE UPDATE triggers raise). They are left as-is;
//     the referenced user row is already scrubbed, so no PII is reachable.
func (s *userService) AnonymizeUser(ctx context.Context, userID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "user.anonymize")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	placeholder := fmt.Sprintf("deleted_%s@erased", uuid.NewString())

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// users: scrub the sign-in identity + credential.
		if e := tx.Model(&User{}).Where("user_id = ?", userID).Updates(map[string]any{
			"email":    placeholder,
			"username": placeholder,
			"phone":    nil,
			"password": nil,
		}).Error; e != nil {
			return e
		}

		// profiles: scrub names + avatar.
		if e := tx.Model(&Profile{}).Where("user_id = ?", userID).Updates(map[string]any{
			"first_name":   "Erased",
			"middle_name":  nil,
			"last_name":    nil,
			"display_name": nil,
			"profile_url":  nil,
		}).Error; e != nil {
			return e
		}

		// user_sessions: scrub network PII (table-scoped to avoid importing authn).
		if e := tx.Table("user_sessions").Where("user_id = ?", userID).Updates(map[string]any{
			"ip_address": nil,
			"user_agent": nil,
		}).Error; e != nil {
			return e
		}

		// user_consents: scrub network PII on the consent ledger.
		if e := tx.Table("user_consents").Where("user_id = ?", userID).Updates(map[string]any{
			"ip_address": nil,
			"user_agent": nil,
		}).Error; e != nil {
			return e
		}

		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "anonymize failed")
		return apperror.NewInternal("failed to anonymize user", err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// assertCanGrantRoles refuses to hand out a role carrying a permission the actor
// does not already hold.
//
// Assigning a role IS granting its permissions, so "may I edit users?" must not
// mean "may I make anyone a super-admin?". Holding user:update alone was enough
// to assign the tenant's super-admin role to yourself. A super-admin is seeded
// with every administrative permission, so the rule does not restrict them and
// needs no special case. Only elevated permissions are gated — account:…:self
// and public:… confer nothing beyond the holder's own account.
func (s *userService) assertCanGrantRoles(repo UserRepository, actorUserUUID uuid.UUID, tenantID int64, roles []Role) error {
	var granting []string
	for _, role := range roles {
		for _, rp := range role.RolePermissions {
			granting = append(granting, rp.Permission.Name)
		}
	}
	if shared.FirstElevatedPermission(granting) == "" {
		return nil
	}

	actor, err := repo.FindByUUID(actorUserUUID)
	if err != nil || actor == nil {
		return apperror.NewForbidden("the acting user could not be resolved")
	}
	held, err := repo.EffectivePermissionNames(actor.UserID, tenantID)
	if err != nil {
		// Fail CLOSED: an unreadable actor permission set is not "holds everything".
		return apperror.NewInternal("could not resolve the acting user's permissions", err)
	}
	if unheld := shared.FirstUnheldElevatedPermission(granting, held); unheld != "" {
		return apperror.NewForbidden(fmt.Sprintf(
			"you cannot assign a role granting %q because you do not hold it", unheld))
	}
	return nil
}

func (s *userService) AssignUserRoles(ctx context.Context, userUUID uuid.UUID, roleUUIDs []uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.assignRoles")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID))

	var userWithRoles *User

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		// Check if user exists and preload identities for tenant validation
		user, err := txUserRepo.FindByUUID(userUUID, "UserIdentities")
		if err != nil || user == nil {
			return apperror.NewNotFound("user not found")
		}

		// Validate tenant ownership.
		if !userHasTenantAccess(user, tenantID) {
			return apperror.NewNotFoundWithReason("user not found or access denied")
		}

		// Validate and assign roles
		roleUUIDStrs := make([]string, len(roleUUIDs))
		for i, id := range roleUUIDs {
			roleUUIDStrs[i] = id.String()
		}
		roles, err := txRoleRepo.FindByUUIDs(roleUUIDStrs, "RolePermissions.Permission")
		if err != nil {
			return err
		}
		if len(roles) != len(roleUUIDs) {
			return apperror.NewNotFound("role not found")
		}
		for _, role := range roles {
			if role.TenantID != tenantID {
				return apperror.NewNotFoundWithReason("role not found or access denied")
			}
		}

		if err := s.assertCanGrantRoles(txUserRepo, actorUserUUID, tenantID, roles); err != nil {
			return err
		}

		for _, role := range roles {

			// Check if user already has this role
			existingUserRole, err := txUserRoleRepo.FindByUserIDAndRoleID(user.UserID, role.RoleID)
			if err != nil {
				return err
			}
			if existingUserRole != nil {
				continue // Skip if already assigned
			}

			// Create user-role association
			userRole := &UserRole{
				UserID: user.UserID,
				RoleID: role.RoleID,
			}

			_, err = txUserRoleRepo.Create(userRole)
			if err != nil {
				return err
			}
		}

		// Fetch user with roles for response
		userWithRoles, err = txUserRepo.FindByUUID(userUUID, "UserIdentities.IdentityProvider", "UserIdentities.Tenant", "Roles")
		if err != nil {
			return err
		}

		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeUserRoleAssigned, 1, tenantID,
			).SetSubject(&userWithRoles.UserUUID, "user")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "assign user roles failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.invalidateUserCache(ctx, userWithRoles.UserIdentities)
	// Revoke all active sessions so the permission change takes effect immediately.
	if s.userTokenRepo != nil {
		_ = s.userTokenRepo.RevokeAllSessionsByUserID(userWithRoles.UserID)
	}
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:     tenantID,
		TargetUserID: &userWithRoles.UserID,
		IPAddress:    middleware.ClientIPFromContext(ctx),
		UserAgent:    ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:     authevent.AuthEventCategoryAuthz,
		EventType:    authevent.AuthEventTypePrivilegePermissionsChanged,
		Severity:     authevent.AuthEventSeverityInfo,
		Result:       authevent.AuthEventResultSuccess,
		Description:  ptr.Ptr(fmt.Sprintf("Roles assigned to user: %s", userWithRoles.Username)),
	})

	return toUserServiceDataResult(userWithRoles), nil
}

func (s *userService) RemoveUserRole(ctx context.Context, userUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.removeRole")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID))

	var userWithRoles *User

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		// Check if user exists and preload identities for tenant validation
		user, err := txUserRepo.FindByUUID(userUUID, "UserIdentities")
		if err != nil || user == nil {
			return apperror.NewNotFound("user not found")
		}

		// Validate tenant ownership.
		if !userHasTenantAccess(user, tenantID) {
			return apperror.NewNotFoundWithReason("user not found or access denied")
		}

		// Find role by UUID
		role, err := txRoleRepo.FindByUUID(roleUUID)
		if err != nil {
			return err
		}
		if role == nil {
			return apperror.NewNotFound("role not found")
		}
		if role.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("role not found or access denied")
		}

		// The tenant owner's super-admin role is inseparable from ownership: it must
		// not be revocable while they remain owner. Ownership must be transferred
		// first (which reassigns super-admin atomically in the tenant member
		// service). Without this guard, anyone holding user:role:assign could strip
		// super-admin from the owner via the parallel user_roles table, bypassing
		// the tenant_members owner protections and locking the owner out.
		if role.Name == shared.RoleSuperAdmin {
			var ownerCount int64
			if cErr := tx.Table("tenant_members").
				Where("tenant_id = ? AND user_id = ? AND role = ?", tenantID, user.UserID, shared.TenantRoleOwner).
				Count(&ownerCount).Error; cErr != nil {
				return apperror.NewInternal("failed to verify tenant ownership", cErr)
			}
			if ownerCount > 0 {
				return apperror.NewValidation("cannot revoke the super-admin role from the tenant owner — transfer ownership first")
			}
		}

		// Remove user-role association
		err = txUserRoleRepo.DeleteByUserIDAndRoleID(user.UserID, role.RoleID)
		if err != nil {
			return err
		}

		// Fetch user with roles for response
		userWithRoles, err = txUserRepo.FindByUUID(userUUID, "UserIdentities.IdentityProvider", "UserIdentities.Tenant", "Roles")
		if err != nil {
			return err
		}

		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeUserRoleRemoved, 1, tenantID,
			).SetSubject(&userWithRoles.UserUUID, "user")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "remove user role failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.invalidateUserCache(ctx, userWithRoles.UserIdentities)
	// Revoke all active sessions so the permission change takes effect immediately.
	if s.userTokenRepo != nil {
		_ = s.userTokenRepo.RevokeAllSessionsByUserID(userWithRoles.UserID)
	}
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:     tenantID,
		TargetUserID: &userWithRoles.UserID,
		IPAddress:    middleware.ClientIPFromContext(ctx),
		UserAgent:    ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:     authevent.AuthEventCategoryAuthz,
		EventType:    authevent.AuthEventTypePrivilegePermissionsChanged,
		Severity:     authevent.AuthEventSeverityInfo,
		Result:       authevent.AuthEventResultSuccess,
		Description:  ptr.Ptr(fmt.Sprintf("Role removed from user: %s", userWithRoles.Username)),
	})

	return toUserServiceDataResult(userWithRoles), nil
}

// Helper functions
func toUserServiceDataResult(user *User) *UserServiceDataResult {
	if user == nil {
		return nil
	}

	// Fullname is derived from Profile (DisplayName, or FirstName + LastName)
	// since users.fullname column was removed. computeFullname returns "" when
	// Profile isn't preloaded or has no name set.
	derivedFullname := computeFullname(user)
	if derivedFullname == "" {
		// Fall back to the transient in-memory value (e.g. just-created users
		// where Profile hasn't been refetched yet).
		derivedFullname = user.Fullname
	}
	result := &UserServiceDataResult{
		UserUUID:        user.UserUUID,
		Username:        user.Username,
		Fullname:        derivedFullname,
		Email:           user.Email,
		Phone:           user.Phone,
		IsEmailVerified: user.IsEmailVerified,
		IsPhoneVerified: user.IsPhoneVerified,
		Status:          user.Status,
		Metadata:        user.Metadata,
		LastLoginAt:     user.LastLoginAt,
		LoginCount:      user.LoginCount,
		EmailVerifiedAt: user.EmailVerifiedAt,
		PhoneVerifiedAt: user.PhoneVerifiedAt,
		ExternalID:      user.ExternalID,
		CreatedBy:       user.CreatedBy,
		UpdatedBy:       user.UpdatedBy,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}

	// Map Tenant if present - get from UserIdentities
	if len(user.UserIdentities) > 0 && user.UserIdentities[0].Tenant != nil {
		result.Tenant = toTenantServiceDataResult(user.UserIdentities[0].Tenant)
	}

	// Map UserIdentities if present
	if user.UserIdentities != nil {
		userIdentities := make([]UserIdentityServiceDataResult, len(user.UserIdentities))
		for i, ui := range user.UserIdentities {
			userIdentities[i] = UserIdentityServiceDataResult{
				UserIdentityUUID: ui.UserIdentityUUID,
				Provider:         ui.Provider,
				Sub:              ui.Sub,
				Metadata:         ui.Metadata,
				CreatedAt:        ui.CreatedAt,
				UpdatedAt:        ui.UpdatedAt,
			}
			// Populated when the caller preloaded UserIdentities.IdentityProvider.
			// The identity's owner is the provider — there is no client to report.
			if ui.IdentityProvider != nil {
				idpUUID := ui.IdentityProvider.IdentityProviderUUID
				userIdentities[i].IdentityProviderUUID = &idpUUID
				userIdentities[i].IdentityProviderName = ui.IdentityProvider.Name
			}
		}
		result.UserIdentities = &userIdentities
	}

	// Map Roles if present
	if user.Roles != nil {
		roles := make([]RoleServiceDataResult, len(user.Roles))
		for i, role := range user.Roles {
			roles[i] = *toRoleServiceDataResult(&role)
		}
		result.Roles = &roles
	}

	return result
}

func (s *userService) GetUserRoles(ctx context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserRolesFilter) ([]RoleServiceDataResult, int64, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.getUserRoles")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()))

	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities")
	if err != nil || user == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "user not found")
		return nil, 0, apperror.NewNotFound("user not found")
	}

	if !userHasTenantAccess(user, tenantID) {
		span.SetStatus(codes.Error, "user not found or access denied")
		return nil, 0, apperror.NewNotFoundWithReason("user not found or access denied")
	}

	filter.UserID = user.UserID
	result, err := s.userRepo.FindRolesPaginated(filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get user roles failed")
		return nil, 0, err
	}

	roles := make([]RoleServiceDataResult, len(result.Data))
	for i, role := range result.Data {
		roles[i] = *toRoleServiceDataResult(&role)
	}

	span.SetStatus(codes.Ok, "")
	return roles, result.Total, nil
}

// GetUserSessions returns the active sessions for a user (admin view). The user
// must belong to the requesting tenant.
func (s *userService) GetUserSessions(ctx context.Context, userUUID uuid.UUID, tenantID int64) ([]*SessionDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.getUserSessions")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()))

	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities")
	if err != nil || user == nil {
		if err != nil {
			span.RecordError(err)
		}
		return nil, apperror.NewNotFound("user not found")
	}
	if !userHasTenantAccess(user, tenantID) {
		return nil, apperror.NewNotFoundWithReason("user not found or access denied")
	}

	tokens, err := s.userTokenRepo.FindActiveSessions(user.UserID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	sessions := make([]*SessionDataResult, len(tokens))
	for i := range tokens {
		t := tokens[i]
		sessions[i] = &SessionDataResult{
			SessionID:         t.UserTokenUUID.String(),
			IPAddress:         t.IPAddress,
			UserAgent:         t.UserAgent,
			LastUsedAt:        t.LastUsedAt,
			ExpiresAt:         t.ExpiresAt,
			AbsoluteExpiresAt: t.AbsoluteExpiresAt,
			CreatedAt:         t.CreatedAt,
		}
	}

	span.SetStatus(codes.Ok, "")
	return sessions, nil
}

// RevokeUserSession revokes a single active session for a user (admin action).
// The user must belong to the requesting tenant; revoke is idempotent.
func (s *userService) RevokeUserSession(ctx context.Context, userUUID uuid.UUID, tenantID int64, sessionUUID uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "user.revokeUserSession")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()))

	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities")
	if err != nil || user == nil {
		if err != nil {
			span.RecordError(err)
		}
		return apperror.NewNotFound("user not found")
	}
	if !userHasTenantAccess(user, tenantID) {
		return apperror.NewNotFoundWithReason("user not found or access denied")
	}

	if err := s.userTokenRepo.RevokeSessionByUUID(user.UserID, sessionUUID); err != nil {
		span.RecordError(err)
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// RevokeAllUserSessions revokes every active session for a user (admin action —
// force global sign-out). The user must belong to the requesting tenant; the
// operation is idempotent.
func (s *userService) RevokeAllUserSessions(ctx context.Context, userUUID uuid.UUID, tenantID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "user.revokeAllUserSessions")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()))

	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities")
	if err != nil || user == nil {
		if err != nil {
			span.RecordError(err)
		}
		return apperror.NewNotFound("user not found")
	}
	if !userHasTenantAccess(user, tenantID) {
		return apperror.NewNotFoundWithReason("user not found or access denied")
	}

	if err := s.userTokenRepo.RevokeAllSessionsByUserID(user.UserID); err != nil {
		span.RecordError(err)
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *userService) GetUserIdentities(ctx context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserIdentitiesFilter) ([]UserIdentityServiceDataResult, int64, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.getUserIdentities")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()))

	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities")
	if err != nil || user == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "user not found")
		return nil, 0, apperror.NewNotFound("user not found")
	}

	if !userHasTenantAccess(user, tenantID) {
		span.SetStatus(codes.Error, "user not found or access denied")
		return nil, 0, apperror.NewNotFoundWithReason("user not found or access denied")
	}

	filter.UserID = user.UserID
	result, err := s.userIdentityRepo.FindUserIdentitiesPaginated(filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get user identities failed")
		return nil, 0, err
	}

	identities := make([]UserIdentityServiceDataResult, len(result.Data))

	// Resolve each identity's PROVIDER for display. This used to resolve the
	// client the identity was created under, which no longer exists — and was
	// never the right thing to show, since an identity is usable from every
	// client connected to its provider.
	idpIDs := make([]int64, 0, len(result.Data))
	seenIDP := make(map[int64]bool, len(result.Data))
	for _, identity := range result.Data {
		if identity.IdentityProviderID > 0 && !seenIDP[identity.IdentityProviderID] {
			seenIDP[identity.IdentityProviderID] = true
			idpIDs = append(idpIDs, identity.IdentityProviderID)
		}
	}
	idpMap := make(map[int64]*IdentityProvider, len(idpIDs))
	for _, id := range idpIDs {
		idp, err := s.identityProviderRepo.FindByID(id)
		if err != nil || idp == nil {
			continue
		}
		// Tenant check: never surface a provider from another tenant.
		if idp.TenantID != tenantID {
			continue
		}
		idpMap[id] = idp
	}

	for i, identity := range result.Data {
		var idpUUID *uuid.UUID
		var idpName string
		if idp := idpMap[identity.IdentityProviderID]; idp != nil {
			u := idp.IdentityProviderUUID
			idpUUID = &u
			idpName = idp.Name
		}
		identities[i] = UserIdentityServiceDataResult{
			UserIdentityUUID:     identity.UserIdentityUUID,
			Provider:             identity.Provider,
			Sub:                  identity.Sub,
			Metadata:             identity.Metadata,
			IdentityProviderUUID: idpUUID,
			IdentityProviderName: idpName,
			CreatedAt:            identity.CreatedAt,
			UpdatedAt:            identity.UpdatedAt,
		}
	}

	span.SetStatus(codes.Ok, "")
	return identities, result.Total, nil
}

// profileFullname renders a display name for a picker: the explicit display
// name when set, otherwise first + last. Empty when no profile was preloaded —
// the picker falls back to username/email rather than showing a blank row.
func profileFullname(p *Profile) string {
	if p == nil {
		return ""
	}
	if p.DisplayName != nil && strings.TrimSpace(*p.DisplayName) != "" {
		return *p.DisplayName
	}
	name := p.FirstName
	if p.LastName != nil && *p.LastName != "" {
		if name != "" {
			name += " "
		}
		name += *p.LastName
	}
	return name
}

// MembershipCandidateDTO is the minimum a member picker needs. It is
// deliberately not the full user projection: these are users from ANOTHER
// tenant (the system tenant) as far as most callers are concerned, so the
// response carries identity enough to choose a person and nothing more.
type MembershipCandidateDTO struct {
	UserUUID uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Fullname string    `json:"fullname,omitempty"`
}

// ListMembershipCandidates lists active SYSTEM-tenant users.
//
// tenant.CreateByUserUUID rejects any user whose home tenant is not the system
// tenant, and nothing exposed that set — so the console offered a picker of the
// caller's own tenant users, every one of which 403'd. This is the missing
// half.
//
// The system tenant is resolved server-side rather than taken as a parameter,
// so this cannot be pointed at an arbitrary tenant.
func (s *userService) ListMembershipCandidates(ctx context.Context, search *string, page, limit int) ([]MembershipCandidateDTO, int64, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.listMembershipCandidates")
	defer span.End()

	systemTenant, err := s.tenantRepo.FindSystem()
	if err != nil {
		return nil, 0, apperror.NewInternal("failed to resolve the system tenant", err)
	}
	if systemTenant == nil {
		return nil, 0, apperror.NewNotFound("system tenant not found")
	}

	result, err := s.userRepo.FindPaginated(UserRepositoryGetFilter{
		TenantID: &systemTenant.TenantID,
		Search:   search,
		Status:   []string{shared.StatusActive},
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		return nil, 0, err
	}

	out := make([]MembershipCandidateDTO, len(result.Data))
	for i := range result.Data {
		u := &result.Data[i]
		out[i] = MembershipCandidateDTO{
			UserUUID: u.UserUUID,
			Username: u.Username,
			Email:    u.Email,
			Fullname: profileFullname(u.Profile),
		}
	}

	span.SetStatus(codes.Ok, "")
	return out, result.Total, nil
}

func (s *userService) FindClientByIdentifier(_ context.Context, identifier string) (*Client, error) {
	return s.clientRepo.FindByIdentifier(identifier)
}

func (s *userService) ForcePasswordChange(ctx context.Context, userUUID uuid.UUID, tenantID int64, force bool) error {
	_, span := otel.Tracer("service").Start(ctx, "user.forcePasswordChange")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()))

	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "user not found")
		return apperror.NewNotFound("user not found")
	}

	if !userHasTenantAccess(user, tenantID) {
		span.SetStatus(codes.Error, "tenant access denied")
		return apperror.NewForbidden("access denied: user does not belong to your tenant")
	}

	if err := s.userRepo.SetForcePasswordChange(userUUID, force); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "force password change update failed")
		return apperror.NewInternal("failed to update force_password_change", err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// SetPassword sets a user's password on behalf of an administrator.
//
// force-password-change was the only lever an operator had, and it does nothing
// until the user manages to sign in on their own — useless for the case it
// exists for, a user locked out of both their password and their inbox. Every
// comparable product (Auth0, Keycloak, Okta) ships an administrative set.
//
// It is held to the SAME rules as self-service rotation (tenant password policy,
// identity-aware validation, reuse history, temp-password clearing) so the two
// paths cannot drift into enforcing different things, and it always evicts the
// target's live credentials: a password reset performed because an account is
// suspected compromised must not leave the attacker's session and refresh token
// spendable.
//
// temporary=true marks the credential as one-time — the user is forced to
// choose their own on next login and the temp-password expiry clock starts.
func (s *userService) SetPassword(ctx context.Context, userUUID uuid.UUID, tenantID int64, newPassword string, temporary bool, actorUserUUID uuid.UUID) error {
	ctx, span := otel.Tracer("service").Start(ctx, "user.setPassword")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID), attribute.Bool("password.temporary", temporary))

	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.Tenant")
	if err != nil || user == nil {
		span.SetStatus(codes.Error, "user not found")
		return apperror.NewNotFound("user not found")
	}
	if !userHasTenantAccess(user, tenantID) {
		span.SetStatus(codes.Error, "tenant access denied")
		return apperror.NewNotFoundWithReason("user not found or access denied")
	}

	actor, err := s.userRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
	if err != nil || actor == nil {
		return apperror.NewNotFoundWithReason("actor user not found")
	}
	if err := ValidateTenantAccess(actor, user.UserIdentities[0].Tenant); err != nil {
		return err
	}

	policy := secpolicy.LoadPasswordPolicy(s.securitySettingRepo, user.TenantID)

	// Identity-aware: the one thing an admin set knows for certain is whose
	// account it is, so a "password" that merely restates the username or email
	// is rejected here as it is on self-service change.
	if err := security.ValidatePasswordPolicyForUser(ctx, newPassword, policy, security.PasswordUserContext{
		Username: user.Username,
		Email:    user.Email,
	}); err != nil {
		return apperror.NewValidation(err.Error())
	}

	// Fail CLOSED. A nil history repo must not read as "no history to violate":
	// that would let a wiring mistake quietly disable reuse protection for every
	// tenant that configured it.
	if policy.HistoryCount > 0 {
		if s.passwordHistoryRepo == nil {
			return apperror.NewInternal("password history is required by policy but is not configured", nil)
		}
		hashes, hErr := s.passwordHistoryRepo.FindRecentHashes(user.UserID, policy.HistoryCount)
		if hErr != nil {
			// An unreadable history is not an empty history.
			return apperror.NewInternal("failed to read password history", hErr)
		}
		for _, h := range hashes {
			if security.ComparePassword([]byte(h), []byte(newPassword)) {
				return apperror.NewValidation("password was used recently and cannot be reused")
			}
		}
	}

	// Hashing stays OUTSIDE the transaction: argon2id is tuned to take real time
	// and holding the row lock across it serializes concurrent writes on users.
	hashed, err := userHashPasswordWithPolicy(ctx, []byte(newPassword), policy)
	if err != nil {
		return apperror.NewInternal("failed to hash password", err)
	}

	now := time.Now()
	updates := map[string]any{
		"password":              string(hashed),
		"password_changed_at":   now,
		"force_password_change": temporary,
	}
	if temporary {
		updates["temporary_password_expires_at"] = now.Add(time.Duration(policy.TempPasswordValidityHours) * time.Hour)
	} else {
		// A permanent password that has replaced a temporary one must stop being
		// subject to temp-password expiry.
		updates["temporary_password_expires_at"] = nil
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if _, e := s.userRepo.WithTx(tx).UpdateByID(user.UserID, updates); e != nil {
			return apperror.NewInternal("failed to update password", e)
		}
		if s.passwordHistoryRepo != nil {
			if e := secpolicy.RecordPasswordHistory(
				s.passwordHistoryRepo.WithTx(tx), user.UserID, policy.HistoryCount, string(hashed),
			); e != nil {
				return apperror.NewInternal("failed to record password history", e)
			}
		}
		// Unlike self-service rotation, NOTHING is spared here. The caller is not
		// the account owner, so there is no session of theirs worth preserving,
		// and an admin reset is the remedy for a suspected compromise.
		return revokeLiveCredentials(tx, user.UserID, shared.SessionRevokePasswordReset)
	})
	if txErr != nil {
		span.RecordError(txErr)
		span.SetStatus(codes.Error, "set password failed")
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:     tenantID,
			ActorUserID:  &actor.UserID,
			TargetUserID: &user.UserID,
			IPAddress:    middleware.ClientIPFromContext(ctx),
			UserAgent:    ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:     authevent.AuthEventCategoryAuthn,
			EventType:    authevent.AuthEventTypePasswordChangeFail,
			Severity:     authevent.AuthEventSeverityWarn,
			Result:       authevent.AuthEventResultFailure,
			Description:  ptr.Ptr(fmt.Sprintf("Administrative password set failed: %s", user.Username)),
		})
		return txErr
	}

	s.invalidateUserCache(ctx, user.UserIdentities)
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:     tenantID,
		ActorUserID:  &actor.UserID,
		TargetUserID: &user.UserID,
		IPAddress:    middleware.ClientIPFromContext(ctx),
		UserAgent:    ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:     authevent.AuthEventCategoryAuthn,
		EventType:    authevent.AuthEventTypePasswordChange,
		Severity:     authevent.AuthEventSeverityInfo,
		Result:       authevent.AuthEventResultSuccess,
		Description:  ptr.Ptr(fmt.Sprintf("Password set by an administrator: %s", user.Username)),
	})
	span.SetStatus(codes.Ok, "")
	return nil
}

// AdminLinkIdentity attaches an existing external identity to a user on behalf
// of an administrator.
//
// The admin surface could list and unlink identities but never link one, so the
// one case an operator is actually called about — a user who signed up again
// through a new IdP and now owns two accounts — had no remedy but asking the
// user to perform the self-service link themselves, which they often cannot do
// because they can no longer reach the original account.
//
// The safety properties are the ones the (tenant_id, sub) UNIQUE constraint
// implies but does not explain: a sub already linked anywhere in the tenant is
// refused rather than moved (silently re-pointing an identity would transfer a
// live login from one account to another), and the provider must be an active,
// non-deleted provider OF THIS TENANT, so this cannot be used to attach an
// identity from a provider the tenant does not own.
func (s *userService) AdminLinkIdentity(ctx context.Context, userUUID uuid.UUID, tenantID int64, providerUUID uuid.UUID, sub string, actorUserUUID uuid.UUID) (*UserIdentityServiceDataResult, error) {
	ctx, span := otel.Tracer("service").Start(ctx, "user.adminLinkIdentity")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID))

	sub = strings.TrimSpace(sub)
	if sub == "" {
		return nil, apperror.NewValidation("sub is required")
	}

	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.Tenant")
	if err != nil || user == nil {
		span.SetStatus(codes.Error, "user not found")
		return nil, apperror.NewNotFound("user not found")
	}
	if !userHasTenantAccess(user, tenantID) {
		span.SetStatus(codes.Error, "tenant access denied")
		return nil, apperror.NewNotFoundWithReason("user not found or access denied")
	}

	actor, err := s.userRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
	if err != nil || actor == nil {
		return nil, apperror.NewNotFoundWithReason("actor user not found")
	}
	if err := ValidateTenantAccess(actor, user.UserIdentities[0].Tenant); err != nil {
		return nil, err
	}

	// Resolved here rather than through identityProviderRepo, whose interface
	// exposes no by-UUID lookup. deleted_at is filtered explicitly: the
	// projection in types.go declares no gorm.DeletedAt, so GORM applies no
	// soft-delete scope and a deleted provider would otherwise still resolve.
	// A cross-tenant or deleted provider is reported as missing rather than
	// forbidden so the endpoint cannot be used to probe other tenants' providers.
	var provider IdentityProvider
	pErr := s.db.WithContext(ctx).
		Where("identity_provider_uuid = ? AND tenant_id = ? AND deleted_at IS NULL", providerUUID, tenantID).
		First(&provider).Error
	if errors.Is(pErr, gorm.ErrRecordNotFound) {
		return nil, apperror.NewNotFound("identity provider not found")
	}
	if pErr != nil {
		span.RecordError(pErr)
		return nil, apperror.NewInternal("failed to resolve identity provider", pErr)
	}
	if provider.Status != shared.StatusActive {
		return nil, apperror.NewValidation("identity provider is not active")
	}

	// Matched on (tenant_id, sub) — the shape of the real uniqueness constraint
	// (migration 022) — and NOT on (tenant, provider, sub): the tenant is the OIDC
	// issuer, so a sub identifies one person per tenant regardless of which
	// provider slug it arrived under. Checking the narrower triple would let a
	// link through that the database then rejects, turning an operator mistake
	// into a 500 instead of a conflict.
	var existing UserIdentity
	eErr := s.db.WithContext(ctx).
		Where("tenant_id = ? AND sub = ?", tenantID, sub).
		First(&existing).Error
	switch {
	case eErr == nil && existing.UserID == user.UserID:
		return nil, apperror.NewConflict("this identity is already linked to this user")
	case eErr == nil:
		// Never silently re-point: moving a sub would transfer a live login from
		// one account to another without either being told.
		return nil, apperror.NewConflict("this identity is already linked to another user")
	case !errors.Is(eErr, gorm.ErrRecordNotFound):
		span.RecordError(eErr)
		// Fail CLOSED: an unreadable existing-link check is not "no existing link".
		return nil, apperror.NewInternal("failed to check existing identity links", eErr)
	}

	identity := &UserIdentity{
		TenantID:           tenantID,
		UserID:             user.UserID,
		IdentityProviderID: provider.IdentityProviderID,
		Provider:           provider.Provider,
		Sub:                sub,
		Metadata:           datatypes.JSON([]byte("{}")),
		ProvisioningSource: ptr.Ptr("admin_link"),
	}
	created, err := s.userIdentityRepo.Create(identity)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "link identity failed")
		return nil, apperror.NewInternal("failed to link identity", err)
	}

	// A new identity changes which subs resolve to this user, and the cached
	// user context is keyed on sub.
	s.invalidateUserCache(ctx, append(user.UserIdentities, *created))
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:     tenantID,
		ActorUserID:  &actor.UserID,
		TargetUserID: &user.UserID,
		IPAddress:    middleware.ClientIPFromContext(ctx),
		UserAgent:    ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:     authevent.AuthEventCategoryUser,
		EventType:    authevent.AuthEventTypeUserUpdated,
		Severity:     authevent.AuthEventSeverityInfo,
		Result:       authevent.AuthEventResultSuccess,
		Description:  ptr.Ptr(fmt.Sprintf("Identity linked by an administrator: %s", provider.Name)),
	})

	span.SetStatus(codes.Ok, "")
	idpUUID := provider.IdentityProviderUUID
	return &UserIdentityServiceDataResult{
		UserIdentityUUID:     created.UserIdentityUUID,
		Provider:             created.Provider,
		Sub:                  created.Sub,
		Metadata:             created.Metadata,
		IdentityProviderUUID: &idpUUID,
		IdentityProviderName: provider.Name,
		CreatedAt:            created.CreatedAt,
		UpdatedAt:            created.UpdatedAt,
	}, nil
}

// isAuthenticatable reports whether a resolved user may still act on a request.
//
// Deactivating, suspending, or un-completing an account only wrote users.status;
// nothing on the request path ever read it back, so an already-issued access
// token kept working until it expired and its refresh token kept minting new
// ones forever. Offboarding an employee was therefore advisory. Every login path
// already refuses a non-active user (authn service_login.go:328, service_sms_login.go:128,
// service_magic_link.go:130), so requiring "active" here only closes the window
// between the login that minted the token and the status change — it cannot lock
// out anyone who could otherwise have signed in.
//
// nil is treated as not authenticatable so a caller that forgets the nil check
// cannot accidentally admit a missing user.
func isAuthenticatable(user *User) bool {
	return user != nil && user.Status == shared.StatusActive
}

// FindBySubAndClientID resolves a *User from a JWT sub claim and client
// identifier. This satisfies the middleware.UserContextProvider interface so
// the middleware can be wired without a direct repository dependency.
func (s *userService) FindBySubAndClientID(ctx context.Context, sub string, clientID string) (*User, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.findBySubAndClientID")
	defer span.End()
	span.SetAttributes(attribute.String("user.sub", sub), attribute.String("client.id", clientID))

	user, err := s.userRepo.FindBySubAndClientID(sub, clientID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find user by sub and client id failed")
		return nil, err
	}
	// This is THE request-path status check: UserContextMiddleware turns a nil
	// user into 401. Every mutation that changes status also invalidates the
	// user-context cache, so a disabled user stops being served within one
	// request rather than at token expiry.
	if !isAuthenticatable(user) {
		span.SetStatus(codes.Ok, "user is not active")
		return nil, nil
	}
	span.SetStatus(codes.Ok, "")
	return user, nil
}

func (s *userService) FindByUserID(ctx context.Context, userID int64) (*User, error) {
	user, err := s.userRepo.FindByID(userID,
		"UserIdentities.Tenant",
		"UserIdentities.IdentityProvider",
		"UserRoles.Role.RolePermissions.Permission",
		"Profile",
	)
	if err != nil {
		return nil, err
	}
	// Same gate as FindBySubAndClientID: this feeds the multi-issuer middleware's
	// AuthContext, which is the second way a request reaches a handler. Leaving it
	// ungated would have let a suspended user keep working through federated
	// tokens after the first-party path started refusing them.
	if !isAuthenticatable(user) {
		return nil, nil
	}
	return user, nil
}

func (s *userService) GetUserMFA(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserMFAResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.getMFA")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID))

	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities")
	if err != nil || user == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "user not found")
		return nil, apperror.NewNotFound("user not found")
	}

	if !userHasTenantAccess(user, tenantID) {
		span.SetStatus(codes.Error, "user not found or access denied")
		return nil, apperror.NewNotFoundWithReason("user not found or access denied")
	}

	var backupCount int64
	if err := s.db.WithContext(ctx).
		Table("user_mfa_backup_codes").
		Where("user_id = ? AND used = false", user.UserID).
		Count(&backupCount).Error; err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("failed to count backup codes", err)
	}

	type webAuthnRow struct {
		CredentialUUID string
		Name           string
		Transport      pq.StringArray
		LastUsedAt     *time.Time
		CreatedAt      time.Time
	}
	var rows []webAuthnRow
	if err := s.db.WithContext(ctx).
		Table("user_mfa_webauthn_credentials").
		Select("credential_uuid, name, transport, last_used_at, created_at").
		Where("user_id = ?", user.UserID).
		Scan(&rows).Error; err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("failed to query webauthn credentials", err)
	}

	keys := make([]UserMFAWebAuthnKeyDTO, 0, len(rows))
	for _, r := range rows {
		key := UserMFAWebAuthnKeyDTO{
			CredentialUUID: r.CredentialUUID,
			Name:           r.Name,
			Transport:      strings.Join([]string(r.Transport), ","),
			CreatedAt:      r.CreatedAt.Format(time.RFC3339),
		}
		if r.LastUsedAt != nil {
			s := r.LastUsedAt.Format(time.RFC3339)
			key.LastUsedAt = &s
		}
		keys = append(keys, key)
	}

	resp := &UserMFAResponseDTO{
		IsTOTPEnabled:     user.IsTOTPEnabled,
		IsWebAuthnEnabled: user.IsWebAuthnEnabled,
		IsSMSEnabled:      s.isSMSVerified(ctx, user.UserID),
		IsEmailOTPEnabled: s.isEmailOTPVerified(ctx, user.UserID),
		BackupCodesCount:  int(backupCount),
		WebAuthnKeys:      keys,
	}
	if user.FirstMFAEnrolledAt != nil {
		s := user.FirstMFAEnrolledAt.Format(time.RFC3339)
		resp.FirstMFAEnrolledAt = &s
	}

	span.SetStatus(codes.Ok, "")
	return resp, nil
}

func userHasTenantAccess(user *User, tenantID int64) bool {
	for _, identity := range user.UserIdentities {
		if identity.TenantID == tenantID {
			return true
		}
	}
	return false
}

func ValidateTenantAccess(actor *User, target *Tenant) error {
	if actor == nil {
		return apperror.NewUnauthorized("actor user not found")
	}
	if target == nil {
		return apperror.NewNotFoundWithReason("tenant not found")
	}
	if len(actor.UserIdentities) == 0 {
		return apperror.NewForbidden("actor user has no identities")
	}
	// Tenant isolation: access is granted only to the actor's own tenant(s).
	// System-tenant identities do NOT get a cross-tenant override here — that
	// override is confined to the tenant package (tenant-management ops only).
	for _, identity := range actor.UserIdentities {
		if identity.TenantID == target.TenantID {
			return nil
		}
	}
	return apperror.NewForbidden("tenant access denied")
}

func toTenantServiceDataResult(t *Tenant) *TenantServiceDataResult {
	if t == nil {
		return nil
	}
	return &TenantServiceDataResult{
		TenantID:    t.TenantID,
		TenantUUID:  t.TenantUUID,
		Name:        t.Name,
		DisplayName: t.DisplayName,
		Description: t.Description,
		// Tenant identifier was dropped; the DNS-safe name is the slug.
		Identifier: t.Name,
		Status:     t.Status,
		IsSystem:   t.IsSystem,
		Metadata:   t.Metadata,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}
}

func toRoleServiceDataResult(role *Role) *RoleServiceDataResult {
	if role == nil {
		return nil
	}
	return &RoleServiceDataResult{
		RoleUUID:    role.RoleUUID,
		Name:        role.Name,
		Description: role.Description,
		IsDefault:   role.IsDefault,
		IsSystem:    role.IsSystem,
		Status:      role.Status,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}

func ToClientServiceDataResult(client *Client) *ClientServiceDataResult {
	if client == nil {
		return nil
	}
	return &ClientServiceDataResult{
		ClientUUID:  client.ClientUUID,
		Name:        client.Name,
		DisplayName: client.DisplayName,
		ClientType:  client.ClientType,
		Domain:      client.Domain,
		Status:      client.Status,
		IsDefault:   client.IsDefault,
		IsSystem:    client.IsSystem,
		CreatedAt:   client.CreatedAt,
		UpdatedAt:   client.UpdatedAt,
	}
}

// isEmailOTPVerified mirrors isSMSVerified for the email factor. Queried
// directly rather than through the mfa service, which this package does not
// depend on — same approach already taken for SMS.
func (s *userService) isEmailOTPVerified(ctx context.Context, userID int64) bool {
	var verified bool
	if err := s.db.WithContext(ctx).
		Table("user_mfa_emails").
		Select("is_verified").
		Where("user_id = ?", userID).
		Scan(&verified).Error; err != nil {
		return false
	}
	return verified
}

func (s *userService) isSMSVerified(ctx context.Context, userID int64) bool {
	var verified bool
	if err := s.db.WithContext(ctx).
		Table("user_mfa_phones").
		Select("is_verified").
		Where("user_id = ?", userID).
		Scan(&verified).Error; err != nil {
		return false
	}
	return verified
}

func (s *userService) EnsureUserInTenant(ctx context.Context, userUUID uuid.UUID, targetTenantID int64) (int64, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.ensureInTenant")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("target.tenant.id", targetTenantID))

	source, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || source == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "source user not found")
		return 0, apperror.NewNotFound("user not found")
	}

	existing, err := s.userRepo.FindByEmailAndTenantID(source.Email, targetTenantID)
	if err != nil {
		span.RecordError(err)
		return 0, apperror.NewInternal("failed to check user existence", err)
	}
	if existing != nil {
		span.SetStatus(codes.Ok, "")
		return existing.UserID, nil
	}

	existingUsername, err := s.userRepo.FindByUsernameAndTenantID(source.Username, targetTenantID)
	if err != nil {
		span.RecordError(err)
		return 0, apperror.NewInternal("failed to check username", err)
	}
	if existingUsername != nil && existingUsername.UserUUID != source.UserUUID {
		span.SetStatus(codes.Error, "username taken in target tenant")
		return 0, apperror.NewConflict("username '" + source.Username + "' is already taken in the target tenant")
	}

	var copiedUser *User
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)

		// Look up the target tenant's default identity provider and client.
		var idp IdentityProvider
		if txErr := tx.Where("tenant_id = ? AND is_system = ?",
			targetTenantID, true).First(&idp).Error; txErr != nil {
			span.SetStatus(codes.Error, "identity provider not found in target tenant")
			return apperror.NewInternal("failed to find identity provider in target tenant", txErr)
		}

		var defaultClient Client
		if txErr := tx.Where("tenant_id = ? AND is_system = ?",
			targetTenantID, true).First(&defaultClient).Error; txErr != nil {
			span.SetStatus(codes.Error, "system client not found in target tenant")
			return apperror.NewInternal("failed to find system client in target tenant", txErr)
		}

		copied := &User{
			TenantID:          targetTenantID,
			Username:          source.Username,
			Email:             source.Email,
			Phone:             source.Phone,
			Password:          source.Password,
			IsEmailVerified:   source.IsEmailVerified,
			IsPhoneVerified:   source.IsPhoneVerified,
			Status:            shared.StatusActive,
			PasswordChangedAt: source.PasswordChangedAt,
		}

		created, txErr := txUserRepo.Create(copied)
		if txErr != nil {
			return txErr
		}
		copiedUser = created

		identity := &UserIdentity{
			TenantID:           targetTenantID,
			UserID:             created.UserID,
			IdentityProviderID: idp.IdentityProviderID,
			Provider:           shared.ProviderMaintainerd,
			Sub:                uuid.New().String(),
			Metadata:           datatypes.JSON([]byte(`{}`)),
		}
		if _, txErr := txUserIdentityRepo.Create(identity); txErr != nil {
			return txErr
		}

		defaultRole, txErr := txRoleRepo.FindByNameAndTenantID(shared.RoleRegistered, targetTenantID)
		if txErr != nil || defaultRole == nil {
			if txErr != nil {
				return txErr
			}
			span.SetStatus(codes.Error, "default role not found in target tenant")
			return apperror.NewInternal("default role not found in target tenant", nil)
		}

		if _, txErr := txUserRoleRepo.Create(&UserRole{
			UserID: created.UserID,
			RoleID: defaultRole.RoleID,
		}); txErr != nil {
			return txErr
		}

		var sourceProfile Profile
		if txErr := tx.Where("user_id = ? AND is_default = ?", source.UserID, true).First(&sourceProfile).Error; txErr == nil {
			copiedProfile := &Profile{
				UserID:      created.UserID,
				FirstName:   sourceProfile.FirstName,
				MiddleName:  sourceProfile.MiddleName,
				LastName:    sourceProfile.LastName,
				DisplayName: sourceProfile.DisplayName,
				Birthdate:   sourceProfile.Birthdate,
				Gender:      sourceProfile.Gender,
				ProfileURL:  sourceProfile.ProfileURL,
				Metadata:    sourceProfile.Metadata,
			}
			if txErr := tx.Create(copiedProfile).Error; txErr != nil {
				return txErr
			}
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "user copy failed")
		return 0, apperror.NewInternal("failed to copy user to target tenant", err)
	}

	span.SetStatus(codes.Ok, "")
	return copiedUser.UserID, nil
}

func (s *userService) GrantRoleByName(ctx context.Context, userUUID uuid.UUID, tenantID int64, roleName string) error {
	_, span := otel.Tracer("service").Start(ctx, "user.grantRoleByName")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID), attribute.String("role.name", roleName))

	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "user not found")
		return apperror.NewNotFound("user not found")
	}

	// Find the user's record in the target tenant (may differ from source userID).
	targetUser, err := s.userRepo.FindByEmailAndTenantID(user.Email, tenantID)
	if err != nil || targetUser == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "user not found in target tenant")
		return apperror.NewNotFound("user not found in target tenant")
	}

	granted := false
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		role, txErr := txRoleRepo.FindByNameAndTenantID(roleName, tenantID)
		if txErr != nil || role == nil {
			if txErr != nil {
				return txErr
			}
			return apperror.NewNotFoundWithReason("role '" + roleName + "' not found in tenant")
		}

		existing, txErr := txUserRoleRepo.FindByUserIDAndRoleID(targetUser.UserID, role.RoleID)
		if txErr != nil {
			return txErr
		}
		if existing != nil {
			span.SetStatus(codes.Ok, "")
			return nil
		}

		if _, txErr := txUserRoleRepo.Create(&UserRole{
			UserID: targetUser.UserID,
			RoleID: role.RoleID,
		}); txErr != nil {
			return txErr
		}

		granted = true
		return nil
	})
	if txErr != nil {
		return txErr
	}

	// Propagate the grant the same way AssignUserRoles and RemoveUserRole do.
	// Without this the new role sat behind the cached user context and existing
	// access tokens, so a tenant ownership transfer — which routes through here to
	// grant super-admin — took up to the cache TTL (~10 minutes) to actually
	// apply. A privilege change that is not visible is a privilege change that has
	// not happened. Only on an actual grant: the already-assigned early return
	// must stay a no-op rather than signing the user out for nothing.
	if granted {
		// Re-read with identities: the cache is keyed by identity sub, so without
		// them there is nothing to invalidate. FindByEmailAndTenantID takes no
		// preloads.
		if withIdentities, ferr := s.userRepo.FindByUUID(targetUser.UserUUID, "UserIdentities"); ferr == nil && withIdentities != nil {
			s.invalidateUserCache(ctx, withIdentities.UserIdentities)
		}
		if s.userTokenRepo != nil {
			_ = s.userTokenRepo.RevokeAllSessionsByUserID(targetUser.UserID)
		}
	}

	span.SetStatus(codes.Ok, "")
	return nil
}
