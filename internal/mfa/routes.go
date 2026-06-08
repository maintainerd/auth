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
	userService middleware.UserContextProvider,
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
		r.With(mfaHandler.RequireStepUpOrEnrolledMFA).Delete("/totp", mfaHandler.DisableTOTP)

		// Backup codes
		r.Get("/backup-codes/count", mfaHandler.GetBackupCodesCount)
		r.With(mfaHandler.RequireStepUpOrEnrolledMFA).Post("/backup-codes/regenerate", mfaHandler.RegenerateBackupCodes)

		// WebAuthn passkey registration
		r.Post("/webauthn/register/begin", mfaHandler.WebAuthnBeginRegistration)
		r.Post("/webauthn/register/finish", mfaHandler.WebAuthnFinishRegistration)

		// WebAuthn passkey authentication
		r.Post("/webauthn/auth/begin", mfaHandler.WebAuthnBeginAuthentication)
		r.Post("/webauthn/auth/finish", mfaHandler.WebAuthnFinishAuthentication)

		// WebAuthn credential management
		r.With(mfaHandler.RequireStepUpOrEnrolledMFA).Delete("/webauthn/{credential_uuid}", mfaHandler.WebAuthnDeleteCredential)
		r.With(mfaHandler.RequireStepUpOrEnrolledMFA).Get("/webauthn/{credential_uuid}/download", mfaHandler.WebAuthnDownloadCredential)

		// Step-up authentication
		r.Post("/step-up/challenge", mfaHandler.IssueStepUpChallenge)
		r.Post("/step-up/send-sms", mfaHandler.SendStepUpSMS)
		r.Post("/step-up/verify", mfaHandler.VerifyStepUp)

		// SMS MFA enrollment
		r.Post("/sms/enroll", mfaHandler.EnrollSMS)
		r.Post("/sms/verify", mfaHandler.VerifySMS)
		r.With(mfaHandler.RequireStepUpOrEnrolledMFA).Delete("/sms", mfaHandler.DisableSMS)

		// Admin — reset another user's MFA. This affects a *different* user, so it
		// keeps the strict step-up requirement (acr=2) rather than the relaxed
		// self-service guard.
		r.With(middleware.RequireStepUp).Post("/admin/users/{user_uuid}/reset", mfaHandler.AdminResetMFA)
	})
}
