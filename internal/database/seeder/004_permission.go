package seeder

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedPermissions(db *gorm.DB, tenantID, apiID int64) error {
	permissions := []model.Permission{
		// PUBLIC
		// All public permissions are automatically assigned to all users.
		// There may be changes on spefific routes that may no longer available a public in the future
		// Like an organization may no longer accept any more registration and etc.
		// Register
		newPermission("public:register", "Register new user", tenantID, apiID),
		newPermission("public:register:pre-check", "Check email/username availability", tenantID, apiID),

		// Login
		newPermission("public:login", "Login with username/email and password", tenantID, apiID),
		newPermission("public:login:mfa-challenge", "Submit MFA code (TOTP, WebAuthn)", tenantID, apiID),

		// Reset password
		newPermission("public:request-password-reset", "Send password reset link", tenantID, apiID),
		newPermission("public:reset-password", "Reset password using token", tenantID, apiID),

		// Oauth2
		newPermission("public:oauth2:redirect", "Redirect to identity provider (SSO login)", tenantID, apiID),
		newPermission("public:oauth2:callback", "Handle OAuth2/OIDC callback", tenantID, apiID),
		newPermission("public:oauth2:signup", "Auto-register via SSO", tenantID, apiID),

		// Captcha
		newPermission("public:captcha", "Get CAPTCHA token or image", tenantID, apiID),

		// Configs
		newPermission("public:config", "Return non-sensitive app config (branding, providers, etc.)", tenantID, apiID),
		newPermission("public:health", "Public service health check", tenantID, apiID),

		// PERSONAL PERMISSIONS
		// All personal permissions are automatically assigned to all users.
		// These permissions are for the users to be able to manage their own data
		// Account Permission
		newPermission("account:request-verify-email:self", "Request email verification", tenantID, apiID),
		newPermission("account:verify-email:self", "Verify email", tenantID, apiID),
		newPermission("account:request-verify-phone:self", "Request phone verification", tenantID, apiID),
		newPermission("account:verify-phone:self", "Verify phone", tenantID, apiID),
		newPermission("account:change-password:self", "Change password (requires old password)", tenantID, apiID),
		newPermission("account:mfa:enroll:self", "Enroll in MFA (TOTP/WebAuthn)", tenantID, apiID),
		newPermission("account:mfa:disable:self", "Disable MFA", tenantID, apiID),
		newPermission("account:mfa:verify:self", "Verify MFA challenge", tenantID, apiID),

		// Authentication
		newPermission("account:auth:logout:self", "Logout from current session", tenantID, apiID),
		newPermission("account:auth:refresh-token:self", "Refresh JWT using refresh token", tenantID, apiID),
		newPermission("account:session:terminate:self", "End own active sessions", tenantID, apiID),

		// Token Permissions
		newPermission("account:token:create:self", "Create API or personal access token", tenantID, apiID),
		newPermission("account:token:read:self", "List own tokens", tenantID, apiID),
		newPermission("account:token:revoke:self", "Revoke own token", tenantID, apiID),

		// User data Permissions
		newPermission("account:user:read:self", "Get own user data", tenantID, apiID),
		newPermission("account:user:update:self", "Update user info", tenantID, apiID),
		newPermission("account:user:delete:self", "Delete own account", tenantID, apiID),
		newPermission("account:user:disable:self", "Temporarily disable own account", tenantID, apiID),

		// Profile permissions
		newPermission("account:profile:read:self", "Get own profile data", tenantID, apiID),
		newPermission("account:profile:update:self", "Update profile info", tenantID, apiID),
		newPermission("account:profile:delete:self", "Delete own profile", tenantID, apiID),

		// Activity Logs
		newPermission("account:audit:read:self", "View own activity logs", tenantID, apiID),

		// STRICT PERMISSIONS
		// These are permissions are assigned only to speicif users that have elevated access
		// TENANT LEVEL ACCESS
		// Tenants
		newPermission("tenant:read", "Read tenants", tenantID, apiID),
		newPermission("tenant:create", "Create tenant", tenantID, apiID),
		newPermission("tenant:update", "Update tenant", tenantID, apiID),
		newPermission("tenant:delete", "Delete tenant", tenantID, apiID),

		// SERVICE LEVEL ACCESS
		// Services
		newPermission("service:read", "Read services", tenantID, apiID),
		newPermission("service:create", "Create service", tenantID, apiID),
		newPermission("service:update", "Update service", tenantID, apiID),
		newPermission("service:delete", "Delete service", tenantID, apiID),
		newPermission("service:policy:assign", "Assign policies to service", tenantID, apiID),
		newPermission("service:policy:remove", "Remove policies from service", tenantID, apiID),

		// Apis
		newPermission("api:read", "Read apis", tenantID, apiID),
		newPermission("api:create", "Create api", tenantID, apiID),
		newPermission("api:update", "Update api", tenantID, apiID),
		newPermission("api:delete", "Delete api", tenantID, apiID),

		// Permissions
		newPermission("permission:read", "Read permissions", tenantID, apiID),
		newPermission("permission:create", "Create permission", tenantID, apiID),
		newPermission("permission:update", "Update permission", tenantID, apiID),
		newPermission("permission:delete", "Delete permission", tenantID, apiID),

		// Policies
		newPermission("policy:read", "Read policies", tenantID, apiID),
		newPermission("policy:create", "Create policy", tenantID, apiID),
		newPermission("policy:update", "Update policy", tenantID, apiID),
		newPermission("policy:delete", "Delete policy", tenantID, apiID),

		// Service logs
		newPermission("service_log:read", "Read service logs", tenantID, apiID),
		newPermission("service_log:create", "Create service log", tenantID, apiID),
		newPermission("service_log:update", "Update service log", tenantID, apiID),
		newPermission("service_log:delete", "Delete service log", tenantID, apiID),

		// USER LEVEL ACCESS
		// Roles
		newPermission("role:read", "Read roles", tenantID, apiID),
		newPermission("role:create", "Create a new role", tenantID, apiID),
		newPermission("role:update", "Update role", tenantID, apiID),
		newPermission("role:delete", "Delete a role", tenantID, apiID),
		newPermission("role:assign", "Assign roles to users", tenantID, apiID),
		newPermission("role:permission:create", "Add permissions to role", tenantID, apiID),
		newPermission("role:permission:delete", "Remove permissions from role", tenantID, apiID),
		newPermission("role:restrict-super-admin", "Prevent elevation to critical roles", tenantID, apiID),

		// Identity Providers
		newPermission("idp:read", "Read identity providers", tenantID, apiID),
		newPermission("idp:create", "Create identity provider", tenantID, apiID),
		newPermission("idp:update", "Update identity provider", tenantID, apiID),
		newPermission("idp:delete", "Delete identity provider", tenantID, apiID),

		// Auth Clients
		newPermission("client:read", "Read auth clients", tenantID, apiID),
		newPermission("client:secret:read", "Get auth client secret", tenantID, apiID),
		newPermission("client:config:read", "Get auth client configurations", tenantID, apiID),
		newPermission("client:create", "Create auth client", tenantID, apiID),
		newPermission("client:update", "Update auth client", tenantID, apiID),
		newPermission("client:delete", "Delete auth client", tenantID, apiID),
		newPermission("client:uri:read", "Read auth client URIs", tenantID, apiID),
		newPermission("client:uri:create", "Create auth client URI", tenantID, apiID),
		newPermission("client:uri:update", "Update auth client URI", tenantID, apiID),
		newPermission("client:uri:delete", "Delete auth client URI", tenantID, apiID),

		// Auth Client API Management
		newPermission("client:api:read", "Read APIs assigned to auth client", tenantID, apiID),
		newPermission("client:api:create", "Add APIs to auth client", tenantID, apiID),
		newPermission("client:api:delete", "Remove APIs from auth client", tenantID, apiID),

		// Auth Client API Permissions
		newPermission("client:api:permission:read", "Read permissions for auth client API", tenantID, apiID),
		newPermission("client:api:permission:create", "Add permissions to auth client API", tenantID, apiID),
		newPermission("client:api:permission:delete", "Remove permissions from auth client API", tenantID, apiID),

		// API Keys
		newPermission("api_key:read", "Read API keys", tenantID, apiID),
		newPermission("api_key:config:read", "Get API key configuration", tenantID, apiID),
		newPermission("api_key:create", "Create API key", tenantID, apiID),
		newPermission("api_key:update", "Update API key and manage API/permission assignments", tenantID, apiID),
		newPermission("api_key:delete", "Delete API key", tenantID, apiID),

		// User Administration
		newPermission("user:read", "Read users", tenantID, apiID),
		newPermission("user:create", "Create user", tenantID, apiID),
		newPermission("user:update", "Update user", tenantID, apiID),
		newPermission("user:delete", "Delete user", tenantID, apiID),
		newPermission("user:disable", "Disable user", tenantID, apiID),
		newPermission("user:enable", "Re-enable user", tenantID, apiID),
		newPermission("user:role:assign", "Assign role to a user", tenantID, apiID),
		newPermission("user:role:remove", "Remove role from a user", tenantID, apiID),
		newPermission("user:invite", "Invite user via email", tenantID, apiID),

		// Auth Logs
		newPermission("auth_log:read", "Read auth logs", tenantID, apiID),
		newPermission("auth_log:create", "Create auth log", tenantID, apiID),
		newPermission("auth_log:update", "Update auth log", tenantID, apiID),
		newPermission("auth_log:delete", "Delete auth log", tenantID, apiID),

		// Signup Flows
		newPermission("signup-flow:read", "Read signup flows", tenantID, apiID),
		newPermission("signup-flow:create", "Create signup flow", tenantID, apiID),
		newPermission("signup-flow:update", "Update signup flow", tenantID, apiID),
		newPermission("signup-flow:delete", "Delete signup flow", tenantID, apiID),

		// Security Settings
		newPermission("security-setting:read", "Read security settings", tenantID, apiID),
		newPermission("security-setting:update", "Update security settings", tenantID, apiID),

		// IP Restriction Rules
		newPermission("ip-restriction-rule:read", "Read IP restriction rules", tenantID, apiID),
		newPermission("ip-restriction-rule:create", "Create IP restriction rule", tenantID, apiID),
		newPermission("ip-restriction-rule:update", "Update IP restriction rule", tenantID, apiID),
		newPermission("ip-restriction-rule:delete", "Delete IP restriction rule", tenantID, apiID),

		// Email Templates
		newPermission("email-template:read", "Read email templates", tenantID, apiID),
		newPermission("email-template:create", "Create email template", tenantID, apiID),
		newPermission("email-template:update", "Update email template", tenantID, apiID),
		newPermission("email-template:delete", "Delete email template", tenantID, apiID),

		// SMS Templates
		newPermission("sms-template:read", "Read SMS templates", tenantID, apiID),
		newPermission("sms-template:create", "Create SMS template", tenantID, apiID),
		newPermission("sms-template:update", "Update SMS template", tenantID, apiID),
		newPermission("sms-template:delete", "Delete SMS template", tenantID, apiID),

		// Login Templates
		newPermission("login-template:read", "Read login templates", tenantID, apiID),
		newPermission("login-template:create", "Create login template", tenantID, apiID),
		newPermission("login-template:update", "Update login template", tenantID, apiID),
		newPermission("login-template:delete", "Delete login template", tenantID, apiID),

		// OTHER PERMISSIONS
		// Email
		newPermission("email:read-config", "View email delivery config", tenantID, apiID),
		newPermission("email:update-config", "Edit SMTP/provider settings", tenantID, apiID),
		newPermission("email:template:update", "Customize templates", tenantID, apiID),
		newPermission("email:send-verification", "Trigger email verification", tenantID, apiID),
		newPermission("email:send-reset-password", "Trigger password reset email", tenantID, apiID),

		// Notifications
		newPermission("notification:read-settings", "Read user notification settings (e.g., enabled types, channels)", tenantID, apiID),
		newPermission("notification:update-settings", "Update preferences (e.g., disable email for logins)", tenantID, apiID),
		newPermission("notification:read-templates", "Read notification templates (admin only)", tenantID, apiID),
		newPermission("notification:update-templates", "Update notification templates and content (admin only)", tenantID, apiID),
		newPermission("notification:send:test", "Send a test notification (email, SMS, in-app)", tenantID, apiID),
		newPermission("notification:send:custom", "Trigger custom or manual notifications (e.g., broadcast, maintenance notice)", tenantID, apiID),
		newPermission("notification:read-log:self", "View notification history (e.g., email sent logs)", tenantID, apiID),
		newPermission("notification:read-log:any", "View notifications sent to other users (admin only)", tenantID, apiID),
		newPermission("notification:disable-channel", "Temporarily suppress delivery channels (e.g., pause email)", tenantID, apiID),
		newPermission("notification:unsubscribe", "Allow user to unsubscribe from optional comms (e.g., marketing)", tenantID, apiID),

		// User Settings
		newPermission("settings:read:self", "Read personal settings (e.g., theme, language, layout)", tenantID, apiID),
		newPermission("settings:update:self", "Update personal preferences", tenantID, apiID),
		newPermission("settings:read:default", "Read system defaults or fallbacks", tenantID, apiID),
		newPermission("settings:update-preferences", "Update stored preferences (e.g., time zone, date format)", tenantID, apiID),
		newPermission("settings:update-theme", "Change visual theme (e.g., dark/light)", tenantID, apiID),
		newPermission("settings:update-language", "Change language or locale", tenantID, apiID),
		newPermission("settings:reset-self", "Reset user settings to defaults", tenantID, apiID),

		// Settings (Admin)
		newPermission("settings:read:any", "View another user's settings (for support tools)", tenantID, apiID),
		newPermission("settings:update:any", "Edit another user's preferences (admin)", tenantID, apiID),
		newPermission("settings:reset:any", "Reset settings for another user", tenantID, apiID),
		newPermission("notification:reset-templates", "Revert templates to default", tenantID, apiID),
		newPermission("notification:disable-system-wide", "Mute system-wide notifications (e.g., maintenance window)", tenantID, apiID),

		// Audit Logs & Monitoring
		newPermission("audit:read:any", "View audit logs for all users", tenantID, apiID),
		newPermission("audit:export", "Export logs for compliance", tenantID, apiID),
		newPermission("system:health-check", "System health metrics", tenantID, apiID),
		newPermission("system:metrics", "Service-level metrics", tenantID, apiID),
		newPermission("system:trace-events", "Debug/trace-level logs (dev only)", tenantID, apiID),

		// Security Policies
		newPermission("security:policy:read", "View MFA, password, session policies", tenantID, apiID),
		newPermission("security:policy:update", "Edit password rules, timeouts, etc.", tenantID, apiID),
		newPermission("security:rotate-keys", "Rotate signing/encryption keys", tenantID, apiID),
		newPermission("security:session:terminate:any", "Kill another user's session", tenantID, apiID),

		// System / Developer Tools
		newPermission("settings:read", "View system settings", tenantID, apiID),
		newPermission("settings:update", "Update runtime settings", tenantID, apiID),
		newPermission("system:reload-config", "Reload config files/env variables", tenantID, apiID),
		newPermission("system:run-migrations", "Apply database migrations", tenantID, apiID),
		newPermission("system:access-db-console", "DB shell/CLI access (dangerous)", tenantID, apiID),

		// Root-Level (Super Admin Only)
		newPermission("root:debug-mode", "Enable/disable debug mode", tenantID, apiID),
		newPermission("root:access-env", "View environment variables", tenantID, apiID),
		newPermission("root:impersonate", "Impersonate any user", tenantID, apiID),
		newPermission("root:hard-delete-user", "Irrecoverably delete user & data", tenantID, apiID),
	}

	for _, perm := range permissions {
		if permissionExists(db, perm.Name, tenantID) {
			log.Printf("⚠️ Permission '%s' already exists, skipping", perm.Name)
			continue
		}

		if err := db.Create(&perm).Error; err != nil {
			log.Printf("❌ Failed to seed permission '%s': %v", perm.Name, err)
			continue
		}

		log.Printf("✅ Permission '%s' seeded successfully", perm.Name)
	}

	return nil
}

func newPermission(name, description string, tenantID, apiID int64) model.Permission {
	return model.Permission{
		PermissionUUID: uuid.New(),
		TenantID:       tenantID,
		Name:           name,
		Description:    description,
		APIID:          apiID,
		Status:         "active",
		IsDefault:      true,
		IsSystem:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func permissionExists(db *gorm.DB, name string, tenantID int64) bool {
	var existing model.Permission
	err := db.Where("name = ? AND tenant_id = ?", name, tenantID).First(&existing).Error
	if err == nil {
		return true
	}
	if err != gorm.ErrRecordNotFound {
		log.Printf("❌ Error checking permission '%s': %v", name, err)
	}
	return false
}

