package webhook

import (
	"context"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

var webhookURLRule = validation.By(func(value any) error {
	raw, _ := value.(string)
	if raw == "" {
		return nil
	}
	if err := validateWebhookURL(context.TODO(), raw, false); err != nil {
		return validation.NewError("validation_webhook_url", err.Error())
	}
	return nil
})

// Validate validates the webhook endpoint create request.
func (r WebhookEndpointCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.URL,
			validation.Required.Error("URL is required"),
			is.URL.Error("URL must be a valid URL"),
			webhookURLRule,
		),
		validation.Field(&r.MaxRetries,
			validation.When(r.MaxRetries != nil, validation.Min(0).Error("Max retries must be at least 0"), validation.Max(10).Error("Max retries must not exceed 10")),
		),
		validation.Field(&r.TimeoutSeconds,
			validation.When(r.TimeoutSeconds != nil, validation.Min(1).Error("Timeout must be at least 1 second"), validation.Max(120).Error("Timeout must not exceed 120 seconds")),
		),
		validation.Field(&r.Description,
			validation.Length(0, 500).Error("Description must not exceed 500 characters"),
		),
		validation.Field(&r.Status,
			validation.When(r.Status != nil, validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
		),
	)
}

// Validate validates the webhook endpoint update request.
func (r WebhookEndpointUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.URL,
			validation.Required.Error("URL is required"),
			is.URL.Error("URL must be a valid URL"),
			webhookURLRule,
		),
		validation.Field(&r.MaxRetries,
			validation.When(r.MaxRetries != nil, validation.Min(0).Error("Max retries must be at least 0"), validation.Max(10).Error("Max retries must not exceed 10")),
		),
		validation.Field(&r.TimeoutSeconds,
			validation.When(r.TimeoutSeconds != nil, validation.Min(1).Error("Timeout must be at least 1 second"), validation.Max(120).Error("Timeout must not exceed 120 seconds")),
		),
		validation.Field(&r.Description,
			validation.Length(0, 500).Error("Description must not exceed 500 characters"),
		),
		validation.Field(&r.Status,
			validation.When(r.Status != nil, validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
		),
	)
}

// Validate validates the webhook endpoint status update request.
func (r WebhookEndpointUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Validate validates the webhook endpoint filter.
func (f WebhookEndpointFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.PaginationRequestDTO),
	)
}
