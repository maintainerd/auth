package idp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestMain installs a valid 32-byte AES-256 key for the whole idp test package so
// encrypt/decrypt round-trips (encryptProviderClientSecret, the strict federation
// decrypt path) work without every test wiring the key by hand. Individual tests
// that need a different key or an override still set config.AppEncryptionKey /
// crypto.EncryptAtRest locally and restore it in cleanup.
func TestMain(m *testing.M) {
	config.AppEncryptionKey = []byte("0123456789abcdef0123456789abcdef")
	os.Exit(m.Run())
}

var (
	errNotFound   = apperror.NewNotFoundWithReason("not found")
	errValidation = apperror.NewValidation("validation error")
)

const tenantID int64 = 1

// testActorUserID is the acting user's internal id, mirrored by both the request
// auth context (withTenantAndUser) and the actor the user repo mock resolves, so
// audit-actor and created_by/updated_by assertions line up.
const testActorUserID int64 = 42

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
		User:   &authctx.AuthUser{UserID: testActorUserID, UserUUID: testUserUUID},
	})
}

// decodeEnvelopeData decodes the "data" member of the standard success envelope
// ({success, data, message}) into T. Handler success tests must assert the body,
// not just the status code, so this keeps that one line long.
func decodeEnvelopeData[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var env struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Message string          `json:"message"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	require.True(t, env.Success, "expected a success envelope, got: %s", w.Body.String())
	var out T
	require.NoError(t, json.Unmarshal(env.Data, &out))
	return out
}

// decodeEnvelopeDataMap decodes the envelope's "data" into a generic map so a
// test can assert on the *absence* of a field (e.g. the lean list projection
// must not carry required_fields), which a typed decode cannot express.
func decodeEnvelopeDataMap(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	return decodeEnvelopeData[map[string]any](t, w)
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
	findSystemByTenantIDFn    func(int64) (*IdentityProvider, error)
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
func (m *mockIdentityProviderRepo) FindSystemByTenantID(tenantID int64) (*IdentityProvider, error) {
	if m.findSystemByTenantIDFn != nil {
		return m.findSystemByTenantIDFn(tenantID)
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

type mockIdentityProviderAllowedAudienceRepo struct {
	findByProviderIDFn   func(int64) ([]IdentityProviderAllowedAudience, error)
	replaceForProviderFn func(int64, int64, []string) error
}

func (m *mockIdentityProviderAllowedAudienceRepo) WithTx(_ *gorm.DB) IdentityProviderAllowedAudienceRepository {
	return m
}
func (m *mockIdentityProviderAllowedAudienceRepo) FindByProviderID(idpID int64) ([]IdentityProviderAllowedAudience, error) {
	if m.findByProviderIDFn != nil {
		return m.findByProviderIDFn(idpID)
	}
	return nil, nil
}
func (m *mockIdentityProviderAllowedAudienceRepo) ReplaceForProvider(tenantID, idpID int64, audiences []string) error {
	if m.replaceForProviderFn != nil {
		return m.replaceForProviderFn(tenantID, idpID, audiences)
	}
	return nil
}

type mockRegistrationFlowRepo struct {
	mockBaseRepo[RegistrationFlow]
	findByUUIDFn                func(any, ...string) (*RegistrationFlow, error)
	findByIDFn                  func(any, ...string) (*RegistrationFlow, error)
	findPaginatedFn             func(RegistrationFlowRepositoryGetFilter) (*PaginationResult[RegistrationFlow], error)
	findByUUIDAndTenantIDFn     func(uuid.UUID, int64, ...string) (*RegistrationFlow, error)
	findByNameAndClientTenantFn func(string, int64, int64) (*RegistrationFlow, error)
	findByNameAndTenantIDFn     func(string, int64) (*RegistrationFlow, error)
	createFn                    func(*RegistrationFlow) (*RegistrationFlow, error)
	createOrUpdateFn            func(*RegistrationFlow) (*RegistrationFlow, error)
	deleteByUUIDFn              func(any) error
}

func (m *mockRegistrationFlowRepo) WithTx(_ *gorm.DB) RegistrationFlowRepository { return m }
func (m *mockRegistrationFlowRepo) Create(e *RegistrationFlow) (*RegistrationFlow, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockRegistrationFlowRepo) CreateOrUpdate(e *RegistrationFlow) (*RegistrationFlow, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockRegistrationFlowRepo) FindByUUID(id any, p ...string) (*RegistrationFlow, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockRegistrationFlowRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockRegistrationFlowRepo) FindPaginated(f RegistrationFlowRepositoryGetFilter) (*PaginationResult[RegistrationFlow], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[RegistrationFlow]{}, nil
}
func (m *mockRegistrationFlowRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64, p ...string) (*RegistrationFlow, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tenantID, p...)
	}
	return nil, nil
}
func (m *mockRegistrationFlowRepo) FindByID(id any, p ...string) (*RegistrationFlow, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockRegistrationFlowRepo) FindByNameAndClientTenant(name string, clientID, tenantID int64) (*RegistrationFlow, error) {
	if m.findByNameAndClientTenantFn != nil {
		return m.findByNameAndClientTenantFn(name, clientID, tenantID)
	}
	return nil, nil
}
func (m *mockRegistrationFlowRepo) FindByNameAndTenantID(name string, tenantID int64) (*RegistrationFlow, error) {
	if m.findByNameAndTenantIDFn != nil {
		return m.findByNameAndTenantIDFn(name, tenantID)
	}
	return nil, nil
}

type mockRegistrationFlowRoleRepo struct {
	mockBaseRepo[RegistrationFlowRole]
	findByRegistrationFlowIDFn            func(int64) ([]RegistrationFlowRole, error)
	findByRegistrationFlowIDPaginatedFn   func(int64, int, int) (*PaginationResult[RegistrationFlowRole], error)
	deleteByRegistrationFlowIDAndRoleIDFn func(int64, int64) error
	deleteByRegistrationFlowIDFn          func(int64) error
	findByRegistrationFlowIDAndRoleIDFn   func(int64, int64) (*RegistrationFlowRole, error)
	createFn                              func(*RegistrationFlowRole) (*RegistrationFlowRole, error)
}

func (m *mockRegistrationFlowRoleRepo) WithTx(_ *gorm.DB) RegistrationFlowRoleRepository { return m }
func (m *mockRegistrationFlowRoleRepo) Create(e *RegistrationFlowRole) (*RegistrationFlowRole, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockRegistrationFlowRoleRepo) FindByRegistrationFlowID(registrationFlowID int64) ([]RegistrationFlowRole, error) {
	if m.findByRegistrationFlowIDFn != nil {
		return m.findByRegistrationFlowIDFn(registrationFlowID)
	}
	return nil, nil
}
func (m *mockRegistrationFlowRoleRepo) FindByRegistrationFlowIDPaginated(registrationFlowID int64, page, limit int) (*PaginationResult[RegistrationFlowRole], error) {
	if m.findByRegistrationFlowIDPaginatedFn != nil {
		return m.findByRegistrationFlowIDPaginatedFn(registrationFlowID, page, limit)
	}
	return &PaginationResult[RegistrationFlowRole]{}, nil
}
func (m *mockRegistrationFlowRoleRepo) DeleteByRegistrationFlowIDAndRoleID(registrationFlowID, roleID int64) error {
	if m.deleteByRegistrationFlowIDAndRoleIDFn != nil {
		return m.deleteByRegistrationFlowIDAndRoleIDFn(registrationFlowID, roleID)
	}
	return nil
}
func (m *mockRegistrationFlowRoleRepo) DeleteByRegistrationFlowID(registrationFlowID int64) error {
	if m.deleteByRegistrationFlowIDFn != nil {
		return m.deleteByRegistrationFlowIDFn(registrationFlowID)
	}
	return nil
}
func (m *mockRegistrationFlowRoleRepo) FindByRegistrationFlowIDAndRoleID(registrationFlowID, roleID int64) (*RegistrationFlowRole, error) {
	if m.findByRegistrationFlowIDAndRoleIDFn != nil {
		return m.findByRegistrationFlowIDAndRoleIDFn(registrationFlowID, roleID)
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
	findRedirectURIsFn                  func(int64) ([]ClientURI, error)
}

func (m *mockClientRepo) WithTx(_ *gorm.DB) ClientRepository { return m }
func (m *mockClientRepo) FindRedirectURIs(clientID int64) ([]ClientURI, error) {
	if m.findRedirectURIsFn != nil {
		return m.findRedirectURIsFn(clientID)
	}
	return nil, nil
}
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

// mockUserRoleRepo backs the grantable-role cap: assertRolesGrantable asks it
// whether the acting user already possesses each role being attached to a flow.
type mockUserRoleRepo struct {
	mockBaseRepo[UserRole]
	findByUserIDFn            func(int64) ([]UserRole, error)
	findByUserIDAndRoleIDFn   func(int64, int64) (*UserRole, error)
	deleteByUserIDAndRoleIDFn func(int64, int64) error
}

func (m *mockUserRoleRepo) WithTx(_ *gorm.DB) UserRoleRepository { return m }
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
func (m *mockUserRoleRepo) DeleteByUserIDAndRoleID(userID, roleID int64) error {
	if m.deleteByUserIDAndRoleIDFn != nil {
		return m.deleteByUserIDAndRoleIDFn(userID, roleID)
	}
	return nil
}

// mockRegistrationFlowInviteCounter stands in for the invite domain's pending
// count so flow deletion can be tested without the invite tables.
type mockRegistrationFlowInviteCounter struct {
	countPendingByRegistrationFlowIDFn func(int64) (int64, error)
}

func (m *mockRegistrationFlowInviteCounter) WithTx(_ *gorm.DB) RegistrationFlowInviteCounter {
	return m
}
func (m *mockRegistrationFlowInviteCounter) CountPendingByRegistrationFlowID(registrationFlowID int64) (int64, error) {
	if m.countPendingByRegistrationFlowIDFn != nil {
		return m.countPendingByRegistrationFlowIDFn(registrationFlowID)
	}
	return 0, nil
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

// mockRegistrationFlowService implements RegistrationFlowService for handler
// tests. Every method guards its function field and returns a *safe* zero value:
// the data-returning methods hand back a non-nil empty result because the
// handler dereferences the result on the success path — a nil would panic
// instead of failing the assertion, which is how a real defect once hid here.
type mockRegistrationFlowService struct {
	getFn         func(RegistrationFlowServiceGetFilter) (*RegistrationFlowServiceListResult, error)
	getByUUIDFn   func(uuid.UUID, int64) (*RegistrationFlowServiceDataResult, error)
	createFn      func(RegistrationFlowCreateInput) (*RegistrationFlowServiceDataResult, error)
	updateFn      func(RegistrationFlowUpdateInput) (*RegistrationFlowServiceDataResult, error)
	setStatusFn   func(uuid.UUID, int64, uuid.UUID, string) (*RegistrationFlowServiceDataResult, error)
	deleteFn      func(uuid.UUID, int64, uuid.UUID) (*RegistrationFlowServiceDataResult, error)
	assignRolesFn func(uuid.UUID, int64, uuid.UUID, []uuid.UUID) ([]RegistrationFlowRoleServiceDataResult, error)
	getRolesFn    func(uuid.UUID, int64, int, int) (*RegistrationFlowRoleServiceListResult, error)
	removeRoleFn  func(uuid.UUID, int64, uuid.UUID, uuid.UUID) (*RegistrationFlowServiceDataResult, error)
}

func (m *mockRegistrationFlowService) Get(_ context.Context, filter RegistrationFlowServiceGetFilter) (*RegistrationFlowServiceListResult, error) {
	if m.getFn != nil {
		return m.getFn(filter)
	}
	return &RegistrationFlowServiceListResult{}, nil
}
func (m *mockRegistrationFlowService) GetByUUID(_ context.Context, id uuid.UUID, tenantID int64) (*RegistrationFlowServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tenantID)
	}
	return &RegistrationFlowServiceDataResult{}, nil
}
func (m *mockRegistrationFlowService) Create(_ context.Context, in RegistrationFlowCreateInput) (*RegistrationFlowServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(in)
	}
	return &RegistrationFlowServiceDataResult{}, nil
}
func (m *mockRegistrationFlowService) Update(_ context.Context, in RegistrationFlowUpdateInput) (*RegistrationFlowServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(in)
	}
	return &RegistrationFlowServiceDataResult{}, nil
}
func (m *mockRegistrationFlowService) SetStatus(_ context.Context, id uuid.UUID, tenantID int64, actorUserUUID uuid.UUID, status string) (*RegistrationFlowServiceDataResult, error) {
	if m.setStatusFn != nil {
		return m.setStatusFn(id, tenantID, actorUserUUID, status)
	}
	return &RegistrationFlowServiceDataResult{}, nil
}
func (m *mockRegistrationFlowService) Delete(_ context.Context, id uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(id, tenantID, actorUserUUID)
	}
	return &RegistrationFlowServiceDataResult{}, nil
}
func (m *mockRegistrationFlowService) AssignRoles(_ context.Context, id uuid.UUID, tenantID int64, actorUserUUID uuid.UUID, roles []uuid.UUID) ([]RegistrationFlowRoleServiceDataResult, error) {
	if m.assignRolesFn != nil {
		return m.assignRolesFn(id, tenantID, actorUserUUID, roles)
	}
	return []RegistrationFlowRoleServiceDataResult{}, nil
}
func (m *mockRegistrationFlowService) GetRoles(_ context.Context, id uuid.UUID, tenantID int64, page, limit int) (*RegistrationFlowRoleServiceListResult, error) {
	if m.getRolesFn != nil {
		return m.getRolesFn(id, tenantID, page, limit)
	}
	return &RegistrationFlowRoleServiceListResult{}, nil
}
func (m *mockRegistrationFlowService) RemoveRole(_ context.Context, id uuid.UUID, tenantID int64, actorUserUUID, roleUUID uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
	if m.removeRoleFn != nil {
		return m.removeRoleFn(id, tenantID, actorUserUUID, roleUUID)
	}
	return &RegistrationFlowServiceDataResult{}, nil
}

// mockRolePermissionNameReader stubs the role→permission-name lookup behind the
// grantable-role cap. The zero value reports no permissions, i.e. a purely
// self-service role, which is what most tests want.
type mockRolePermissionNameReader struct {
	findPermissionNamesByRoleIDFn func(int64) ([]string, error)
}

func (m *mockRolePermissionNameReader) WithTx(_ *gorm.DB) RolePermissionNameReader { return m }

func (m *mockRolePermissionNameReader) FindPermissionNamesByRoleID(roleID int64) ([]string, error) {
	if m.findPermissionNamesByRoleIDFn != nil {
		return m.findPermissionNamesByRoleIDFn(roleID)
	}
	return nil, nil
}
