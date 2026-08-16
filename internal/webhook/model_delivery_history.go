package webhook

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeliveryHistory records the outcome of a webhook delivery attempt.
type DeliveryHistory struct {
	DeliveryHistoryID   int64      `gorm:"column:delivery_history_id;primaryKey;autoIncrement" json:"-"`
	DeliveryHistoryUUID uuid.UUID  `gorm:"column:delivery_history_uuid;type:uuid;uniqueIndex;not null" json:"delivery_history_id"`
	WebhookEndpointID   int64      `gorm:"column:webhook_endpoint_id;not null" json:"webhook_endpoint_id"`
	EventID             uuid.UUID  `gorm:"column:event_id;type:uuid;not null" json:"event_id"`
	EventType           string     `gorm:"column:event_type;type:varchar(100);not null" json:"event_type"`
	TenantID            int64      `gorm:"column:tenant_id;not null" json:"tenant_id"`
	AttemptCount        int        `gorm:"column:attempt_count;not null;default:1" json:"attempt_count"`
	ResponseStatus      *int       `gorm:"column:response_status" json:"response_status"`
	ResponseSummary     string     `gorm:"column:response_summary;type:text" json:"response_summary"`
	ErrorReason         string     `gorm:"column:error_reason;type:text" json:"error_reason"`
	NextRetryTime       *time.Time `gorm:"column:next_retry_time" json:"next_retry_time"`
	FinalStatus         string     `gorm:"column:final_status;type:varchar(20);not null;default:'pending'" json:"final_status"`
	IsReplay            bool       `gorm:"column:is_replay;not null;default:false" json:"is_replay"`
	CreatedAt           time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (DeliveryHistory) TableName() string { return "webhook_delivery_history" }

func (d *DeliveryHistory) BeforeCreate(tx *gorm.DB) error {
	if d.DeliveryHistoryUUID == uuid.Nil {
		d.DeliveryHistoryUUID = uuid.New()
	}
	return nil
}
