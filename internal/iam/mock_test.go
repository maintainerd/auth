package iam

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
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
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
	testTenantUUID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testUserUUID     = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testResourceUUID = uuid.MustParse("00000000-0000-0000-0000-000000000099")
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

func strPtr(v string) *string { return &v }

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

type mockAPIRepo struct {
	mockBaseRepo[API]
	findByUUIDFn              func(any, ...string) (*API, error)
	findByUUIDAndTenantIDFn   func(uuid.UUID, int64) (*API, error)
	findByNameFn              func(string, int64) (*API, error)
	findByIdentifierFn        func(string, int64) (*API, error)
	findPaginatedFn           func(APIRepositoryGetFilter) (*PaginationResult[API], error)
	setStatusByUUIDFn         func(uuid.UUID, int64, string) error
	countByServiceIDFn        func(int64, int64) (int64, error)
	deleteByUUIDAndTenantIDFn func(uuid.UUID, int64) error
	createOrUpdateFn          func(*API) (*API, error)
	deleteByUUIDFn            func(any) error
}

func (m *mockAPIRepo) WithTx(_ *gorm.DB) APIRepository { return m }
func (m *mockAPIRepo) FindByUUID(id any, p ...string) (*API, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockAPIRepo) CreateOrUpdate(e *API) (*API, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockAPIRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockAPIRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64) (*API, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tenantID)
	}
	return m.FindByUUID(id)
}
func (m *mockAPIRepo) FindByName(name string, tenantID int64) (*API, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name, tenantID)
	}
	return nil, nil
}
func (m *mockAPIRepo) FindByIdentifier(identifier string, tenantID int64) (*API, error) {
	if m.findByIdentifierFn != nil {
		return m.findByIdentifierFn(identifier, tenantID)
	}
	return nil, nil
}
func (m *mockAPIRepo) FindPaginated(f APIRepositoryGetFilter) (*PaginationResult[API], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[API]{}, nil
}
func (m *mockAPIRepo) SetStatusByUUID(id uuid.UUID, tenantID int64, status string) error {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tenantID, status)
	}
	return nil
}
func (m *mockAPIRepo) CountByServiceID(serviceID int64, tenantID int64) (int64, error) {
	if m.countByServiceIDFn != nil {
		return m.countByServiceIDFn(serviceID, tenantID)
	}
	return 0, nil
}
func (m *mockAPIRepo) DeleteByUUIDAndTenantID(id uuid.UUID, tenantID int64) error {
	if m.deleteByUUIDAndTenantIDFn != nil {
		return m.deleteByUUIDAndTenantIDFn(id, tenantID)
	}
	return nil
}

type mockPermissionRepo struct {
	mockBaseRepo[Permission]
	findByUUIDFn              func(any, ...string) (*Permission, error)
	findByUUIDsFn             func([]string, ...string) ([]Permission, error)
	findByUUIDAndTenantIDFn   func(uuid.UUID, int64) (*Permission, error)
	findByNameFn              func(string, int64) (*Permission, error)
	findPaginatedFn           func(PermissionRepositoryGetFilter) (*PaginationResult[Permission], error)
	deleteByUUIDAndTenantIDFn func(uuid.UUID, int64) error
	createOrUpdateFn          func(*Permission) (*Permission, error)
}

func (m *mockPermissionRepo) WithTx(_ *gorm.DB) PermissionRepository { return m }
func (m *mockPermissionRepo) FindByUUID(id any, p ...string) (*Permission, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockPermissionRepo) FindByUUIDs(ids []string, p ...string) ([]Permission, error) {
	if m.findByUUIDsFn != nil {
		return m.findByUUIDsFn(ids, p...)
	}
	return nil, nil
}
func (m *mockPermissionRepo) FindByUUIDsAndTenantID(uuids []string, tenantID int64) ([]Permission, error) {
	return m.FindByUUIDs(uuids)
}
func (m *mockPermissionRepo) CreateOrUpdate(e *Permission) (*Permission, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockPermissionRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64) (*Permission, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tenantID)
	}
	return nil, nil
}
func (m *mockPermissionRepo) FindByName(name string, tenantID int64) (*Permission, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name, tenantID)
	}
	return nil, nil
}
func (m *mockPermissionRepo) FindPaginated(f PermissionRepositoryGetFilter) (*PaginationResult[Permission], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[Permission]{}, nil
}
func (m *mockPermissionRepo) DeleteByUUIDAndTenantID(id uuid.UUID, tenantID int64) error {
	if m.deleteByUUIDAndTenantIDFn != nil {
		return m.deleteByUUIDAndTenantIDFn(id, tenantID)
	}
	return nil
}

type mockPolicyRepo struct {
	mockBaseRepo[Policy]
	findByUUIDAndTenantIDFn   func(uuid.UUID, int64) (*Policy, error)
	findByNameFn              func(string, int64) (*Policy, error)
	findByNameAndVersionFn    func(string, string, int64) (*Policy, error)
	findSystemPoliciesFn      func(int64) ([]Policy, error)
	findPaginatedFn           func(PolicyRepositoryGetFilter) (*PaginationResult[Policy], error)
	setStatusByUUIDFn         func(uuid.UUID, int64, string) error
	setSystemStatusByUUIDFn   func(uuid.UUID, int64, bool) error
	deleteByUUIDAndTenantIDFn func(uuid.UUID, int64) error
	deleteByUUIDAndTenantFn   func(uuid.UUID, int64) error
	createFn                  func(*Policy) (*Policy, error)
	createOrUpdateFn          func(*Policy) (*Policy, error)
	updateByUUIDFn            func(any, any) (*Policy, error)
}

func (m *mockPolicyRepo) WithTx(_ *gorm.DB) PolicyRepository { return m }
func (m *mockPolicyRepo) Create(e *Policy) (*Policy, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockPolicyRepo) CreateOrUpdate(e *Policy) (*Policy, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockPolicyRepo) UpdateByUUID(id, data any) (*Policy, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	if p, ok := data.(*Policy); ok {
		return p, nil
	}
	return nil, nil
}
func (m *mockPolicyRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64) (*Policy, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tenantID)
	}
	return nil, nil
}
func (m *mockPolicyRepo) FindByName(name string, tenantID int64) (*Policy, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name, tenantID)
	}
	return nil, nil
}
func (m *mockPolicyRepo) FindByNameAndVersion(name string, version string, tenantID int64) (*Policy, error) {
	if m.findByNameAndVersionFn != nil {
		return m.findByNameAndVersionFn(name, version, tenantID)
	}
	return nil, nil
}
func (m *mockPolicyRepo) FindSystemPolicies(tenantID int64) ([]Policy, error) {
	if m.findSystemPoliciesFn != nil {
		return m.findSystemPoliciesFn(tenantID)
	}
	return nil, nil
}
func (m *mockPolicyRepo) FindPaginated(f PolicyRepositoryGetFilter) (*PaginationResult[Policy], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[Policy]{}, nil
}
func (m *mockPolicyRepo) SetStatusByUUID(id uuid.UUID, tenantID int64, status string) error {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tenantID, status)
	}
	return nil
}
func (m *mockPolicyRepo) SetSystemStatusByUUID(id uuid.UUID, tenantID int64, isSystem bool) error {
	if m.setSystemStatusByUUIDFn != nil {
		return m.setSystemStatusByUUIDFn(id, tenantID, isSystem)
	}
	return nil
}
func (m *mockPolicyRepo) DeleteByUUIDAndTenantID(id uuid.UUID, tenantID int64) error {
	if m.deleteByUUIDAndTenantIDFn != nil {
		return m.deleteByUUIDAndTenantIDFn(id, tenantID)
	}
	if m.deleteByUUIDAndTenantFn != nil {
		return m.deleteByUUIDAndTenantFn(id, tenantID)
	}
	return nil
}

type mockRoleRepo struct {
	mockBaseRepo[Role]
	findByUUIDFn                 func(any, ...string) (*Role, error)
	findByNameAndTenantIDFn      func(string, int64) (*Role, error)
	findByUUIDAndTenantIDFn      func(uuid.UUID, int64) (*Role, error)
	findAllByTenantIDFn          func(int64) ([]Role, error)
	findPaginatedFn              func(RoleRepositoryGetFilter) (*PaginationResult[Role], error)
	getPermissionsByRoleUUIDFn   func(RoleRepositoryGetPermissionsFilter) (*PaginationResult[Permission], error)
	getPermsByRoleUUIDFn         func(RoleRepositoryGetPermissionsFilter) (*PaginationResult[Permission], error)
	setStatusByUUIDFn            func(uuid.UUID, string) error
	setDefaultStatusByUUIDFn     func(uuid.UUID, bool) error
	setSystemStatusByUUIDFn      func(uuid.UUID, bool) error
	findRegisteredRoleForSetupFn func(int64) (*Role, error)
	findSuperAdminRoleForSetupFn func(int64) (*Role, error)
	createOrUpdateFn             func(*Role) (*Role, error)
	deleteByUUIDFn               func(any) error
}

func (m *mockRoleRepo) WithTx(_ *gorm.DB) RoleRepository { return m }
func (m *mockRoleRepo) FindByUUID(id any, p ...string) (*Role, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockRoleRepo) CreateOrUpdate(e *Role) (*Role, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockRoleRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockRoleRepo) FindByNameAndTenantID(name string, tenantID int64) (*Role, error) {
	if m.findByNameAndTenantIDFn != nil {
		return m.findByNameAndTenantIDFn(name, tenantID)
	}
	return nil, nil
}
func (m *mockRoleRepo) FindByUUIDAndTenantID(roleUUID uuid.UUID, tenantID int64) (*Role, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(roleUUID, tenantID)
	}
	return m.FindByUUID(roleUUID)
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
func (m *mockRoleRepo) GetPermissionsByRoleUUID(f RoleRepositoryGetPermissionsFilter) (*PaginationResult[Permission], error) {
	if m.getPermissionsByRoleUUIDFn != nil {
		return m.getPermissionsByRoleUUIDFn(f)
	}
	if m.getPermsByRoleUUIDFn != nil {
		return m.getPermsByRoleUUIDFn(f)
	}
	return &PaginationResult[Permission]{}, nil
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

type mockRolePermissionRepo struct {
	mockBaseRepo[RolePermission]
	assignFn                    func(*RolePermission) (*RolePermission, error)
	createFn                    func(*RolePermission) (*RolePermission, error)
	findByRoleAndPermissionFn   func(int64, int64) (*RolePermission, error)
	findAllByRoleIDFn           func(int64) ([]RolePermission, error)
	findAllByPermissionIDFn     func(int64) ([]RolePermission, error)
	removeByRoleAndPermissionFn func(int64, int64) error
	setDefaultStatusByUUIDFn    func(uuid.UUID, bool) error
}

func (m *mockRolePermissionRepo) WithTx(_ *gorm.DB) RolePermissionRepository { return m }
func (m *mockRolePermissionRepo) Create(e *RolePermission) (*RolePermission, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockRolePermissionRepo) Assign(rp *RolePermission) (*RolePermission, error) {
	if m.assignFn != nil {
		return m.assignFn(rp)
	}
	if m.createFn != nil {
		return m.createFn(rp)
	}
	return rp, nil
}
func (m *mockRolePermissionRepo) FindByRoleAndPermission(roleID, permissionID int64) (*RolePermission, error) {
	if m.findByRoleAndPermissionFn != nil {
		return m.findByRoleAndPermissionFn(roleID, permissionID)
	}
	return nil, nil
}
func (m *mockRolePermissionRepo) FindAllByRoleID(roleID int64) ([]RolePermission, error) {
	if m.findAllByRoleIDFn != nil {
		return m.findAllByRoleIDFn(roleID)
	}
	return nil, nil
}
func (m *mockRolePermissionRepo) FindAllByPermissionID(permissionID int64) ([]RolePermission, error) {
	if m.findAllByPermissionIDFn != nil {
		return m.findAllByPermissionIDFn(permissionID)
	}
	return nil, nil
}
func (m *mockRolePermissionRepo) RemoveByRoleAndPermission(roleID, permissionID int64) error {
	if m.removeByRoleAndPermissionFn != nil {
		return m.removeByRoleAndPermissionFn(roleID, permissionID)
	}
	return nil
}
func (m *mockRolePermissionRepo) SetDefaultStatusByUUID(id uuid.UUID, isDefault bool) error {
	if m.setDefaultStatusByUUIDFn != nil {
		return m.setDefaultStatusByUUIDFn(id, isDefault)
	}
	return nil
}

type mockServiceRepo struct {
	mockBaseRepo[Service]
	findByUUIDFn               func(any, ...string) (*Service, error)
	findByNameFn               func(string) (*Service, error)
	findByNameAndTenantIDFn    func(string, int64) (*Service, error)
	findByTenantIDFn           func(int64) ([]Service, error)
	findPaginatedFn            func(ServiceRepositoryGetFilter) (*PaginationResult[Service], error)
	findServicesByPolicyUUIDFn func(uuid.UUID, ServiceRepositoryGetFilter) (*PaginationResult[Service], error)
	setStatusByUUIDFn          func(uuid.UUID, string) error
	countPoliciesByServiceIDFn func(int64) (int64, error)
	createOrUpdateFn           func(*Service) (*Service, error)
	deleteByUUIDFn             func(any) error
}

func (m *mockServiceRepo) WithTx(_ *gorm.DB) ServiceRepository { return m }
func (m *mockServiceRepo) FindByUUID(id any, p ...string) (*Service, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockServiceRepo) CreateOrUpdate(e *Service) (*Service, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockServiceRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockServiceRepo) FindByName(name string) (*Service, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name)
	}
	return nil, nil
}
func (m *mockServiceRepo) FindByNameAndTenantID(name string, tenantID int64) (*Service, error) {
	if m.findByNameAndTenantIDFn != nil {
		return m.findByNameAndTenantIDFn(name, tenantID)
	}
	return nil, nil
}
func (m *mockServiceRepo) FindByTenantID(tenantID int64) ([]Service, error) {
	if m.findByTenantIDFn != nil {
		return m.findByTenantIDFn(tenantID)
	}
	return nil, nil
}
func (m *mockServiceRepo) FindPaginated(f ServiceRepositoryGetFilter) (*PaginationResult[Service], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[Service]{}, nil
}
func (m *mockServiceRepo) FindServicesByPolicyUUID(id uuid.UUID, f ServiceRepositoryGetFilter) (*PaginationResult[Service], error) {
	if m.findServicesByPolicyUUIDFn != nil {
		return m.findServicesByPolicyUUIDFn(id, f)
	}
	return &PaginationResult[Service]{}, nil
}
func (m *mockServiceRepo) SetStatusByUUID(id uuid.UUID, status string) error {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, status)
	}
	return nil
}
func (m *mockServiceRepo) CountPoliciesByServiceID(serviceID int64) (int64, error) {
	if m.countPoliciesByServiceIDFn != nil {
		return m.countPoliciesByServiceIDFn(serviceID)
	}
	return 0, nil
}

type mockTenantServiceRepo struct {
	mockBaseRepo[TenantService]
	findPaginatedFn            func(TenantServiceRepositoryGetFilter) (*PaginationResult[TenantService], error)
	findByTenantAndServiceFn   func(int64, int64) (*TenantService, error)
	deleteByTenantAndServiceFn func(int64, int64) error
	createOrUpdateFn           func(*TenantService) (*TenantService, error)
}

func (m *mockTenantServiceRepo) WithTx(_ *gorm.DB) TenantServiceRepository { return m }
func (m *mockTenantServiceRepo) CreateOrUpdate(e *TenantService) (*TenantService, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockTenantServiceRepo) FindPaginated(f TenantServiceRepositoryGetFilter) (*PaginationResult[TenantService], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[TenantService]{}, nil
}
func (m *mockTenantServiceRepo) FindByTenantAndService(tenantID, serviceID int64) (*TenantService, error) {
	if m.findByTenantAndServiceFn != nil {
		return m.findByTenantAndServiceFn(tenantID, serviceID)
	}
	return &TenantService{TenantID: tenantID, ServiceID: serviceID}, nil
}
func (m *mockTenantServiceRepo) DeleteByTenantAndService(tenantID, serviceID int64) error {
	if m.deleteByTenantAndServiceFn != nil {
		return m.deleteByTenantAndServiceFn(tenantID, serviceID)
	}
	return nil
}

type mockServicePolicyRepo struct {
	mockBaseRepo[ServicePolicy]
	findPaginatedFn            func(ServicePolicyRepositoryGetFilter) (*PaginationResult[ServicePolicy], error)
	findByServiceAndPolicyFn   func(int64, int64) (*ServicePolicy, error)
	deleteByServiceAndPolicyFn func(int64, int64) error
	deleteByServiceAndPolicy   func(int64, int64) error
	findPoliciesByServiceIDFn  func(int64) ([]Policy, error)
	findServicesByPolicyIDFn   func(int64) ([]Service, error)
	createFn                   func(*ServicePolicy) (*ServicePolicy, error)
	createOrUpdateFn           func(*ServicePolicy) (*ServicePolicy, error)
}

func (m *mockServicePolicyRepo) WithTx(_ *gorm.DB) ServicePolicyRepository { return m }
func (m *mockServicePolicyRepo) Create(e *ServicePolicy) (*ServicePolicy, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockServicePolicyRepo) CreateOrUpdate(e *ServicePolicy) (*ServicePolicy, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockServicePolicyRepo) FindPaginated(f ServicePolicyRepositoryGetFilter) (*PaginationResult[ServicePolicy], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[ServicePolicy]{}, nil
}
func (m *mockServicePolicyRepo) FindByServiceAndPolicy(serviceID, policyID int64) (*ServicePolicy, error) {
	if m.findByServiceAndPolicyFn != nil {
		return m.findByServiceAndPolicyFn(serviceID, policyID)
	}
	return nil, nil
}
func (m *mockServicePolicyRepo) DeleteByServiceAndPolicy(serviceID, policyID int64) error {
	if m.deleteByServiceAndPolicyFn != nil {
		return m.deleteByServiceAndPolicyFn(serviceID, policyID)
	}
	if m.deleteByServiceAndPolicy != nil {
		return m.deleteByServiceAndPolicy(serviceID, policyID)
	}
	return nil
}
func (m *mockServicePolicyRepo) FindPoliciesByServiceID(serviceID int64) ([]Policy, error) {
	if m.findPoliciesByServiceIDFn != nil {
		return m.findPoliciesByServiceIDFn(serviceID)
	}
	return nil, nil
}
func (m *mockServicePolicyRepo) FindServicesByPolicyID(policyID int64) ([]Service, error) {
	if m.findServicesByPolicyIDFn != nil {
		return m.findServicesByPolicyIDFn(policyID)
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

type mockClientRepo struct {
	mockBaseRepo[Client]
	findByUUIDFn func(any, ...string) (*Client, error)
}

func (m *mockClientRepo) WithTx(_ *gorm.DB) ClientRepository { return m }
func (m *mockClientRepo) FindByUUID(id any, p ...string) (*Client, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}

type failingAuthorizationTokenInvalidator struct{}

func (failingAuthorizationTokenInvalidator) InvalidateRoleChange(context.Context, ...int64) error {
	return assert.AnError
}

func (failingAuthorizationTokenInvalidator) InvalidatePermissionChange(context.Context, int64) error {
	return assert.AnError
}

type mockAPIService struct {
	getFn                func(APIServiceGetFilter) (*APIServiceGetResult, error)
	getByUUIDFn          func(uuid.UUID, int64) (*APIServiceDataResult, error)
	getServiceIDByUUIDFn func(uuid.UUID) (int64, error)
	createFn             func(int64, string, string, string, string, bool, string) (*APIServiceDataResult, error)
	updateFn             func(uuid.UUID, int64, string, string, string, string, string) (*APIServiceDataResult, error)
	setStatusByUUIDFn    func(uuid.UUID, int64, string) (*APIServiceDataResult, error)
	deleteByUUIDFn       func(uuid.UUID, int64) (*APIServiceDataResult, error)
}

func (m *mockAPIService) Get(_ context.Context, f APIServiceGetFilter) (*APIServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &APIServiceGetResult{}, nil
}
func (m *mockAPIService) GetByUUID(_ context.Context, id uuid.UUID, tenantID int64) (*APIServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tenantID)
	}
	return &APIServiceDataResult{}, nil
}
func (m *mockAPIService) GetServiceIDByUUID(_ context.Context, id uuid.UUID, _ int64) (int64, error) {
	if m.getServiceIDByUUIDFn != nil {
		return m.getServiceIDByUUIDFn(id)
	}
	return 0, nil
}
func (m *mockAPIService) Create(_ context.Context, tenantID int64, name, displayName, description, status string, isSystem bool, serviceUUID string) (*APIServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tenantID, name, displayName, description, status, isSystem, serviceUUID)
	}
	return &APIServiceDataResult{}, nil
}
func (m *mockAPIService) Update(_ context.Context, id uuid.UUID, tenantID int64, name, displayName, description, status, serviceUUID string) (*APIServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tenantID, name, displayName, description, status, serviceUUID)
	}
	return &APIServiceDataResult{}, nil
}
func (m *mockAPIService) SetStatusByUUID(_ context.Context, id uuid.UUID, tenantID int64, status string) (*APIServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tenantID, status)
	}
	return &APIServiceDataResult{}, nil
}
func (m *mockAPIService) DeleteByUUID(_ context.Context, id uuid.UUID, tenantID int64) (*APIServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id, tenantID)
	}
	return &APIServiceDataResult{}, nil
}

type mockPermissionService struct {
	getFn                   func(PermissionServiceGetFilter) (*PermissionServiceGetResult, error)
	getByUUIDFn             func(uuid.UUID, int64) (*PermissionServiceDataResult, error)
	createFn                func(int64, string, string, string, bool, string) (*PermissionServiceDataResult, error)
	updateFn                func(uuid.UUID, int64, string, string, string) (*PermissionServiceDataResult, error)
	setStatusFn             func(uuid.UUID, int64, string) (*PermissionServiceDataResult, error)
	setActiveStatusByUUIDFn func(uuid.UUID, int64) (*PermissionServiceDataResult, error)
	deleteByUUIDFn          func(uuid.UUID, int64) (*PermissionServiceDataResult, error)
}

func (m *mockPermissionService) Get(_ context.Context, f PermissionServiceGetFilter) (*PermissionServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &PermissionServiceGetResult{}, nil
}
func (m *mockPermissionService) GetByUUID(_ context.Context, id uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tenantID)
	}
	return &PermissionServiceDataResult{}, nil
}
func (m *mockPermissionService) Create(_ context.Context, tenantID int64, name string, description string, status string, isSystem bool, apiUUID string) (*PermissionServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tenantID, name, description, status, isSystem, apiUUID)
	}
	return &PermissionServiceDataResult{}, nil
}
func (m *mockPermissionService) Update(_ context.Context, id uuid.UUID, tenantID int64, name string, description string, status string) (*PermissionServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tenantID, name, description, status)
	}
	return &PermissionServiceDataResult{}, nil
}
func (m *mockPermissionService) SetActiveStatusByUUID(_ context.Context, id uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error) {
	if m.setActiveStatusByUUIDFn != nil {
		return m.setActiveStatusByUUIDFn(id, tenantID)
	}
	return &PermissionServiceDataResult{}, nil
}
func (m *mockPermissionService) SetStatus(_ context.Context, id uuid.UUID, tenantID int64, status string) (*PermissionServiceDataResult, error) {
	if m.setStatusFn != nil {
		return m.setStatusFn(id, tenantID, status)
	}
	return &PermissionServiceDataResult{}, nil
}
func (m *mockPermissionService) DeleteByUUID(_ context.Context, id uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id, tenantID)
	}
	return &PermissionServiceDataResult{}, nil
}

type mockPolicyService struct {
	getFn                     func(PolicyServiceGetFilter) (*PolicyServiceGetResult, error)
	getByUUIDFn               func(uuid.UUID, int64) (*PolicyServiceDataResult, error)
	getServicesByPolicyUUIDFn func(uuid.UUID, int64, PolicyServiceServicesFilter) (*PolicyServiceServicesResult, error)
	createFn                  func(int64, string, *string, datatypes.JSON, string, string, bool) (*PolicyServiceDataResult, error)
	updateFn                  func(uuid.UUID, int64, string, *string, datatypes.JSON, string, string) (*PolicyServiceDataResult, error)
	setStatusByUUIDFn         func(uuid.UUID, int64, string) (*PolicyServiceDataResult, error)
	deleteByUUIDFn            func(uuid.UUID, int64) (*PolicyServiceDataResult, error)
}

func (m *mockPolicyService) Get(_ context.Context, f PolicyServiceGetFilter) (*PolicyServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &PolicyServiceGetResult{}, nil
}
func (m *mockPolicyService) GetByUUID(_ context.Context, id uuid.UUID, tenantID int64) (*PolicyServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tenantID)
	}
	return &PolicyServiceDataResult{}, nil
}
func (m *mockPolicyService) GetServicesByPolicyUUID(_ context.Context, id uuid.UUID, tenantID int64, f PolicyServiceServicesFilter) (*PolicyServiceServicesResult, error) {
	if m.getServicesByPolicyUUIDFn != nil {
		return m.getServicesByPolicyUUIDFn(id, tenantID, f)
	}
	return &PolicyServiceServicesResult{}, nil
}
func (m *mockPolicyService) Create(_ context.Context, tenantID int64, name string, description *string, document datatypes.JSON, version string, status string, isSystem bool) (*PolicyServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tenantID, name, description, document, version, status, isSystem)
	}
	return &PolicyServiceDataResult{}, nil
}
func (m *mockPolicyService) Update(_ context.Context, id uuid.UUID, tenantID int64, name string, description *string, document datatypes.JSON, version string, status string) (*PolicyServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tenantID, name, description, document, version, status)
	}
	return &PolicyServiceDataResult{}, nil
}
func (m *mockPolicyService) SetStatusByUUID(_ context.Context, id uuid.UUID, tenantID int64, status string) (*PolicyServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tenantID, status)
	}
	return &PolicyServiceDataResult{}, nil
}
func (m *mockPolicyService) DeleteByUUID(_ context.Context, id uuid.UUID, tenantID int64) (*PolicyServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id, tenantID)
	}
	return &PolicyServiceDataResult{}, nil
}

type mockRoleService struct {
	getFn                func(RoleServiceGetFilter) (*RoleServiceGetResult, error)
	getByUUIDFn          func(uuid.UUID, int64) (*RoleServiceDataResult, error)
	getRolePermissionsFn func(RoleServiceGetPermissionsFilter) (*RoleServiceGetPermissionsResult, error)
	createFn             func(string, string, bool, bool, string, string, uuid.UUID) (*RoleServiceDataResult, error)
	updateFn             func(uuid.UUID, int64, string, string, bool, bool, string, uuid.UUID) (*RoleServiceDataResult, error)
	setStatusByUUIDFn    func(uuid.UUID, int64, string, uuid.UUID) (*RoleServiceDataResult, error)
	deleteByUUIDFn       func(uuid.UUID, int64, uuid.UUID) (*RoleServiceDataResult, error)
	addRolePermsFn       func(uuid.UUID, int64, []uuid.UUID, uuid.UUID) (*RoleServiceDataResult, error)
	removeRolePermsFn    func(uuid.UUID, int64, uuid.UUID, uuid.UUID) (*RoleServiceDataResult, error)
}

func (m *mockRoleService) Get(_ context.Context, f RoleServiceGetFilter) (*RoleServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &RoleServiceGetResult{}, nil
}
func (m *mockRoleService) GetByUUID(_ context.Context, id uuid.UUID, tenantID int64) (*RoleServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tenantID)
	}
	return &RoleServiceDataResult{}, nil
}
func (m *mockRoleService) GetRolePermissions(_ context.Context, f RoleServiceGetPermissionsFilter) (*RoleServiceGetPermissionsResult, error) {
	if m.getRolePermissionsFn != nil {
		return m.getRolePermissionsFn(f)
	}
	return &RoleServiceGetPermissionsResult{}, nil
}
func (m *mockRoleService) Create(_ context.Context, name string, description string, isDefault bool, isSystem bool, status string, tenantUUID string, actor uuid.UUID) (*RoleServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(name, description, isDefault, isSystem, status, tenantUUID, actor)
	}
	return &RoleServiceDataResult{}, nil
}
func (m *mockRoleService) Update(_ context.Context, id uuid.UUID, tenantID int64, name string, description string, isDefault bool, isSystem bool, status string, actor uuid.UUID) (*RoleServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tenantID, name, description, isDefault, isSystem, status, actor)
	}
	return &RoleServiceDataResult{}, nil
}
func (m *mockRoleService) SetStatusByUUID(_ context.Context, id uuid.UUID, tenantID int64, status string, actor uuid.UUID) (*RoleServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tenantID, status, actor)
	}
	return &RoleServiceDataResult{}, nil
}
func (m *mockRoleService) DeleteByUUID(_ context.Context, id uuid.UUID, tenantID int64, actor uuid.UUID) (*RoleServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id, tenantID, actor)
	}
	return &RoleServiceDataResult{}, nil
}
func (m *mockRoleService) AddRolePermissions(_ context.Context, id uuid.UUID, tenantID int64, permissions []uuid.UUID, actor uuid.UUID) (*RoleServiceDataResult, error) {
	if m.addRolePermsFn != nil {
		return m.addRolePermsFn(id, tenantID, permissions, actor)
	}
	return &RoleServiceDataResult{}, nil
}
func (m *mockRoleService) RemoveRolePermissions(_ context.Context, id uuid.UUID, tenantID int64, permission uuid.UUID, actor uuid.UUID) (*RoleServiceDataResult, error) {
	if m.removeRolePermsFn != nil {
		return m.removeRolePermsFn(id, tenantID, permission, actor)
	}
	return &RoleServiceDataResult{}, nil
}

type mockServiceService struct {
	getFn             func(ServiceServiceGetFilter) (*ServiceServiceGetResult, error)
	getByUUIDFn       func(uuid.UUID, int64) (*ServiceServiceDataResult, error)
	createFn          func(string, string, string, string, bool, string, int64) (*ServiceServiceDataResult, error)
	updateFn          func(uuid.UUID, int64, string, string, string, string, bool, string) (*ServiceServiceDataResult, error)
	setStatusByUUIDFn func(uuid.UUID, int64, string) (*ServiceServiceDataResult, error)
	deleteByUUIDFn    func(uuid.UUID, int64) (*ServiceServiceDataResult, error)
	assignPolicyFn    func(uuid.UUID, uuid.UUID, int64) error
	removePolicyFn    func(uuid.UUID, uuid.UUID, int64) error
}

func (m *mockServiceService) Get(_ context.Context, f ServiceServiceGetFilter) (*ServiceServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &ServiceServiceGetResult{}, nil
}
func (m *mockServiceService) GetByUUID(_ context.Context, id uuid.UUID, tenantID int64) (*ServiceServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tenantID)
	}
	return &ServiceServiceDataResult{}, nil
}
func (m *mockServiceService) Create(_ context.Context, name string, displayName string, description string, version string, isSystem bool, status string, tenantID int64) (*ServiceServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(name, displayName, description, version, isSystem, status, tenantID)
	}
	return &ServiceServiceDataResult{}, nil
}
func (m *mockServiceService) Update(_ context.Context, id uuid.UUID, tenantID int64, name string, displayName string, description string, version string, isSystem bool, status string) (*ServiceServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tenantID, name, displayName, description, version, isSystem, status)
	}
	return &ServiceServiceDataResult{}, nil
}
func (m *mockServiceService) SetStatusByUUID(_ context.Context, id uuid.UUID, tenantID int64, status string) (*ServiceServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tenantID, status)
	}
	return &ServiceServiceDataResult{}, nil
}
func (m *mockServiceService) DeleteByUUID(_ context.Context, id uuid.UUID, tenantID int64) (*ServiceServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id, tenantID)
	}
	return &ServiceServiceDataResult{}, nil
}
func (m *mockServiceService) AssignPolicy(_ context.Context, serviceUUID uuid.UUID, policyUUID uuid.UUID, tenantID int64) error {
	if m.assignPolicyFn != nil {
		return m.assignPolicyFn(serviceUUID, policyUUID, tenantID)
	}
	return nil
}
func (m *mockServiceService) RemovePolicy(_ context.Context, serviceUUID uuid.UUID, policyUUID uuid.UUID, tenantID int64) error {
	if m.removePolicyFn != nil {
		return m.removePolicyFn(serviceUUID, policyUUID, tenantID)
	}
	return nil
}

type mockAuthorizationService struct {
	policyBundleFn func(ServiceIdentity) (*PolicyBundle, string, error)
	authorizeFn    func(AuthzRequest) Decision
}

func (m *mockAuthorizationService) PolicyBundle(_ context.Context, identity ServiceIdentity) (*PolicyBundle, string, error) {
	if m.policyBundleFn != nil {
		return m.policyBundleFn(identity)
	}
	return &PolicyBundle{Service: identity.ServiceName, Version: "v1"}, `"v1"`, nil
}

func (m *mockAuthorizationService) Authorize(_ context.Context, req AuthzRequest) Decision {
	if m.authorizeFn != nil {
		return m.authorizeFn(req)
	}
	return Decision{Allowed: true, Reason: "matched allow"}
}

var _ = datatypes.JSON{}
