package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockDataErasureService is a function-field mock of DataErasureService.
type mockDataErasureService struct {
	requestFn func(RequestErasureInput) (*DataErasureRequestResult, error)
	processFn func() error
}

func (m *mockDataErasureService) RequestErasure(_ context.Context, in RequestErasureInput) (*DataErasureRequestResult, error) {
	if m.requestFn != nil {
		return m.requestFn(in)
	}
	return &DataErasureRequestResult{UUID: testResourceUUID, Status: "pending", ScheduledAt: time.Now(), CreatedAt: time.Now()}, nil
}

func (m *mockDataErasureService) ProcessPendingErasureRequests(_ context.Context) error {
	if m.processFn != nil {
		return m.processFn()
	}
	return nil
}

func erasureResult() *DataErasureRequestResult {
	return &DataErasureRequestResult{
		UUID:        testResourceUUID,
		Status:      "pending",
		Reason:      "user request",
		ScheduledAt: time.Now().Add(30 * 24 * time.Hour),
		CreatedAt:   time.Now(),
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Admin endpoint — POST /users/{user_uuid}/erasure-requests
// ──────────────────────────────────────────────────────────────────────────────

func TestDataErasureHandler_RequestAdmin_NoTenant(t *testing.T) {
	h := NewDataErasureHandler(&mockDataErasureService{}, &mockUserRepo{})
	w := httptest.NewRecorder()
	h.RequestAdmin(w, httptest.NewRequest(http.MethodPost, "/users/abc/erasure-requests", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDataErasureHandler_RequestAdmin_InvalidUUID(t *testing.T) {
	h := NewDataErasureHandler(&mockDataErasureService{}, &mockUserRepo{})
	r := withTenantAndActor(httptest.NewRequest(http.MethodPost, "/users/bad/erasure-requests", nil), 5)
	r = withChiParam(r, "user_uuid", "not-a-uuid")
	w := httptest.NewRecorder()
	h.RequestAdmin(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDataErasureHandler_RequestAdmin_UserNotFound(t *testing.T) {
	repo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return nil, nil }}
	h := NewDataErasureHandler(&mockDataErasureService{}, repo)
	r := withTenantAndActor(httptest.NewRequest(http.MethodPost, "/users/"+testUserUUID.String()+"/erasure-requests", nil), 5)
	r = withChiParam(r, "user_uuid", testUserUUID.String())
	w := httptest.NewRecorder()
	h.RequestAdmin(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDataErasureHandler_RequestAdmin_CrossTenantRejected(t *testing.T) {
	repo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) {
		return &User{UserID: 99, TenantID: 999}, nil // different tenant
	}}
	h := NewDataErasureHandler(&mockDataErasureService{}, repo)
	r := withTenantAndActor(httptest.NewRequest(http.MethodPost, "/users/"+testUserUUID.String()+"/erasure-requests", nil), 5)
	r = withChiParam(r, "user_uuid", testUserUUID.String())
	w := httptest.NewRecorder()
	h.RequestAdmin(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDataErasureHandler_RequestAdmin_BadJSON(t *testing.T) {
	repo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) {
		return &User{UserID: 99, TenantID: tenantID}, nil
	}}
	h := NewDataErasureHandler(&mockDataErasureService{}, repo)
	r := withTenantAndActor(badJSONReq(t, http.MethodPost, "/users/"+testUserUUID.String()+"/erasure-requests"), 5)
	r = withChiParam(r, "user_uuid", testUserUUID.String())
	w := httptest.NewRecorder()
	h.RequestAdmin(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDataErasureHandler_RequestAdmin_ServiceError(t *testing.T) {
	repo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) {
		return &User{UserID: 99, TenantID: tenantID}, nil
	}}
	svc := &mockDataErasureService{requestFn: func(RequestErasureInput) (*DataErasureRequestResult, error) {
		return nil, assert.AnError
	}}
	h := NewDataErasureHandler(svc, repo)
	r := withTenantAndActor(jsonReq(t, http.MethodPost, "/users/"+testUserUUID.String()+"/erasure-requests", map[string]any{"reason": "gdpr"}), 5)
	r = withChiParam(r, "user_uuid", testUserUUID.String())
	w := httptest.NewRecorder()
	h.RequestAdmin(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDataErasureHandler_RequestAdmin_Success(t *testing.T) {
	repo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) {
		return &User{UserID: 99, TenantID: tenantID}, nil
	}}
	svc := &mockDataErasureService{requestFn: func(in RequestErasureInput) (*DataErasureRequestResult, error) {
		assert.Equal(t, tenantID, in.TenantID)
		assert.Equal(t, int64(99), in.UserID)
		// Admin-initiated: requested_by_admin_id set, requested_by_user_id nil.
		if assert.NotNil(t, in.RequestedByAdminID) {
			assert.Equal(t, int64(5), *in.RequestedByAdminID)
		}
		assert.Nil(t, in.RequestedByUserID)
		assert.Equal(t, "gdpr", in.Reason)
		return erasureResult(), nil
	}}
	h := NewDataErasureHandler(svc, repo)
	r := withTenantAndActor(jsonReq(t, http.MethodPost, "/users/"+testUserUUID.String()+"/erasure-requests", map[string]any{"reason": "gdpr"}), 5)
	r = withChiParam(r, "user_uuid", testUserUUID.String())
	w := httptest.NewRecorder()
	h.RequestAdmin(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "pending")
}

func TestDataErasureHandler_RequestAdmin_EmptyBodyOK(t *testing.T) {
	repo := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) {
		return &User{UserID: 99, TenantID: tenantID}, nil
	}}
	svc := &mockDataErasureService{requestFn: func(in RequestErasureInput) (*DataErasureRequestResult, error) {
		assert.Equal(t, "", in.Reason)
		return erasureResult(), nil
	}}
	h := NewDataErasureHandler(svc, repo)
	r := withTenantAndActor(httptest.NewRequest(http.MethodPost, "/users/"+testUserUUID.String()+"/erasure-requests", nil), 5)
	r = withChiParam(r, "user_uuid", testUserUUID.String())
	w := httptest.NewRecorder()
	h.RequestAdmin(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ──────────────────────────────────────────────────────────────────────────────
// Self endpoint — POST /me/erasure-request
// ──────────────────────────────────────────────────────────────────────────────

func TestDataErasureHandler_RequestSelf_NoAuth(t *testing.T) {
	h := NewDataErasureHandler(&mockDataErasureService{}, &mockUserRepo{})
	w := httptest.NewRecorder()
	h.RequestSelf(w, httptest.NewRequest(http.MethodPost, "/me/erasure-request", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDataErasureHandler_RequestSelf_NoTenant(t *testing.T) {
	h := NewDataErasureHandler(&mockDataErasureService{}, &mockUserRepo{})
	// User present but no tenant.
	r := withUserNoTenant(httptest.NewRequest(http.MethodPost, "/me/erasure-request", nil))
	w := httptest.NewRecorder()
	h.RequestSelf(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDataErasureHandler_RequestSelf_BadJSON(t *testing.T) {
	h := NewDataErasureHandler(&mockDataErasureService{}, &mockUserRepo{})
	r := withTenantAndActor(badJSONReq(t, http.MethodPost, "/me/erasure-request"), 7)
	w := httptest.NewRecorder()
	h.RequestSelf(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDataErasureHandler_RequestSelf_ServiceError(t *testing.T) {
	svc := &mockDataErasureService{requestFn: func(RequestErasureInput) (*DataErasureRequestResult, error) {
		return nil, assert.AnError
	}}
	h := NewDataErasureHandler(svc, &mockUserRepo{})
	r := withTenantAndActor(jsonReq(t, http.MethodPost, "/me/erasure-request", map[string]any{"reason": "x"}), 7)
	w := httptest.NewRecorder()
	h.RequestSelf(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDataErasureHandler_RequestSelf_Success(t *testing.T) {
	svc := &mockDataErasureService{requestFn: func(in RequestErasureInput) (*DataErasureRequestResult, error) {
		assert.Equal(t, tenantID, in.TenantID)
		assert.Equal(t, int64(7), in.UserID)
		// Self-initiated: requested_by_user_id set, requested_by_admin_id nil.
		if assert.NotNil(t, in.RequestedByUserID) {
			assert.Equal(t, int64(7), *in.RequestedByUserID)
		}
		assert.Nil(t, in.RequestedByAdminID)
		return erasureResult(), nil
	}}
	h := NewDataErasureHandler(svc, &mockUserRepo{})
	r := withTenantAndActor(jsonReq(t, http.MethodPost, "/me/erasure-request", map[string]any{"reason": "bye"}), 7)
	w := httptest.NewRecorder()
	h.RequestSelf(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "pending")
}
