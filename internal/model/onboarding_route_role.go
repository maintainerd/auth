package model

import (
	"time"
)

type OnboardingRouteRole struct {
	OnboardingRouteRoleID int64     `gorm:"column:onboarding_route_role_id;primaryKey"`
	OnboardingRouteID     int64     `gorm:"column:onboarding_route_id"`
	RoleID                int64     `gorm:"column:role_id"`
	CreatedAt             time.Time `gorm:"column:created_at;autoCreateTime"`

	// Relationships
	OnboardingRoute *OnboardingRoute `gorm:"foreignKey:OnboardingRouteID;references:OnboardingRouteID"`
	Role            *Role            `gorm:"foreignKey:RoleID;references:RoleID"`
}

func (OnboardingRouteRole) TableName() string {
	return "onboarding_route_roles"
}
