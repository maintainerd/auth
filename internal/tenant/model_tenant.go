package tenant

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Tenant struct {
	TenantID    int64          `gorm:"column:tenant_id;primaryKey"`
	TenantUUID  uuid.UUID      `gorm:"column:tenant_uuid"`
	Name        string         `gorm:"column:name;not null"`
	DisplayName string         `gorm:"column:display_name"`
	Description string         `gorm:"column:description"`
	Status      string         `gorm:"column:status;not null;default:'active'"`
	IsSystem    bool           `gorm:"column:is_system;not null;default:false"`
	Metadata    datatypes.JSON `gorm:"column:metadata;type:jsonb;not null;default:'{}'"`
	CreatedBy   *int64         `gorm:"column:created_by"`
	UpdatedBy   *int64         `gorm:"column:updated_by"`
	CreatedAt   time.Time      `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (Tenant) TableName() string {
	return "tenants"
}

func (t *Tenant) BeforeCreate(_ *gorm.DB) error {
	if t.TenantUUID == uuid.Nil {
		t.TenantUUID = uuid.New()
	}
	return nil
}
