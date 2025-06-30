package defaultseeder

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

func SeedRoles(db *gorm.DB) {
	roles := []model.Role{
		{
			RoleUUID:    uuid.New(),
			Name:        "super-admin",
			Description: strPtr("Super Admin"),
			IsDefault:   true,
			CreatedAt:   time.Now(),
		},
		{
			RoleUUID:    uuid.New(),
			Name:        "admin",
			Description: strPtr("Admin"),
			IsDefault:   true,
			CreatedAt:   time.Now(),
		},
		{
			RoleUUID:    uuid.New(),
			Name:        "registered",
			Description: strPtr("Registered User"),
			IsDefault:   true,
			CreatedAt:   time.Now(),
		},
	}

	for _, role := range roles {
		var existing model.Role
		if err := db.Where("name = ?", role.Name).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&role).Error; err != nil {
					log.Printf("❌ Failed to seed role %s: %v", role.Name, err)
				} else {
					log.Printf("✅ Role %s seeded successfully", role.Name)
				}
			} else {
				log.Printf("❌ Error checking role %s: %v", role.Name, err)
			}
		} else {
			log.Printf("⚠️ Role %s already exists, skipping", role.Name)
		}
	}
}

func strPtr(s string) *string {
	return &s
}
