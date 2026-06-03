package branding

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

const testTenantID int64 = 1

var (
	testTenantUUID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testResourceUUID = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	errNotFound      = apperror.NewNotFoundWithReason("not found")
	errValidation    = apperror.NewValidation("validation error")
)

func withTenant(r *http.Request) *http.Request {
	tenant := &authctx.AuthTenant{TenantID: testTenantID, TenantUUID: testTenantUUID}
	return middleware.WithAuthContext(r, &authctx.AuthContext{Tenant: tenant})
}

func withUser(r *http.Request) *http.Request {
	user := &authctx.AuthUser{UserUUID: testResourceUUID}
	return middleware.WithAuthContext(r, &authctx.AuthContext{User: user})
}

func withTenantAndUser(r *http.Request) *http.Request {
	tenant := &authctx.AuthTenant{TenantID: testTenantID, TenantUUID: testTenantUUID}
	user := &authctx.AuthUser{UserUUID: testResourceUUID}
	return middleware.WithAuthContext(r, &authctx.AuthContext{Tenant: tenant, User: user})
}

func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		rctx = chi.NewRouteContext()
	}
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
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

func badJSONReq(t *testing.T, method, url string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(method, url, strings.NewReader("{invalid"))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func validPagination() PaginationRequestDTO {
	return PaginationRequestDTO{Page: 1, Limit: 10, SortBy: "created_at", SortOrder: "asc"}
}

// ---------------------------------------------------------------------------
// Mock: BrandingService
// ---------------------------------------------------------------------------

type mockBrandingService struct {
	getFn    func(tenantID int64) (*BrandingServiceDataResult, error)
	updateFn func(tenantID int64, companyName, logoURL, faviconURL, primaryColor, secondaryColor, accentColor, fontFamily, customCSS, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error)
}

func (m *mockBrandingService) Get(ctx context.Context, tenantID int64) (*BrandingServiceDataResult, error) {
	if m.getFn != nil {
		return m.getFn(tenantID)
	}
	return nil, nil
}

func (m *mockBrandingService) Update(ctx context.Context, tenantID int64, companyName, logoURL, faviconURL, primaryColor, secondaryColor, accentColor, fontFamily, customCSS, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(tenantID, companyName, logoURL, faviconURL, primaryColor, secondaryColor, accentColor, fontFamily, customCSS, supportURL, privacyPolicyURL, termsOfServiceURL)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: EmailTemplateService
// ---------------------------------------------------------------------------

type mockEmailTemplateService struct {
	getAllFn       func(tid int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*EmailTemplateServiceListResult, error)
	getByUUIDFn    func(id uuid.UUID, tid int64) (*EmailTemplateServiceDataResult, error)
	createFn       func(tid int64, name, subject, bodyHTML string, bodyPlain *string, status string, isDefault bool) (*EmailTemplateServiceDataResult, error)
	updateFn       func(id uuid.UUID, tid int64, name, subject, bodyHTML string, bodyPlain *string, status string) (*EmailTemplateServiceDataResult, error)
	updateStatusFn func(id uuid.UUID, tid int64, status string) (*EmailTemplateServiceDataResult, error)
	deleteFn       func(id uuid.UUID, tid int64) (*EmailTemplateServiceDataResult, error)
}

func (m *mockEmailTemplateService) GetAll(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*EmailTemplateServiceListResult, error) {
	if m.getAllFn != nil {
		return m.getAllFn(tenantID, name, status, isDefault, isSystem, page, limit, sortBy, sortOrder)
	}
	return &EmailTemplateServiceListResult{}, nil
}

func (m *mockEmailTemplateService) GetByUUID(ctx context.Context, id uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tenantID)
	}
	return nil, nil
}

func (m *mockEmailTemplateService) Create(ctx context.Context, tenantID int64, name, subject, bodyHTML string, bodyPlain *string, status string, isDefault bool) (*EmailTemplateServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tenantID, name, subject, bodyHTML, bodyPlain, status, isDefault)
	}
	return nil, nil
}

func (m *mockEmailTemplateService) Update(ctx context.Context, id uuid.UUID, tenantID int64, name, subject, bodyHTML string, bodyPlain *string, status string) (*EmailTemplateServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tenantID, name, subject, bodyHTML, bodyPlain, status)
	}
	return nil, nil
}

func (m *mockEmailTemplateService) UpdateStatus(ctx context.Context, id uuid.UUID, tenantID int64, status string) (*EmailTemplateServiceDataResult, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(id, tenantID, status)
	}
	return nil, nil
}

func (m *mockEmailTemplateService) Delete(ctx context.Context, id uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(id, tenantID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: SMSTemplateService
// ---------------------------------------------------------------------------

type mockSMSTemplateService struct {
	getAllFn       func(tid int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*SMSTemplateServiceListResult, error)
	getByUUIDFn    func(id uuid.UUID, tid int64) (*SMSTemplateServiceDataResult, error)
	createFn       func(tid int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error)
	updateFn       func(id uuid.UUID, tid int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error)
	updateStatusFn func(id uuid.UUID, tid int64, status string) (*SMSTemplateServiceDataResult, error)
	deleteFn       func(id uuid.UUID, tid int64) (*SMSTemplateServiceDataResult, error)
}

func (m *mockSMSTemplateService) GetAll(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*SMSTemplateServiceListResult, error) {
	if m.getAllFn != nil {
		return m.getAllFn(tenantID, name, status, isDefault, isSystem, page, limit, sortBy, sortOrder)
	}
	return &SMSTemplateServiceListResult{}, nil
}

func (m *mockSMSTemplateService) GetByUUID(ctx context.Context, id uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tenantID)
	}
	return nil, nil
}

func (m *mockSMSTemplateService) Create(ctx context.Context, tenantID int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tenantID, name, description, message, senderID, status)
	}
	return nil, nil
}

func (m *mockSMSTemplateService) Update(ctx context.Context, id uuid.UUID, tenantID int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tenantID, name, description, message, senderID, status)
	}
	return nil, nil
}

func (m *mockSMSTemplateService) UpdateStatus(ctx context.Context, id uuid.UUID, tenantID int64, status string) (*SMSTemplateServiceDataResult, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(id, tenantID, status)
	}
	return nil, nil
}

func (m *mockSMSTemplateService) Delete(ctx context.Context, id uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(id, tenantID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: LoginTemplateService
// ---------------------------------------------------------------------------

type mockLoginTemplateService struct {
	getAllFn       func(tid int64, name *string, status []string, template *string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*LoginTemplateServiceListResult, error)
	getByUUIDFn    func(id uuid.UUID, tid int64) (*LoginTemplateServiceDataResult, error)
	createFn       func(tid int64, name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error)
	updateFn       func(id uuid.UUID, tid int64, name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error)
	updateStatusFn func(id uuid.UUID, tid int64, status string) (*LoginTemplateServiceDataResult, error)
	deleteFn       func(id uuid.UUID, tid int64) (*LoginTemplateServiceDataResult, error)
}

func (m *mockLoginTemplateService) GetAll(ctx context.Context, tenantID int64, name *string, status []string, template *string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*LoginTemplateServiceListResult, error) {
	if m.getAllFn != nil {
		return m.getAllFn(tenantID, name, status, template, isDefault, isSystem, page, limit, sortBy, sortOrder)
	}
	return &LoginTemplateServiceListResult{}, nil
}

func (m *mockLoginTemplateService) GetByUUID(ctx context.Context, id uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tenantID)
	}
	return nil, nil
}

func (m *mockLoginTemplateService) Create(ctx context.Context, tenantID int64, name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tenantID, name, description, template, metadata, status)
	}
	return nil, nil
}

func (m *mockLoginTemplateService) Update(ctx context.Context, id uuid.UUID, tenantID int64, name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tenantID, name, description, template, metadata, status)
	}
	return nil, nil
}

func (m *mockLoginTemplateService) UpdateStatus(ctx context.Context, id uuid.UUID, tenantID int64, status string) (*LoginTemplateServiceDataResult, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(id, tenantID, status)
	}
	return nil, nil
}

func (m *mockLoginTemplateService) Delete(ctx context.Context, id uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(id, tenantID)
	}
	return nil, nil
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
