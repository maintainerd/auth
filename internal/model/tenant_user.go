package model

import (
	"time"

	"github.com/google/uuid"
)

type TenantUser struct {
	TenantUserID   int64     `gorm:"column:tenant_user_id;primaryKey"`
	TenantUserUUID uuid.UUID `gorm:"column:tenant_user_uuid;unique;not null"`
	TenantID       int64     `gorm:"column:tenant_id;not null"`
	UserID         int64     `gorm:"column:user_id;not null"`
	Role           string    `gorm:"column:role;not null;default:'member'"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (TenantUser) TableName() string {
	return "tenant_users"
}

func (t *TenantUser) BeforeCreate(tx any) (err error) {
	if t.TenantUserUUID == uuid.Nil {
		t.TenantUserUUID = uuid.New()
	}
	return
}
