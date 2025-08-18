package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AuthLog struct {
	AuthLogID       int64          `gorm:"column:auth_log_id;primaryKey"`
	AuthLogUUID     uuid.UUID      `gorm:"column:auth_log_uuid"`
	UserID          int64          `gorm:"column:user_id"`
	EventType       string         `gorm:"column:event_type"`
	Description     *string        `gorm:"column:description"`
	IPAddress       *string        `gorm:"column:ip_address"`
	UserAgent       *string        `gorm:"column:user_agent"`
	Metadata        datatypes.JSON `gorm:"column:metadata"`
	AuthContainerID int64          `gorm:"column:auth_container_id"`
	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	User          *User          `gorm:"foreignKey:UserID;references:UserID"`
	AuthContainer *AuthContainer `gorm:"foreignKey:AuthContainerID;references:AuthContainerID"`
}

func (AuthLog) TableName() string {
	return "auth_logs"
}

func (al *AuthLog) BeforeCreate(tx *gorm.DB) (err error) {
	if al.AuthLogUUID == uuid.Nil {
		al.AuthLogUUID = uuid.New()
	}
	return
}
