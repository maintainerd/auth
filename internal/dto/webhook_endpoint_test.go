package dto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/model"
)

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func validWebhookCreate() WebhookEndpointCreateRequestDTO {
	return WebhookEndpointCreateRequestDTO{
		URL:         "https://example.com/hook",
		Secret:      "s3cret",
		Events:      []string{"user.created", "user.deleted"},
		Description: "Test hook",
	}
}

func TestWebhookEndpointCreateRequestDTO_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, validWebhookCreate().Validate())
	})

	t.Run("valid with optional fields", func(t *testing.T) {
		d := validWebhookCreate()
		retries := 5
		timeout := 60
		status := model.StatusActive
		d.MaxRetries = &retries
		d.TimeoutSeconds = &timeout
		d.Status = &status
		assert.NoError(t, d.Validate())
	})

	t.Run("missing url", func(t *testing.T) {
		d := validWebhookCreate()
		d.URL = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid url", func(t *testing.T) {
		d := validWebhookCreate()
		d.URL = "not-a-url"
		require.Error(t, d.Validate())
	})

	t.Run("missing events", func(t *testing.T) {
		d := validWebhookCreate()
		d.Events = nil
		require.Error(t, d.Validate())
	})

	t.Run("empty events", func(t *testing.T) {
		d := validWebhookCreate()
		d.Events = []string{}
		require.Error(t, d.Validate())
	})

	t.Run("event name too long", func(t *testing.T) {
		d := validWebhookCreate()
		d.Events = []string{strings.Repeat("a", 101)}
		require.Error(t, d.Validate())
	})

	t.Run("max_retries too high", func(t *testing.T) {
		d := validWebhookCreate()
		v := 11
		d.MaxRetries = &v
		require.Error(t, d.Validate())
	})

	t.Run("timeout too high", func(t *testing.T) {
		d := validWebhookCreate()
		v := 121
		d.TimeoutSeconds = &v
		require.Error(t, d.Validate())
	})

	t.Run("description too long", func(t *testing.T) {
		d := validWebhookCreate()
		d.Description = strings.Repeat("a", 501)
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validWebhookCreate()
		bad := "pending"
		d.Status = &bad
		require.Error(t, d.Validate())
	})

	t.Run("valid status active", func(t *testing.T) {
		d := validWebhookCreate()
		active := model.StatusActive
		d.Status = &active
		assert.NoError(t, d.Validate())
	})

	t.Run("valid status inactive", func(t *testing.T) {
		d := validWebhookCreate()
		inactive := model.StatusInactive
		d.Status = &inactive
		assert.NoError(t, d.Validate())
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func validWebhookUpdate() WebhookEndpointUpdateRequestDTO {
	return WebhookEndpointUpdateRequestDTO{
		URL:         "https://example.com/hook",
		Events:      []string{"user.created"},
		Description: "Updated hook",
	}
}

func TestWebhookEndpointUpdateRequestDTO_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, validWebhookUpdate().Validate())
	})

	t.Run("missing url", func(t *testing.T) {
		d := validWebhookUpdate()
		d.URL = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid url", func(t *testing.T) {
		d := validWebhookUpdate()
		d.URL = "not-a-url"
		require.Error(t, d.Validate())
	})

	t.Run("missing events", func(t *testing.T) {
		d := validWebhookUpdate()
		d.Events = nil
		require.Error(t, d.Validate())
	})

	t.Run("max_retries too high", func(t *testing.T) {
		d := validWebhookUpdate()
		v := 11
		d.MaxRetries = &v
		require.Error(t, d.Validate())
	})

	t.Run("timeout too high", func(t *testing.T) {
		d := validWebhookUpdate()
		v := 121
		d.TimeoutSeconds = &v
		require.Error(t, d.Validate())
	})

	t.Run("description too long", func(t *testing.T) {
		d := validWebhookUpdate()
		d.Description = strings.Repeat("a", 501)
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validWebhookUpdate()
		bad := "pending"
		d.Status = &bad
		require.Error(t, d.Validate())
	})
}

// ---------------------------------------------------------------------------
// UpdateStatus
// ---------------------------------------------------------------------------

func TestWebhookEndpointUpdateStatusRequestDTO_Validate(t *testing.T) {
	t.Run("valid active", func(t *testing.T) {
		d := WebhookEndpointUpdateStatusRequestDTO{Status: model.StatusActive}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid inactive", func(t *testing.T) {
		d := WebhookEndpointUpdateStatusRequestDTO{Status: model.StatusInactive}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing status", func(t *testing.T) {
		d := WebhookEndpointUpdateStatusRequestDTO{}
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := WebhookEndpointUpdateStatusRequestDTO{Status: "paused"}
		require.Error(t, d.Validate())
	})
}

// ---------------------------------------------------------------------------
// Filter
// ---------------------------------------------------------------------------

func TestWebhookEndpointFilterDTO_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		d := WebhookEndpointFilterDTO{
			PaginationRequestDTO: PaginationRequestDTO{Page: 1, Limit: 10},
		}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid with status", func(t *testing.T) {
		d := WebhookEndpointFilterDTO{
			Status:               []string{model.StatusActive},
			PaginationRequestDTO: PaginationRequestDTO{Page: 1, Limit: 10},
		}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing pagination", func(t *testing.T) {
		d := WebhookEndpointFilterDTO{}
		require.Error(t, d.Validate())
	})
}
