package webhook

import (
	"time"

	"github.com/google/uuid"
)

type DeliveryHistoryResponseDTO struct {
	DeliveryHistoryUUID uuid.UUID `json:"delivery_history_id"`
	// EventID is what POST /webhook-replay actually replays. The DTO omitted it,
	// so the console had nothing to send and posted the delivery-history UUID
	// under the wrong key instead — every replay 400'd on "Invalid event_id UUID".
	EventID         uuid.UUID `json:"event_id"`
	EventType       string    `json:"event_type"`
	AttemptCount    int       `json:"attempt_count"`
	FinalStatus     string    `json:"final_status"`
	ResponseStatus  *int      `json:"response_status,omitempty"`
	ResponseSummary string    `json:"response_summary"`
	IsReplay        bool      `json:"is_replay"`
	CreatedAt       time.Time `json:"created_at"`
}

// WebhookEndpointResponseDTO is the JSON representation of a webhook endpoint.
type WebhookEndpointResponseDTO struct {
	WebhookEndpointID string    `json:"webhook_endpoint_id"`
	URL               string    `json:"url"`
	SigningSecret     string    `json:"signing_secret,omitempty"` // only returned on create
	SubscribeAll      bool      `json:"subscribe_all"`
	MaxRetries        int       `json:"max_retries"`
	TimeoutSeconds    int       `json:"timeout_seconds"`
	Status            string    `json:"status"`
	Description       string    `json:"description"`
	LastTriggeredAt   *string   `json:"last_triggered_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// WebhookEndpointCreateRequestDTO is the request body for creating a webhook endpoint.
// Secret is now generated server-side; caller does not supply it.
type WebhookEndpointCreateRequestDTO struct {
	URL            string  `json:"url"`
	SubscribeAll   bool    `json:"subscribe_all"`
	MaxRetries     *int    `json:"max_retries"`
	TimeoutSeconds *int    `json:"timeout_seconds"`
	Description    string  `json:"description"`
	Status         *string `json:"status,omitempty"`
}

// WebhookEndpointUpdateRequestDTO is the request body for updating a webhook endpoint.
type WebhookEndpointUpdateRequestDTO struct {
	URL            string  `json:"url"`
	RotateSecret   bool    `json:"rotate_secret"`
	SubscribeAll   bool    `json:"subscribe_all"`
	MaxRetries     *int    `json:"max_retries"`
	TimeoutSeconds *int    `json:"timeout_seconds"`
	Description    string  `json:"description"`
	Status         *string `json:"status,omitempty"`
}

// WebhookEndpointUpdateStatusRequestDTO is the request body for updating
// webhook endpoint status.
type WebhookEndpointUpdateStatusRequestDTO struct {
	Status string `json:"status"`
}

// WebhookEndpointFilterDTO holds filter parameters for listing webhook
// endpoints.
type WebhookEndpointFilterDTO struct {
	URL    *string
	Status []string `json:"status"`
	PaginationRequestDTO
}
