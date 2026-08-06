package client

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// maxClientConfigBytes caps the free-form config blob. It mirrors
// security-relevant columns (grant types, auth method, TTLs), so an unbounded
// body is both a cheap DoS and a blob nobody can review.
const maxClientConfigBytes = 16 * 1024

// clientDomainPattern accepts a bare hostname (auth.example.com) or an absolute
// https URL. The domain becomes the token `iss` and is compared in the
// private_key_jwt audience check, so arbitrary text there is load-bearing.
var clientDomainPattern = regexp.MustCompile(`^(https://)?[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?(:[0-9]{1,5})?(/.*)?$`)

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
			// Required, or an empty string passes In() and only fails at the DB
			// CHECK constraint — surfacing as a 500 instead of a 400.
			validation.Required.Error("Client Type is required"),
			validation.In(shared.ClientTypeTraditional, shared.ClientTypeSPA, shared.ClientTypeMobile, shared.ClientTypeM2M).Error("Invalid client Type"),
		),
		validation.Field(&r.Domain,
			validation.Required.Error("Domain is required"),
			// varchar(253) in the DB; it is also used as the token issuer, so it
			// must be a real host/URL rather than arbitrary text.
			validation.Length(3, 253).Error("Domain must be between 3 and 253 characters"),
			validation.Match(clientDomainPattern).Error("Domain must be a hostname or https URL"),
		),
		validation.Field(&r.Config,
			validation.Required.Error("Config is required"),
			validation.Length(0, maxClientConfigBytes).Error("Config must not exceed 16KB"),
			// The advanced keys inside config are mirrored into runtime columns; a
			// malformed one would otherwise be dropped silently.
			validation.By(validateClientConfig),
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
		validation.Field(&r.BrandingUUID,
			validation.When(r.BrandingUUID != nil && *r.BrandingUUID != "",
				is.UUID.Error("branding_id must be a valid UUID"),
			),
		),
		validation.Field(&r.ServiceUUID,
			validation.When(r.ServiceUUID != nil && *r.ServiceUUID != "",
				is.UUID.Error("service_id must be a valid UUID"),
			),
		),
		validation.Field(&r.BackchannelLogoutURI,
			validation.When(r.BackchannelLogoutURI != nil && *r.BackchannelLogoutURI != "",
				validation.Length(1, 2048).Error("backchannel_logout_uri must be at most 2048 characters"),
				is.URL.Error("backchannel_logout_uri must be a valid URL"),
			),
		),
		validation.Field(&r.FrontchannelLogoutURI,
			validation.When(r.FrontchannelLogoutURI != nil && *r.FrontchannelLogoutURI != "",
				validation.Length(1, 2048).Error("frontchannel_logout_uri must be at most 2048 characters"),
				is.URL.Error("frontchannel_logout_uri must be a valid URL"),
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
			validation.Required.Error("Client Type is required"),
			validation.In(shared.ClientTypeTraditional, shared.ClientTypeSPA, shared.ClientTypeMobile, shared.ClientTypeM2M).Error("Invalid client Type"),
		),
		validation.Field(&r.Domain,
			validation.Required.Error("Domain is required"),
			// varchar(253) in the DB; it is also used as the token issuer, so it
			// must be a real host/URL rather than arbitrary text.
			validation.Length(3, 253).Error("Domain must be between 3 and 253 characters"),
			validation.Match(clientDomainPattern).Error("Domain must be a hostname or https URL"),
		),
		validation.Field(&r.Config,
			validation.Required.Error("Config is required"),
			validation.Length(0, maxClientConfigBytes).Error("Config must not exceed 16KB"),
			// The advanced keys inside config are mirrored into runtime columns; a
			// malformed one would otherwise be dropped silently.
			validation.By(validateClientConfig),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),
		validation.Field(&r.ServiceUUID,
			validation.When(r.ServiceUUID != nil && *r.ServiceUUID != "",
				is.UUID.Error("service_id must be a valid UUID"),
			),
		),
		validation.Field(&r.BackchannelLogoutURI,
			validation.When(r.BackchannelLogoutURI != nil && *r.BackchannelLogoutURI != "",
				validation.Length(1, 2048).Error("backchannel_logout_uri must be at most 2048 characters"),
				is.URL.Error("backchannel_logout_uri must be a valid URL"),
			),
		),
		validation.Field(&r.FrontchannelLogoutURI,
			validation.When(r.FrontchannelLogoutURI != nil && *r.FrontchannelLogoutURI != "",
				validation.Length(1, 2048).Error("frontchannel_logout_uri must be at most 2048 characters"),
				is.URL.Error("frontchannel_logout_uri must be a valid URL"),
			),
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
			validation.In(shared.ClientURITypeRedirect, shared.ClientURITypeOrigin, shared.ClientURITypeLogout, shared.ClientURITypeLogin, shared.ClientURITypeCORSOrigin).Error("Type must be one of: redirect_uri, origin_uri, logout_uri, login_uri, cors_origin_uri"),
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

// maxSecretGracePeriodHours caps how long a rotated-out client secret stays
// valid. It was unbounded, so a value like 876000 would keep a compromised
// secret working for a century — which defeats the point of rotating.
//
// clientService.RotateSecret is the enforcement point, because every transport
// reaches it and the gRPC handler does not go through this DTO. The rule below
// is kept so an HTTP caller still gets a field-level 400.
const maxSecretGracePeriodHours = 168 // 7 days

func (r RotateSecretRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.GracePeriodHours,
			validation.Min(0).Error("grace_period_hours cannot be negative"),
			validation.Max(maxSecretGracePeriodHours).Error("grace_period_hours must not exceed 168 (7 days)"),
		),
	)
}

// Validate ensures the caller named a real status rather than relying on a
// server-side toggle.
func (r ClientSetStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required,
			validation.In(shared.StatusActive, shared.StatusInactive),
		),
	)
}
