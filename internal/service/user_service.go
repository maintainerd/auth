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
	Email              string
	Phone              string
	IsEmailVerified    bool
	IsPhoneVerified    bool
	IsProfileCompleted bool
	IsAccountCompleted bool
	IsActive           bool
	AuthContainer      *AuthContainerServiceDataResult
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
	Username          *string
	Email             *string
	Phone             *string
	IsActive          *bool
	AuthContainerUUID *string
	Page              int
	Limit             int
	SortBy            string
	SortOrder         string
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
	Create(username string, email string, phone string, password string, authContainerUUID string, creatorUserUUID uuid.UUID) (*UserServiceDataResult, error)
	Update(userUUID uuid.UUID, username string, email string, phone string) (*UserServiceDataResult, error)
	SetActiveStatus(userUUID uuid.UUID, isActive bool) (*UserServiceDataResult, error)
	DeleteByUUID(userUUID uuid.UUID) (*UserServiceDataResult, error)
	AssignUserRoles(userUUID uuid.UUID, roleUUIDs []uuid.UUID) (*UserServiceDataResult, error)
	RemoveUserRole(userUUID uuid.UUID, roleUUID uuid.UUID) (*UserServiceDataResult, error)
}

type userService struct {
	db                   *gorm.DB
	userRepo             repository.UserRepository
	userIdentityRepo     repository.UserIdentityRepository
	userRoleRepo         repository.UserRoleRepository
	roleRepo             repository.RoleRepository
	authContainerRepo    repository.AuthContainerRepository
	identityProviderRepo repository.IdentityProviderRepository
	authClientRepo       repository.AuthClientRepository
}

func NewUserService(
	db *gorm.DB,
	userRepo repository.UserRepository,
	userIdentityRepo repository.UserIdentityRepository,
	userRoleRepo repository.UserRoleRepository,
	roleRepo repository.RoleRepository,
	authContainerRepo repository.AuthContainerRepository,
	identityProviderRepo repository.IdentityProviderRepository,
	authClientRepo repository.AuthClientRepository,
) UserService {
	return &userService{
		db:                   db,
		userRepo:             userRepo,
		userIdentityRepo:     userIdentityRepo,
		userRoleRepo:         userRoleRepo,
		roleRepo:             roleRepo,
		authContainerRepo:    authContainerRepo,
		identityProviderRepo: identityProviderRepo,
		authClientRepo:       authClientRepo,
	}
}

func (s *userService) Get(filter UserServiceGetFilter) (*UserServiceGetResult, error) {
	// Convert auth container UUID to ID if provided
	var authContainerID *int64
	if filter.AuthContainerUUID != nil {
		authContainerUUIDParsed, err := uuid.Parse(*filter.AuthContainerUUID)
		if err != nil {
			return nil, errors.New("invalid auth container UUID")
		}

		authContainer, err := s.authContainerRepo.FindByUUID(authContainerUUIDParsed)
		if err != nil || authContainer == nil {
			return nil, errors.New("auth container not found")
		}
		authContainerID = &authContainer.AuthContainerID
	}

	// Build query filter
	queryFilter := repository.UserRepositoryGetFilter{
		Username:        filter.Username,
		Email:           filter.Email,
		Phone:           filter.Phone,
		IsActive:        filter.IsActive,
		AuthContainerID: authContainerID,
		Page:            filter.Page,
		Limit:           filter.Limit,
		SortBy:          filter.SortBy,
		SortOrder:       filter.SortOrder,
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
	user, err := s.userRepo.FindByUUID(userUUID, "AuthContainer", "UserIdentities.AuthClient", "Roles")
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}

	return toUserServiceDataResult(user), nil
}

func (s *userService) Create(username string, email string, phone string, password string, authContainerUUID string, creatorUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	var createdUser *model.User

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txAuthContainerRepo := s.authContainerRepo.WithTx(tx)
		txAuthClientRepo := s.authClientRepo.WithTx(tx)

		// Parse auth container UUID
		authContainerUUIDParsed, err := uuid.Parse(authContainerUUID)
		if err != nil {
			return errors.New("invalid auth container UUID")
		}

		// Validate auth container exists
		targetAuthContainer, err := txAuthContainerRepo.FindByUUID(authContainerUUIDParsed, "Organization")
		if err != nil || targetAuthContainer == nil {
			return errors.New("auth container not found")
		}

		// Get creator user with auth container and organization info
		creatorUser, err := txUserRepo.FindByUUID(creatorUserUUID, "AuthContainer.Organization")
		if err != nil || creatorUser == nil {
			return errors.New("creator user not found")
		}

		// Validate auth container access permissions
		if err := s.validateAuthContainerAccess(creatorUser, targetAuthContainer); err != nil {
			return err
		}

		// Check if user already exists by username
		existingUser, err := txUserRepo.FindByUsername(username, targetAuthContainer.AuthContainerID)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return errors.New("username already exists")
		}

		// Check if user already exists by email
		existingUser, err = txUserRepo.FindByEmail(email, targetAuthContainer.AuthContainerID)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return errors.New("email already exists")
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		// Create user
		hashedPasswordStr := string(hashedPassword)
		newUser := &model.User{
			Username:        username,
			Email:           email,
			Phone:           phone,
			Password:        &hashedPasswordStr,
			IsActive:        true,
			AuthContainerID: targetAuthContainer.AuthContainerID,
		}

		_, err = txUserRepo.Create(newUser)
		if err != nil {
			return err
		}

		// Find default auth client for this auth container
		defaultAuthClient, err := txAuthClientRepo.FindDefaultByAuthContainerID(targetAuthContainer.AuthContainerID)
		if err != nil || defaultAuthClient == nil {
			return errors.New("default auth client not found for auth container")
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
		createdUser, err = txUserRepo.FindByUUID(newUser.UserUUID, "AuthContainer", "UserIdentities.AuthClient", "Roles")
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

func (s *userService) Update(userUUID uuid.UUID, username string, email string, phone string) (*UserServiceDataResult, error) {
	var updatedUser *model.User

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)

		// Check if user exists
		user, err := txUserRepo.FindByUUID(userUUID)
		if err != nil || user == nil {
			return errors.New("user not found")
		}

		// Check if username is taken by another user
		if username != user.Username {
			existingUser, err := txUserRepo.FindByUsername(username, user.AuthContainerID)
			if err != nil {
				return err
			}
			if existingUser != nil && existingUser.UserID != user.UserID {
				return errors.New("username already exists")
			}
		}

		// Check if email is taken by another user
		if email != user.Email {
			existingUser, err := txUserRepo.FindByEmail(email, user.AuthContainerID)
			if err != nil {
				return err
			}
			if existingUser != nil && existingUser.UserID != user.UserID {
				return errors.New("email already exists")
			}
		}

		// Update user
		user.Username = username
		user.Email = email
		user.Phone = phone

		_, err = txUserRepo.UpdateByUUID(userUUID, user)
		if err != nil {
			return err
		}

		// Fetch updated user with relationships
		updatedUser, err = txUserRepo.FindByUUID(userUUID, "AuthContainer", "UserIdentities.AuthClient", "Roles")
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

func (s *userService) SetActiveStatus(userUUID uuid.UUID, isActive bool) (*UserServiceDataResult, error) {
	// Check if user exists
	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}

	// Update active status
	err = s.userRepo.SetActiveStatus(userUUID, isActive)
	if err != nil {
		return nil, err
	}

	// Fetch updated user with relationships
	updatedUser, err := s.userRepo.FindByUUID(userUUID, "AuthContainer", "UserIdentities.AuthClient", "Roles")
	if err != nil {
		return nil, err
	}

	return toUserServiceDataResult(updatedUser), nil
}

func (s *userService) DeleteByUUID(userUUID uuid.UUID) (*UserServiceDataResult, error) {
	// Check if user exists
	user, err := s.userRepo.FindByUUID(userUUID, "AuthContainer", "UserIdentities.AuthClient", "Roles")
	if err != nil || user == nil {
		return nil, errors.New("user not found")
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
		userWithRoles, err = txUserRepo.FindByUUID(userUUID, "AuthContainer", "UserIdentities.AuthClient", "Roles")
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
		userWithRoles, err = txUserRepo.FindByUUID(userUUID, "AuthContainer", "UserIdentities.AuthClient", "Roles")
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

	// Map AuthContainer if present
	if user.AuthContainer != nil {
		result.AuthContainer = toAuthContainerServiceDataResult(user.AuthContainer)
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

// validateAuthContainerAccess validates if a user can create users in the target auth container
func (s *userService) validateAuthContainerAccess(creatorUser *model.User, targetAuthContainer *model.AuthContainer) error {
	if creatorUser.AuthContainer == nil {
		return errors.New("creator user has no auth container")
	}

	if targetAuthContainer.Organization == nil {
		return errors.New("target auth container has no organization")
	}

	if creatorUser.AuthContainer.Organization == nil {
		return errors.New("creator user's auth container has no organization")
	}

	creatorOrg := creatorUser.AuthContainer.Organization
	targetOrg := targetAuthContainer.Organization

	// Rule 1: Users can always create users in their own auth container
	if creatorUser.AuthContainerID == targetAuthContainer.AuthContainerID {
		return nil
	}

	// Rule 2: Root organization users can create users in any auth container
	if creatorOrg.IsRoot {
		return nil
	}

	// Rule 3: Default organization users can create users in any auth container EXCEPT root
	if creatorOrg.IsDefault {
		if targetOrg.IsRoot {
			return errors.New("users from default organization cannot create users in root organization")
		}
		return nil
	}

	// Rule 4: External organization users can ONLY create users in their own auth container
	if !creatorOrg.IsRoot && !creatorOrg.IsDefault {
		return errors.New("users from external organizations can only create users within their own auth container")
	}

	return errors.New("access denied: cannot create user in target auth container")
}
