package federation

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/require"
)

const testTenantID int64 = 1

var (
	errNotFound      = apperror.NewNotFoundWithReason("not found")
	testTenantUUID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testUserID       = int64(7)
	testResourceUUID = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	testClientUUID   = uuid.MustParse("00000000-0000-0000-0000-0000000000cc")
)

// mockWIFService is a function-field mock of WorkloadIdentityFederationService.
type mockWIFService struct {
	getAllFn   func(int64, int, int, string, string) (*WorkloadIdentityFederationServiceListResult, error)
	getFn      func(int64, uuid.UUID) (*WorkloadIdentityFederationServiceDataResult, error)
	createFn   func(int64, WorkloadIdentityFederationCreateInput) (*WorkloadIdentityFederationServiceDataResult, error)
	updateFn   func(int64, uuid.UUID, WorkloadIdentityFederationUpdateInput) (*WorkloadIdentityFederationServiceDataResult, error)
	deleteFn   func(int64, uuid.UUID) (*WorkloadIdentityFederationServiceDataResult, error)
	exchangeFn func(WorkloadExchangeInput) (*WorkloadExchangeResult, *apperror.OAuthError)
}

func (m *mockWIFService) GetAll(_ context.Context, tid int64, page, limit int, sortBy, sortOrder string) (*WorkloadIdentityFederationServiceListResult, error) {
	if m.getAllFn != nil {
		return m.getAllFn(tid, page, limit, sortBy, sortOrder)
	}
	return &WorkloadIdentityFederationServiceListResult{}, nil
}

func (m *mockWIFService) GetByUUID(_ context.Context, tid int64, id uuid.UUID) (*WorkloadIdentityFederationServiceDataResult, error) {
	if m.getFn != nil {
		return m.getFn(tid, id)
	}
	return nil, nil
}

func (m *mockWIFService) Create(_ context.Context, tid int64, in WorkloadIdentityFederationCreateInput) (*WorkloadIdentityFederationServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tid, in)
	}
	return nil, nil
}

func (m *mockWIFService) Update(_ context.Context, tid int64, id uuid.UUID, in WorkloadIdentityFederationUpdateInput) (*WorkloadIdentityFederationServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(tid, id, in)
	}
	return nil, nil
}

func (m *mockWIFService) Delete(_ context.Context, tid int64, id uuid.UUID) (*WorkloadIdentityFederationServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(tid, id)
	}
	return nil, nil
}

func (m *mockWIFService) ExchangeWorkloadToken(_ context.Context, in WorkloadExchangeInput) (*WorkloadExchangeResult, *apperror.OAuthError) {
	if m.exchangeFn != nil {
		return m.exchangeFn(in)
	}
	return nil, nil
}

func wifResult() *WorkloadIdentityFederationServiceDataResult {
	return &WorkloadIdentityFederationServiceDataResult{
		WorkloadIdentityFederationUUID: testResourceUUID,
		ClientUUID:                     testClientUUID,
		Name:                           "github-actions",
		IssuerURL:                      "https://token.actions.githubusercontent.com",
		Audience:                       "https://api.maintainerd.local",
		SubjectClaim:                   "sub",
		SubjectPattern:                 "repo:org/repo:*",
		AllowedScopes:                  []string{"deploy:write"},
		AttributeMapping:               map[string]string{"repository": "service_name"},
		IsActive:                       true,
	}
}

// withTenant attaches a tenant + user auth context to the request.
func withTenant(r *http.Request) *http.Request {
	tenant := &authctx.AuthTenant{TenantID: testTenantID, TenantUUID: testTenantUUID}
	user := &authctx.AuthUser{UserID: testUserID}
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

// validCreateBody returns a request body that passes DTO validation.
func validCreateBody() map[string]any {
	return map[string]any{
		"client_uuid":     testClientUUID.String(),
		"name":            "github-actions",
		"issuer_url":      "https://token.actions.githubusercontent.com",
		"audience":        "https://api.maintainerd.local",
		"subject_pattern": "repo:org/repo:*",
		"allowed_scopes":  []string{"deploy:write"},
	}
}

// validUpdateBody returns a request body that passes DTO validation.
func validUpdateBody() map[string]any {
	return map[string]any{
		"name":            "github-actions",
		"issuer_url":      "https://token.actions.githubusercontent.com",
		"audience":        "https://api.maintainerd.local",
		"subject_pattern": "repo:org/repo:*",
		"allowed_scopes":  []string{"deploy:write"},
	}
}

// base64URLPayload encodes a raw JWT payload segment (no padding) for tests.
func base64URLPayload(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}
