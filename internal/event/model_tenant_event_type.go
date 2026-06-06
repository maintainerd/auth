package event

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantEventType represents the per-tenant master switch for an event type.
// Absence of a row means enabled (default-on); only deliberate "off" overrides are stored.
type TenantEventType struct {
	TenantEventTypeID   int64     `gorm:"column:tenant_event_type_id;primaryKey;autoIncrement" json:"tenant_event_type_id"`
	TenantEventTypeUUID uuid.UUID `gorm:"column:tenant_event_type_uuid;type:uuid;uniqueIndex;not null" json:"tenant_event_type_uuid"`
	TenantID            int64     `gorm:"column:tenant_id;not null" json:"tenant_id"`
	EventTypeID         int64     `gorm:"column:event_type_id;not null" json:"event_type_id"`
	Enabled             bool      `gorm:"column:enabled;not null;default:true" json:"enabled"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (TenantEventType) TableName() string { return "tenant_event_types" }

func (t *TenantEventType) BeforeCreate(tx *gorm.DB) error {
	if t.TenantEventTypeUUID == uuid.Nil {
		t.TenantEventTypeUUID = uuid.New()
	}
	return nil
}
