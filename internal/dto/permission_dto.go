package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// Permission output structure
type PermissionResponseDto struct {
	PermissionUUID uuid.UUID       `json:"permission_id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	API            *APIResponseDto `json:"api,omitempty"`
	Status         string          `json:"status"`
	IsDefault      bool            `json:"is_default"`
	IsSystem       bool            `json:"is_system"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// Create Permission request DTO
type PermissionCreateRequestDto struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	APIUUID     string `json:"api_id"`
}

// Validation
func (r PermissionCreateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In("active", "inactive").Error("Status must be one of: active, inactive"),
		),
		validation.Field(&r.APIUUID,
			validation.Required.Error("API UUID is required"),
		),
	)
}

// Update Permission request DTO
type PermissionUpdateRequestDto struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// Validation
func (r PermissionUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In("active", "inactive").Error("Status must be one of: active, inactive"),
		),
	)
}

// API listing / filter DTO
type PermissionFilterDto struct {
	Name           *string `json:"name"`
	Description    *string `json:"description"`
	APIUUID        *string `json:"api_id"`
	RoleUUID       *string `json:"role_id"`
	AuthClientUUID *string `json:"client_id"`
	Status         *string `json:"status"`
	IsDefault      *bool   `json:"is_default"`
	IsSystem       *bool   `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDto
}

// Permission status update DTO
type PermissionStatusUpdateDto struct {
	Status string `json:"status"`
}

func (r PermissionStatusUpdateDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In("active", "inactive").Error("Status must be one of: active, inactive"),
		),
	)
}
