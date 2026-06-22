package authn

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

func (dto ResetPasswordRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.NewPassword, validation.Required.Error("New password is required")),
		// Token is optional in request body - can come from signed URL instead
	)
}


