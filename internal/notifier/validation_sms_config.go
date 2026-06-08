package notifier

import validation "github.com/go-ozzo/ozzo-validation/v4"

// Validate validates the SMS config update request.
func (r SMSConfigUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Provider,
			validation.Required.Error("Provider is required"),
			validation.In("twilio", "sns", "vonage", "messagebird", "log").Error("Provider must be one of: twilio, sns, vonage, messagebird, log"),
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
