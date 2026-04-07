package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"

	"github.com/maintainerd/auth/internal/model"
)

// Role output structure
type RoleResponseDTO struct {
	RoleUUID    uuid.UUID                `json:"role_id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Permissions *[]PermissionResponseDTO `json:"permissions,omitempty"`
	IsDefault   bool                     `json:"is_default"`
	IsSystem    bool                     `json:"is_system"`
	Status      string                   `json:"status"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
}

// Create or update role request dto
type RoleCreateOrUpdateRequestDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

func (r RoleCreateOrUpdateRequestDTO) Validate() error {
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
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Add permissions to role request dto
type RoleAddPermissionsRequestDTO struct {
	Permissions []uuid.UUID `json:"permissions"`
}

func (r RoleAddPermissionsRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Permissions,
			validation.Required.Error("Permission UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

// Role listing
type RoleFilterDTO struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsDefault   *bool   `json:"is_default"`
	IsSystem    *bool   `json:"is_system"`
	Status      *string `json:"status"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the role filter DTO.
func (f RoleFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(f.Status != nil,
				validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}
