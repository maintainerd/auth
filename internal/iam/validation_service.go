package iam

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

func (r ServiceCreateOrUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Service name is required"),
			validation.Length(3, 50).Error("Service name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(3, 100).Error("Display name must be between 3 and 100 characters"),
		),
		validation.Field(&r.Description,
			validation.Length(0, 255).Error("Description must be at most 255 characters"),
		),
		validation.Field(&r.Version,
			validation.Required.Error("Version is required"),
			// The column is VARCHAR(20); without this a longer value reached Postgres
			// as a 22001 string-data-right-truncation and surfaced as a 500. Length
			// only, no format rule — the seeder writes "v1", which is not semver.
			validation.Length(1, 20).Error("Version must be between 1 and 20 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusMaintenance, shared.StatusDeprecated, shared.StatusInactive).Error("Status must be one of: active, maintenance, deprecated, inactive"),
		),
	)
}

func (f ServiceFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusMaintenance, shared.StatusDeprecated, shared.StatusInactive).Error("Status must be one of: active, maintenance, deprecated, inactive")),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}

func (r ServiceStatusUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusMaintenance, shared.StatusDeprecated, shared.StatusInactive).Error("Status must be one of: active, maintenance, deprecated, inactive"),
		),
	)
}
