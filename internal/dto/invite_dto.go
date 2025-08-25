package dto

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// EmailRegex is a basic regex for validating email format.
// Note: This is simple and not RFC 5322-complete.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// SendPrivateInviteRequest represents the payload to invite an internal user.
type SendPrivateInviteRequest struct {
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

// Validate validates the invite request fields.
func (r SendPrivateInviteRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Email,
			validation.Required.Error("Email is required"),
			validation.Length(3, 100).Error("Email must be between 3 and 100 characters"),
			validation.Match(emailRegex).Error("Invalid email format"),
		),
		validation.Field(&r.Roles,
			validation.Required.Error("Roles are required"),
			validation.Each(validation.Length(1, 100)), // You can add more specific validation for UUID format here if needed
		),
	)
}
