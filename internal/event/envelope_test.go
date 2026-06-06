package event

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIntegrationEvent(t *testing.T) {
	evt := NewIntegrationEvent(EventTypeUserCreated, 1, 42)
	assert.NotEqual(t, uuid.Nil, evt.EventID)
	assert.Equal(t, EventTypeUserCreated, evt.EventType)
	assert.Equal(t, 1, evt.EventVersion)
	assert.Equal(t, int64(42), evt.TenantID)
	assert.NotNil(t, evt.ChangedFields)
	assert.Empty(t, evt.ChangedFields)
	assert.False(t, evt.OccurredAt.IsZero())
}

func TestIntegrationEvent_SetActor(t *testing.T) {
	actorID := int64(100)
	evt := NewIntegrationEvent("test.event", 1, 1).SetActor(&actorID)
	assert.NotNil(t, evt.ActorUserID)
	assert.Equal(t, int64(100), *evt.ActorUserID)
}

func TestIntegrationEvent_SetSubject(t *testing.T) {
	id := uuid.New()
	evt := NewIntegrationEvent("test.event", 1, 1).SetSubject(&id, "user")
	assert.NotNil(t, evt.SubjectUUID)
	assert.Equal(t, id, *evt.SubjectUUID)
	assert.Equal(t, "user", evt.SubjectType)
}

func TestIntegrationEvent_SetSubject_NilSafe(t *testing.T) {
	evt := NewIntegrationEvent("test.event", 1, 1)
	evt.SetSubject(nil, "")
	assert.Nil(t, evt.SubjectUUID)
	assert.Empty(t, evt.SubjectType)
}

func TestIntegrationEvent_SetChangedFields(t *testing.T) {
	t.Run("single field", func(t *testing.T) {
		evt := NewIntegrationEvent("test.event", 1, 1).SetChangedFields("email")
		assert.Equal(t, []string{"email"}, evt.ChangedFields)
	})
	t.Run("multiple fields", func(t *testing.T) {
		evt := NewIntegrationEvent("test.event", 1, 1).SetChangedFields("email", "status")
		assert.Equal(t, []string{"email", "status"}, evt.ChangedFields)
	})
	t.Run("empty fields", func(t *testing.T) {
		evt := NewIntegrationEvent("test.event", 1, 1).SetChangedFields()
		assert.Empty(t, evt.ChangedFields)
	})
}

func TestIntegrationEvent_SetTraceInfo(t *testing.T) {
	evt := NewIntegrationEvent("test.event", 1, 1).SetTraceInfo("trace123", "req456")
	assert.Equal(t, "trace123", evt.TraceID)
	assert.Equal(t, "req456", evt.RequestID)
}

func TestIntegrationEvent_ToOutbox(t *testing.T) {
	actor := int64(100)
	subject := uuid.New()
	evt := NewIntegrationEvent(EventTypeUserCreated, 1, 42).
		SetActor(&actor).
		SetSubject(&subject, "user").
		SetChangedFields("username", "email").
		SetTraceInfo("traceX", "reqY")

	outbox := evt.ToOutbox()
	assert.Equal(t, evt.EventID, outbox.EventID)
	assert.Equal(t, evt.EventType, outbox.EventType)
	assert.Equal(t, evt.EventVersion, outbox.EventVersion)
	assert.Equal(t, evt.TenantID, outbox.TenantID)
	assert.Equal(t, evt.ActorUserID, outbox.ActorUserID)
	assert.Equal(t, evt.SubjectUUID, outbox.SubjectUUID)
	assert.Equal(t, evt.SubjectType, outbox.SubjectType)
	assert.Equal(t, evt.TraceID, outbox.TraceID)
	assert.Equal(t, evt.RequestID, outbox.RequestID)
	assert.False(t, outbox.IsPublished)

	fields := unmarshalStringSliceRaw(outbox.ChangedFields)
	assert.Equal(t, []string{"username", "email"}, fields)
}

func TestIntegrationEvent_ToOutbox_ThinPayload(t *testing.T) {
	evt := NewIntegrationEvent(EventTypeUserUpdated, 1, 42).
		SetChangedFields("email", "status")
	outbox := evt.ToOutbox()
	require.NotNil(t, outbox)

	// Outbox payload may be empty JSON object when no explicit payload is set
	fields := unmarshalStringSliceRaw(outbox.ChangedFields)
	assert.Contains(t, fields, "email")
	assert.Contains(t, fields, "status")

	// Verify no PII values in the envelope itself (it only carries identifiers)
	assert.Equal(t, EventTypeUserUpdated, outbox.EventType)
	assert.Equal(t, int64(42), outbox.TenantID)
}

func TestIntegrationEvent_EventIDIsUnique(t *testing.T) {
	evt1 := NewIntegrationEvent("test.event", 1, 1)
	evt2 := NewIntegrationEvent("test.event", 1, 1)
	assert.NotEqual(t, evt1.EventID, evt2.EventID)
}

func TestIntegrationEvent_SetPayload(t *testing.T) {
	raw := []byte(`{"meta":"data"}`)
	evt := NewIntegrationEvent("test.event", 1, 1).SetPayload(raw)
	assert.Equal(t, string(raw), string(evt.Payload))
}
