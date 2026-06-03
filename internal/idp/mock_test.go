package idp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authctx"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	errNotFound   = apperror.NewNotFoundWithReason("not found")
	errValidation = apperror.NewValidation("validation error")
)

const tenantID int64 = 1

var (
	testTenantUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testUserUUID   = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

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

func newMockGormDBRegex(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return gormDB, mock
}

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

func validPagination() PaginationRequestDTO {
	return PaginationRequestDTO{Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc}
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

type mockIdentityProviderRepo struct {
	mockBaseRepo[IdentityProvider]
	findByUUIDFn              func(any, ...string) (*IdentityProvider, error)
	findByNameFn              func(string, int64) (*IdentityProvider, error)
	findByIdentifierFn        func(string) (*IdentityProvider, error)
	findDefaultByTenantIDFn   func(int64) (*IdentityProvider, error)
	findByTenantAndProviderFn func(int64, string) (*IdentityProvider, error)
	findAllByTenantIDFn       func(int64) ([]IdentityProvider, error)
	findPaginatedFn           func(IdentityProviderRepositoryGetFilter) (*PaginationResult[IdentityProvider], error)
	createOrUpdateFn          func(*IdentityProvider) (*IdentityProvider, error)
	deleteByUUIDFn            func(any) error
}

func (m *mockIdentityProviderRepo) WithTx(_ *gorm.DB) IdentityProviderRepository { return m }
func (m *mockIdentityProviderRepo) FindByUUID(id any, p ...string) (*IdentityProvider, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockIdentityProviderRepo) CreateOrUpdate(e *IdentityProvider) (*IdentityProvider, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockIdentityProviderRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockIdentityProviderRepo) FindByName(name string, tenantID int64) (*IdentityProvider, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name, tenantID)
	}
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindByIdentifier(identifier string) (*IdentityProvider, error) {
	if m.findByIdentifierFn != nil {
		return m.findByIdentifierFn(identifier)
	}
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindDefaultByTenantID(tenantID int64) (*IdentityProvider, error) {
	if m.findDefaultByTenantIDFn != nil {
		return m.findDefaultByTenantIDFn(tenantID)
	}
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindByTenantAndProvider(tenantID int64, provider string) (*IdentityProvider, error) {
	if m.findByTenantAndProviderFn != nil {
		return m.findByTenantAndProviderFn(tenantID, provider)
	}
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindAllByTenantID(tenantID int64) ([]IdentityProvider, error) {
	if m.findAllByTenantIDFn != nil {
		return m.findAllByTenantIDFn(tenantID)
	}
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindPaginated(f IdentityProviderRepositoryGetFilter) (*PaginationResult[IdentityProvider], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[IdentityProvider]{}, nil
}

type mockSignupFlowRepo struct {
	mockBaseRepo[SignupFlow]
	findByUUIDFn                  func(any, ...string) (*SignupFlow, error)
	findPaginatedFn               func(SignupFlowRepositoryGetFilter) (*PaginationResult[SignupFlow], error)
	findByUUIDAndTenantIDFn       func(uuid.UUID, int64, ...string) (*SignupFlow, error)
	findByIdentifierAndClientIDFn func(string, int64) (*SignupFlow, error)
	findByNameFn                  func(string) (*SignupFlow, error)
	createFn                      func(*SignupFlow) (*SignupFlow, error)
	createOrUpdateFn              func(*SignupFlow) (*SignupFlow, error)
	deleteByUUIDFn                func(any) error
}

func (m *mockSignupFlowRepo) WithTx(_ *gorm.DB) SignupFlowRepository { return m }
func (m *mockSignupFlowRepo) Create(e *SignupFlow) (*SignupFlow, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockSignupFlowRepo) CreateOrUpdate(e *SignupFlow) (*SignupFlow, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockSignupFlowRepo) FindByUUID(id any, p ...string) (*SignupFlow, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockSignupFlowRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockSignupFlowRepo) FindPaginated(f SignupFlowRepositoryGetFilter) (*PaginationResult[SignupFlow], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[SignupFlow]{}, nil
}
func (m *mockSignupFlowRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64, p ...string) (*SignupFlow, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tenantID, p...)
	}
	return nil, nil
}
func (m *mockSignupFlowRepo) FindByIdentifierAndClientID(identifier string, clientID int64) (*SignupFlow, error) {
	if m.findByIdentifierAndClientIDFn != nil {
		return m.findByIdentifierAndClientIDFn(identifier, clientID)
	}
	return nil, nil
}
func (m *mockSignupFlowRepo) FindByName(name string) (*SignupFlow, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name)
	}
	return nil, nil
}

type mockSignupFlowRoleRepo struct {
	mockBaseRepo[SignupFlowRole]
	findBySignupFlowIDFn            func(int64) ([]SignupFlowRole, error)
	findBySignupFlowIDPaginatedFn   func(int64, int, int) ([]SignupFlowRole, int64, error)
	deleteBySignupFlowIDAndRoleIDFn func(int64, int64) error
	findBySignupFlowIDAndRoleIDFn   func(int64, int64) (*SignupFlowRole, error)
	createFn                        func(*SignupFlowRole) (*SignupFlowRole, error)
}

func (m *mockSignupFlowRoleRepo) WithTx(_ *gorm.DB) SignupFlowRoleRepository { return m }
func (m *mockSignupFlowRoleRepo) Create(e *SignupFlowRole) (*SignupFlowRole, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockSignupFlowRoleRepo) FindBySignupFlowID(signupFlowID int64) ([]SignupFlowRole, error) {
	if m.findBySignupFlowIDFn != nil {
		return m.findBySignupFlowIDFn(signupFlowID)
	}
	return nil, nil
}
func (m *mockSignupFlowRoleRepo) FindBySignupFlowIDPaginated(signupFlowID int64, page, limit int) ([]SignupFlowRole, int64, error) {
	if m.findBySignupFlowIDPaginatedFn != nil {
		return m.findBySignupFlowIDPaginatedFn(signupFlowID, page, limit)
	}
	return nil, 0, nil
}
func (m *mockSignupFlowRoleRepo) DeleteBySignupFlowIDAndRoleID(signupFlowID, roleID int64) error {
	if m.deleteBySignupFlowIDAndRoleIDFn != nil {
		return m.deleteBySignupFlowIDAndRoleIDFn(signupFlowID, roleID)
	}
	return nil
}
func (m *mockSignupFlowRoleRepo) FindBySignupFlowIDAndRoleID(signupFlowID, roleID int64) (*SignupFlowRole, error) {
	if m.findBySignupFlowIDAndRoleIDFn != nil {
		return m.findBySignupFlowIDAndRoleIDFn(signupFlowID, roleID)
	}
	return nil, nil
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

type mockUserRepo struct {
	mockBaseRepo[User]
	findByUUIDFn             func(any, ...string) (*User, error)
	findByIDFn               func(any, ...string) (*User, error)
	findByEmailFn            func(string) (*User, error)
	findByEmailAndTenantIDFn func(string, int64) (*User, error)
	createFn                 func(*User) (*User, error)
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

type mockClientRepo struct {
	mockBaseRepo[Client]
	findByUUIDFn                        func(any, ...string) (*Client, error)
	findSystemFn                        func() (*Client, error)
	findByClientIDAndIdentityProviderFn func(string, string) (*Client, error)
}

func (m *mockClientRepo) WithTx(_ *gorm.DB) ClientRepository { return m }
func (m *mockClientRepo) FindByUUID(id any, p ...string) (*Client, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*Client, error) {
	if m.findByClientIDAndIdentityProviderFn != nil {
		return m.findByClientIDAndIdentityProviderFn(clientID, identityProviderIdentifier)
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

type mockIdentityProviderService struct {
	getFn             func(IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error)
	getByUUIDFn       func(uuid.UUID, int64) (*IdentityProviderServiceDataResult, error)
	createFn          func(string, string, string, string, datatypes.JSON, string, string, int64, uuid.UUID) (*IdentityProviderServiceDataResult, error)
	updateFn          func(uuid.UUID, string, string, string, string, datatypes.JSON, string, int64, uuid.UUID) (*IdentityProviderServiceDataResult, error)
	setStatusByUUIDFn func(uuid.UUID, string, int64, uuid.UUID) (*IdentityProviderServiceDataResult, error)
	deleteByUUIDFn    func(uuid.UUID, int64, uuid.UUID) (*IdentityProviderServiceDataResult, error)
}

func (m *mockIdentityProviderService) Get(_ context.Context, f IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &IdentityProviderServiceGetResult{}, nil
}
func (m *mockIdentityProviderService) GetByUUID(_ context.Context, id uuid.UUID, tenantID int64) (*IdentityProviderServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tenantID)
	}
	return nil, nil
}
func (m *mockIdentityProviderService) Create(_ context.Context, name, displayName, provider, providerType string, config datatypes.JSON, status, tenantUUID string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(name, displayName, provider, providerType, config, status, tenantUUID, tenantID, actorUserUUID)
	}
	return nil, nil
}
func (m *mockIdentityProviderService) Update(_ context.Context, id uuid.UUID, name, displayName, provider, providerType string, config datatypes.JSON, status string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, name, displayName, provider, providerType, config, status, tenantID, actorUserUUID)
	}
	return nil, nil
}
func (m *mockIdentityProviderService) SetStatusByUUID(_ context.Context, id uuid.UUID, status string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, status, tenantID, actorUserUUID)
	}
	return nil, nil
}
func (m *mockIdentityProviderService) DeleteByUUID(_ context.Context, id uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id, tenantID, actorUserUUID)
	}
	return nil, nil
}

type mockSignupFlowService struct {
	getAllFn       func(int64, *string, *string, []string, *uuid.UUID, int, int, string, string) (*SignupFlowServiceListResult, error)
	getByUUIDFn    func(uuid.UUID, int64) (*SignupFlowServiceDataResult, error)
	createFn       func(int64, string, string, map[string]any, string, uuid.UUID) (*SignupFlowServiceDataResult, error)
	updateFn       func(uuid.UUID, int64, string, string, map[string]any, string) (*SignupFlowServiceDataResult, error)
	updateStatusFn func(uuid.UUID, int64, string) (*SignupFlowServiceDataResult, error)
	deleteFn       func(uuid.UUID, int64) (*SignupFlowServiceDataResult, error)
	assignRolesFn  func(uuid.UUID, int64, []uuid.UUID) ([]SignupFlowRoleServiceDataResult, error)
	getRolesFn     func(uuid.UUID, int64, int, int) (*SignupFlowRoleServiceListResult, error)
	removeRoleFn   func(uuid.UUID, int64, uuid.UUID) error
}

func (m *mockSignupFlowService) GetAll(_ context.Context, tenantID int64, name, identifier *string, status []string, clientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*SignupFlowServiceListResult, error) {
	if m.getAllFn != nil {
		return m.getAllFn(tenantID, name, identifier, status, clientUUID, page, limit, sortBy, sortOrder)
	}
	return &SignupFlowServiceListResult{}, nil
}
func (m *mockSignupFlowService) GetByUUID(_ context.Context, id uuid.UUID, tenantID int64) (*SignupFlowServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tenantID)
	}
	return nil, nil
}
func (m *mockSignupFlowService) Create(_ context.Context, tenantID int64, name, desc string, config map[string]any, status string, clientUUID uuid.UUID) (*SignupFlowServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tenantID, name, desc, config, status, clientUUID)
	}
	return nil, nil
}
func (m *mockSignupFlowService) Update(_ context.Context, id uuid.UUID, tenantID int64, name, desc string, config map[string]any, status string) (*SignupFlowServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tenantID, name, desc, config, status)
	}
	return nil, nil
}
func (m *mockSignupFlowService) UpdateStatus(_ context.Context, id uuid.UUID, tenantID int64, status string) (*SignupFlowServiceDataResult, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(id, tenantID, status)
	}
	return nil, nil
}
func (m *mockSignupFlowService) Delete(_ context.Context, id uuid.UUID, tenantID int64) (*SignupFlowServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(id, tenantID)
	}
	return nil, nil
}
func (m *mockSignupFlowService) AssignRoles(_ context.Context, id uuid.UUID, tenantID int64, roles []uuid.UUID) ([]SignupFlowRoleServiceDataResult, error) {
	if m.assignRolesFn != nil {
		return m.assignRolesFn(id, tenantID, roles)
	}
	return nil, nil
}
func (m *mockSignupFlowService) GetRoles(_ context.Context, id uuid.UUID, tenantID int64, page, limit int) (*SignupFlowRoleServiceListResult, error) {
	if m.getRolesFn != nil {
		return m.getRolesFn(id, tenantID, page, limit)
	}
	return &SignupFlowRoleServiceListResult{}, nil
}
func (m *mockSignupFlowService) RemoveRole(_ context.Context, id uuid.UUID, tenantID int64, roleUUID uuid.UUID) error {
	if m.removeRoleFn != nil {
		return m.removeRoleFn(id, tenantID, roleUUID)
	}
	return nil
}
