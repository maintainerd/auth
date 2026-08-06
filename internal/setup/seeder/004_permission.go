package seeder

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	clientmodel "github.com/maintainerd/maintainerd-auth/internal/client"
	model "github.com/maintainerd/maintainerd-auth/internal/iam"
	"gorm.io/gorm"
)

var systemOnlyPermissions = []string{
	"tenant:create",
	"tenant:delete",
	// The signing-key surface is deployment-global, not tenant-scoped: Rotate
	// mints the tenant_id IS NULL key, and Retire/MarkCompromised resolve a bare
	// kid with no tenant predicate. Seeding this into an ordinary tenant would let
	// its admin disown the key every other tenant's tokens verify against.
	"security:rotate-keys",
}

func SeedPermissions(db *gorm.DB, tenantID, apiID int64) error {
	sysCheck := isSystemTenant(db, tenantID)
	permissions := defaultPermissions(tenantID, apiID)

	for _, perm := range permissions {
		if !isSeedableForTenant(perm.Name, sysCheck) {
			slog.Info("Skipping system-only permission for non-system tenant", "name", perm.Name, "tenant_id", tenantID)
			continue
		}

		exists, err := permissionExists(db, perm.Name, tenantID)
		if err != nil {
			return fmt.Errorf("failed to check permission %q: %w", perm.Name, err)
		}
		if exists {
			slog.Info("Permission already exists, skipping", "name", perm.Name)
			continue
		}

		if err := db.Create(&perm).Error; err != nil {
			return fmt.Errorf("failed to seed permission %q: %w", perm.Name, err)
		}

		slog.Info("Permission seeded", "name", perm.Name)
	}

	return pruneRetiredPermissions(db, tenantID, sysCheck)
}

// isSeedableForTenant reports whether this tenant is entitled to the name.
// Seeding and pruning both route through it so the two cannot drift: a name the
// seeder refuses to create for a tenant is a name the prune must remove from it,
// not one the prune quietly treats as legitimate.
func isSeedableForTenant(name string, systemTenant bool) bool {
	return systemTenant || !slices.Contains(systemOnlyPermissions, name)
}

// retainedPermissionNames is the catalog as it applies to one tenant: exactly
// the names SeedPermissions would create for it, and therefore exactly the names
// pruneRetiredPermissions must leave alone. The tenant/api IDs are irrelevant
// here — only the names are.
func retainedPermissionNames(systemTenant bool) []string {
	catalog := defaultPermissions(0, 0)
	names := make([]string, 0, len(catalog))
	for _, perm := range catalog {
		if !isSeedableForTenant(perm.Name, systemTenant) {
			continue
		}
		names = append(names, perm.Name)
	}
	return names
}

// pruneRetiredPermissions soft-deletes the seeded permissions a tenant still
// holds that defaultPermissions no longer defines, after detaching them from
// every role and client grant.
//
// The catalog is not append-only — names are retired whenever the guard they
// described is deleted, renamed, or folded into a coarser one — but creation
// alone never converges: a tenant bootstrapped by an older build keeps the
// retired rows forever. That is the same false-assurance failure the catalog
// shrink was meant to fix, only invisible: the console still lists
// root:impersonate or audit:export, an administrator still composes a role out
// of them, and the grant still authorises nothing. Seeding without pruning fixes
// only databases that do not exist yet.
//
// Scope is deliberately narrow and one-directional. Only is_system rows are
// touched, so permissions an operator minted through permission:create survive;
// and the step only ever removes access, so a partial run cannot widen anyone's
// authority.
func pruneRetiredPermissions(db *gorm.DB, tenantID int64, systemTenant bool) error {
	var retired []model.Permission
	if err := db.
		Where("tenant_id = ? AND is_system = ? AND name NOT IN ?", tenantID, true, retainedPermissionNames(systemTenant)).
		Find(&retired).Error; err != nil {
		return fmt.Errorf("failed to list retired permissions: %w", err)
	}
	if len(retired) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(retired))
	names := make([]string, 0, len(retired))
	for _, perm := range retired {
		ids = append(ids, perm.PermissionID)
		names = append(names, perm.Name)
	}

	// Detach before deleting. The permission row is soft-deleted, so the
	// ON DELETE CASCADE on role_permissions.permission_id and
	// client_permissions.permission_id never fires; leaving the join rows behind
	// would keep a retired name inside a role's membership and inside a client's
	// granted API permissions, which is exactly the grant being withdrawn.
	if err := db.Where("permission_id IN ?", ids).Delete(&model.RolePermission{}).Error; err != nil {
		return fmt.Errorf("failed to detach retired permissions from roles: %w", err)
	}
	if err := db.Where("permission_id IN ?", ids).Delete(&clientmodel.ClientPermission{}).Error; err != nil {
		return fmt.Errorf("failed to detach retired permissions from clients: %w", err)
	}
	if err := db.Where("permission_id IN ?", ids).Delete(&model.Permission{}).Error; err != nil {
		return fmt.Errorf("failed to prune retired permissions: %w", err)
	}

	slog.Warn("Retired permissions pruned", "tenant_id", tenantID, "count", len(names), "names", names)
	return nil
}

func isSystemTenant(db *gorm.DB, tenantID int64) bool {
	var result struct{ IsSystem bool }
	if err := db.Table("tenants").Select("is_system").Where("tenant_id = ?", tenantID).Scan(&result).Error; err != nil {
		return false
	}
	return result.IsSystem
}

// defaultPermissions is the seeded permission catalog.
//
// INVARIANT: every name here is enforced by a real guard — a
// PermissionMiddleware on an HTTP route or an entry in
// internal/server/grpc_permissions.go — and every guarded name is listed here.
// TestSeededPermissionsMatchEnforcedPermissions holds both directions.
//
// The catalog used to carry ~73 aspirational names (root:impersonate,
// user:disable, user:role:assign, security:session:terminate:any, audit:export,
// the whole notification:*/system:*/settings:{read,update,*:any}/public:*
// families …) for endpoints that were never built or that are actually guarded
// by a different name. Nothing rejected them, so an administrator could compose
// a role out of them, hand it to an operator, and ship a permission grant that
// authorised exactly nothing — the catalog was reporting an access-control
// surface the server did not have. A permission an auditor can read but the
// server never checks is worse than a missing one: it manufactures false
// assurance. Names are added here only once something enforces them.
//
// This list is authoritative in both directions: SeedPermissions creates what is
// missing and pruneRetiredPermissions withdraws what is no longer here, so a
// tenant's seeded catalog converges on it rather than accumulating whatever
// every past build once shipped.
func defaultPermissions(tenantID, apiID int64) []model.Permission {
	return []model.Permission{
		// PERSONAL PERMISSIONS
		// These permissions let a user manage their own data. They are granted to
		// the "registered" role every user holds (see 009_role_permission.go).
		// Account
		newPermission("account:change-password:self", "Change password (requires old password)", tenantID, apiID),
		newPermission("account:mfa:read:self", "Read own MFA status and factors", tenantID, apiID),
		newPermission("account:mfa:enroll:self", "Enroll in MFA (TOTP/WebAuthn)", tenantID, apiID),
		newPermission("account:mfa:disable:self", "Disable MFA", tenantID, apiID),
		newPermission("account:mfa:verify:self", "Verify MFA challenge", tenantID, apiID),
		newPermission("account:mfa:reset:self", "Reset own MFA (clear all own factors)", tenantID, apiID),

		// Sessions
		newPermission("account:session:read:self", "List own active sessions", tenantID, apiID),
		newPermission("account:session:terminate:self", "End own active sessions", tenantID, apiID),

		// Linked identities (SSO / federation) for own account
		newPermission("account:identity:read:self", "List own linked identities", tenantID, apiID),
		newPermission("account:identity:link:self", "Link a new identity to own account", tenantID, apiID),
		newPermission("account:identity:unlink:self", "Unlink an identity from own account", tenantID, apiID),

		// User data
		newPermission("account:user:read:self", "Get own user data", tenantID, apiID),
		newPermission("account:user:update:self", "Update user info", tenantID, apiID),
		newPermission("account:user:delete:self", "Delete own account", tenantID, apiID),

		// Profile
		newPermission("account:profile:read:self", "Get own profile data", tenantID, apiID),
		newPermission("account:profile:update:self", "Update profile info", tenantID, apiID),
		newPermission("account:profile:delete:self", "Delete own profile", tenantID, apiID),

		// Personal settings
		newPermission("settings:read:self", "Read personal settings (e.g., theme, language, layout)", tenantID, apiID),
		newPermission("settings:update:self", "Update personal preferences", tenantID, apiID),

		// STRICT PERMISSIONS
		// Management-plane access, assigned only to elevated roles.
		// TENANT LEVEL ACCESS
		// Tenants
		newPermission("tenant:read", "Read tenants", tenantID, apiID),
		newPermission("tenant:create", "Create tenant", tenantID, apiID),
		newPermission("tenant:update", "Update tenant", tenantID, apiID),
		newPermission("tenant:delete", "Delete tenant", tenantID, apiID),

		// Tenant members
		// Gates the membership-candidate picker the console reads before adding a
		// member; the write itself is a tenant:update.
		newPermission("tenant:member:create", "Browse candidates for tenant membership", tenantID, apiID),

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

		// USER LEVEL ACCESS
		// Roles
		newPermission("role:read", "Read roles", tenantID, apiID),
		newPermission("role:create", "Create a new role", tenantID, apiID),
		newPermission("role:update", "Update role", tenantID, apiID),
		newPermission("role:delete", "Delete a role", tenantID, apiID),
		newPermission("role:permission:create", "Add permissions to role", tenantID, apiID),
		newPermission("role:permission:delete", "Remove permissions from role", tenantID, apiID),

		// Identity Providers
		newPermission("idp:read", "Read identity providers", tenantID, apiID),
		newPermission("idp:create", "Create identity provider", tenantID, apiID),
		newPermission("idp:update", "Update identity provider", tenantID, apiID),
		newPermission("idp:delete", "Delete identity provider", tenantID, apiID),

		// Auth Clients
		newPermission("client:read", "Read auth clients", tenantID, apiID),
		newPermission("client:secret:read", "Get auth client secret", tenantID, apiID),
		newPermission("client:secret:rotate", "Rotate auth client secret", tenantID, apiID),
		newPermission("client:config:read", "Get auth client configurations", tenantID, apiID),
		newPermission("client:create", "Create auth client", tenantID, apiID),
		newPermission("client:update", "Update auth client", tenantID, apiID),
		newPermission("client:delete", "Delete auth client", tenantID, apiID),
		newPermission("client:uri:read", "Read auth client URIs", tenantID, apiID),
		newPermission("client:uri:create", "Create auth client URI", tenantID, apiID),
		newPermission("client:uri:update", "Update auth client URI", tenantID, apiID),
		newPermission("client:uri:delete", "Delete auth client URI", tenantID, apiID),

		// Auth Client Identity Provider Connections
		newPermission("client:identity_provider:read", "Read auth client identity provider connections", tenantID, apiID),
		newPermission("client:identity_provider:create", "Connect an identity provider to an auth client", tenantID, apiID),
		newPermission("client:identity_provider:update", "Update an auth client identity provider connection", tenantID, apiID),
		newPermission("client:identity_provider:delete", "Detach an identity provider from an auth client", tenantID, apiID),

		// Auth Client API Management
		newPermission("client:api:read", "Read APIs assigned to auth client", tenantID, apiID),
		newPermission("client:api:create", "Add APIs to auth client", tenantID, apiID),
		newPermission("client:api:delete", "Remove APIs from auth client", tenantID, apiID),

		// Auth Client API Permissions
		newPermission("client:api:permission:read", "Read permissions for auth client API", tenantID, apiID),
		newPermission("client:api:permission:create", "Add permissions to auth client API", tenantID, apiID),
		newPermission("client:api:permission:delete", "Remove permissions from auth client API", tenantID, apiID),

		// Auth Client Role Assignments
		newPermission("client:role:read", "Read roles assigned to a client", tenantID, apiID),
		newPermission("client:role:create", "Assign a role to a client", tenantID, apiID),
		newPermission("client:role:delete", "Remove a role from a client", tenantID, apiID),

		// User Administration
		// Enabling/disabling a user, assigning roles, revoking sessions, resetting
		// lockouts and unlinking identities are all guarded by user:update — there
		// is deliberately no finer-grained name for them, because none is enforced.
		newPermission("user:read", "Read users", tenantID, apiID),
		newPermission("user:create", "Create user", tenantID, apiID),
		newPermission("user:update", "Update user (status, roles, sessions, devices, identities)", tenantID, apiID),
		newPermission("user:delete", "Delete user", tenantID, apiID),
		newPermission("user:invite", "Invite user via email", tenantID, apiID),
		newPermission("user:mfa:reset", "Reset a user's MFA (all factors or a single method)", tenantID, apiID),

		// Auth Events (OWASP-compliant security event log)
		newPermission("auth_event:read", "Read and export auth events", tenantID, apiID),

		// Audit Logs
		newPermission("audit:read", "Read management audit logs", tenantID, apiID),

		// Registration flows (specialized registration presets)
		newPermission("registration-flow:read", "Read registration flows", tenantID, apiID),
		newPermission("registration-flow:create", "Create registration flow", tenantID, apiID),
		newPermission("registration-flow:update", "Update registration flow", tenantID, apiID),
		newPermission("registration-flow:delete", "Delete registration flow", tenantID, apiID),

		// Security Settings (password, MFA, lockout, session, token, threat policy)
		newPermission("security-setting:read", "Read security settings", tenantID, apiID),
		newPermission("security-setting:update", "Update security settings", tenantID, apiID),

		// Signing keys. Listed here only because the lifecycle endpoints now exist
		// and guard on this name (oauth.OAuthInternalRouteWithKeys); while the
		// guard existed and the row did not, list/rotate/retire/compromise 403'd
		// every role including super-admin.
		newPermission("security:rotate-keys", "List, rotate, retire and disown OAuth signing keys", tenantID, apiID),

		// IP Restriction Rules
		newPermission("ip-restriction-rule:read", "Read IP restriction rules", tenantID, apiID),
		newPermission("ip-restriction-rule:create", "Create IP restriction rule", tenantID, apiID),
		newPermission("ip-restriction-rule:update", "Update IP restriction rule", tenantID, apiID),
		newPermission("ip-restriction-rule:delete", "Delete IP restriction rule", tenantID, apiID),

		// Email Templates
		newPermission("email-template:read", "Read email templates", tenantID, apiID),
		newPermission("email-template:update", "Update email template", tenantID, apiID),

		// SMS Templates
		newPermission("sms-template:read", "Read SMS templates", tenantID, apiID),
		newPermission("sms-template:update", "Update SMS template", tenantID, apiID),

		// Branding
		newPermission("branding:read", "Read tenant branding", tenantID, apiID),
		newPermission("branding:create", "Create tenant branding", tenantID, apiID),
		newPermission("branding:update", "Update tenant branding", tenantID, apiID),
		newPermission("branding:delete", "Delete tenant branding", tenantID, apiID),
		newPermission("branding:activate", "Activate/deactivate tenant branding", tenantID, apiID),

		// Tenant Settings
		newPermission("tenant-setting:read", "Read tenant settings", tenantID, apiID),
		newPermission("tenant-setting:update", "Update tenant settings", tenantID, apiID),

		// Email Config
		newPermission("email-config:read", "Read email delivery configuration", tenantID, apiID),
		newPermission("email-config:update", "Update email delivery configuration", tenantID, apiID),

		// SMS Config
		newPermission("sms-config:read", "Read SMS delivery configuration", tenantID, apiID),
		newPermission("sms-config:update", "Update SMS delivery configuration", tenantID, apiID),

		// Webhook Endpoints
		newPermission("webhook-endpoint:read", "Read webhook endpoints", tenantID, apiID),
		newPermission("webhook-endpoint:create", "Create webhook endpoint", tenantID, apiID),
		newPermission("webhook-endpoint:update", "Update webhook endpoint", tenantID, apiID),
		newPermission("webhook-endpoint:delete", "Delete webhook endpoint", tenantID, apiID),

		// Workload Identity Federations (keyless external workload auth: K8s, GitHub Actions, GitLab CI)
		newPermission("workload-identity-federation:read", "Read workload identity federations", tenantID, apiID),
		newPermission("workload-identity-federation:create", "Create workload identity federation", tenantID, apiID),
		newPermission("workload-identity-federation:update", "Update workload identity federation", tenantID, apiID),
		newPermission("workload-identity-federation:delete", "Delete workload identity federation", tenantID, apiID),
	}
}

func newPermission(name, description string, tenantID, apiID int64) model.Permission {
	return model.Permission{
		PermissionUUID: uuid.New(),
		TenantID:       tenantID,
		Name:           name,
		Description:    description,
		APIID:          apiID,
		Status:         "active",
		IsSystem:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func permissionExists(db *gorm.DB, name string, tenantID int64) (bool, error) {
	var existing model.Permission
	err := db.Where("name = ? AND tenant_id = ?", name, tenantID).First(&existing).Error
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}
