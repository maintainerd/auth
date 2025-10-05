package service

import (
	"errors"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/runner"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type SetupService interface {
	GetSetupStatus() (*dto.SetupStatusResponseDto, error)
	CreateOrganization(req dto.CreateOrganizationRequestDto) (*dto.CreateOrganizationResponseDto, error)
	CreateAdmin(req dto.CreateAdminRequestDto) (*dto.CreateAdminResponseDto, error)
}

type setupService struct {
	db                   *gorm.DB
	organizationRepo     repository.OrganizationRepository
	userRepo             repository.UserRepository
	authContainerRepo    repository.AuthContainerRepository
	authClientRepo       repository.AuthClientRepository
	identityProviderRepo repository.IdentityProviderRepository
	roleRepo             repository.RoleRepository
	userRoleRepo         repository.UserRoleRepository
	userTokenRepo        repository.UserTokenRepository
}

func NewSetupService(
	db *gorm.DB,
	organizationRepo repository.OrganizationRepository,
	userRepo repository.UserRepository,
	authContainerRepo repository.AuthContainerRepository,
	authClientRepo repository.AuthClientRepository,
	identityProviderRepo repository.IdentityProviderRepository,
	roleRepo repository.RoleRepository,
	userRoleRepo repository.UserRoleRepository,
	userTokenRepo repository.UserTokenRepository,
) SetupService {
	return &setupService{
		db:                   db,
		organizationRepo:     organizationRepo,
		userRepo:             userRepo,
		authContainerRepo:    authContainerRepo,
		authClientRepo:       authClientRepo,
		identityProviderRepo: identityProviderRepo,
		roleRepo:             roleRepo,
		userRoleRepo:         userRoleRepo,
		userTokenRepo:        userTokenRepo,
	}
}

func (s *setupService) GetSetupStatus() (*dto.SetupStatusResponseDto, error) {
	// Check if organization exists
	organizations, err := s.organizationRepo.FindAll()
	if err != nil {
		return nil, err
	}
	isOrganizationSetup := len(organizations) > 0

	// Check if admin user exists (super-admin role in default auth container)
	isAdminSetup := false
	if isOrganizationSetup {
		// Find default auth container
		defaultContainer, err := s.authContainerRepo.FindDefault()
		if err == nil && defaultContainer != nil {
			// Check if super-admin user exists
			superAdmin, err := s.userRepo.FindSuperAdmin()
			if err == nil && superAdmin != nil {
				isAdminSetup = true
			}
		}
	}

	return &dto.SetupStatusResponseDto{
		IsOrganizationSetup: isOrganizationSetup,
		IsAdminSetup:        isAdminSetup,
		IsSetupComplete:     isOrganizationSetup && isAdminSetup,
	}, nil
}

func (s *setupService) CreateOrganization(req dto.CreateOrganizationRequestDto) (*dto.CreateOrganizationResponseDto, error) {
	// Check if organization already exists
	organizations, err := s.organizationRepo.FindAll()
	if err != nil {
		return nil, err
	}
	if len(organizations) > 0 {
		return nil, errors.New("organization already exists: setup can only be run once")
	}

	var createdOrg *model.Organization
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txOrgRepo := s.organizationRepo.WithTx(tx)

		// Create organization directly (no longer using seeder)
		newOrg := &model.Organization{
			Name:        req.Name,
			Description: req.Description,
			Email:       req.Email,
			Phone:       req.Phone,
			IsActive:    true,
		}

		var err error
		createdOrg, err = txOrgRepo.Create(newOrg)
		if err != nil {
			return err
		}

		// Run all other seeders (excluding organization)
		if err := runner.RunSeeders(tx, "v0.1.0", createdOrg.OrganizationID); err != nil {
			return errors.New("failed to initialize organization structure: " + err.Error())
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert to response DTO
	orgResponse := dto.OrganizationResponseDto{
		OrganizationUUID: createdOrg.OrganizationUUID,
		Name:             createdOrg.Name,
		Description:      getStringValue(createdOrg.Description),
		Email:            getStringValue(createdOrg.Email),
		Phone:            getStringValue(createdOrg.Phone),
		IsActive:         createdOrg.IsActive,
		CreatedAt:        createdOrg.CreatedAt,
		UpdatedAt:        createdOrg.UpdatedAt,
	}

	return &dto.CreateOrganizationResponseDto{
		Message:      "Organization created successfully",
		Organization: orgResponse,
	}, nil
}

func (s *setupService) CreateAdmin(req dto.CreateAdminRequestDto) (*dto.CreateAdminResponseDto, error) {
	// Check if organization exists
	organizations, err := s.organizationRepo.FindAll()
	if err != nil {
		return nil, err
	}
	if len(organizations) == 0 {
		return nil, errors.New("organization must be created first")
	}

	// Check if admin already exists
	superAdmin, err := s.userRepo.FindSuperAdmin()
	if err != nil {
		return nil, err
	}
	if superAdmin != nil {
		return nil, errors.New("admin user already exists: setup can only be run once")
	}

	// Get default auth container
	defaultContainer, err := s.authContainerRepo.FindDefault()
	if err != nil {
		return nil, err
	}
	if defaultContainer == nil {
		return nil, errors.New("default auth container not found")
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

		// Check if user already exists
		existingUser, err := txUserRepo.FindByEmail(req.Email, defaultContainer.AuthContainerID)
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
			Email:           req.Email,
			Password:        ptr(string(hashedPassword)),
			AuthContainerID: defaultContainer.AuthContainerID,
			IsEmailVerified: true,
			IsActive:        true,
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
		if err := tx.Create(userIdentity).Error; err != nil {
			return err
		}

		// Get super-admin role
		superAdminRole, err := txRoleRepo.FindByNameAndAuthContainerID("super-admin", defaultContainer.AuthContainerID)
		if err != nil {
			return err
		}
		if superAdminRole == nil {
			return errors.New("super-admin role not found")
		}

		// Assign super-admin role
		userRole := &model.UserRole{
			UserID: createdUser.UserID,
			RoleID: superAdminRole.RoleID,
		}
		_, err = txUserRoleRepo.Create(userRole)
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
		Email:           createdUser.Email,
		IsEmailVerified: createdUser.IsEmailVerified,
		IsActive:        createdUser.IsActive,
		CreatedAt:       createdUser.CreatedAt,
		UpdatedAt:       createdUser.UpdatedAt,
	}

	return &dto.CreateAdminResponseDto{
		Message: "Admin user created successfully",
		User:    userResponse,
	}, nil
}

// Helper function to get string value from pointer
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Helper function to create string pointer
func ptr(s string) *string {
	return &s
}
