package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authctx"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
)

type mockSessionValidator struct {
	validateAndTouchFn func(uuid.UUID, int64) error
}

func (m *mockSessionValidator) ValidateAndTouch(_ context.Context, sessionUUID uuid.UUID, userID int64) error {
	if m.validateAndTouchFn != nil {
		return m.validateAndTouchFn(sessionUUID, userID)
	}
	return nil
}

func TestSessionValidationMiddleware_NoHeader(t *testing.T) {
	svc := &mockSessionValidator{}
	handler := SessionValidationMiddleware(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSessionValidationMiddleware_InvalidUUID(t *testing.T) {
	svc := &mockSessionValidator{}
	handler := SessionValidationMiddleware(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called")
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Session-ID", "not-a-uuid")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSessionValidationMiddleware_NoUser(t *testing.T) {
	svc := &mockSessionValidator{}
	handler := SessionValidationMiddleware(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called")
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Session-ID", uuid.New().String())
	// No auth context set on request.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSessionValidationMiddleware_ValidSession(t *testing.T) {
	sessionUUID := uuid.New()
	userUUID := uuid.New()

	svc := &mockSessionValidator{
		validateAndTouchFn: func(sid uuid.UUID, uid int64) error {
			assert.Equal(t, sessionUUID, sid)
			return nil
		},
	}
	handler := SessionValidationMiddleware(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Session-ID", sessionUUID.String())
	r = WithAuthContext(r, &authctx.AuthContext{
		User: &authctx.AuthUser{UserUUID: userUUID, UserID: 42},
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSessionValidationMiddleware_UnauthorizedError(t *testing.T) {
	svc := &mockSessionValidator{
		validateAndTouchFn: func(uuid.UUID, int64) error {
			return apperror.NewUnauthorized("session expired")
		},
	}
	handler := SessionValidationMiddleware(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called")
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Session-ID", uuid.New().String())
	r = WithAuthContext(r, &authctx.AuthContext{
		User: &authctx.AuthUser{UserUUID: uuid.New(), UserID: 1},
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSessionValidationMiddleware_GenericError(t *testing.T) {
	svc := &mockSessionValidator{
		validateAndTouchFn: func(uuid.UUID, int64) error {
			return errors.New("db connection lost")
		},
	}
	handler := SessionValidationMiddleware(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called")
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Session-ID", uuid.New().String())
	r = WithAuthContext(r, &authctx.AuthContext{
		User: &authctx.AuthUser{UserUUID: uuid.New(), UserID: 1},
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
