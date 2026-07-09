package auditlog

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ManagementAuditLog struct {
	ManagementAuditLogID   int64          `gorm:"column:management_audit_log_id;primaryKey;not null"`
	ManagementAuditLogUUID uuid.UUID      `gorm:"column:management_audit_log_uuid;type:uuid;not null"`
	TenantID               int64          `gorm:"column:tenant_id;not null"`
	ActorUserID            *int64         `gorm:"column:actor_user_id"`
	ActorClientID          *int64         `gorm:"column:actor_client_id"`
	Action                 string         `gorm:"column:action;type:varchar(100);not null"`
	ResourceType           string         `gorm:"column:resource_type;type:varchar(100);not null"`
	ResourceID             string         `gorm:"column:resource_id;type:varchar(255);not null"`
	ResourceUUID           *uuid.UUID     `gorm:"column:resource_uuid;type:uuid"`
	Changes                datatypes.JSON `gorm:"column:changes;type:jsonb;not null;default:'{}'"`
	IPAddress              *string        `gorm:"column:ip_address;type:inet"`
	UserAgent              *string        `gorm:"column:user_agent;type:text"`
	TraceID                *string        `gorm:"column:trace_id;type:varchar(64)"`
	RequestID              *string        `gorm:"column:request_id;type:varchar(255)"`
	Outcome                string         `gorm:"column:outcome;type:varchar(20);not null;default:'success'"`
	ErrorMessage           *string        `gorm:"column:error_message;type:text"`
	CreatedAt              time.Time      `gorm:"column:created_at;autoCreateTime;not null"`

	// Read-only presentation fields populated by audit-log read queries.
	ActorUserName   *string `gorm:"->;column:actor_user_name"`
	ActorClientName *string `gorm:"->;column:actor_client_name"`
}

func (ManagementAuditLog) TableName() string {
	return "management_audit_log"
}

func (m *ManagementAuditLog) BeforeCreate(_ *gorm.DB) error {
	if m.ManagementAuditLogUUID == uuid.Nil {
		m.ManagementAuditLogUUID = uuid.New()
	}
	if len(m.Changes) == 0 {
		m.Changes = datatypes.JSON("{}")
	}
	return nil
}
