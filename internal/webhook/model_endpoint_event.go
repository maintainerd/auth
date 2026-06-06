package webhook

import (
	"time"

	"gorm.io/gorm"
)

// WebhookEndpointEvent represents an M:N subscription — one webhook endpoint
// subscribed to one event type.
type WebhookEndpointEvent struct {
	WebhookEndpointEventID int64     `gorm:"column:webhook_endpoint_event_id;primaryKey;autoIncrement" json:"webhook_endpoint_event_id"`
	WebhookEndpointID      int64     `gorm:"column:webhook_endpoint_id;not null" json:"webhook_endpoint_id"`
	EventTypeID            int64     `gorm:"column:event_type_id;not null" json:"event_type_id"`
	CreatedAt              time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (WebhookEndpointEvent) TableName() string { return "webhook_endpoint_events" }

func (w *WebhookEndpointEvent) BeforeCreate(tx *gorm.DB) error {
	return nil
}
