package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// EmailConfigResponseDTO is the JSON representation of an email config record.
type EmailConfigResponseDTO struct {
	EmailConfigID string    `json:"email_config_id"`
	Provider      string    `json:"provider"`
	Host          string    `json:"host"`
	Port          int       `json:"port"`
	Username      string    `json:"username"`
	FromAddress   string    `json:"from_address"`
	FromName      string    `json:"from_name"`
	ReplyTo       string    `json:"reply_to"`
	Encryption    string    `json:"encryption"`
	TestMode      bool      `json:"test_mode"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// EmailConfigUpdateRequestDTO is the request body for updating email config.
type EmailConfigUpdateRequestDTO struct {
	Provider    string `json:"provider"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	FromAddress string `json:"from_address"`
	FromName    string `json:"from_name"`
	ReplyTo     string `json:"reply_to"`
	Encryption  string `json:"encryption"`
	TestMode    *bool  `json:"test_mode"`
}

// Validate validates the email config update request.
func (r EmailConfigUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Provider,
			validation.Required.Error("Provider is required"),
			validation.In("smtp", "ses", "sendgrid", "mailgun", "postmark", "resend").Error("Provider must be one of: smtp, ses, sendgrid, mailgun, postmark, resend"),
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
		validation.Field(&r.Username,
			validation.Length(0, 255).Error("Username must not exceed 255 characters"),
		),
	)
}
