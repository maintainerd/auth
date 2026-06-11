package idp

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthFlowRole struct {
	AuthFlowRoleID   int64       `gorm:"column:auth_flow_role_id;primaryKey;autoIncrement" json:"auth_flow_role_id"`
	AuthFlowRoleUUID uuid.UUID   `gorm:"column:auth_flow_role_uuid;type:uuid;uniqueIndex;not null" json:"auth_flow_role_uuid"`
	AuthFlowID       int64       `gorm:"column:auth_flow_id;not null" json:"auth_flow_id"`
	RoleID             int64       `gorm:"column:role_id;not null" json:"role_id"`
	AuthFlow         *AuthFlow `gorm:"foreignKey:AuthFlowID;references:AuthFlowID" json:"auth_flow,omitempty"`
	Role               *Role       `gorm:"foreignKey:RoleID;references:RoleID" json:"role,omitempty"`
	CreatedAt          time.Time   `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (AuthFlowRole) TableName() string {
	return "auth_flow_roles"
}

func (sfr *AuthFlowRole) BeforeCreate(tx *gorm.DB) error {
	if sfr.AuthFlowRoleUUID == uuid.Nil {
		sfr.AuthFlowRoleUUID = uuid.New()
	}
	return nil
}
