package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"golang.org/x/crypto/bcrypt"
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
	IsActive           bool
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
	AuthClient       *AuthClientServiceDataResult
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type UserServiceGetFilter struct {
	Username   *string
	Email      *string
	Phone      *string
	IsActive   *bool
	TenantUUID *string
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
	Get(filter UserServiceGetFilter) (*UserServiceGetResult, error)
	GetByUUID(userUUID uuid.UUID) (*UserServiceDataResult, error)
	Create(username string, fullname string, email *string, phone *string, password string, tenantUUID string, creatorUserUUID uuid.UUID) (*UserServiceDataResult, error)
	Update(userUUID uuid.UUID, username string, fullname string, email *string, phone *string, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error)
	SetActiveStatus(userUUID uuid.UUID, isActive bool, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error)
	DeleteByUUID(userUUID uuid.UUID, deleterUserUUID uuid.UUID) (*UserServiceDataResult, error)
	AssignUserRoles(userUUID uuid.UUID, roleUUIDs []uuid.UUID) (*UserServiceDataResult, error)
	RemoveUserRole(userUUID uuid.UUID, roleUUID uuid.UUID) (*UserServiceDataResult, error)
}

type userService struct {
	db                   *gorm.DB
	userRepo             repository.UserRepository
	userIdentityRepo     repository.UserIdentityRepository
	userRoleRepo         repository.UserRoleRepository
	roleRepo             repository.RoleRepository
	tenantRepo           repository.TenantRepository
	identityProviderRepo repository.IdentityProviderRepository
	authClientRepo       repository.AuthClientRepository
}

func NewUserService(
	db *gorm.DB,
	userRepo repository.UserRepository,
	userIdentityRepo repository.UserIdentityRepository,
	userRoleRepo repository.UserRoleRepository,
	roleRepo repository.RoleRepository,
	tenantRepo repository.TenantRepository,
	identityProviderRepo repository.IdentityProviderRepository,
	authClientRepo repository.AuthClientRepository,
) UserService {
	return &userService{
		db:                   db,
		userRepo:             userRepo,
		userIdentityRepo:     userIdentityRepo,
		userRoleRepo:         userRoleRepo,
		roleRepo:             roleRepo,
		tenantRepo:           tenantRepo,
		identityProviderRepo: identityProviderRepo,
		authClientRepo:       authClientRepo,
	}
}

func (s *userService) Get(filter UserServiceGetFilter) (*UserServiceGetResult, error) {
	// Convert tenant UUID to ID if provided
	var tenantID *int64
	if filter.TenantUUID != nil {
		tenantUUIDParsed, err := uuid.Parse(*filter.TenantUUID)
		if err != nil {
			return nil, errors.New("invalid tenant UUID")
		}

		tenant, err := s.tenantRepo.FindByUUID(tenantUUIDParsed)
		if err != nil || tenant == nil {
			return nil, errors.New("tenant not found")
		}
		tenantID = &tenant.TenantID
	}

	// Build query filter
	queryFilter := repository.UserRepositoryGetFilter{
		Username:  filter.Username,
		Email:     filter.Email,
		Phone:     filter.Phone,
		IsActive:  filter.IsActive,
		TenantID:  tenantID,
		Page:      filter.Page,
		Limit:     filter.Limit,
		SortBy:    filter.SortBy,
		SortOrder: filter.SortOrder,
	}

	result, err := s.userRepo.FindPaginated(queryFilter)
	if err != nil {
		return nil, err
	}

	// Build response data
	resData := make([]UserServiceDataResult, len(result.Data))
	for i, rdata := range result.Data {
		resData[i] = *toUserServiceDataResult(&rdata)
	}

	return &UserServiceGetResult{
		Data:       resData,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *userService) GetByUUID(userUUID uuid.UUID) (*UserServiceDataResult, error) {
	user, err := s.userRepo.FindByUUID(userUUID, "Tenant", "UserIdentities.AuthClient", "Roles")
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}

	return toUserServiceDataResult(user), nil
}

func (s *userService) Create(username string, fullname string, email *string, phone *string, password string, tenantUUID string, creatorUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	var createdUser *model.User

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txTenantRepo := s.tenantRepo.WithTx(tx)
		txAuthClientRepo := s.authClientRepo.WithTx(tx)

		// Parse tenant UUID
		tenantUUIDParsed, err := uuid.Parse(tenantUUID)
		if err != nil {
			return errors.New("invalid tenant UUID")
		}

		// Validate tenant exists
		targetTenant, err := txTenantRepo.FindByUUID(tenantUUIDParsed)
		if err != nil || targetTenant == nil {
			return errors.New("tenant not found")
		}

		// Get creator user with tenant info
		creatorUser, err := txUserRepo.FindByUUID(creatorUserUUID, "Tenant")
		if err != nil || creatorUser == nil {
			return errors.New("creator user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(creatorUser, targetTenant); err != nil {
			return err
		}

		// Check if user already exists by username
		existingUser, err := txUserRepo.FindByUsername(username, targetTenant.TenantID)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return errors.New("username already exists")
		}

		// Check if user already exists by email (only if email is provided)
		if email != nil && *email != "" {
			existingUser, err = txUserRepo.FindByEmail(*email, targetTenant.TenantID)
			if err != nil {
				return err
			}
			if existingUser != nil {
				return errors.New("email already exists")
			}
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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
			IsActive: true,
			TenantID: targetTenant.TenantID,
		}

		_, err = txUserRepo.Create(newUser)
		if err != nil {
			return err
		}

		// Find default auth client for this tenant
		defaultAuthClient, err := txAuthClientRepo.FindDefaultByTenantID(targetTenant.TenantID)
		if err != nil || defaultAuthClient == nil {
			return errors.New("default auth client not found for tenant")
		}

		// Create default user identity
		userIdentity := &model.UserIdentity{
			UserID:       newUser.UserID,
			AuthClientID: defaultAuthClient.AuthClientID,
			Provider:     "default",
			Sub:          newUser.UserUUID.String(), // Use user UUID as sub for default provider
			Metadata:     datatypes.JSON([]byte(`{}`)),
		}

		_, err = txUserIdentityRepo.Create(userIdentity)
		if err != nil {
			return err
		}

		// Fetch created user with relationships
		createdUser, err = txUserRepo.FindByUUID(newUser.UserUUID, "Tenant", "UserIdentities.AuthClient", "Roles")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toUserServiceDataResult(createdUser), nil
}

func (s *userService) Update(userUUID uuid.UUID, username string, fullname string, email *string, phone *string, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	var updatedUser *model.User

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)

		// Check if target user exists
		user, err := txUserRepo.FindByUUID(userUUID, "Tenant")
		if err != nil || user == nil {
			return errors.New("user not found")
		}

		// Get updater user with tenant info
		updaterUser, err := txUserRepo.FindByUUID(updaterUserUUID, "Tenant")
		if err != nil || updaterUser == nil {
			return errors.New("updater user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(updaterUser, user.Tenant); err != nil {
			return err
		}

		// Check if username is taken by another user
		if username != user.Username {
			existingUser, err := txUserRepo.FindByUsername(username, user.TenantID)
			if err != nil {
				return err
			}
			if existingUser != nil && existingUser.UserID != user.UserID {
				return errors.New("username already exists")
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
			existingUser, err := txUserRepo.FindByEmail(*email, user.TenantID)
			if err != nil {
				return err
			}
			if existingUser != nil && existingUser.UserID != user.UserID {
				return errors.New("email already exists")
			}
		}

		// Update user
		user.Username = username
		user.Fullname = fullname
		if email != nil {
			user.Email = emailStr
		}
		if phone != nil {
			user.Phone = phoneStr
		}

		_, err = txUserRepo.UpdateByUUID(userUUID, user)
		if err != nil {
			return err
		}

		// Fetch updated user with relationships
		updatedUser, err = txUserRepo.FindByUUID(userUUID, "Tenant", "UserIdentities.AuthClient", "Roles")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toUserServiceDataResult(updatedUser), nil
}

func (s *userService) SetActiveStatus(userUUID uuid.UUID, isActive bool, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	// Check if target user exists
	user, err := s.userRepo.FindByUUID(userUUID, "Tenant")
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}

	// Get updater user with tenant info
	updaterUser, err := s.userRepo.FindByUUID(updaterUserUUID, "Tenant")
	if err != nil || updaterUser == nil {
		return nil, errors.New("updater user not found")
	}

	// Validate tenant access permissions
	if err := ValidateTenantAccess(updaterUser, user.Tenant); err != nil {
		return nil, err
	}

	// Update active status
	err = s.userRepo.SetActiveStatus(userUUID, isActive)
	if err != nil {
		return nil, err
	}

	// Fetch updated user with relationships
	updatedUser, err := s.userRepo.FindByUUID(userUUID, "Tenant", "UserIdentities.AuthClient", "Roles")
	if err != nil {
		return nil, err
	}

	return toUserServiceDataResult(updatedUser), nil
}

func (s *userService) DeleteByUUID(userUUID uuid.UUID, deleterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	// Check if target user exists
	user, err := s.userRepo.FindByUUID(userUUID, "Tenant", "UserIdentities.AuthClient", "Roles")
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}

	// Get deleter user with tenant info
	deleterUser, err := s.userRepo.FindByUUID(deleterUserUUID, "Tenant")
	if err != nil || deleterUser == nil {
		return nil, errors.New("deleter user not found")
	}

	// Validate tenant access permissions
	if err := ValidateTenantAccess(deleterUser, user.Tenant); err != nil {
		return nil, err
	}

	// Delete user (cascade will handle related records)
	err = s.userRepo.DeleteByUUID(userUUID)
	if err != nil {
		return nil, err
	}

	return toUserServiceDataResult(user), nil
}

func (s *userService) AssignUserRoles(userUUID uuid.UUID, roleUUIDs []uuid.UUID) (*UserServiceDataResult, error) {
	var userWithRoles *model.User

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		// Check if user exists
		user, err := txUserRepo.FindByUUID(userUUID)
		if err != nil || user == nil {
			return errors.New("user not found")
		}

		// Validate and assign roles
		for _, roleUUID := range roleUUIDs {
			// Find role by UUID
			role, err := txRoleRepo.FindByUUID(roleUUID)
			if err != nil {
				return err
			}
			if role == nil {
				return errors.New("role not found")
			}

			// Check if user already has this role
			existingUserRole, err := txUserRoleRepo.FindByUserIDAndRoleID(user.UserID, role.RoleID)
			if err != nil && err != gorm.ErrRecordNotFound {
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
		userWithRoles, err = txUserRepo.FindByUUID(userUUID, "Tenant", "UserIdentities.AuthClient", "Roles")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toUserServiceDataResult(userWithRoles), nil
}

func (s *userService) RemoveUserRole(userUUID uuid.UUID, roleUUID uuid.UUID) (*UserServiceDataResult, error) {
	var userWithRoles *model.User

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		// Check if user exists
		user, err := txUserRepo.FindByUUID(userUUID)
		if err != nil || user == nil {
			return errors.New("user not found")
		}

		// Find role by UUID
		role, err := txRoleRepo.FindByUUID(roleUUID)
		if err != nil {
			return err
		}
		if role == nil {
			return errors.New("role not found")
		}

		// Remove user-role association
		err = txUserRoleRepo.DeleteByUserIDAndRoleID(user.UserID, role.RoleID)
		if err != nil {
			return err
		}

		// Fetch user with roles for response
		userWithRoles, err = txUserRepo.FindByUUID(userUUID, "Tenant", "UserIdentities.AuthClient", "Roles")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

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
		IsActive:           user.IsActive,
		CreatedAt:          user.CreatedAt,
		UpdatedAt:          user.UpdatedAt,
	}

	// Map Tenant if present
	if user.Tenant != nil {
		result.Tenant = toTenantServiceDataResult(user.Tenant)
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
			// Map AuthClient if present
			if ui.AuthClient != nil {
				userIdentities[i].AuthClient = toAuthClientServiceDataResult(ui.AuthClient)
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
