package seeder

import (
	"log"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedRoles(db *gorm.DB, authContainerID int64) (map[string]model.Role, error) {
	roles := map[string]model.Role{
		"super-admin": {
			RoleUUID:        uuid.New(),
			Name:            "super-admin",
			Description:     "Full system access with all permissions",
			AuthContainerID: authContainerID,
			IsActive:        true,
			IsDefault:       true,
		},
	}

	roleMap := make(map[string]model.Role)

	for roleName, role := range roles {
		var existing model.Role
		err := db.
			Where("name = ? AND auth_container_id = ?", role.Name, authContainerID).
			First(&existing).Error

		if err == nil {
			log.Printf("⚠️ Role '%s' already exists (ID: %d)", role.Name, existing.RoleID)
			roleMap[roleName] = existing
			continue
		}

		if err != gorm.ErrRecordNotFound {
			log.Printf("❌ Error checking role '%s': %v", role.Name, err)
			return nil, err
		}

		if err := db.Create(&role).Error; err != nil {
			log.Printf("❌ Failed to seed role '%s': %v", role.Name, err)
			return nil, err
		}

		log.Printf("✅ Role '%s' seeded successfully (ID: %d)", role.Name, role.RoleID)
		roleMap[roleName] = role
	}

	return roleMap, nil
}
