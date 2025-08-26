package seeder

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedPermissions(db *gorm.DB, apiID, authContainerID int64) {
	permissions := []model.Permission{
		// PUBLIC
		// All public permissions are automatically assigned to all users.
		// There may be changes on spefific routes that may no longer available a public in the future
		// Like an organization may no longer accept any more registration and etc.
		// Register
		newPermission("public:register", "Register new user", apiID, authContainerID),
		newPermission("public:register:pre-check", "Check email/username availability", apiID, authContainerID),
		// Login
		newPermission("public:login", "Login with username/email and password", apiID, authContainerID),
		newPermission("public:login:mfa-challenge", "Submit MFA code (TOTP, WebAuthn)", apiID, authContainerID),
		// Reset password
		newPermission("public:request-password-reset", "Send password reset link", apiID, authContainerID),
		newPermission("public:reset-password", "Reset password using token", apiID, authContainerID),
		// Oauth2
		newPermission("public:oauth2:redirect", "Redirect to identity provider (SSO login)", apiID, authContainerID),
		newPermission("public:oauth2:callback", "Handle OAuth2/OIDC callback", apiID, authContainerID),
		newPermission("public:oauth2:signup", "Auto-register via SSO", apiID, authContainerID),
		// Captcha
		newPermission("public:captcha", "Get CAPTCHA token or image", apiID, authContainerID),
		// Configs
		newPermission("public:config", "Return non-sensitive app config (branding, providers, etc.)", apiID, authContainerID),
		newPermission("public:health", "Public service health check", apiID, authContainerID),

		// PERSONAL PERMISSIONS
		// All personal permissions are automatically assigned to all users.
		// These permissions are for the users to be able to manage their own data
		// Account Permission
		newPermission("account:request-verify-email:self", "Request email verification", apiID, authContainerID),
		newPermission("account:verify-email:self", "Verify email", apiID, authContainerID),
		newPermission("account:request-verify-phone:self", "Request phone verification", apiID, authContainerID),
		newPermission("account:verify-phone:self", "Verify phone", apiID, authContainerID),
		newPermission("account:change-password:self", "Change password (requires old password)", apiID, authContainerID),
		newPermission("account:mfa:enroll:self", "Enroll in MFA (TOTP/WebAuthn)", apiID, authContainerID),
		newPermission("account:mfa:disable:self", "Disable MFA", apiID, authContainerID),
		newPermission("account:mfa:verify:self", "Verify MFA challenge", apiID, authContainerID),
		// Authentication
		newPermission("account:auth:logout:self", "Logout from current session", apiID, authContainerID),
		newPermission("account:auth:refresh-token:self", "Refresh JWT using refresh token", apiID, authContainerID),
		newPermission("account:session:terminate:self", "End own active sessions", apiID, authContainerID),
		// Token Permissions
		newPermission("account:token:create:self", "Create API or personal access token", apiID, authContainerID),
		newPermission("account:token:read:self", "List own tokens", apiID, authContainerID),
		newPermission("account:token:revoke:self", "Revoke own token", apiID, authContainerID),
		// User data Permissions
		newPermission("account:user:read:self", "Get own user data", apiID, authContainerID),
		newPermission("account:user:update:self", "Update user info", apiID, authContainerID),
		newPermission("account:user:delete:self", "Delete own account", apiID, authContainerID),
		newPermission("account:user:disable:self", "Temporarily disable own account", apiID, authContainerID),
		// Profile permissions
		newPermission("account:profile:read:self", "Get own profile data", apiID, authContainerID),
		newPermission("account:profile:update:self", "Update profile info", apiID, authContainerID),
		newPermission("account:profile:delete:self", "Delete own profile", apiID, authContainerID),
		// Activity Logs
		newPermission("account:audit:read:self", "View own activity logs", apiID, authContainerID),

		// STRICT PERMISSIONS
		// These are permissions are assigned only to speicif users that have elevated access
		// User Administration (Admin Only)
		newPermission("user:read:any", "View any user profile", apiID, authContainerID),
		newPermission("user:update:any", "Edit user details", apiID, authContainerID),
		newPermission("user:delete:any", "Delete any user", apiID, authContainerID),
		newPermission("user:disable:any", "Disable any user", apiID, authContainerID),
		newPermission("user:enable:any", "Re-enable any user", apiID, authContainerID),
		newPermission("user:assign-role", "Assign role to a user", apiID, authContainerID),
		newPermission("user:remove-role", "Remove role from a user", apiID, authContainerID),
		newPermission("user:impersonate", "Temporarily act as another user", apiID, authContainerID),
		newPermission("user:invite", "Invite user via email", apiID, authContainerID),
		// Roles
		newPermission("role:read", "List roles", apiID, authContainerID),
		newPermission("role:create", "Create a new role", apiID, authContainerID),
		newPermission("role:update", "Update role", apiID, authContainerID),
		newPermission("role:delete", "Delete a role", apiID, authContainerID),
		newPermission("role:assign", "Assign roles to users", apiID, authContainerID),
		newPermission("role:restrict-super-admin", "Prevent elevation to critical roles", apiID, authContainerID),
		// Permissions
		newPermission("permission:read", "List permissions", apiID, authContainerID),
		newPermission("permission:create", "Create a new permission", apiID, authContainerID),
		newPermission("permission:update", "Update permission", apiID, authContainerID),
		newPermission("permission:delete", "Delete a permission", apiID, authContainerID),
		newPermission("permission:assign", "Assign permission to users", apiID, authContainerID),
		newPermission("permission:restrict-super-admin", "Prevent elevation to critical permissions", apiID, authContainerID),
		// Organizations
		newPermission("org:create", "Create organization", apiID, authContainerID),
		newPermission("org:read", "View organization details", apiID, authContainerID),
		newPermission("org:update", "Update org info", apiID, authContainerID),
		newPermission("org:delete", "Delete organization", apiID, authContainerID),
		newPermission("org:invite-user", "Invite user to join org", apiID, authContainerID),
		newPermission("org:remove-user", "Remove user from org", apiID, authContainerID),
		newPermission("org:assign-owner", "Change organization ownership", apiID, authContainerID),
		newPermission("org:read-users", "List organization members", apiID, authContainerID),
		newPermission("org:update-role", "Modify member roles in org", apiID, authContainerID),
		// Identity Provider
		newPermission("idp:read", "View identity provider config", apiID, authContainerID),
		newPermission("idp:create", "Add identity provider", apiID, authContainerID),
		newPermission("idp:update", "Update IdP settings", apiID, authContainerID),
		newPermission("idp:delete", "Remove identity provider", apiID, authContainerID),
		newPermission("idp:restrict-issuer", "Limit specific domains or providers", apiID, authContainerID),
		// Email
		newPermission("email:read-config", "View email delivery config", apiID, authContainerID),
		newPermission("email:update-config", "Edit SMTP/provider settings", apiID, authContainerID),
		newPermission("email:template:update", "Customize templates", apiID, authContainerID),
		newPermission("email:send-verification", "Trigger email verification", apiID, authContainerID),
		newPermission("email:send-reset-password", "Trigger password reset email", apiID, authContainerID),
		// Notifications
		newPermission("notification:read-settings", "Read user notification settings (e.g., enabled types, channels)", apiID, authContainerID),
		newPermission("notification:update-settings", "Update preferences (e.g., disable email for logins)", apiID, authContainerID),
		newPermission("notification:read-templates", "Read notification templates (admin only)", apiID, authContainerID),
		newPermission("notification:update-templates", "Update notification templates and content (admin only)", apiID, authContainerID),
		newPermission("notification:send:test", "Send a test notification (email, SMS, in-app)", apiID, authContainerID),
		newPermission("notification:send:custom", "Trigger custom or manual notifications (e.g., broadcast, maintenance notice)", apiID, authContainerID),
		newPermission("notification:read-log:self", "View notification history (e.g., email sent logs)", apiID, authContainerID),
		newPermission("notification:read-log:any", "View notifications sent to other users (admin only)", apiID, authContainerID),
		newPermission("notification:disable-channel", "Temporarily suppress delivery channels (e.g., pause email)", apiID, authContainerID),
		newPermission("notification:unsubscribe", "Allow user to unsubscribe from optional comms (e.g., marketing)", apiID, authContainerID),
		// User Settings
		newPermission("settings:read:self", "Read personal settings (e.g., theme, language, layout)", apiID, authContainerID),
		newPermission("settings:update:self", "Update personal preferences", apiID, authContainerID),
		newPermission("settings:read:default", "Read system defaults or fallbacks", apiID, authContainerID),
		newPermission("settings:update-preferences", "Update stored preferences (e.g., time zone, date format)", apiID, authContainerID),
		newPermission("settings:update-theme", "Change visual theme (e.g., dark/light)", apiID, authContainerID),
		newPermission("settings:update-language", "Change language or locale", apiID, authContainerID),
		newPermission("settings:reset-self", "Reset user settings to defaults", apiID, authContainerID),
		// Settings (Admin)
		newPermission("settings:read:any", "View another user’s settings (for support tools)", apiID, authContainerID),
		newPermission("settings:update:any", "Edit another user’s preferences (admin)", apiID, authContainerID),
		newPermission("settings:reset:any", "Reset settings for another user", apiID, authContainerID),
		newPermission("notification:reset-templates", "Revert templates to default", apiID, authContainerID),
		newPermission("notification:disable-system-wide", "Mute system-wide notifications (e.g., maintenance window)", apiID, authContainerID),
		// Audit Logs & Monitoring
		newPermission("audit:read:any", "View audit logs for all users", apiID, authContainerID),
		newPermission("audit:export", "Export logs for compliance", apiID, authContainerID),
		newPermission("system:health-check", "System health metrics", apiID, authContainerID),
		newPermission("system:metrics", "Service-level metrics", apiID, authContainerID),
		newPermission("system:trace-events", "Debug/trace-level logs (dev only)", apiID, authContainerID),
		// Security Policies
		newPermission("security:policy:read", "View MFA, password, session policies", apiID, authContainerID),
		newPermission("security:policy:update", "Edit password rules, timeouts, etc.", apiID, authContainerID),
		newPermission("security:rotate-keys", "Rotate signing/encryption keys", apiID, authContainerID),
		newPermission("security:session:terminate:any", "Kill another user’s session", apiID, authContainerID),
		// System / Developer Tools
		newPermission("settings:read", "View system settings", apiID, authContainerID),
		newPermission("settings:update", "Update runtime settings", apiID, authContainerID),
		newPermission("system:reload-config	", "Reload config files/env variables", apiID, authContainerID),
		newPermission("system:run-migrations", "Apply database migrations", apiID, authContainerID),
		newPermission("system:access-db-console", "DB shell/CLI access (dangerous)", apiID, authContainerID),
		// Root-Level (Super Admin Only)
		newPermission("root:debug-mode", "Enable/disable debug mode", apiID, authContainerID),
		newPermission("root:access-env", "View environment variables", apiID, authContainerID),
		newPermission("root:impersonate", "Impersonate any user", apiID, authContainerID),
		newPermission("root:hard-delete-user", "Irrecoverably delete user & data", apiID, authContainerID),
		// Compliance-Only Permissions
		newPermission("compliance:export-user-data", "Export user data (GDPR)", apiID, authContainerID),
		newPermission("compliance:delete-user-data", "Delete user data permanently", apiID, authContainerID),
		newPermission("compliance:request-data-access", "Request access log or report", apiID, authContainerID),
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

func newPermission(name, description string, apiID, authContainerID int64) model.Permission {
	return model.Permission{
		PermissionUUID:  uuid.New(),
		Name:            name,
		Description:     description,
		IsActive:        true,
		IsDefault:       true,
		APIID:           apiID,
		AuthContainerID: authContainerID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
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
