package authn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
)

// mockAccountLinkService is a function-field mock of AccountLinkRequestService.
type mockAccountLinkService struct {
	confirmFn func(token string, authUserID, authTenantID int64) (*AccountLinkConfirmResult, error)
}

func (m *mockAccountLinkService) Initiate(_ context.Context, _ InitiateAccountLinkInput) (*AccountLinkRequest, error) {
	return &AccountLinkRequest{}, nil
}

func (m *mockAccountLinkService) Confirm(_ context.Context, token string, authUserID, authTenantID int64) (*AccountLinkConfirmResult, error) {
	if m.confirmFn != nil {
		return m.confirmFn(token, authUserID, authTenantID)
	}
	return &AccountLinkConfirmResult{UUID: uuid.NewString(), ProviderName: "google"}, nil
}

func (m *mockAccountLinkService) ExpireStale(_ context.Context) (int64, error) { return 0, nil }

func linkAuth(r *http.Request, userID, tenantID int64) *http.Request {
	return middleware.WithAuthContext(r, &authctx.AuthContext{
		User:   &authctx.AuthUser{UserID: userID},
		Tenant: &authctx.AuthTenant{TenantID: tenantID},
	})
}

func linkTokenParam(r *http.Request, token string) *http.Request {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		rctx = chi.NewRouteContext()
	}
	rctx.URLParams.Add("token", token)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestAccountLinkHandler_Confirm_NoAuth(t *testing.T) {
	h := NewAccountLinkHandler(&mockAccountLinkService{})
	w := httptest.NewRecorder()
	h.Confirm(w, httptest.NewRequest(http.MethodPost, "/account-link/tok/confirm", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAccountLinkHandler_Confirm_MissingToken(t *testing.T) {
	h := NewAccountLinkHandler(&mockAccountLinkService{})
	r := linkAuth(httptest.NewRequest(http.MethodPost, "/account-link//confirm", nil), 7, 1)
	r = linkTokenParam(r, "")
	w := httptest.NewRecorder()
	h.Confirm(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAccountLinkHandler_Confirm_NotFound(t *testing.T) {
	svc := &mockAccountLinkService{confirmFn: func(string, int64, int64) (*AccountLinkConfirmResult, error) {
		return nil, apperror.NewNotFoundWithReason("link request not found")
	}}
	h := NewAccountLinkHandler(svc)
	r := linkAuth(httptest.NewRequest(http.MethodPost, "/account-link/tok/confirm", nil), 7, 1)
	r = linkTokenParam(r, "tok")
	w := httptest.NewRecorder()
	h.Confirm(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAccountLinkHandler_Confirm_Conflict(t *testing.T) {
	svc := &mockAccountLinkService{confirmFn: func(string, int64, int64) (*AccountLinkConfirmResult, error) {
		return nil, apperror.NewConflict("link request has expired")
	}}
	h := NewAccountLinkHandler(svc)
	r := linkAuth(httptest.NewRequest(http.MethodPost, "/account-link/tok/confirm", nil), 7, 1)
	r = linkTokenParam(r, "tok")
	w := httptest.NewRecorder()
	h.Confirm(w, r)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAccountLinkHandler_Confirm_Forbidden(t *testing.T) {
	svc := &mockAccountLinkService{confirmFn: func(string, int64, int64) (*AccountLinkConfirmResult, error) {
		return nil, apperror.NewForbidden("you must be signed in to the account being linked")
	}}
	h := NewAccountLinkHandler(svc)
	r := linkAuth(httptest.NewRequest(http.MethodPost, "/account-link/tok/confirm", nil), 7, 1)
	r = linkTokenParam(r, "tok")
	w := httptest.NewRecorder()
	h.Confirm(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAccountLinkHandler_Confirm_Success(t *testing.T) {
	svc := &mockAccountLinkService{confirmFn: func(token string, authUserID, authTenantID int64) (*AccountLinkConfirmResult, error) {
		assert.Equal(t, "tok123", token)
		assert.Equal(t, int64(7), authUserID)
		assert.Equal(t, int64(1), authTenantID)
		return &AccountLinkConfirmResult{UUID: uuid.NewString(), ExistingUserID: 7, ProviderName: "google"}, nil
	}}
	h := NewAccountLinkHandler(svc)
	r := linkAuth(httptest.NewRequest(http.MethodPost, "/account-link/tok123/confirm", nil), 7, 1)
	r = linkTokenParam(r, "tok123")
	w := httptest.NewRecorder()
	h.Confirm(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "confirmed")
}
