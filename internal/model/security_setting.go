package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SecuritySetting struct {
	SecuritySettingID   int64          `gorm:"column:security_setting_id;primaryKey;autoIncrement" json:"security_setting_id"`
	SecuritySettingUUID uuid.UUID      `gorm:"column:security_setting_uuid;type:uuid;uniqueIndex;not null" json:"security_setting_uuid"`
	TenantID            int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	GeneralConfig       datatypes.JSON `gorm:"column:general_config;type:jsonb;default:'{}'" json:"general_config"`
	PasswordConfig      datatypes.JSON `gorm:"column:password_config;type:jsonb;default:'{}'" json:"password_config"`
	SessionConfig       datatypes.JSON `gorm:"column:session_config;type:jsonb;default:'{}'" json:"session_config"`
	ThreatConfig        datatypes.JSON `gorm:"column:threat_config;type:jsonb;default:'{}'" json:"threat_config"`
	IpConfig            datatypes.JSON `gorm:"column:ip_config;type:jsonb;default:'{}'" json:"ip_config"`
	Version             int            `gorm:"column:version;default:1" json:"version"`
	CreatedBy           *int64         `gorm:"column:created_by" json:"created_by"`
	UpdatedBy           *int64         `gorm:"column:updated_by" json:"updated_by"`
	CreatedAt           time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Tenant    *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
	Creator   *User   `gorm:"foreignKey:CreatedBy;references:UserID"`
	Updater   *User   `gorm:"foreignKey:UpdatedBy;references:UserID"`
}

func (SecuritySetting) TableName() string {
	return "security_settings"
}

func (ss *SecuritySetting) BeforeCreate(tx *gorm.DB) error {
	if ss.SecuritySettingUUID == uuid.Nil {
		ss.SecuritySettingUUID = uuid.New()
	}
	return nil
}
