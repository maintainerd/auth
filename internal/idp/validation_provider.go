package idp

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/auth/internal/shared"
)

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
			validation.In(shared.IDPProviderInternal, shared.IDPProviderCognito, shared.IDPProviderAuth0, shared.IDPProviderGoogle, shared.IDPProviderFacebook, shared.IDPProviderGitHub, shared.IDPProviderMicrosoft, shared.IDPProviderApple, shared.IDPProviderLinkedIn, shared.IDPProviderTwitter).Error("Provider must be one of: internal, cognito, auth0, google, facebook, github, microsoft, apple, linkedin, twitter"),
		),
		validation.Field(&r.ProviderType,
			validation.Required.Error("Provider type is required"),
			validation.In(shared.IDPTypeIdentity, shared.IDPTypeSocial).Error("Provider type must be either 'identity' or 'social'"),
		),
		validation.Field(&r.Config,
			validation.Required.Error("Config is required"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
		),
		validation.Field(&r.TenantUUID,
			validation.Required.Error("Tenant UUID is required"),
			is.UUID.Error("Tenant UUID must be a valid UUID"),
		),
	)
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
			validation.In(shared.IDPProviderInternal, shared.IDPProviderCognito, shared.IDPProviderAuth0, shared.IDPProviderGoogle, shared.IDPProviderFacebook, shared.IDPProviderGitHub, shared.IDPProviderMicrosoft, shared.IDPProviderApple, shared.IDPProviderLinkedIn, shared.IDPProviderTwitter).Error("Provider must be one of: internal, cognito, auth0, google, facebook, github, microsoft, apple, linkedin, twitter"),
		),
		validation.Field(&r.ProviderType,
			validation.Required.Error("Provider type is required"),
			validation.In(shared.IDPTypeIdentity, shared.IDPTypeSocial).Error("Provider type must be either 'identity' or 'social'"),
		),
		validation.Field(&r.Config,
			validation.Required.Error("Config is required"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
		),
	)
}

func (r IdentityProviderStatusUpdateDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
		),
	)
}

// Validate validates the identity provider filter DTO.
func (f IdentityProviderFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Provider,
			validation.When(len(f.Provider) > 0,
				validation.Each(validation.In(shared.IDPProviderInternal, shared.IDPProviderCognito, shared.IDPProviderAuth0, shared.IDPProviderGoogle, shared.IDPProviderFacebook, shared.IDPProviderGitHub, shared.IDPProviderMicrosoft, shared.IDPProviderApple, shared.IDPProviderLinkedIn, shared.IDPProviderTwitter).Error("Invalid identity provider")),
			),
		),
		validation.Field(&f.ProviderType,
			validation.When(f.ProviderType != nil,
				validation.In(shared.IDPTypeIdentity, shared.IDPTypeSocial).Error("Provider type must be one of: identity, social"),
			),
		),
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}
