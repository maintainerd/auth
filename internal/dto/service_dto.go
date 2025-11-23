package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// Service output structure
type ServiceResponseDto struct {
	ServiceUUID uuid.UUID `json:"service_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Status      string    `json:"status"`
	IsPublic    bool      `json:"is_public"`
	IsDefault   bool      `json:"is_default"`
	IsSystem    bool      `json:"is_system"`
	APICount    int64     `json:"api_count"`
	PolicyCount int64     `json:"policy_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Create or update service request dto
type ServiceCreateOrUpdateRequestDto struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Status      string `json:"status"`
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
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In("active", "maintenance", "deprecated", "inactive").Error("Status must be one of: active, maintenance, deprecated, inactive"),
		),
		validation.Field(&r.IsPublic,
			validation.In(true, false).Error("Is public is required"),
		),
	)
}

// Service listing filters
type ServiceFilterDto struct {
	Name        *string  `json:"name"`
	DisplayName *string  `json:"display_name"`
	Description *string  `json:"description"`
	Version     *string  `json:"version"`
	Status      []string `json:"status"`
	IsPublic    *bool    `json:"is_public"`
	IsDefault   *bool    `json:"is_default"`
	IsSystem    *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDto
}

// Service status update request dto
type ServiceStatusUpdateRequestDto struct {
	Status string `json:"status"`
}

func (r ServiceStatusUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In("active", "maintenance", "deprecated", "inactive").Error("Status must be one of: active, maintenance, deprecated, inactive"),
		),
	)
}
