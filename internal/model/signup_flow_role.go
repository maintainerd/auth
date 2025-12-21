package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SignupFlowRole struct {
	SignupFlowRoleID   int64       `gorm:"column:signup_flow_role_id;primaryKey;autoIncrement" json:"signup_flow_role_id"`
	SignupFlowRoleUUID uuid.UUID   `gorm:"column:signup_flow_role_uuid;type:uuid;uniqueIndex;not null" json:"signup_flow_role_uuid"`
	SignupFlowID       int64       `gorm:"column:signup_flow_id;not null" json:"signup_flow_id"`
	RoleID             int64       `gorm:"column:role_id;not null" json:"role_id"`
	SignupFlow         *SignupFlow `gorm:"foreignKey:SignupFlowID;references:SignupFlowID" json:"signup_flow,omitempty"`
	Role               *Role       `gorm:"foreignKey:RoleID;references:RoleID" json:"role,omitempty"`
	CreatedAt          time.Time   `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (SignupFlowRole) TableName() string {
	return "signup_flow_roles"
}

func (sfr *SignupFlowRole) BeforeCreate(tx *gorm.DB) error {
	if sfr.SignupFlowRoleUUID == uuid.Nil {
		sfr.SignupFlowRoleUUID = uuid.New()
	}
	return nil
}
