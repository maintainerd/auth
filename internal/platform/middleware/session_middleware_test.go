package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
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

	// No JWT claims at all: this is the setup / health-probe path,
	// which carries its own auth rules and is not what the sid rule is about.
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

// A JWT with no `sid` used to sail past session validation entirely, which made
// it strictly more powerful than one that had a sid: logout, "sign out
// everywhere", session revocation and password change all operate on
// user_sessions, so a token with nothing to look up survived all of them.
func TestSessionValidationMiddleware_TokenWithoutSessionBinding(t *testing.T) {
	newHandler := func(t *testing.T, called *bool) http.Handler {
		t.Helper()
		return SessionValidationMiddleware(&mockSessionValidator{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*called = true
			w.WriteHeader(http.StatusOK)
		}))
	}

	t.Run("user token with no sid is rejected", func(t *testing.T) {
		called := false
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = WithJWTClaims(r, &JWTClaims{Sub: uuid.New().String()})
		r = WithAuthContext(r, &authctx.AuthContext{
			User: &authctx.AuthUser{UserUUID: uuid.New(), UserID: 1},
		})
		w := httptest.NewRecorder()
		newHandler(t, &called).ServeHTTP(w, r)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, called)
	})

	// Grants that have no browser session by construction must keep working;
	// requiring a sid of them would be wrong, not safer.
	for _, subjectType := range []string{"client", "service", "device", "ciba", "exchange"} {
		t.Run("sub_type "+subjectType+" is exempt", func(t *testing.T) {
			called := false
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r = WithJWTClaims(r, &JWTClaims{Sub: "svc", SubjectType: subjectType})
			w := httptest.NewRecorder()
			newHandler(t, &called).ServeHTTP(w, r)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.True(t, called)
		})
	}

	t.Run("a sid on the token is still validated against the session store", func(t *testing.T) {
		sessionUUID := uuid.New()
		var seen uuid.UUID
		svc := &mockSessionValidator{validateAndTouchFn: func(sid uuid.UUID, _ int64) error {
			seen = sid
			return nil
		}}
		handler := SessionValidationMiddleware(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = WithJWTClaims(r, &JWTClaims{Sub: uuid.New().String(), SessionID: sessionUUID.String()})
		r = WithAuthContext(r, &authctx.AuthContext{
			User: &authctx.AuthUser{UserUUID: uuid.New(), UserID: 7},
		})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, sessionUUID, seen)
	})
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
