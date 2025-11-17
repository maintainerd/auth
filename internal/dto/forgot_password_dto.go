package dto

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/auth/internal/util"
)

// ForgotPasswordRequestDto represents the request payload for forgot password
type ForgotPasswordRequestDto struct {
	Email string `json:"email"`
}

func (r *ForgotPasswordRequestDto) Validate() error {
	// Sanitize inputs first
	r.Email = util.SanitizeInput(r.Email)

	return validation.ValidateStruct(r,
		validation.Field(&r.Email,
			validation.Required.Error("Email is required"),
			is.Email.Error("Email must be a valid email address"),
			validation.Length(1, 255).Error("Email must not exceed 255 characters"),
		),
	)
}

// ForgotPasswordResponseDto represents the response for forgot password request
type ForgotPasswordResponseDto struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}
