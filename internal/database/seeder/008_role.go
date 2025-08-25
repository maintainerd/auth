package seeder

import (
	"log"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedRoles(db *gorm.DB, authContainerID int64) (map[string]model.Role, error) {
	roles := []model.Role{
		newRole("super-admin", "Super Admin", authContainerID),
	}

	roleMap := make(map[string]model.Role)

	for _, r := range roles {
		var existing model.Role
		err := db.Where("name = ? AND auth_container_id = ?", r.Name, authContainerID).First(&existing).Error
		if err == nil {
			log.Printf("⚠️ Role '%s' already exists, skipping", r.Name)
			roleMap[r.Name] = existing
			continue
		}

		if err := db.Create(&r).Error; err != nil {
			return nil, err
		}

		log.Printf("✅ Role '%s' seeded", r.Name)
		roleMap[r.Name] = r
	}

	return roleMap, nil
}

func newRole(name, description string, authContainerID int64) model.Role {
	return model.Role{
		RoleUUID:        uuid.New(),
		Name:            name,
		Description:     description,
		IsDefault:       true,
		IsActive:        true,
		AuthContainerID: authContainerID,
	}
}
