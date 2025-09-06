package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ServicePolicy struct {
	ServicePolicyID   int64     `gorm:"column:service_policy_id;primaryKey"`
	ServicePolicyUUID uuid.UUID `gorm:"column:service_policy_uuid;unique"`
	ServiceID         int64     `gorm:"column:service_id"`
	PolicyID          int64     `gorm:"column:policy_id"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime"`

	// Relationships
	Service *Service `gorm:"foreignKey:ServiceID;references:ServiceID"`
	Policy  *Policy  `gorm:"foreignKey:PolicyID;references:PolicyID"`
}

func (ServicePolicy) TableName() string {
	return "service_policies"
}

func (sp *ServicePolicy) BeforeCreate(tx *gorm.DB) (err error) {
	if sp.ServicePolicyUUID == uuid.Nil {
		sp.ServicePolicyUUID = uuid.New()
	}
	return
}
