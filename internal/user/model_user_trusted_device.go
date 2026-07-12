package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserTrustedDevice struct {
	UserTrustedDeviceID   int64          `gorm:"column:user_trusted_device_id;primaryKey"`
	UserTrustedDeviceUUID uuid.UUID      `gorm:"column:user_trusted_device_uuid;unique;not null"`
	UserID                int64          `gorm:"column:user_id;not null"`
	TenantID              int64          `gorm:"column:tenant_id;not null"`
	DeviceFingerprint     string         `gorm:"column:device_fingerprint;not null"`
	DeviceTokenHash       string         `gorm:"column:device_token_hash;not null"`
	DeviceName            *string        `gorm:"column:device_name"`
	Location              *string        `gorm:"column:location"`
	IPAddress             *string        `gorm:"column:ip_address"`
	UserAgent             *string        `gorm:"column:user_agent"`
	TrustedUntil          time.Time      `gorm:"column:trusted_until;not null"`
	LastSeenAt            *time.Time     `gorm:"column:last_seen_at"`
	CreatedAt             time.Time      `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt             time.Time      `gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt             gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (UserTrustedDevice) TableName() string {
	return "user_trusted_devices"
}

func (d *UserTrustedDevice) BeforeCreate(tx *gorm.DB) (err error) {
	if d.UserTrustedDeviceUUID == uuid.Nil {
		d.UserTrustedDeviceUUID = uuid.New()
	}
	return
}
