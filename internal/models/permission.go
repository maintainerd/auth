package models

import (
	"time"

	"github.com/google/uuid"
)

type Permission struct {
	PermissionID   int64      `gorm:"primaryKey;column:permission_id"`
	PermissionUUID uuid.UUID  `gorm:"type:uuid;not null;unique;column:permission_uuid;index:idx_permissions_permission_uuid"`
	Name           string     `gorm:"type:varchar(255);not null;unique;index:idx_permissions_name"`
	Description    *string    `gorm:"type:text"`
	IsDefault      bool       `gorm:"default:false"`
	CreatedAt      time.Time  `gorm:"autoCreateTime"`
	UpdatedAt      *time.Time `gorm:"autoUpdateTime"`
}

// TableName overrides the default table name
func (Permission) TableName() string {
	return "permissions"
}
