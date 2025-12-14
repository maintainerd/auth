package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Response DTOs
type APIKeyResponseDto struct {
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
type APIKeyApiResponseDto struct {
	APIKeyApiID uuid.UUID               `json:"api_key_api_id"`
	Api         APIResponseDto          `json:"api"`
	Permissions []PermissionResponseDto `json:"permissions,omitempty"`
	CreatedAt   time.Time               `json:"created_at"`
}

type APIKeyApisResponseDto struct {
	APIs []APIKeyApiResponseDto `json:"apis"`
}

// API Key APIs pagination request DTO
type APIKeyApisGetRequestDto struct {
	PaginationRequestDto
}

// Add APIs to API key request dto
type AddAPIKeyApisRequestDto struct {
	ApiUUIDs []uuid.UUID `json:"api_uuids"`
}

func (r AddAPIKeyApisRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.ApiUUIDs,
			validation.Required.Error("API UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

// Add permissions to API key API request dto
type AddAPIKeyPermissionsRequestDto struct {
	PermissionUUIDs []uuid.UUID `json:"permission_uuids"`
}

func (r AddAPIKeyPermissionsRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PermissionUUIDs,
			validation.Required.Error("Permission UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

// API key status update DTO
type APIKeyStatusUpdateDto struct {
	Status string `json:"status"`
}

func (r APIKeyStatusUpdateDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In("active", "inactive").Error("Status must be either 'active' or 'inactive'"),
		),
	)
}

type APIKeyPermissionResponseDto struct {
	APIKeyPermissionID uuid.UUID              `json:"api_key_permission_id"`
	APIKey             *APIKeyResponseDto     `json:"api_key,omitempty"`
	Permission         *PermissionResponseDto `json:"permission,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
}

// API Key creation response DTO (includes the plain key)
type APIKeyCreateResponseDto struct {
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
type APIKeyApiPermissionsResponseDto struct {
	Permissions []PermissionResponseDto `json:"permissions"`
}

// Request DTOs
type APIKeyCreateRequestDto struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Config      datatypes.JSON `json:"config,omitempty"`
	ExpiresAt   *time.Time     `json:"expires_at,omitempty"`
	RateLimit   *int           `json:"rate_limit,omitempty"`
	Status      string         `json:"status,omitempty"`
}

func (dto APIKeyCreateRequestDto) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Name, validation.Required, validation.Length(1, 100)),
		validation.Field(&dto.Description, validation.Length(0, 500)),
		validation.Field(&dto.Status, validation.In("active", "inactive")),
		validation.Field(&dto.RateLimit, validation.Min(1)),
	)
}

type APIKeyUpdateRequestDto struct {
	Name        *string        `json:"name,omitempty"`
	Description *string        `json:"description,omitempty"`
	Config      datatypes.JSON `json:"config,omitempty"`
	ExpiresAt   *time.Time     `json:"expires_at,omitempty"`
	RateLimit   *int           `json:"rate_limit,omitempty"`
	Status      *string        `json:"status,omitempty"`
}

func (dto APIKeyUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Name, validation.Length(1, 100)),
		validation.Field(&dto.Description, validation.Length(0, 500)),
		validation.Field(&dto.Status, validation.In("active", "inactive")),
		validation.Field(&dto.RateLimit, validation.Min(1)),
	)
}

// Query parameter DTOs
type APIKeyGetRequestDto struct {
	PaginationRequestDto
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

func (dto APIKeyGetRequestDto) Validate() error {
	if err := dto.PaginationRequestDto.Validate(); err != nil {
		return err
	}
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Status, validation.In("active", "inactive")),
	)
}
