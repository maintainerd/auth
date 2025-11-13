package seeder

import (
	"log"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedRoles(db *gorm.DB, tenantID int64) (map[string]model.Role, error) {
	// Define roles in order of creation (registered first, then super-admin)
	roleDefinitions := []struct {
		key  string
		role model.Role
	}{
		{
			key: "registered",
			role: model.Role{
				RoleUUID:    uuid.New(),
				Name:        "registered",
				Description: "Basic registered user with account access permissions",
				TenantID:    tenantID,
				IsActive:    true,
				IsDefault:   true, // registered is the default role assigned to all users
				IsSystem:    true, // registered is a system role
			},
		},
		{
			key: "super-admin",
			role: model.Role{
				RoleUUID:    uuid.New(),
				Name:        "super-admin",
				Description: "Full system access with all permissions",
				TenantID:    tenantID,
				IsActive:    true,
				IsDefault:   false, // super-admin is not the default role
				IsSystem:    true,  // super-admin is a system role
			},
		},
	}

	roleMap := make(map[string]model.Role)

	// Process roles in order to ensure registered is created first
	for _, roleDef := range roleDefinitions {
		var existing model.Role
		err := db.
			Where("name = ? AND tenant_id = ?", roleDef.role.Name, tenantID).
			First(&existing).Error

		if err == nil {
			log.Printf("⚠️ Role '%s' already exists (ID: %d)", roleDef.role.Name, existing.RoleID)
			roleMap[roleDef.key] = existing
			continue
		}

		if err != gorm.ErrRecordNotFound {
			log.Printf("❌ Error checking role '%s': %v", roleDef.role.Name, err)
			return nil, err
		}

		if err := db.Create(&roleDef.role).Error; err != nil {
			log.Printf("❌ Failed to seed role '%s': %v", roleDef.role.Name, err)
			return nil, err
		}

		log.Printf("✅ Role '%s' seeded successfully (ID: %d)", roleDef.role.Name, roleDef.role.RoleID)
		roleMap[roleDef.key] = roleDef.role
	}

	return roleMap, nil
}
