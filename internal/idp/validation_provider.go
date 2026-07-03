package idp

import (
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// isExternalProviderType reports whether a provider type needs upstream OIDC
// credentials (issuer + provider_client_id). System providers do not.
func isExternalProviderType(providerType string) bool {
	return providerType == shared.IDPTypeSocial || providerType == shared.IDPTypeEnterprise
}

// Validation for create request
func (r IdentityProviderCreateRequestDTO) Validate() error {
	requireExternalCreds := isExternalProviderType(r.ProviderType) && r.Status == shared.StatusActive
	requireTokenFederation := r.AllowTokenFederation && r.Status == shared.StatusActive
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
			validation.In(shared.IDPProviderMaintainerd, shared.IDPProviderCognito, shared.IDPProviderAuth0, shared.IDPProviderGoogle, shared.IDPProviderFacebook, shared.IDPProviderGitHub, shared.IDPProviderGitLab, shared.IDPProviderMicrosoft, shared.IDPProviderApple, shared.IDPProviderLinkedIn, shared.IDPProviderTwitter).Error("Provider must be one of: maintainerd, cognito, auth0, google, facebook, github, gitlab, microsoft, apple, linkedin, twitter"),
		),
		validation.Field(&r.ProviderType,
			validation.Required.Error("Provider type is required"),
			validation.In(shared.IDPTypeSystem, shared.IDPTypeSocial, shared.IDPTypeEnterprise).Error("Provider type must be one of: system, social, enterprise"),
		),
		validation.Field(&r.Issuer,
			validation.When(requireExternalCreds || requireTokenFederation, validation.Required.Error("Issuer is required for active social/enterprise or token-federation providers")),
			validation.When(strings.TrimSpace(r.Issuer) != "", is.URL.Error("Issuer must be a valid URL")),
		),
		validation.Field(&r.ProviderClientID,
			validation.When(requireExternalCreds, validation.Required.Error("Provider client ID is required for active social/enterprise providers")),
		),
		validation.Field(&r.AllowedAudiences,
			validation.When(requireTokenFederation,
				validation.By(func(value interface{}) error {
					auds, ok := value.([]string)
					if !ok || len(auds) == 0 {
						return validation.NewError("validation_required", "At least one allowed audience is required when token federation is enabled")
					}
					return nil
				}),
			),
		),
		validation.Field(&r.EmailDomains,
			validation.When(len(r.EmailDomains) > 0,
				validation.Each(is.Domain.Error("Each email domain must be a valid domain")),
			),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
		),
	)
}

// Validation for update request
func (r IdentityProviderUpdateRequestDTO) Validate() error {
	requireExternalCreds := isExternalProviderType(r.ProviderType) && r.Status == shared.StatusActive
	requireTokenFederation := r.AllowTokenFederation && r.Status == shared.StatusActive
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
			validation.In(shared.IDPProviderMaintainerd, shared.IDPProviderCognito, shared.IDPProviderAuth0, shared.IDPProviderGoogle, shared.IDPProviderFacebook, shared.IDPProviderGitHub, shared.IDPProviderGitLab, shared.IDPProviderMicrosoft, shared.IDPProviderApple, shared.IDPProviderLinkedIn, shared.IDPProviderTwitter).Error("Provider must be one of: maintainerd, cognito, auth0, google, facebook, github, gitlab, microsoft, apple, linkedin, twitter"),
		),
		validation.Field(&r.ProviderType,
			validation.Required.Error("Provider type is required"),
			validation.In(shared.IDPTypeSystem, shared.IDPTypeSocial, shared.IDPTypeEnterprise).Error("Provider type must be one of: system, social, enterprise"),
		),
		validation.Field(&r.Issuer,
			validation.When(requireExternalCreds || requireTokenFederation, validation.Required.Error("Issuer is required for active social/enterprise or token-federation providers")),
			validation.When(strings.TrimSpace(r.Issuer) != "", is.URL.Error("Issuer must be a valid URL")),
		),
		validation.Field(&r.ProviderClientID,
			validation.When(requireExternalCreds, validation.Required.Error("Provider client ID is required for active social/enterprise providers")),
		),
		validation.Field(&r.AllowedAudiences,
			validation.When(requireTokenFederation,
				validation.By(func(value interface{}) error {
					auds, ok := value.([]string)
					if !ok || len(auds) == 0 {
						return validation.NewError("validation_required", "At least one allowed audience is required when token federation is enabled")
					}
					return nil
				}),
			),
		),
		validation.Field(&r.EmailDomains,
			validation.When(len(r.EmailDomains) > 0,
				validation.Each(is.Domain.Error("Each email domain must be a valid domain")),
			),
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
				validation.Each(validation.In(shared.IDPProviderMaintainerd, shared.IDPProviderCognito, shared.IDPProviderAuth0, shared.IDPProviderGoogle, shared.IDPProviderFacebook, shared.IDPProviderGitHub, shared.IDPProviderGitLab, shared.IDPProviderMicrosoft, shared.IDPProviderApple, shared.IDPProviderLinkedIn, shared.IDPProviderTwitter).Error("Invalid identity provider")),
			),
		),
		validation.Field(&f.ProviderType,
			validation.When(f.ProviderType != nil,
				validation.In(shared.IDPTypeSystem, shared.IDPTypeSocial, shared.IDPTypeEnterprise).Error("Provider type must be one of: system, social, enterprise"),
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
