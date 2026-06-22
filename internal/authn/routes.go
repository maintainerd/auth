package authn

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

// EmailVerificationRoute handles internal email-verification routes
// (no client_id/provider_id required). Mounted on the management surface (port 8080).
func EmailVerificationRoute(r chi.Router, emailVerificationHandler *EmailVerificationHandler) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		r.Post("/email-verification/send", emailVerificationHandler.SendVerificationEmail)
		r.Post("/email-verification/verify", emailVerificationHandler.VerifyEmail)
	})
}

// EmailVerificationPublicRoute handles public email-verification routes
// (send requires client_id and provider_id; verify is self-contained).
// Mounted on the public surface (port 8081).
func EmailVerificationPublicRoute(r chi.Router, emailVerificationHandler *EmailVerificationHandler) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		r.Post("/email-verification/send", emailVerificationHandler.SendVerificationEmailPublic)
		r.Post("/email-verification/verify", emailVerificationHandler.VerifyEmail)
	})
}

// ForgotPasswordRoute handles internal forgot password routes (no client_id/provider_id required)
func ForgotPasswordRoute(r chi.Router, forgotPasswordHandler *ForgotPasswordHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Internal forgot password (no client_id/provider_id required)
		r.Post("/forgot-password", forgotPasswordHandler.ForgotPassword)
	})
}

// ForgotPasswordPublicRoute handles public forgot password routes (requires client_id and provider_id)
func ForgotPasswordPublicRoute(r chi.Router, forgotPasswordHandler *ForgotPasswordHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Public forgot password (with client_id and provider_id)
		r.Post("/forgot-password", forgotPasswordHandler.ForgotPasswordPublic)
	})
}

// LoginRoute handles internal login routes (no client_id/provider_id required)
func LoginRoute(r chi.Router, loginHandler *LoginHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Internal login (no client_id/provider_id required)
		r.Post("/login", loginHandler.Login)

		// Login MFA second step (issues an acr=2 session on success)
		r.Post("/login/mfa/verify", loginHandler.MFALoginVerify)
		r.Post("/login/mfa/send-sms", loginHandler.MFALoginSendSMS)
		r.Post("/login/mfa/send-email-otp", loginHandler.MFALoginSendEmailOTP)
		r.Post("/login/mfa/webauthn/begin", loginHandler.MFALoginWebAuthnBegin)

		// Refresh endpoint — exchanges a refresh token for a new token set
		r.Post("/refresh-token", loginHandler.RefreshToken)

		// Logout endpoint (clears cookies if they exist)
		r.Post("/logout", loginHandler.Logout)
	})
}

// LoginPublicRoute handles public login routes (requires client_id and provider_id)
func LoginPublicRoute(r chi.Router, loginHandler *LoginHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Public login (with client_id and provider_id)
		r.Post("/login", loginHandler.LoginPublic)

		// Login MFA second step (client_id/provider_id passed as query params)
		r.Post("/login/mfa/verify", loginHandler.MFALoginVerify)
		r.Post("/login/mfa/send-sms", loginHandler.MFALoginSendSMS)
		r.Post("/login/mfa/send-email-otp", loginHandler.MFALoginSendEmailOTP)
		r.Post("/login/mfa/webauthn/begin", loginHandler.MFALoginWebAuthnBegin)

		// Logout endpoint (clears cookies if they exist)
		r.Post("/logout", loginHandler.Logout)
	})
}

// MagicLinkPublicRoute mounts public magic-link routes.
// `send` requires client_id; `verify` extracts the context from the signed link.
// Mounted on the public surface (port 8081).
func MagicLinkPublicRoute(r chi.Router, magicLinkHandler *MagicLinkHandler) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		r.Post("/magic-link/send", magicLinkHandler.SendMagicLinkPublic)
		r.Post("/magic-link/verify", magicLinkHandler.VerifyMagicLink)
	})
}



// RegisterRoute handles internal register routes (no client_id/provider_id required)
func RegisterRoute(r chi.Router, registerHandler *RegisterHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Internal registration (no client_id/provider_id required)
		r.Post("/register", registerHandler.Register)

		// Internal registration with invite
		r.Post("/register/invite", registerHandler.RegisterInvite)
	})
}

// RegisterPublicRoute handles public register routes (requires client_id and provider_id)
func RegisterPublicRoute(r chi.Router, registerHandler *RegisterHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Public registration (with client_id and provider_id)
		r.Post("/register", registerHandler.RegisterPublic)

		// Public registration with invite
		r.Post("/register/invite", registerHandler.RegisterInvitePublic)
	})
}

// ResetPasswordRoute handles internal reset password routes (no client_id/provider_id required)
func ResetPasswordRoute(r chi.Router, resetPasswordHandler *ResetPasswordHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Internal reset password (no client_id/provider_id required)
		r.Post("/reset-password", resetPasswordHandler.ResetPassword)
	})
}

// ResetPasswordPublicRoute handles public reset password routes (requires client_id and provider_id)
func ResetPasswordPublicRoute(r chi.Router, resetPasswordHandler *ResetPasswordHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		// Public reset password (with client_id and provider_id)
		r.Post("/reset-password", resetPasswordHandler.ResetPasswordPublic)
	})
}

// SMSLoginInternalRoute mounts unauthenticated SMS one-time-code login endpoints
// for the internal surface (port 8080). Requires tenant_id in the request body.
func SMSLoginInternalRoute(
	r chi.Router,
	smsLoginHandler *SMSLoginHandler,
) {
	r.Route("/sms-login", func(r chi.Router) {
		// Send OTP to phone number (unauthenticated, tenant-scoped)
		r.Post("/send", smsLoginHandler.SendOTPInternal)
		// Verify OTP and obtain tokens (unauthenticated, tenant-scoped)
		r.Post("/verify", smsLoginHandler.VerifyOTPInternal)
	})
}

// SMSLoginPublicRoute mounts unauthenticated SMS one-time-code login endpoints
// for the public surface (port 8081). Requires client_id in the request body.
func SMSLoginPublicRoute(
	r chi.Router,
	smsLoginHandler *SMSLoginHandler,
) {
	r.Route("/sms-login", func(r chi.Router) {
		// Send OTP to phone number (unauthenticated, client-scoped)
		r.Post("/send", smsLoginHandler.SendOTPPublic)
		// Verify OTP and obtain tokens (unauthenticated, client-scoped)
		r.Post("/verify", smsLoginHandler.VerifyOTPPublic)
	})
}
