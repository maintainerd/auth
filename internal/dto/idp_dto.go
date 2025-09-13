package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Identity provider output structure
type IdentityProviderResponseDto struct {
	IdentityProviderUUID uuid.UUID                 `json:"identity_provider_uuid"`
	Name                 string                    `json:"name"`
	DisplayName          string                    `json:"display_name"`
	ProviderType         string                    `json:"provider_type"`
	Identifier           string                    `json:"identifier"`
	Config               *datatypes.JSON           `json:"config,omitempty"`
	AuthContainer        *AuthContainerResponseDto `json:"auth_container,omitempty"`
	IsActive             bool                      `json:"is_active"`
	IsDefault            bool                      `json:"is_default"`
	CreatedAt            time.Time                 `json:"created_at"`
	UpdatedAt            time.Time                 `json:"updated_at"`
}

// Create identity provider request DTO
type IdentityProviderCreateOrUpdateRequestDto struct {
	Name              string         `json:"name"`
	DisplayName       string         `json:"display_name"`
	ProviderType      string         `json:"provider_type"`
	Config            datatypes.JSON `json:"config"`
	IsActive          bool           `json:"is_active"`
	AuthContainerUUID string         `json:"auth_container_uuid"`
}

// Validation
func (r IdentityProviderCreateOrUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(8, 200).Error("Display name must be between 8 and 200 characters"),
		),
		validation.Field(&r.ProviderType,
			validation.Required.Error("Provider type is required"),
			validation.Length(8, 200).Error("Provider type must be between 8 and 200 characters"),
		),
		validation.Field(&r.Config,
			validation.Required.Error("Config is required"),
		),
		validation.Field(&r.IsActive,
			validation.In(true, false).Error("Is active is required"),
		),
		validation.Field(&r.AuthContainerUUID,
			validation.Required.Error("Auth container UUID is required"),
		),
	)
}

// Identity provider listing / filter DTO
type IdentityProviderFilterDto struct {
	Name              *string `json:"name"`
	DisplayName       *string `json:"display_name"`
	ProviderType      *string `json:"provider_type"`
	Identifier        *string `json:"identifier"`
	IsActive          *bool   `json:"is_active"`
	IsDefault         *bool   `json:"is_default"`
	AuthContainerUUID *string `json:"auth_container_uuid"`

	// Pagination and sorting
	PaginationRequestDto
}
