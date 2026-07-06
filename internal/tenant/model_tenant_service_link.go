package tenant

import "time"

// TenantServiceLink is DEPRECATED — the tenant_services table has been removed.
// services.tenant_id is the authoritative tenant-scope relationship.
type TenantServiceLink struct {
	TenantServiceID int64     `gorm:"column:tenant_service_id;primaryKey"`
	TenantID        int64     `gorm:"column:tenant_id"`
	ServiceID       int64     `gorm:"column:service_id"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (TenantServiceLink) TableName() string {
	return "tenant_services"
}
