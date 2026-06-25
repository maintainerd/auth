package mfa

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

// MFAInternalRoute mounts MFA endpoints for the internal console surface
// (port 8080). It includes both authenticated self-service MFA and admin
// remediation endpoints because this surface is private/VPN-scoped.
func MFAInternalRoute(
	r chi.Router,
	mfaHandler *MFAHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/mfa", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		mountSelfMFARoutes(r, mfaHandler)
		mountAdminMFARoutes(r, mfaHandler)
	})
}

// MFAPublicRoute mounts MFA endpoints for the public identity surface
// (port 8081). It intentionally excludes admin remediation routes.
func MFAPublicRoute(
	r chi.Router,
	mfaHandler *MFAHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/mfa", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		mountSelfMFARoutes(r, mfaHandler)
	})
}

func mountSelfMFARoutes(r chi.Router, mfaHandler *MFAHandler) {
	// All routes below are self-service: they act only on the authenticated
	// user's own MFA, so each is gated by an "account:mfa:*:self" permission
	// (granted to the registered role). Destructive ones additionally require
	// step-up or proof of an enrolled factor.

	// Overall MFA status
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:read:self"})).
		Get("/status", mfaHandler.GetStatus)

	// TOTP
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"})).
		Post("/totp/enroll", mfaHandler.BeginTOTPEnrollment)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"})).
		Post("/totp/verify", mfaHandler.FinishTOTPEnrollment)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:disable:self"}), mfaHandler.RequireStepUpOrEnrolledMFA).
		Delete("/totp", mfaHandler.DisableTOTP)

	// Backup codes
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:read:self"})).
		Get("/backup-codes/count", mfaHandler.GetBackupCodesCount)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"}), mfaHandler.RequireStepUpOrEnrolledMFA).
		Post("/backup-codes/regenerate", mfaHandler.RegenerateBackupCodes)

	// WebAuthn passkey registration
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"})).
		Post("/webauthn/register/begin", mfaHandler.WebAuthnBeginRegistration)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"})).
		Post("/webauthn/register/finish", mfaHandler.WebAuthnFinishRegistration)

	// WebAuthn passkey authentication (assertion ceremony for step-up)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:verify:self"})).
		Post("/webauthn/auth/begin", mfaHandler.WebAuthnBeginAuthentication)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:verify:self"})).
		Post("/webauthn/auth/finish", mfaHandler.WebAuthnFinishAuthentication)

	// WebAuthn credential management
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:disable:self"}), mfaHandler.RequireStepUpOrEnrolledMFA).
		Delete("/webauthn/{credential_uuid}", mfaHandler.WebAuthnDeleteCredential)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:read:self"}), mfaHandler.RequireStepUpOrEnrolledMFA).
		Get("/webauthn/{credential_uuid}/download", mfaHandler.WebAuthnDownloadCredential)

	// Step-up authentication
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:verify:self"})).
		Post("/step-up/challenge", mfaHandler.IssueStepUpChallenge)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:verify:self"})).
		Post("/step-up/send-sms", mfaHandler.SendStepUpSMS)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:verify:self"})).
		Post("/step-up/send-email-otp", mfaHandler.SendStepUpEmailOTP)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:verify:self"})).
		Post("/step-up/verify", mfaHandler.VerifyStepUp)

	// SMS MFA enrollment
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"})).
		Post("/sms/enroll", mfaHandler.EnrollSMS)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"})).
		Post("/sms/verify", mfaHandler.VerifySMS)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:disable:self"}), mfaHandler.RequireStepUpOrEnrolledMFA).
		Delete("/sms", mfaHandler.DisableSMS)

	// Email OTP MFA enrollment
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"})).
		Post("/email-otp/enroll", mfaHandler.EnrollEmailOTP)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"})).
		Post("/email-otp/verify", mfaHandler.VerifyEmailOTP)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:disable:self"}), mfaHandler.RequireStepUpOrEnrolledMFA).
		Delete("/email-otp", mfaHandler.DisableEmailOTP)

	// Self-service — reset all of the caller's *own* MFA factors. The target is
	// always the authenticated user (no target param), so this can never touch
	// another account. Same relaxed guard as the per-method self disables above.
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:reset:self"}), mfaHandler.RequireStepUpOrEnrolledMFA).
		Post("/reset", mfaHandler.SelfResetMFA)
}

func mountAdminMFARoutes(r chi.Router, mfaHandler *MFAHandler) {
	// Admin — reset another user's MFA. This affects a *different* user, so it
	// requires the "user:mfa:reset" permission and keeps the strict step-up
	// requirement (acr=2) rather than the relaxed self-service guard. Self
	// resets use the per-method self endpoints above (DELETE /totp, /sms,
	// /webauthn/{id}), which operate only on the authenticated user.
	r.With(
		middleware.PermissionMiddleware([]string{"user:mfa:reset"}),
		middleware.RequireStepUp,
	).Post("/admin/users/{user_uuid}/reset", mfaHandler.AdminResetMFA)
	// Reset a single factor (totp | webauthn | sms | backup_code) — e.g. wipe a
	// lost phone's TOTP/SMS while leaving a registered passkey intact.
	r.With(
		middleware.PermissionMiddleware([]string{"user:mfa:reset"}),
		middleware.RequireStepUp,
	).Post("/admin/users/{user_uuid}/reset/{method}", mfaHandler.AdminResetMFAMethod)
}
