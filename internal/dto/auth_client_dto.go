package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type AuthClientSecretResponseDto struct {
	ClientID     string  `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
}

type AuthClientURIResponseDto struct {
	AuthClientURIUUID uuid.UUID `json:"uri_id"`
	URI               string    `json:"uri"`
	Type              string    `json:"type"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type AuthClientURIsResponseDto struct {
	URIs []AuthClientURIResponseDto `json:"uris"`
}

type AuthClientApisResponseDto struct {
	APIs []AuthClientApiResponseDto `json:"apis"`
}

type AuthClientApiPermissionsResponseDto struct {
	Permissions []PermissionResponseDto `json:"permissions"`
}

type SuccessResponseDto struct {
	Message string `json:"message"`
}

// Auth client output structure
type AuthClientResponseDto struct {
	AuthClientUUID   uuid.UUID                    `json:"client_id"`
	Name             string                       `json:"name"`
	DisplayName      string                       `json:"display_name"`
	ClientType       string                       `json:"client_type"`
	Domain           *string                      `json:"domain,omitempty"`
	URIs             []AuthClientURIResponseDto   `json:"uris,omitempty"`
	IdentityProvider *IdentityProviderResponseDto `json:"identity_provider,omitempty"`
	Permissions      *[]PermissionResponseDto     `json:"permissions,omitempty"`
	Status           string                       `json:"status"`
	IsDefault        bool                         `json:"is_default"`
	IsSystem         bool                         `json:"is_system"`
	CreatedAt        time.Time                    `json:"created_at"`
	UpdatedAt        time.Time                    `json:"updated_at"`
}

// Create auth client request DTO
type AuthClientCreateRequestDto struct {
	Name                 string         `json:"name"`
	DisplayName          string         `json:"display_name"`
	ClientType           string         `json:"client_type"`
	Domain               string         `json:"domain"`
	Config               datatypes.JSON `json:"config"`
	Status               string         `json:"status"`
	IdentityProviderUUID string         `json:"identity_provider_id"`
}

// Validation
func (r AuthClientCreateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display Name is required"),
			validation.Length(8, 200).Error("Display Name must be between 8 and 200 characters"),
		),
		validation.Field(&r.ClientType,
			validation.In("traditional", "spa", "mobile", "m2m").Error("Invalid client Type"),
		),
		validation.Field(&r.Domain,
			validation.Required.Error("Domain is required"),
			validation.Length(3, 100).Error("Domain must be between 3 and 100 characters"),
		),
		validation.Field(&r.Config,
			validation.Required.Error("Config is required"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In("active", "inactive").Error("Status must be one of: active, inactive"),
		),
		validation.Field(&r.IdentityProviderUUID,
			validation.Required.Error("Identity Provider UUID is required"),
		),
	)
}

// Update auth client request DTO
type AuthClientUpdateRequestDto struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name"`
	ClientType  string         `json:"client_type"`
	Domain      string         `json:"domain"`
	Config      datatypes.JSON `json:"config"`
	Status      string         `json:"status"`
}

// Validation
func (r AuthClientUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display Name is required"),
			validation.Length(8, 200).Error("Display Name must be between 8 and 200 characters"),
		),
		validation.Field(&r.ClientType,
			validation.In("traditional", "spa", "mobile", "m2m").Error("Client Type is required"),
		),
		validation.Field(&r.Domain,
			validation.Required.Error("Domain is required"),
			validation.Length(3, 100).Error("Domain must be between 3 and 100 characters"),
		),
		validation.Field(&r.Config,
			validation.Required.Error("Config is required"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In("active", "inactive").Error("Status must be one of: active, inactive"),
		),
	)
}

// Create or update auth client URI request DTO
type AuthClientURICreateOrUpdateRequestDto struct {
	URI  string `json:"uri"`
	Type string `json:"type"`
}

// Validation
func (r AuthClientURICreateOrUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.URI,
			validation.Required.Error("URI is required"),
			validation.Length(5, 200).Error("URI must be between 5 and 200 characters"),
		),
		validation.Field(&r.Type,
			validation.Required.Error("Type is required"),
			validation.In("redirect-uri", "origin-uri", "logout-uri", "login-uri", "cors-origin-uri").Error("Type must be one of: redirect-uri, origin-uri, logout-uri, login-uri, cors-origin-uri"),
		),
	)
}

// Auth client listing / filter DTO
type AuthClientFilterDto struct {
	Name                 *string  `json:"name"`
	DisplayName          *string  `json:"display_name"`
	ClientType           []string `json:"client_type"`
	IdentityProviderUUID *string  `json:"identity_provider_id"`
	Status               []string `json:"status"`
	IsDefault            *bool    `json:"is_default"`
	IsSystem             *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDto
}

// Add permissions to auth client request dto
type AuthClientAddPermissionsRequestDto struct {
	Permissions []uuid.UUID `json:"permissions"`
}

func (r AuthClientAddPermissionsRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Permissions,
			validation.Required.Error("Permission UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

// Auth Client API DTOs
type AuthClientApiResponseDto struct {
	AuthClientApiUUID uuid.UUID               `json:"auth_client_api_id"`
	Api               APIResponseDto          `json:"api"`
	Permissions       []PermissionResponseDto `json:"permissions,omitempty"`
	CreatedAt         time.Time               `json:"created_at"`
}

// Add APIs to auth client request dto
type AddAuthClientApisRequest struct {
	ApiUUIDs []uuid.UUID `json:"api_uuids"`
}

func (r AddAuthClientApisRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.ApiUUIDs,
			validation.Required.Error("API UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

// Add permissions to auth client API request dto
type AddAuthClientApiPermissionsRequest struct {
	PermissionUUIDs []uuid.UUID `json:"permission_uuids"`
}

func (r AddAuthClientApiPermissionsRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PermissionUUIDs,
			validation.Required.Error("Permission UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}
