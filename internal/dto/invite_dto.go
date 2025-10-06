package dto

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// EmailRegex is a basic regex for validating email format.
// Note: This is simple and not RFC 5322-complete.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// SendInviteRequest represents the payload to invite a user.
type SendInviteRequest struct {
	Email string      `json:"email"` // Email address of the user to invite
	Roles []uuid.UUID `json:"roles"` // List of role UUIDs to assign to the invited user
}

// Validate validates the invite request fields.
func (r SendInviteRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Email,
			validation.Required.Error("Email is required"),
			validation.Length(3, 100).Error("Email must be between 3 and 100 characters"),
			validation.Match(emailRegex).Error("Invalid email format"),
		),
		validation.Field(&r.Roles,
			validation.Required.Error("Roles are required"),
			validation.Length(1, 10).Error("Must provide between 1 and 10 roles"),
		),
	)
}
