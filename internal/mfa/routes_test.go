package mfa

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestMFARouteMountsEndpoints(t *testing.T) {
	router := chi.NewRouter()
	MFARoute(router, NewMFAHandler(&mockMFAService{}, &mockWebAuthnService{}), nil, nil)

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
		{http.MethodPost, "/mfa/admin/users/" + mfaTestUserUUID.String() + "/reset"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			assert.True(t, router.Match(match, tt.method, tt.path))
		})
	}
}
