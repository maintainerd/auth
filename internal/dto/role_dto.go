package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// Role output structure
type RoleResponseDto struct {
	RoleUUID    uuid.UUID `json:"role_uuid"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsDefault   bool      `json:"is_default"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

// Create or update role request dto
type RoleCreateOrUpdateRequestDto struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
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
		validation.Field(&r.IsActive,
			validation.Required.Error("Is active is required"),
		),
	)
}

// Role listing
type RoleFilterDto struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsDefault   *bool   `json:"is_default"`
	IsActive    *bool   `json:"is_active"`

	// Pagination and sorting
	PaginationRequestDto
}
