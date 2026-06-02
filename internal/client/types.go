package client

import (
	"time"

	"github.com/google/uuid"
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

// Add permissions to API key API request dto
type AddAPIKeyPermissionsRequestDTO struct {
	PermissionUUIDs []uuid.UUID `json:"permission_uuids"`
}

// API key status update DTO
type APIKeyStatusUpdateDTO struct {
	Status string `json:"status"`
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

type APIKeyUpdateRequestDTO struct {
	Name        *string        `json:"name,omitempty"`
	Description *string        `json:"description,omitempty"`
	Config      datatypes.JSON `json:"config,omitempty"`
	ExpiresAt   *time.Time     `json:"expires_at,omitempty"`
	RateLimit   *int           `json:"rate_limit,omitempty"`
	Status      *string        `json:"status,omitempty"`
}

// Query parameter DTOs
type APIKeyGetRequestDTO struct {
	PaginationRequestDTO
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
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

// Create or update auth client URI request DTO
type ClientURICreateOrUpdateRequestDTO struct {
	URI  string `json:"uri"`
	Type string `json:"type"`
}

// Validation

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

// Add permissions to auth client request dto
type ClientAddPermissionsRequestDTO struct {
	Permissions []uuid.UUID `json:"permissions"`
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

// Add permissions to auth client API request dto
type AddClientAPIPermissionsRequestDTO struct {
	PermissionUUIDs []uuid.UUID `json:"permission_uuids"`
}

type APIResponseDTO struct {
	APIUUID     uuid.UUID `json:"api_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	APIType     string    `json:"api_type"`
	Identifier  string    `json:"identifier"`
	Status      string    `json:"status"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PermissionResponseDTO struct {
	PermissionUUID uuid.UUID `json:"permission_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Status         string    `json:"status"`
	IsDefault      bool      `json:"is_default"`
	IsSystem       bool      `json:"is_system"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type IdentityProviderResponseDTO struct {
	IdentityProviderUUID uuid.UUID `json:"identity_provider_id"`
	Name                 string    `json:"name"`
	DisplayName          string    `json:"display_name"`
	Provider             string    `json:"provider"`
	ProviderType         string    `json:"provider_type"`
	Identifier           string    `json:"identifier"`
	Status               string    `json:"status"`
	IsDefault            bool      `json:"is_default"`
	IsSystem             bool      `json:"is_system"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}
