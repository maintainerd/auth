package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/maintainerd/auth/internal/model"
)

// Response DTOs
type APIKeyResponseDTO struct {
	APIKeyID    uuid.UUID  `json:"api_key_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	KeyPrefix   string     `json:"key_prefix"`
	ExpiresAt   *time.Time `json:"expires_at"`

	RateLimit *int      `json:"rate_limit"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// API Key API DTOs
type APIKeyAPIResponseDTO struct {
	APIKeyAPIID uuid.UUID               `json:"api_key_api_id"`
	API         APIResponseDTO          `json:"api"`
	Permissions []PermissionResponseDTO `json:"permissions,omitempty"`
	CreatedAt   time.Time               `json:"created_at"`
}

type APIKeyAPIsResponseDTO struct {
	APIs []APIKeyAPIResponseDTO `json:"apis"`
}

// API Key APIs pagination request DTO
type APIKeyAPIsGetRequestDTO struct {
	PaginationRequestDTO
}

// Add APIs to API key request dto
type AddAPIKeyAPIsRequestDTO struct {
	APIUUIDs []uuid.UUID `json:"api_uuids"`
}

func (r AddAPIKeyAPIsRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.APIUUIDs,
			validation.Required.Error("API UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

// Add permissions to API key API request dto
type AddAPIKeyPermissionsRequestDTO struct {
	PermissionUUIDs []uuid.UUID `json:"permission_uuids"`
}

func (r AddAPIKeyPermissionsRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PermissionUUIDs,
			validation.Required.Error("Permission UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

// API key status update DTO
type APIKeyStatusUpdateDTO struct {
	Status string `json:"status"`
}

func (r APIKeyStatusUpdateDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
		),
	)
}

type APIKeyPermissionResponseDTO struct {
	APIKeyPermissionID uuid.UUID              `json:"api_key_permission_id"`
	APIKey             *APIKeyResponseDTO     `json:"api_key,omitempty"`
	Permission         *PermissionResponseDTO `json:"permission,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
}

// API Key creation response DTO (includes the plain key)
type APIKeyCreateResponseDTO struct {
	APIKeyID    uuid.UUID  `json:"api_key_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	KeyPrefix   string     `json:"key_prefix"`
	Key         string     `json:"key"` // The actual API key that should be stored securely
	ExpiresAt   *time.Time `json:"expires_at"`

	RateLimit *int      `json:"rate_limit"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// API Key API permissions response DTO
type APIKeyAPIPermissionsResponseDTO struct {
	Permissions []PermissionResponseDTO `json:"permissions"`
}

// Request DTOs
type APIKeyCreateRequestDTO struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Config      datatypes.JSON `json:"config,omitempty"`
	ExpiresAt   *time.Time     `json:"expires_at,omitempty"`
	RateLimit   *int           `json:"rate_limit,omitempty"`
	Status      string         `json:"status,omitempty"`
}

func (dto APIKeyCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Name, validation.Required, validation.Length(1, 100)),
		validation.Field(&dto.Description, validation.Length(0, 500)),
		validation.Field(&dto.Status, validation.In(model.StatusActive, model.StatusInactive)),
		validation.Field(&dto.RateLimit, validation.Min(1)),
	)
}

type APIKeyUpdateRequestDTO struct {
	Name        *string        `json:"name,omitempty"`
	Description *string        `json:"description,omitempty"`
	Config      datatypes.JSON `json:"config,omitempty"`
	ExpiresAt   *time.Time     `json:"expires_at,omitempty"`
	RateLimit   *int           `json:"rate_limit,omitempty"`
	Status      *string        `json:"status,omitempty"`
}

func (dto APIKeyUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Name, validation.Length(1, 100)),
		validation.Field(&dto.Description, validation.Length(0, 500)),
		validation.Field(&dto.Status, validation.In(model.StatusActive, model.StatusInactive)),
		validation.Field(&dto.RateLimit, validation.Min(1)),
	)
}

// Query parameter DTOs
type APIKeyGetRequestDTO struct {
	PaginationRequestDTO
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

func (dto APIKeyGetRequestDTO) Validate() error {
	if err := dto.PaginationRequestDTO.Validate(); err != nil {
		return err
	}
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Status, validation.In(model.StatusActive, model.StatusInactive)),
	)
}
