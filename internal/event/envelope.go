package event

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// IntegrationEvent is the canonical envelope for all integration events.
// It is thin by design — identifiers and changed field names only, never values.
type IntegrationEvent struct {
	EventID       uuid.UUID      `json:"event_id"`
	EventType     string         `json:"event_type"`
	EventVersion  int            `json:"event_version"`
	TenantID      int64          `json:"tenant_id"`
	ActorUserID   *int64         `json:"actor_user_id,omitempty"`
	SubjectUUID   *uuid.UUID     `json:"subject_id,omitempty"`
	SubjectType   string         `json:"subject_type,omitempty"`
	ChangedFields []string       `json:"changed_fields"`
	Payload       datatypes.JSON `json:"payload,omitempty"`
	OccurredAt    time.Time      `json:"occurred_at"`
	TraceID       string         `json:"trace_id,omitempty"`
	RequestID     string         `json:"request_id,omitempty"`
}

// NewIntegrationEvent creates a new IntegrationEvent with defaults populated.
func NewIntegrationEvent(eventType string, version int, tenantID int64) *IntegrationEvent {
	return &IntegrationEvent{
		EventID:       uuid.New(),
		EventType:     eventType,
		EventVersion:  version,
		TenantID:      tenantID,
		ChangedFields: []string{},
		OccurredAt:    time.Now().UTC(),
	}
}

// SetActor sets the actor (who triggered the mutation).
func (e *IntegrationEvent) SetActor(userID *int64) *IntegrationEvent {
	e.ActorUserID = userID
	return e
}

// SetSubject sets the subject of the event (the resource that changed).
func (e *IntegrationEvent) SetSubject(uuid *uuid.UUID, subjectType string) *IntegrationEvent {
	e.SubjectUUID = uuid
	e.SubjectType = subjectType
	return e
}

// SetChangedFields sets the list of field names that changed (no values).
func (e *IntegrationEvent) SetChangedFields(fields ...string) *IntegrationEvent {
	e.ChangedFields = fields
	return e
}

// SetTraceInfo sets trace and request context.
func (e *IntegrationEvent) SetTraceInfo(traceID, requestID string) *IntegrationEvent {
	e.TraceID = traceID
	e.RequestID = requestID
	return e
}

// SetPayload sets additional JSON payload (must not contain PII values).
func (e *IntegrationEvent) SetPayload(payload datatypes.JSON) *IntegrationEvent {
	e.Payload = payload
	return e
}

// ToOutbox converts an IntegrationEvent to an Outbox row for persistence.
func (e *IntegrationEvent) ToOutbox() *Outbox {
	changedFieldsJSON, _ := marshalStringSlice(e.ChangedFields)
	return &Outbox{
		OutboxUUID:    uuid.New(),
		EventID:       e.EventID,
		EventType:     e.EventType,
		EventVersion:  e.EventVersion,
		TenantID:      e.TenantID,
		ActorUserID:   e.ActorUserID,
		SubjectUUID:   e.SubjectUUID,
		SubjectType:   e.SubjectType,
		ChangedFields: changedFieldsJSON,
		Payload:       e.Payload,
		OccurredAt:    e.OccurredAt,
		TraceID:       e.TraceID,
		RequestID:     e.RequestID,
		IsPublished:   false,
	}
}
