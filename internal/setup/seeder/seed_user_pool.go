package seeder

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/user"
	"gorm.io/gorm"
)

func SeedUserPool(db *gorm.DB, tenantID int64) (*user.UserPool, error) {
	var existing user.UserPool
	err := db.Where("tenant_id = ? AND is_system = ?", tenantID, true).First(&existing).Error
	if err == nil {
		slog.Info("System user pool already exists, skipping", "user_pool_id", existing.UserPoolID)
		return &existing, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed lookup for system user pool: %w", err)
	}

	identifier, err := crypto.GenerateIdentifier(24)
	if err != nil {
		return nil, fmt.Errorf("failed to generate user pool identifier: %w", err)
	}

	pool := user.UserPool{
		UserPoolUUID: uuid.New(),
		TenantID:     tenantID,
		Name:         "system",
		DisplayName:  "System User Pool",
		Identifier:   identifier,
		IsSystem:     true,
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := db.Create(&pool).Error; err != nil {
		return nil, fmt.Errorf("failed to create system user pool: %w", err)
	}

	slog.Info("System user pool created", "user_pool_id", pool.UserPoolID)
	return &pool, nil
}
