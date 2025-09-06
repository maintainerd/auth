package seeder

import (
	"errors"
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
	}

	for _, assign := range assignments {
		role, ok := roles[assign.Role]
		if !ok {
			log.Printf("❌ Role '%s' not found, skipping permission assignment", assign.Role)
			continue
		}

		// Reload the role to get the generated RoleID
		if err := db.Where("role_uuid = ?", role.RoleUUID).First(&role).Error; err != nil {
			log.Printf("❌ Failed to reload role '%s': %v", assign.Role, err)
			continue
		}

		var allPermissions []model.Permission

		for _, pattern := range assign.Permissions {
			var temp []model.Permission
			var err error

			if pattern == "*" {
				err = db.Where("auth_container_id = ?", authContainerID).Find(&temp).Error
			} else if strings.HasSuffix(pattern, "*") {
				prefix := strings.TrimSuffix(pattern, "*")
				err = db.Where("name LIKE ? AND auth_container_id = ?", prefix+"%", authContainerID).Find(&temp).Error
			} else {
				err = db.Where("name = ? AND auth_container_id = ?", pattern, authContainerID).Find(&temp).Error
			}

			if err != nil {
				log.Printf("❌ Failed to fetch permissions for pattern '%s': %v", pattern, err)
				continue
			}

			allPermissions = append(allPermissions, temp...)
		}

		for _, perm := range allPermissions {
			// Reload permission to ensure PermissionID is populated
			if err := db.Where("permission_uuid = ?", perm.PermissionUUID).First(&perm).Error; err != nil {
				log.Printf("❌ Failed to reload permission '%s': %v", perm.Name, err)
				continue
			}

			var existing model.RolePermission
			err := db.Where("role_id = ? AND permission_id = ?", role.RoleID, perm.PermissionID).First(&existing).Error

			if errors.Is(err, gorm.ErrRecordNotFound) {
				rp := model.RolePermission{
					RoleID:             role.RoleID,
					PermissionID:       perm.PermissionID,
					RolePermissionUUID: uuid.New(),
				}
				if err := db.Create(&rp).Error; err != nil {
					log.Printf("❌ Failed to assign permission '%s' to role '%s': %v", perm.Name, role.Name, err)
				} else {
					log.Printf("✅ Assigned permission '%s' to role '%s'", perm.Name, role.Name)
				}
			} else if err != nil {
				log.Printf("❌ Failed to check existing permission '%s' for role '%s': %v", perm.Name, role.Name, err)
			}
		}
	}
}
