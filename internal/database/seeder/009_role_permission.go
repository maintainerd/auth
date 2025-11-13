package seeder

import (
	"log"
	"slices"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedRolePermissions(db *gorm.DB, roles map[string]model.Role) error {
	// Get all permissions
	var permissions []model.Permission
	if err := db.Find(&permissions).Error; err != nil {
		log.Printf("❌ Failed to fetch permissions: %v", err)
		return err
	}

	// Assign account permissions to registered role
	registeredRole, exists := roles["registered"]
	if exists {
		accountPermissions := []string{
			// Account permissions
			"account:request-verify-email:self",
			"account:verify-email:self",
			"account:request-verify-phone:self",
			"account:verify-phone:self",
			"account:change-password:self",
			"account:mfa:enroll:self",
			"account:mfa:disable:self",
			"account:mfa:verify:self",
			// Authentication
			"account:auth:logout:self",
			"account:auth:refresh-token:self",
			"account:session:terminate:self",
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
		}

		for _, permission := range permissions {
			// Check if this permission should be assigned to registered role
			if !slices.Contains(accountPermissions, permission.Name) {
				continue
			}

			var existing model.RolePermission
			err := db.
				Where("role_id = ? AND permission_id = ?", registeredRole.RoleID, permission.PermissionID).
				First(&existing).Error

			if err == nil {
				// Permission already assigned
				continue
			}

			if err != gorm.ErrRecordNotFound {
				log.Printf("❌ Error checking role permission: %v", err)
				continue
			}

			// Create new role permission
			rolePermission := model.RolePermission{
				RoleID:       registeredRole.RoleID,
				PermissionID: permission.PermissionID,
			}

			if err := db.Create(&rolePermission).Error; err != nil {
				log.Printf("❌ Failed to assign permission '%s' to role '%s': %v", permission.Name, registeredRole.Name, err)
				continue
			}
		}

		log.Printf("✅ Account permissions assigned to registered role")
	} else {
		log.Printf("⚠️ Registered role not found, skipping account permission assignment")
	}

	// Assign all permissions to super-admin role
	superAdminRole, exists := roles["super-admin"]
	if !exists {
		log.Printf("⚠️ Super-admin role not found, skipping permission assignment")
		return nil
	}

	for _, permission := range permissions {
		var existing model.RolePermission
		err := db.
			Where("role_id = ? AND permission_id = ?", superAdminRole.RoleID, permission.PermissionID).
			First(&existing).Error

		if err == nil {
			// Permission already assigned
			continue
		}

		if err != gorm.ErrRecordNotFound {
			log.Printf("❌ Error checking role permission: %v", err)
			continue
		}

		// Create new role permission
		rolePermission := model.RolePermission{
			RoleID:       superAdminRole.RoleID,
			PermissionID: permission.PermissionID,
		}

		if err := db.Create(&rolePermission).Error; err != nil {
			log.Printf("❌ Failed to assign permission '%s' to role '%s': %v", permission.Name, superAdminRole.Name, err)
			continue
		}
	}

	log.Printf("✅ All permissions assigned to super-admin role")
	return nil
}
