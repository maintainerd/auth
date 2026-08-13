package authevent

import "time"

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

// AuthEventResponseDTO is the API response for a single auth event.
type AuthEventResponseDTO struct {
	AuthEventID string `json:"auth_event_id"`
	// Internal integer FKs (tenant_id, actor_user_id, target_user_id) are
	// intentionally NOT exposed — the response is already scoped to the authed
	// tenant, and only UUIDs may leave the service.
	IPAddress   string          `json:"ip_address"`
	UserAgent   *string         `json:"user_agent,omitempty"`
	Category    string          `json:"category"`
	EventType   string          `json:"event_type"`
	Severity    string          `json:"severity"`
	Result      string          `json:"result"`
	Description *string         `json:"description,omitempty"`
	ErrorReason *string         `json:"error_reason,omitempty"`
	TraceID     *string         `json:"trace_id,omitempty"`
	Metadata    *map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}
