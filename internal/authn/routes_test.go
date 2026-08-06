package authn

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// Every interactive-auth builder in this package is a *PublicRoute; the
// internal-plane twins this table used to also cover (EmailVerificationRoute,
// ForgotPasswordRoute, LoginRoute, ResetPasswordRoute, SMSLoginInternalRoute)
// were mounted by no router and are gone. A row here for a non-public builder
// means dead surface has come back.
func TestAuthnRoutes_RegisterEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		register func(chi.Router)
		request  string
	}{
		{"email verification public", func(r chi.Router) {
			EmailVerificationPublicRoute(r, NewEmailVerificationHandler(&mockEmailVerificationService{}))
		}, "/email-verification/send"},
		{"forgot password public", func(r chi.Router) {
			ForgotPasswordPublicRoute(r, NewForgotPasswordHandler(&mockForgotPasswordService{}))
		}, "/forgot-password"},
		{"login public", func(r chi.Router) { LoginPublicRoute(r, NewLoginHandler(&mockLoginService{})) }, "/login"},
		{"login public refresh", func(r chi.Router) { LoginPublicRoute(r, NewLoginHandler(&mockLoginService{})) }, "/refresh-token"},
		{"login public mfa verify", func(r chi.Router) { LoginPublicRoute(r, NewLoginHandler(&mockLoginService{})) }, "/login/mfa/verify"},
		{"login public mfa send sms", func(r chi.Router) { LoginPublicRoute(r, NewLoginHandler(&mockLoginService{})) }, "/login/mfa/send-sms"},
		{"login public mfa send email otp", func(r chi.Router) {
			LoginPublicRoute(r, NewLoginHandler(&mockLoginService{}))
		}, "/login/mfa/send-email-otp"},
		{"login public mfa webauthn begin", func(r chi.Router) {
			LoginPublicRoute(r, NewLoginHandler(&mockLoginService{}))
		}, "/login/mfa/webauthn/begin"},
		{"magic link public", func(r chi.Router) { MagicLinkPublicRoute(r, NewMagicLinkHandler(&mockMagicLinkService{})) }, "/magic-link/send"},
		{"register public", func(r chi.Router) {
			RegisterPublicRoute(r, NewRegisterHandler(&mockRegisterService{}))
		}, "/register"},
		{"register public invite", func(r chi.Router) {
			RegisterPublicRoute(r, NewRegisterHandler(&mockRegisterService{}))
		}, "/register/invite"},
		{"reset password public", func(r chi.Router) { ResetPasswordPublicRoute(r, NewResetPasswordHandler(&mockResetPasswordService{})) }, "/reset-password"},
		{"sms login public", func(r chi.Router) { SMSLoginPublicRoute(r, NewSMSLoginHandler(&mockSMSLoginService{})) }, "/sms-login/send"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := chi.NewRouter()
			tt.register(router)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, tt.request, nil))

			assert.NotEqual(t, http.StatusNotFound, w.Code)
		})
	}
}
