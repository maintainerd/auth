package client

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
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	errNotFound = apperror.NewNotFoundWithReason("not found")
)

const tenantID int64 = 1

var (
	testTenantUUID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testUserUUID     = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testResourceUUID = uuid.MustParse("00000000-0000-0000-0000-000000000099")
)

func validPagination() PaginationRequestDTO {
	return PaginationRequestDTO{Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc}
}

func withTenant(r *http.Request) *http.Request {
	return middleware.WithAuthContext(r, &authctx.AuthContext{
		Tenant: &authctx.AuthTenant{TenantID: tenantID, TenantUUID: testTenantUUID},
	})
}

func withUser(r *http.Request) *http.Request {
	return middleware.WithAuthContext(r, &authctx.AuthContext{
		User: &authctx.AuthUser{UserUUID: testUserUUID},
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

type mockAPIKeyService struct {
	getFn                 func(APIKeyServiceGetFilter, uuid.UUID) (*APIKeyServiceGetResult, error)
	getByUUIDFn           func(uuid.UUID, int64, uuid.UUID) (*APIKeyServiceDataResult, error)
	getConfigByUUIDFn     func(uuid.UUID, int64) (datatypes.JSON, error)
	createFn              func(int64, string, string, datatypes.JSON, *time.Time, string) (*APIKeyServiceDataResult, string, error)
	updateFn              func(uuid.UUID, int64, *string, *string, datatypes.JSON, *time.Time, *string, uuid.UUID) (*APIKeyServiceDataResult, error)
	setStatusByUUIDFn     func(uuid.UUID, int64, string) (*APIKeyServiceDataResult, error)
	deleteFn              func(uuid.UUID, int64, uuid.UUID) (*APIKeyServiceDataResult, error)
	getAPIKeyAPIsFn       func(int64, uuid.UUID, int, int, string, string) (*APIKeyAPIServicePaginatedResult, error)
	addAPIKeyAPIsFn       func(int64, uuid.UUID, []uuid.UUID) error
	removeAPIKeyAPIFn     func(int64, uuid.UUID, uuid.UUID) error
	getAPIKeyAPIPermsFn   func(int64, uuid.UUID, uuid.UUID) ([]PermissionServiceDataResult, error)
	addAPIKeyAPIPermsFn   func(int64, uuid.UUID, uuid.UUID, []uuid.UUID) error
	removeAPIKeyAPIPermFn func(int64, uuid.UUID, uuid.UUID, uuid.UUID) error
}

func (m *mockAPIKeyService) Get(_ context.Context, f APIKeyServiceGetFilter, u uuid.UUID) (*APIKeyServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f, u)
	}
	return &APIKeyServiceGetResult{}, nil
}
func (m *mockAPIKeyService) GetByUUID(_ context.Context, id uuid.UUID, tid int64, u uuid.UUID) (*APIKeyServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tid, u)
	}
	return nil, nil
}
func (m *mockAPIKeyService) GetConfigByUUID(_ context.Context, id uuid.UUID, tid int64) (datatypes.JSON, error) {
	if m.getConfigByUUIDFn != nil {
		return m.getConfigByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockAPIKeyService) Create(_ context.Context, tid int64, n, desc string, cfg datatypes.JSON, exp *time.Time, s string) (*APIKeyServiceDataResult, string, error) {
	if m.createFn != nil {
		return m.createFn(tid, n, desc, cfg, exp, s)
	}
	return nil, "", nil
}
func (m *mockAPIKeyService) Update(_ context.Context, id uuid.UUID, tid int64, n, desc *string, cfg datatypes.JSON, exp *time.Time, s *string, u uuid.UUID) (*APIKeyServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tid, n, desc, cfg, exp, s, u)
	}
	return nil, nil
}
func (m *mockAPIKeyService) SetStatusByUUID(_ context.Context, id uuid.UUID, tid int64, s string) (*APIKeyServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tid, s)
	}
	return nil, nil
}
func (m *mockAPIKeyService) Delete(_ context.Context, id uuid.UUID, tid int64, u uuid.UUID) (*APIKeyServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(id, tid, u)
	}
	return nil, nil
}
func (m *mockAPIKeyService) GetAPIKeyAPIs(_ context.Context, tid int64, id uuid.UUID, pg, lim int, sb, so string) (*APIKeyAPIServicePaginatedResult, error) {
	if m.getAPIKeyAPIsFn != nil {
		return m.getAPIKeyAPIsFn(tid, id, pg, lim, sb, so)
	}
	return &APIKeyAPIServicePaginatedResult{}, nil
}
func (m *mockAPIKeyService) AddAPIKeyAPIs(_ context.Context, tid int64, id uuid.UUID, apis []uuid.UUID) error {
	if m.addAPIKeyAPIsFn != nil {
		return m.addAPIKeyAPIsFn(tid, id, apis)
	}
	return nil
}
func (m *mockAPIKeyService) RemoveAPIKeyAPI(_ context.Context, tid int64, id, api uuid.UUID) error {
	if m.removeAPIKeyAPIFn != nil {
		return m.removeAPIKeyAPIFn(tid, id, api)
	}
	return nil
}
func (m *mockAPIKeyService) GetAPIKeyAPIPermissions(_ context.Context, tid int64, id, api uuid.UUID) ([]PermissionServiceDataResult, error) {
	if m.getAPIKeyAPIPermsFn != nil {
		return m.getAPIKeyAPIPermsFn(tid, id, api)
	}
	return nil, nil
}
func (m *mockAPIKeyService) AddAPIKeyAPIPermissions(_ context.Context, tid int64, id, api uuid.UUID, perms []uuid.UUID) error {
	if m.addAPIKeyAPIPermsFn != nil {
		return m.addAPIKeyAPIPermsFn(tid, id, api, perms)
	}
	return nil
}
func (m *mockAPIKeyService) RemoveAPIKeyAPIPermission(_ context.Context, tid int64, id, api, perm uuid.UUID) error {
	if m.removeAPIKeyAPIPermFn != nil {
		return m.removeAPIKeyAPIPermFn(tid, id, api, perm)
	}
	return nil
}

type mockClientService struct {
	getFn                 func(ClientServiceGetFilter) (*ClientServiceGetResult, error)
	getByUUIDFn           func(uuid.UUID, int64) (*ClientServiceDataResult, error)
	getSecretByUUIDFn     func(uuid.UUID, int64) (*ClientSecretServiceDataResult, error)
	getConfigByUUIDFn     func(uuid.UUID, int64) (datatypes.JSON, error)
	createFn func(int64, string, string, string, string, datatypes.JSON, string, bool, string, *uuid.UUID, bool, *string, *string, *bool, *bool, uuid.UUID) (*ClientCreateServiceResult, error)
	updateFn func(uuid.UUID, int64, string, string, string, string, datatypes.JSON, string, bool, *uuid.UUID, *bool, *string, *string, *bool, *bool, uuid.UUID) (*ClientServiceDataResult, error)
	setStatusByUUIDFn     func(uuid.UUID, int64, string, uuid.UUID) (*ClientServiceDataResult, error)
	deleteByUUIDFn        func(uuid.UUID, int64, uuid.UUID) (*ClientServiceDataResult, error)
	createURIFn           func(uuid.UUID, int64, string, string, uuid.UUID) (*ClientServiceDataResult, error)
	updateURIFn           func(uuid.UUID, int64, uuid.UUID, string, string, uuid.UUID) (*ClientServiceDataResult, error)
	deleteURIFn           func(uuid.UUID, int64, uuid.UUID, uuid.UUID) (*ClientServiceDataResult, error)
	getConnectionsFn      func(uuid.UUID, int64) ([]ClientIdentityProviderServiceDataResult, error)
	addConnectionFn       func(uuid.UUID, int64, uuid.UUID, bool, bool, int, uuid.UUID) (*ClientServiceDataResult, error)
	updateConnectionFn    func(uuid.UUID, int64, uuid.UUID, bool, bool, int, uuid.UUID) (*ClientServiceDataResult, error)
	removeConnectionFn    func(uuid.UUID, int64, uuid.UUID, uuid.UUID) (*ClientServiceDataResult, error)
	getClientAPIsFn       func(int64, uuid.UUID) ([]ClientAPIServiceDataResult, error)
	addClientAPIsFn       func(int64, uuid.UUID, []uuid.UUID) error
	removeClientAPIFn     func(int64, uuid.UUID, uuid.UUID) error
	getClientAPIPermsFn   func(int64, uuid.UUID, uuid.UUID) ([]PermissionServiceDataResult, error)
	addClientAPIPermsFn   func(int64, uuid.UUID, uuid.UUID, []uuid.UUID) error
	removeClientAPIPermFn func(int64, uuid.UUID, uuid.UUID, uuid.UUID) error
	rotateSecretFn        func(uuid.UUID, int64, uuid.UUID, int) (string, error)
	isManagementClientFn  func(string) bool
}

func (m *mockClientService) IsManagementClient(_ context.Context, identifier string) bool {
	if m.isManagementClientFn != nil {
		return m.isManagementClientFn(identifier)
	}
	return false
}

func (m *mockClientService) Get(_ context.Context, f ClientServiceGetFilter) (*ClientServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &ClientServiceGetResult{}, nil
}
func (m *mockClientService) GetByUUID(_ context.Context, id uuid.UUID, tid int64) (*ClientServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockClientService) GetSecretByUUID(_ context.Context, id uuid.UUID, tid int64) (*ClientSecretServiceDataResult, error) {
	if m.getSecretByUUIDFn != nil {
		return m.getSecretByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockClientService) GetConfigByUUID(_ context.Context, id uuid.UUID, tid int64) (datatypes.JSON, error) {
	if m.getConfigByUUIDFn != nil {
		return m.getConfigByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockClientService) Create(_ context.Context, tid int64, n, dn, ct, d string, cfg datatypes.JSON, s string, isDef bool, idpUUID string, brandingUUID *uuid.UUID, allowRegistration bool, backchannelLogoutURI *string, frontchannelLogoutURI *string, backchannelLogoutSessionRequired *bool, dPoPRequired *bool, actor uuid.UUID) (*ClientCreateServiceResult, error) {
	if m.createFn != nil {
		return m.createFn(tid, n, dn, ct, d, cfg, s, isDef, idpUUID, brandingUUID, allowRegistration, backchannelLogoutURI, frontchannelLogoutURI, backchannelLogoutSessionRequired, dPoPRequired, actor)
	}
	return nil, nil
}
func (m *mockClientService) Update(_ context.Context, id uuid.UUID, tid int64, n, dn, ct, d string, cfg datatypes.JSON, s string, isDef bool, brandingUUID *uuid.UUID, allowRegistration *bool, backchannelLogoutURI *string, frontchannelLogoutURI *string, backchannelLogoutSessionRequired *bool, dPoPRequired *bool, actor uuid.UUID) (*ClientServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tid, n, dn, ct, d, cfg, s, isDef, brandingUUID, allowRegistration, backchannelLogoutURI, frontchannelLogoutURI, backchannelLogoutSessionRequired, dPoPRequired, actor)
	}
	return nil, nil
}
func (m *mockClientService) RotateSecret(_ context.Context, id uuid.UUID, tid int64, actor uuid.UUID, gracePeriodHours int) (string, error) {
	if m.rotateSecretFn != nil {
		return m.rotateSecretFn(id, tid, actor, gracePeriodHours)
	}
	return "", nil
}
func (m *mockClientService) SetStatusByUUID(_ context.Context, id uuid.UUID, tid int64, s string, actor uuid.UUID) (*ClientServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tid, s, actor)
	}
	return nil, nil
}
func (m *mockClientService) DeleteByUUID(_ context.Context, id uuid.UUID, tid int64, actor uuid.UUID) (*ClientServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id, tid, actor)
	}
	return nil, nil
}
func (m *mockClientService) CreateURI(_ context.Context, id uuid.UUID, tid int64, uri, uriType string, actor uuid.UUID) (*ClientServiceDataResult, error) {
	if m.createURIFn != nil {
		return m.createURIFn(id, tid, uri, uriType, actor)
	}
	return nil, nil
}
func (m *mockClientService) UpdateURI(_ context.Context, id uuid.UUID, tid int64, uriID uuid.UUID, uri, uriType string, actor uuid.UUID) (*ClientServiceDataResult, error) {
	if m.updateURIFn != nil {
		return m.updateURIFn(id, tid, uriID, uri, uriType, actor)
	}
	return nil, nil
}
func (m *mockClientService) DeleteURI(_ context.Context, id uuid.UUID, tid int64, uriID uuid.UUID, actor uuid.UUID) (*ClientServiceDataResult, error) {
	if m.deleteURIFn != nil {
		return m.deleteURIFn(id, tid, uriID, actor)
	}
	return nil, nil
}
func (m *mockClientService) GetConnections(_ context.Context, id uuid.UUID, tid int64) ([]ClientIdentityProviderServiceDataResult, error) {
	if m.getConnectionsFn != nil {
		return m.getConnectionsFn(id, tid)
	}
	return nil, nil
}
func (m *mockClientService) AddConnection(_ context.Context, id uuid.UUID, tid int64, idpID uuid.UUID, isDefault, enabled bool, displayOrder int, actor uuid.UUID) (*ClientServiceDataResult, error) {
	if m.addConnectionFn != nil {
		return m.addConnectionFn(id, tid, idpID, isDefault, enabled, displayOrder, actor)
	}
	return nil, nil
}
func (m *mockClientService) UpdateConnection(_ context.Context, id uuid.UUID, tid int64, connID uuid.UUID, isDefault, enabled bool, displayOrder int, actor uuid.UUID) (*ClientServiceDataResult, error) {
	if m.updateConnectionFn != nil {
		return m.updateConnectionFn(id, tid, connID, isDefault, enabled, displayOrder, actor)
	}
	return nil, nil
}
func (m *mockClientService) RemoveConnection(_ context.Context, id uuid.UUID, tid int64, connID uuid.UUID, actor uuid.UUID) (*ClientServiceDataResult, error) {
	if m.removeConnectionFn != nil {
		return m.removeConnectionFn(id, tid, connID, actor)
	}
	return nil, nil
}
func (m *mockClientService) GetClientAPIs(_ context.Context, tid int64, id uuid.UUID) ([]ClientAPIServiceDataResult, error) {
	if m.getClientAPIsFn != nil {
		return m.getClientAPIsFn(tid, id)
	}
	return nil, nil
}
func (m *mockClientService) AddClientAPIs(_ context.Context, tid int64, id uuid.UUID, apis []uuid.UUID) error {
	if m.addClientAPIsFn != nil {
		return m.addClientAPIsFn(tid, id, apis)
	}
	return nil
}
func (m *mockClientService) RemoveClientAPI(_ context.Context, tid int64, id, api uuid.UUID) error {
	if m.removeClientAPIFn != nil {
		return m.removeClientAPIFn(tid, id, api)
	}
	return nil
}
func (m *mockClientService) GetClientAPIPermissions(_ context.Context, tid int64, id, api uuid.UUID) ([]PermissionServiceDataResult, error) {
	if m.getClientAPIPermsFn != nil {
		return m.getClientAPIPermsFn(tid, id, api)
	}
	return nil, nil
}
func (m *mockClientService) AddClientAPIPermissions(_ context.Context, tid int64, id, api uuid.UUID, perms []uuid.UUID) error {
	if m.addClientAPIPermsFn != nil {
		return m.addClientAPIPermsFn(tid, id, api, perms)
	}
	return nil
}
func (m *mockClientService) RemoveClientAPIPermission(_ context.Context, tid int64, id, api, perm uuid.UUID) error {
	if m.removeClientAPIPermFn != nil {
		return m.removeClientAPIPermFn(tid, id, api, perm)
	}
	return nil
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

func expectClientIdentityProviderConnectionInsert(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE \(client_id = \$1 AND identity_provider_id = \$2 AND deleted_at IS NULL\).*LIMIT \$3`).
		WithArgs(int64(1), int64(1), 1).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectQuery(`INSERT INTO "client_identity_providers"`).
		WithArgs(sqlmock.AnyArg(), int64(1), int64(1), int64(1), true, true, 0, int64(1), int64(1), sqlmock.AnyArg(), sqlmock.AnyArg(), nil).
		WillReturnRows(sqlmock.NewRows([]string{"client_identity_provider_id"}).AddRow(int64(1)))
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

type mockAPIRepo struct {
	mockBaseRepo[API]
	findByUUIDFn func(any, ...string) (*API, error)
}

func (m *mockAPIRepo) WithTx(_ *gorm.DB) APIRepository { return m }
func (m *mockAPIRepo) FindByUUID(id any, p ...string) (*API, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockAPIRepo) FindByUUIDs(uuids []string, p ...string) ([]API, error) {
	results := make([]API, 0, len(uuids))
	for _, id := range uuids {
		uid, err := uuid.Parse(id)
		if err != nil {
			uid = uuid.Nil
		}
		a, err := m.FindByUUID(uid, p...)
		if err != nil {
			return nil, err
		}
		if a != nil {
			results = append(results, *a)
		}
	}
	return results, nil
}

type mockPermissionRepo struct {
	mockBaseRepo[Permission]
	findByUUIDFn func(any, ...string) (*Permission, error)
}

func (m *mockPermissionRepo) WithTx(_ *gorm.DB) PermissionRepository { return m }
func (m *mockPermissionRepo) FindByUUID(id any, p ...string) (*Permission, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockPermissionRepo) FindByUUIDs(uuids []string, p ...string) ([]Permission, error) {
	results := make([]Permission, 0, len(uuids))
	for _, id := range uuids {
		uid, err := uuid.Parse(id)
		if err != nil {
			uid = uuid.Nil
		}
		p, err := m.FindByUUID(uid, p...)
		if err != nil {
			return nil, err
		}
		if p != nil {
			results = append(results, *p)
		}
	}
	return results, nil
}

type mockIdentityProviderRepo struct {
	mockBaseRepo[IdentityProvider]
	findByUUIDFn func(any, ...string) (*IdentityProvider, error)
}

func (m *mockIdentityProviderRepo) WithTx(_ *gorm.DB) IdentityProviderRepository { return m }
func (m *mockIdentityProviderRepo) FindByUUID(id any, p ...string) (*IdentityProvider, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
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
	findByUUIDFn func(any, ...string) (*User, error)
}

func (m *mockUserRepo) WithTx(_ *gorm.DB) UserRepository { return m }
func (m *mockUserRepo) FindByUUID(id any, p ...string) (*User, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
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
	findPaginatedFn                     func(ClientRepositoryGetFilter) (*PaginationResult[Client], error)
	setStatusByUUIDFn                   func(uuid.UUID, int64, string) error
	findByClientIDAndIdentityProviderFn func(string, string) (*Client, error)
	findByIdentifierFn                  func(string) (*Client, error)
	findSystemByTenantIdentifierFn      func(string) (*Client, error)
	findSystemByTenantIdentifierNameFn  func(string, string) (*Client, error)
	deleteByUUIDAndTenantIDFn           func(uuid.UUID, int64) error
	createFn                            func(*Client) (*Client, error)
	createOrUpdateFn                    func(*Client) (*Client, error)
	updateByUUIDFn                      func(any, any) (*Client, error)
	deleteByUUIDFn                      func(any) error
}

func (m *mockClientRepo) WithTx(_ *gorm.DB) ClientRepository { return m }
func (m *mockClientRepo) Create(e *Client) (*Client, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	if e != nil && e.ClientID == 0 {
		e.ClientID = 1
	}
	return e, nil
}
func (m *mockClientRepo) CreateOrUpdate(e *Client) (*Client, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	if e != nil && e.ClientID == 0 {
		e.ClientID = 1
	}
	return e, nil
}
func (m *mockClientRepo) FindByUUID(id any, p ...string) (*Client, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockClientRepo) UpdateByUUID(id, data any) (*Client, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockClientRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockClientRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64) (*Client, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tenantID)
	}
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, "Tenant", "Branding", "ConnectedProviders.IdentityProvider", "ClientURIs")
	}
	return nil, nil
}
func (m *mockClientRepo) FindByNameAndIdentityProvider(name string, idpID int64, tenantID int64) (*Client, error) {
	if m.findByNameAndIdentityProviderFn != nil {
		return m.findByNameAndIdentityProviderFn(name, idpID, tenantID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByNameAndTenantID(name string, tenantID int64) (*Client, error) {
	if m.findByNameAndTenantIDFn != nil {
		return m.findByNameAndTenantIDFn(name, tenantID)
	}
	if m.findByNameAndIdentityProviderFn != nil {
		return m.findByNameAndIdentityProviderFn(name, 0, tenantID)
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
func (m *mockClientRepo) FindPaginated(f ClientRepositoryGetFilter) (*PaginationResult[Client], error) {
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
func (m *mockClientRepo) FindByClientIDAndIdentityProvider(clientID, idpIdentifier string) (*Client, error) {
	if m.findByClientIDAndIdentityProviderFn != nil {
		return m.findByClientIDAndIdentityProviderFn(clientID, idpIdentifier)
	}
	return nil, nil
}

func (m *mockClientRepo) FindByIdentifier(identifier string) (*Client, error) {
	if m.findByIdentifierFn != nil {
		return m.findByIdentifierFn(identifier)
	}
	return nil, nil
}

func (m *mockClientRepo) FindSystemByTenantIdentifier(tenantIdentifier string) (*Client, error) {
	if m.findSystemByTenantIdentifierFn != nil {
		return m.findSystemByTenantIdentifierFn(tenantIdentifier)
	}
	return nil, nil
}
func (m *mockClientRepo) FindSystemByTenantIdentifierAndName(tenantIdentifier, name string) (*Client, error) {
	if m.findSystemByTenantIdentifierNameFn != nil {
		return m.findSystemByTenantIdentifierNameFn(tenantIdentifier, name)
	}
	return m.FindSystemByTenantIdentifier(tenantIdentifier)
}

func (m *mockClientRepo) DeleteByUUIDAndTenantID(id uuid.UUID, tenantID int64) error {
	if m.deleteByUUIDAndTenantIDFn != nil {
		return m.deleteByUUIDAndTenantIDFn(id, tenantID)
	}
	return nil
}

type mockClientURIRepo struct {
	mockBaseRepo[ClientURI]
	findByUUIDAndTenantIDFn   func(string, int64) (*ClientURI, error)
	findByURIAndTypeFn        func(string, string, int64, int64) (*ClientURI, error)
	findByClientIDAndTypeFn   func(int64, string, int64) ([]ClientURI, error)
	deleteByUUIDAndTenantIDFn func(string, int64) error
	createFn                  func(*ClientURI) (*ClientURI, error)
	createOrUpdateFn          func(*ClientURI) (*ClientURI, error)
	updateByUUIDFn            func(any, any) (*ClientURI, error)
}

func (m *mockClientURIRepo) WithTx(_ *gorm.DB) ClientURIRepository { return m }
func (m *mockClientURIRepo) Create(e *ClientURI) (*ClientURI, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockClientURIRepo) CreateOrUpdate(e *ClientURI) (*ClientURI, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockClientURIRepo) UpdateByUUID(id, data any) (*ClientURI, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockClientURIRepo) FindByUUIDAndTenantID(id string, tenantID int64) (*ClientURI, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tenantID)
	}
	return nil, nil
}
func (m *mockClientURIRepo) FindByURIAndType(uri, uriType string, clientID, tenantID int64) (*ClientURI, error) {
	if m.findByURIAndTypeFn != nil {
		return m.findByURIAndTypeFn(uri, uriType, clientID, tenantID)
	}
	return nil, nil
}
func (m *mockClientURIRepo) FindByClientIDAndType(clientID int64, uriType string, tenantID int64) ([]ClientURI, error) {
	if m.findByClientIDAndTypeFn != nil {
		return m.findByClientIDAndTypeFn(clientID, uriType, tenantID)
	}
	return nil, nil
}
func (m *mockClientURIRepo) DeleteByUUIDAndTenantID(id string, tenantID int64) error {
	if m.deleteByUUIDAndTenantIDFn != nil {
		return m.deleteByUUIDAndTenantIDFn(id, tenantID)
	}
	return nil
}

type mockClientAPIRepo struct {
	mockBaseRepo[ClientAPI]
	findByClientAndAPIFn           func(int64, int64) (*ClientAPI, error)
	findByClientUUIDFn             func(uuid.UUID) ([]ClientAPI, error)
	findByClientUUIDAndAPIUUIDFn   func(uuid.UUID, uuid.UUID) (*ClientAPI, error)
	removeByClientAndAPIFn         func(int64, int64) error
	removeByClientUUIDAndAPIUUIDFn func(uuid.UUID, uuid.UUID) error
	createFn                       func(*ClientAPI) (*ClientAPI, error)
}

func (m *mockClientAPIRepo) WithTx(_ *gorm.DB) ClientAPIRepository { return m }
func (m *mockClientAPIRepo) Create(e *ClientAPI) (*ClientAPI, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockClientAPIRepo) FindByClientAndAPI(clientID, apiID int64) (*ClientAPI, error) {
	if m.findByClientAndAPIFn != nil {
		return m.findByClientAndAPIFn(clientID, apiID)
	}
	return nil, nil
}
func (m *mockClientAPIRepo) FindByClientUUID(clientUUID uuid.UUID) ([]ClientAPI, error) {
	if m.findByClientUUIDFn != nil {
		return m.findByClientUUIDFn(clientUUID)
	}
	return nil, nil
}
func (m *mockClientAPIRepo) FindByClientUUIDAndAPIUUID(clientUUID, apiUUID uuid.UUID) (*ClientAPI, error) {
	if m.findByClientUUIDAndAPIUUIDFn != nil {
		return m.findByClientUUIDAndAPIUUIDFn(clientUUID, apiUUID)
	}
	return nil, nil
}
func (m *mockClientAPIRepo) RemoveByClientAndAPI(clientID, apiID int64) error {
	if m.removeByClientAndAPIFn != nil {
		return m.removeByClientAndAPIFn(clientID, apiID)
	}
	return nil
}
func (m *mockClientAPIRepo) RemoveByClientUUIDAndAPIUUID(clientUUID, apiUUID uuid.UUID) error {
	if m.removeByClientUUIDAndAPIUUIDFn != nil {
		return m.removeByClientUUIDAndAPIUUIDFn(clientUUID, apiUUID)
	}
	return nil
}

type mockClientPermissionRepo struct {
	mockBaseRepo[ClientPermission]
	findByClientAPIAndPermissionFn   func(int64, int64) (*ClientPermission, error)
	removeByClientAPIAndPermissionFn func(int64, int64) error
	findByClientAPIIDFn              func(int64) ([]ClientPermission, error)
	createFn                         func(*ClientPermission) (*ClientPermission, error)
}

func (m *mockClientPermissionRepo) WithTx(_ *gorm.DB) ClientPermissionRepository { return m }
func (m *mockClientPermissionRepo) Create(e *ClientPermission) (*ClientPermission, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockClientPermissionRepo) FindByClientAPIAndPermission(clientAPIID, permissionID int64) (*ClientPermission, error) {
	if m.findByClientAPIAndPermissionFn != nil {
		return m.findByClientAPIAndPermissionFn(clientAPIID, permissionID)
	}
	return nil, nil
}
func (m *mockClientPermissionRepo) RemoveByClientAPIAndPermission(clientAPIID, permissionID int64) error {
	if m.removeByClientAPIAndPermissionFn != nil {
		return m.removeByClientAPIAndPermissionFn(clientAPIID, permissionID)
	}
	return nil
}
func (m *mockClientPermissionRepo) FindByClientAPIID(clientAPIID int64) ([]ClientPermission, error) {
	if m.findByClientAPIIDFn != nil {
		return m.findByClientAPIIDFn(clientAPIID)
	}
	return nil, nil
}

type mockAPIKeyRepo struct {
	mockBaseRepo[APIKey]
	findByUUIDFn              func(any, ...string) (*APIKey, error)
	findByUUIDAndTenantIDFn   func(string, int64) (*APIKey, error)
	findByKeyHashFn           func(string) (*APIKey, error)
	findByKeyPrefixFn         func(string) (*APIKey, error)
	deleteByUUIDAndTenantIDFn func(string, int64) error
	findPaginatedFn           func(APIKeyRepositoryGetFilter) (*PaginationResult[APIKey], error)
	createFn                  func(*APIKey) (*APIKey, error)
	updateByUUIDFn            func(any, any) (*APIKey, error)
	deleteByUUIDFn            func(any) error
}

func (m *mockAPIKeyRepo) WithTx(_ *gorm.DB) APIKeyRepository { return m }
func (m *mockAPIKeyRepo) Create(e *APIKey) (*APIKey, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockAPIKeyRepo) FindByUUID(id any, p ...string) (*APIKey, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockAPIKeyRepo) UpdateByUUID(id, data any) (*APIKey, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockAPIKeyRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockAPIKeyRepo) FindByUUIDAndTenantID(id string, tenantID int64) (*APIKey, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tenantID)
	}
	return nil, nil
}
func (m *mockAPIKeyRepo) FindByKeyHash(keyHash string) (*APIKey, error) {
	if m.findByKeyHashFn != nil {
		return m.findByKeyHashFn(keyHash)
	}
	return nil, nil
}
func (m *mockAPIKeyRepo) FindByKeyPrefix(keyPrefix string) (*APIKey, error) {
	if m.findByKeyPrefixFn != nil {
		return m.findByKeyPrefixFn(keyPrefix)
	}
	return nil, nil
}
func (m *mockAPIKeyRepo) DeleteByUUIDAndTenantID(id string, tenantID int64) error {
	if m.deleteByUUIDAndTenantIDFn != nil {
		return m.deleteByUUIDAndTenantIDFn(id, tenantID)
	}
	return nil
}
func (m *mockAPIKeyRepo) FindPaginated(f APIKeyRepositoryGetFilter) (*PaginationResult[APIKey], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[APIKey]{}, nil
}

type mockAPIKeyAPIRepo struct {
	mockBaseRepo[APIKeyAPI]
	findByAPIKeyAndAPIFn           func(int64, int64) (*APIKeyAPI, error)
	findByAPIKeyUUIDFn             func(uuid.UUID) ([]APIKeyAPI, error)
	findByAPIKeyUUIDPaginatedFn    func(uuid.UUID, int, int, string, string) (*PaginationResult[APIKeyAPI], error)
	findByAPIKeyUUIDAndAPIUUIDFn   func(uuid.UUID, uuid.UUID) (*APIKeyAPI, error)
	removeByAPIKeyAndAPIFn         func(int64, int64) error
	removeByAPIKeyUUIDAndAPIUUIDFn func(uuid.UUID, uuid.UUID) error
	createFn                       func(*APIKeyAPI) (*APIKeyAPI, error)
}

func (m *mockAPIKeyAPIRepo) WithTx(_ *gorm.DB) APIKeyAPIRepository { return m }
func (m *mockAPIKeyAPIRepo) Create(e *APIKeyAPI) (*APIKeyAPI, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockAPIKeyAPIRepo) FindByAPIKeyAndAPI(apiKeyID, apiID int64) (*APIKeyAPI, error) {
	if m.findByAPIKeyAndAPIFn != nil {
		return m.findByAPIKeyAndAPIFn(apiKeyID, apiID)
	}
	return nil, nil
}
func (m *mockAPIKeyAPIRepo) FindByAPIKeyUUID(apiKeyUUID uuid.UUID) ([]APIKeyAPI, error) {
	if m.findByAPIKeyUUIDFn != nil {
		return m.findByAPIKeyUUIDFn(apiKeyUUID)
	}
	return nil, nil
}
func (m *mockAPIKeyAPIRepo) FindByAPIKeyUUIDPaginated(apiKeyUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*PaginationResult[APIKeyAPI], error) {
	if m.findByAPIKeyUUIDPaginatedFn != nil {
		return m.findByAPIKeyUUIDPaginatedFn(apiKeyUUID, page, limit, sortBy, sortOrder)
	}
	return &PaginationResult[APIKeyAPI]{}, nil
}
func (m *mockAPIKeyAPIRepo) FindByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID uuid.UUID) (*APIKeyAPI, error) {
	if m.findByAPIKeyUUIDAndAPIUUIDFn != nil {
		return m.findByAPIKeyUUIDAndAPIUUIDFn(apiKeyUUID, apiUUID)
	}
	return nil, nil
}
func (m *mockAPIKeyAPIRepo) RemoveByAPIKeyAndAPI(apiKeyID, apiID int64) error {
	if m.removeByAPIKeyAndAPIFn != nil {
		return m.removeByAPIKeyAndAPIFn(apiKeyID, apiID)
	}
	return nil
}
func (m *mockAPIKeyAPIRepo) RemoveByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID uuid.UUID) error {
	if m.removeByAPIKeyUUIDAndAPIUUIDFn != nil {
		return m.removeByAPIKeyUUIDAndAPIUUIDFn(apiKeyUUID, apiUUID)
	}
	return nil
}

type mockAPIKeyPermissionRepo struct {
	mockBaseRepo[APIKeyPermission]
	findByAPIKeyAPIAndPermissionFn   func(int64, int64) (*APIKeyPermission, error)
	removeByAPIKeyAPIAndPermissionFn func(int64, int64) error
	findByAPIKeyAPIIDFn              func(int64) ([]APIKeyPermission, error)
	createFn                         func(*APIKeyPermission) (*APIKeyPermission, error)
}

func (m *mockAPIKeyPermissionRepo) WithTx(_ *gorm.DB) APIKeyPermissionRepository { return m }
func (m *mockAPIKeyPermissionRepo) Create(e *APIKeyPermission) (*APIKeyPermission, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockAPIKeyPermissionRepo) FindByAPIKeyAPIAndPermission(apiKeyAPIID, permissionID int64) (*APIKeyPermission, error) {
	if m.findByAPIKeyAPIAndPermissionFn != nil {
		return m.findByAPIKeyAPIAndPermissionFn(apiKeyAPIID, permissionID)
	}
	return nil, nil
}
func (m *mockAPIKeyPermissionRepo) RemoveByAPIKeyAPIAndPermission(apiKeyAPIID, permissionID int64) error {
	if m.removeByAPIKeyAPIAndPermissionFn != nil {
		return m.removeByAPIKeyAPIAndPermissionFn(apiKeyAPIID, permissionID)
	}
	return nil
}
func (m *mockAPIKeyPermissionRepo) FindByAPIKeyAPIID(apiKeyAPIID int64) ([]APIKeyPermission, error) {
	if m.findByAPIKeyAPIIDFn != nil {
		return m.findByAPIKeyAPIIDFn(apiKeyAPIID)
	}
	return nil, nil
}

// buildConnSvc builds a ClientService for the identity-provider-connection tests.
// The connection repo is constructed internally over gormDB, so connection
// operations are driven through the supplied sqlmock.
func buildConnSvc(gormDB *gorm.DB, clientRepo *mockClientRepo, idpRepo *mockIdentityProviderRepo, userRepo *mockUserRepo) ClientService {
	return NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, idpRepo,
		&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{},
		&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
}
