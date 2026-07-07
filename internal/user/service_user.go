package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/event"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
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
	Client           *ClientServiceDataResult
	CreatedAt        time.Time
	UpdatedAt        time.Time
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
	AssignUserRoles(ctx context.Context, userUUID uuid.UUID, roleUUIDs []uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	RemoveUserRole(ctx context.Context, userUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	GetUserRoles(ctx context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserRolesFilter) ([]RoleServiceDataResult, int64, error)
	GetUserIdentities(ctx context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserIdentitiesFilter) ([]UserIdentityServiceDataResult, int64, error)
	GetUserSessions(ctx context.Context, userUUID uuid.UUID, tenantID int64) ([]*SessionDataResult, error)
	RevokeUserSession(ctx context.Context, userUUID uuid.UUID, tenantID int64, sessionUUID uuid.UUID) error
	// FindBySubAndClientID resolves a user from a JWT sub claim and client ID.
	// Used by UserContextMiddleware to populate the request context.
	FindBySubAndClientID(ctx context.Context, sub string, clientID string) (*User, error)
	// FindByUserID loads a user by primary key with roles, permissions,
	// identities, and identity tenants preloaded. Used by multi-issuer
	// middleware to build AuthContext for federated principals.
	FindByUserID(ctx context.Context, userID int64) (*User, error)
	// ForcePasswordChange sets or clears the force_password_change flag for a user.
	ForcePasswordChange(ctx context.Context, userUUID uuid.UUID, tenantID int64, force bool) error
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
		if err = security.ValidatePasswordPolicy(password, policy); err != nil {
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

		// Record password history
		secpolicy.RecordPasswordHistory(s.passwordHistoryRepo, newUser.UserID, policy.HistoryCount, hashedPasswordStr)

		// Find default auth client for this tenant
		defaultClient, err := txClientRepo.FindDefaultByTenantID(targetTenant.TenantID)
		if err != nil || defaultClient == nil {
			return apperror.NewNotFoundWithReason("default auth client not found for tenant")
		}

		// Create default user identity
		userIdentity := &UserIdentity{
			TenantID:           targetTenant.TenantID,
			UserID:             newUser.UserID,
			ClientID:           &defaultClient.ClientID,
			IdentityProviderID: &defaultClient.IdentityProviderID,
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
		createdUser, err = txUserRepo.FindByUUID(newUser.UserUUID, "UserIdentities.Client", "UserIdentities.Tenant", "Roles", "Profile")
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
		var changed []string
		if username != user.Username {
			changed = append(changed, "username")
		}
		if status != user.Status {
			changed = append(changed, "status")
		}
		if email != nil && emailStr != user.Email {
			changed = append(changed, "email")
		}
		if phone != nil && phoneStr != user.Phone {
			changed = append(changed, "phone")
		}
		if metadata != nil {
			changed = append(changed, "metadata")
		}

		user.Username = username
		user.Status = status
		if email != nil {
			user.Email = emailStr
		}
		if phone != nil {
			user.Phone = phoneStr
		}
		if metadata != nil {
			user.Metadata = metadata
		}

		_, err = txUserRepo.UpdateByUUID(userUUID, user)
		if err != nil {
			return err
		}

		// Fetch updated user with relationships
		updatedUser, err = txUserRepo.FindByUUID(userUUID, "UserIdentities.Client", "UserIdentities.Tenant", "Roles", "Profile")
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
		u, e := txUserRepo.FindByUUID(userUUID, "UserIdentities.Client", "UserIdentities.Tenant", "Roles")
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
	updatedUser, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.Client", "UserIdentities.Tenant", "Roles")
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
	updatedUser, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.Client", "UserIdentities.Tenant", "Roles")
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
	updatedUser, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.Client", "UserIdentities.Tenant", "Roles")
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
	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.Client", "UserIdentities.Tenant", "Roles")
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

func (s *userService) AssignUserRoles(ctx context.Context, userUUID uuid.UUID, roleUUIDs []uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
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
		roles, err := txRoleRepo.FindByUUIDs(roleUUIDStrs)
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
		userWithRoles, err = txUserRepo.FindByUUID(userUUID, "UserIdentities.Client", "UserIdentities.Tenant", "Roles")
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

		// Remove user-role association
		err = txUserRoleRepo.DeleteByUserIDAndRoleID(user.UserID, role.RoleID)
		if err != nil {
			return err
		}

		// Fetch user with roles for response
		userWithRoles, err = txUserRepo.FindByUUID(userUUID, "UserIdentities.Client", "UserIdentities.Tenant", "Roles")
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
			// Map Client if present
			if ui.Client != nil {
				userIdentities[i].Client = ToClientServiceDataResult(ui.Client)
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

	clientIDs := make([]int64, 0, len(result.Data))
	for _, identity := range result.Data {
		if identity.ClientID != nil && *identity.ClientID > 0 {
			clientIDs = append(clientIDs, *identity.ClientID)
		}
	}
	clientMap := make(map[int64]*ClientServiceDataResult)
	if len(clientIDs) > 0 {
		clients, err := s.clientRepo.FindByIDs(clientIDs)
		if err == nil {
			for i := range clients {
				if clients[i].TenantID != tenantID {
					continue
				}
				res := ToClientServiceDataResult(&clients[i])
				clientMap[clients[i].ClientID] = res
			}
		}
	}

	for i, identity := range result.Data {
		var clientResult *ClientServiceDataResult
		if identity.ClientID != nil {
			clientResult = clientMap[*identity.ClientID]
		}
		identities[i] = UserIdentityServiceDataResult{
			UserIdentityUUID: identity.UserIdentityUUID,
			Provider:         identity.Provider,
			Sub:              identity.Sub,
			Metadata:         identity.Metadata,
			Client:           clientResult,
			CreatedAt:        identity.CreatedAt,
			UpdatedAt:        identity.UpdatedAt,
		}
	}

	span.SetStatus(codes.Ok, "")
	return identities, result.Total, nil
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
	span.SetStatus(codes.Ok, "")
	return user, nil
}

func (s *userService) FindByUserID(ctx context.Context, userID int64) (*User, error) {
	return s.userRepo.FindByID(userID,
		"UserIdentities.Tenant",
		"UserIdentities.Client.IdentityProvider",
		"UserRoles.Role.RolePermissions.Permission",
		"Profile",
	)
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
		Identifier:  t.Identifier,
		Status:      t.Status,
		IsSystem:    t.IsSystem,
		Metadata:    t.Metadata,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
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
			ClientID:           &defaultClient.ClientID,
			IdentityProviderID: &idp.IdentityProviderID,
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

	return s.db.Transaction(func(tx *gorm.DB) error {
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

		return nil
	})
}
