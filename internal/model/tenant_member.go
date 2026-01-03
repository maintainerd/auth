package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantMember struct {
	TenantMemberID   int64     `gorm:"column:tenant_member_id;primaryKey"`
	TenantMemberUUID uuid.UUID `gorm:"column:tenant_member_uuid;unique;not null"`
	TenantID         int64     `gorm:"column:tenant_id;not null"`
	UserID           int64     `gorm:"column:user_id;not null"`
	Role             string    `gorm:"column:role;not null;default:'member'"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (TenantMember) TableName() string {
	return "tenant_members"
}

func (t *TenantMember) BeforeCreate(tx *gorm.DB) (err error) {
	if t.TenantMemberUUID == uuid.Nil {
		t.TenantMemberUUID = uuid.New()
	}
	return
}
