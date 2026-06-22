package authn

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestAuthnRoutes_RegisterEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		register func(chi.Router)
		request  string
	}{
		{"email verification internal", func(r chi.Router) {
			EmailVerificationRoute(r, NewEmailVerificationHandler(&mockEmailVerificationService{}))
		}, "/email-verification/send"},
		{"email verification public", func(r chi.Router) {
			EmailVerificationPublicRoute(r, NewEmailVerificationHandler(&mockEmailVerificationService{}))
		}, "/email-verification/send"},
		{"forgot password internal", func(r chi.Router) { ForgotPasswordRoute(r, NewForgotPasswordHandler(&mockForgotPasswordService{})) }, "/forgot-password"},
		{"forgot password public", func(r chi.Router) {
			ForgotPasswordPublicRoute(r, NewForgotPasswordHandler(&mockForgotPasswordService{}))
		}, "/forgot-password"},
		{"login internal", func(r chi.Router) { LoginRoute(r, NewLoginHandler(&mockLoginService{})) }, "/login"},
		{"login public", func(r chi.Router) { LoginPublicRoute(r, NewLoginHandler(&mockLoginService{})) }, "/login"},
		{"magic link internal", func(r chi.Router) { MagicLinkRoute(r, NewMagicLinkHandler(&mockMagicLinkService{})) }, "/magic-link/send"},
		{"magic link public", func(r chi.Router) { MagicLinkPublicRoute(r, NewMagicLinkHandler(&mockMagicLinkService{})) }, "/magic-link/send"},
		{"register internal", func(r chi.Router) {
			RegisterRoute(r, NewRegisterHandler(&mockRegisterService{}))
		}, "/register"},
		{"register internal invite", func(r chi.Router) {
			RegisterRoute(r, NewRegisterHandler(&mockRegisterService{}))
		}, "/register/invite"},
		{"register public", func(r chi.Router) {
			RegisterPublicRoute(r, NewRegisterHandler(&mockRegisterService{}))
		}, "/register"},
		{"register public invite", func(r chi.Router) {
			RegisterPublicRoute(r, NewRegisterHandler(&mockRegisterService{}))
		}, "/register/invite"},
		{"reset password internal", func(r chi.Router) { ResetPasswordRoute(r, NewResetPasswordHandler(&mockResetPasswordService{})) }, "/reset-password"},
		{"reset password public", func(r chi.Router) { ResetPasswordPublicRoute(r, NewResetPasswordHandler(&mockResetPasswordService{})) }, "/reset-password"},
		{"sms login internal", func(r chi.Router) { SMSLoginInternalRoute(r, NewSMSLoginHandler(&mockSMSLoginService{})) }, "/sms-login/send"},
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
