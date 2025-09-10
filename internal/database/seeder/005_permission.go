package seeder

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedPermissions(db *gorm.DB, apiID int64) {
	permissions := []model.Permission{
		// PUBLIC
		// All public permissions are automatically assigned to all users.
		// There may be changes on spefific routes that may no longer available a public in the future
		// Like an organization may no longer accept any more registration and etc.
		// Register
		newPermission("public:register", "Register new user", apiID),
		newPermission("public:register:pre-check", "Check email/username availability", apiID),

		// Login
		newPermission("public:login", "Login with username/email and password", apiID),
		newPermission("public:login:mfa-challenge", "Submit MFA code (TOTP, WebAuthn)", apiID),

		// Reset password
		newPermission("public:request-password-reset", "Send password reset link", apiID),
		newPermission("public:reset-password", "Reset password using token", apiID),

		// Oauth2
		newPermission("public:oauth2:redirect", "Redirect to identity provider (SSO login)", apiID),
		newPermission("public:oauth2:callback", "Handle OAuth2/OIDC callback", apiID),
		newPermission("public:oauth2:signup", "Auto-register via SSO", apiID),

		// Captcha
		newPermission("public:captcha", "Get CAPTCHA token or image", apiID),

		// Configs
		newPermission("public:config", "Return non-sensitive app config (branding, providers, etc.)", apiID),
		newPermission("public:health", "Public service health check", apiID),

		// PERSONAL PERMISSIONS
		// All personal permissions are automatically assigned to all users.
		// These permissions are for the users to be able to manage their own data
		// Account Permission
		newPermission("account:request-verify-email:self", "Request email verification", apiID),
		newPermission("account:verify-email:self", "Verify email", apiID),
		newPermission("account:request-verify-phone:self", "Request phone verification", apiID),
		newPermission("account:verify-phone:self", "Verify phone", apiID),
		newPermission("account:change-password:self", "Change password (requires old password)", apiID),
		newPermission("account:mfa:enroll:self", "Enroll in MFA (TOTP/WebAuthn)", apiID),
		newPermission("account:mfa:disable:self", "Disable MFA", apiID),
		newPermission("account:mfa:verify:self", "Verify MFA challenge", apiID),

		// Authentication
		newPermission("account:auth:logout:self", "Logout from current session", apiID),
		newPermission("account:auth:refresh-token:self", "Refresh JWT using refresh token", apiID),
		newPermission("account:session:terminate:self", "End own active sessions", apiID),

		// Token Permissions
		newPermission("account:token:create:self", "Create API or personal access token", apiID),
		newPermission("account:token:read:self", "List own tokens", apiID),
		newPermission("account:token:revoke:self", "Revoke own token", apiID),

		// User data Permissions
		newPermission("account:user:read:self", "Get own user data", apiID),
		newPermission("account:user:update:self", "Update user info", apiID),
		newPermission("account:user:delete:self", "Delete own account", apiID),
		newPermission("account:user:disable:self", "Temporarily disable own account", apiID),

		// Profile permissions
		newPermission("account:profile:read:self", "Get own profile data", apiID),
		newPermission("account:profile:update:self", "Update profile info", apiID),
		newPermission("account:profile:delete:self", "Delete own profile", apiID),

		// Activity Logs
		newPermission("account:audit:read:self", "View own activity logs", apiID),

		// STRICT PERMISSIONS
		// These are permissions are assigned only to speicif users that have elevated access
		// ORGANIZATION LEVEL ACCESS
		// Organization Management
		newPermission("organization:read", "List organizations", apiID),
		newPermission("organization:create", "Create organization", apiID),
		newPermission("organization:update", "Update organization", apiID),
		newPermission("organization:delete", "Delete organization", apiID),

		// SERVICE LEVEL ACCESS
		// Service Management
		newPermission("service:read", "List services", apiID),
		newPermission("service:create", "Create service", apiID),
		newPermission("service:update", "Update service", apiID),
		newPermission("service:delete", "Delete service", apiID),

		// Apis
		newPermission("api:read", "List apis", apiID),
		newPermission("api:create", "Create api", apiID),
		newPermission("api:update", "Update api", apiID),
		newPermission("api:delete", "Delete api", apiID),

		// Permissions
		newPermission("permission:read", "List permissions", apiID),
		newPermission("permission:create", "Create permission", apiID),
		newPermission("permission:update", "Update permission", apiID),
		newPermission("permission:delete", "Delete permission", apiID),

		// USER LEVEL ACCESS
		// Auth Container Management
		newPermission("auth_container:read", "List auth containers", apiID),
		newPermission("auth_container:create", "Create auth container", apiID),
		newPermission("auth_container:update", "Update auth container", apiID),
		newPermission("auth_container:delete", "Delete auth container", apiID),

		// Roles
		newPermission("role:read", "List roles", apiID),
		newPermission("role:create", "Create a new role", apiID),
		newPermission("role:update", "Update role", apiID),
		newPermission("role:delete", "Delete a role", apiID),
		newPermission("role:assign", "Assign roles to users", apiID),
		newPermission("role:restrict-super-admin", "Prevent elevation to critical roles", apiID),

		// User Administration
		newPermission("user:read:any", "View any user profile", apiID),
		newPermission("user:update:any", "Edit user details", apiID),
		newPermission("user:delete:any", "Delete any user", apiID),
		newPermission("user:disable:any", "Disable any user", apiID),
		newPermission("user:enable:any", "Re-enable any user", apiID),
		newPermission("user:assign-role", "Assign role to a user", apiID),
		newPermission("user:remove-role", "Remove role from a user", apiID),
		newPermission("user:impersonate", "Temporarily act as another user", apiID),
		newPermission("user:invite", "Invite user via email", apiID),

		// Identity Provider
		newPermission("idp:read", "View identity provider config", apiID),
		newPermission("idp:create", "Add identity provider", apiID),
		newPermission("idp:update", "Update IdP settings", apiID),
		newPermission("idp:delete", "Remove identity provider", apiID),
		newPermission("idp:restrict-issuer", "Limit specific domains or providers", apiID),

		// Email
		newPermission("email:read-config", "View email delivery config", apiID),
		newPermission("email:update-config", "Edit SMTP/provider settings", apiID),
		newPermission("email:template:update", "Customize templates", apiID),
		newPermission("email:send-verification", "Trigger email verification", apiID),
		newPermission("email:send-reset-password", "Trigger password reset email", apiID),

		// Notifications
		newPermission("notification:read-settings", "Read user notification settings (e.g., enabled types, channels)", apiID),
		newPermission("notification:update-settings", "Update preferences (e.g., disable email for logins)", apiID),
		newPermission("notification:read-templates", "Read notification templates (admin only)", apiID),
		newPermission("notification:update-templates", "Update notification templates and content (admin only)", apiID),
		newPermission("notification:send:test", "Send a test notification (email, SMS, in-app)", apiID),
		newPermission("notification:send:custom", "Trigger custom or manual notifications (e.g., broadcast, maintenance notice)", apiID),
		newPermission("notification:read-log:self", "View notification history (e.g., email sent logs)", apiID),
		newPermission("notification:read-log:any", "View notifications sent to other users (admin only)", apiID),
		newPermission("notification:disable-channel", "Temporarily suppress delivery channels (e.g., pause email)", apiID),
		newPermission("notification:unsubscribe", "Allow user to unsubscribe from optional comms (e.g., marketing)", apiID),

		// User Settings
		newPermission("settings:read:self", "Read personal settings (e.g., theme, language, layout)", apiID),
		newPermission("settings:update:self", "Update personal preferences", apiID),
		newPermission("settings:read:default", "Read system defaults or fallbacks", apiID),
		newPermission("settings:update-preferences", "Update stored preferences (e.g., time zone, date format)", apiID),
		newPermission("settings:update-theme", "Change visual theme (e.g., dark/light)", apiID),
		newPermission("settings:update-language", "Change language or locale", apiID),
		newPermission("settings:reset-self", "Reset user settings to defaults", apiID),

		// Settings (Admin)
		newPermission("settings:read:any", "View another user’s settings (for support tools)", apiID),
		newPermission("settings:update:any", "Edit another user’s preferences (admin)", apiID),
		newPermission("settings:reset:any", "Reset settings for another user", apiID),
		newPermission("notification:reset-templates", "Revert templates to default", apiID),
		newPermission("notification:disable-system-wide", "Mute system-wide notifications (e.g., maintenance window)", apiID),

		// Audit Logs & Monitoring
		newPermission("audit:read:any", "View audit logs for all users", apiID),
		newPermission("audit:export", "Export logs for compliance", apiID),
		newPermission("system:health-check", "System health metrics", apiID),
		newPermission("system:metrics", "Service-level metrics", apiID),
		newPermission("system:trace-events", "Debug/trace-level logs (dev only)", apiID),

		// Security Policies
		newPermission("security:policy:read", "View MFA, password, session policies", apiID),
		newPermission("security:policy:update", "Edit password rules, timeouts, etc.", apiID),
		newPermission("security:rotate-keys", "Rotate signing/encryption keys", apiID),
		newPermission("security:session:terminate:any", "Kill another user’s session", apiID),

		// System / Developer Tools
		newPermission("settings:read", "View system settings", apiID),
		newPermission("settings:update", "Update runtime settings", apiID),
		newPermission("system:reload-config	", "Reload config files/env variables", apiID),
		newPermission("system:run-migrations", "Apply database migrations", apiID),
		newPermission("system:access-db-console", "DB shell/CLI access (dangerous)", apiID),

		// Root-Level (Super Admin Only)
		newPermission("root:debug-mode", "Enable/disable debug mode", apiID),
		newPermission("root:access-env", "View environment variables", apiID),
		newPermission("root:impersonate", "Impersonate any user", apiID),
		newPermission("root:hard-delete-user", "Irrecoverably delete user & data", apiID),

		// Compliance-Only Permissions
		newPermission("compliance:export-user-data", "Export user data (GDPR)", apiID),
		newPermission("compliance:delete-user-data", "Delete user data permanently", apiID),
		newPermission("compliance:request-data-access", "Request access log or report", apiID),
	}

	for _, perm := range permissions {
		if permissionExists(db, perm.Name) {
			log.Printf("⚠️ Permission '%s' already exists, skipping", perm.Name)
			continue
		}

		if err := db.Create(&perm).Error; err != nil {
			log.Printf("❌ Failed to seed permission '%s': %v", perm.Name, err)
			continue
		}

		log.Printf("✅ Permission '%s' seeded successfully", perm.Name)
	}
}

func newPermission(name, description string, apiID int64) model.Permission {
	return model.Permission{
		PermissionUUID: uuid.New(),
		Name:           name,
		Description:    description,
		APIID:          apiID,
		IsActive:       true,
		IsDefault:      true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func permissionExists(db *gorm.DB, name string) bool {
	var existing model.Permission
	err := db.Where("name = ?", name).First(&existing).Error
	if err == nil {
		return true
	}
	if err != gorm.ErrRecordNotFound {
		log.Printf("❌ Error checking permission '%s': %v", name, err)
	}
	return false
}
