package idp

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RegistrationFlowRole struct {
	RegistrationFlowRoleID   int64             `gorm:"column:registration_flow_role_id;primaryKey;autoIncrement" json:"registration_flow_role_id"`
	RegistrationFlowRoleUUID uuid.UUID         `gorm:"column:registration_flow_role_uuid;type:uuid;uniqueIndex;not null" json:"registration_flow_role_uuid"`
	RegistrationFlowID       int64             `gorm:"column:registration_flow_id;not null" json:"registration_flow_id"`
	RoleID                   int64             `gorm:"column:role_id;not null" json:"role_id"`
	RegistrationFlow         *RegistrationFlow `gorm:"foreignKey:RegistrationFlowID;references:RegistrationFlowID" json:"registration_flow,omitempty"`
	Role                     *Role             `gorm:"foreignKey:RoleID;references:RoleID" json:"role,omitempty"`
	CreatedAt                time.Time         `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (RegistrationFlowRole) TableName() string {
	return "registration_flow_roles"
}

func (sfr *RegistrationFlowRole) BeforeCreate(tx *gorm.DB) error {
	if sfr.RegistrationFlowRoleUUID == uuid.Nil {
		sfr.RegistrationFlowRoleUUID = uuid.New()
	}
	return nil
}
