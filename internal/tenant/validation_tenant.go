package tenant

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/shared"
)

var tenantNamePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

func (r TenantCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
			validation.Match(tenantNamePattern).Error("Name must contain only lowercase letters, numbers, and hyphens"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended).Error("Status must be active, inactive, pending, or suspended"),
		),
	)
}

func (r TenantUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
			validation.Match(tenantNamePattern).Error("Name must contain only lowercase letters, numbers, and hyphens"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended).Error("Status must be active, inactive, pending, or suspended"),
		),
	)
}

func (r TenantSetStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended).Error("Status must be active, inactive, pending, or suspended"),
		),
	)
}

func (r TenantFilterDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PaginationRequestDTO),
	)
}
