package event

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EventRoute represents a per-tenant broker (RabbitMQ) routing configuration.
type EventRoute struct {
	EventRouteID   int64     `gorm:"column:event_route_id;primaryKey;autoIncrement" json:"event_route_id"`
	EventRouteUUID uuid.UUID `gorm:"column:event_route_uuid;type:uuid;uniqueIndex;not null" json:"event_route_uuid"`
	TenantID       int64     `gorm:"column:tenant_id;not null" json:"tenant_id"`
	EventTypeID    int64     `gorm:"column:event_type_id;not null" json:"event_type_id"`
	Channel        string    `gorm:"column:channel;type:varchar(50);not null;default:'rabbitmq'" json:"channel"`
	Destination    string    `gorm:"column:destination;type:varchar(255);not null" json:"destination"`
	Enabled        bool      `gorm:"column:enabled;not null;default:true" json:"enabled"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (EventRoute) TableName() string { return "event_routes" }

func (er *EventRoute) BeforeCreate(tx *gorm.DB) error {
	if er.EventRouteUUID == uuid.Nil {
		er.EventRouteUUID = uuid.New()
	}
	return nil
}
