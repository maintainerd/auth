package dto

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/maintainerd/auth/internal/platform/security"
)

// SendEmailVerificationRequestDTO represents the request payload to (re)send an email verification code.
type SendEmailVerificationRequestDTO struct {
	Email string `json:"email"`
}

func (r *SendEmailVerificationRequestDTO) Validate() error {
	r.Email = security.SanitizeInput(r.Email)

	return validation.ValidateStruct(r,
		validation.Field(&r.Email,
			validation.Required.Error("Email is required"),
			is.Email.Error("Email must be a valid email address"),
			validation.Length(1, 255).Error("Email must not exceed 255 characters"),
		),
	)
}

// SendEmailVerificationResponseDTO represents the response for a send-verification request.
type SendEmailVerificationResponseDTO struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// VerifyEmailRequestDTO represents the request payload to consume a verification code.
type VerifyEmailRequestDTO struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

func (r *VerifyEmailRequestDTO) Validate() error {
	r.Email = security.SanitizeInput(r.Email)
	r.OTP = security.SanitizeInput(r.OTP)

	return validation.ValidateStruct(r,
		validation.Field(&r.Email,
			validation.Required.Error("Email is required"),
			is.Email.Error("Email must be a valid email address"),
			validation.Length(1, 255).Error("Email must not exceed 255 characters"),
		),
		validation.Field(&r.OTP,
			validation.Required.Error("Verification code is required"),
			validation.Length(4, 12).Error("Verification code must be between 4 and 12 characters"),
		),
	)
}

// VerifyEmailResponseDTO represents the response after a successful verification.
type VerifyEmailResponseDTO struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}
