package authn

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
)

func publicAuthSurface(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(contextWithPublicAuthSurface(r.Context())))
	})
}

// AccountLinkConfirmRoute mounts the social-login account-link confirmation
// endpoint. It requires the caller to be authenticated as the existing account
// being linked (JWT + user context), so it is not part of the unauthenticated
// public auth surface.
func AccountLinkConfirmRoute(
	r chi.Router,
	handler *AccountLinkHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.Post("/account-link/{token}/confirm", handler.Confirm)
	})
}

// EmailVerificationRoute handles internal email-verification routes
// (no client_id/provider_id required). Mounted on the management surface (port 8080).
func EmailVerificationRoute(r chi.Router, emailVerificationHandler *EmailVerificationHandler) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))

		r.Post("/email-verification/send", emailVerificationHandler.SendVerificationEmail)
		r.Post("/email-verification/verify", emailVerificationHandler.VerifyEmailInternal)
	})
}

// EmailVerificationPublicRoute handles public email-verification routes
// (both operations require client_id and reject tenant_id).
// Mounted on the public surface (port 8081).
func EmailVerificationPublicRoute(r chi.Router, emailVerificationHandler *EmailVerificationHandler) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))
		r.Use(publicAuthSurface)

		r.Post("/email-verification/send", emailVerificationHandler.SendVerificationEmailPublic)
		r.Post("/email-verification/verify", emailVerificationHandler.VerifyEmailPublic)
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

// ForgotPasswordPublicRoute handles public forgot password routes (requires client_id or tenant_id)
func ForgotPasswordPublicRoute(r chi.Router, forgotPasswordHandler *ForgotPasswordHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))
		r.Use(publicAuthSurface)

		// Public forgot password (with client_id or tenant_id)
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

		LoginMFAInternalRoute(r, loginHandler)

		// Refresh endpoint — exchanges a refresh token for a new token set
		r.Post("/refresh-token", loginHandler.RefreshToken)

		// Logout endpoint (clears cookies if they exist)
		r.Post("/logout", loginHandler.Logout)
	})
}

// LoginPublicRoute handles public login routes (requires client_id or tenant_id)
func LoginPublicRoute(r chi.Router, loginHandler *LoginHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))
		r.Use(publicAuthSurface)

		// Public login (with client_id or tenant_id)
		r.Post("/login", loginHandler.LoginPublic)

		LoginMFAPublicRoute(r, loginHandler)

		// Refresh endpoint — exchanges a refresh token for a new token set.
		r.Post("/refresh-token", loginHandler.RefreshToken)

		// Logout endpoint (clears cookies if they exist)
		r.Post("/logout", loginHandler.Logout)
	})
}

// LoginMFAInternalRoute mounts internal login-MFA endpoints. Internal MFA is
// tenant-scoped and lives only on the private/internal API surface.
func LoginMFAInternalRoute(r chi.Router, loginHandler *LoginHandler) {
	r.Post("/login/mfa/verify", loginHandler.MFALoginVerifyInternal)
	r.Post("/login/mfa/send-sms", loginHandler.MFALoginSendSMSInternal)
	r.Post("/login/mfa/send-email-otp", loginHandler.MFALoginSendEmailOTPInternal)
	r.Post("/login/mfa/webauthn/begin", loginHandler.MFALoginWebAuthnBeginInternal)
}

// LoginMFAPublicRoute mounts public login-MFA endpoints. Public MFA accepts
// exactly one public authentication context: client_id or tenant_id.
func LoginMFAPublicRoute(r chi.Router, loginHandler *LoginHandler) {
	r.Post("/login/mfa/verify", loginHandler.MFALoginVerifyPublic)
	r.Post("/login/mfa/send-sms", loginHandler.MFALoginSendSMSPublic)
	r.Post("/login/mfa/send-email-otp", loginHandler.MFALoginSendEmailOTPPublic)
	r.Post("/login/mfa/webauthn/begin", loginHandler.MFALoginWebAuthnBeginPublic)
}

// MagicLinkPublicRoute mounts public magic-link routes.
// `send` requires client_id or tenant_id; `verify` extracts the context from the signed link.
// Mounted on the public surface (port 8081).
func MagicLinkPublicRoute(r chi.Router, magicLinkHandler *MagicLinkHandler) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))
		r.Use(publicAuthSurface)

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

// RegisterPublicRoute handles public register routes (requires client_id or tenant_id)
func RegisterPublicRoute(r chi.Router, registerHandler *RegisterHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))
		r.Use(publicAuthSurface)

		// Public registration (with client_id or tenant_id)
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

// ResetPasswordPublicRoute handles public reset password routes (signed client_id or tenant_id)
func ResetPasswordPublicRoute(r chi.Router, resetPasswordHandler *ResetPasswordHandler) {
	// Apply stricter limits for auth endpoints (inherits global security middleware)
	r.Group(func(r chi.Router) {
		// Stricter request size limit for auth endpoints (1MB vs 10MB global)
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))

		// Stricter timeout for auth operations (30s vs 60s global)
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))
		r.Use(publicAuthSurface)

		// Public reset password (with signed client_id or tenant_id)
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
// for the public surface (port 8081). Requires client_id or tenant_id.
func SMSLoginPublicRoute(
	r chi.Router,
	smsLoginHandler *SMSLoginHandler,
) {
	r.Route("/sms-login", func(r chi.Router) {
		r.Use(publicAuthSurface)
		// Send OTP to phone number (unauthenticated, client-scoped)
		r.Post("/send", smsLoginHandler.SendOTPPublic)
		// Verify OTP and obtain tokens (unauthenticated, client-scoped)
		r.Post("/verify", smsLoginHandler.VerifyOTPPublic)
	})
}

// RegistrationContextPublicRoute mounts the public signup-requirements read.
// Same unauthenticated public-auth surface as RegisterPublicRoute, so client
// resolution behaves identically on both.
func RegistrationContextPublicRoute(r chi.Router, handler *RegistrationContextHandler) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024))
		r.Use(middleware.TimeoutMiddleware(30 * time.Second))
		r.Use(publicAuthSurface)

		r.Get("/registration_context", handler.GetPublic)
	})
}
