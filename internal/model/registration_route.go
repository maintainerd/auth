package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RegistrationRoute struct {
	RegistrationRouteID   int64     `gorm:"column:registration_route_id;primaryKey"`
	RegistrationRouteUUID uuid.UUID `gorm:"column:registration_route_uuid;unique"`
	Name                  string    `gorm:"column:name"`
	Identifier            string    `gorm:"column:identifier;unique"`
	Description           string    `gorm:"column:description"`
	AuthContainerID       int64     `gorm:"column:auth_container_id"`
	IsActive              bool      `gorm:"column:is_active;default:true"`
	CreatedAt             time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt             time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	AuthContainer *AuthContainer `gorm:"foreignKey:AuthContainerID;references:AuthContainerID"`
}

func (RegistrationRoute) TableName() string {
	return "registration_routes"
}

func (rr *RegistrationRoute) BeforeCreate(tx *gorm.DB) (err error) {
	if rr.RegistrationRouteUUID == uuid.Nil {
		rr.RegistrationRouteUUID = uuid.New()
	}
	return
}
