package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/maintainerd/auth/internal/model"
)

// Identity provider list response structure (without config and tenant)
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

// Identity provider detail response structure (with config and tenant)
type IdentityProviderDetailResponseDTO struct {
	IdentityProviderUUID uuid.UUID          `json:"identity_provider_id"`
	Name                 string             `json:"name"`
	DisplayName          string             `json:"display_name"`
	Provider             string             `json:"provider"`
	ProviderType         string             `json:"provider_type"`
	Identifier           string             `json:"identifier"`
	Config               *datatypes.JSON    `json:"config,omitempty"`
	Tenant               *TenantResponseDTO `json:"tenant,omitempty"`
	Status               string             `json:"status"`
	IsDefault            bool               `json:"is_default"`
	IsSystem             bool               `json:"is_system"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

// Create identity provider request DTO
type IdentityProviderCreateRequestDTO struct {
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name"`
	Provider     string         `json:"provider"`
	ProviderType string         `json:"provider_type"`
	Config       datatypes.JSON `json:"config"`
	Status       string         `json:"status"`
	TenantUUID   string         `json:"tenant_id"`
}

// Validation for create request
func (r IdentityProviderCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(8, 200).Error("Display name must be between 8 and 200 characters"),
		),
		validation.Field(&r.Provider,
			validation.Required.Error("Provider is required"),
			validation.In(model.IDPProviderInternal, model.IDPProviderCognito, model.IDPProviderAuth0, model.IDPProviderGoogle, model.IDPProviderFacebook, model.IDPProviderGitHub, model.IDPProviderMicrosoft, model.IDPProviderApple, model.IDPProviderLinkedIn, model.IDPProviderTwitter).Error("Provider must be one of: internal, cognito, auth0, google, facebook, github, microsoft, apple, linkedin, twitter"),
		),
		validation.Field(&r.ProviderType,
			validation.Required.Error("Provider type is required"),
			validation.In(model.IDPTypeIdentity, model.IDPTypeSocial).Error("Provider type must be either 'identity' or 'social'"),
		),
		validation.Field(&r.Config,
			validation.Required.Error("Config is required"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
		),
		validation.Field(&r.TenantUUID,
			validation.Required.Error("Tenant UUID is required"),
			is.UUID.Error("Tenant UUID must be a valid UUID"),
		),
	)
}

// Update identity provider request DTO (without tenant_id)
type IdentityProviderUpdateRequestDTO struct {
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name"`
	Provider     string         `json:"provider"`
	ProviderType string         `json:"provider_type"`
	Config       datatypes.JSON `json:"config"`
	Status       string         `json:"status"`
}

// Validation for update request
func (r IdentityProviderUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(8, 200).Error("Display name must be between 8 and 200 characters"),
		),
		validation.Field(&r.Provider,
			validation.Required.Error("Provider is required"),
			validation.In(model.IDPProviderInternal, model.IDPProviderCognito, model.IDPProviderAuth0, model.IDPProviderGoogle, model.IDPProviderFacebook, model.IDPProviderGitHub, model.IDPProviderMicrosoft, model.IDPProviderApple, model.IDPProviderLinkedIn, model.IDPProviderTwitter).Error("Provider must be one of: internal, cognito, auth0, google, facebook, github, microsoft, apple, linkedin, twitter"),
		),
		validation.Field(&r.ProviderType,
			validation.Required.Error("Provider type is required"),
			validation.In(model.IDPTypeIdentity, model.IDPTypeSocial).Error("Provider type must be either 'identity' or 'social'"),
		),
		validation.Field(&r.Config,
			validation.Required.Error("Config is required"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
		),
	)
}

// Identity provider status update DTO
type IdentityProviderStatusUpdateDTO struct {
	Status string `json:"status"`
}

func (r IdentityProviderStatusUpdateDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
		),
	)
}

// Identity provider listing / filter DTO
type IdentityProviderFilterDTO struct {
	Name         *string  `json:"name"`
	DisplayName  *string  `json:"display_name"`
	Provider     []string `json:"provider"`
	ProviderType *string  `json:"provider_type"`
	Identifier   *string  `json:"identifier"`
	Status       []string `json:"status"`
	IsDefault    *bool    `json:"is_default"`
	IsSystem     *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the identity provider filter DTO.
func (f IdentityProviderFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Provider,
			validation.When(len(f.Provider) > 0,
				validation.Each(validation.In(model.IDPProviderInternal, model.IDPProviderCognito, model.IDPProviderAuth0, model.IDPProviderGoogle, model.IDPProviderFacebook, model.IDPProviderGitHub, model.IDPProviderMicrosoft, model.IDPProviderApple, model.IDPProviderLinkedIn, model.IDPProviderTwitter).Error("Invalid identity provider")),
			),
		),
		validation.Field(&f.ProviderType,
			validation.When(f.ProviderType != nil,
				validation.In(model.IDPTypeIdentity, model.IDPTypeSocial).Error("Provider type must be one of: identity, social"),
			),
		),
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}
