package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"

	"github.com/maintainerd/auth/internal/model"
)

// Service output structure
type ServiceResponseDTO struct {
	ServiceUUID uuid.UUID `json:"service_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Status      string    `json:"status"`
	IsSystem    bool      `json:"is_system"`
	APICount    int64     `json:"api_count"`
	PolicyCount int64     `json:"policy_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Create or update service request dto
type ServiceCreateOrUpdateRequestDTO struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Status      string `json:"status"`
}

func (r ServiceCreateOrUpdateRequestDTO) Validate() error {
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
			validation.In(model.StatusActive, model.StatusMaintenance, model.StatusDeprecated, model.StatusInactive).Error("Status must be one of: active, maintenance, deprecated, inactive"),
		),
	)
}

// Service listing filters
type ServiceFilterDTO struct {
	Name        *string  `json:"name"`
	DisplayName *string  `json:"display_name"`
	Description *string  `json:"description"`
	Version     *string  `json:"version"`
	Status      []string `json:"status"`
	IsSystem    *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the service filter DTO.
func (f ServiceFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(model.StatusActive, model.StatusMaintenance, model.StatusDeprecated, model.StatusInactive).Error("Status must be one of: active, maintenance, deprecated, inactive")),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}

// Service status update request dto
type ServiceStatusUpdateRequestDTO struct {
	Status string `json:"status"`
}

func (r ServiceStatusUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusMaintenance, model.StatusDeprecated, model.StatusInactive).Error("Status must be one of: active, maintenance, deprecated, inactive"),
		),
	)
}
