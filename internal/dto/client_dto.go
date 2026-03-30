package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/maintainerd/auth/internal/model"
)

type ClientSecretResponseDto struct {
	ClientID     string  `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
}

type ClientURIResponseDto struct {
	ClientURIUUID uuid.UUID `json:"uri_id"`
	URI           string    `json:"uri"`
	Type          string    `json:"type"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ClientURIsResponseDto struct {
	URIs []ClientURIResponseDto `json:"uris"`
}

type ClientAPIsResponseDto struct {
	APIs []ClientAPIResponseDto `json:"apis"`
}

type ClientAPIPermissionsResponseDto struct {
	Permissions []PermissionResponseDto `json:"permissions"`
}

// Auth client output structure
type ClientResponseDto struct {
	ClientUUID       uuid.UUID                    `json:"client_id"`
	Name             string                       `json:"name"`
	DisplayName      string                       `json:"display_name"`
	ClientType       string                       `json:"client_type"`
	Domain           *string                      `json:"domain,omitempty"`
	URIs             []ClientURIResponseDto       `json:"uris,omitempty"`
	IdentityProvider *IdentityProviderResponseDto `json:"identity_provider,omitempty"`
	Permissions      *[]PermissionResponseDto     `json:"permissions,omitempty"`
	Status           string                       `json:"status"`
	IsDefault        bool                         `json:"is_default"`
	IsSystem         bool                         `json:"is_system"`
	CreatedAt        time.Time                    `json:"created_at"`
	UpdatedAt        time.Time                    `json:"updated_at"`
}

// Create auth client request DTO
type ClientCreateRequestDto struct {
	Name                 string         `json:"name"`
	DisplayName          string         `json:"display_name"`
	ClientType           string         `json:"client_type"`
	Domain               string         `json:"domain"`
	Config               datatypes.JSON `json:"config"`
	Status               string         `json:"status"`
	IdentityProviderUUID string         `json:"identity_provider_id"`
}

// Validation
func (r ClientCreateRequestDto) Validate() error {
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
			validation.In(model.ClientTypeTraditional, model.ClientTypeSPA, model.ClientTypeMobile, model.ClientTypeM2M).Error("Invalid client Type"),
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
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be one of: active, inactive"),
		),
		validation.Field(&r.IdentityProviderUUID,
			validation.Required.Error("Identity Provider UUID is required"),
			is.UUID.Error("Identity Provider UUID must be a valid UUID"),
		),
	)
}

// Update auth client request DTO
type ClientUpdateRequestDto struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name"`
	ClientType  string         `json:"client_type"`
	Domain      string         `json:"domain"`
	Config      datatypes.JSON `json:"config"`
	Status      string         `json:"status"`
}

// Validation
func (r ClientUpdateRequestDto) Validate() error {
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
			validation.In(model.ClientTypeTraditional, model.ClientTypeSPA, model.ClientTypeMobile, model.ClientTypeM2M).Error("Client Type is required"),
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
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be one of: active, inactive"),
		),
	)
}

// Create or update auth client URI request DTO
type ClientURICreateOrUpdateRequestDto struct {
	URI  string `json:"uri"`
	Type string `json:"type"`
}

// Validation
func (r ClientURICreateOrUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.URI,
			validation.Required.Error("URI is required"),
			validation.Length(5, 200).Error("URI must be between 5 and 200 characters"),
		),
		validation.Field(&r.Type,
			validation.Required.Error("Type is required"),
			validation.In(model.ClientURITypeRedirect, model.ClientURITypeOrigin, model.ClientURITypeLogout, model.ClientURITypeLogin, model.ClientURITypeCORSOrigin).Error("Type must be one of: redirect-uri, origin-uri, logout-uri, login-uri, cors-origin-uri"),
		),
	)
}

// Auth client listing / filter DTO
type ClientFilterDto struct {
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

// Validate validates the client filter DTO.
func (f ClientFilterDto) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.ClientType,
			validation.When(len(f.ClientType) > 0,
				validation.Each(validation.In(model.ClientTypeTraditional, model.ClientTypeSPA, model.ClientTypeMobile, model.ClientTypeM2M).Error("Client type must be one of: traditional, spa, mobile, m2m")),
			),
		),
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.IdentityProviderUUID,
			validation.When(f.IdentityProviderUUID != nil,
				is.UUID.Error("Identity provider ID must be a valid UUID"),
			),
		),
		validation.Field(&f.PaginationRequestDto),
	)
}

// Add permissions to auth client request dto
type ClientAddPermissionsRequestDto struct {
	Permissions []uuid.UUID `json:"permissions"`
}

func (r ClientAddPermissionsRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Permissions,
			validation.Required.Error("Permission UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

// Auth Client API DTOs
type ClientAPIResponseDto struct {
	ClientAPIUUID uuid.UUID               `json:"client_api_id"`
	API           APIResponseDto          `json:"api"`
	Permissions   []PermissionResponseDto `json:"permissions,omitempty"`
	CreatedAt     time.Time               `json:"created_at"`
}

// Add APIs to auth client request dto
type AddClientAPIsRequestDto struct {
	APIUUIDs []uuid.UUID `json:"api_uuids"`
}

func (r AddClientAPIsRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.APIUUIDs,
			validation.Required.Error("API UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

// Add permissions to auth client API request dto
type AddClientAPIPermissionsRequestDto struct {
	PermissionUUIDs []uuid.UUID `json:"permission_uuids"`
}

func (r AddClientAPIPermissionsRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PermissionUUIDs,
			validation.Required.Error("Permission UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}
