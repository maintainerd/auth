package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TenantSetting holds tenant-level operational configuration such as rate
// limits, audit settings, maintenance windows, and feature flags.
type TenantSetting struct {
	TenantSettingID   int64          `gorm:"column:tenant_setting_id;primaryKey;autoIncrement" json:"tenant_setting_id"`
	TenantSettingUUID uuid.UUID      `gorm:"column:tenant_setting_uuid;type:uuid;uniqueIndex;not null" json:"tenant_setting_uuid"`
	TenantID          int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	RateLimitConfig   datatypes.JSON `gorm:"column:rate_limit_config;type:jsonb;default:'{}'" json:"rate_limit_config"`
	AuditConfig       datatypes.JSON `gorm:"column:audit_config;type:jsonb;default:'{}'" json:"audit_config"`
	MaintenanceConfig datatypes.JSON `gorm:"column:maintenance_config;type:jsonb;default:'{}'" json:"maintenance_config"`
	FeatureFlags      datatypes.JSON `gorm:"column:feature_flags;type:jsonb;default:'{}'" json:"feature_flags"`
	CreatedAt         time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
}

// TableName returns the database table name for TenantSetting.
func (TenantSetting) TableName() string {
	return "tenant_settings"
}

// BeforeCreate sets a new UUID on the TenantSetting before it is inserted into
// the database if one has not already been assigned.
func (ts *TenantSetting) BeforeCreate(tx *gorm.DB) error {
	if ts.TenantSettingUUID == uuid.Nil {
		ts.TenantSettingUUID = uuid.New()
	}
	return nil
}
