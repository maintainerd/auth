package event

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testPublicIDResolver struct {
	tenantID int64
	tenant   uuid.UUID
	userID   int64
	user     uuid.UUID
}

func (r testPublicIDResolver) TenantUUIDByID(_ context.Context, id int64) (uuid.UUID, bool) {
	if id == r.tenantID {
		return r.tenant, true
	}
	return uuid.Nil, false
}

func (r testPublicIDResolver) UserUUIDByID(_ context.Context, id int64) (uuid.UUID, bool) {
	if id == r.userID {
		return r.user, true
	}
	return uuid.Nil, false
}

func TestOutboxPayloadUsesPublicUUIDs(t *testing.T) {
	tenantUUID := uuid.New()
	actorUUID := uuid.New()
	subjectUUID := uuid.New()
	actorID := int64(100)
	outbox := NewIntegrationEvent(EventTypeUserUpdated, 1, 42).
		SetActor(&actorID).
		SetSubject(&subjectUUID, "user").
		SetChangedFields("email").
		ToOutbox()

	payload, err := OutboxPayload(context.Background(), outbox, testPublicIDResolver{
		tenantID: 42,
		tenant:   tenantUUID,
		userID:   actorID,
		user:     actorUUID,
	})
	require.NoError(t, err)

	assert.Equal(t, tenantUUID.String(), payload["tenant_id"])
	assert.Equal(t, actorUUID.String(), payload["actor_user_id"])
	assert.NotEqual(t, outbox.TenantID, payload["tenant_id"])
	assert.NotEqual(t, *outbox.ActorUserID, payload["actor_user_id"])

	body, err := json.Marshal(payload)
	require.NoError(t, err)
	assert.NotContains(t, string(body), `"tenant_id":42`)
	assert.NotContains(t, string(body), `"actor_user_id":100`)
}

func TestOutboxPayloadRejectsUnresolvedTenant(t *testing.T) {
	outbox := NewIntegrationEvent(EventTypeUserUpdated, 1, 42).ToOutbox()

	payload, err := OutboxPayload(context.Background(), outbox, testPublicIDResolver{})
	require.Error(t, err)
	assert.Nil(t, payload)
}

func TestOutboxPayloadUsesTenantSubjectAsFallback(t *testing.T) {
	tenantUUID := uuid.New()
	outbox := NewIntegrationEvent(EventTypeTenantDeleted, 1, 42).
		SetSubject(&tenantUUID, "tenant").
		ToOutbox()

	payload, err := OutboxPayload(context.Background(), outbox, testPublicIDResolver{})
	require.NoError(t, err)
	assert.Equal(t, tenantUUID.String(), payload["tenant_id"])
}
