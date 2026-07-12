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

		// Username change
		r.With(middleware.PermissionMiddleware([]string{"account:user:update:self"}), middleware.RequireStepUp).
			Put("/username", accountHandler.ChangeUsername)

		// Account deletion
		r.With(middleware.PermissionMiddleware([]string{"account:user:delete:self"}), middleware.RequireStepUp).
			Delete("/", accountHandler.DeleteAccount)

		// Account data export (GDPR / data portability)
		r.With(middleware.PermissionMiddleware([]string{"account:user:read:self"})).
			Get("/export", accountHandler.ExportAccountData)

		// Backup codes — generate and store securely (an MFA recovery factor).
		r.With(middleware.PermissionMiddleware([]string{"account:mfa:enroll:self"}), middleware.RequireStepUp).
			Post("/backup-codes", accountHandler.GenerateBackupCodes)

		// Session management
		r.With(middleware.PermissionMiddleware([]string{"account:session:read:self"})).
			Get("/sessions", accountHandler.ListSessions)
		r.With(middleware.PermissionMiddleware([]string{"account:session:terminate:self"}), sensitiveActionStepUp).
			Delete("/sessions", accountHandler.RevokeAllSessions)
		r.With(middleware.PermissionMiddleware([]string{"account:session:terminate:self"}), sensitiveActionStepUp).
			Delete("/sessions/{session_uuid}", accountHandler.RevokeSession)

		// Consent recording (self-service)
		if consentHandler != nil {
			r.With(middleware.PermissionMiddleware([]string{"account:user:update:self"})).
				Post("/consent", consentHandler.RecordConsent)
		}
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

		// Get user by UUID
		r.With(middleware.PermissionMiddleware([]string{"user:read"})).
			Get("/{user_uuid}", userHandler.GetUser)

		// Create user
		r.With(middleware.PermissionMiddleware([]string{"user:create"})).
			Post("/", userHandler.CreateUser)

		// Update user
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Put("/{user_uuid}", userHandler.UpdateUser)

		// Set user status
		r.With(middleware.PermissionMiddleware([]string{"user:update"}), middleware.RequireStepUp).
			Patch("/{user_uuid}/status", userHandler.SetUserStatus)

		// Verify email (also marks account as completed)
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Patch("/{user_uuid}/verify-email", userHandler.VerifyEmail)

		// Verify phone
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Patch("/{user_uuid}/verify-phone", userHandler.VerifyPhone)

		// Mark account as completed
		r.With(middleware.PermissionMiddleware([]string{"user:update"})).
			Patch("/{user_uuid}/complete-account", userHandler.CompleteAccount)

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
		r.With(middleware.PermissionMiddleware([]string{"account:user:delete:self"})).
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
