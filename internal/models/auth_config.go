package models

import (
	"time"

	"github.com/google/uuid"
)

type AuthConfig struct {
	AuthConfigID   int64      `gorm:"primaryKey;column:auth_config_id"`
	AuthConfigUUID uuid.UUID  `gorm:"type:uuid;not null;unique;column:auth_config_uuid"`
	Version        string     `gorm:"type:varchar(20);not null"`
	IsActive       bool       `gorm:"default:true"`
	IsApplied      bool       `gorm:"default:true"`
	CreatedAt      time.Time  `gorm:"autoCreateTime"`
	UpdatedAt      *time.Time `gorm:"autoUpdateTime"`
}

// TableName overrides the table name used by GORM
func (AuthConfig) TableName() string {
	return "auth_config"
}
