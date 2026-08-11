package notifier

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// Validate validates the email config update request.
func (r EmailConfigUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Provider,
			validation.Required.Error("Provider is required"),
			// maintainerd delivers over SMTP only; any provider (SES, Mailgun,
			// SendGrid, …) is reached through its SMTP relay.
			validation.In("smtp").Error("Provider must be smtp"),
		),
		validation.Field(&r.FromAddress,
			validation.Required.Error("From address is required"),
			is.EmailFormat.Error("From address must be a valid email"),
			validation.Length(1, 255).Error("From address must not exceed 255 characters"),
		),
		validation.Field(&r.FromName,
			validation.Length(0, 255).Error("From name must not exceed 255 characters"),
		),
		validation.Field(&r.ReplyTo,
			validation.When(r.ReplyTo != "", is.EmailFormat.Error("Reply-to must be a valid email")),
			validation.Length(0, 255).Error("Reply-to must not exceed 255 characters"),
		),
		validation.Field(&r.Encryption,
			validation.When(r.Encryption != "", validation.In("tls", "ssl", "none").Error("Encryption must be one of: tls, ssl, none")),
		),
		validation.Field(&r.Host,
			validation.Length(0, 255).Error("Host must not exceed 255 characters"),
		),
		validation.Field(&r.Port,
			validation.When(r.Port != 0, validation.Min(1), validation.Max(65535).Error("Port must be between 1 and 65535")),
		),
		validation.Field(&r.Username,
			validation.Length(0, 255).Error("Username must not exceed 255 characters"),
		),
	)
}
