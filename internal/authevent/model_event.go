package authevent

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AuthEvent represents a security event stored in the auth_events table.
// Events are immutable (append-only) following OWASP tamper-protection guidance.
type AuthEvent struct {
	AuthEventID   int64          `gorm:"column:auth_event_id;primaryKey;autoIncrement"`
	AuthEventUUID uuid.UUID      `gorm:"column:auth_event_uuid;type:uuid;uniqueIndex;not null"`
	TenantID      int64          `gorm:"column:tenant_id;not null"`
	ActorUserID   *int64         `gorm:"column:actor_user_id"`
	TargetUserID  *int64         `gorm:"column:target_user_id"`
	IPAddress     string         `gorm:"column:ip_address;type:inet;not null"`
	UserAgent     *string        `gorm:"column:user_agent;type:text"`
	Category      string         `gorm:"column:category;type:varchar(20);not null"`
	EventType     string         `gorm:"column:event_type;type:varchar(60);not null"`
	Severity      string         `gorm:"column:severity;type:varchar(10);not null;default:INFO"`
	Result        string         `gorm:"column:result;type:varchar(10);not null"`
	Description   *string        `gorm:"column:description;type:text"`
	ErrorReason   *string        `gorm:"column:error_reason;type:varchar(255)"`
	TraceID       *string        `gorm:"column:trace_id;type:varchar(32)"`
	Metadata      datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime;not null"`

	// Relationships
}

// TableName returns the database table name for GORM.
func (AuthEvent) TableName() string {
	return "auth_events"
}

// BeforeCreate generates a UUID if one is not already set.
func (ae *AuthEvent) BeforeCreate(_ *gorm.DB) error {
	if ae.AuthEventUUID == uuid.Nil {
		ae.AuthEventUUID = uuid.New()
	}
	return nil
}
