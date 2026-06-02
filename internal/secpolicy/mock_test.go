package secpolicy

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

func withSecurityCtx(r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.ClientIPKey, "1.2.3.4")
	ctx = context.WithValue(ctx, middleware.UserAgentKey, "test-agent")
	return r.WithContext(ctx)
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

type mockIPRestrictionRuleRepo struct {
	mockBaseRepo[IPRestrictionRule]
	findByUUIDFn              func(any, ...string) (*IPRestrictionRule, error)
	findByTenantIDFn          func(int64) ([]IPRestrictionRule, error)
	findByTenantIDAndStatusFn func(int64, string) ([]IPRestrictionRule, error)
	findByTenantIDAndTypeFn   func(int64, string) ([]IPRestrictionRule, error)
	findPaginatedFn           func(IPRestrictionRuleRepositoryGetFilter) (*PaginationResult[IPRestrictionRule], error)
	createFn                  func(*IPRestrictionRule) (*IPRestrictionRule, error)
	updateByUUIDFn            func(any, any) (*IPRestrictionRule, error)
	deleteByUUIDFn            func(any) error
}

func (m *mockIPRestrictionRuleRepo) WithTx(_ *gorm.DB) IPRestrictionRuleRepository { return m }
func (m *mockIPRestrictionRuleRepo) FindByUUID(id any, p ...string) (*IPRestrictionRule, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockIPRestrictionRuleRepo) FindByTenantID(tid int64) ([]IPRestrictionRule, error) {
	if m.findByTenantIDFn != nil {
		return m.findByTenantIDFn(tid)
	}
	return nil, nil
}
func (m *mockIPRestrictionRuleRepo) FindByTenantIDAndStatus(tid int64, status string) ([]IPRestrictionRule, error) {
	if m.findByTenantIDAndStatusFn != nil {
		return m.findByTenantIDAndStatusFn(tid, status)
	}
	return nil, nil
}
func (m *mockIPRestrictionRuleRepo) FindByTenantIDAndType(tid int64, ruleType string) ([]IPRestrictionRule, error) {
	if m.findByTenantIDAndTypeFn != nil {
		return m.findByTenantIDAndTypeFn(tid, ruleType)
	}
	return nil, nil
}
func (m *mockIPRestrictionRuleRepo) FindPaginated(filter IPRestrictionRuleRepositoryGetFilter) (*PaginationResult[IPRestrictionRule], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(filter)
	}
	return &PaginationResult[IPRestrictionRule]{}, nil
}
func (m *mockIPRestrictionRuleRepo) Create(e *IPRestrictionRule) (*IPRestrictionRule, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockIPRestrictionRuleRepo) UpdateByUUID(id, data any) (*IPRestrictionRule, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	if rule, ok := data.(*IPRestrictionRule); ok {
		return rule, nil
	}
	return nil, nil
}
func (m *mockIPRestrictionRuleRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}

type mockSecuritySettingRepo struct {
	mockBaseRepo[SecuritySetting]
	findByUUIDFn            func(any, ...string) (*SecuritySetting, error)
	findByUserPoolIDFn      func(int64) (*SecuritySetting, error)
	findDefaultByTenantIDFn func(int64) (*SecuritySetting, error)
	findPaginatedFn         func(SecuritySettingRepositoryGetFilter) (*PaginationResult[SecuritySetting], error)
	createFn                func(*SecuritySetting) (*SecuritySetting, error)
	createOrUpdateFn        func(*SecuritySetting) (*SecuritySetting, error)
	updateByUUIDFn          func(any, any) (*SecuritySetting, error)
	incrementVersionFn      func(int64) error
}

func (m *mockSecuritySettingRepo) WithTx(_ *gorm.DB) SecuritySettingRepository { return m }
func (m *mockSecuritySettingRepo) FindByUUID(id any, p ...string) (*SecuritySetting, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockSecuritySettingRepo) FindByUserPoolID(userPoolID int64) (*SecuritySetting, error) {
	if m.findByUserPoolIDFn != nil {
		return m.findByUserPoolIDFn(userPoolID)
	}
	return nil, nil
}
func (m *mockSecuritySettingRepo) FindDefaultByTenantID(tenantID int64) (*SecuritySetting, error) {
	if m.findDefaultByTenantIDFn != nil {
		return m.findDefaultByTenantIDFn(tenantID)
	}
	return nil, nil
}
func (m *mockSecuritySettingRepo) FindPaginated(filter SecuritySettingRepositoryGetFilter) (*PaginationResult[SecuritySetting], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(filter)
	}
	return &PaginationResult[SecuritySetting]{}, nil
}
func (m *mockSecuritySettingRepo) Create(e *SecuritySetting) (*SecuritySetting, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockSecuritySettingRepo) CreateOrUpdate(e *SecuritySetting) (*SecuritySetting, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockSecuritySettingRepo) UpdateByUUID(id, data any) (*SecuritySetting, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	if setting, ok := data.(*SecuritySetting); ok {
		return setting, nil
	}
	return nil, nil
}
func (m *mockSecuritySettingRepo) IncrementVersion(securitySettingID int64) error {
	if m.incrementVersionFn != nil {
		return m.incrementVersionFn(securitySettingID)
	}
	return nil
}

type mockSecuritySettingsAuditRepo struct {
	mockBaseRepo[SecuritySettingsAudit]
	createFn                  func(*SecuritySettingsAudit) (*SecuritySettingsAudit, error)
	findBySecuritySettingIDFn func(int64) ([]SecuritySettingsAudit, error)
	findByUserPoolIDFn        func(int64) ([]SecuritySettingsAudit, error)
	findPaginatedFn           func(SecuritySettingsAuditRepositoryGetFilter) (*PaginationResult[SecuritySettingsAudit], error)
}

func (m *mockSecuritySettingsAuditRepo) WithTx(_ *gorm.DB) SecuritySettingsAuditRepository {
	return m
}
func (m *mockSecuritySettingsAuditRepo) Create(e *SecuritySettingsAudit) (*SecuritySettingsAudit, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockSecuritySettingsAuditRepo) FindBySecuritySettingID(securitySettingID int64) ([]SecuritySettingsAudit, error) {
	if m.findBySecuritySettingIDFn != nil {
		return m.findBySecuritySettingIDFn(securitySettingID)
	}
	return nil, nil
}
func (m *mockSecuritySettingsAuditRepo) FindByUserPoolID(userPoolID int64) ([]SecuritySettingsAudit, error) {
	if m.findByUserPoolIDFn != nil {
		return m.findByUserPoolIDFn(userPoolID)
	}
	return nil, nil
}
func (m *mockSecuritySettingsAuditRepo) FindPaginated(filter SecuritySettingsAuditRepositoryGetFilter) (*PaginationResult[SecuritySettingsAudit], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(filter)
	}
	return &PaginationResult[SecuritySettingsAudit]{}, nil
}

type mockSecuritySettingService struct {
	getByUserPoolIDFn          func(context.Context, int64) (*SecuritySettingServiceDataResult, error)
	getMFAConfigFn             func(int64) (map[string]any, error)
	getPasswordConfigFn        func(int64) (map[string]any, error)
	getSessionConfigFn         func(int64) (map[string]any, error)
	getThreatConfigFn          func(int64) (map[string]any, error)
	getLockoutConfigFn         func(int64) (map[string]any, error)
	getRegistrationConfigFn    func(int64) (map[string]any, error)
	getTokenConfigFn           func(int64) (map[string]any, error)
	updateMFAConfigFn          func(int64, map[string]any, int64, string, string) (*SecuritySettingServiceDataResult, error)
	updatePasswordConfigFn     func(int64, map[string]any, int64, string, string) (*SecuritySettingServiceDataResult, error)
	updateSessionConfigFn      func(int64, map[string]any, int64, string, string) (*SecuritySettingServiceDataResult, error)
	updateThreatConfigFn       func(int64, map[string]any, int64, string, string) (*SecuritySettingServiceDataResult, error)
	updateLockoutConfigFn      func(int64, map[string]any, int64, string, string) (*SecuritySettingServiceDataResult, error)
	updateRegistrationConfigFn func(int64, map[string]any, int64, string, string) (*SecuritySettingServiceDataResult, error)
	updateTokenConfigFn        func(int64, map[string]any, int64, string, string) (*SecuritySettingServiceDataResult, error)
}

func (m *mockSecuritySettingService) GetByUserPoolID(ctx context.Context, userPoolID int64) (*SecuritySettingServiceDataResult, error) {
	if m.getByUserPoolIDFn != nil {
		return m.getByUserPoolIDFn(ctx, userPoolID)
	}
	return &SecuritySettingServiceDataResult{}, nil
}
func (m *mockSecuritySettingService) GetMFAConfig(_ context.Context, userPoolID int64) (map[string]any, error) {
	if m.getMFAConfigFn != nil {
		return m.getMFAConfigFn(userPoolID)
	}
	return map[string]any{}, nil
}
func (m *mockSecuritySettingService) GetPasswordConfig(_ context.Context, userPoolID int64) (map[string]any, error) {
	if m.getPasswordConfigFn != nil {
		return m.getPasswordConfigFn(userPoolID)
	}
	return map[string]any{}, nil
}
func (m *mockSecuritySettingService) GetSessionConfig(_ context.Context, userPoolID int64) (map[string]any, error) {
	if m.getSessionConfigFn != nil {
		return m.getSessionConfigFn(userPoolID)
	}
	return map[string]any{}, nil
}
func (m *mockSecuritySettingService) GetThreatConfig(_ context.Context, userPoolID int64) (map[string]any, error) {
	if m.getThreatConfigFn != nil {
		return m.getThreatConfigFn(userPoolID)
	}
	return map[string]any{}, nil
}
func (m *mockSecuritySettingService) GetLockoutConfig(_ context.Context, userPoolID int64) (map[string]any, error) {
	if m.getLockoutConfigFn != nil {
		return m.getLockoutConfigFn(userPoolID)
	}
	return map[string]any{}, nil
}
func (m *mockSecuritySettingService) GetRegistrationConfig(_ context.Context, userPoolID int64) (map[string]any, error) {
	if m.getRegistrationConfigFn != nil {
		return m.getRegistrationConfigFn(userPoolID)
	}
	return map[string]any{}, nil
}
func (m *mockSecuritySettingService) GetTokenConfig(_ context.Context, userPoolID int64) (map[string]any, error) {
	if m.getTokenConfigFn != nil {
		return m.getTokenConfigFn(userPoolID)
	}
	return map[string]any{}, nil
}
func (m *mockSecuritySettingService) UpdateMFAConfig(_ context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	if m.updateMFAConfigFn != nil {
		return m.updateMFAConfigFn(userPoolID, config, updatedBy, ipAddress, userAgent)
	}
	return &SecuritySettingServiceDataResult{}, nil
}
func (m *mockSecuritySettingService) UpdatePasswordConfig(_ context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	if m.updatePasswordConfigFn != nil {
		return m.updatePasswordConfigFn(userPoolID, config, updatedBy, ipAddress, userAgent)
	}
	return &SecuritySettingServiceDataResult{}, nil
}
func (m *mockSecuritySettingService) UpdateSessionConfig(_ context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	if m.updateSessionConfigFn != nil {
		return m.updateSessionConfigFn(userPoolID, config, updatedBy, ipAddress, userAgent)
	}
	return &SecuritySettingServiceDataResult{}, nil
}
func (m *mockSecuritySettingService) UpdateThreatConfig(_ context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	if m.updateThreatConfigFn != nil {
		return m.updateThreatConfigFn(userPoolID, config, updatedBy, ipAddress, userAgent)
	}
	return &SecuritySettingServiceDataResult{}, nil
}
func (m *mockSecuritySettingService) UpdateLockoutConfig(_ context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	if m.updateLockoutConfigFn != nil {
		return m.updateLockoutConfigFn(userPoolID, config, updatedBy, ipAddress, userAgent)
	}
	return &SecuritySettingServiceDataResult{}, nil
}
func (m *mockSecuritySettingService) UpdateRegistrationConfig(_ context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	if m.updateRegistrationConfigFn != nil {
		return m.updateRegistrationConfigFn(userPoolID, config, updatedBy, ipAddress, userAgent)
	}
	return &SecuritySettingServiceDataResult{}, nil
}
func (m *mockSecuritySettingService) UpdateTokenConfig(_ context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	if m.updateTokenConfigFn != nil {
		return m.updateTokenConfigFn(userPoolID, config, updatedBy, ipAddress, userAgent)
	}
	return &SecuritySettingServiceDataResult{}, nil
}

type mockIPRestrictionRuleService struct {
	getAllFn       func(int64, *string, []string, *string, *string, int, int, string, string) (*IPRestrictionRuleServiceListResult, error)
	getByUUIDFn    func(int64, uuid.UUID) (*IPRestrictionRuleServiceDataResult, error)
	createFn       func(int64, string, string, string, string, int64) (*IPRestrictionRuleServiceDataResult, error)
	updateFn       func(int64, uuid.UUID, string, string, string, string, int64) (*IPRestrictionRuleServiceDataResult, error)
	updateStatusFn func(int64, uuid.UUID, string, int64) (*IPRestrictionRuleServiceDataResult, error)
	deleteFn       func(int64, uuid.UUID) (*IPRestrictionRuleServiceDataResult, error)
}

func (m *mockIPRestrictionRuleService) GetAll(_ context.Context, tenantID int64, ruleType *string, status []string, ipAddress, description *string, page, limit int, sortBy, sortOrder string) (*IPRestrictionRuleServiceListResult, error) {
	if m.getAllFn != nil {
		return m.getAllFn(tenantID, ruleType, status, ipAddress, description, page, limit, sortBy, sortOrder)
	}
	return &IPRestrictionRuleServiceListResult{}, nil
}
func (m *mockIPRestrictionRuleService) GetByUUID(_ context.Context, tenantID int64, ipRestrictionRuleUUID uuid.UUID) (*IPRestrictionRuleServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(tenantID, ipRestrictionRuleUUID)
	}
	return nil, nil
}
func (m *mockIPRestrictionRuleService) Create(_ context.Context, tenantID int64, description, ruleType, ipAddress, status string, createdBy int64) (*IPRestrictionRuleServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tenantID, description, ruleType, ipAddress, status, createdBy)
	}
	return &IPRestrictionRuleServiceDataResult{}, nil
}
func (m *mockIPRestrictionRuleService) Update(_ context.Context, tenantID int64, ipRestrictionRuleUUID uuid.UUID, description, ruleType, ipAddress, status string, updatedBy int64) (*IPRestrictionRuleServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(tenantID, ipRestrictionRuleUUID, description, ruleType, ipAddress, status, updatedBy)
	}
	return &IPRestrictionRuleServiceDataResult{}, nil
}
func (m *mockIPRestrictionRuleService) UpdateStatus(_ context.Context, tenantID int64, ipRestrictionRuleUUID uuid.UUID, status string, updatedBy int64) (*IPRestrictionRuleServiceDataResult, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(tenantID, ipRestrictionRuleUUID, status, updatedBy)
	}
	return &IPRestrictionRuleServiceDataResult{}, nil
}
func (m *mockIPRestrictionRuleService) Delete(_ context.Context, tenantID int64, ipRestrictionRuleUUID uuid.UUID) (*IPRestrictionRuleServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(tenantID, ipRestrictionRuleUUID)
	}
	return &IPRestrictionRuleServiceDataResult{}, nil
}
