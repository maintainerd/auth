package event

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EventType represents a canonical integration event type in the catalog.
type EventType struct {
	EventTypeID   int64     `gorm:"column:event_type_id;primaryKey;autoIncrement" json:"event_type_id"`
	EventTypeUUID uuid.UUID `gorm:"column:event_type_uuid;type:uuid;uniqueIndex;not null" json:"event_type_uuid"`
	TenantID      int64     `gorm:"column:tenant_id;not null" json:"tenant_id"`
	Key           string    `gorm:"column:key;type:varchar(100);not null" json:"key"`
	Category      string    `gorm:"column:category;type:varchar(50);not null" json:"category"`
	Description   string    `gorm:"column:description;type:text" json:"description"`
	Version       int       `gorm:"column:version;not null;default:1" json:"version"`
	IsActive      bool      `gorm:"column:is_active;not null;default:true" json:"is_active"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (EventType) TableName() string { return "event_types" }

func (e *EventType) BeforeCreate(tx *gorm.DB) error {
	if e.EventTypeUUID == uuid.Nil {
		e.EventTypeUUID = uuid.New()
	}
	return nil
}
