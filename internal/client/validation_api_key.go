package client

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/auth/internal/shared"
)

func (r AddAPIKeyAPIsRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.APIUUIDs,
			validation.Required.Error("API UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

func (r AddAPIKeyPermissionsRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PermissionUUIDs,
			validation.Required.Error("Permission UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

func (r APIKeyStatusUpdateDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
		),
	)
}

func (dto APIKeyCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Name, validation.Required, validation.Length(1, 100)),
		validation.Field(&dto.Description, validation.Length(0, 500)),
		validation.Field(&dto.Status, validation.In(shared.StatusActive, shared.StatusInactive)),
	)
}

func (dto APIKeyUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Name, validation.Length(1, 100)),
		validation.Field(&dto.Description, validation.Length(0, 500)),
		validation.Field(&dto.Status, validation.In(shared.StatusActive, shared.StatusInactive)),
	)
}

func (dto APIKeyGetRequestDTO) Validate() error {
	if err := dto.PaginationRequestDTO.Validate(); err != nil {
		return err
	}
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Status, validation.In(shared.StatusActive, shared.StatusInactive)),
	)
}
