package dto

import (
	"time"

	"github.com/google/uuid"
	model "github.com/maintainerd/auth/internal/model"
)

type RoleDTO struct {
	RoleUUID        uuid.UUID `json:"role_uuid"`
	Name            string    `json:"name"`
	Description     string    `json:"description"` // changed from *string to string
	IsDefault       bool      `json:"is_default"`
	AuthContainerID int64     `json:"auth_container_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

func ToRoleDTO(role *model.Role) RoleDTO {
	return RoleDTO{
		RoleUUID:        role.RoleUUID,
		Name:            role.Name,
		Description:     role.Description,
		IsDefault:       role.IsDefault,
		AuthContainerID: role.AuthContainerID,
		CreatedAt:       role.CreatedAt,
		UpdatedAt:       role.UpdatedAt,
	}
}
