package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DataErasureRequest tracks the lifecycle of a GDPR Article 17 (right to
// erasure) request: received → in_progress → completed (or rejected / on_hold).
// It is the authoritative work list for the background erasure worker.
//
// Backed by migration 080_create_data_erasure_requests_table.go.
type DataErasureRequest struct {
	DataErasureRequestID   int64      `gorm:"column:data_erasure_request_id;primaryKey;autoIncrement"`
	DataErasureRequestUUID uuid.UUID  `gorm:"column:data_erasure_request_uuid;type:uuid;uniqueIndex;not null"`
	TenantID               int64      `gorm:"column:tenant_id;not null"`
	UserID                 int64      `gorm:"column:user_id;not null"`
	RequestedByUserID      *int64     `gorm:"column:requested_by_user_id"`
	RequestedByAdminID     *int64     `gorm:"column:requested_by_admin_id"`
	Status                 string     `gorm:"column:status;type:varchar(30);not null;default:'pending'"`
	Reason                 string     `gorm:"column:reason;type:text;not null;default:''"`
	RejectionReason        *string    `gorm:"column:rejection_reason;type:text"`
	LegalHold              bool       `gorm:"column:legal_hold;not null;default:false"`
	LegalHoldReason        *string    `gorm:"column:legal_hold_reason;type:text"`
	ScheduledAt            time.Time  `gorm:"column:scheduled_at;not null"`
	StartedAt              *time.Time `gorm:"column:started_at"`
	CompletedAt            *time.Time `gorm:"column:completed_at"`
	CreatedAt              time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt              time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName returns the database table name for DataErasureRequest.
func (DataErasureRequest) TableName() string {
	return "data_erasure_requests"
}

// BeforeCreate assigns a UUID before insert when one has not been set.
func (d *DataErasureRequest) BeforeCreate(tx *gorm.DB) error {
	if d.DataErasureRequestUUID == uuid.Nil {
		d.DataErasureRequestUUID = uuid.New()
	}
	return nil
}
