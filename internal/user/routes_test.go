package user

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestAccountRoute(t *testing.T) {
	r := chi.NewRouter()
	h := NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil)
	AccountRoute(r, h, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/account/sessions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRecoveryRoute(t *testing.T) {
	r := chi.NewRouter()
	h := NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil)
	RecoveryRoute(r, h)

	req := jsonReq(t, http.MethodPost, "/recovery/backup-code", map[string]string{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileRoute(t *testing.T) {
	r := chi.NewRouter()
	h := NewProfileHandler(&mockProfileService{})
	ProfileRoute(r, h, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserRoute(t *testing.T) {
	r := chi.NewRouter()
	h := NewUserHandler(&mockUserService{})
	UserRoute(r, h, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserSettingRoute(t *testing.T) {
	r := chi.NewRouter()
	h := NewUserSettingHandler(&mockUserSettingService{})
	UserSettingRoute(r, h, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/user-settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
