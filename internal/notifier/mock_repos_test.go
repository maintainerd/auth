package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authctx"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Shared test sentinels and helpers
// ---------------------------------------------------------------------------

const testTenantID int64 = 1

var (
	testTenantUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	errNotFound    = apperror.NewNotFoundWithReason("not found")
)

func withTenant(r *http.Request) *http.Request {
	tenant := &authctx.AuthTenant{TenantID: testTenantID, TenantUUID: testTenantUUID}
	return middleware.WithAuthContext(r, &authctx.AuthContext{Tenant: tenant})
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

// ---------------------------------------------------------------------------
// Mock: EmailConfigRepository
// ---------------------------------------------------------------------------

type mockEmailConfigRepo struct {
	findByTenantIDFn func(int64) (*EmailConfig, error)
	createOrUpdateFn func(*EmailConfig) (*EmailConfig, error)
}

func (m *mockEmailConfigRepo) WithTx(_ *gorm.DB) EmailConfigRepository { return m }
func (m *mockEmailConfigRepo) FindByTenantID(tenantID int64) (*EmailConfig, error) {
	if m.findByTenantIDFn != nil {
		return m.findByTenantIDFn(tenantID)
	}
	return nil, nil
}
func (m *mockEmailConfigRepo) Create(e *EmailConfig) (*EmailConfig, error) { return e, nil }
func (m *mockEmailConfigRepo) CreateOrUpdate(e *EmailConfig) (*EmailConfig, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockEmailConfigRepo) FindAll(p ...string) ([]EmailConfig, error)           { return nil, nil }
func (m *mockEmailConfigRepo) FindByUUID(id any, p ...string) (*EmailConfig, error) { return nil, nil }
func (m *mockEmailConfigRepo) FindByUUIDs(ids []string, p ...string) ([]EmailConfig, error) {
	return nil, nil
}
func (m *mockEmailConfigRepo) FindByID(id any, p ...string) (*EmailConfig, error) { return nil, nil }
func (m *mockEmailConfigRepo) UpdateByUUID(id, data any) (*EmailConfig, error)    { return nil, nil }
func (m *mockEmailConfigRepo) UpdateByID(id, data any) (*EmailConfig, error)      { return nil, nil }
func (m *mockEmailConfigRepo) DeleteByUUID(id any) error                          { return nil }
func (m *mockEmailConfigRepo) DeleteByID(id any) error                            { return nil }
func (m *mockEmailConfigRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[EmailConfig], error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: SMSConfigRepository
// ---------------------------------------------------------------------------

type mockSMSConfigRepo struct {
	findByTenantIDFn func(int64) (*SMSConfig, error)
	createOrUpdateFn func(*SMSConfig) (*SMSConfig, error)
}

func (m *mockSMSConfigRepo) WithTx(_ *gorm.DB) SMSConfigRepository { return m }
func (m *mockSMSConfigRepo) FindByTenantID(tenantID int64) (*SMSConfig, error) {
	if m.findByTenantIDFn != nil {
		return m.findByTenantIDFn(tenantID)
	}
	return nil, nil
}
func (m *mockSMSConfigRepo) Create(e *SMSConfig) (*SMSConfig, error) { return e, nil }
func (m *mockSMSConfigRepo) CreateOrUpdate(e *SMSConfig) (*SMSConfig, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockSMSConfigRepo) FindAll(p ...string) ([]SMSConfig, error)           { return nil, nil }
func (m *mockSMSConfigRepo) FindByUUID(id any, p ...string) (*SMSConfig, error) { return nil, nil }
func (m *mockSMSConfigRepo) FindByUUIDs(ids []string, p ...string) ([]SMSConfig, error) {
	return nil, nil
}
func (m *mockSMSConfigRepo) FindByID(id any, p ...string) (*SMSConfig, error) { return nil, nil }
func (m *mockSMSConfigRepo) UpdateByUUID(id, data any) (*SMSConfig, error)    { return nil, nil }
func (m *mockSMSConfigRepo) UpdateByID(id, data any) (*SMSConfig, error)      { return nil, nil }
func (m *mockSMSConfigRepo) DeleteByUUID(id any) error                        { return nil }
func (m *mockSMSConfigRepo) DeleteByID(id any) error                          { return nil }
func (m *mockSMSConfigRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[SMSConfig], error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: EmailConfigService
// ---------------------------------------------------------------------------

type mockEmailConfigService struct {
	getFn    func(int64) (*EmailConfigServiceDataResult, error)
	updateFn func(int64, string, string, int, string, string, string, string, string, string, string, *bool) (*EmailConfigServiceDataResult, error)
}

func (m *mockEmailConfigService) Get(ctx context.Context, tenantID int64) (*EmailConfigServiceDataResult, error) {
	if m.getFn != nil {
		return m.getFn(tenantID)
	}
	return nil, nil
}

func (m *mockEmailConfigService) Update(ctx context.Context, tenantID int64, provider, host string, port int, username, password, fromAddress, fromName, replyTo, encryption, logoURL string, testMode *bool) (*EmailConfigServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(tenantID, provider, host, port, username, password, fromAddress, fromName, replyTo, encryption, logoURL, testMode)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: SMSConfigService
// ---------------------------------------------------------------------------

type mockSMSConfigService struct {
	getFn    func(int64) (*SMSConfigServiceDataResult, error)
	updateFn func(int64, string, string, string, string, string, *int, *bool) (*SMSConfigServiceDataResult, error)
}

func (m *mockSMSConfigService) Get(ctx context.Context, tenantID int64) (*SMSConfigServiceDataResult, error) {
	if m.getFn != nil {
		return m.getFn(tenantID)
	}
	return nil, nil
}

func (m *mockSMSConfigService) Update(ctx context.Context, tenantID int64, provider, accountSID, authToken, fromNumber, senderID string, dailySendLimit *int, testMode *bool) (*SMSConfigServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(tenantID, provider, accountSID, authToken, fromNumber, senderID, nil, testMode)
	}
	return nil, nil
}
