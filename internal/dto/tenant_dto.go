package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// Tenant output structure
type TenantResponseDto struct {
	TenantUUID  uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Identifier  string    `json:"identifier"`
	IsActive    bool      `json:"is_active"`
	IsPublic    bool      `json:"is_public"`
	IsDefault   bool      `json:"is_default"`
	IsSystem    bool      `json:"is_system"`
	Metadata    any       `json:"metadata,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Create Tenant request DTO
type TenantCreateRequestDto struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
	IsPublic    bool   `json:"is_public"`
}

// Validation
func (r TenantCreateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.IsActive,
			validation.In(true, false).Error("Is active is required"),
		),
		validation.Field(&r.IsPublic,
			validation.In(true, false).Error("Is public is required"),
		),
	)
}

// Update Tenant request DTO
type TenantUpdateRequestDto struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
	IsPublic    bool   `json:"is_public"`
}

// Validation
func (r TenantUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.IsActive,
			validation.In(true, false).Error("Is active is required"),
		),
		validation.Field(&r.IsPublic,
			validation.In(true, false).Error("Is public is required"),
		),
	)
}

// API listing / filter DTO
type TenantFilterDto struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Identifier  *string `json:"identifier"`
	IsActive    *bool   `json:"is_active"`
	IsPublic    *bool   `json:"is_public"`
	IsDefault   *bool   `json:"is_default"`
	IsSystem    *bool   `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDto
}

// Validation
func (r TenantFilterDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PaginationRequestDto),
	)
}
