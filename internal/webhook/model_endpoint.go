package webhook

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// WebhookEndpoint represents an outbound event notification subscription
// belonging to a tenant. Multiple endpoints may exist per tenant, each
// subscribing to a different set of events via webhook_endpoint_events.
type WebhookEndpoint struct {
	WebhookEndpointID   int64          `gorm:"column:webhook_endpoint_id;primaryKey;autoIncrement" json:"-"`
	WebhookEndpointUUID uuid.UUID      `gorm:"column:webhook_endpoint_uuid;type:uuid;uniqueIndex;not null" json:"webhook_endpoint_id"`
	TenantID            int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	URL                 string         `gorm:"column:url;type:text;not null" json:"url"`
	SecretEncrypted     string         `gorm:"column:secret_encrypted;type:text" json:"-"`
	SubscribeAll        bool           `gorm:"column:subscribe_all;not null;default:false" json:"subscribe_all"`
	MaxRetries          int            `gorm:"column:max_retries;not null;default:3" json:"max_retries"`
	TimeoutSeconds      int            `gorm:"column:timeout_seconds;not null;default:30" json:"timeout_seconds"`
	Description         string         `gorm:"column:description;type:text" json:"description"`
	Metadata            datatypes.JSON `gorm:"column:metadata;type:jsonb;not null;default:'{}'" json:"metadata"`
	Status              string         `gorm:"column:status;type:varchar(20);not null;default:'active'" json:"status"`
	ConsecutiveFailures int            `gorm:"column:consecutive_failures;not null;default:0" json:"consecutive_failures"`
	LastTriggeredAt     *time.Time     `gorm:"column:last_triggered_at" json:"last_triggered_at"`
	CreatedBy           *int64         `gorm:"column:created_by" json:"created_by,omitempty"`
	UpdatedBy           *int64         `gorm:"column:updated_by" json:"updated_by,omitempty"`
	CreatedAt           time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}

// TableName returns the database table name for WebhookEndpoint.
func (WebhookEndpoint) TableName() string {
	return "webhook_endpoints"
}

// BeforeCreate sets a new UUID on the WebhookEndpoint before it is inserted
// into the database if one has not already been assigned.
func (we *WebhookEndpoint) BeforeCreate(tx *gorm.DB) error {
	if we.WebhookEndpointUUID == uuid.Nil {
		we.WebhookEndpointUUID = uuid.New()
	}
	return nil
}
