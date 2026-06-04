package setup

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/client"
	"github.com/maintainerd/auth/internal/iam"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/tenant"
	"github.com/maintainerd/auth/internal/user"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	errValidation = apperror.NewValidation("validation error")
)

func strPtr(v string) *string { return &v }

func newMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return gormDB, mock
}

type mockBaseRepo[T any] struct{}

func (m *mockBaseRepo[T]) Create(e *T) (*T, error)                            { return e, nil }
func (m *mockBaseRepo[T]) CreateOrUpdate(e *T) (*T, error)                    { return e, nil }
func (m *mockBaseRepo[T]) FindAll(preloads ...string) ([]T, error)            { return nil, nil }
func (m *mockBaseRepo[T]) FindByUUID(id any, p ...string) (*T, error)         { return nil, nil }
func (m *mockBaseRepo[T]) FindByUUIDs(ids []string, p ...string) ([]T, error) { return nil, nil }
func (m *mockBaseRepo[T]) FindByID(id any, p ...string) (*T, error)           { return nil, nil }
func (m *mockBaseRepo[T]) UpdateByUUID(id, data any) (*T, error)              { return nil, nil }
func (m *mockBaseRepo[T]) UpdateByID(id, data any) (*T, error)                { return nil, nil }
func (m *mockBaseRepo[T]) DeleteByUUID(id any) error                          { return nil }
func (m *mockBaseRepo[T]) DeleteByID(id any) error                            { return nil }
func (m *mockBaseRepo[T]) Paginate(c map[string]any, page, limit int, p ...string) (*PaginationResult[T], error) {
	return nil, nil
}

type mockSetupService struct {
	getSetupStatusFn func() (*SetupStatusResponseDTO, error)
	createTenantFn   func(req CreateTenantRequestDTO) (*CreateTenantResponseDTO, error)
	createAdminFn    func(req CreateAdminRequestDTO) (*CreateAdminResponseDTO, error)
	createProfileFn  func(req CreateProfileRequestDTO) (*CreateProfileResponseDTO, error)
	completeSetupFn  func() (*CompleteSetupResponseDTO, error)
}

func (m *mockSetupService) GetSetupStatus(_ context.Context) (*SetupStatusResponseDTO, error) {
	if m.getSetupStatusFn != nil {
		return m.getSetupStatusFn()
	}
	return &SetupStatusResponseDTO{}, nil
}
func (m *mockSetupService) CreateTenant(_ context.Context, req CreateTenantRequestDTO) (*CreateTenantResponseDTO, error) {
	if m.createTenantFn != nil {
		return m.createTenantFn(req)
	}
	return &CreateTenantResponseDTO{}, nil
}
func (m *mockSetupService) CreateAdmin(_ context.Context, req CreateAdminRequestDTO) (*CreateAdminResponseDTO, error) {
	if m.createAdminFn != nil {
		return m.createAdminFn(req)
	}
	return &CreateAdminResponseDTO{}, nil
}
func (m *mockSetupService) CreateProfile(_ context.Context, req CreateProfileRequestDTO) (*CreateProfileResponseDTO, error) {
	if m.createProfileFn != nil {
		return m.createProfileFn(req)
	}
	return &CreateProfileResponseDTO{}, nil
}
func (m *mockSetupService) CompleteSetup(_ context.Context) (*CompleteSetupResponseDTO, error) {
	if m.completeSetupFn != nil {
		return m.completeSetupFn()
	}
	return &CompleteSetupResponseDTO{}, nil
}

type mockSetupStateRepo struct {
	complete       bool
	isCompleteErr  error
	markCompleteFn func(string, time.Time) (*SetupState, error)
}

func (m *mockSetupStateRepo) WithTx(_ *gorm.DB) SetupStateRepository { return m }
func (m *mockSetupStateRepo) FindByKey(key string) (*SetupState, error) {
	if m.complete {
		now := time.Now()
		return &SetupState{Key: key, IsComplete: true, CompletedAt: &now}, nil
	}
	return nil, nil
}
func (m *mockSetupStateRepo) IsComplete(_ string) (bool, error) {
	if m.isCompleteErr != nil {
		return false, m.isCompleteErr
	}
	return m.complete, nil
}
func (m *mockSetupStateRepo) MarkComplete(key string, completedAt time.Time) (*SetupState, error) {
	if m.markCompleteFn != nil {
		return m.markCompleteFn(key, completedAt)
	}
	m.complete = true
	return &SetupState{Key: key, IsComplete: true, CompletedAt: &completedAt}, nil
}

type mockTenantRepo struct {
	mockBaseRepo[Tenant]
	findAllFn               func(...string) ([]Tenant, error)
	findByUUIDFn            func(any, ...string) (*Tenant, error)
	findByNameFn            func(string) (*Tenant, error)
	findByIdentifierFn      func(string) (*Tenant, error)
	findSystemFn            func() (*Tenant, error)
	findPaginatedFn         func(tenant.TenantRepositoryGetFilter) (*PaginationResult[Tenant], error)
	setStatusByUUIDFn       func(uuid.UUID, string) error
	setSystemStatusByUUIDFn func(uuid.UUID, bool) error
	createFn                func(*Tenant) (*Tenant, error)
	createOrUpdateFn        func(*Tenant) (*Tenant, error)
}

func (m *mockTenantRepo) WithTx(_ *gorm.DB) TenantRepository { return m }
func (m *mockTenantRepo) FindAll(p ...string) ([]Tenant, error) {
	if m.findAllFn != nil {
		return m.findAllFn(p...)
	}
	return nil, nil
}
func (m *mockTenantRepo) FindByUUID(id any, p ...string) (*Tenant, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockTenantRepo) FindByName(name string) (*Tenant, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name)
	}
	return nil, nil
}
func (m *mockTenantRepo) FindByIdentifier(identifier string) (*Tenant, error) {
	if m.findByIdentifierFn != nil {
		return m.findByIdentifierFn(identifier)
	}
	return nil, nil
}
func (m *mockTenantRepo) FindSystem() (*Tenant, error) {
	if m.findSystemFn != nil {
		return m.findSystemFn()
	}
	return nil, nil
}
func (m *mockTenantRepo) FindPaginated(f tenant.TenantRepositoryGetFilter) (*PaginationResult[Tenant], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[Tenant]{}, nil
}
func (m *mockTenantRepo) SetStatusByUUID(id uuid.UUID, status string) error {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, status)
	}
	return nil
}
func (m *mockTenantRepo) SetSystemStatusByUUID(id uuid.UUID, isSystem bool) error {
	if m.setSystemStatusByUUIDFn != nil {
		return m.setSystemStatusByUUIDFn(id, isSystem)
	}
	return nil
}
func (m *mockTenantRepo) Create(e *Tenant) (*Tenant, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockTenantRepo) CreateOrUpdate(e *Tenant) (*Tenant, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}

func (m *mockTenantRepo) DeleteCascade(_ context.Context, _ *gorm.DB, _ int64, _ []any) error {
	return nil
}

type mockUserRepo struct {
	mockBaseRepo[User]
	findByUUIDFn             func(any, ...string) (*User, error)
	findByUsernameFn         func(string) (*User, error)
	findByEmailFn            func(string) (*User, error)
	findByEmailAndTenantIDFn func(string, int64) (*User, error)
	findByPhoneFn            func(string) (*User, error)
	findSuperAdminFn         func() (*User, error)
	findRolesFn              func(int64) ([]user.Role, error)
	findBySubAndClientIDFn   func(string, string) (*User, error)
	findPaginatedFn          func(user.UserRepositoryGetFilter) (*PaginationResult[User], error)
	setEmailVerifiedFn       func(uuid.UUID, bool) error
	setStatusFn              func(uuid.UUID, string) error
	setForcePasswordChangeFn func(uuid.UUID, bool) error
	setPendingEmailFn        func(uuid.UUID, string, string, time.Time) error
	clearEmailChangeFn       func(uuid.UUID) error
	updateEmailFn            func(uuid.UUID, string) error
	updateUsernameFn         func(uuid.UUID, string) error
	findByPendingEmailFn     func(string) (*User, error)
	createFn                 func(*User) (*User, error)
	updateByUUIDFn           func(any, any) (*User, error)
}

func (m *mockUserRepo) WithTx(_ *gorm.DB) UserRepository { return m }
func (m *mockUserRepo) Create(e *User) (*User, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockUserRepo) FindByUUID(id any, p ...string) (*User, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockUserRepo) UpdateByUUID(id, data any) (*User, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByUsername(username string) (*User, error) {
	if m.findByUsernameFn != nil {
		return m.findByUsernameFn(username)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByEmail(email string) (*User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(email)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByEmailAndTenantID(email string, tenantID int64) (*User, error) {
	if m.findByEmailAndTenantIDFn != nil {
		return m.findByEmailAndTenantIDFn(email, tenantID)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByPhone(phone string) (*User, error) {
	if m.findByPhoneFn != nil {
		return m.findByPhoneFn(phone)
	}
	return nil, nil
}
func (m *mockUserRepo) FindSuperAdmin() (*User, error) {
	if m.findSuperAdminFn != nil {
		return m.findSuperAdminFn()
	}
	return nil, nil
}
func (m *mockUserRepo) FindRoles(userID int64) ([]user.Role, error) {
	if m.findRolesFn != nil {
		return m.findRolesFn(userID)
	}
	return nil, nil
}
func (m *mockUserRepo) FindRolesPaginated(_ user.GetUserRolesFilter) (*PaginationResult[user.Role], error) {
	return &PaginationResult[user.Role]{}, nil
}
func (m *mockUserRepo) FindBySubAndClientID(sub, clientID string) (*User, error) {
	if m.findBySubAndClientIDFn != nil {
		return m.findBySubAndClientIDFn(sub, clientID)
	}
	return nil, nil
}
func (m *mockUserRepo) FindPaginated(f user.UserRepositoryGetFilter) (*PaginationResult[User], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[User]{}, nil
}
func (m *mockUserRepo) SetEmailVerified(userUUID uuid.UUID, verified bool) error {
	if m.setEmailVerifiedFn != nil {
		return m.setEmailVerifiedFn(userUUID, verified)
	}
	return nil
}
func (m *mockUserRepo) SetStatus(userUUID uuid.UUID, status string) error {
	if m.setStatusFn != nil {
		return m.setStatusFn(userUUID, status)
	}
	return nil
}
func (m *mockUserRepo) SetForcePasswordChange(userUUID uuid.UUID, force bool) error {
	if m.setForcePasswordChangeFn != nil {
		return m.setForcePasswordChangeFn(userUUID, force)
	}
	return nil
}
func (m *mockUserRepo) SetPendingEmail(userUUID uuid.UUID, pendingEmail, otp string, expiresAt time.Time) error {
	if m.setPendingEmailFn != nil {
		return m.setPendingEmailFn(userUUID, pendingEmail, otp, expiresAt)
	}
	return nil
}
func (m *mockUserRepo) ClearEmailChange(userUUID uuid.UUID) error {
	if m.clearEmailChangeFn != nil {
		return m.clearEmailChangeFn(userUUID)
	}
	return nil
}
func (m *mockUserRepo) UpdateEmail(userUUID uuid.UUID, email string) error {
	if m.updateEmailFn != nil {
		return m.updateEmailFn(userUUID, email)
	}
	return nil
}
func (m *mockUserRepo) UpdateUsername(userUUID uuid.UUID, username string) error {
	if m.updateUsernameFn != nil {
		return m.updateUsernameFn(userUUID, username)
	}
	return nil
}
func (m *mockUserRepo) FindByPendingEmail(email string) (*User, error) {
	if m.findByPendingEmailFn != nil {
		return m.findByPendingEmailFn(email)
	}
	return nil, nil
}

type mockProfileRepo struct {
	mockBaseRepo[Profile]
	findByUserIDFn         func(int64) (*Profile, error)
	findDefaultByUserIDFn  func(int64) (*Profile, error)
	findAllByUserIDFn      func(user.ProfileRepositoryGetFilter) (*PaginationResult[Profile], error)
	updateByUserIDFn       func(int64, *Profile) error
	deleteByUserIDFn       func(int64) error
	unsetDefaultProfilesFn func(int64) error
	createFn               func(*Profile) (*Profile, error)
}

func (m *mockProfileRepo) WithTx(_ *gorm.DB) ProfileRepository { return m }
func (m *mockProfileRepo) Create(e *Profile) (*Profile, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockProfileRepo) FindByUserID(userID int64) (*Profile, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(userID)
	}
	return nil, nil
}
func (m *mockProfileRepo) FindDefaultByUserID(userID int64) (*Profile, error) {
	if m.findDefaultByUserIDFn != nil {
		return m.findDefaultByUserIDFn(userID)
	}
	return nil, nil
}
func (m *mockProfileRepo) FindAllByUserID(f user.ProfileRepositoryGetFilter) (*PaginationResult[Profile], error) {
	if m.findAllByUserIDFn != nil {
		return m.findAllByUserIDFn(f)
	}
	return &PaginationResult[Profile]{}, nil
}
func (m *mockProfileRepo) UpdateByUserID(userID int64, updatedProfile *Profile) error {
	if m.updateByUserIDFn != nil {
		return m.updateByUserIDFn(userID, updatedProfile)
	}
	return nil
}
func (m *mockProfileRepo) DeleteByUserID(userID int64) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(userID)
	}
	return nil
}
func (m *mockProfileRepo) UnsetDefaultProfiles(userID int64) error {
	if m.unsetDefaultProfilesFn != nil {
		return m.unsetDefaultProfilesFn(userID)
	}
	return nil
}

type mockClientRepo struct {
	mockBaseRepo[Client]
	findByUUIDFn                        func(any, ...string) (*Client, error)
	findByUUIDAndTenantIDFn             func(uuid.UUID, int64) (*Client, error)
	findByNameAndIdentityProviderFn     func(string, int64, int64) (*Client, error)
	findByNameAndTenantIDFn             func(string, int64) (*Client, error)
	findByClientIDFn                    func(string, int64) (*Client, error)
	findAllByTenantIDFn                 func(int64) ([]Client, error)
	findSystemFn                        func() (*Client, error)
	findDefaultByTenantIDFn             func(int64) (*Client, error)
	findPaginatedFn                     func(client.ClientRepositoryGetFilter) (*PaginationResult[Client], error)
	setStatusByUUIDFn                   func(uuid.UUID, int64, string) error
	findByClientIDAndIdentityProviderFn func(string, string) (*Client, error)
	deleteByUUIDAndTenantIDFn           func(uuid.UUID, int64) error
}

func (m *mockClientRepo) WithTx(_ *gorm.DB) ClientRepository { return m }
func (m *mockClientRepo) FindByUUID(id any, p ...string) (*Client, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64) (*Client, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tenantID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByNameAndIdentityProvider(name string, identityProviderID int64, tenantID int64) (*Client, error) {
	if m.findByNameAndIdentityProviderFn != nil {
		return m.findByNameAndIdentityProviderFn(name, identityProviderID, tenantID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByNameAndTenantID(name string, tenantID int64) (*Client, error) {
	if m.findByNameAndTenantIDFn != nil {
		return m.findByNameAndTenantIDFn(name, tenantID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByClientID(clientID string, tenantID int64) (*Client, error) {
	if m.findByClientIDFn != nil {
		return m.findByClientIDFn(clientID, tenantID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindAllByTenantID(tenantID int64) ([]Client, error) {
	if m.findAllByTenantIDFn != nil {
		return m.findAllByTenantIDFn(tenantID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindSystem() (*Client, error) {
	if m.findSystemFn != nil {
		return m.findSystemFn()
	}
	return nil, nil
}
func (m *mockClientRepo) FindDefaultByTenantID(tenantID int64) (*Client, error) {
	if m.findDefaultByTenantIDFn != nil {
		return m.findDefaultByTenantIDFn(tenantID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindPaginated(f client.ClientRepositoryGetFilter) (*PaginationResult[Client], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[Client]{}, nil
}
func (m *mockClientRepo) SetStatusByUUID(id uuid.UUID, tenantID int64, status string) error {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tenantID, status)
	}
	return nil
}
func (m *mockClientRepo) FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*Client, error) {
	if m.findByClientIDAndIdentityProviderFn != nil {
		return m.findByClientIDAndIdentityProviderFn(clientID, identityProviderIdentifier)
	}
	return nil, nil
}
func (m *mockClientRepo) DeleteByUUIDAndTenantID(id uuid.UUID, tenantID int64) error {
	if m.deleteByUUIDAndTenantIDFn != nil {
		return m.deleteByUUIDAndTenantIDFn(id, tenantID)
	}
	return nil
}

type mockRoleRepo struct {
	mockBaseRepo[Role]
	findByNameAndTenantIDFn      func(string, int64) (*Role, error)
	findAllByTenantIDFn          func(int64) ([]Role, error)
	findPaginatedFn              func(RoleRepositoryGetFilter) (*PaginationResult[Role], error)
	setStatusByUUIDFn            func(uuid.UUID, string) error
	setDefaultStatusByUUIDFn     func(uuid.UUID, bool) error
	setSystemStatusByUUIDFn      func(uuid.UUID, bool) error
	findRegisteredRoleForSetupFn func(int64) (*Role, error)
	findSuperAdminRoleForSetupFn func(int64) (*Role, error)
}

func (m *mockRoleRepo) WithTx(_ *gorm.DB) RoleRepository { return m }
func (m *mockRoleRepo) FindByNameAndTenantID(name string, tenantID int64) (*Role, error) {
	if m.findByNameAndTenantIDFn != nil {
		return m.findByNameAndTenantIDFn(name, tenantID)
	}
	return nil, nil
}
func (m *mockRoleRepo) FindAllByTenantID(tenantID int64) ([]Role, error) {
	if m.findAllByTenantIDFn != nil {
		return m.findAllByTenantIDFn(tenantID)
	}
	return nil, nil
}
func (m *mockRoleRepo) FindPaginated(f RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[Role]{}, nil
}
func (m *mockRoleRepo) GetPermissionsByRoleUUID(iam.RoleRepositoryGetPermissionsFilter) (*PaginationResult[iam.Permission], error) {
	return &PaginationResult[iam.Permission]{}, nil
}
func (m *mockRoleRepo) SetStatusByUUID(id uuid.UUID, status string) error {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, status)
	}
	return nil
}
func (m *mockRoleRepo) SetDefaultStatusByUUID(id uuid.UUID, isDefault bool) error {
	if m.setDefaultStatusByUUIDFn != nil {
		return m.setDefaultStatusByUUIDFn(id, isDefault)
	}
	return nil
}
func (m *mockRoleRepo) SetSystemStatusByUUID(id uuid.UUID, isSystem bool) error {
	if m.setSystemStatusByUUIDFn != nil {
		return m.setSystemStatusByUUIDFn(id, isSystem)
	}
	return nil
}
func (m *mockRoleRepo) FindRegisteredRoleForSetup(tenantID int64) (*Role, error) {
	if m.findRegisteredRoleForSetupFn != nil {
		return m.findRegisteredRoleForSetupFn(tenantID)
	}
	return nil, nil
}
func (m *mockRoleRepo) FindSuperAdminRoleForSetup(tenantID int64) (*Role, error) {
	if m.findSuperAdminRoleForSetupFn != nil {
		return m.findSuperAdminRoleForSetupFn(tenantID)
	}
	return nil, nil
}

type mockUserRoleRepo struct {
	mockBaseRepo[UserRole]
	findByUserIDFn             func(int64) ([]UserRole, error)
	findByUserIDAndRoleIDFn    func(int64, int64) (*UserRole, error)
	findDefaultRolesByUserIDFn func(int64) ([]UserRole, error)
	deleteByUserIDFn           func(int64) error
	deleteByUserIDAndRoleIDFn  func(int64, int64) error
	createFn                   func(*UserRole) (*UserRole, error)
}

func (m *mockUserRoleRepo) WithTx(_ *gorm.DB) UserRoleRepository { return m }
func (m *mockUserRoleRepo) Create(e *UserRole) (*UserRole, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockUserRoleRepo) FindByUserID(userID int64) ([]UserRole, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(userID)
	}
	return nil, nil
}
func (m *mockUserRoleRepo) FindByUserIDAndRoleID(userID, roleID int64) (*UserRole, error) {
	if m.findByUserIDAndRoleIDFn != nil {
		return m.findByUserIDAndRoleIDFn(userID, roleID)
	}
	return nil, nil
}
func (m *mockUserRoleRepo) FindDefaultRolesByUserID(userID int64) ([]UserRole, error) {
	if m.findDefaultRolesByUserIDFn != nil {
		return m.findDefaultRolesByUserIDFn(userID)
	}
	return nil, nil
}
func (m *mockUserRoleRepo) DeleteByUserID(userID int64) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(userID)
	}
	return nil
}
func (m *mockUserRoleRepo) DeleteByUserIDAndRoleID(userID, roleID int64) error {
	if m.deleteByUserIDAndRoleIDFn != nil {
		return m.deleteByUserIDAndRoleIDFn(userID, roleID)
	}
	return nil
}

type mockUserIdentityRepo struct {
	mockBaseRepo[UserIdentity]
	findByUserIDFn             func(int64) ([]UserIdentity, error)
	findByUserIDAndClientIDFn  func(int64, int64) (*UserIdentity, error)
	findByProviderAndSubFn     func(string, string) (*UserIdentity, error)
	findByUserIDAndProviderFn  func(int64, string) (*UserIdentity, error)
	findByIdentityProviderIDFn func(int64) ([]UserIdentity, error)
	deleteByUserIDFn           func(int64) error
	createFn                   func(*UserIdentity) (*UserIdentity, error)
}

func (m *mockUserIdentityRepo) WithTx(_ *gorm.DB) UserIdentityRepository { return m }
func (m *mockUserIdentityRepo) Create(e *UserIdentity) (*UserIdentity, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockUserIdentityRepo) FindByUserID(userID int64) ([]UserIdentity, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(userID)
	}
	return nil, nil
}
func (m *mockUserIdentityRepo) FindUserIdentitiesPaginated(_ user.GetUserIdentitiesFilter) (*PaginationResult[user.UserIdentity], error) {
	return &PaginationResult[user.UserIdentity]{}, nil
}
func (m *mockUserIdentityRepo) FindByUserIDAndClientID(userID, clientID int64) (*UserIdentity, error) {
	if m.findByUserIDAndClientIDFn != nil {
		return m.findByUserIDAndClientIDFn(userID, clientID)
	}
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByProviderAndSub(provider, sub string) (*UserIdentity, error) {
	if m.findByProviderAndSubFn != nil {
		return m.findByProviderAndSubFn(provider, sub)
	}
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByUserIDAndProvider(userID int64, provider string) (*UserIdentity, error) {
	if m.findByUserIDAndProviderFn != nil {
		return m.findByUserIDAndProviderFn(userID, provider)
	}
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByIdentityProviderID(idpID int64) ([]UserIdentity, error) {
	if m.findByIdentityProviderIDFn != nil {
		return m.findByIdentityProviderIDFn(idpID)
	}
	return nil, nil
}
func (m *mockUserIdentityRepo) DeleteByUserID(userID int64) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(userID)
	}
	return nil
}

type mockTenantMemberRepo struct {
	mockBaseRepo[TenantMember]
	findByTenantMemberUUIDFn func(uuid.UUID) (*TenantMember, error)
	findByTenantAndUserFn    func(int64, int64) (*TenantMember, error)
	findByTenantFn           func(tenant.TenantMemberRepositoryListFilter) (*PaginationResult[TenantMember], error)
	findAllByUserFn          func(int64) ([]TenantMember, error)
	createFn                 func(*TenantMember) (*TenantMember, error)
}

func (m *mockTenantMemberRepo) WithTx(_ *gorm.DB) TenantMemberRepository { return m }
func (m *mockTenantMemberRepo) Create(e *TenantMember) (*TenantMember, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockTenantMemberRepo) FindByTenantMemberUUID(id uuid.UUID) (*TenantMember, error) {
	if m.findByTenantMemberUUIDFn != nil {
		return m.findByTenantMemberUUIDFn(id)
	}
	return nil, nil
}
func (m *mockTenantMemberRepo) FindByTenantAndUser(tenantID int64, userID int64) (*TenantMember, error) {
	if m.findByTenantAndUserFn != nil {
		return m.findByTenantAndUserFn(tenantID, userID)
	}
	return nil, nil
}
func (m *mockTenantMemberRepo) FindByTenant(filter tenant.TenantMemberRepositoryListFilter) (*PaginationResult[TenantMember], error) {
	if m.findByTenantFn != nil {
		return m.findByTenantFn(filter)
	}
	return &PaginationResult[TenantMember]{}, nil
}
func (m *mockTenantMemberRepo) FindAllByUser(userID int64) ([]TenantMember, error) {
	if m.findAllByUserFn != nil {
		return m.findAllByUserFn(userID)
	}
	return nil, nil
}
