package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
)

// Role output structure
type RoleResponseDto struct {
	RoleUUID    uuid.UUID                `json:"role_id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Permissions *[]PermissionResponseDto `json:"permissions,omitempty"`
	IsDefault   bool                     `json:"is_default"`
	IsSystem    bool                     `json:"is_system"`
	Status      string                   `json:"status"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
}

// Create or update role request dto
type RoleCreateOrUpdateRequestDto struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

func (r RoleCreateOrUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Role name is required"),
			validation.Length(3, 20).Error("Role name must be between 3 and 20 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 100).Error("Description must be between 8 and 100 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In("active", "inactive").Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Add permissions to role request dto
type RoleAddPermissionsRequestDto struct {
	Permissions []uuid.UUID `json:"permissions"`
}

func (r RoleAddPermissionsRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Permissions,
			validation.Required.Error("Permission UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

// Role listing
type RoleFilterDto struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsDefault   *bool   `json:"is_default"`
	IsSystem    *bool   `json:"is_system"`
	Status      *string `json:"status"`

	// Pagination and sorting
	PaginationRequestDto
}
