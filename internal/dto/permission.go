package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"

	"github.com/maintainerd/auth/internal/model"
)

// Permission output structure
type PermissionResponseDTO struct {
	PermissionUUID uuid.UUID       `json:"permission_id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	API            *APIResponseDTO `json:"api,omitempty"`
	Status         string          `json:"status"`
	IsDefault      bool            `json:"is_default"`
	IsSystem       bool            `json:"is_system"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// Create Permission request DTO
type PermissionCreateRequestDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	APIUUID     string `json:"api_id"`
}

// Validation
func (r PermissionCreateRequestDTO) Validate() error {
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
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be one of: active, inactive"),
		),
		validation.Field(&r.APIUUID,
			validation.Required.Error("API ID is required"),
			is.UUID.Error("API ID must be a valid UUID"),
		),
	)
}

// Update Permission request DTO
type PermissionUpdateRequestDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// Validation
func (r PermissionUpdateRequestDTO) Validate() error {
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
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be one of: active, inactive"),
		),
	)
}

// API listing / filter DTO
type PermissionFilterDTO struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	APIUUID     *string `json:"api_id"`
	RoleUUID    *string `json:"role_id"`
	ClientUUID  *string `json:"client_id"`
	Status      *string `json:"status"`
	IsDefault   *bool   `json:"is_default"`
	IsSystem    *bool   `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the permission filter DTO.
func (f PermissionFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(f.Status != nil,
				validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}

// Permission status update DTO
type PermissionStatusUpdateDTO struct {
	Status string `json:"status"`
}

func (r PermissionStatusUpdateDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be one of: active, inactive"),
		),
	)
}
