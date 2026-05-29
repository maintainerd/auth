package tenant

import (
	"time"
)

type TenantServiceLink struct {
	TenantServiceID int64     `gorm:"column:tenant_service_id;primaryKey"`
	TenantID        int64     `gorm:"column:tenant_id"`
	ServiceID       int64     `gorm:"column:service_id"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID;constraint:OnDelete:CASCADE"`
}

func (TenantServiceLink) TableName() string {
	return "tenant_services"
}
