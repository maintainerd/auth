package client

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/auth/internal/shared"
)

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
			validation.When(r.IdentityProviderUUID != "",
				is.UUID.Error("Identity Provider UUID must be a valid UUID"),
			),
		),
	)
}

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

func (r AddClientIdentityProviderRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.IdentityProviderUUID,
			validation.Required.Error("Identity provider ID is required"),
		),
		validation.Field(&r.DisplayOrder,
			validation.Min(0).Error("Display order must be zero or greater"),
		),
	)
}

func (r UpdateClientIdentityProviderRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.DisplayOrder,
			validation.Min(0).Error("Display order must be zero or greater"),
		),
	)
}

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

func (r ClientAddPermissionsRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Permissions,
			validation.Required.Error("Permission UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

func (r AddClientAPIsRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.APIUUIDs,
			validation.Required.Error("API UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

func (r AddClientAPIPermissionsRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PermissionUUIDs,
			validation.Required.Error("Permission UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}
