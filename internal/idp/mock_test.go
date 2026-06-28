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
	findByUUIDSafeFn          func(any, ...string) (*IdentityProvider, error)
	findByNameFn              func(string, int64) (*IdentityProvider, error)
	findByIdentifierFn        func(string) (*IdentityProvider, error)
	findByIssuerFn            func(string) (*IdentityProvider, error)
	findByIDFn                func(any, ...string) (*IdentityProvider, error)
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
func (m *mockIdentityProviderRepo) FindByUUIDSafe(id any, p ...string) (*IdentityProvider, error) {
	if m.findByUUIDSafeFn != nil {
		return m.findByUUIDSafeFn(id, p...)
	}
	// Default to the regular find so existing tests that only stub findByUUIDFn
	// continue to exercise the read path.
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindByID(id any, p ...string) (*IdentityProvider, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id, p...)
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
func (m *mockIdentityProviderRepo) FindByIssuer(issuer string) (*IdentityProvider, error) {
	if m.findByIssuerFn != nil {
		return m.findByIssuerFn(issuer)
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

type mockIdentityProviderEmailDomainRepo struct {
	findByTenantAndDomainFn func(int64, string) (*IdentityProviderEmailDomain, error)
	findByProviderIDFn      func(int64) ([]IdentityProviderEmailDomain, error)
	replaceForProviderFn    func(int64, int64, []string) error
}

func (m *mockIdentityProviderEmailDomainRepo) WithTx(_ *gorm.DB) IdentityProviderEmailDomainRepository {
	return m
}
func (m *mockIdentityProviderEmailDomainRepo) FindByTenantAndDomain(tenantID int64, domain string) (*IdentityProviderEmailDomain, error) {
	if m.findByTenantAndDomainFn != nil {
		return m.findByTenantAndDomainFn(tenantID, domain)
	}
	return nil, nil
}
func (m *mockIdentityProviderEmailDomainRepo) FindByProviderID(idpID int64) ([]IdentityProviderEmailDomain, error) {
	if m.findByProviderIDFn != nil {
		return m.findByProviderIDFn(idpID)
	}
	return nil, nil
}
func (m *mockIdentityProviderEmailDomainRepo) ReplaceForProvider(tenantID, idpID int64, domains []string) error {
	if m.replaceForProviderFn != nil {
		return m.replaceForProviderFn(tenantID, idpID, domains)
	}
	return nil
}

type mockAuthFlowRepo struct {
	mockBaseRepo[AuthFlow]
	findByUUIDFn                  func(any, ...string) (*AuthFlow, error)
	findPaginatedFn               func(AuthFlowRepositoryGetFilter) (*PaginationResult[AuthFlow], error)
	findByUUIDAndTenantIDFn       func(uuid.UUID, int64, ...string) (*AuthFlow, error)
	findByIdentifierAndClientIDFn func(string, int64) (*AuthFlow, error)
	findByClientIDFn              func(int64) ([]AuthFlow, error)
	findByNameFn                  func(string) (*AuthFlow, error)
	findByNameAndTenantIDFn       func(string, int64) (*AuthFlow, error)
	createFn                      func(*AuthFlow) (*AuthFlow, error)
	createOrUpdateFn              func(*AuthFlow) (*AuthFlow, error)
	deleteByUUIDFn                func(any) error
}

func (m *mockAuthFlowRepo) WithTx(_ *gorm.DB) AuthFlowRepository { return m }
func (m *mockAuthFlowRepo) Create(e *AuthFlow) (*AuthFlow, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockAuthFlowRepo) CreateOrUpdate(e *AuthFlow) (*AuthFlow, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockAuthFlowRepo) FindByUUID(id any, p ...string) (*AuthFlow, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockAuthFlowRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockAuthFlowRepo) FindPaginated(f AuthFlowRepositoryGetFilter) (*PaginationResult[AuthFlow], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[AuthFlow]{}, nil
}
func (m *mockAuthFlowRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64, p ...string) (*AuthFlow, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tenantID, p...)
	}
	return nil, nil
}
func (m *mockAuthFlowRepo) FindByIdentifierAndClientID(identifier string, clientID int64) (*AuthFlow, error) {
	if m.findByIdentifierAndClientIDFn != nil {
		return m.findByIdentifierAndClientIDFn(identifier, clientID)
	}
	return nil, nil
}
func (m *mockAuthFlowRepo) FindByClientID(clientID int64) ([]AuthFlow, error) {
	if m.findByClientIDFn != nil {
		return m.findByClientIDFn(clientID)
	}
	return nil, nil
}
func (m *mockAuthFlowRepo) FindByName(name string) (*AuthFlow, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name)
	}
	return nil, nil
}
func (m *mockAuthFlowRepo) FindByNameAndTenantID(name string, tenantID int64) (*AuthFlow, error) {
	if m.findByNameAndTenantIDFn != nil {
		return m.findByNameAndTenantIDFn(name, tenantID)
	}
	return nil, nil
}

type mockAuthFlowRoleRepo struct {
	mockBaseRepo[AuthFlowRole]
	findByAuthFlowIDFn            func(int64) ([]AuthFlowRole, error)
	findByAuthFlowIDPaginatedFn   func(int64, int, int) ([]AuthFlowRole, int64, error)
	deleteByAuthFlowIDAndRoleIDFn func(int64, int64) error
	findByAuthFlowIDAndRoleIDFn   func(int64, int64) (*AuthFlowRole, error)
	createFn                      func(*AuthFlowRole) (*AuthFlowRole, error)
}

func (m *mockAuthFlowRoleRepo) WithTx(_ *gorm.DB) AuthFlowRoleRepository { return m }
func (m *mockAuthFlowRoleRepo) Create(e *AuthFlowRole) (*AuthFlowRole, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockAuthFlowRoleRepo) FindByAuthFlowID(authFlowID int64) ([]AuthFlowRole, error) {
	if m.findByAuthFlowIDFn != nil {
		return m.findByAuthFlowIDFn(authFlowID)
	}
	return nil, nil
}
func (m *mockAuthFlowRoleRepo) FindByAuthFlowIDPaginated(authFlowID int64, page, limit int) ([]AuthFlowRole, int64, error) {
	if m.findByAuthFlowIDPaginatedFn != nil {
		return m.findByAuthFlowIDPaginatedFn(authFlowID, page, limit)
	}
	return nil, 0, nil
}
func (m *mockAuthFlowRoleRepo) DeleteByAuthFlowIDAndRoleID(authFlowID, roleID int64) error {
	if m.deleteByAuthFlowIDAndRoleIDFn != nil {
		return m.deleteByAuthFlowIDAndRoleIDFn(authFlowID, roleID)
	}
	return nil
}
func (m *mockAuthFlowRoleRepo) FindByAuthFlowIDAndRoleID(authFlowID, roleID int64) (*AuthFlowRole, error) {
	if m.findByAuthFlowIDAndRoleIDFn != nil {
		return m.findByAuthFlowIDAndRoleIDFn(authFlowID, roleID)
	}
	return nil, nil
}

type mockAuthFlowCallbackURIRepo struct {
	mockBaseRepo[AuthFlowCallbackURI]
	findByAuthFlowIDFn                 func(int64) ([]AuthFlowCallbackURI, error)
	findByAuthFlowIDPaginatedFn        func(int64, int, int) ([]AuthFlowCallbackURI, int64, error)
	deleteByAuthFlowIDAndClientURIIDFn func(int64, int64) error
	findByAuthFlowIDAndClientURIIDFn   func(int64, int64) (*AuthFlowCallbackURI, error)
	createFn                           func(*AuthFlowCallbackURI) (*AuthFlowCallbackURI, error)
}

func (m *mockAuthFlowCallbackURIRepo) WithTx(_ *gorm.DB) AuthFlowCallbackURIRepository { return m }
func (m *mockAuthFlowCallbackURIRepo) Create(e *AuthFlowCallbackURI) (*AuthFlowCallbackURI, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockAuthFlowCallbackURIRepo) FindByAuthFlowID(authFlowID int64) ([]AuthFlowCallbackURI, error) {
	if m.findByAuthFlowIDFn != nil {
		return m.findByAuthFlowIDFn(authFlowID)
	}
	return nil, nil
}
func (m *mockAuthFlowCallbackURIRepo) FindByAuthFlowIDPaginated(authFlowID int64, page, limit int) ([]AuthFlowCallbackURI, int64, error) {
	if m.findByAuthFlowIDPaginatedFn != nil {
		return m.findByAuthFlowIDPaginatedFn(authFlowID, page, limit)
	}
	return nil, 0, nil
}
func (m *mockAuthFlowCallbackURIRepo) DeleteByAuthFlowIDAndClientURIID(authFlowID, clientURIID int64) error {
	if m.deleteByAuthFlowIDAndClientURIIDFn != nil {
		return m.deleteByAuthFlowIDAndClientURIIDFn(authFlowID, clientURIID)
	}
	return nil
}
func (m *mockAuthFlowCallbackURIRepo) FindByAuthFlowIDAndClientURIID(authFlowID, clientURIID int64) (*AuthFlowCallbackURI, error) {
	if m.findByAuthFlowIDAndClientURIIDFn != nil {
		return m.findByAuthFlowIDAndClientURIIDFn(authFlowID, clientURIID)
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
	findByIDFn                          func(any, ...string) (*Client, error)
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
func (m *mockClientRepo) FindByID(id any, p ...string) (*Client, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id, p...)
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
	createFn          func(IdentityProviderCreateInput) (*IdentityProviderServiceDataResult, error)
	updateFn          func(IdentityProviderUpdateInput) (*IdentityProviderServiceDataResult, error)
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
func (m *mockIdentityProviderService) Create(_ context.Context, in IdentityProviderCreateInput) (*IdentityProviderServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(in)
	}
	return nil, nil
}
func (m *mockIdentityProviderService) Update(_ context.Context, in IdentityProviderUpdateInput) (*IdentityProviderServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(in)
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

type mockAuthFlowService struct {
	getAllFn       func(int64, *string, *string, []string, *uuid.UUID, int, int, string, string) (*AuthFlowServiceListResult, error)
	getByUUIDFn    func(uuid.UUID, int64) (*AuthFlowServiceDataResult, error)
	createFn       func(int64, string, string, string, uuid.UUID) (*AuthFlowServiceDataResult, error)
	updateFn       func(uuid.UUID, int64, string, string, string) (*AuthFlowServiceDataResult, error)
	updateStatusFn func(uuid.UUID, int64, string) (*AuthFlowServiceDataResult, error)
	deleteFn       func(uuid.UUID, int64) (*AuthFlowServiceDataResult, error)
	assignRolesFn  func(uuid.UUID, int64, []uuid.UUID) ([]AuthFlowRoleServiceDataResult, error)
	getRolesFn     func(uuid.UUID, int64, int, int) (*AuthFlowRoleServiceListResult, error)
	removeRoleFn   func(uuid.UUID, int64, uuid.UUID) error

	assignCallbackURIsFn func(uuid.UUID, int64, []uuid.UUID) ([]AuthFlowCallbackURIServiceDataResult, error)
	getCallbackURIsFn    func(uuid.UUID, int64, int, int) (*AuthFlowCallbackURIServiceListResult, error)
	removeCallbackURIFn  func(uuid.UUID, int64, uuid.UUID) error
}

func (m *mockAuthFlowService) GetAll(_ context.Context, tenantID int64, name, identifier *string, status []string, clientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*AuthFlowServiceListResult, error) {
	if m.getAllFn != nil {
		return m.getAllFn(tenantID, name, identifier, status, clientUUID, page, limit, sortBy, sortOrder)
	}
	return &AuthFlowServiceListResult{}, nil
}
func (m *mockAuthFlowService) GetByUUID(_ context.Context, id uuid.UUID, tenantID int64) (*AuthFlowServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tenantID)
	}
	return nil, nil
}
func (m *mockAuthFlowService) Create(_ context.Context, tenantID int64, name, desc, status, destination string, clientUUID uuid.UUID, _ *uuid.UUID, _, _ []uuid.UUID, _ bool, _ bool, _ string) (*AuthFlowServiceDataResult, error) {
	return m.createFn(tenantID, name, desc, status, clientUUID)
}
func (m *mockAuthFlowService) Update(_ context.Context, id uuid.UUID, tenantID int64, name, desc, status string, _ *uuid.UUID, _, _ []uuid.UUID, _ bool, _ bool, _ string) (*AuthFlowServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tenantID, name, desc, status)
	}
	return nil, nil
}
func (m *mockAuthFlowService) UpdateStatus(_ context.Context, id uuid.UUID, tenantID int64, status string) (*AuthFlowServiceDataResult, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(id, tenantID, status)
	}
	return nil, nil
}
func (m *mockAuthFlowService) Delete(_ context.Context, id uuid.UUID, tenantID int64) (*AuthFlowServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(id, tenantID)
	}
	return nil, nil
}
func (m *mockAuthFlowService) AssignRoles(_ context.Context, id uuid.UUID, tenantID int64, roles []uuid.UUID) ([]AuthFlowRoleServiceDataResult, error) {
	if m.assignRolesFn != nil {
		return m.assignRolesFn(id, tenantID, roles)
	}
	return nil, nil
}
func (m *mockAuthFlowService) GetRoles(_ context.Context, id uuid.UUID, tenantID int64, page, limit int) (*AuthFlowRoleServiceListResult, error) {
	if m.getRolesFn != nil {
		return m.getRolesFn(id, tenantID, page, limit)
	}
	return &AuthFlowRoleServiceListResult{}, nil
}
func (m *mockAuthFlowService) RemoveRole(_ context.Context, id uuid.UUID, tenantID int64, roleUUID uuid.UUID) error {
	if m.removeRoleFn != nil {
		return m.removeRoleFn(id, tenantID, roleUUID)
	}
	return nil
}
func (m *mockAuthFlowService) AssignCallbackURIs(_ context.Context, id uuid.UUID, tenantID int64, clientURIUUIDs []uuid.UUID) ([]AuthFlowCallbackURIServiceDataResult, error) {
	if m.assignCallbackURIsFn != nil {
		return m.assignCallbackURIsFn(id, tenantID, clientURIUUIDs)
	}
	return nil, nil
}
func (m *mockAuthFlowService) GetCallbackURIs(_ context.Context, id uuid.UUID, tenantID int64, page, limit int) (*AuthFlowCallbackURIServiceListResult, error) {
	if m.getCallbackURIsFn != nil {
		return m.getCallbackURIsFn(id, tenantID, page, limit)
	}
	return &AuthFlowCallbackURIServiceListResult{}, nil
}
func (m *mockAuthFlowService) RemoveCallbackURI(_ context.Context, id uuid.UUID, tenantID int64, clientURIUUID uuid.UUID) error {
	if m.removeCallbackURIFn != nil {
		return m.removeCallbackURIFn(id, tenantID, clientURIUUID)
	}
	return nil
}
