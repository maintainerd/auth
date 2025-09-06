package model

import (
	"time"
)

type OnboardingRouteRole struct {
	OnboardingRoleID int64     `gorm:"column:onboarding_role_id;primaryKey"`
	OnboardingID     int64     `gorm:"column:onboarding_id"`
	RoleID           int64     `gorm:"column:role_id"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`

	// Relationships
	Onboarding *Onboarding `gorm:"foreignKey:OnboardingID;references:OnboardingID"`
	Role       *Role       `gorm:"foreignKey:RoleID;references:RoleID"`
}

func (OnboardingRouteRole) TableName() string {
	return "onboarding_roles"
}
