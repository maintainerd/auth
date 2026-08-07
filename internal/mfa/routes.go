package mfa

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
)

// MFAInternalRoute mounts MFA endpoints for the internal console surface
// (port 8080): the step-up ceremony plus admin remediation.
//
// Self-service ENROLLMENT is deliberately absent. Managing your own factors
// belongs to the identity app, which owns every account-management flow; the
// console duplicated those screens and so duplicated their routes. Two mounts
// of the same enrollment surface means two places a policy or step-up guard has
// to be kept correct, and only one of them is exercised. The step-up routes
// stay because the console needs them to satisfy its OWN sensitive actions
// (an admin elevating to acr=2), not to manage factors.
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

		mountStepUpMFARoutes(r, mfaHandler)
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

		mountStepUpMFARoutes(r, mfaHandler)
		mountSelfMFARoutes(r, mfaHandler)
	})
}

// mountStepUpMFARoutes mounts the step-up ceremony: reading which factors the
// caller holds, and using one to prove possession right now.
//
// Split out from enrollment because both surfaces need it. Any authenticated
// UI has to be able to answer a 403 step_up_required, whether or not it also
// lets you manage factors.
func mountStepUpMFARoutes(r chi.Router, mfaHandler *MFAHandler) {
	// Overall MFA status — which methods are enrolled and allowed, so a client
	// can present the right challenge.
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:read:self"})).
		Get("/status", mfaHandler.GetStatus)

	// WebAuthn passkey authentication (assertion ceremony for step-up).
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:verify:self"})).
		Post("/webauthn/auth/begin", mfaHandler.WebAuthnBeginAuthentication)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:verify:self"})).
		Post("/webauthn/auth/finish", mfaHandler.WebAuthnFinishAuthentication)

	// Step-up authentication.
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:verify:self"})).
		Post("/step-up/challenge", mfaHandler.IssueStepUpChallenge)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:verify:self"})).
		Post("/step-up/send-sms", mfaHandler.SendStepUpSMS)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:verify:self"})).
		Post("/step-up/send-email-otp", mfaHandler.SendStepUpEmailOTP)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:verify:self"})).
		Post("/step-up/verify", mfaHandler.VerifyStepUp)
}

// mountSelfMFARoutes mounts self-service factor MANAGEMENT — enrolling,
// disabling and recovering the caller's own factors. Public surface only; see
// MFAInternalRoute for why the console does not get these.
func mountSelfMFARoutes(r chi.Router, mfaHandler *MFAHandler) {
	// All routes below are self-service: they act only on the authenticated
	// user's own MFA, so each is gated by an "account:mfa:*:self" permission
	// (granted to the registered role). Destructive ones additionally require
	// the CALLER to hold a fresh step-up (RequireFreshStepUp) — an enrolled
	// factor on the account is not proof the caller possesses it. ADDITIVE ones
	// (enrolling a new factor) carry RequireStepUpForNewFactor, which applies the
	// same demand once the account already holds a factor: an attacker who
	// enrols their own authenticator on a hijacked acr=1 session can step up with
	// it and clear every gate above, so the additive side has to be shut too. The
	// first-ever enrollment stays open — there is nothing to step up with yet.

	// GET /status lives in mountStepUpMFARoutes — both surfaces need it.

	// TOTP
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"}), mfaHandler.RequireStepUpForNewFactor).
		Post("/totp/enroll", mfaHandler.BeginTOTPEnrollment)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"}), mfaHandler.RequireStepUpForNewFactor).
		Post("/totp/verify", mfaHandler.FinishTOTPEnrollment)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:disable:self"}), mfaHandler.RequireFreshStepUp).
		Delete("/totp", mfaHandler.DisableTOTP)

	// Backup codes
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:read:self"})).
		Get("/backup-codes/count", mfaHandler.GetBackupCodesCount)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"}), mfaHandler.RequireFreshStepUp).
		Post("/backup-codes/regenerate", mfaHandler.RegenerateBackupCodes)

	// WebAuthn passkey registration
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"}), mfaHandler.RequireStepUpForNewFactor).
		Post("/webauthn/register/begin", mfaHandler.WebAuthnBeginRegistration)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"}), mfaHandler.RequireStepUpForNewFactor).
		Post("/webauthn/register/finish", mfaHandler.WebAuthnFinishRegistration)

	// The WebAuthn assertion ceremony lives in mountStepUpMFARoutes.

	// WebAuthn credential management
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:disable:self"}), mfaHandler.RequireFreshStepUp).
		Delete("/webauthn/{credential_uuid}", mfaHandler.WebAuthnDeleteCredential)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:read:self"}), mfaHandler.RequireFreshStepUp).
		Get("/webauthn/{credential_uuid}/download", mfaHandler.WebAuthnDownloadCredential)

	// The step-up endpoints live in mountStepUpMFARoutes.

	// SMS MFA enrollment
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"}), mfaHandler.RequireStepUpForNewFactor).
		Post("/sms/enroll", mfaHandler.EnrollSMS)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"}), mfaHandler.RequireStepUpForNewFactor).
		Post("/sms/verify", mfaHandler.VerifySMS)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:disable:self"}), mfaHandler.RequireFreshStepUp).
		Delete("/sms", mfaHandler.DisableSMS)

	// Email OTP MFA enrollment
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"}), mfaHandler.RequireStepUpForNewFactor).
		Post("/email-otp/enroll", mfaHandler.EnrollEmailOTP)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"}), mfaHandler.RequireStepUpForNewFactor).
		Post("/email-otp/verify", mfaHandler.VerifyEmailOTP)
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:disable:self"}), mfaHandler.RequireFreshStepUp).
		Delete("/email-otp", mfaHandler.DisableEmailOTP)

	// Self-service — reset all of the caller's *own* MFA factors. The target is
	// always the authenticated user (no target param), so this can never touch
	// another account. Same fresh-step-up guard as the per-method self disables
	// above: this is the single most destructive MFA action a session can take.
	r.With(middleware.PermissionMiddleware([]string{"account:mfa:reset:self"}), mfaHandler.RequireFreshStepUp).
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
