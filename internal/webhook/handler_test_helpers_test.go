package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authctx"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/stretchr/testify/require"
)

const testTenantID int64 = 1

var (
	errNotFound      = apperror.NewNotFoundWithReason("not found")
	testTenantUUID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testResourceUUID = uuid.MustParse("00000000-0000-0000-0000-000000000099")
)

type mockWebhookEndpointService struct {
	getAllFn       func(int64, []string, int, int, string, string) (*WebhookEndpointServiceListResult, error)
	getByUUIDFn    func(int64, uuid.UUID) (*WebhookEndpointServiceDataResult, error)
	createFn       func(int64, string, bool, *int, *int, string, string) (*WebhookEndpointServiceDataResult, error)
	updateFn       func(int64, uuid.UUID, string, bool, bool, *int, *int, string, string) (*WebhookEndpointServiceDataResult, error)
	updateStatusFn func(int64, uuid.UUID, string) (*WebhookEndpointServiceDataResult, error)
	deleteFn       func(int64, uuid.UUID) (*WebhookEndpointServiceDataResult, error)
}

func (m *mockWebhookEndpointService) GetAll(_ context.Context, tid int64, status []string, page, limit int, sortBy, sortOrder string) (*WebhookEndpointServiceListResult, error) {
	if m.getAllFn != nil {
		return m.getAllFn(tid, status, page, limit, sortBy, sortOrder)
	}
	return &WebhookEndpointServiceListResult{}, nil
}

func (m *mockWebhookEndpointService) GetByUUID(_ context.Context, tid int64, id uuid.UUID) (*WebhookEndpointServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(tid, id)
	}
	return nil, nil
}

func (m *mockWebhookEndpointService) Create(_ context.Context, tid int64, url string, subscribeAll bool, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tid, url, subscribeAll, maxRetries, timeoutSeconds, description, status)
	}
	return nil, nil
}

func (m *mockWebhookEndpointService) Update(_ context.Context, tid int64, id uuid.UUID, url string, rotateSecret bool, subscribeAll bool, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(tid, id, url, rotateSecret, subscribeAll, maxRetries, timeoutSeconds, description, status)
	}
	return nil, nil
}

func (m *mockWebhookEndpointService) UpdateStatus(_ context.Context, tid int64, id uuid.UUID, status string) (*WebhookEndpointServiceDataResult, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(tid, id, status)
	}
	return nil, nil
}

func (m *mockWebhookEndpointService) Delete(_ context.Context, tid int64, id uuid.UUID) (*WebhookEndpointServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(tid, id)
	}
	return nil, nil
}

func withTenant(r *http.Request) *http.Request {
	tenant := &authctx.AuthTenant{TenantID: testTenantID, TenantUUID: testTenantUUID}
	return middleware.WithAuthContext(r, &authctx.AuthContext{Tenant: tenant})
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
