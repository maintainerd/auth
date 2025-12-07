package model

import (
	"time"

	"github.com/google/uuid"
)

type ApiPermission struct {
	ApiPermissionID   int64     `gorm:"column:api_permission_id;primaryKey;autoIncrement"`
	ApiPermissionUUID uuid.UUID `gorm:"column:api_permission_uuid;type:uuid;not null;uniqueIndex"`
	APIID             int64     `gorm:"column:api_id;not null;index"`
	PermissionID      int64     `gorm:"column:permission_id;not null;index"`
	CreatedAt         time.Time `gorm:"column:created_at;default:now()"`

	// Relationships
	API        API        `gorm:"foreignKey:APIID;references:APIID"`
	Permission Permission `gorm:"foreignKey:PermissionID;references:PermissionID"`
}

func (ApiPermission) TableName() string {
	return "api_permissions"
}
