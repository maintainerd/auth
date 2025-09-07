package seeder

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedRolePermissions(db *gorm.DB, roles map[string]model.Role) error {
	superAdminRole, ok := roles["super-admin"]
	if !ok {
		return fmt.Errorf("super-admin role not found in roles map")
	}

	// Fetch all permissions
	var permissions []model.Permission
	if err := db.Find(&permissions).Error; err != nil {
		return err
	}

	for _, perm := range permissions {
		// Skip if already exists
		var existing model.RolePermission
		err := db.Where("role_id = ? AND permission_id = ?", superAdminRole.RoleID, perm.PermissionID).
			First(&existing).Error
		if err == nil {
			log.Printf("⚠️ Permission '%s' already assigned to role '%s', skipping", perm.Name, superAdminRole.Name)
			continue
		}

		rp := model.RolePermission{
			RolePermissionUUID: uuid.New(),
			RoleID:             superAdminRole.RoleID,
			PermissionID:       perm.PermissionID,
			CreatedAt:          time.Now(),
		}

		if err := db.Create(&rp).Error; err != nil {
			log.Printf("❌ Failed to assign permission '%s' to role '%s': %v", perm.Name, superAdminRole.Name, err)
			continue
		}

		log.Printf("✅ Assigned permission '%s' to role '%s'", perm.Name, superAdminRole.Name)
	}

	return nil
}
