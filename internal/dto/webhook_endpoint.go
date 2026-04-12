package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/maintainerd/auth/internal/model"
)

// WebhookEndpointResponseDTO is the JSON representation of a webhook endpoint.
type WebhookEndpointResponseDTO struct {
	WebhookEndpointID string    `json:"webhook_endpoint_id"`
	URL               string    `json:"url"`
	Events            any       `json:"events"`
	MaxRetries        int       `json:"max_retries"`
	TimeoutSeconds    int       `json:"timeout_seconds"`
	Status            string    `json:"status"`
	Description       string    `json:"description"`
	LastTriggeredAt   *string   `json:"last_triggered_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// WebhookEndpointCreateRequestDTO is the request body for creating a webhook
// endpoint.
type WebhookEndpointCreateRequestDTO struct {
	URL            string   `json:"url"`
	Secret         string   `json:"secret"`
	Events         []string `json:"events"`
	MaxRetries     *int     `json:"max_retries"`
	TimeoutSeconds *int     `json:"timeout_seconds"`
	Description    string   `json:"description"`
	Status         *string  `json:"status,omitempty"`
}

// Validate validates the webhook endpoint create request.
func (r WebhookEndpointCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.URL,
			validation.Required.Error("URL is required"),
			is.URL.Error("URL must be a valid URL"),
		),
		validation.Field(&r.Events,
			validation.Required.Error("Events list is required"),
			validation.Each(validation.Length(1, 100).Error("Event name must be between 1 and 100 characters")),
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
			validation.When(r.Status != nil, validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'")),
		),
	)
}

// WebhookEndpointUpdateRequestDTO is the request body for updating a webhook
// endpoint.
type WebhookEndpointUpdateRequestDTO struct {
	URL            string   `json:"url"`
	Secret         string   `json:"secret"`
	Events         []string `json:"events"`
	MaxRetries     *int     `json:"max_retries"`
	TimeoutSeconds *int     `json:"timeout_seconds"`
	Description    string   `json:"description"`
	Status         *string  `json:"status,omitempty"`
}

// Validate validates the webhook endpoint update request.
func (r WebhookEndpointUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.URL,
			validation.Required.Error("URL is required"),
			is.URL.Error("URL must be a valid URL"),
		),
		validation.Field(&r.Events,
			validation.Required.Error("Events list is required"),
			validation.Each(validation.Length(1, 100).Error("Event name must be between 1 and 100 characters")),
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
			validation.When(r.Status != nil, validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'")),
		),
	)
}

// WebhookEndpointUpdateStatusRequestDTO is the request body for updating
// webhook endpoint status.
type WebhookEndpointUpdateStatusRequestDTO struct {
	Status string `json:"status"`
}

// Validate validates the webhook endpoint status update request.
func (r WebhookEndpointUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// WebhookEndpointFilterDTO holds filter parameters for listing webhook
// endpoints.
type WebhookEndpointFilterDTO struct {
	Status []string `json:"status"`
	PaginationRequestDTO
}

// Validate validates the webhook endpoint filter.
func (f WebhookEndpointFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.PaginationRequestDTO),
	)
}
