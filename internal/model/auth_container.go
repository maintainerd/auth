package model

import (
	"time"

	"github.com/google/uuid"
)

type AuthContainer struct {
	AuthContainerID   int64      `gorm:"column:auth_container_id;primaryKey"`
	AuthContainerUUID uuid.UUID  `gorm:"column:auth_container_uuid;type:uuid;not null;unique;index:idx_auth_containers_auth_container_uuid"`
	Name              string     `gorm:"column:name;type:varchar(255);not null;index:idx_auth_containers_name"`
	Description       string     `gorm:"column:description;type:text;not null"`
	IsActive          bool       `gorm:"column:is_active;type:boolean;default:false;index:idx_auth_containers_is_active"`
	IsDefault         bool       `gorm:"column:is_default;type:boolean;default:false;index:idx_auth_containers_is_default"`
	OrganizationID    int64      `gorm:"column:organization_id;type:integer;not null"`
	CreatedAt         time.Time  `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt         *time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`

	// Optional: define relationship if Organization model is available
	Organization *Organization `gorm:"foreignKey:OrganizationID;references:OrganizationID;constraint:OnDelete:CASCADE"`
}

func (AuthContainer) TableName() string {
	return "auth_containers"
}
