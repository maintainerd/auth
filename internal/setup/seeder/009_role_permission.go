package seeder

import (
	"fmt"
	"log/slog"
	"slices"

	model "github.com/maintainerd/maintainerd-auth/internal/iam"
	"gorm.io/gorm"
)

// registeredAccountPermissions are the self-service / own-account permissions
// granted to the "registered" role. Every user gets the registered role by
// default (including admins), so these are intentionally EXCLUDED from the
// super-admin role to avoid duplicate grants — an admin already has them via
// registered. The registered role holds ONLY account-scoped permissions; it must
// never carry administrative permissions.
var registeredAccountPermissions = []string{
	// Account permissions
	"account:request-verify-email:self",
	"account:verify-email:self",
	"account:request-verify-phone:self",
	"account:verify-phone:self",
	"account:change-password:self",
	"account:mfa:read:self",
	"account:mfa:enroll:self",
	"account:mfa:disable:self",
	"account:mfa:verify:self",
	"account:mfa:reset:self",
	// Authentication
	"account:auth:logout:self",
	"account:auth:refresh-token:self",
	"account:session:read:self",
	"account:session:terminate:self",
	// Linked identities (SSO / federation)
	"account:identity:read:self",
	"account:identity:link:self",
	"account:identity:unlink:self",
	// Token permissions
	"account:token:create:self",
	"account:token:read:self",
	"account:token:revoke:self",
	// User data permissions
	"account:user:read:self",
	"account:user:update:self",
	"account:user:delete:self",
	"account:user:disable:self",
	// Profile permissions
	"account:profile:read:self",
	"account:profile:update:self",
	"account:profile:delete:self",
	// Activity logs
	"account:audit:read:self",
	// Personal settings
	"settings:read:self",
	"settings:update:self",
	// Notifications
	"notification:read-log:self",
}

func SeedRolePermissions(db *gorm.DB, roles map[string]model.Role) error {
	// Permissions are tenant-bound. Scope to the tenant of the roles being
	// seeded so a newly created tenant is granted ONLY its own permissions —
	// without this filter a second tenant would inherit every other tenant's
	// permissions (cross-tenant leak). All roles in the map share one tenant.
	var tenantID int64
	for _, r := range roles {
		tenantID = r.TenantID
		break
	}

	var permissions []model.Permission
	if err := db.Where("tenant_id = ?", tenantID).Find(&permissions).Error; err != nil {
		return fmt.Errorf("failed to fetch permissions: %w", err)
	}

	// Assign account (self-service) permissions to the registered role.
	if registeredRole, exists := roles["registered"]; exists {
		for _, permission := range permissions {
			if !slices.Contains(registeredAccountPermissions, permission.Name) {
				continue
			}
			if err := assignRolePermission(db, registeredRole, permission); err != nil {
				return err
			}
		}
		slog.Info("Account permissions assigned to registered role")
	} else {
		slog.Warn("Registered role not found, skipping account permission assignment")
	}

	// Assign administrative permissions to the super-admin role — i.e. every
	// permission EXCEPT the registered account/self permissions. Admins also hold
	// the registered role, so granting the self permissions here too would just be
	// a redundant, repeated set of rows.
	superAdminRole, exists := roles["super-admin"]
	if !exists {
		slog.Warn("Super-admin role not found, skipping permission assignment")
		return nil
	}

	for _, permission := range permissions {
		if slices.Contains(registeredAccountPermissions, permission.Name) {
			continue // already granted via the registered role every admin also has
		}
		if err := assignRolePermission(db, superAdminRole, permission); err != nil {
			return err
		}
	}

	slog.Info("Administrative permissions assigned to super-admin role (account/self permissions excluded)")
	return nil
}

// assignRolePermission idempotently links a permission to a role, skipping the
// insert when the row already exists.
func assignRolePermission(db *gorm.DB, role model.Role, permission model.Permission) error {
	var existing model.RolePermission
	err := db.
		Where("role_id = ? AND permission_id = ?", role.RoleID, permission.PermissionID).
		First(&existing).Error
	if err == nil {
		return nil // already assigned
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("error checking role permission %q for %q: %w", permission.Name, role.Name, err)
	}

	if err := db.Create(&model.RolePermission{
		RoleID:       role.RoleID,
		PermissionID: permission.PermissionID,
	}).Error; err != nil {
		return fmt.Errorf("failed to assign permission %q to role %q: %w", permission.Name, role.Name, err)
	}
	return nil
}
