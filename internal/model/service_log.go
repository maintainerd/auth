package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ServiceLog struct {
	ServiceLogID   int64          `gorm:"column:service_log_id;primaryKey"`
	ServiceLogUUID uuid.UUID      `gorm:"column:service_log_uuid;unique"`
	ServiceID      int64          `gorm:"column:service_id"`
	Level          string         `gorm:"column:level"`
	Message        string         `gorm:"column:message"`
	Metadata       datatypes.JSON `gorm:"column:metadata"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime"`

	// Relationships
	Service *Service `gorm:"foreignKey:ServiceID;references:ServiceID"`
}

func (ServiceLog) TableName() string {
	return "service_logs"
}

func (sl *ServiceLog) BeforeCreate(tx *gorm.DB) (err error) {
	if sl.ServiceLogUUID == uuid.Nil {
		sl.ServiceLogUUID = uuid.New()
	}
	return
}
