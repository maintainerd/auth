package user

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
)

func UserTrustedDeviceRoute(
	r chi.Router,
	deviceHandler *UserTrustedDeviceHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/me", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"account:user:read:self"})).
			Get("/devices", deviceHandler.ListMyDevices)

		r.With(middleware.PermissionMiddleware([]string{"account:user:update:self"})).
			Delete("/devices/{device_uuid}", deviceHandler.DeleteMyDevice)
	})
}

// sensitiveActionStepUp is a policy-aware step-up middleware (provided by the
// mfa package). It gates sensitive identity changes (email change) on a fresh
// step-up only when the tenant policy require_mfa_for_sensitive_actions is
// enabled AND the user has an enrolled MFA factor; otherwise it is a pass-
// through. When nil, the strict middleware.RequireStepUp is used as a safe
// fallback (preserving prior behavior).
func AccountRoute(
	r chi.Router,
	accountHandler *AccountHandler,
	consentHandler *UserConsentHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	sensitiveActionStepUp func(http.Handler) http.Handler,
	rateLimitMiddleware ...middleware.Middleware,
) {
	if sensitiveActionStepUp == nil {
		sensitiveActionStepUp = middleware.RequireStepUp
	}
	r.Route("/account", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.Get("/", accountHandler.GetAccount)

		// Email change flow — updating the account's sign-in identity. Gated on a
		// policy-aware step-up (see sensitiveActionStepUp).
		r.With(middleware.PermissionMiddleware([]string{"account:user:update:self"}), sensitiveActionStepUp).
			Post("/email/change", accountHandler.InitiateEmailChange)
		r.With(middleware.PermissionMiddleware([]string{"account:user:update:self"}), sensitiveActionStepUp).
			Post("/email/verify", accountHandler.VerifyEmailChange)

		// Phone verification flow — the user proves ownership of their own phone
		// number (like MFA SMS enroll, no step-up required).
		r.With(middleware.PermissionMiddleware([]string{"account:user:update:self"})).
			Post("/phone/send-verification", accountHandler.SendPhoneVerification)
		r.With(middleware.PermissionMiddleware([]string{"account:user:update:self"})).
			Post("/phone/verify", accountHandler.VerifyPhone)

		// Username change.
		//
		// Gated on the policy-aware step-up, NOT the strict
		// middleware.RequireStepUp it used to carry. Strict step-up hard-requires
		// acr=2, which a password-only user can never reach — changing your
		// username was simply impossible for them, and the SPA could only report
		// "No second factor is available on your account." The body's
		// current_password requirement is what carries the proof-of-knowledge
		// here, exactly as it does for /password below.
		r.With(middleware.PermissionMiddleware([]string{"account:user:update:self"}), sensitiveActionStepUp).
			Put("/username", accountHandler.ChangeUsername)

		// Password change (self-service rotation).
		//
		// Gated on the policy-aware step-up, NOT the strict middleware.RequireStepUp
		// used by /username and DELETE /account. Strict step-up demands acr=2 within
		// the step-up TTL, which a password-only user can never satisfy — it would
		// make password rotation impossible for exactly the users who have no second
		// factor to fall back on. Post-compromise rotation is the one flow that must
		// not be blockable. The body's current_password requirement is what carries
		// the proof-of-knowledge here.
		r.With(middleware.PermissionMiddleware([]string{"account:change-password:self"}), sensitiveActionStepUp).
			Put("/password", accountHandler.ChangePassword)

		// Account deletion
		r.With(middleware.PermissionMiddleware([]string{"account:user:delete:self"}), middleware.RequireStepUp).
			Delete("/", accountHandler.DeleteAccount)

		// Account data export (GDPR / data portability)
		r.With(middleware.PermissionMiddleware([]string{"account:user:read:self"})).
			Get("/export", accountHandler.ExportAccountData)

		// Backup codes live ONLY on POST /mfa/backup-codes/regenerate, which is what
		// both SPAs call. The POST /account/backup-codes that used to sit here was a
		// second generator with no caller, and it was not merely redundant: it minted
		// a hardcoded 10 codes without consulting the tenant's MFA policy, so a
		// tenant that forbids backup_code or configures a different
		// recovery_codes_count could have that decision bypassed by calling it.

		// Session management
		r.With(middleware.PermissionMiddleware([]string{"account:session:read:self"})).
			Get("/sessions", accountHandler.ListSessions)
		r.With(middleware.PermissionMiddleware([]string{"account:session:terminate:self"}), sensitiveActionStepUp).
			Delete("/sessions", accountHandler.RevokeAllSessions)
		// "Sign out my other devices" — spares the caller's own session. A
		// distinct path rather than a flag on DELETE /sessions above, so the
		// destructive variant is never what you get from a dropped parameter.
		// chi matches the static segment ahead of {session_uuid}, and no session
		// UUID can spell "others".
		r.With(middleware.PermissionMiddleware([]string{"account:session:terminate:self"}), sensitiveActionStepUp).
			Delete("/sessions/others", accountHandler.RevokeOtherSessions)
		r.With(middleware.PermissionMiddleware([]string{"account:session:terminate:self"}), sensitiveActionStepUp).
			Delete("/sessions/{session_uuid}", accountHandler.RevokeSession)

		// Consent recording (self-service)
		if consentHandler != nil {
			r.With(middleware.PermissionMiddleware([]string{"account:user:update:self"})).
				Post("/consent", consentHandler.RecordConsent)
		}
	})
}

// AccountSelfReadRoute mounts the read-only slice of AccountRoute for the
// internal console surface.
//
// GET /account is what renders "signed in as", so it has to stay. The other
// thirteen endpoints on AccountRoute — email/phone/username/password changes,
// deletion, export, session management, consent — are account MANAGEMENT, and
// the identity app is the single place that does that now. Mounting them here
// as well would leave a second, unexercised copy of every one of those guards.
func AccountSelfReadRoute(
	r chi.Router,
	accountHandler *AccountHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/account", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.Get("/", accountHandler.GetAccount)
	})
}

// ProfileSelfReadRoute mounts the read-only slice of ProfileRoute for the
// internal console surface.
//
// Two endpoints, both of which the console needs to DISPLAY people:
//
//   - GET /profile — the signed-in admin's own profile (name and avatar in the
//     top nav).
//   - GET /profiles/{uuid}/picture — where an uploaded avatar is actually
//     served from. Every profile_url the console renders points here, including
//     on the admin user-management screens, so dropping it would break avatars
//     for other users too. The handler's own owner-or-user:read check is what
//     keeps that from becoming a way to enumerate profiles.
//
// Profile EDITING is absent: self-editing belongs to the identity app, and an
// admin editing someone else goes through /users/{uuid}/profiles on UserRoute.
func ProfileSelfReadRoute(
	r chi.Router,
	profileHandler *ProfileHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/profile", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"account:profile:read:self"})).
			Get("/", profileHandler.Get)
	})

	r.Route("/profiles", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		r.With(middleware.PermissionMiddleware([]string{"account:profile:read:self"})).
			Get("/{profile_uuid}/picture", profileHandler.GetPicture)
	})
}

// RecoveryRoute mounts unauthenticated account recovery endpoints.
func RecoveryRoute(
	r chi.Router,
	accountHandler *AccountHandler,
) {
	r.Route("/recovery", func(r chi.Router) {
		// Backup-code based account recovery (issues tokens)
		r.Post("/backup-code", accountHandler.VerifyBackupCode)
	})
}

func ProfileRoute(
	r chi.Router,
	profileHandler *ProfileHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	// /profile - Default profile operations (shortcut for convenience)
	r.Route("/profile", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		// Get default profile
		r.With(middleware.PermissionMiddleware([]string{"account:profile:read:self"})).
			Get("/", profileHandler.Get)

		// Create or update default profile (combined for convenience)
		r.With(middleware.PermissionMiddleware([]string{"account:profile:update:self"})).
			Post("/", profileHandler.CreateOrUpdate)

		// Update default profile
		r.With(middleware.PermissionMiddleware([]string{"account:profile:update:self"})).
			Put("/", profileHandler.CreateOrUpdate)

		// Delete default profile
		r.With(middleware.PermissionMiddleware([]string{"account:profile:delete:self"})).
			Delete("/", profileHandler.Delete)
	})

	// /profiles - All profiles operations (including default, with full CRUD)
	r.Route("/profiles", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		// Get all profiles with pagination and filtering
		r.With(middleware.PermissionMiddleware([]string{"account:profile:read:self"})).
			Get("/", profileHandler.GetAll)

		// Create new profile (auto-generate UUID)
		r.With(middleware.PermissionMiddleware([]string{"account:profile:update:self"})).
			Post("/", profileHandler.CreateProfile)

		// Get specific profile by UUID
		r.With(middleware.PermissionMiddleware([]string{"account:profile:read:self"})).
			Get("/{profile_uuid}", profileHandler.GetByUUID)

		// Update specific profile by UUID
		r.With(middleware.PermissionMiddleware([]string{"account:profile:update:self"})).
			Put("/{profile_uuid}", profileHandler.UpdateProfile)

		// Set a profile as the default/active profile
		r.With(middleware.PermissionMiddleware([]string{"account:profile:update:self"})).
			Put("/{profile_uuid}/set-default", profileHandler.SetDefaultProfile)

		// Delete specific profile by UUID
		r.With(middleware.PermissionMiddleware([]string{"account:profile:delete:self"})).
			Delete("/{profile_uuid}", profileHandler.DeleteByUUID)

		// Avatar upload/removal. Both write the caller's OWN profile — the service
		// refuses a UUID belonging to another user — so they need no permission
		// beyond the one that already governs editing a profile.
		r.With(middleware.PermissionMiddleware([]string{"account:profile:update:self"})).
			Post("/{profile_uuid}/picture", profileHandler.UploadPicture)
		r.With(middleware.PermissionMiddleware([]string{"account:profile:update:self"})).
			Delete("/{profile_uuid}/picture", profileHandler.DeletePicture)

		// Serving the image is a READ of profile data these endpoints already
		// return, so it carries the profile read permission and stays inside this
		// authenticated group: an open avatar endpoint would let anyone holding a
		// profile UUID confirm which ones exist.
		r.With(middleware.PermissionMiddleware([]string{"account:profile:read:self"})).
			Get("/{profile_uuid}/picture", profileHandler.GetPicture)
	})
}

func UserRoute(
	r chi.Router,
	userHandler *UserHandler,
	profileHandler *ProfileHandler,
	deviceHandler *UserTrustedDeviceHandler,
	consentHandler *UserConsentHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/users", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		// Get users with pagination and filtering
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/", userHandler.GetUsers)

		// Membership candidates (system-tenant users). Registered BEFORE
		// /{user_uuid} so chi does not treat the literal path as a UUID.
		r.With(middleware.PermissionMiddleware([]string{"tenant:member:create"})).
			Get("/membership-candidates", userHandler.ListMembershipCandidates)

		// Get user by UUID
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/{user_uuid}", userHandler.GetUser)

		// Create user
		r.With(middleware.PermissionMiddleware([]string{"user:create"})).
			Post("/", userHandler.CreateUser)

		// Update user.
		//
		// Step-up required. This endpoint rewrites the account's sign-in identity
		// (email, username) and its status, so it is at least as destructive as
		// PATCH /status and DELETE below, both of which already demand it. Without
		// it, a stolen acr=1 admin session could repoint a victim account at an
		// attacker-controlled inbox and take it over through forgot-password.
		r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
			Put("/{user_uuid}", userHandler.UpdateUser)

		// Set a user's password administratively (the operator remedy for a user
		// locked out of both their password and their inbox — force-password-change
		// alone does nothing until they sign in).
		r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
			Put("/{user_uuid}/password", userHandler.SetUserPassword)

		// Set user status
		r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
			Patch("/{user_uuid}/status", userHandler.SetUserStatus)

		// Verify email (also marks account as completed)
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Patch("/{user_uuid}/verify-email", userHandler.VerifyEmail)

		// Verify phone
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Patch("/{user_uuid}/verify-phone", userHandler.VerifyPhone)

		// PATCH /{user_uuid}/complete-account is deliberately absent: no SPA ever
		// called it, and PATCH /{user_uuid}/verify-email already marks the account
		// completed as part of verifying. The capability itself is not lost — the
		// gRPC control plane still exposes CompleteAccount.

		// Delete user
		r.With(middleware.PermissionMiddleware([]string{"user:delete"}), middleware.RequireStepUp).
			Delete("/{user_uuid}", userHandler.DeleteUser)

		// Force password change on next login
		r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
			Put("/{user_uuid}/force-password-change", userHandler.ForcePasswordChange)

		// Role management
		// Get user roles
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/{user_uuid}/roles", userHandler.GetUserRoles)

		// Get user MFA configuration
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/{user_uuid}/mfa", userHandler.GetUserMFA)

		// Get user identities
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/{user_uuid}/identities", userHandler.GetUserIdentities)

		// Link an existing external (federated) identity to a user. The operator
		// remedy for a user who created a duplicate account through a new IdP;
		// self-service linking cannot help them once they have lost access to the
		// original account. Step-up: attaching a sub to an account grants whoever
		// controls that sub a way in.
		r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
			Post("/{user_uuid}/identities", userHandler.LinkUserIdentity)

		// Unlink an external (federated) identity from a user
		r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
			Delete("/{user_uuid}/identities/{identity_uuid}", userHandler.UnlinkUserIdentity)

		// Trusted devices
		if deviceHandler != nil {
			r.With(middleware.PermissionMiddleware([]string{"user:read"})).
				Get("/{user_uuid}/devices", deviceHandler.GetUserDevices)
			// Revoke a user's trusted device (admin)
			r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
				Delete("/{user_uuid}/devices/{device_uuid}", deviceHandler.DeleteUserDevice)
		}

		// Consents
		if consentHandler != nil {
			r.With(middleware.PermissionMiddleware([]string{"user:read"})).
				Get("/{user_uuid}/consents", consentHandler.GetUserConsents)
			// Withdraw a user's consent (admin) — GDPR right to withdraw
			r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
				Post("/{user_uuid}/consents/withdraw", consentHandler.WithdrawUserConsent)
		}

		// Session management
		// Get user active sessions
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/{user_uuid}/sessions", userHandler.GetUserSessions)

		// Revoke a single user session
		r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
			Delete("/{user_uuid}/sessions/{session_uuid}", userHandler.RevokeUserSession)

		// Revoke ALL of a user's sessions (force global sign-out)
		r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
			Delete("/{user_uuid}/sessions", userHandler.RevokeAllUserSessions)

		// Unlock a user's failed-login lockout (admin remediation)
		r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
			Post("/{user_uuid}/unlock", userHandler.UnlockUser)

		// Assign roles to user (managing a user's role assignments is a user:update)
		r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
			Post("/{user_uuid}/roles", userHandler.AssignRoles)

		// Remove role from user
		r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
			Delete("/{user_uuid}/roles/{role_uuid}", userHandler.RemoveRole)

		// Profile management (admin access to user profiles)
		// Get all profiles for a user
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/{user_uuid}/profiles", profileHandler.AdminGetAllProfiles)

		// Create new profile for a user
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Post("/{user_uuid}/profiles", profileHandler.AdminCreateProfile)

		// Get specific profile by UUID
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/{user_uuid}/profiles/{profile_uuid}", profileHandler.AdminGetProfile)

		// Update specific profile by UUID
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Put("/{user_uuid}/profiles/{profile_uuid}", profileHandler.AdminUpdateProfile)

		// Set specific profile as default (admin)
		// Delete specific profile by UUID
		r.With(middleware.PermissionMiddleware([]string{"user:delete"})).
			Delete("/{user_uuid}/profiles/{profile_uuid}", profileHandler.AdminDeleteProfile)
	})
}

// DataErasureAdminRoute mounts the admin GDPR erasure endpoint on the internal
// port (8080). It coexists with the /users subrouter registered by UserRoute.
func DataErasureAdminRoute(
	r chi.Router,
	handler *DataErasureHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		// Admin requests erasure of a target user's data (GDPR Art.17). Step-up
		// required — this is the most destructive action on a user.
		r.With(middleware.PermissionMiddleware([]string{"user:delete"}), middleware.RequireStepUp).
			Post("/users/{user_uuid}/erasure-requests", handler.RequestAdmin)
	})
}

// DataErasureSelfRoute mounts the self-service GDPR erasure endpoint under /me.
// It coexists with the /me subrouter registered by UserTrustedDeviceRoute.
func DataErasureSelfRoute(
	r chi.Router,
	handler *DataErasureHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		// Authenticated user requests erasure of their own account (GDPR Art.17).
		//
		// Step-up required, matching DELETE /account and the admin erasure endpoint.
		// This schedules an irreversible multi-table anonymisation of the caller's
		// account; carrying a weaker gate than the strictly less destructive
		// DELETE /account meant a hijacked acr=1 session could permanently destroy
		// the victim's account with a single unauthenticated-strength request.
		r.With(middleware.PermissionMiddleware([]string{"account:user:delete:self"}), middleware.RequireStepUp).
			Post("/me/erasure-request", handler.RequestSelf)
	})
}

func UserSettingRoute(
	r chi.Router,
	userSettingHandler *UserSettingHandler,
	userService middleware.UserContextProvider,
	appCache *cache.Cache,
	rateLimitMiddleware ...middleware.Middleware,
) {
	r.Route("/user-settings", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))
		r.Use(middleware.OptionalMiddleware(rateLimitMiddleware...))

		// Create or update user settings - requires settings update permission
		r.With(middleware.PermissionMiddleware([]string{"settings:update:self"})).
			Post("/", userSettingHandler.CreateOrUpdate)

		// Get user settings - requires settings read permission
		r.With(middleware.PermissionMiddleware([]string{"settings:read:self"})).
			Get("/", userSettingHandler.Get)

		// Delete user settings - requires settings update permission (since it's modifying settings)
		r.With(middleware.PermissionMiddleware([]string{"settings:update:self"})).
			Delete("/", userSettingHandler.Delete)
	})
}
