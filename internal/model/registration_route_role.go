package model

import (
	"time"
)

type RegistrationRouteRole struct {
	RegistrationRouteRoleID int64     `gorm:"column:registration_route_role_id;primaryKey"`
	RegistrationRouteID     int64     `gorm:"column:registration_route_id;type:integer;not null;index:idx_registration_route_roles_route_id"`
	RoleID                  int64     `gorm:"column:role_id;type:integer;not null;index:idx_registration_route_roles_role_id"`
	CreatedAt               time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime"`

	// Relationships
	RegistrationRoute *RegistrationRoute `gorm:"foreignKey:RegistrationRouteID;references:RegistrationRouteID;constraint:OnDelete:CASCADE"`
	Role              *Role              `gorm:"foreignKey:RoleID;references:RoleID;constraint:OnDelete:CASCADE"`
}

func (RegistrationRouteRole) TableName() string {
	return "registration_route_roles"
}
