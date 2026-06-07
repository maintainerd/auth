package user

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/shared"
)

// Validate validates the user pool create request DTO.
func (dto UserPoolCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Name,
			validation.Required.Error("Name is required"),
			validation.RuneLength(2, 100).Error("Name must be 2-100 characters")),
		validation.Field(&dto.DisplayName,
			validation.RuneLength(0, 150).Error("Display name must be at most 150 characters")),
		validation.Field(&dto.Status,
			validation.When(dto.Status != "",
				validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"))),
	)
}

// Validate validates the user pool set-status request DTO.
func (dto UserPoolSetStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
	)
}

// Validate validates the user pool update request DTO.
func (dto UserPoolUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Name,
			validation.Required.Error("Name is required"),
			validation.RuneLength(2, 100).Error("Name must be 2-100 characters")),
		validation.Field(&dto.DisplayName,
			validation.RuneLength(0, 150).Error("Display name must be at most 150 characters")),
		validation.Field(&dto.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
	)
}
