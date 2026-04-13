package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/model"
)

// AuthEventFilterDTO holds query parameters for listing auth events.
type AuthEventFilterDTO struct {
	Category  *string `json:"category"`
	EventType *string `json:"event_type"`
	Severity  *string `json:"severity"`
	Result    *string `json:"result"`
	DateFrom  *string `json:"date_from"`
	DateTo    *string `json:"date_to"`
	PaginationRequestDTO
}

// Validate validates the filter parameters.
func (f AuthEventFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Category,
			validation.NilOrNotEmpty,
			validation.In(
				model.AuthEventCategoryAuthn,
				model.AuthEventCategoryAuthz,
				model.AuthEventCategorySession,
				model.AuthEventCategoryUser,
				model.AuthEventCategorySystem,
			).Error("Category must be one of: AUTHN, AUTHZ, SESSION, USER, SYSTEM"),
		),
		validation.Field(&f.Severity,
			validation.NilOrNotEmpty,
			validation.In(
				model.AuthEventSeverityInfo,
				model.AuthEventSeverityWarn,
				model.AuthEventSeverityCritical,
			).Error("Severity must be one of: INFO, WARN, CRITICAL"),
		),
		validation.Field(&f.Result,
			validation.NilOrNotEmpty,
			validation.In(
				model.AuthEventResultSuccess,
				model.AuthEventResultFailure,
			).Error("Result must be one of: success, failure"),
		),
		validation.Field(&f.EventType,
			validation.NilOrNotEmpty,
			validation.Length(1, 60).Error("EventType cannot exceed 60 characters"),
		),
		validation.Field(&f.Page,
			validation.Required.Error("Page is required"),
			validation.Min(1).Error("Page must be greater than 0"),
		),
		validation.Field(&f.Limit,
			validation.Required.Error("Limit is required"),
			validation.Min(1).Error("Limit must be greater than 0"),
			validation.Max(100).Error("Limit cannot exceed 100"),
		),
		validation.Field(&f.SortBy,
			validation.Length(0, 50).Error("SortBy cannot exceed 50 characters"),
		),
		validation.Field(&f.SortOrder,
			validation.In(SortOrderAsc, SortOrderDesc).Error("Order must be either 'asc' or 'desc'"),
		),
	)
}

// AuthEventResponseDTO is the API response for a single auth event.
type AuthEventResponseDTO struct {
	AuthEventID  string          `json:"auth_event_id"`
	TenantID     int64           `json:"tenant_id"`
	ActorUserID  *int64          `json:"actor_user_id,omitempty"`
	TargetUserID *int64          `json:"target_user_id,omitempty"`
	IPAddress    string          `json:"ip_address"`
	UserAgent    *string         `json:"user_agent,omitempty"`
	Category     string          `json:"category"`
	EventType    string          `json:"event_type"`
	Severity     string          `json:"severity"`
	Result       string          `json:"result"`
	Description  *string         `json:"description,omitempty"`
	ErrorReason  *string         `json:"error_reason,omitempty"`
	TraceID      *string         `json:"trace_id,omitempty"`
	Metadata     *map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}
