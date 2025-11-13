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

type AuthClientConfigResponseDto struct {
	Config datatypes.JSON `json:"config"`
}

type AuthClientRedirectURIResponseDto struct {
	AuthClientRedirectURIUUID uuid.UUID `json:"redirect_uri_id"`
	RedirectURI               string    `json:"redirect_uri"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

// Auth client output structure
type AuthClientResponseDto struct {
	AuthClientUUID   uuid.UUID                          `json:"auth_client_id"`
	Name             string                             `json:"name"`
	DisplayName      string                             `json:"display_name"`
	ClientType       string                             `json:"client_type"`
	Domain           *string                            `json:"domain,omitempty"`
	RedirectURIs     []AuthClientRedirectURIResponseDto `json:"redirect_uris,omitempty"`
	IdentityProvider *IdentityProviderResponseDto       `json:"identity_provider,omitempty"`
	Permissions      *[]PermissionResponseDto           `json:"permissions,omitempty"`
	IsActive         bool                               `json:"is_active"`
	IsDefault        bool                               `json:"is_default"`
	CreatedAt        time.Time                          `json:"created_at"`
	UpdatedAt        time.Time                          `json:"updated_at"`
}

// Create auth client request DTO
type AuthClientCreateRequestDto struct {
	Name                 string         `json:"name"`
	DisplayName          string         `json:"display_name"`
	ClientType           string         `json:"client_type"`
	Domain               string         `json:"domain"`
	Config               datatypes.JSON `json:"config"`
	IsActive             bool           `json:"is_active"`
	IdentityProviderUUID string         `json:"identity_provider_uuid"`
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
		validation.Field(&r.IsActive,
			validation.In(true, false).Error("Is active is required"),
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
	IsActive    bool           `json:"is_active"`
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
		validation.Field(&r.IsActive,
			validation.In(true, false).Error("Is active is required"),
		),
	)
}

// Create or update auth client redirect uri request DTO
type AuthClientRedirectURICreateOrUpdateRequestDto struct {
	RedirectURI string `json:"redirect_uri"`
}

// Validation
func (r AuthClientRedirectURICreateOrUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.RedirectURI,
			validation.Required.Error("Redirect URI is required"),
			validation.Length(5, 200).Error("Redirect URI must be between 5 and 200 characters"),
		),
	)
}

// Auth client listing / filter DTO
type AuthClientFilterDto struct {
	Name                 *string `json:"name"`
	DisplayName          *string `json:"display_name"`
	ClientType           *string `json:"client_type"`
	IdentityProviderUUID *string `json:"identity_provider_uuid"`
	IsActive             *bool   `json:"is_active"`
	IsDefault            *bool   `json:"is_default"`

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
