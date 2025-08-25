package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OnboardingRoute struct {
	OnboardingRouteID   int64     `gorm:"column:onboarding_route_id;primaryKey"`
	OnboardingRouteUUID uuid.UUID `gorm:"column:onboarding_route_uuid;unique"`
	Name                string    `gorm:"column:name"`
	Identifier          string    `gorm:"column:identifier;unique"`
	Description         string    `gorm:"column:description"`
	AuthClientID        int64     `gorm:"column:auth_client_id"`
	IsActive            bool      `gorm:"column:is_active;default:true"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	AuthClient *AuthClient `gorm:"foreignKey:AuthClientID;references:AuthClientID"`
}

func (OnboardingRoute) TableName() string {
	return "onboarding_routes"
}

func (rr *OnboardingRoute) BeforeCreate(tx *gorm.DB) (err error) {
	if rr.OnboardingRouteUUID == uuid.Nil {
		rr.OnboardingRouteUUID = uuid.New()
	}
	return
}
