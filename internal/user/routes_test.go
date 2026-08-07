package user

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// mountedRoutes enumerates the routes a router actually registered, as
// "METHOD /path" strings.
//
// Walk rather than Match or a served request. A served request is useless here:
// every one of these paths sits under a chi group carrying JWTAuthMiddleware,
// so it answers 401 whether or not the route exists. And Match reports true for
// ANY method on a group's own mount point — chi registers that node for all
// methods — so it cannot tell a removed "PUT /profile" from a live one. Walk
// reads the route table itself, which is the thing under test.
func mountedRoutes(t *testing.T, r chi.Routes) map[string]bool {
	t.Helper()
	got := map[string]bool{}
	require.NoError(t, chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		got[method+" "+strings.TrimSuffix(route, "/")] = true
		return nil
	}))
	return got
}

// The internal console surface displays people; it does not let anyone manage
// their own account. Those flows live in the identity app, and mounting a
// second copy here would mean two sets of the same step-up and policy guards,
// only one of which anything exercises.
func TestAccountSelfReadRoute_MountsOnlyTheSignedInAsRead(t *testing.T) {
	r := chi.NewRouter()
	AccountSelfReadRoute(r, NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil), nil, nil)

	// Exhaustive, not a spot-check: anything added to this surface later has to
	// be justified against the comment above rather than slipping in.
	assert.Equal(t, map[string]bool{"GET /account": true}, mountedRoutes(t, r),
		`the console needs GET /account for "signed in as" and nothing else`)
}

func TestProfileSelfReadRoute_MountsOnlyTheDisplayReads(t *testing.T) {
	r := chi.NewRouter()
	ProfileSelfReadRoute(r, NewProfileHandler(&mockProfileService{}), nil, nil)

	assert.Equal(t, map[string]bool{
		// The signed-in admin's own profile — name and avatar in the top nav.
		"GET /profile": true,
		// Where an uploaded avatar is actually served from. Every profile_url the
		// console renders points here, including on the admin user-management
		// screens, so losing it would break other users' avatars too.
		"GET /profiles/{profile_uuid}/picture": true,
	}, mountedRoutes(t, r), "profile EDITING belongs to the identity app (self) or UserRoute (admin)")
}

func TestUserRoute(t *testing.T) {
	r := chi.NewRouter()
	h := NewUserHandler(&mockUserService{})
	UserRoute(r, h, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// routeHasStepUp reports whether middleware.RequireStepUp sits in the chain chi
// composed for one route. Compared by function pointer because RequireStepUp is
// a plain func value; chi.Walk unwraps the inline `With` chain, so route-level
// middleware is visible here.
func routeHasStepUp(t *testing.T, r chi.Routes, methodAndPattern string) bool {
	t.Helper()
	found := false
	require.NoError(t, chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		if method+" "+strings.TrimSuffix(route, "/") != methodAndPattern {
			return nil
		}
		for _, mw := range mws {
			if reflect.ValueOf(mw).Pointer() == reflect.ValueOf(middleware.RequireStepUp).Pointer() {
				found = true
			}
		}
		return nil
	}))
	return found
}

// PUT /users/{user_uuid} rewrites the account's sign-in identity (email,
// username) and its status, so it is at least as destructive as PATCH /status
// and DELETE, both of which already required step-up. Carrying no step-up meant
// a stolen acr=1 admin session could repoint a victim's account at an
// attacker-controlled inbox and take it over through forgot-password.
func TestUserRoute_UpdateRequiresStepUp(t *testing.T) {
	r := chi.NewRouter()
	UserRoute(r, NewUserHandler(&mockUserService{}), nil, nil, nil, nil, nil)

	assert.True(t, routeHasStepUp(t, r, "PUT /users/{user_uuid}"))
	assert.True(t, routeHasStepUp(t, r, "PUT /users/{user_uuid}/password"))
	assert.True(t, routeHasStepUp(t, r, "POST /users/{user_uuid}/identities"))
	// Negative control: a read route has no step-up, so a routeHasStepUp that
	// simply always answered true could not pass this test.
	assert.False(t, routeHasStepUp(t, r, "GET /users"))
}

// POST /me/erasure-request schedules an irreversible multi-table anonymisation
// of the caller's own account. It used to carry only a permission check while
// the strictly LESS destructive DELETE /account demanded step-up, so a hijacked
// acr=1 session could permanently destroy the victim's account.
func TestDataErasureSelfRoute_RequiresStepUp(t *testing.T) {
	r := chi.NewRouter()
	DataErasureSelfRoute(r, NewDataErasureHandler(nil, nil), nil, nil)

	assert.True(t, routeHasStepUp(t, r, "POST /me/erasure-request"))
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
