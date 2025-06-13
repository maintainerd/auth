package model

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	RoleID      int64      `gorm:"primaryKey;column:role_id"`
	RoleUUID    uuid.UUID  `gorm:"type:uuid;not null;unique;column:role_uuid;index:idx_roles_role_uuid"`
	Name        string     `gorm:"type:varchar(255);not null;unique;index:idx_roles_name"`
	Description *string    `gorm:"type:text"`
	IsDefault   bool       `gorm:"default:false"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   *time.Time `gorm:"autoUpdateTime"`
}

// TableName overrides the default table name
func (Role) TableName() string {
	return "roles"
}
