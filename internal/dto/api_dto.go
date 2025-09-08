package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// API output structure
type APIResponseDto struct {
	APIUUID     uuid.UUID          `json:"api_uuid"`
	Name        string             `json:"name"`
	DisplayName string             `json:"display_name"`
	Description string             `json:"description"`
	APIType     string             `json:"api_type"`
	Identifier  string             `json:"identifier"`
	Service     ServiceResponseDto `json:"service"`
	IsActive    bool               `json:"is_active"`
	IsDefault   bool               `json:"is_default"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// Create API request DTO
type APICreateRequestDto struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	APIType     string `json:"api_type"`
	IsActive    bool   `json:"is_active"`
	IsDefault   bool   `json:"is_default"`
	ServiceUUID string `json:"service_uuid"`
}

// Validation
func (r APICreateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("API name is required"),
			validation.Length(3, 50).Error("API name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(3, 50).Error("Display name must be between 3 and 50 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.APIType,
			validation.Required.Error("API type is required"),
			validation.Length(3, 50).Error("API type must be between 3 and 50 characters"),
		),
		validation.Field(&r.IsActive,
			validation.In(true, false).Error("Is active is required"),
		),
		validation.Field(&r.IsDefault,
			validation.In(true, false).Error("Is default is required"),
		),
		validation.Field(&r.ServiceUUID,
			validation.Required.Error("Service UUID is required"),
		),
	)
}

// Update API request DTO
type APIUpdateRequestDto struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	APIType     string `json:"api_type"`
	IsActive    bool   `json:"is_active"`
	IsDefault   bool   `json:"is_default"`
}

// Validation
func (r APIUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("API name is required"),
			validation.Length(3, 50).Error("API name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(3, 50).Error("Display name must be between 3 and 50 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.APIType,
			validation.Required.Error("API type is required"),
			validation.Length(3, 50).Error("API type must be between 3 and 50 characters"),
		),
		validation.Field(&r.IsActive,
			validation.In(true, false).Error("Is active is required"),
		),
		validation.Field(&r.IsDefault,
			validation.In(true, false).Error("Is default is required"),
		),
	)
}

// API listing / filter DTO
type APIFilterDto struct {
	Name        *string `json:"name"`
	DisplayName *string `json:"display_name"`
	Description *string `json:"description"`
	APIType     *string `json:"api_type"`
	Identifier  *string `json:"identifier"`
	ServiceUUID *string `json:"service_uuid"`
	IsActive    *bool   `json:"is_active"`
	IsDefault   *bool   `json:"is_default"`

	// Pagination and sorting
	PaginationRequestDto
}
