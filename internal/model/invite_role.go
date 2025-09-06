package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InviteRole struct {
	InviteRoleID   int64     `gorm:"column:invite_role_id;primaryKey"`
	InviteRoleUUID uuid.UUID `gorm:"column:invite_role_uuid;unique"`
	InviteID       int64     `gorm:"column:invite_id"`
	RoleID         int64     `gorm:"column:role_id"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`

	// Relationships
	Invite Invite `gorm:"foreignKey:InviteID"`
	Role   Role   `gorm:"foreignKey:RoleID"`
}

func (InviteRole) TableName() string {
	return "invite_roles"
}

func (ir *InviteRole) BeforeCreate(tx *gorm.DB) (err error) {
	if ir.InviteRoleUUID == uuid.Nil {
		ir.InviteRoleUUID = uuid.New()
	}
	return
}
