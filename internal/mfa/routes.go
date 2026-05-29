package mfa

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

// MFARoute mounts all MFA-related endpoints under /mfa.
func MFARoute(
	r chi.Router,
	mfaHandler *MFAHandler,
	userService UserService,
	appCache *cache.Cache,
) {
	r.Route("/mfa", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// Overall MFA status
		r.Get("/status", mfaHandler.GetStatus)

		// TOTP
		r.Post("/totp/enroll", mfaHandler.BeginTOTPEnrollment)
		r.Post("/totp/verify", mfaHandler.FinishTOTPEnrollment)
		r.Delete("/totp", mfaHandler.DisableTOTP)

		// Backup codes
		r.Get("/backup-codes/count", mfaHandler.GetBackupCodesCount)
		r.Post("/backup-codes/regenerate", mfaHandler.RegenerateBackupCodes)

		// WebAuthn passkey registration
		r.Post("/webauthn/register/begin", mfaHandler.WebAuthnBeginRegistration)
		r.Post("/webauthn/register/finish", mfaHandler.WebAuthnFinishRegistration)

		// WebAuthn passkey authentication
		r.Post("/webauthn/auth/begin", mfaHandler.WebAuthnBeginAuthentication)
		r.Post("/webauthn/auth/finish", mfaHandler.WebAuthnFinishAuthentication)

		// WebAuthn credential management
		r.Delete("/webauthn/{credential_uuid}", mfaHandler.WebAuthnDeleteCredential)

		// Step-up authentication
		r.Post("/step-up/challenge", mfaHandler.IssueStepUpChallenge)
		r.Post("/step-up/verify", mfaHandler.VerifyStepUp)

		// Admin — reset another user's MFA
		r.Post("/admin/users/{user_uuid}/reset", mfaHandler.AdminResetMFA)
	})
}
