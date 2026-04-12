package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SecuritySetting holds pool-level security configuration as a set of JSONB
// columns. Each user pool has exactly one SecuritySetting row.
type SecuritySetting struct {
	SecuritySettingID   int64          `gorm:"column:security_setting_id;primaryKey;autoIncrement" json:"security_setting_id"`
	SecuritySettingUUID uuid.UUID      `gorm:"column:security_setting_uuid;type:uuid;uniqueIndex;not null" json:"security_setting_uuid"`
	UserPoolID          int64          `gorm:"column:user_pool_id;not null" json:"user_pool_id"`
	MFAConfig           datatypes.JSON `gorm:"column:mfa_config;type:jsonb;default:'{}'" json:"mfa_config"`
	PasswordConfig      datatypes.JSON `gorm:"column:password_config;type:jsonb;default:'{}'" json:"password_config"`
	SessionConfig       datatypes.JSON `gorm:"column:session_config;type:jsonb;default:'{}'" json:"session_config"`
	ThreatConfig        datatypes.JSON `gorm:"column:threat_config;type:jsonb;default:'{}'" json:"threat_config"`
	LockoutConfig       datatypes.JSON `gorm:"column:lockout_config;type:jsonb;default:'{}'" json:"lockout_config"`
	RegistrationConfig  datatypes.JSON `gorm:"column:registration_config;type:jsonb;default:'{}'" json:"registration_config"`
	TokenConfig         datatypes.JSON `gorm:"column:token_config;type:jsonb;default:'{}'" json:"token_config"`
	Version             int            `gorm:"column:version;default:1" json:"version"`
	CreatedBy           *int64         `gorm:"column:created_by" json:"created_by"`
	UpdatedBy           *int64         `gorm:"column:updated_by" json:"updated_by"`
	CreatedAt           time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	UserPool *UserPool `gorm:"foreignKey:UserPoolID;references:UserPoolID"`
	Creator  *User     `gorm:"foreignKey:CreatedBy;references:UserID"`
	Updater  *User     `gorm:"foreignKey:UpdatedBy;references:UserID"`
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
