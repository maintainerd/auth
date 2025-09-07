package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// Service output structure
type ServiceResponseDto struct {
	ServiceUUID uuid.UUID `json:"service_uuid"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	IsActive    bool      `json:"is_active"`
	IsPublic    bool      `json:"is_public"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Create or update service request dto
type ServiceCreateOrUpdateRequestDto struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	IsActive    bool   `json:"is_active"`
	IsPublic    bool   `json:"is_public"`
}

func (r ServiceCreateOrUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Service name is required"),
			validation.Length(3, 50).Error("Service name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(3, 100).Error("Display name must be between 3 and 100 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 255).Error("Description must be between 8 and 255 characters"),
		),
		validation.Field(&r.Version,
			validation.Required.Error("Version is required"),
		),
		validation.Field(&r.IsActive,
			validation.In(true, false).Error("Is active is required"),
		),
		validation.Field(&r.IsPublic,
			validation.In(true, false).Error("Is public is required"),
		),
	)
}

// Service listing filters
type ServiceFilterDto struct {
	Name        *string `json:"name"`
	DisplayName *string `json:"display_name"`
	Description *string `json:"description"`
	Version     *string `json:"version"`
	IsActive    *bool   `json:"is_active"`
	IsPublic    *bool   `json:"is_public"`
	IsDefault   *bool   `json:"is_default"`

	// Pagination and sorting
	PaginationRequestDto
}
