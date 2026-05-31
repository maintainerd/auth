package webhook

import (
	"time"
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

// WebhookEndpointUpdateStatusRequestDTO is the request body for updating
// webhook endpoint status.
type WebhookEndpointUpdateStatusRequestDTO struct {
	Status string `json:"status"`
}

// WebhookEndpointFilterDTO holds filter parameters for listing webhook
// endpoints.
type WebhookEndpointFilterDTO struct {
	Status []string `json:"status"`
	PaginationRequestDTO
}
