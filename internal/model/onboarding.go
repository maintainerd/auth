package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Onboarding struct {
	OnboardingID   int64     `gorm:"column:onboarding_id;primaryKey"`
	OnboardingUUID uuid.UUID `gorm:"column:onboarding_uuid;unique"`
	Name           string    `gorm:"column:name"`
	Identifier     string    `gorm:"column:identifier;unique"`
	Description    string    `gorm:"column:description"`
	AuthClientID   int64     `gorm:"column:auth_client_id"`
	IsActive       bool      `gorm:"column:is_active;default:true"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	AuthClient *AuthClient `gorm:"foreignKey:AuthClientID;references:AuthClientID"`
}

func (Onboarding) TableName() string {
	return "onboardings"
}

func (rr *Onboarding) BeforeCreate(tx *gorm.DB) (err error) {
	if rr.OnboardingUUID == uuid.Nil {
		rr.OnboardingUUID = uuid.New()
	}
	return
}
