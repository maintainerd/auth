package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/platform/runner"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/maintainerd/maintainerd-auth/internal/user"
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
	CreateProfile(ctx context.Context, req CreateProfileRequestDTO) (*CreateProfileResponseDTO, error)
	RegisterControlService(ctx context.Context, req RegisterControlServiceRequestDTO) (*RegisterControlServiceResponseDTO, error)
	EnsureControlClient(ctx context.Context, req EnsureControlClientRequestDTO) (*EnsureControlClientResponseDTO, error)
	EnsureResourceAPI(ctx context.Context, req EnsureResourceAPIRequestDTO) (*EnsureResourceAPIResponseDTO, error)
	EnsureRole(ctx context.Context, req EnsureRoleRequestDTO) (*EnsureRoleResponseDTO, error)
	EnsureConsoleClient(ctx context.Context, req EnsureConsoleClientRequestDTO) (*EnsureConsoleClientResponseDTO, error)
	CompleteSetup(ctx context.Context) (*CompleteSetupResponseDTO, error)
}

type setupService struct {
	db                 *gorm.DB
	userRepo           UserRepository
	tenantRepo         TenantRepository
	tenantMemberRepo   TenantMemberRepository
	clientRepo         ClientRepository
	roleRepo           RoleRepository
	userRoleRepo       UserRoleRepository
	userIdentityRepo   UserIdentityRepository
	profileRepo        ProfileRepository
	serviceRepo        ServiceRepository
	policyRepo         PolicyRepository
	servicePolicyRepo  ServicePolicyRepository
	apiRepo            APIRepository
	permissionRepo     PermissionRepository
	rolePermissionRepo RolePermissionRepository
	clientURIRepo      ClientURIRepository
}

// ControlRegistrationDeps carries the repositories the orchestrator-provisioning
// RPCs need. They are grouped rather than added to the already-long constructor
// so a caller that does not provision (the REST wizard) can omit them, and
// requireProvisioningDeps then refuses those RPCs instead of half-running them.
type ControlRegistrationDeps struct {
	ServiceRepo        ServiceRepository
	PolicyRepo         PolicyRepository
	ServicePolicyRepo  ServicePolicyRepository
	APIRepo            APIRepository
	PermissionRepo     PermissionRepository
	RolePermissionRepo RolePermissionRepository
	ClientURIRepo      ClientURIRepository
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
		switch value := option.(type) {
		case ControlRegistrationDeps:
			controlDeps = value
		}
	}
	return &setupService{
		db:                 db,
		userRepo:           userRepo,
		tenantRepo:         tenantRepo,
		tenantMemberRepo:   tenantMemberRepo,
		clientRepo:         clientRepo,
		roleRepo:           roleRepo,
		userRoleRepo:       userRoleRepo,
		userIdentityRepo:   userIdentityRepo,
		profileRepo:        profileRepo,
		serviceRepo:        controlDeps.ServiceRepo,
		policyRepo:         controlDeps.PolicyRepo,
		servicePolicyRepo:  controlDeps.ServicePolicyRepo,
		apiRepo:            controlDeps.APIRepo,
		permissionRepo:     controlDeps.PermissionRepo,
		rolePermissionRepo: controlDeps.RolePermissionRepo,
		clientURIRepo:      controlDeps.ClientURIRepo,
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
	// Setup completion is tracked via the system tenant's status: the bootstrap
	// is "complete" once the system tenant has status != "pending" (which happens
	// once it has an admin + owner). No separate setup_states table.
	isSetupComplete := false

	if isTenantSetup {
		// Find default tenant
		defaultTenant, err := s.tenantRepo.FindSystem()
		if err == nil && defaultTenant != nil {
			isSetupComplete = defaultTenant.Status == "active"

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
			Metadata:    metadataJSON,
			Status:      "pending", // Not yet complete: bootstrap finishes (admin + owner created) before it becomes active.
			IsSystem:    true,      // This is a system tenant that cannot be deleted
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

		// The bootstrap super-admin is the highest-privilege account in the system, and
		// it was the ONE creation path that skipped the password policy entirely — no
		// blocklist, no breach check, no strength floor, and a weaker minimum length
		// than every tenant user. "password" was accepted, on an unauthenticated route.
		//
		// The tenant's own settings do not exist yet at this point in setup, so the
		// shipped default policy is the correct standard to hold it to.
		if perr := security.ValidatePasswordPolicyWithContext(
			ctx, req.Password, secpolicy.DefaultPasswordPolicy(),
		); perr != nil {
			return apperror.NewValidation(perr.Error())
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

		// The client's identity-provider link lives in client_identity_providers;
		// the clients row carries no identity_provider_id. Every identity names
		// its provider (the column is NOT NULL — an identity with no provider
		// matches no client and would lock the user out), so the bootstrap client
		// must already be connected to the built-in provider by this point.
		identityProviderID := defaultClient.DefaultConnectedIdentityProviderID()
		if identityProviderID == 0 {
			return apperror.NewValidation("default client is not connected to an identity provider")
		}

		// Create user identity
		userIdentity := &UserIdentity{
			TenantID:           defaultTenant.TenantID,
			UserID:             createdUser.UserID,
			IdentityProviderID: identityProviderID,
			Provider:           shared.ProviderMaintainerd,
			Sub:                uuid.New().String(),
			Metadata:           datatypes.JSON([]byte("{}")),
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

		// Mark the system tenant as active — admin + owner now exist.
		defaultTenant.Status = "active"
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

		// The policy is BUILT here from the request, not read from a seeded row.
		//
		// Seeding it meant every instance shipped with a standing grant that existed
		// before the service holding it did, covering permission families that had
		// nothing behind them — so the day a guard appeared in one of those families,
		// every holder passed it without anyone reviewing the widening. Constructing
		// it at registration puts the grant in the same request as the principal
		// receiving it, and validates it against permissions that actually exist.
		policy, err := s.ensureControlPolicy(txPolicyRepo, sysTenant.TenantID, req.PolicyName, req.AllowedActions)
		if err != nil {
			return err
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
				TenantID:    sysTenant.TenantID,
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

// CreateProfile creates the initial profile for the bootstrapped super-admin.
//
// OPTIONAL during first run. The normal path is that the admin signs in through
// the identity app and is asked for their name there, which is also where the
// display name is derived — so setup does NOT gate on it and the console does
// not call it. This endpoint exists for an unattended bootstrap that wants to
// seed the profile before anyone signs in, where no service principal yet
// exists to call the authenticated UserProfileService.
//
// Idempotent, so a retried bootstrap does not fail, and locked once setup
// completes.
func (s *setupService) CreateProfile(ctx context.Context, req CreateProfileRequestDTO) (*CreateProfileResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "setup.createProfile")
	defer span.End()

	if err := s.ensureSetupOpen(); err != nil {
		return nil, err
	}

	superAdmin, err := s.userRepo.FindSuperAdmin()
	if err != nil {
		return nil, err
	}
	if superAdmin == nil {
		return nil, apperror.NewValidation("admin must be created before the profile")
	}

	// Idempotent: setup steps may be retried. If the admin already has a profile,
	// return it rather than failing, so a re-run of the bootstrap unblocks.
	if existing, ferr := s.profileRepo.FindByUserID(superAdmin.UserID); ferr == nil && existing != nil {
		span.SetStatus(codes.Ok, "")
		return &CreateProfileResponseDTO{Profile: toSetupProfileResponseDTO(existing)}, nil
	}

	profile := &Profile{
		UserID:      superAdmin.UserID,
		FirstName:   req.FirstName,
		MiddleName:  req.MiddleName,
		LastName:    req.LastName,
		DisplayName: req.DisplayName,
		Gender:      req.Gender,
		ProfileURL:  req.ProfileURL,
		IsDefault:   true,
		CreatedBy:   &superAdmin.UserID,
	}
	if req.Birthdate != nil && *req.Birthdate != "" {
		if bd, perr := time.Parse("2006-01-02", *req.Birthdate); perr == nil {
			profile.Birthdate = &bd
		}
	}
	if len(req.Metadata) > 0 {
		if raw, merr := json.Marshal(req.Metadata); merr == nil {
			profile.Metadata = datatypes.JSON(raw)
		}
	}

	created, err := s.profileRepo.Create(profile)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create profile failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return &CreateProfileResponseDTO{Profile: toSetupProfileResponseDTO(created)}, nil
}

func toSetupProfileResponseDTO(p *Profile) ProfileResponseDTO {
	dto := ProfileResponseDTO{
		ProfileUUID: p.ProfileUUID.String(),
		FirstName:   p.FirstName,
		MiddleName:  p.MiddleName,
		LastName:    p.LastName,
		DisplayName: p.DisplayName,
		// The shared ProfileResponseDTO renders birthdate as "YYYY-MM-DD" so a GET
		// can be round-tripped straight into a PUT; the model stores *time.Time.
		Birthdate:  user.BirthdateString(p.Birthdate),
		Gender:     p.Gender,
		ProfileURL: p.ProfileURL,
		IsDefault:  p.IsDefault,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
	if len(p.Metadata) > 0 {
		_ = json.Unmarshal(p.Metadata, &dto.Metadata)
	}
	return dto
}

func (s *setupService) CompleteSetup(ctx context.Context) (*CompleteSetupResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "setup.complete")
	defer span.End()

	status, err := s.GetSetupStatus(ctx)
	if err != nil {
		return nil, err
	}
	if status.IsSetupComplete {
		// Idempotent: a core-side retry after a partial failure re-reads the same
		// state and gets the same answer. Nothing to "close" — bootstrap is over
		// because the system tenant exists, and ensureSetupOpen reads that directly.
		return &CompleteSetupResponseDTO{IsSetupComplete: true}, nil
	}
	// Bootstrap is complete once the system tenant exists and it has an owner.
	// The admin's PROFILE is deliberately not part of this: it is collected on
	// first sign-in through the identity app (first/last name, gender), which is
	// also where the display name is derived. Gating the lock on it meant setup
	// could never finish — the tenant stayed `pending`, and
	// AuthEndpointTenantStatusMiddleware then refused the very login that would
	// have created the profile.
	if !status.IsTenantSetup || !status.IsAdminSetup {
		return nil, apperror.NewValidation("tenant and admin setup must be completed before locking setup")
	}

	// Mark the system tenant as active — this is what records that bootstrap
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
	systemTenant.Status = "active"
	if _, err := s.tenantRepo.CreateOrUpdate(systemTenant); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "complete setup failed")
		return nil, err
	}

	// Single-use: the credential is spent the moment bootstrap finishes, so the
	// value core was handed at provision time cannot be replayed against this
	// instance afterwards. Failing here fails the whole call — reporting setup
	// complete while the credential is still open is the state this prevents, and
	// the retry re-enters through the already-complete branch above.
	span.SetStatus(codes.Ok, "")
	return &CompleteSetupResponseDTO{IsSetupComplete: true}, nil
}

// ensureSetupOpen is the single gate every setup call passes through. It closes
// on two independent conditions, and both are one-way.
//
//  1. Setup finished. An ACTIVE system tenant is the durable, replica-shared fact
//     that this instance has been bootstrapped. It survives a crash-loop, and the
//     single-system-tenant constraint settles the race when two replicas reach a
//     fresh instance together — which is why there is no separate "setup closed"
//     flag anywhere. A second copy of this boolean could only drift from it.
//
//  2. The window expired. Orchestrated setup spans many calls, so it cannot close
//     on first write the way the REST wizard does; without a deadline, an
//     orchestrator that dies mid-provision leaves tenant, client and policy
//     creation reachable to anyone holding the bootstrap credential, forever.
func (s *setupService) ensureSetupOpen() error {
	systemTenant, err := s.tenantRepo.FindSystem()
	if err != nil {
		return err
	}
	if systemTenant != nil && systemTenant.Status == "active" {
		return apperror.NewConflict("setup is complete and locked")
	}
	if err := s.ensureSetupWindowOpen(); err != nil {
		return err
	}
	return nil
}

// setupProcessStart anchors the setup window. Process start, not first request:
// an attacker who reaches the instance before the orchestrator does must not be
// able to restart the clock by being the one to open it.
var setupProcessStart = time.Now()

// setupWindowTTL is read through a variable so tests can drive expiry without
// mutating process-wide config.
var setupWindowTTL = func() time.Duration { return config.SetupWindowTTL }

func (s *setupService) ensureSetupWindowOpen() error {
	// ORCHESTRATED instances only.
	//
	// The deadline exists because machine-driven setup spans many calls with
	// nobody watching: if the orchestrator dies halfway, tenant/client/policy
	// creation stays reachable to anyone holding the bootstrap credential.
	//
	// A STANDALONE instance bootstraps through the REST wizard, which is a person
	// filling in forms. Applying the same deadline there means an operator who
	// starts the wizard and is interrupted comes back to an instance that has
	// silently locked itself, recoverable only by restarting the container — a
	// self-inflicted outage in the one flow every self-hosted install runs. The
	// wizard's own protection is different and unchanged: it closes the moment the
	// system tenant is active, and refuseWhenOrchestrated shuts it entirely on an
	// instance an orchestrator owns.
	if !bootstrapControlPlaneEnabled() {
		return nil
	}
	ttl := setupWindowTTL()
	if ttl <= 0 {
		return nil
	}
	if elapsed := time.Since(setupProcessStart); elapsed > ttl {
		return apperror.NewForbidden(fmt.Sprintf(
			"the setup window closed %s ago (SETUP_WINDOW_TTL=%s): an unfinished provision fails closed rather than staying open — restart this instance to provision it again",
			(elapsed - ttl).Truncate(time.Second), ttl))
	}
	return nil
}
