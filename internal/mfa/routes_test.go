package mfa

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
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
