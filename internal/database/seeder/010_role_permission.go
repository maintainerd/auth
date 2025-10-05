package seeder

import (
	"log"

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
