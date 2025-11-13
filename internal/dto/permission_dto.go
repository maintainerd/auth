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
	IsActive       bool            `json:"is_active"`
	IsDefault      bool            `json:"is_default"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// Create Permission request DTO
type PermissionCreateRequestDto struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
	APIUUID     string `json:"api_uuid"`
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
		validation.Field(&r.IsActive,
			validation.In(true, false).Error("Is active is required"),
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
	IsActive    bool   `json:"is_active"`
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
		validation.Field(&r.IsActive,
			validation.In(true, false).Error("Is active is required"),
		),
	)
}

// API listing / filter DTO
type PermissionFilterDto struct {
	Name           *string `json:"name"`
	Description    *string `json:"description"`
	APIUUID        *string `json:"api_uuid"`
	RoleUUID       *string `json:"role_uuid"`
	AuthClientUUID *string `json:"auth_client_uuid"`
	IsActive       *bool   `json:"is_active"`
	IsDefault      *bool   `json:"is_default"`

	// Pagination and sorting
	PaginationRequestDto
}
