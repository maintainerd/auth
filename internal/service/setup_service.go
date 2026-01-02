package service

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/runner"
	"github.com/maintainerd/auth/internal/util"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SetupService interface {
	GetSetupStatus() (*dto.SetupStatusResponseDto, error)
	CreateTenant(req dto.CreateTenantRequestDto) (*dto.CreateTenantResponseDto, error)
	CreateAdmin(req dto.CreateAdminRequestDto) (*dto.CreateAdminResponseDto, error)
	CreateProfile(req dto.CreateProfileRequestDto) (*dto.CreateProfileResponseDto, error)
}

type setupService struct {
	db                   *gorm.DB
	userRepo             repository.UserRepository
	tenantRepo           repository.TenantRepository
	tenantMemberRepo     repository.TenantMemberRepository
	authClientRepo       repository.AuthClientRepository
	identityProviderRepo repository.IdentityProviderRepository
	roleRepo             repository.RoleRepository
	userRoleRepo         repository.UserRoleRepository
	userTokenRepo        repository.UserTokenRepository
	userIdentityRepo     repository.UserIdentityRepository
	profileRepo          repository.ProfileRepository
}

func NewSetupService(
	db *gorm.DB,
	userRepo repository.UserRepository,
	tenantRepo repository.TenantRepository,
	tenantMemberRepo repository.TenantMemberRepository,
	authClientRepo repository.AuthClientRepository,
	identityProviderRepo repository.IdentityProviderRepository,
	roleRepo repository.RoleRepository,
	userRoleRepo repository.UserRoleRepository,
	userTokenRepo repository.UserTokenRepository,
	userIdentityRepo repository.UserIdentityRepository,
	profileRepo repository.ProfileRepository,
) SetupService {
	return &setupService{
		db:                   db,
		userRepo:             userRepo,
		tenantRepo:           tenantRepo,
		tenantMemberRepo:     tenantMemberRepo,
		authClientRepo:       authClientRepo,
		identityProviderRepo: identityProviderRepo,
		roleRepo:             roleRepo,
		userRoleRepo:         userRoleRepo,
		userTokenRepo:        userTokenRepo,
		userIdentityRepo:     userIdentityRepo,
		profileRepo:          profileRepo,
	}
}

func (s *setupService) GetSetupStatus() (*dto.SetupStatusResponseDto, error) {
	// Check if tenant exists
	tenants, err := s.tenantRepo.FindAll()
	if err != nil {
		return nil, err
	}
	isTenantSetup := len(tenants) > 0

	// Check if admin user exists (super-admin role in default tenant)
	isAdminSetup := false
	isProfileSetup := false

	if isTenantSetup {
		// Find default tenant
		defaultTenant, err := s.tenantRepo.FindDefault()
		if err == nil && defaultTenant != nil {
			// Check if super-admin user exists
			superAdmin, err := s.userRepo.FindSuperAdmin()
			if err == nil && superAdmin != nil {
				isAdminSetup = true

				// Check if profile exists for admin user
				existingProfile, err := s.profileRepo.FindByUserID(superAdmin.UserID)
				if err == nil && existingProfile != nil {
					isProfileSetup = true
				}
			}
		}
	}

	return &dto.SetupStatusResponseDto{
		IsTenantSetup:   isTenantSetup,
		IsAdminSetup:    isAdminSetup,
		IsProfileSetup:  isProfileSetup,
		IsSetupComplete: isTenantSetup && isAdminSetup && isProfileSetup,
	}, nil
}

func (s *setupService) CreateTenant(req dto.CreateTenantRequestDto) (*dto.CreateTenantResponseDto, error) {
	// Check if tenant already exists
	tenants, err := s.tenantRepo.FindAll()
	if err != nil {
		return nil, err
	}
	if len(tenants) > 0 {
		return nil, errors.New("tenant already exists: setup can only be run once")
	}

	var createdTenant *model.Tenant
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txTenantRepo := s.tenantRepo.WithTx(tx)

		// Generate identifier
		identifier := util.GenerateIdentifier(24)

		// Handle description (optional field)
		description := ""
		if req.Description != nil {
			description = *req.Description
		}

		// Handle metadata (optional field)
		var metadataJSON datatypes.JSON
		if req.Metadata != nil {
			metadataBytes, err := json.Marshal(req.Metadata)
			if err != nil {
				return errors.New("invalid metadata format")
			}
			metadataJSON = metadataBytes
		} else {
			metadataJSON = datatypes.JSON([]byte("{}"))
		}

		// Create tenant directly (no longer using seeder)
		newTenant := &model.Tenant{
			Name:        req.Name,
			DisplayName: req.DisplayName,
			Description: description,
			Identifier:  identifier,
			Metadata:    metadataJSON,
			Status:      "active",
			IsDefault:   true, // This is the default tenant for new users
			IsSystem:    true, // This is a system tenant that cannot be deleted
		}

		var err error
		createdTenant, err = txTenantRepo.Create(newTenant)
		if err != nil {
			return err
		}

		// Run all other seeders
		if err := runner.RunSeeders(tx, "v0.1.0"); err != nil {
			return errors.New("failed to initialize tenant structure: " + err.Error())
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert to response DTO
	var metadata any
	if len(createdTenant.Metadata) > 0 {
		if err := json.Unmarshal(createdTenant.Metadata, &metadata); err == nil {
			// Only include if unmarshaling was successful
		} else {
			metadata = nil
		}
	}

	tenantResponse := dto.TenantResponseDto{
		TenantUUID:  createdTenant.TenantUUID,
		Name:        createdTenant.Name,
		Description: createdTenant.Description,
		Identifier:  createdTenant.Identifier,
		Status:      createdTenant.Status,
		IsPublic:    createdTenant.IsPublic,
		IsDefault:   createdTenant.IsDefault,
		IsSystem:    createdTenant.IsSystem,
		Metadata:    metadata,
		CreatedAt:   createdTenant.CreatedAt,
		UpdatedAt:   createdTenant.UpdatedAt,
	}

	// Get default auth client and identity provider for user reference
	defaultClient, err := s.authClientRepo.FindDefault()
	if err != nil {
		return nil, err
	}

	var defaultClientID, defaultProviderID string
	if defaultClient != nil && defaultClient.ClientID != nil {
		defaultClientID = *defaultClient.ClientID
		if defaultClient.IdentityProvider != nil {
			defaultProviderID = defaultClient.IdentityProvider.Identifier
		}
	}

	return &dto.CreateTenantResponseDto{
		Tenant:            tenantResponse,
		DefaultClientID:   defaultClientID,
		DefaultProviderID: defaultProviderID,
	}, nil
}

func (s *setupService) CreateAdmin(req dto.CreateAdminRequestDto) (*dto.CreateAdminResponseDto, error) {
	// Check if tenant exists
	tenants, err := s.tenantRepo.FindAll()
	if err != nil {
		return nil, err
	}
	if len(tenants) == 0 {
		return nil, errors.New("tenant must be created first")
	}

	// Check if admin already exists
	superAdmin, err := s.userRepo.FindSuperAdmin()
	if err != nil {
		return nil, err
	}
	if superAdmin != nil {
		return nil, errors.New("admin user already exists: setup can only be run once")
	}

	// Get default tenant
	defaultTenant, err := s.tenantRepo.FindDefault()
	if err != nil {
		return nil, err
	}
	if defaultTenant == nil {
		return nil, errors.New("default tenant not found")
	}

	// Get default auth client
	defaultClient, err := s.authClientRepo.FindDefault()
	if err != nil {
		return nil, err
	}
	if defaultClient == nil {
		return nil, errors.New("default auth client not found")
	}

	var createdUser *model.User
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txTenantMemberRepo := s.tenantMemberRepo.WithTx(tx)

		// Check if user already exists
		existingUser, err := txUserRepo.FindByEmail(req.Email, defaultTenant.TenantID)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return errors.New("user with this email already exists")
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		// Create admin user
		newUser := &model.User{
			Username:        req.Username,
			Fullname:        req.Fullname,
			Email:           req.Email,
			Password:        util.Ptr(string(hashedPassword)),
			TenantID:        defaultTenant.TenantID,
			IsEmailVerified: true,
			Status:          "active",
		}

		createdUser, err = txUserRepo.Create(newUser)
		if err != nil {
			return err
		}

		// Create user identity
		userIdentity := &model.UserIdentity{
			UserID:       createdUser.UserID,
			AuthClientID: defaultClient.AuthClientID,
			Provider:     "default",
			Sub:          createdUser.UserUUID.String(),
		}
		_, err = txUserIdentityRepo.Create(userIdentity)
		if err != nil {
			return err
		}

		// Get registered role for setup admin
		registeredRole, err := txRoleRepo.FindRegisteredRoleForSetup(defaultTenant.TenantID)
		if err != nil {
			return err
		}
		if registeredRole == nil {
			return errors.New("registered role not found (is_default=true, is_system=true, name='registered')")
		}

		// Assign registered role
		registeredUserRole := &model.UserRole{
			UserID: createdUser.UserID,
			RoleID: registeredRole.RoleID,
		}
		_, err = txUserRoleRepo.Create(registeredUserRole)
		if err != nil {
			return err
		}

		// Get super-admin system role
		superAdminRole, err := txRoleRepo.FindSuperAdminRoleForSetup(defaultTenant.TenantID)
		if err != nil {
			return err
		}
		if superAdminRole == nil {
			return errors.New("super-admin role not found (is_system=true, name='super-admin')")
		}

		// Assign super-admin role
		superAdminUserRole := &model.UserRole{
			UserID: createdUser.UserID,
			RoleID: superAdminRole.RoleID,
		}
		_, err = txUserRoleRepo.Create(superAdminUserRole)
		if err != nil {
			return err
		}

		// Add user to tenant_members as owner
		tenantMember := &model.TenantMember{
			TenantID: defaultTenant.TenantID,
			UserID:   createdUser.UserID,
			Role:     "owner",
		}
		_, err = txTenantMemberRepo.Create(tenantMember)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert to response DTO
	userResponse := dto.UserResponseDto{
		UserUUID:        createdUser.UserUUID,
		Username:        createdUser.Username,
		Fullname:        createdUser.Fullname,
		Email:           createdUser.Email,
		IsEmailVerified: createdUser.IsEmailVerified,
		Status:          createdUser.Status,
		CreatedAt:       createdUser.CreatedAt,
		UpdatedAt:       createdUser.UpdatedAt,
	}

	return &dto.CreateAdminResponseDto{
		User: userResponse,
	}, nil
}

// Helper function to get string value from pointer
func (s *setupService) CreateProfile(req dto.CreateProfileRequestDto) (*dto.CreateProfileResponseDto, error) {
	// Find the super admin user (the only user during setup)
	user, err := s.userRepo.FindSuperAdmin()
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("no admin user found")
	}

	// Check if profile already exists for this user
	existingProfile, err := s.profileRepo.FindByUserID(user.UserID)
	if err != nil {
		return nil, err
	}

	if existingProfile != nil {
		return nil, errors.New("profile already exists for this user")
	}

	// Parse birthdate if provided
	var birthdate *time.Time
	if req.Birthdate != nil && *req.Birthdate != "" {
		parsed, err := time.Parse("2006-01-02", *req.Birthdate)
		if err != nil {
			return nil, errors.New("invalid birthdate format, must be YYYY-MM-DD")
		}
		birthdate = &parsed
	}

	var createdProfile *model.Profile
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txProfileRepo := s.profileRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Handle metadata (optional field with custom fields)
		var metadataJSON datatypes.JSON
		if req.Metadata != nil {
			metadataBytes, err := json.Marshal(req.Metadata)
			if err != nil {
				return errors.New("invalid metadata format")
			}
			metadataJSON = metadataBytes
		} else {
			metadataJSON = datatypes.JSON([]byte("{}"))
		}

		// Create new profile
		newProfile := &model.Profile{
			UserID:      user.UserID,
			FirstName:   req.FirstName,
			MiddleName:  req.MiddleName,
			LastName:    req.LastName,
			Suffix:      req.Suffix,
			DisplayName: req.DisplayName,
			Bio:         req.Bio,
			IsDefault:   true, // First profile during setup is always default
			Birthdate:   birthdate,
			Gender:      req.Gender,
			Phone:       req.Phone,
			Email:       req.Email,
			Address:     req.Address,
			City:        req.City,
			Country:     req.Country,
			Timezone:    req.Timezone,
			Language:    req.Language,
			ProfileURL:  req.ProfileURL,
			Metadata:    metadataJSON,
		}

		createdProfile, err = txProfileRepo.Create(newProfile)
		if err != nil {
			return err
		}

		// Update user's is_profile_completed flag
		_, err = txUserRepo.UpdateByUUID(user.UserUUID, map[string]interface{}{
			"is_profile_completed": true,
			"is_account_completed": true,
		})
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert to response DTO using the existing helper function
	profileResponse := dto.NewProfileResponse(createdProfile)

	return &dto.CreateProfileResponseDto{
		Profile: *profileResponse,
	}, nil
}
