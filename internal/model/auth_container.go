package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthContainer struct {
	AuthContainerID   int64      `gorm:"column:auth_container_id;primaryKey"`
	AuthContainerUUID uuid.UUID  `gorm:"column:auth_container_uuid"`
	Name              string     `gorm:"column:name"`
	Description       string     `gorm:"column:description"`
	Identifier        string     `gorm:"column:identifier"`
	IsActive          bool       `gorm:"column:is_active"`
	IsDefault         bool       `gorm:"column:is_default"`
	OrganizationID    int64      `gorm:"column:organization_id"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         *time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Organization *Organization `gorm:"foreignKey:OrganizationID;references:OrganizationID"`
}

func (AuthContainer) TableName() string {
	return "auth_containers"
}

func (ac *AuthContainer) BeforeCreate(tx *gorm.DB) (err error) {
	if ac.AuthContainerUUID == uuid.Nil {
		ac.AuthContainerUUID = uuid.New()
	}
	return
}
