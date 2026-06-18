package tenant

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TenantSetting holds tenant-level operational configuration such as rate
// limits, audit settings, and maintenance windows.
type TenantSetting struct {
	TenantSettingID   int64          `gorm:"column:tenant_setting_id;primaryKey;autoIncrement" json:"tenant_setting_id"`
	TenantSettingUUID uuid.UUID      `gorm:"column:tenant_setting_uuid;type:uuid;uniqueIndex;not null" json:"tenant_setting_uuid"`
	TenantID          int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	RateLimitConfig   datatypes.JSON `gorm:"column:rate_limit_config;type:jsonb;default:'{}'" json:"rate_limit_config"`
	AuditConfig       datatypes.JSON `gorm:"column:audit_config;type:jsonb;default:'{}'" json:"audit_config"`
	MaintenanceConfig datatypes.JSON `gorm:"column:maintenance_config;type:jsonb;default:'{}'" json:"maintenance_config"`
	CreatedAt         time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (TenantSetting) TableName() string {
	return "tenant_settings"
}

func (ts *TenantSetting) BeforeCreate(_ *gorm.DB) error {
	if ts.TenantSettingUUID == uuid.Nil {
		ts.TenantSettingUUID = uuid.New()
	}
	return nil
}
