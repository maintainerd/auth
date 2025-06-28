package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type AuthAuditLog struct {
	AuthAuditLogID   int64     `gorm:"primaryKey;column:auth_audit_log_id"`
	AuthAuditLogUUID uuid.UUID `gorm:"type:uuid;not null;unique;column:auth_audit_log_uuid"`

	UserID      *int64         `gorm:"index:idx_auth_audit_logs_user_id"`
	EventType   string         `gorm:"type:varchar(100);not null;index:idx_auth_audit_logs_event_type"`
	Description *string        `gorm:"type:text"`
	IPAddress   *string        `gorm:"type:varchar(100)"`
	UserAgent   *string        `gorm:"type:text"`
	Metadata    datatypes.JSON `gorm:"type:jsonb"`

	CreatedAt time.Time `gorm:"autoCreateTime"`

	// Relation
	User *User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:SET NULL"`
}

// TableName overrides the default table name
func (AuthAuditLog) TableName() string {
	return "auth_audit_logs"
}
