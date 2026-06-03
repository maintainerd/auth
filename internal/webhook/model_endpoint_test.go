package webhook

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookEndpoint_TableName(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		assert.Equal(t, "webhook_endpoints", (WebhookEndpoint{}).TableName())
	})
}

func TestWebhookEndpoint_BeforeCreate(t *testing.T) {
	t.Run("sets uuid when missing", func(t *testing.T) {
		endpoint := &WebhookEndpoint{}

		require.NoError(t, endpoint.BeforeCreate(nil))

		assert.NotEqual(t, uuid.Nil, endpoint.WebhookEndpointUUID)
	})

	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		endpoint := &WebhookEndpoint{WebhookEndpointUUID: existing}

		require.NoError(t, endpoint.BeforeCreate(nil))

		assert.Equal(t, existing, endpoint.WebhookEndpointUUID)
	})
}
