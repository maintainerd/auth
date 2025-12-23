package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SecuritySettingsAudit struct {
	SecuritySettingsAuditID   int64          `gorm:"column:security_settings_audit_id;primaryKey;autoIncrement" json:"security_settings_audit_id"`
	SecuritySettingsAuditUUID uuid.UUID      `gorm:"column:security_settings_audit_uuid;type:uuid;uniqueIndex;not null" json:"security_settings_audit_uuid"`
	TenantID                  int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	SecuritySettingID         int64          `gorm:"column:security_setting_id;not null" json:"security_setting_id"`
	ChangeType                string         `gorm:"column:change_type;type:varchar(50);not null" json:"change_type"`
	OldConfig                 datatypes.JSON `gorm:"column:old_config;type:jsonb" json:"old_config"`
	NewConfig                 datatypes.JSON `gorm:"column:new_config;type:jsonb" json:"new_config"`
	IpAddress                 string         `gorm:"column:ip_address;type:varchar(50)" json:"ip_address"`
	UserAgent                 string         `gorm:"column:user_agent;type:text" json:"user_agent"`
	CreatedBy                 *int64         `gorm:"column:created_by" json:"created_by"`
	CreatedAt                 time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt                 time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Tenant          *Tenant          `gorm:"foreignKey:TenantID;references:TenantID"`
	SecuritySetting *SecuritySetting `gorm:"foreignKey:SecuritySettingID;references:SecuritySettingID"`
	Creator         *User            `gorm:"foreignKey:CreatedBy;references:UserID"`
}

func (SecuritySettingsAudit) TableName() string {
	return "security_settings_audit"
}

func (ssa *SecuritySettingsAudit) BeforeCreate(tx *gorm.DB) error {
	if ssa.SecuritySettingsAuditUUID == uuid.Nil {
		ssa.SecuritySettingsAuditUUID = uuid.New()
	}
	return nil
}
