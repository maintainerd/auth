package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AuthLog struct {
	AuthLogID       int64          `gorm:"column:auth_log_id;primaryKey"`
	AuthLogUUID     uuid.UUID      `gorm:"column:auth_log_uuid;type:uuid;not null;unique;index:idx_auth_logs_auth_log_uuid"`
	UserID          int64          `gorm:"column:user_id;type:integer;not null;index:idx_auth_logs_user_id"`
	EventType       string         `gorm:"column:event_type;type:varchar(100);not null;index:idx_auth_logs_event_type"`
	Description     *string        `gorm:"column:description;type:text"`
	IPAddress       *string        `gorm:"column:ip_address;type:varchar(100)"`
	UserAgent       *string        `gorm:"column:user_agent;type:text"`
	Metadata        datatypes.JSON `gorm:"column:metadata;type:jsonb"`
	AuthContainerID int64          `gorm:"column:auth_container_id;type:integer;not null;index:idx_auth_logs_auth_container_id"`
	CreatedAt       time.Time      `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`

	// Relationships
	User          *User          `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
	AuthContainer *AuthContainer `gorm:"foreignKey:AuthContainerID;references:AuthContainerID;constraint:OnDelete:CASCADE"`
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
