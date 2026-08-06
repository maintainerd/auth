package mfa

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
)

func TestMFAInternalRouteMountsSelfServiceAndAdminEndpoints(t *testing.T) {
	router := chi.NewRouter()
	MFAInternalRoute(router, NewMFAHandler(&mockMFAService{}, &mockWebAuthnService{}), nil, nil)

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/mfa/status"},
		{http.MethodPost, "/mfa/totp/enroll"},
		{http.MethodPost, "/mfa/totp/verify"},
		{http.MethodDelete, "/mfa/totp"},
		{http.MethodGet, "/mfa/backup-codes/count"},
		{http.MethodPost, "/mfa/backup-codes/regenerate"},
		{http.MethodPost, "/mfa/webauthn/register/begin"},
		{http.MethodPost, "/mfa/webauthn/register/finish"},
		{http.MethodPost, "/mfa/webauthn/auth/begin"},
		{http.MethodPost, "/mfa/webauthn/auth/finish"},
		{http.MethodDelete, "/mfa/webauthn/" + mfaTestCredentialUUID.String()},
		{http.MethodPost, "/mfa/step-up/challenge"},
		{http.MethodPost, "/mfa/step-up/verify"},
		{http.MethodPost, "/mfa/reset"},
		{http.MethodPost, "/mfa/admin/users/" + mfaTestUserUUID.String() + "/reset"},
		{http.MethodPost, "/mfa/admin/users/" + mfaTestUserUUID.String() + "/reset/totp"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			assert.True(t, router.Match(match, tt.method, tt.path))
		})
	}
}

func TestMFAPublicRouteMountsOnlySelfServiceEndpoints(t *testing.T) {
	router := chi.NewRouter()
	MFAPublicRoute(router, NewMFAHandler(&mockMFAService{}, &mockWebAuthnService{}), nil, nil)

	selfService := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/mfa/status"},
		{http.MethodPost, "/mfa/totp/enroll"},
		{http.MethodPost, "/mfa/totp/verify"},
		{http.MethodDelete, "/mfa/totp"},
		{http.MethodPost, "/mfa/webauthn/register/begin"},
		{http.MethodPost, "/mfa/webauthn/register/finish"},
		{http.MethodPost, "/mfa/step-up/verify"},
		{http.MethodPost, "/mfa/reset"},
	}

	for _, tt := range selfService {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			assert.True(t, router.Match(match, tt.method, tt.path))
		})
	}

	admin := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/mfa/admin/users/" + mfaTestUserUUID.String() + "/reset"},
		{http.MethodPost, "/mfa/admin/users/" + mfaTestUserUUID.String() + "/reset/totp"},
	}

	for _, tt := range admin {
		t.Run("not mounted "+tt.method+" "+tt.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			assert.False(t, router.Match(match, tt.method, tt.path))
		})
	}
}

// mfaEnrollmentRoutes is every route that ADDS an MFA factor to the caller's own
// account. Each one mints a credential that /mfa/step-up/verify will accept, so
// each must be gated once the account already holds a factor. Paths are relative
// to the /mfa mount that MFAInternalRoute/MFAPublicRoute wrap around
// mountSelfMFARoutes.
var mfaEnrollmentRoutes = []string{
	"/totp/enroll",
	"/totp/verify",
	"/webauthn/register/begin",
	"/webauthn/register/finish",
	"/sms/enroll",
	"/sms/verify",
	"/email-otp/enroll",
	"/email-otp/verify",
}

// enrollPermittedRequest builds a request whose caller holds
// account:mfa:enroll:self but has NOT stepped up — the exact shape of a hijacked
// acr=1 session.
func enrollPermittedRequest(path string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	return middleware.WithAuthContext(req, &authctx.AuthContext{
		User: &authctx.AuthUser{
			UserID:   mfaTestUserID,
			UserUUID: mfaTestUserUUID,
			Roles: []authctx.AuthRole{{
				Name:        "registered",
				Permissions: []authctx.AuthPermission{{Name: "account:mfa:enroll:self"}},
			}},
		},
		Tenant: &authctx.AuthTenant{TenantID: mfaTestTenantID, TenantUUID: mfaTestTenantUUID},
	})
}

// INVERTED intent. Enrollment carried PermissionMiddleware only, so every route
// below reached its handler on a plain acr=1 session: an attacker holding a
// stolen session could enrol their own TOTP/passkey, complete
// /mfa/step-up/verify with it, and reach acr=2 — clearing every step-up gate on
// the victim's account, including /mfa/reset.
func TestSelfMFARoutes_EnrollmentIsGatedOnceAFactorExists(t *testing.T) {
	router := chi.NewRouter()
	mountSelfMFARoutes(router, NewMFAHandler(&mockMFAService{
		userHasMFAFn: func(context.Context, int64) (bool, error) { return true, nil },
	}, &mockWebAuthnService{}))

	for _, path := range mfaEnrollmentRoutes {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, enrollPermittedRequest(path))

			assert.Equal(t, http.StatusForbidden, rec.Code)
			assert.Contains(t, rec.Body.String(), "step_up_required")
		})
	}
}

// The bootstrap path must stay open: an account with no factor cannot step up,
// so gating its first enrollment would make MFA impossible to turn on.
func TestSelfMFARoutes_FirstEnrollmentNeedsNoStepUp(t *testing.T) {
	router := chi.NewRouter()
	mountSelfMFARoutes(router, NewMFAHandler(&mockMFAService{
		userHasMFAFn: func(context.Context, int64) (bool, error) { return false, nil },
	}, &mockWebAuthnService{}))

	for _, path := range mfaEnrollmentRoutes {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, enrollPermittedRequest(path))

			assert.NotContains(t, rec.Body.String(), "step_up_required")
		})
	}
}
