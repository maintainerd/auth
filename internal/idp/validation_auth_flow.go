package idp

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/auth/internal/shared"
)

func (r AuthFlowCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Signup flow name is required"),
			validation.Length(1, 100).Error("Signup flow name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
		validation.Field(&r.ClientUUID,
			validation.Required.Error("Auth client UUID is required"),
			is.UUID.Error("Invalid auth client UUID format"),
		),
	)
}

func (r AuthFlowUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Signup flow name is required"),
			validation.Length(1, 100).Error("Signup flow name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

func (r AuthFlowUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

func (f AuthFlowFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.ClientUUID,
			validation.When(f.ClientUUID != nil,
				is.UUID.Error("Client ID must be a valid UUID"),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}

func (r AuthFlowAssignRolesRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.RoleUUIDs,
			validation.Required.Error("Role UUIDs are required"),
			validation.Length(1, 0).Error("At least one role UUID is required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

func (r AuthFlowAssignCallbackURIsRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.ClientURIUUIDs,
			validation.Required.Error("Client URI UUIDs are required"),
			validation.Length(1, 0).Error("At least one client URI UUID is required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}
