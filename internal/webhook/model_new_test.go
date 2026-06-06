package webhook

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWebhookEndpointEvent_TableName(t *testing.T) {
	m := WebhookEndpointEvent{}
	assert.Equal(t, "webhook_endpoint_events", m.TableName())
}

func TestWebhookEndpointEvent_BeforeCreate(t *testing.T) {
	m := &WebhookEndpointEvent{
		WebhookEndpointID: 1,
		EventTypeID:       10,
	}
	err := m.BeforeCreate(nil)
	assert.NoError(t, err)
}

func TestDeliveryHistory_TableName(t *testing.T) {
	m := DeliveryHistory{}
	assert.Equal(t, "webhook_delivery_history", m.TableName())
}

func TestDeliveryHistory_BeforeCreate(t *testing.T) {
	m := &DeliveryHistory{
		WebhookEndpointID: 7,
		EventID:           uuid.New(),
		EventType:         "user.created",
		TenantID:          1,
		FinalStatus:       "pending",
	}
	err := m.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, m.DeliveryHistoryUUID)
}

func TestDeliveryHistory_UUIDGeneratedOnce(t *testing.T) {
	m := &DeliveryHistory{
		DeliveryHistoryUUID: uuid.New(),
	}
	existing := m.DeliveryHistoryUUID
	err := m.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.Equal(t, existing, m.DeliveryHistoryUUID, "should not overwrite existing UUID")
}

func TestDeliveryHistory_IsReplayDefaults(t *testing.T) {
	m := &DeliveryHistory{}
	assert.False(t, m.IsReplay)
}
