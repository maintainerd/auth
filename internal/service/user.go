package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/cache"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/security"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserServiceDataResult struct {
	UserUUID           uuid.UUID
	Username           string
	Fullname           string
	Email              string
	Phone              string
	IsEmailVerified    bool
	IsPhoneVerified    bool
	IsProfileCompleted bool
	IsAccountCompleted bool
	Status             string
	Metadata           datatypes.JSON
	Tenant             *TenantServiceDataResult
	UserIdentities     *[]UserIdentityServiceDataResult
	Roles              *[]RoleServiceDataResult
	CreatedAt          time.Time
	UpdatedAt          time.Time
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
	Username     *string
	Email        *string
	Phone        *string
	Status       []string
	TenantID     int64
	RoleUUID     *string
	UserPoolUUID *string
	ClientUUID   *string
	Page         int
	Limit        int
	SortBy       string
	SortOrder    string
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
	Create(ctx context.Context, username string, fullname string, email *string, phone *string, password string, status string, metadata datatypes.JSON, tenantUUID string, creatorUserUUID uuid.UUID) (*UserServiceDataResult, error)
	Update(ctx context.Context, userUUID uuid.UUID, tenantID int64, username string, fullname string, email *string, phone *string, status string, metadata datatypes.JSON, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error)
	SetStatus(ctx context.Context, userUUID uuid.UUID, tenantID int64, status string, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error)
	VerifyEmail(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	VerifyPhone(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	CompleteAccount(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	DeleteByUUID(ctx context.Context, userUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*UserServiceDataResult, error)
	AssignUserRoles(ctx context.Context, userUUID uuid.UUID, roleUUIDs []uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	RemoveUserRole(ctx context.Context, userUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	GetUserRoles(ctx context.Context, userUUID uuid.UUID) ([]RoleServiceDataResult, error)
	GetUserIdentities(ctx context.Context, userUUID uuid.UUID) ([]UserIdentityServiceDataResult, error)
	// FindBySubAndClientID resolves a user from a JWT sub claim and client ID.
	// Used by UserContextMiddleware to populate the request context.
	FindBySubAndClientID(ctx context.Context, sub string, clientID string) (*model.User, error)
}

type userService struct {
	db                   *gorm.DB
	userRepo             repository.UserRepository
	userIdentityRepo     repository.UserIdentityRepository
	userRoleRepo         repository.UserRoleRepository
	roleRepo             repository.RoleRepository
	tenantRepo           repository.TenantRepository
	identityProviderRepo repository.IdentityProviderRepository
	clientRepo           repository.ClientRepository
	userPoolRepo         repository.UserPoolRepository
	cacheInvalidator     cache.Invalidator
}

func NewUserService(
	db *gorm.DB,
	userRepo repository.UserRepository,
	userIdentityRepo repository.UserIdentityRepository,
	userRoleRepo repository.UserRoleRepository,
	roleRepo repository.RoleRepository,
	tenantRepo repository.TenantRepository,
	identityProviderRepo repository.IdentityProviderRepository,
	clientRepo repository.ClientRepository,
	userPoolRepo repository.UserPoolRepository,
	cacheInvalidator cache.Invalidator,
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
		userPoolRepo:         userPoolRepo,
		cacheInvalidator:     cacheInvalidator,
	}
}

// invalidateUserCache clears all cached user-context entries for the given
// user's identities. Call this after any mutation that changes data visible
// in the user-context cache (user fields, roles, status, etc.).
func (s *userService) invalidateUserCache(ctx context.Context, identities []model.UserIdentity) {
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
func (s *userService) findDefaultRole(roleRepo repository.RoleRepository, tenantID int64) (*model.Role, error) {
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

	// Convert user pool UUID to ID if provided
	var userPoolID *int64
	if filter.UserPoolUUID != nil {
		poolUUIDParsed, err := uuid.Parse(*filter.UserPoolUUID)
		if err != nil {
			return nil, apperror.NewValidation("invalid user pool UUID")
		}
		pool, err := s.userPoolRepo.FindByUUID(poolUUIDParsed)
		if err != nil || pool == nil {
			return nil, apperror.NewNotFound("user pool not found")
		}
		userPoolID = &pool.UserPoolID
	}

	// Build query filter
	queryFilter := repository.UserRepositoryGetFilter{
		Username:   filter.Username,
		Email:      filter.Email,
		Phone:      filter.Phone,
		Status:     filter.Status,
		TenantID:   &filter.TenantID,
		RoleID:     roleID,
		ClientID:   clientID,
		UserPoolID: userPoolID,
		Page:       filter.Page,
		Limit:      filter.Limit,
		SortBy:     filter.SortBy,
		SortOrder:  filter.SortOrder,
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
	hasTenantAccess := false
	for _, identity := range user.UserIdentities {
		if identity.TenantID == tenantID {
			hasTenantAccess = true
			break
		}
	}
	if !hasTenantAccess {
		span.SetStatus(codes.Error, "user not found or access denied")
		return nil, apperror.NewNotFoundWithReason("user not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	return toUserServiceDataResult(user), nil
}

func (s *userService) Create(ctx context.Context, username string, fullname string, email *string, phone *string, password string, status string, metadata datatypes.JSON, tenantUUID string, creatorUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.create")
	defer span.End()
	span.SetAttributes(attribute.String("user.username", username))

	var createdUser *model.User

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

		// Check if user already exists by username
		existingUser, err := txUserRepo.FindByUsername(username)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return apperror.NewConflict("username already exists")
		}

		// Check if user already exists by email (only if email is provided)
		if email != nil && *email != "" {
			existingUser, err = txUserRepo.FindByEmail(*email)
			if err != nil {
				return err
			}
			if existingUser != nil {
				return apperror.NewConflict("email already exists")
			}
		}

		// Hash password
		hashedPassword, err := security.HashPassword([]byte(password))
		if err != nil {
			return err
		}

		// Create user
		hashedPasswordStr := string(hashedPassword)

		// Convert optional pointers to strings
		emailStr := ""
		if email != nil {
			emailStr = *email
		}
		phoneStr := ""
		if phone != nil {
			phoneStr = *phone
		}

		newUser := &model.User{
			Username: username,
			Fullname: fullname,
			Email:    emailStr,
			Phone:    phoneStr,
			Password: &hashedPasswordStr,
			Status:   status,
			Metadata: metadata,
		}

		_, err = txUserRepo.Create(newUser)
		if err != nil {
			return err
		}

		// Find default auth client for this tenant
		defaultClient, err := txClientRepo.FindDefaultByTenantID(targetTenant.TenantID)
		if err != nil || defaultClient == nil {
			return apperror.NewNotFoundWithReason("default auth client not found for tenant")
		}

		// Create default user identity
		userIdentity := &model.UserIdentity{
			TenantID: targetTenant.TenantID,
			UserID:   newUser.UserID,
			ClientID: defaultClient.ClientID,
			Provider: model.ProviderDefault,
			Sub:      newUser.UserUUID.String(), // Use user UUID as sub for default provider
			Metadata: datatypes.JSON([]byte(`{}`)),
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

		userRole := &model.UserRole{
			UserID: newUser.UserID,
			RoleID: defaultRole.RoleID,
		}

		_, err = txUserRoleRepo.Create(userRole)
		if err != nil {
			return err
		}

		// Fetch created user with relationships
		createdUser, err = txUserRepo.FindByUUID(newUser.UserUUID, "UserIdentities.Client", "UserIdentities.Tenant", "Roles")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create user failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toUserServiceDataResult(createdUser), nil
}

func (s *userService) Update(ctx context.Context, userUUID uuid.UUID, tenantID int64, username string, fullname string, email *string, phone *string, status string, metadata datatypes.JSON, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.update")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID))

	var updatedUser *model.User

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)

		// Check if target user exists
		user, err := txUserRepo.FindByUUID(userUUID, "UserIdentities")
		if err != nil || user == nil {
			return apperror.NewNotFound("user not found")
		}

		// Validate tenant ownership - check if user has an identity in this tenant
		hasTenantAccess := false
		for _, identity := range user.UserIdentities {
			if identity.TenantID == tenantID {
				hasTenantAccess = true
				break
			}
		}
		if !hasTenantAccess {
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

		// Check if username is taken by another user
		if username != user.Username {
			existingUser, err := txUserRepo.FindByUsername(username)
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

		// Check if email is taken by another user (only if email is provided and different)
		if email != nil && *email != "" && *email != user.Email {
			existingUser, err := txUserRepo.FindByEmail(*email)
			if err != nil {
				return err
			}
			if existingUser != nil && existingUser.UserID != user.UserID {
				return apperror.NewConflict("email already exists")
			}
		}

		// Update user
		user.Username = username
		user.Fullname = fullname
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
		updatedUser, err = txUserRepo.FindByUUID(userUUID, "UserIdentities.Client", "UserIdentities.Tenant", "Roles")
		if err != nil {
			return err
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
	return toUserServiceDataResult(updatedUser), nil
}

func (s *userService) SetStatus(ctx context.Context, userUUID uuid.UUID, tenantID int64, status string, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.setStatus")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID), attribute.String("user.status", status))

	// Check if target user exists
	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities")
	if err != nil || user == nil {
		return nil, apperror.NewNotFound("user not found")
	}

	// Validate tenant ownership - check if user has an identity in this tenant
	hasTenantAccess := false
	for _, identity := range user.UserIdentities {
		if identity.TenantID == tenantID {
			hasTenantAccess = true
			break
		}
	}
	if !hasTenantAccess {
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

	err = s.userRepo.SetStatus(userUUID, status)
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
	hasTenantAccess := false
	for _, identity := range user.UserIdentities {
		if identity.TenantID == tenantID {
			hasTenantAccess = true
			break
		}
	}
	if !hasTenantAccess {
		return nil, apperror.NewNotFoundWithReason("user not found or access denied")
	}

	// Update is_email_verified and is_account_completed
	_, err = s.userRepo.UpdateByUUID(userUUID, map[string]any{
		"is_email_verified":    true,
		"is_account_completed": true,
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
	hasTenantAccess := false
	for _, identity := range user.UserIdentities {
		if identity.TenantID == tenantID {
			hasTenantAccess = true
			break
		}
	}
	if !hasTenantAccess {
		return nil, apperror.NewNotFoundWithReason("user not found or access denied")
	}

	// Update is_phone_verified
	_, err = s.userRepo.UpdateByUUID(userUUID, map[string]any{
		"is_phone_verified": true,
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
	hasTenantAccess := false
	for _, identity := range user.UserIdentities {
		if identity.TenantID == tenantID {
			hasTenantAccess = true
			break
		}
	}
	if !hasTenantAccess {
		return nil, apperror.NewNotFoundWithReason("user not found or access denied")
	}

	// Update is_account_completed
	_, err = s.userRepo.UpdateByUUID(userUUID, map[string]any{
		"is_account_completed": true,
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

func (s *userService) DeleteByUUID(ctx context.Context, userUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.delete")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID))

	// Check if target user exists
	user, err := s.userRepo.FindByUUID(userUUID, "UserIdentities.Client", "UserIdentities", "Roles")
	if err != nil || user == nil {
		return nil, apperror.NewNotFound("user not found")
	}

	// Validate tenant ownership - check if user has an identity in this tenant
	hasTenantAccess := false
	for _, identity := range user.UserIdentities {
		if identity.TenantID == tenantID {
			hasTenantAccess = true
			break
		}
	}
	if !hasTenantAccess {
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

	// Invalidate cache before deletion (identities will be gone after)
	s.invalidateUserCache(ctx, user.UserIdentities)

	// Delete user (cascade will handle related records)
	err = s.userRepo.DeleteByUUID(userUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete user failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toUserServiceDataResult(user), nil
}

func (s *userService) AssignUserRoles(ctx context.Context, userUUID uuid.UUID, roleUUIDs []uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.assignRoles")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID))

	var userWithRoles *model.User

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		// Check if user exists and preload identities for tenant validation
		user, err := txUserRepo.FindByUUID(userUUID, "UserIdentities")
		if err != nil || user == nil {
			return apperror.NewNotFound("user not found")
		}

		// Validate tenant ownership - check if user has an identity in this tenant
		hasTenantAccess := false
		for _, identity := range user.UserIdentities {
			if identity.TenantID == tenantID {
				hasTenantAccess = true
				break
			}
		}
		if !hasTenantAccess {
			return apperror.NewNotFoundWithReason("user not found or access denied")
		}

		// Validate and assign roles
		for _, roleUUID := range roleUUIDs {
			// Find role by UUID
			role, err := txRoleRepo.FindByUUID(roleUUID)
			if err != nil {
				return err
			}
			if role == nil {
				return apperror.NewNotFound("role not found")
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
			userRole := &model.UserRole{
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

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "assign user roles failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.invalidateUserCache(ctx, userWithRoles.UserIdentities)

	return toUserServiceDataResult(userWithRoles), nil
}

func (s *userService) RemoveUserRole(ctx context.Context, userUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.removeRole")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()), attribute.Int64("tenant.id", tenantID))

	var userWithRoles *model.User

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		// Check if user exists and preload identities for tenant validation
		user, err := txUserRepo.FindByUUID(userUUID, "UserIdentities")
		if err != nil || user == nil {
			return apperror.NewNotFound("user not found")
		}

		// Validate tenant ownership - check if user has an identity in this tenant
		hasTenantAccess := false
		for _, identity := range user.UserIdentities {
			if identity.TenantID == tenantID {
				hasTenantAccess = true
				break
			}
		}
		if !hasTenantAccess {
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

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "remove user role failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.invalidateUserCache(ctx, userWithRoles.UserIdentities)

	return toUserServiceDataResult(userWithRoles), nil
}

// Helper functions
func toUserServiceDataResult(user *model.User) *UserServiceDataResult {
	if user == nil {
		return nil
	}

	result := &UserServiceDataResult{
		UserUUID:           user.UserUUID,
		Username:           user.Username,
		Fullname:           user.Fullname,
		Email:              user.Email,
		Phone:              user.Phone,
		IsEmailVerified:    user.IsEmailVerified,
		IsPhoneVerified:    user.IsPhoneVerified,
		IsProfileCompleted: user.IsProfileCompleted,
		IsAccountCompleted: user.IsAccountCompleted,
		Status:             user.Status,
		Metadata:           user.Metadata,
		CreatedAt:          user.CreatedAt,
		UpdatedAt:          user.UpdatedAt,
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

func (s *userService) GetUserRoles(ctx context.Context, userUUID uuid.UUID) ([]RoleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.getUserRoles")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()))

	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "user not found")
		return nil, apperror.NewNotFound("user not found")
	}

	roles, err := s.userRepo.FindRoles(user.UserID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get user roles failed")
		return nil, err
	}

	result := make([]RoleServiceDataResult, len(roles))
	for i, role := range roles {
		result[i] = *toRoleServiceDataResult(&role)
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *userService) GetUserIdentities(ctx context.Context, userUUID uuid.UUID) ([]UserIdentityServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user.getUserIdentities")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()))

	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "user not found")
		return nil, apperror.NewNotFound("user not found")
	}

	identities, err := s.userIdentityRepo.FindByUserID(user.UserID)
	if err != nil {
		return nil, err
	}

	result := make([]UserIdentityServiceDataResult, len(identities))
	for i, identity := range identities {
		// Load Client if needed
		var Client *ClientServiceDataResult
		if identity.ClientID > 0 {
			ac, err := s.clientRepo.FindByID(identity.ClientID)
			if err == nil && ac != nil {
				Client = ToClientServiceDataResult(ac)
			}
		}

		result[i] = UserIdentityServiceDataResult{
			UserIdentityUUID: identity.UserIdentityUUID,
			Provider:         identity.Provider,
			Sub:              identity.Sub,
			Metadata:         identity.Metadata,
			Client:           Client,
			CreatedAt:        identity.CreatedAt,
			UpdatedAt:        identity.UpdatedAt,
		}
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

// FindBySubAndClientID resolves a *model.User from a JWT sub claim and client
// identifier. This satisfies the middleware.UserContextProvider interface so
// the middleware can be wired without a direct repository dependency.
func (s *userService) FindBySubAndClientID(ctx context.Context, sub string, clientID string) (*model.User, error) {
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
