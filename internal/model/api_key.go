package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type APIKey struct {
	APIKeyID    int64          `gorm:"column:api_key_id;primaryKey"`
	APIKeyUUID  uuid.UUID      `gorm:"column:api_key_uuid;unique"`
	TenantID    int64          `gorm:"column:tenant_id;not null"`
	Name        string         `gorm:"column:name"`
	Description string         `gorm:"column:description"`
	KeyHash     string         `gorm:"column:key_hash;unique"`
	KeyPrefix   string         `gorm:"column:key_prefix"`
	Config      datatypes.JSON `gorm:"column:config"`
	ExpiresAt   *time.Time     `gorm:"column:expires_at"`

	RateLimit *int      `gorm:"column:rate_limit"`
	Status    string    `gorm:"column:status;default:'active'"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	APIKeyApis []APIKeyApi `gorm:"foreignKey:APIKeyID;references:APIKeyID"`
}

func (APIKey) TableName() string {
	return "api_keys"
}

func (ak *APIKey) BeforeCreate(tx *gorm.DB) (err error) {
	if ak.APIKeyUUID == uuid.Nil {
		ak.APIKeyUUID = uuid.New()
	}
	return
}
