package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// EmailConfig holds tenant-level SMTP/SES/SendGrid delivery configuration.
type EmailConfig struct {
	EmailConfigID     int64          `gorm:"column:email_config_id;primaryKey;autoIncrement" json:"email_config_id"`
	EmailConfigUUID   uuid.UUID      `gorm:"column:email_config_uuid;type:uuid;uniqueIndex;not null" json:"email_config_uuid"`
	TenantID          int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	Provider          string         `gorm:"column:provider;type:varchar(50);not null" json:"provider"`
	Host              string         `gorm:"column:host;type:varchar(255)" json:"host"`
	Port              int            `gorm:"column:port" json:"port"`
	Username          string         `gorm:"column:username;type:varchar(255)" json:"username"`
	PasswordEncrypted string         `gorm:"column:password_encrypted;type:text" json:"-"`
	FromAddress       string         `gorm:"column:from_address;type:varchar(255);not null" json:"from_address"`
	FromName          string         `gorm:"column:from_name;type:varchar(255)" json:"from_name"`
	ReplyTo           string         `gorm:"column:reply_to;type:varchar(255)" json:"reply_to"`
	Encryption        string         `gorm:"column:encryption;type:varchar(20)" json:"encryption"`
	TestMode          bool           `gorm:"column:test_mode;not null;default:false" json:"test_mode"`
	Status            string         `gorm:"column:status;type:varchar(20);not null;default:'active'" json:"status"`
	Metadata          datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt         time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
}

// TableName returns the database table name for EmailConfig.
func (EmailConfig) TableName() string {
	return "email_config"
}

// BeforeCreate sets a new UUID on the EmailConfig before it is inserted into
// the database if one has not already been assigned.
func (ec *EmailConfig) BeforeCreate(tx *gorm.DB) error {
	if ec.EmailConfigUUID == uuid.Nil {
		ec.EmailConfigUUID = uuid.New()
	}
	return nil
}
