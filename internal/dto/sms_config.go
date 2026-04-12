package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// SMSConfigResponseDTO is the JSON representation of an SMS config record.
type SMSConfigResponseDTO struct {
	SMSConfigID string    `json:"sms_config_id"`
	Provider    string    `json:"provider"`
	AccountSID  string    `json:"account_sid"`
	FromNumber  string    `json:"from_number"`
	SenderID    string    `json:"sender_id"`
	TestMode    bool      `json:"test_mode"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SMSConfigUpdateRequestDTO is the request body for updating SMS config.
type SMSConfigUpdateRequestDTO struct {
	Provider   string `json:"provider"`
	AccountSID string `json:"account_sid"`
	AuthToken  string `json:"auth_token"`
	FromNumber string `json:"from_number"`
	SenderID   string `json:"sender_id"`
	TestMode   *bool  `json:"test_mode"`
}

// Validate validates the SMS config update request.
func (r SMSConfigUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Provider,
			validation.Required.Error("Provider is required"),
			validation.In("twilio", "sns", "vonage", "messagebird").Error("Provider must be one of: twilio, sns, vonage, messagebird"),
		),
		validation.Field(&r.AccountSID,
			validation.Length(0, 255).Error("Account SID must not exceed 255 characters"),
		),
		validation.Field(&r.FromNumber,
			validation.Length(0, 50).Error("From number must not exceed 50 characters"),
		),
		validation.Field(&r.SenderID,
			validation.Length(0, 50).Error("Sender ID must not exceed 50 characters"),
		),
	)
}
