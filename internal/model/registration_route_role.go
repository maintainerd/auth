package model

import (
	"time"
)

type RegistrationRouteRole struct {
	RegistrationRouteRoleID int64     `gorm:"column:registration_route_role_id;primaryKey"`
	RegistrationRouteID     int64     `gorm:"column:registration_route_id"`
	RoleID                  int64     `gorm:"column:role_id"`
	CreatedAt               time.Time `gorm:"column:created_at;autoCreateTime"`

	// Relationships
	RegistrationRoute *RegistrationRoute `gorm:"foreignKey:RegistrationRouteID;references:RegistrationRouteID"`
	Role              *Role              `gorm:"foreignKey:RoleID;references:RoleID"`
}

func (RegistrationRouteRole) TableName() string {
	return "registration_route_roles"
}
