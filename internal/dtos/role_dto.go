package dtos

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/models"
)

type RoleDTO struct {
	RoleUUID    uuid.UUID  `json:"role_uuid"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	IsDefault   bool       `json:"is_default"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

func ToRoleDTO(role *models.Role) RoleDTO {
	return RoleDTO{
		RoleUUID:    role.RoleUUID,
		Name:        role.Name,
		Description: role.Description,
		IsDefault:   role.IsDefault,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}
