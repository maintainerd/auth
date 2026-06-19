package iam

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/auth/internal/shared"
)

func (r RoleCreateOrUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Role name is required"),
			validation.Length(3, 20).Error("Role name must be between 3 and 20 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 100).Error("Description must be between 8 and 100 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

func (r RoleAddPermissionsRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Permissions,
			validation.Required.Error("Permission UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

func (f RoleFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}
