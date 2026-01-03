package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantUser struct {
	TenantUserID   int64     `gorm:"column:tenant_user_id;primaryKey"`
	TenantUserUUID uuid.UUID `gorm:"column:tenant_user_uuid;unique"`
	TenantID       int64     `gorm:"column:tenant_id"`
	UserID         int64     `gorm:"column:user_id"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID;constraint:OnDelete:CASCADE"`
	User   *User   `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

func (TenantUser) TableName() string {
	return "tenant_users"
}

func (t *TenantUser) BeforeCreate(tx *gorm.DB) (err error) {
	if t.TenantUserUUID == uuid.Nil {
		t.TenantUserUUID = uuid.New()
	}
	return
}
