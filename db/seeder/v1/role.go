package seeder

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedRoles(db *gorm.DB) {
	roles := []model.Role{
		newRole("super-admin", "Super Admin"),
		newRole("admin", "Admin"),
		newRole("registered", "Registered"),
	}

	for _, role := range roles {
		if roleExists(db, role.Name) {
			log.Printf("⚠️ Role '%s' already exists, skipping", role.Name)
			continue
		}
		if err := db.Create(&role).Error; err != nil {
			log.Printf("❌ Failed to seed role '%s': %v", role.Name, err)
			continue
		}
		log.Printf("✅ Role '%s' seeded successfully", role.Name)
	}
}

func newRole(name, description string) model.Role {
	return model.Role{
		RoleUUID:    uuid.New(),
		Name:        name,
		Description: strPtr(description),
		IsDefault:   true,
		CreatedAt:   time.Now(),
	}
}

func roleExists(db *gorm.DB, name string) bool {
	var existing model.Role
	err := db.Where("name = ?", name).First(&existing).Error

	if err == nil {
		return true
	}

	if err != gorm.ErrRecordNotFound {
		log.Printf("❌ Error checking role '%s': %v", name, err)
	}

	return false
}

func strPtr(s string) *string {
	return &s
}
