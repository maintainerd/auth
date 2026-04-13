package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SMSConfig holds tenant-level SMS delivery configuration (Twilio, SNS,
// Vonage, MessageBird, etc.).
type SMSConfig struct {
	SMSConfigID        int64          `gorm:"column:sms_config_id;primaryKey;autoIncrement" json:"sms_config_id"`
	SMSConfigUUID      uuid.UUID      `gorm:"column:sms_config_uuid;type:uuid;uniqueIndex;not null" json:"sms_config_uuid"`
	TenantID           int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	Provider           string         `gorm:"column:provider;type:varchar(50);not null" json:"provider"`
	AccountSID         string         `gorm:"column:account_sid;type:varchar(255)" json:"account_sid"`
	AuthTokenEncrypted string         `gorm:"column:auth_token_encrypted;type:text" json:"-"`
	FromNumber         string         `gorm:"column:from_number;type:varchar(50)" json:"from_number"`
	SenderID           string         `gorm:"column:sender_id;type:varchar(50)" json:"sender_id"`
	TestMode           bool           `gorm:"column:test_mode;not null;default:false" json:"test_mode"`
	Status             string         `gorm:"column:status;type:varchar(20);not null;default:'active'" json:"status"`
	Metadata           datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
}

// TableName returns the database table name for SMSConfig.
func (SMSConfig) TableName() string {
	return "sms_config"
}

// BeforeCreate sets a new UUID on the SMSConfig before it is inserted into the
// database if one has not already been assigned.
func (sc *SMSConfig) BeforeCreate(tx *gorm.DB) error {
	if sc.SMSConfigUUID == uuid.Nil {
		sc.SMSConfigUUID = uuid.New()
	}
	return nil
}
