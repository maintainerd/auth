package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// WebhookEndpoint represents an outbound event notification subscription
// belonging to a tenant. Multiple endpoints may exist per tenant, each
// subscribing to a different set of events.
type WebhookEndpoint struct {
	WebhookEndpointID   int64          `gorm:"column:webhook_endpoint_id;primaryKey;autoIncrement" json:"webhook_endpoint_id"`
	WebhookEndpointUUID uuid.UUID      `gorm:"column:webhook_endpoint_uuid;type:uuid;uniqueIndex;not null" json:"webhook_endpoint_uuid"`
	TenantID            int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	URL                 string         `gorm:"column:url;type:text;not null" json:"url"`
	SecretEncrypted     string         `gorm:"column:secret_encrypted;type:text" json:"secret_encrypted"`
	Events              datatypes.JSON `gorm:"column:events;type:jsonb;default:'[]'" json:"events"`
	MaxRetries          int            `gorm:"column:max_retries;not null;default:3" json:"max_retries"`
	TimeoutSeconds      int            `gorm:"column:timeout_seconds;not null;default:30" json:"timeout_seconds"`
	Status              string         `gorm:"column:status;type:varchar(20);not null;default:'active'" json:"status"`
	Description         string         `gorm:"column:description;type:text" json:"description"`
	Metadata            datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'" json:"metadata"`
	LastTriggeredAt     *time.Time     `gorm:"column:last_triggered_at" json:"last_triggered_at"`
	CreatedAt           time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
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
