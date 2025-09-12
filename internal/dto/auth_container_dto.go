package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// AuthContainer output structure
type AuthContainerResponseDto struct {
	AuthContainerUUID uuid.UUID                `json:"auth_container_uuid"`
	Name              string                   `json:"name"`
	Description       string                   `json:"description"`
	Identifier        string                   `json:"identifier"`
	Organization      *OrganizationResponseDto `json:"organization,omitempty"`
	IsActive          bool                     `json:"is_active"`
	IsPublic          bool                     `json:"is_public"`
	IsDefault         bool                     `json:"is_default"`
	CreatedAt         time.Time                `json:"created_at"`
	UpdatedAt         time.Time                `json:"updated_at"`
}

// Create Auth Container request DTO
type AuthContainerCreateRequestDto struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	IsActive         bool   `json:"is_active"`
	IsPublic         bool   `json:"is_public"`
	OrganizationUUID string `json:"organization_uuid"`
}

// Validation
func (r AuthContainerCreateRequestDto) Validate() error {
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
		validation.Field(&r.OrganizationUUID,
			validation.Required.Error("Organization UUID is required"),
		),
	)
}

// Update Auth Container request DTO
type AuthContainerUpdateRequestDto struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
	IsPublic    bool   `json:"is_public"`
}

// Validation
func (r AuthContainerUpdateRequestDto) Validate() error {
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
type AuthContainerFilterDto struct {
	Name             *string `json:"name"`
	Description      *string `json:"description"`
	Identifier       *string `json:"identifier"`
	OrganizationUUID *string `json:"organization_uuid"`
	IsActive         *bool   `json:"is_active"`
	IsPublic         *bool   `json:"is_public"`
	IsDefault        *bool   `json:"is_default"`

	// Pagination and sorting
	PaginationRequestDto
}
