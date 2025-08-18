package seeder

import (
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedRolePermissions(db *gorm.DB, roles map[string]model.Role, authContainerID int64) {
	type assignment struct {
		Role        string
		Permissions []string
	}

	assignments := []assignment{
		{
			Role: "super-admin",
			Permissions: []string{
				"*",
			},
		},
		{
			Role: "admin",
			Permissions: []string{
				"user:*",
				"org:*",
				"role:*",
				"permission:*",
				"settings:*",
				"notification:*",
				"audit:*",
				"system:*",
				"idp:*",
				"email:*",
			},
		},
		{
			Role: "registered",
			Permissions: []string{
				"user:read:self",
				"user:update:self",
				"user:disable:self",
				"user:delete:self",
				"auth:*",
				"mfa:*",
				"token:*",
				"audit:read:self",
				"session:terminate:self",
				"settings:read:self",
				"settings:update:self",
				"notification:read-settings",
				"notification:update-settings",
				"notification:read-log:self",
			},
		},
	}

	for _, assign := range assignments {
		role, ok := roles[assign.Role]
		if !ok {
			log.Printf("❌ Role '%s' not found, skipping permission assignment", assign.Role)
			continue
		}

		var allPermissions []model.Permission

		for _, pattern := range assign.Permissions {
			var temp []model.Permission

			if pattern == "*" {
				db.Where("auth_container_id = ?", authContainerID).Find(&temp)
			} else if strings.HasSuffix(pattern, "*") {
				prefix := strings.TrimSuffix(pattern, "*")
				db.Where("name LIKE ? AND auth_container_id = ?", prefix+"%", authContainerID).Find(&temp)
			} else {
				db.Where("name = ? AND auth_container_id = ?", pattern, authContainerID).Find(&temp)
			}

			allPermissions = append(allPermissions, temp...)
		}

		// Insert only missing role-permissions
		for _, perm := range allPermissions {
			var existing model.RolePermission
			err := db.Where("role_id = ? AND permission_id = ?", role.RoleID, perm.PermissionID).
				First(&existing).Error

			if err == gorm.ErrRecordNotFound {
				rp := model.RolePermission{
					RoleID:             role.RoleID,
					PermissionID:       perm.PermissionID,
					RolePermissionUUID: uuid.New(),
				}
				if err := db.Create(&rp).Error; err != nil {
					log.Printf("❌ Failed to assign permission '%s' to role '%s': %v", perm.Name, role.Name, err)
				}
			} else if err != nil {
				log.Printf("❌ Failed to check existing permission '%s' for role '%s': %v", perm.Name, role.Name, err)
			}
		}

		log.Printf("✅ Assigned %d permission(s) to role '%s'", len(allPermissions), role.Name)
	}
}
