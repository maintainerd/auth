package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RegistrationRoute struct {
	RegistrationRouteID   int64     `gorm:"column:registration_route_id;primaryKey"`
	RegistrationRouteUUID uuid.UUID `gorm:"column:registration_route_uuid;type:uuid;not null;unique"`
	Name                  string    `gorm:"column:name;type:varchar(100);not null"`
	Identifier            string    `gorm:"column:identifier;type:varchar(255);not null;unique;index:idx_registration_routes_identifier"`
	Description           string    `gorm:"column:description;type:text;not null"`
	AuthContainerID       int64     `gorm:"column:auth_container_id;type:integer;not null;index:idx_registration_routes_auth_container_id"`
	IsActive              bool      `gorm:"column:is_active;type:boolean;default:true;index:idx_registration_routes_is_active"`
	CreatedAt             time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt             time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`

	// Relationships
	AuthContainer *AuthContainer `gorm:"foreignKey:AuthContainerID;references:AuthContainerID;constraint:OnDelete:CASCADE"`
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
