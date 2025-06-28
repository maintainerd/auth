package seeders

import (
	"log"
	"time"

	"github.com/maintainerd/auth/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func SeedRoles(db *gorm.DB) {
	roles := []models.Role{
		{
			RoleUUID:    uuid.New(),
			Name:        "Super Admin",
			Description: strPtr("System-wide administrator with full access"),
			IsDefault:   false,
			CreatedAt:   time.Now(),
		},
		{
			RoleUUID:    uuid.New(),
			Name:        "Admin",
			Description: strPtr("Administrator with elevated permissions"),
			IsDefault:   false,
			CreatedAt:   time.Now(),
		},
		{
			RoleUUID:    uuid.New(),
			Name:        "User",
			Description: strPtr("Standard user with limited access"),
			IsDefault:   true,
			CreatedAt:   time.Now(),
		},
	}

	for _, role := range roles {
		var existing models.Role
		// Check by Name to avoid duplicates
		if err := db.Where("name = ?", role.Name).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&role).Error; err != nil {
					log.Printf("❌ Failed to seed role %s: %v\n", role.Name, err)
				} else {
					log.Printf("✅ Role %s seeded successfully\n", role.Name)
				}
			} else {
				log.Printf("❌ Error checking role %s: %v\n", role.Name, err)
			}
		} else {
			log.Printf("⚠️ Role %s already exists, skipping\n", role.Name)
		}
	}
}

func strPtr(s string) *string {
	return &s
}
