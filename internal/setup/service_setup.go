package setup

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/platform/runner"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/setup/seeder"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var setupHashPassword = security.HashPassword
var setupRunSeeders = runner.RunSeeders

type SetupService interface {
	GetSetupStatus(ctx context.Context) (*SetupStatusResponseDTO, error)
	CreateTenant(ctx context.Context, req CreateTenantRequestDTO) (*CreateTenantResponseDTO, error)
	CreateAdmin(ctx context.Context, req CreateAdminRequestDTO) (*CreateAdminResponseDTO, error)
	RegisterControlService(ctx context.Context, req RegisterControlServiceRequestDTO) (*RegisterControlServiceResponseDTO, error)
	CompleteSetup(ctx context.Context) (*CompleteSetupResponseDTO, error)
}

type setupService struct {
	db                *gorm.DB
	userRepo          UserRepository
	tenantRepo        TenantRepository
	tenantMemberRepo  TenantMemberRepository
	clientRepo        ClientRepository
	roleRepo          RoleRepository
	userRoleRepo      UserRoleRepository
	userIdentityRepo  UserIdentityRepository
	profileRepo       ProfileRepository
	serviceRepo       ServiceRepository
	policyRepo        PolicyRepository
	servicePolicyRepo ServicePolicyRepository
}

type ControlRegistrationDeps struct {
	ServiceRepo       ServiceRepository
	PolicyRepo        PolicyRepository
	ServicePolicyRepo ServicePolicyRepository
}

func NewSetupService(
	db *gorm.DB,
	userRepo UserRepository,
	tenantRepo TenantRepository,
	tenantMemberRepo TenantMemberRepository,
	clientRepo ClientRepository,
	roleRepo RoleRepository,
	userRoleRepo UserRoleRepository,
	userIdentityRepo UserIdentityRepository,
	profileRepo ProfileRepository,
	setupOptions ...any,
) SetupService {
	controlDeps := ControlRegistrationDeps{}
	for _, option := range setupOptions {
		if value, ok := option.(ControlRegistrationDeps); ok {
			controlDeps = value
		}
	}
	return &setupService{
		db:                db,
		userRepo:          userRepo,
		tenantRepo:        tenantRepo,
		tenantMemberRepo:  tenantMemberRepo,
		clientRepo:        clientRepo,
		roleRepo:          roleRepo,
		userRoleRepo:      userRoleRepo,
		userIdentityRepo:  userIdentityRepo,
		profileRepo:       profileRepo,
		serviceRepo:       controlDeps.ServiceRepo,
		policyRepo:        controlDeps.PolicyRepo,
		servicePolicyRepo: controlDeps.ServicePolicyRepo,
	}
}

func (s *setupService) GetSetupStatus(ctx context.Context) (*SetupStatusResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "setup.getStatus")
	defer span.End()

	// Check if tenant exists
	tenants, err := s.tenantRepo.FindAll()
	if err != nil {
		return nil, err
	}
	isTenantSetup := len(tenants) > 0

	// Check if admin user exists (super-admin role in default tenant)
	isAdminSetup := false
	isProfileSetup := false
	// Setup completion is now tracked on the system tenant itself: the bootstrap
	// is "complete" once the system tenant is marked is_completed (which happens
	// once it has an admin + owner). No separate setup_states table.
	isSetupComplete := false

	if isTenantSetup {
		// Find default tenant
		defaultTenant, err := s.tenantRepo.FindSystem()
		if err == nil && defaultTenant != nil {
			isSetupComplete = defaultTenant.IsCompleted

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

	span.SetStatus(codes.Ok, "")
	return &SetupStatusResponseDTO{
		IsTenantSetup:   isTenantSetup,
		IsAdminSetup:    isAdminSetup,
		IsProfileSetup:  isProfileSetup,
		IsSetupComplete: isSetupComplete,
	}, nil
}

func (s *setupService) CreateTenant(ctx context.Context, req CreateTenantRequestDTO) (*CreateTenantResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "setup.createTenant")
	defer span.End()

	if err := s.ensureSetupOpen(); err != nil {
		return nil, err
	}

	// Check if tenant already exists
	tenants, err := s.tenantRepo.FindAll()
	if err != nil {
		return nil, err
	}
	if len(tenants) > 0 {
		return nil, apperror.NewConflict("tenant already exists: setup can only be run once")
	}

	var createdTenant *Tenant
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txTenantRepo := s.tenantRepo.WithTx(tx)

		// Generate identifier
		identifier, err := crypto.GenerateIdentifier(24)
		if err != nil {
			return err
		}

		// Handle description (optional field)
		description := ""
		if req.Description != nil {
			description = *req.Description
		}

		// Handle metadata (optional field)
		var metadataJSON datatypes.JSON
		if req.Metadata != nil {
			metadataJSON, _ = json.Marshal(req.Metadata)
		} else {
			metadataJSON = datatypes.JSON([]byte("{}"))
		}

		// Create tenant directly (no longer using seeder)
		newTenant := &Tenant{
			Name:        req.Name,
			DisplayName: req.DisplayName,
			Description: description,
			Identifier:  identifier,
			Metadata:    metadataJSON,
			Status:      shared.StatusActive,
			IsSystem:    true, // This is a system tenant that cannot be deleted
			// Not yet complete: bootstrap finishes (admin + owner created) before
			// CompleteSetup flips this to true.
			IsCompleted: false,
		}

		createdTenant, err = txTenantRepo.Create(newTenant)
		if err != nil {
			return err
		}

		// Run all other seeders
		if err := setupRunSeeders(tx, "v0.1.0"); err != nil {
			return apperror.NewInternal("failed to initialize tenant structure", err)
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create tenant failed")
		return nil, err
	}

	// Convert to response DTO
	tenantResponse := TenantResponseDTO{
		TenantUUID:  createdTenant.TenantUUID,
		Name:        createdTenant.Name,
		DisplayName: createdTenant.DisplayName,
		Description: createdTenant.Description,
		Identifier:  createdTenant.Identifier,
		Status:      createdTenant.Status,
		IsSystem:    createdTenant.IsSystem,
		Metadata:    createdTenant.Metadata,
		CreatedAt:   createdTenant.CreatedAt,
		UpdatedAt:   createdTenant.UpdatedAt,
	}

	// Get system client and identity provider for user reference
	defaultClient, err := s.clientRepo.FindSystem()
	if err != nil {
		return nil, err
	}

	var defaultClientID, defaultProviderID string
	if defaultClient != nil && defaultClient.Identifier != nil {
		defaultClientID = *defaultClient.Identifier
		if defaultClient.IdentityProvider != nil {
			defaultProviderID = defaultClient.IdentityProvider.Identifier
		}
	}

	span.SetStatus(codes.Ok, "")
	return &CreateTenantResponseDTO{
		Tenant:            tenantResponse,
		DefaultClientID:   defaultClientID,
		DefaultProviderID: defaultProviderID,
	}, nil
}

func (s *setupService) CreateAdmin(ctx context.Context, req CreateAdminRequestDTO) (*CreateAdminResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "setup.createAdmin")
	defer span.End()

	if err := s.ensureSetupOpen(); err != nil {
		return nil, err
	}

	// Check if tenant exists
	tenants, err := s.tenantRepo.FindAll()
	if err != nil {
		return nil, err
	}
	if len(tenants) == 0 {
		return nil, apperror.NewValidation("tenant must be created first")
	}

	// Check if admin already exists
	superAdmin, err := s.userRepo.FindSuperAdmin()
	if err != nil {
		return nil, err
	}
	if superAdmin != nil {
		return nil, apperror.NewConflict("admin user already exists: setup can only be run once")
	}

	// Get default tenant
	defaultTenant, err := s.tenantRepo.FindSystem()
	if err != nil {
		return nil, err
	}
	if defaultTenant == nil {
		return nil, apperror.NewNotFoundWithReason("default tenant not found")
	}

	// Bind the super-admin's identity to the seeded auth-console system client
	// by explicit name. The console is the only surface that exists at boot.
	defaultClient, err := s.clientRepo.FindByNameAndTenantID(shared.SystemClientNameAuthConsole, defaultTenant.TenantID)
	if err != nil {
		return nil, err
	}
	if defaultClient == nil {
		return nil, apperror.NewNotFoundWithReason("auth-console system client not found")
	}

	var createdUser *User
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txTenantMemberRepo := s.tenantMemberRepo.WithTx(tx)
		txTenantRepo := s.tenantRepo.WithTx(tx)

		// Check if user already exists (scoped to the system tenant)
		existingUser, err := txUserRepo.FindByEmailAndTenantID(req.Email, defaultTenant.TenantID)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return apperror.NewConflict("user with this email already exists")
		}

		// Hash password
		hashedPassword, err := setupHashPassword(ctx, []byte(req.Password))
		if err != nil {
			return err
		}

		// Create admin user
		now := time.Now()
		fullname := req.Username
		if req.Fullname != nil && *req.Fullname != "" {
			fullname = *req.Fullname
		}
		newUser := &User{
			TenantID:          defaultTenant.TenantID,
			Username:          req.Username,
			Fullname:          fullname,
			Email:             req.Email,
			Password:          ptr.Ptr(string(hashedPassword)),
			IsEmailVerified:   true,
			Status:            shared.StatusActive,
			PasswordChangedAt: &now,
		}

		createdUser, err = txUserRepo.Create(newUser)
		if err != nil {
			return err
		}

		// The client's identity-provider link now lives in the
		// client_identity_providers join table — the clients row no longer carries
		// identity_provider_id, and FindByNameAndTenantID does not resolve the
		// transient Client.IdentityProviderID field. Persist it only when known;
		// NULL is valid for the built-in maintainerd identity (the FK
		// fk_user_identities_idp is nullable). Passing &0 here would point at a
		// non-existent provider and violate the foreign key.
		var identityProviderID *int64
		if defaultClient.IdentityProviderID != 0 {
			id := defaultClient.IdentityProviderID
			identityProviderID = &id
		}

		// Create user identity
		userIdentity := &UserIdentity{
			TenantID:           defaultTenant.TenantID,
			UserID:             createdUser.UserID,
			ClientID:           defaultClient.ClientID,
			IdentityProviderID: identityProviderID,
			Provider:           shared.ProviderMaintainerd,
			Sub:                uuid.New().String(),
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
			return apperror.NewNotFoundWithReason("registered role not found (is_default=true, is_system=true, name='registered')")
		}

		// Assign registered role
		registeredUserRole := &UserRole{
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
			return apperror.NewNotFoundWithReason("super-admin role not found (is_system=true, name='super-admin')")
		}

		// Assign super-admin role
		superAdminUserRole := &UserRole{
			UserID: createdUser.UserID,
			RoleID: superAdminRole.RoleID,
		}
		_, err = txUserRoleRepo.Create(superAdminUserRole)
		if err != nil {
			return err
		}

		// Add user to tenant_members as owner
		tenantMember := &TenantMember{
			TenantID: defaultTenant.TenantID,
			UserID:   createdUser.UserID,
			Role:     "owner",
		}
		_, err = txTenantMemberRepo.Create(tenantMember)
		if err != nil {
			return err
		}

		// Mark the system tenant as completed — admin + owner now exist.
		defaultTenant.IsCompleted = true
		if _, err := txTenantRepo.CreateOrUpdate(defaultTenant); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create admin failed")
		return nil, err
	}

	// Convert to response DTO
	userResponse := UserResponseDTO{
		UserUUID:        createdUser.UserUUID,
		Username:        createdUser.Username,
		Fullname:        createdUser.Fullname,
		Email:           createdUser.Email,
		IsEmailVerified: createdUser.IsEmailVerified,
		Status:          createdUser.Status,
		CreatedAt:       createdUser.CreatedAt,
		UpdatedAt:       createdUser.UpdatedAt,
	}

	span.SetStatus(codes.Ok, "")
	return &CreateAdminResponseDTO{
		User: userResponse,
	}, nil
}

// Helper function to get string value from pointer

func (s *setupService) RegisterControlService(ctx context.Context, req RegisterControlServiceRequestDTO) (*RegisterControlServiceResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "setup.registerControlService")
	defer span.End()

	if err := s.ensureSetupOpen(); err != nil {
		return nil, err
	}
	if s.serviceRepo == nil || s.policyRepo == nil || s.servicePolicyRepo == nil || s.db == nil {
		err := apperror.NewInternal("control registration dependencies are not configured", errors.New("missing setup control registration dependencies"))
		span.RecordError(err)
		span.SetStatus(codes.Error, "register control service failed")
		return nil, err
	}

	sysTenant, err := s.tenantRepo.FindSystem()
	if err != nil {
		return nil, err
	}
	if sysTenant == nil {
		return nil, apperror.NewValidation("tenant must be created first")
	}

	version := req.Version
	if version == "" {
		version = "v1"
	}
	description := ""
	if req.Description != nil {
		description = *req.Description
	}

	var registeredService *Service
	var controlPolicy *Policy
	var alreadyExisted bool
	var policyWasAttached bool

	err = s.db.Transaction(func(tx *gorm.DB) error {
		txServiceRepo := s.serviceRepo.WithTx(tx)
		txPolicyRepo := s.policyRepo.WithTx(tx)
		txServicePolicyRepo := s.servicePolicyRepo.WithTx(tx)

		policy, err := txPolicyRepo.FindByNameAndVersion(seeder.SystemControlPolicyName, "v1", sysTenant.TenantID)
		if err != nil {
			return err
		}
		if policy == nil {
			return apperror.NewValidation("control policy is not seeded")
		}
		controlPolicy = policy

		service, err := txServiceRepo.FindByNameAndTenantID(req.Name, sysTenant.TenantID)
		if err != nil {
			return err
		}
		if service != nil {
			alreadyExisted = true
			registeredService = service
		} else {
			service = &Service{
				ServiceUUID: uuid.New(),
				Name:        req.Name,
				DisplayName: req.DisplayName,
				Description: description,
				Version:     version,
				Status:      shared.StatusActive,
				IsSystem:    false,
			}
			if _, err := txServiceRepo.CreateOrUpdate(service); err != nil {
				return err
			}
			if err := tx.Create(&TenantServiceLink{TenantID: sysTenant.TenantID, ServiceID: service.ServiceID}).Error; err != nil {
				return err
			}
			registeredService = service
		}

		existingAttachment, err := txServicePolicyRepo.FindByServiceAndPolicy(registeredService.ServiceID, policy.PolicyID)
		if err != nil {
			return err
		}
		if existingAttachment == nil {
			_, err = txServicePolicyRepo.Create(&ServicePolicy{
				ServicePolicyUUID: uuid.New(),
				ServiceID:         registeredService.ServiceID,
				PolicyID:          policy.PolicyID,
			})
			if err != nil {
				return err
			}
			policyWasAttached = true
		}

		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "register control service failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return &RegisterControlServiceResponseDTO{
		ServiceUUID:       registeredService.ServiceUUID.String(),
		Name:              registeredService.Name,
		DisplayName:       registeredService.DisplayName,
		PolicyUUID:        controlPolicy.PolicyUUID.String(),
		PolicyName:        controlPolicy.Name,
		AlreadyExisted:    alreadyExisted,
		PolicyWasAttached: policyWasAttached,
	}, nil
}

func (s *setupService) CompleteSetup(ctx context.Context) (*CompleteSetupResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "setup.complete")
	defer span.End()

	status, err := s.GetSetupStatus(ctx)
	if err != nil {
		return nil, err
	}
	if status.IsSetupComplete {
		return &CompleteSetupResponseDTO{IsSetupComplete: true}, nil
	}
	if !status.IsTenantSetup || !status.IsAdminSetup || !status.IsProfileSetup {
		return nil, apperror.NewValidation("tenant, admin, and profile setup must be completed before locking setup")
	}

	// Mark the system tenant as completed — this is what records that bootstrap
	// finished (the system tenant now has an admin + owner).
	systemTenant, err := s.tenantRepo.FindSystem()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "complete setup failed")
		return nil, err
	}
	if systemTenant == nil {
		return nil, apperror.NewValidation("system tenant not found")
	}
	systemTenant.IsCompleted = true
	if _, err := s.tenantRepo.CreateOrUpdate(systemTenant); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "complete setup failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return &CompleteSetupResponseDTO{IsSetupComplete: true}, nil
}

func (s *setupService) ensureSetupOpen() error {
	// Setup is open until the system tenant exists and is marked completed.
	systemTenant, err := s.tenantRepo.FindSystem()
	if err != nil {
		// No system tenant yet → setup is still open.
		return nil
	}
	if systemTenant != nil && systemTenant.IsCompleted {
		return apperror.NewConflict("setup is complete and locked")
	}
	return nil
}
