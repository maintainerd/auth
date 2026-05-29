package webhook

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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
			validation.When(r.Status != nil, validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
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
			validation.When(r.Status != nil, validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
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
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
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

// WebhookEndpoint represents an outbound event notification subscription
// belonging to a tenant. Multiple endpoints may exist per tenant, each
// subscribing to a different set of events.
type WebhookEndpoint struct {
	WebhookEndpointID   int64          `gorm:"column:webhook_endpoint_id;primaryKey;autoIncrement" json:"webhook_endpoint_id"`
	WebhookEndpointUUID uuid.UUID      `gorm:"column:webhook_endpoint_uuid;type:uuid;uniqueIndex;not null" json:"webhook_endpoint_uuid"`
	TenantID            int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	URL                 string         `gorm:"column:url;type:text;not null" json:"url"`
	SecretEncrypted     string         `gorm:"column:secret_encrypted;type:text" json:"-"`
	Events              datatypes.JSON `gorm:"column:events;type:jsonb;default:'[]'" json:"events"`
	MaxRetries          int            `gorm:"column:max_retries;not null;default:3" json:"max_retries"`
	TimeoutSeconds      int            `gorm:"column:timeout_seconds;not null;default:30" json:"timeout_seconds"`
	Status              string         `gorm:"column:status;type:varchar(20);not null;default:'active'" json:"status"`
	Description         string         `gorm:"column:description;type:text" json:"description"`
	Metadata            datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'" json:"metadata"`
	LastTriggeredAt     *time.Time     `gorm:"column:last_triggered_at" json:"last_triggered_at"`
	CreatedBy           *int64         `gorm:"column:created_by" json:"created_by,omitempty"`
	UpdatedBy           *int64         `gorm:"column:updated_by" json:"updated_by,omitempty"`
	CreatedAt           time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`

	// Relationships
}

// TableName returns the database table name for WebhookEndpoint.
func (WebhookEndpoint) TableName() string {
	return "webhook_endpoints"
}

// BeforeCreate sets a new UUID on the WebhookEndpoint before it is inserted
// into the database if one has not already been assigned.
func (we *WebhookEndpoint) BeforeCreate(tx *gorm.DB) error {
	if we.WebhookEndpointUUID == uuid.Nil {
		we.WebhookEndpointUUID = uuid.New()
	}
	return nil
}
