package event

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Outbox represents a durable integration event record in the transactional outbox.
type Outbox struct {
	OutboxID      int64          `gorm:"column:outbox_id;primaryKey;autoIncrement" json:"outbox_id"`
	OutboxUUID    uuid.UUID      `gorm:"column:outbox_uuid;type:uuid;uniqueIndex;not null" json:"outbox_uuid"`
	EventID       uuid.UUID      `gorm:"column:event_id;type:uuid;not null" json:"event_id"`
	EventType     string         `gorm:"column:event_type;type:varchar(100);not null" json:"event_type"`
	EventVersion  int            `gorm:"column:event_version;not null;default:1" json:"event_version"`
	TenantID      int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	ActorUserID   *int64         `gorm:"column:actor_user_id" json:"actor_user_id"`
	SubjectUUID   *uuid.UUID     `gorm:"column:subject_uuid;type:uuid" json:"subject_uuid"`
	SubjectType   string         `gorm:"column:subject_type;type:varchar(50)" json:"subject_type"`
	ChangedFields datatypes.JSON `gorm:"column:changed_fields;type:jsonb;default:'[]'" json:"changed_fields"`
	Payload       datatypes.JSON `gorm:"column:payload;type:jsonb;not null;default:'{}'" json:"payload"`
	OccurredAt    time.Time      `gorm:"column:occurred_at;not null;autoCreateTime" json:"occurred_at"`
	TraceID       string         `gorm:"column:trace_id;type:varchar(255)" json:"trace_id"`
	RequestID     string         `gorm:"column:request_id;type:varchar(255)" json:"request_id"`
	IsPublished   bool           `gorm:"column:is_published;not null;default:false" json:"is_published"`
	PublishedAt   *time.Time     `gorm:"column:published_at" json:"published_at"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Outbox) TableName() string { return "integration_event_outbox" }

func (o *Outbox) BeforeCreate(tx *gorm.DB) error {
	if o.OutboxUUID == uuid.Nil {
		o.OutboxUUID = uuid.New()
	}
	if o.EventID == uuid.Nil {
		o.EventID = uuid.New()
	}
	return nil
}
