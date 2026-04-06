package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
)

// withInvalidUserContext injects a non-*model.User value at UserContextKey to
// trigger the type-assertion failure branch (line 44-48 in invite_handler.go).
func withInvalidUserContext(r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserContextKey, "not-a-user-struct")
	return r.WithContext(ctx)
}

func TestInviteHandler_Send_NoTenant(t *testing.T) {
	h := NewInviteHandler(&mockInviteService{})
	r := httptest.NewRequest(http.MethodPost, "/invites", nil)
	w := httptest.NewRecorder()
	h.Send(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestInviteHandler_Send_NoUser(t *testing.T) {
	h := NewInviteHandler(&mockInviteService{})
	r := withTenant(httptest.NewRequest(http.MethodPost, "/invites", nil))
	w := httptest.NewRecorder()
	h.Send(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestInviteHandler_Send_BadJSON(t *testing.T) {
	h := NewInviteHandler(&mockInviteService{})
	r := withTenantAndUser(httptest.NewRequest(http.MethodPost, "/invites", nil))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Send(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInviteHandler_Send_ServiceError(t *testing.T) {
	svc := &mockInviteService{
		sendInviteFn: func(tid int64, email string, uid int64, roles []string) (*model.Invite, error) {
			return nil, assert.AnError
		},
	}
	h := NewInviteHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/invites", map[string]interface{}{
		"email": "user@example.com",
		"roles": []string{testResourceUUID.String()},
	}))
	w := httptest.NewRecorder()
	h.Send(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestInviteHandler_Send_InvalidUserType(t *testing.T) {
	// Injects a non-*model.User at UserContextKey → type assertion fails → 500.
	h := NewInviteHandler(&mockInviteService{})
	r := withTenant(withInvalidUserContext(httptest.NewRequest(http.MethodPost, "/invites", nil)))
	w := httptest.NewRecorder()
	h.Send(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestInviteHandler_Send_ValidationError(t *testing.T) {
	// Email present but Roles omitted → req.Validate() returns error → 400.
	h := NewInviteHandler(&mockInviteService{})
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/invites", map[string]any{
		"email": "user@example.com",
	}))
	w := httptest.NewRecorder()
	h.Send(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInviteHandler_Send_Success(t *testing.T) {
	svc := &mockInviteService{
		sendInviteFn: func(tid int64, email string, uid int64, roles []string) (*model.Invite, error) {
			return &model.Invite{}, nil
		},
	}
	h := NewInviteHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/invites", map[string]interface{}{
		"email": "user@example.com",
		"roles": []string{testResourceUUID.String()},
	}))
	w := httptest.NewRecorder()
	h.Send(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
