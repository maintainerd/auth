package client

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/datatypes"
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
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
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
		validation.Field(&dto.Status, validation.In(shared.StatusActive, shared.StatusInactive)),
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
		validation.Field(&dto.Status, validation.In(shared.StatusActive, shared.StatusInactive)),
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
		validation.Field(&dto.Status, validation.In(shared.StatusActive, shared.StatusInactive)),
	)
}

type ClientSecretResponseDTO struct {
	ClientID     string  `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
}

// ClientCreateSecretResponseDTO is returned exactly once at client creation.
// The plaintext secret is never stored and cannot be retrieved again.
type ClientCreateSecretResponseDTO struct {
	ClientUUID   string `json:"client_uuid"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// RotateSecretRequestDTO controls secret rotation behaviour.
type RotateSecretRequestDTO struct {
	// GracePeriodHours keeps the old secret valid for this many hours (0 = revoke immediately).
	GracePeriodHours int `json:"grace_period_hours"`
}

// RotateSecretResponseDTO is returned exactly once after rotation.
type RotateSecretResponseDTO struct {
	ClientID                string  `json:"client_id"`
	ClientSecret            string  `json:"client_secret"`
	PreviousSecretExpiresAt *string `json:"previous_secret_expires_at,omitempty"`
}

type ClientURIResponseDTO struct {
	ClientURIUUID uuid.UUID `json:"uri_id"`
	URI           string    `json:"uri"`
	Type          string    `json:"type"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ClientURIsResponseDTO struct {
	URIs []ClientURIResponseDTO `json:"uris"`
}

type ClientAPIsResponseDTO struct {
	APIs []ClientAPIResponseDTO `json:"apis"`
}

type ClientAPIPermissionsResponseDTO struct {
	Permissions []PermissionResponseDTO `json:"permissions"`
}

// Auth client output structure
type ClientResponseDTO struct {
	ClientUUID       uuid.UUID                    `json:"client_id"`
	Name             string                       `json:"name"`
	DisplayName      string                       `json:"display_name"`
	ClientType       string                       `json:"client_type"`
	Domain           *string                      `json:"domain,omitempty"`
	URIs             []ClientURIResponseDTO       `json:"uris,omitempty"`
	IdentityProvider *IdentityProviderResponseDTO `json:"identity_provider,omitempty"`
	Permissions      *[]PermissionResponseDTO     `json:"permissions,omitempty"`
	Status           string                       `json:"status"`
	IsDefault        bool                         `json:"is_default"`
	IsSystem         bool                         `json:"is_system"`
	CreatedAt        time.Time                    `json:"created_at"`
	UpdatedAt        time.Time                    `json:"updated_at"`
}

// Create auth client request DTO
type ClientCreateRequestDTO struct {
	Name                 string         `json:"name"`
	DisplayName          string         `json:"display_name"`
	ClientType           string         `json:"client_type"`
	Domain               string         `json:"domain"`
	Config               datatypes.JSON `json:"config"`
	Status               string         `json:"status"`
	IdentityProviderUUID string         `json:"identity_provider_id"`
}

// Validation
func (r ClientCreateRequestDTO) Validate() error {
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
			validation.In(shared.ClientTypeTraditional, shared.ClientTypeSPA, shared.ClientTypeMobile, shared.ClientTypeM2M).Error("Invalid client Type"),
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
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),
		validation.Field(&r.IdentityProviderUUID,
			validation.Required.Error("Identity Provider UUID is required"),
			is.UUID.Error("Identity Provider UUID must be a valid UUID"),
		),
	)
}

// Update auth client request DTO
type ClientUpdateRequestDTO struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name"`
	ClientType  string         `json:"client_type"`
	Domain      string         `json:"domain"`
	Config      datatypes.JSON `json:"config"`
	Status      string         `json:"status"`
}

// Validation
func (r ClientUpdateRequestDTO) Validate() error {
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
			validation.In(shared.ClientTypeTraditional, shared.ClientTypeSPA, shared.ClientTypeMobile, shared.ClientTypeM2M).Error("Client Type is required"),
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
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),
	)
}

// Create or update auth client URI request DTO
type ClientURICreateOrUpdateRequestDTO struct {
	URI  string `json:"uri"`
	Type string `json:"type"`
}

// Validation
func (r ClientURICreateOrUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.URI,
			validation.Required.Error("URI is required"),
			validation.Length(5, 200).Error("URI must be between 5 and 200 characters"),
		),
		validation.Field(&r.Type,
			validation.Required.Error("Type is required"),
			validation.In(shared.ClientURITypeRedirect, shared.ClientURITypeOrigin, shared.ClientURITypeLogout, shared.ClientURITypeLogin, shared.ClientURITypeCORSOrigin).Error("Type must be one of: redirect-uri, origin-uri, logout-uri, login-uri, cors-origin-uri"),
		),
	)
}

// Auth client listing / filter DTO
type ClientFilterDTO struct {
	Name                 *string  `json:"name"`
	DisplayName          *string  `json:"display_name"`
	ClientType           []string `json:"client_type"`
	IdentityProviderUUID *string  `json:"identity_provider_id"`
	Status               []string `json:"status"`
	IsDefault            *bool    `json:"is_default"`
	IsSystem             *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the client filter DTO.
func (f ClientFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.ClientType,
			validation.When(len(f.ClientType) > 0,
				validation.Each(validation.In(shared.ClientTypeTraditional, shared.ClientTypeSPA, shared.ClientTypeMobile, shared.ClientTypeM2M).Error("Client type must be one of: traditional, spa, mobile, m2m")),
			),
		),
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.IdentityProviderUUID,
			validation.When(f.IdentityProviderUUID != nil,
				is.UUID.Error("Identity provider ID must be a valid UUID"),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}

// Add permissions to auth client request dto
type ClientAddPermissionsRequestDTO struct {
	Permissions []uuid.UUID `json:"permissions"`
}

func (r ClientAddPermissionsRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Permissions,
			validation.Required.Error("Permission UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

// Auth Client API DTOs
type ClientAPIResponseDTO struct {
	ClientAPIUUID uuid.UUID               `json:"client_api_id"`
	API           APIResponseDTO          `json:"api"`
	Permissions   []PermissionResponseDTO `json:"permissions,omitempty"`
	CreatedAt     time.Time               `json:"created_at"`
}

// Add APIs to auth client request dto
type AddClientAPIsRequestDTO struct {
	APIUUIDs []uuid.UUID `json:"api_uuids"`
}

func (r AddClientAPIsRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.APIUUIDs,
			validation.Required.Error("API UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

// Add permissions to auth client API request dto
type AddClientAPIPermissionsRequestDTO struct {
	PermissionUUIDs []uuid.UUID `json:"permission_uuids"`
}

func (r AddClientAPIPermissionsRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PermissionUUIDs,
			validation.Required.Error("Permission UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}
