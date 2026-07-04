package user

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/notifier"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	errNotFound   = apperror.NewNotFoundWithReason("not found")
	errValidation = apperror.NewValidation("validation error")
	errForbidden  = apperror.NewForbidden("access denied")
)

const tenantID int64 = 1

var (
	testTenantUUID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testUserUUID     = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testResourceUUID = uuid.MustParse("00000000-0000-0000-0000-000000000099")
)

func withTenant(r *http.Request) *http.Request {
	return middleware.WithAuthContext(r, &authctx.AuthContext{
		Tenant: &authctx.AuthTenant{TenantID: tenantID, TenantUUID: testTenantUUID},
	})
}

func withTenantAndUser(r *http.Request) *http.Request {
	return middleware.WithAuthContext(r, &authctx.AuthContext{
		Tenant: &authctx.AuthTenant{TenantID: tenantID, TenantUUID: testTenantUUID},
		User:   &authctx.AuthUser{UserUUID: testUserUUID},
	})
}

func withUserNoTenant(r *http.Request) *http.Request {
	return middleware.WithAuthContext(r, &authctx.AuthContext{
		User: &authctx.AuthUser{UserID: 42, UserUUID: testUserUUID},
	})
}

func withTenantAndActor(r *http.Request, actorUserID int64) *http.Request {
	return middleware.WithAuthContext(r, &authctx.AuthContext{
		Tenant: &authctx.AuthTenant{TenantID: tenantID, TenantUUID: testTenantUUID},
		User:   &authctx.AuthUser{UserID: actorUserID, UserUUID: testUserUUID},
	})
}

func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		rctx = chi.NewRouteContext()
	}
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func badJSONReq(t *testing.T, method, target string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(method, target, strings.NewReader("{bad json"))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func jsonReq(t *testing.T, method, url string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	r := httptest.NewRequest(method, url, &buf)
	r.Header.Set("Content-Type", "application/json")
	return r
}

func strPtr(v string) *string   { return &v }
func int64Ptr(v int64) *int64   { return &v }

func validPagination() PaginationRequestDTO {
	return PaginationRequestDTO{Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc}
}

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

// mockUserOTPRepo mocks notifier.UserOTPRepository for account phone-verification tests.
type mockUserOTPRepo struct {
	createFn          func(*notifier.UserOTP) (*notifier.UserOTP, error)
	findValidFn       func(channel, recipient string) (*notifier.UserOTP, error)
	findValidByUserFn func(userID int64, channel string) (*notifier.UserOTP, error)
	recordFailFn      func(id int64, maxAttempts int) error
	markUsedFn        func(id int64) error
}

func (m *mockUserOTPRepo) WithTx(*gorm.DB) notifier.UserOTPRepository { return m }
func (m *mockUserOTPRepo) Create(e *notifier.UserOTP) (*notifier.UserOTP, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockUserOTPRepo) CreateOrUpdate(e *notifier.UserOTP) (*notifier.UserOTP, error) {
	return e, nil
}
func (m *mockUserOTPRepo) FindValid(channel, recipient string) (*notifier.UserOTP, error) {
	if m.findValidFn != nil {
		return m.findValidFn(channel, recipient)
	}
	return nil, nil
}
func (m *mockUserOTPRepo) FindValidByUserAndChannel(userID int64, channel string) (*notifier.UserOTP, error) {
	if m.findValidByUserFn != nil {
		return m.findValidByUserFn(userID, channel)
	}
	return nil, nil
}
func (m *mockUserOTPRepo) RecordFailure(id int64, maxAttempts int) error {
	if m.recordFailFn != nil {
		return m.recordFailFn(id, maxAttempts)
	}
	return nil
}
func (m *mockUserOTPRepo) MarkUsed(id int64) error {
	if m.markUsedFn != nil {
		return m.markUsedFn(id)
	}
	return nil
}
func (m *mockUserOTPRepo) DeleteExpired(time.Time) (int64, error) { return 0, nil }

type mockUserRepo struct {
	mockBaseRepo[User]
	findByUUIDFn             func(any, ...string) (*User, error)
	findByIDFn               func(any, ...string) (*User, error)
	findByUsernameFn         func(string) (*User, error)
	findByEmailFn            func(string) (*User, error)
	findByEmailAndTenantIDFn func(string, int64) (*User, error)
	findByPhoneFn            func(string) (*User, error)
	findSuperAdminFn         func() (*User, error)
	findRolesFn              func(int64) ([]Role, error)
	findRolesPaginatedFn     func(GetUserRolesFilter) (*PaginationResult[Role], error)
	findBySubAndClientIDFn   func(string, string) (*User, error)
	findPaginatedFn          func(UserRepositoryGetFilter) (*PaginationResult[User], error)
	setEmailVerifiedFn       func(uuid.UUID, bool) error
	setStatusFn              func(uuid.UUID, string) error
	setForcePasswordChangeFn func(uuid.UUID, bool) error
	updateEmailFn            func(uuid.UUID, string) error
	updateUsernameFn         func(uuid.UUID, string) error
	createFn                 func(*User) (*User, error)
	updateByUUIDFn           func(any, any) (*User, error)
	updateByIDFn             func(any, any) (*User, error)
	deleteByUUIDFn           func(any) error
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
func (m *mockUserRepo) FindByID(id any, p ...string) (*User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockUserRepo) UpdateByUUID(id, data any) (*User, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockUserRepo) UpdateByID(id, data any) (*User, error) {
	if m.updateByIDFn != nil {
		return m.updateByIDFn(id, data)
	}
	return nil, nil
}
func (m *mockUserRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
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
	if m.findByEmailFn != nil {
		return m.findByEmailFn(email)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByUsernameAndTenantID(username string, tenantID int64) (*User, error) {
	if m.findByUsernameFn != nil {
		return m.findByUsernameFn(username)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByPhoneAndTenantID(phone string, tenantID int64) (*User, error) {
	if m.findByPhoneFn != nil {
		return m.findByPhoneFn(phone)
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
func (m *mockUserRepo) FindRoles(userID int64) ([]Role, error) {
	if m.findRolesFn != nil {
		return m.findRolesFn(userID)
	}
	return nil, nil
}
func (m *mockUserRepo) FindRolesPaginated(f GetUserRolesFilter) (*PaginationResult[Role], error) {
	if m.findRolesPaginatedFn != nil {
		return m.findRolesPaginatedFn(f)
	}
	return &PaginationResult[Role]{}, nil
}
func (m *mockUserRepo) FindBySubAndClientID(sub, clientID string) (*User, error) {
	if m.findBySubAndClientIDFn != nil {
		return m.findBySubAndClientIDFn(sub, clientID)
	}
	return nil, nil
}
func (m *mockUserRepo) FindPaginated(f UserRepositoryGetFilter) (*PaginationResult[User], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[User]{}, nil
}
func (m *mockUserRepo) SetEmailVerified(id uuid.UUID, verified bool) error {
	if m.setEmailVerifiedFn != nil {
		return m.setEmailVerifiedFn(id, verified)
	}
	return nil
}
func (m *mockUserRepo) SetStatus(id uuid.UUID, status string) error {
	if m.setStatusFn != nil {
		return m.setStatusFn(id, status)
	}
	return nil
}
func (m *mockUserRepo) SetForcePasswordChange(id uuid.UUID, force bool) error {
	if m.setForcePasswordChangeFn != nil {
		return m.setForcePasswordChangeFn(id, force)
	}
	return nil
}
func (m *mockUserRepo) UpdateEmail(id uuid.UUID, email string) error {
	if m.updateEmailFn != nil {
		return m.updateEmailFn(id, email)
	}
	return nil
}
func (m *mockUserRepo) UpdateUsername(id uuid.UUID, username string) error {
	if m.updateUsernameFn != nil {
		return m.updateUsernameFn(id, username)
	}
	return nil
}

type mockProfileRepo struct {
	mockBaseRepo[Profile]
	findByUUIDFn           func(any, ...string) (*Profile, error)
	findByUserIDFn         func(int64) (*Profile, error)
	findDefaultByUserIDFn  func(int64) (*Profile, error)
	findAllByUserIDFn      func(ProfileRepositoryGetFilter) (*PaginationResult[Profile], error)
	updateByUserIDFn       func(int64, *Profile) error
	deleteByUserIDFn       func(int64) error
	unsetDefaultProfilesFn func(int64) error
	unsetDefaultFn         func(int64) error
	createFn               func(*Profile) (*Profile, error)
	createOrUpdateFn       func(*Profile) (*Profile, error)
	updateByUUIDFn         func(any, any) (*Profile, error)
	deleteByUUIDFn         func(any) error
}

func (m *mockProfileRepo) WithTx(_ *gorm.DB) ProfileRepository { return m }
func (m *mockProfileRepo) Create(e *Profile) (*Profile, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockProfileRepo) CreateOrUpdate(e *Profile) (*Profile, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockProfileRepo) FindByUUID(id any, p ...string) (*Profile, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockProfileRepo) UpdateByUUID(id, data any) (*Profile, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockProfileRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
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
func (m *mockProfileRepo) FindAllByUserID(f ProfileRepositoryGetFilter) (*PaginationResult[Profile], error) {
	if m.findAllByUserIDFn != nil {
		return m.findAllByUserIDFn(f)
	}
	return &PaginationResult[Profile]{}, nil
}
func (m *mockProfileRepo) UpdateByUserID(userID int64, p *Profile) error {
	if m.updateByUserIDFn != nil {
		return m.updateByUserIDFn(userID, p)
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
	if m.unsetDefaultFn != nil {
		return m.unsetDefaultFn(userID)
	}
	return nil
}

type mockUserSettingRepo struct {
	mockBaseRepo[UserSetting]
	findByUUIDFn     func(any, ...string) (*UserSetting, error)
	findByUserIDFn   func(int64) (*UserSetting, error)
	updateByUserIDFn func(int64, *UserSetting) error
	deleteByUserIDFn func(int64) error
	createFn         func(*UserSetting) (*UserSetting, error)
	updateByUUIDFn   func(any, any) (*UserSetting, error)
	deleteByUUIDFn   func(any) error
}

func (m *mockUserSettingRepo) WithTx(_ *gorm.DB) UserSettingRepository { return m }
func (m *mockUserSettingRepo) Create(e *UserSetting) (*UserSetting, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockUserSettingRepo) FindByUUID(id any, p ...string) (*UserSetting, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockUserSettingRepo) UpdateByUUID(id, data any) (*UserSetting, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockUserSettingRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockUserSettingRepo) FindByUserID(userID int64) (*UserSetting, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(userID)
	}
	return nil, nil
}
func (m *mockUserSettingRepo) UpdateByUserID(userID int64, us *UserSetting) error {
	if m.updateByUserIDFn != nil {
		return m.updateByUserIDFn(userID, us)
	}
	return nil
}
func (m *mockUserSettingRepo) DeleteByUserID(userID int64) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(userID)
	}
	return nil
}

type mockTenantRepo struct {
	mockBaseRepo[Tenant]
	findByUUIDFn func(any, ...string) (*Tenant, error)
}

func (m *mockTenantRepo) WithTx(_ *gorm.DB) TenantRepository { return m }
func (m *mockTenantRepo) FindByUUID(id any, p ...string) (*Tenant, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}

type mockRoleRepo struct {
	mockBaseRepo[Role]
	findByUUIDFn            func(any, ...string) (*Role, error)
	findByNameAndTenantIDFn func(string, int64) (*Role, error)
	findPaginatedFn         func(RoleRepositoryGetFilter) (*PaginationResult[Role], error)
}

func (m *mockRoleRepo) WithTx(_ *gorm.DB) RoleRepository { return m }
func (m *mockRoleRepo) FindByUUID(id any, p ...string) (*Role, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockRoleRepo) FindByUUIDs(uuids []string, p ...string) ([]Role, error) {
	results := make([]Role, 0, len(uuids))
	for _, id := range uuids {
		r, err := m.FindByUUID(id, p...)
		if err != nil {
			return nil, err
		}
		if r != nil {
			results = append(results, *r)
		}
	}
	return results, nil
}
func (m *mockRoleRepo) FindByNameAndTenantID(name string, tenantID int64) (*Role, error) {
	if m.findByNameAndTenantIDFn != nil {
		return m.findByNameAndTenantIDFn(name, tenantID)
	}
	return nil, nil
}
func (m *mockRoleRepo) FindPaginated(f RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[Role]{}, nil
}

type mockClientRepo struct {
	mockBaseRepo[Client]
	findByUUIDAndTenantIDFn             func(uuid.UUID, int64) (*Client, error)
	findByIDFn                          func(any, ...string) (*Client, error)
	findDefaultByTenantIDFn             func(int64) (*Client, error)
	findByClientIDAndIdentityProviderFn func(string, string) (*Client, error)
}

func (m *mockClientRepo) WithTx(_ *gorm.DB) ClientRepository { return m }
func (m *mockClientRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64) (*Client, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tenantID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByID(id any, p ...string) (*Client, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockClientRepo) FindDefaultByTenantID(tenantID int64) (*Client, error) {
	if m.findDefaultByTenantIDFn != nil {
		return m.findDefaultByTenantIDFn(tenantID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByClientIDAndIdentityProvider(clientID, idpIdentifier string) (*Client, error) {
	if m.findByClientIDAndIdentityProviderFn != nil {
		return m.findByClientIDAndIdentityProviderFn(clientID, idpIdentifier)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByIdentifier(identifier string) (*Client, error) { return nil, nil }
func (m *mockClientRepo) FindSystemByTenantIdentifier(tenantIdentifier string) (*Client, error) {
	return nil, nil
}
func (m *mockClientRepo) FindSystemByTenantIdentifierAndName(tenantIdentifier, name string) (*Client, error) {
	return m.FindSystemByTenantIdentifier(tenantIdentifier)
}

func (m *mockClientRepo) FindByIDs(ids []int64) ([]Client, error) {
	results := make([]Client, 0, len(ids))
	for _, id := range ids {
		c, err := m.FindByID(id)
		if err != nil {
			return nil, err
		}
		if c != nil {
			results = append(results, *c)
		}
	}
	return results, nil
}

type mockIdentityProviderRepo struct {
	mockBaseRepo[IdentityProvider]
	findByIdentifierFn func(string) (*IdentityProvider, error)
}

func (m *mockIdentityProviderRepo) WithTx(_ *gorm.DB) IdentityProviderRepository { return m }
func (m *mockIdentityProviderRepo) FindByIdentifier(identifier string) (*IdentityProvider, error) {
	if m.findByIdentifierFn != nil {
		return m.findByIdentifierFn(identifier)
	}
	return nil, nil
}

type mockUserIdentityRepo struct {
	mockBaseRepo[UserIdentity]
	findByUserIDFn                func(int64) ([]UserIdentity, error)
	findUserIdentitiesPaginatedFn func(GetUserIdentitiesFilter) (*PaginationResult[UserIdentity], error)
	findByUserIDAndClientIDFn     func(int64, int64) (*UserIdentity, error)
	findByUserIDAndProviderFn     func(int64, string) (*UserIdentity, error)
	findByIdentityProviderIDFn    func(int64) ([]UserIdentity, error)
	deleteByUserIDFn              func(int64) error
	createFn                      func(*UserIdentity) (*UserIdentity, error)
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
func (m *mockUserIdentityRepo) FindUserIdentitiesPaginated(f GetUserIdentitiesFilter) (*PaginationResult[UserIdentity], error) {
	if m.findUserIdentitiesPaginatedFn != nil {
		return m.findUserIdentitiesPaginatedFn(f)
	}
	return &PaginationResult[UserIdentity]{}, nil
}
func (m *mockUserIdentityRepo) FindByUserIDAndClientID(userID, clientID int64) (*UserIdentity, error) {
	if m.findByUserIDAndClientIDFn != nil {
		return m.findByUserIDAndClientIDFn(userID, clientID)
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

type mockUserService struct {
	getFn                  func(UserServiceGetFilter) (*UserServiceGetResult, error)
	getByUUIDFn            func(uuid.UUID, int64) (*UserServiceDataResult, error)
	createFn               func(string, *string, *string, string, string, datatypes.JSON, string, uuid.UUID) (*UserServiceDataResult, error)
	updateFn               func(uuid.UUID, int64, string, *string, *string, string, datatypes.JSON, uuid.UUID) (*UserServiceDataResult, error)
	setStatusFn            func(uuid.UUID, int64, string, uuid.UUID) (*UserServiceDataResult, error)
	verifyEmailFn          func(uuid.UUID, int64) (*UserServiceDataResult, error)
	verifyPhoneFn          func(uuid.UUID, int64) (*UserServiceDataResult, error)
	completeAccountFn      func(uuid.UUID, int64) (*UserServiceDataResult, error)
	deleteByUUIDFn         func(uuid.UUID, int64, uuid.UUID) (*UserServiceDataResult, error)
	assignUserRolesFn      func(uuid.UUID, []uuid.UUID, int64) (*UserServiceDataResult, error)
	removeUserRoleFn       func(uuid.UUID, uuid.UUID, int64) (*UserServiceDataResult, error)
	getUserRolesFn         func(uuid.UUID, int64, GetUserRolesFilter) ([]RoleServiceDataResult, int64, error)
	getUserIdentitiesFn    func(uuid.UUID, int64, GetUserIdentitiesFilter) ([]UserIdentityServiceDataResult, int64, error)
	getUserIdentsFn        func(uuid.UUID, int64, GetUserIdentitiesFilter) ([]UserIdentityServiceDataResult, int64, error)
	findBySubAndClientIDFn func(string, string) (*User, error)
	findByUserIDFn         func(int64) (*User, error)
	forcePasswordChangeFn  func(uuid.UUID, bool) error
	getUserMFAFn           func(uuid.UUID, int64) (*UserMFAResponseDTO, error)
	ensureUserInTenantFn   func(uuid.UUID, int64) (int64, error)
	grantRoleByNameFn      func(uuid.UUID, int64, string) error
}

func (m *mockUserService) Get(_ context.Context, f UserServiceGetFilter) (*UserServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &UserServiceGetResult{}, nil
}
func (m *mockUserService) GetByUUID(_ context.Context, id uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tenantID)
	}
	return &UserServiceDataResult{}, nil
}
func (m *mockUserService) Create(_ context.Context, username string, email *string, phone *string, password string, status string, metadata datatypes.JSON, tenantUUID string, creatorUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(username, email, phone, password, status, metadata, tenantUUID, creatorUserUUID)
	}
	return &UserServiceDataResult{}, nil
}
func (m *mockUserService) Update(_ context.Context, userUUID uuid.UUID, tenantID int64, username string, email *string, phone *string, status string, metadata datatypes.JSON, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(userUUID, tenantID, username, email, phone, status, metadata, updaterUserUUID)
	}
	return &UserServiceDataResult{}, nil
}
func (m *mockUserService) SetStatus(_ context.Context, userUUID uuid.UUID, tenantID int64, status string, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	if m.setStatusFn != nil {
		return m.setStatusFn(userUUID, tenantID, status, updaterUserUUID)
	}
	return &UserServiceDataResult{}, nil
}
func (m *mockUserService) VerifyEmail(_ context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	if m.verifyEmailFn != nil {
		return m.verifyEmailFn(userUUID, tenantID)
	}
	return &UserServiceDataResult{}, nil
}
func (m *mockUserService) VerifyPhone(_ context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	if m.verifyPhoneFn != nil {
		return m.verifyPhoneFn(userUUID, tenantID)
	}
	return &UserServiceDataResult{}, nil
}
func (m *mockUserService) CompleteAccount(_ context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	if m.completeAccountFn != nil {
		return m.completeAccountFn(userUUID, tenantID)
	}
	return &UserServiceDataResult{}, nil
}
func (m *mockUserService) DeleteByUUID(_ context.Context, userUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(userUUID, tenantID, deleterUserUUID)
	}
	return &UserServiceDataResult{}, nil
}
func (m *mockUserService) AssignUserRoles(_ context.Context, userUUID uuid.UUID, roleUUIDs []uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	if m.assignUserRolesFn != nil {
		return m.assignUserRolesFn(userUUID, roleUUIDs, tenantID)
	}
	return &UserServiceDataResult{}, nil
}
func (m *mockUserService) RemoveUserRole(_ context.Context, userUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	if m.removeUserRoleFn != nil {
		return m.removeUserRoleFn(userUUID, roleUUID, tenantID)
	}
	return &UserServiceDataResult{}, nil
}
func (m *mockUserService) GetUserRoles(_ context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserRolesFilter) ([]RoleServiceDataResult, int64, error) {
	if m.getUserRolesFn != nil {
		return m.getUserRolesFn(userUUID, tenantID, filter)
	}
	return nil, 0, nil
}
func (m *mockUserService) GetUserIdentities(_ context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserIdentitiesFilter) ([]UserIdentityServiceDataResult, int64, error) {
	if m.getUserIdentitiesFn != nil {
		return m.getUserIdentitiesFn(userUUID, tenantID, filter)
	}
	if m.getUserIdentsFn != nil {
		return m.getUserIdentsFn(userUUID, tenantID, filter)
	}
	return nil, 0, nil
}
func (m *mockUserService) GetUserSessions(_ context.Context, _ uuid.UUID, _ int64) ([]*SessionDataResult, error) {
	return nil, nil
}
func (m *mockUserService) RevokeUserSession(_ context.Context, _ uuid.UUID, _ int64, _ uuid.UUID) error {
	return nil
}
func (m *mockUserService) FindBySubAndClientID(_ context.Context, sub string, clientID string) (*User, error) {
	if m.findBySubAndClientIDFn != nil {
		return m.findBySubAndClientIDFn(sub, clientID)
	}
	return nil, nil
}
func (m *mockUserService) FindByUserID(_ context.Context, userID int64) (*User, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(userID)
	}
	return nil, nil
}
func (m *mockUserService) ForcePasswordChange(_ context.Context, userUUID uuid.UUID, tenantID int64, force bool) error {
	if m.forcePasswordChangeFn != nil {
		return m.forcePasswordChangeFn(userUUID, force)
	}
	return nil
}
func (m *mockUserService) GetUserMFA(_ context.Context, userUUID uuid.UUID, tenantID int64) (*UserMFAResponseDTO, error) {
	if m.getUserMFAFn != nil {
		return m.getUserMFAFn(userUUID, tenantID)
	}
	return &UserMFAResponseDTO{}, nil
}

func (m *mockUserService) EnsureUserInTenant(_ context.Context, userUUID uuid.UUID, tenantID int64) (int64, error) {
	if m.ensureUserInTenantFn != nil {
		return m.ensureUserInTenantFn(userUUID, tenantID)
	}
	return 0, nil
}

func (m *mockUserService) GrantRoleByName(_ context.Context, userUUID uuid.UUID, tenantID int64, roleName string) error {
	if m.grantRoleByNameFn != nil {
		return m.grantRoleByNameFn(userUUID, tenantID, roleName)
	}
	return nil
}

type mockIdentityUnlinker struct {
	adminUnlinkIdentityFn func(context.Context, int64, int64, uuid.UUID, string) error
}

func (m *mockIdentityUnlinker) AdminUnlinkIdentity(ctx context.Context, tenantID int64, actorUserID int64, userUUID uuid.UUID, identityUUID string) error {
	if m.adminUnlinkIdentityFn != nil {
		return m.adminUnlinkIdentityFn(ctx, tenantID, actorUserID, userUUID, identityUUID)
	}
	return nil
}

type mockProfileService struct {
	createOrUpdateFn         func(uuid.UUID, string, *string, *string, *string, *string, *string, *time.Time, *string, *string, *string, *string, *string, *string, *string, *string, *string, map[string]any) (*ProfileServiceDataResult, error)
	createOrUpdateSpecificFn func(uuid.UUID, uuid.UUID, string, *string, *string, *string, *string, *string, *time.Time, *string, *string, *string, *string, *string, *string, *string, *string, *string, map[string]any) (*ProfileServiceDataResult, error)
	getByUUIDFn              func(uuid.UUID, uuid.UUID) (*ProfileServiceDataResult, error)
	getByUserUUIDFn          func(uuid.UUID) (*ProfileServiceDataResult, error)
	getAllFn                 func(uuid.UUID, *string, *string, *string, *string, *string, *string, *bool, int, int, string, string) (*ProfileServiceListResult, error)
	setDefaultProfileFn      func(uuid.UUID, uuid.UUID) (*ProfileServiceDataResult, error)
	setDefaultFn             func(uuid.UUID, uuid.UUID) (*ProfileServiceDataResult, error)
	deleteByUUIDFn           func(uuid.UUID, uuid.UUID) (*ProfileServiceDataResult, error)
}

func (m *mockProfileService) CreateOrUpdateProfile(_ context.Context, userUUID uuid.UUID, firstName string, middleName, lastName, suffix, displayName, bio *string, birthdate *time.Time, gender *string, phone, email, address *string, city, country *string, timezone, language *string, profileURL *string, metadata map[string]any) (*ProfileServiceDataResult, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(userUUID, firstName, middleName, lastName, suffix, displayName, bio, birthdate, gender, phone, email, address, city, country, timezone, language, profileURL, metadata)
	}
	return &ProfileServiceDataResult{}, nil
}
func (m *mockProfileService) CreateOrUpdateSpecificProfile(_ context.Context, profileUUID uuid.UUID, userUUID uuid.UUID, firstName string, middleName, lastName, suffix, displayName, bio *string, birthdate *time.Time, gender *string, phone, email, address *string, city, country *string, timezone, language *string, profileURL *string, metadata map[string]any) (*ProfileServiceDataResult, error) {
	if m.createOrUpdateSpecificFn != nil {
		return m.createOrUpdateSpecificFn(profileUUID, userUUID, firstName, middleName, lastName, suffix, displayName, bio, birthdate, gender, phone, email, address, city, country, timezone, language, profileURL, metadata)
	}
	return &ProfileServiceDataResult{}, nil
}
func (m *mockProfileService) GetByUUID(_ context.Context, profileUUID uuid.UUID, userUUID uuid.UUID) (*ProfileServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(profileUUID, userUUID)
	}
	return &ProfileServiceDataResult{}, nil
}
func (m *mockProfileService) GetByUserUUID(_ context.Context, userUUID uuid.UUID) (*ProfileServiceDataResult, error) {
	if m.getByUserUUIDFn != nil {
		return m.getByUserUUIDFn(userUUID)
	}
	return nil, nil
}
func (m *mockProfileService) GetAll(_ context.Context, userUUID uuid.UUID, firstName, lastName, email, phone, city, country *string, isDefault *bool, page, limit int, sortBy, sortOrder string) (*ProfileServiceListResult, error) {
	if m.getAllFn != nil {
		return m.getAllFn(userUUID, firstName, lastName, email, phone, city, country, isDefault, page, limit, sortBy, sortOrder)
	}
	return &ProfileServiceListResult{}, nil
}
func (m *mockProfileService) SetDefaultProfile(_ context.Context, profileUUID uuid.UUID, userUUID uuid.UUID) (*ProfileServiceDataResult, error) {
	if m.setDefaultProfileFn != nil {
		return m.setDefaultProfileFn(profileUUID, userUUID)
	}
	if m.setDefaultFn != nil {
		return m.setDefaultFn(profileUUID, userUUID)
	}
	return &ProfileServiceDataResult{}, nil
}
func (m *mockProfileService) DeleteByUUID(_ context.Context, profileUUID uuid.UUID, userUUID uuid.UUID) (*ProfileServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(profileUUID, userUUID)
	}
	return &ProfileServiceDataResult{}, nil
}

type mockUserSettingService struct {
	createOrUpdateFn            func(uuid.UUID, *string, *string, *string, map[string]any, *string, *bool, *bool, *bool, *string, *bool, *time.Time, *time.Time, *string, *string, *string, *string) (*UserSettingServiceDataResult, error)
	createOrUpdateUserSettingFn func(uuid.UUID, *string, *string, *string, map[string]any, *string, *bool, *bool, *bool, *string, *bool, *time.Time, *time.Time, *string, *string, *string, *string) (*UserSettingServiceDataResult, error)
	getByUUIDFn                 func(uuid.UUID, int64) (*UserSettingServiceDataResult, error)
	getByUserUUIDFn             func(uuid.UUID) (*UserSettingServiceDataResult, error)
	deleteByUUIDFn              func(uuid.UUID, int64) (*UserSettingServiceDataResult, error)
}

func (m *mockUserSettingService) CreateOrUpdateUserSetting(_ context.Context, userUUID uuid.UUID, timezone, preferredLanguage, locale *string, socialLinks map[string]any, preferredContactMethod *string, marketingEmailConsent, smsNotificationsConsent, pushNotificationsConsent *bool, profileVisibility *string, dataProcessingConsent *bool, termsAcceptedAt, privacyPolicyAcceptedAt *time.Time, emergencyContactName, emergencyContactPhone, emergencyContactEmail, emergencyContactRelation *string) (*UserSettingServiceDataResult, error) {
	if m.createOrUpdateUserSettingFn != nil {
		return m.createOrUpdateUserSettingFn(userUUID, timezone, preferredLanguage, locale, socialLinks, preferredContactMethod, marketingEmailConsent, smsNotificationsConsent, pushNotificationsConsent, profileVisibility, dataProcessingConsent, termsAcceptedAt, privacyPolicyAcceptedAt, emergencyContactName, emergencyContactPhone, emergencyContactEmail, emergencyContactRelation)
	}
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(userUUID, timezone, preferredLanguage, locale, socialLinks, preferredContactMethod, marketingEmailConsent, smsNotificationsConsent, pushNotificationsConsent, profileVisibility, dataProcessingConsent, termsAcceptedAt, privacyPolicyAcceptedAt, emergencyContactName, emergencyContactPhone, emergencyContactEmail, emergencyContactRelation)
	}
	return &UserSettingServiceDataResult{}, nil
}
func (m *mockUserSettingService) GetByUUID(_ context.Context, userSettingUUID uuid.UUID, userID int64) (*UserSettingServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(userSettingUUID, userID)
	}
	return &UserSettingServiceDataResult{}, nil
}
func (m *mockUserSettingService) GetByUserUUID(_ context.Context, userUUID uuid.UUID) (*UserSettingServiceDataResult, error) {
	if m.getByUserUUIDFn != nil {
		return m.getByUserUUIDFn(userUUID)
	}
	return nil, nil
}
func (m *mockUserSettingService) DeleteByUUID(_ context.Context, userSettingUUID uuid.UUID, userID int64) (*UserSettingServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(userSettingUUID, userID)
	}
	return &UserSettingServiceDataResult{}, nil
}

type mockUserTokenRepo struct {
	mockBaseRepo[UserToken]
	findByUserIDFn              func(int64) ([]UserToken, error)
	findActiveTokensByUserIDFn  func(int64) ([]UserToken, error)
	findByUserIDAndTokenTypeFn  func(int64, string) ([]UserToken, error)
	revokeByUUIDFn              func(uuid.UUID) error
	revokeAllByUserIDFn         func(int64) error
	deleteByUserIDFn            func(int64) error
	deleteExpiredTokensFn       func(time.Time) error
	findActiveSessionsFn        func(int64) ([]UserToken, error)
	findActiveSessionByUUIDFn   func(int64, uuid.UUID) (*UserToken, error)
	countActiveSessionsFn       func(int64) (int64, error)
	touchSessionFn              func(int64, uuid.UUID, time.Time) error
	revokeSessionByUUIDFn       func(int64, uuid.UUID) error
	revokeAllSessionsByUserIDFn func(int64) error
	createFn                    func(*UserToken) (*UserToken, error)
}

func (m *mockUserTokenRepo) WithTx(_ *gorm.DB) UserTokenRepository { return m }
func (m *mockUserTokenRepo) Create(e *UserToken) (*UserToken, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockUserTokenRepo) FindByUserID(userID int64) ([]UserToken, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(userID)
	}
	return nil, nil
}
func (m *mockUserTokenRepo) FindActiveTokensByUserID(userID int64) ([]UserToken, error) {
	if m.findActiveTokensByUserIDFn != nil {
		return m.findActiveTokensByUserIDFn(userID)
	}
	return nil, nil
}
func (m *mockUserTokenRepo) FindByUserIDAndTokenType(userID int64, tokenType string) ([]UserToken, error) {
	if m.findByUserIDAndTokenTypeFn != nil {
		return m.findByUserIDAndTokenTypeFn(userID, tokenType)
	}
	return nil, nil
}
func (m *mockUserTokenRepo) RevokeByUUID(tokenUUID uuid.UUID) error {
	if m.revokeByUUIDFn != nil {
		return m.revokeByUUIDFn(tokenUUID)
	}
	return nil
}
func (m *mockUserTokenRepo) RevokeAllByUserID(userID int64) error {
	if m.revokeAllByUserIDFn != nil {
		return m.revokeAllByUserIDFn(userID)
	}
	return nil
}
func (m *mockUserTokenRepo) DeleteByUserID(userID int64) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(userID)
	}
	return nil
}
func (m *mockUserTokenRepo) DeleteExpiredTokens(before time.Time) error {
	if m.deleteExpiredTokensFn != nil {
		return m.deleteExpiredTokensFn(before)
	}
	return nil
}
func (m *mockUserTokenRepo) FindActiveSessions(userID int64) ([]UserToken, error) {
	if m.findActiveSessionsFn != nil {
		return m.findActiveSessionsFn(userID)
	}
	return nil, nil
}
func (m *mockUserTokenRepo) FindActiveSessionByUUID(userID int64, sessionUUID uuid.UUID) (*UserToken, error) {
	if m.findActiveSessionByUUIDFn != nil {
		return m.findActiveSessionByUUIDFn(userID, sessionUUID)
	}
	return nil, nil
}
func (m *mockUserTokenRepo) CountActiveSessions(userID int64) (int64, error) {
	if m.countActiveSessionsFn != nil {
		return m.countActiveSessionsFn(userID)
	}
	return 0, nil
}
func (m *mockUserTokenRepo) TouchSession(userID int64, sessionUUID uuid.UUID, now time.Time) error {
	if m.touchSessionFn != nil {
		return m.touchSessionFn(userID, sessionUUID, now)
	}
	return nil
}
func (m *mockUserTokenRepo) RevokeSessionByUUID(userID int64, sessionUUID uuid.UUID) error {
	if m.revokeSessionByUUIDFn != nil {
		return m.revokeSessionByUUIDFn(userID, sessionUUID)
	}
	return nil
}
func (m *mockUserTokenRepo) RevokeAllSessionsByUserID(userID int64) error {
	if m.revokeAllSessionsByUserIDFn != nil {
		return m.revokeAllSessionsByUserIDFn(userID)
	}
	return nil
}

type mockUserMFABackupCodeRepo struct {
	mockBaseRepo[UserMFABackupCode]
	createBulkFn              func([]*UserMFABackupCode) error
	findUnusedByUserIDFn      func(int64) ([]UserMFABackupCode, error)
	findByUserIDAndCodeHashFn func(int64, string) (*UserMFABackupCode, error)
	markUsedFn                func(int64) error
	deleteAllByUserIDFn       func(int64) error
}

func (m *mockUserMFABackupCodeRepo) WithTx(_ *gorm.DB) UserMFABackupCodeRepository { return m }
func (m *mockUserMFABackupCodeRepo) CreateBulk(codes []*UserMFABackupCode) error {
	if m.createBulkFn != nil {
		return m.createBulkFn(codes)
	}
	return nil
}
func (m *mockUserMFABackupCodeRepo) FindUnusedByUserID(userID int64) ([]UserMFABackupCode, error) {
	if m.findUnusedByUserIDFn != nil {
		return m.findUnusedByUserIDFn(userID)
	}
	return nil, nil
}
func (m *mockUserMFABackupCodeRepo) FindByUserIDAndCodeHash(userID int64, codeHash string) (*UserMFABackupCode, error) {
	if m.findByUserIDAndCodeHashFn != nil {
		return m.findByUserIDAndCodeHashFn(userID, codeHash)
	}
	return nil, nil
}
func (m *mockUserMFABackupCodeRepo) MarkUsed(id int64) error {
	if m.markUsedFn != nil {
		return m.markUsedFn(id)
	}
	return nil
}
func (m *mockUserMFABackupCodeRepo) DeleteAllByUserID(userID int64) error {
	if m.deleteAllByUserIDFn != nil {
		return m.deleteAllByUserIDFn(userID)
	}
	return nil
}
