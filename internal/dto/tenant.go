package dto

import (
	"regexp"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"

	"github.com/maintainerd/auth/internal/model"
)

var tenantNamePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// Tenant output structure
type TenantResponseDTO struct {
	TenantUUID  uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Identifier  string    `json:"identifier"`
	Status      string    `json:"status"`
	IsPublic    bool      `json:"is_public"`
	IsSystem    bool      `json:"is_system"`
	Metadata    any       `json:"metadata,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Create Tenant request DTO
type TenantCreateRequestDTO struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IsPublic    bool   `json:"is_public"`
}

// Validation
func (r TenantCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
			validation.Match(tenantNamePattern).Error("Name must contain only lowercase letters, numbers, and hyphens"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive, model.StatusPending, model.StatusSuspended).Error("Status must be active, inactive, pending, or suspended"),
		),
		validation.Field(&r.IsPublic,
			validation.In(true, false).Error("Is public is required"),
		),
	)
}

// Update Tenant request DTO
type TenantUpdateRequestDTO struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IsPublic    bool   `json:"is_public"`
}

// Validation
func (r TenantUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
			validation.Match(tenantNamePattern).Error("Name must contain only lowercase letters, numbers, and hyphens"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive, model.StatusPending, model.StatusSuspended).Error("Status must be active, inactive, pending, or suspended"),
		),
		validation.Field(&r.IsPublic,
			validation.In(true, false).Error("Is public is required"),
		),
	)
}

// API listing / filter DTO
type TenantFilterDTO struct {
	Name        *string  `json:"name"`
	DisplayName *string  `json:"display_name"`
	Description *string  `json:"description"`
	Identifier  *string  `json:"identifier"`
	Status      []string `json:"status"`
	IsPublic    *bool    `json:"is_public"`
	IsSystem    *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validation
func (r TenantFilterDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PaginationRequestDTO),
	)
}
