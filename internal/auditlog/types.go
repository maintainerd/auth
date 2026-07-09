package auditlog

// ManagementAuditLogFilter carries filtering and pagination parameters
// for management audit log queries.
type ManagementAuditLogFilter struct {
	ResourceType string
	Action       string
	ActorUserID  *int64
	Page         int
	Limit        int
}

// ManagementAuditLogFilterDTO is the request shape for filter parameters
// (kept for potential future use; actual parsing reads from query params).
type ManagementAuditLogFilterDTO struct {
	ResourceType string `json:"resource_type"`
	Action       string `json:"action"`
	ActorUserID  *int64 `json:"actor_user_id"`
	Page         int    `json:"page"`
	Limit        int    `json:"limit"`
}

// ManagementAuditLogResponseDTO is the API response shape for a single audit log entry.
type ManagementAuditLogResponseDTO struct {
	UUID            string  `json:"uuid"`
	Action          string  `json:"action"`
	ResourceType    string  `json:"resource_type"`
	ResourceID      string  `json:"resource_id"`
	Outcome         string  `json:"outcome"`
	IPAddress       string  `json:"ip_address,omitempty"`
	CreatedAt       string  `json:"created_at"`
	ActorUserID     *int64  `json:"actor_user_id,omitempty"`
	ActorUserName   *string `json:"actor_user_name,omitempty"`
	ActorClientID   *int64  `json:"actor_client_id,omitempty"`
	ActorClientName *string `json:"actor_client_name,omitempty"`
	Changes         string  `json:"changes,omitempty"`
	ErrorMessage    *string `json:"error_message,omitempty"`
	TraceID         *string `json:"trace_id,omitempty"`
}
