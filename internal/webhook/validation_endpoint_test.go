package webhook

import (
	"context"
	"strings"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookURLRule(t *testing.T) {
	t.Run("empty string is skipped", func(t *testing.T) {
		require.NoError(t, validation.Validate("", webhookURLRule))
	})

	t.Run("invalid url returns validation error", func(t *testing.T) {
		err := validation.Validate("http://example.com/hook", webhookURLRule)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "https")
	})

	t.Run("valid url", func(t *testing.T) {
		require.NoError(t, validateWebhookURL(context.Background(), "https://example.com/hook", false))
	})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func validWebhookCreate() WebhookEndpointCreateRequestDTO {
	return WebhookEndpointCreateRequestDTO{
		URL:          "https://example.com/hook",
		SubscribeAll: true,
		Description:  "Test hook",
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
		status := shared.StatusActive
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

	t.Run("http url rejected", func(t *testing.T) {
		d := validWebhookCreate()
		d.URL = "http://example.com/hook"
		require.Error(t, d.Validate())
	})

	t.Run("private ip url rejected", func(t *testing.T) {
		d := validWebhookCreate()
		d.URL = "https://127.0.0.1/hook"
		require.Error(t, d.Validate())
	})

	t.Run("description too long", func(t *testing.T) {
		d := validWebhookCreate()
		d.Description = strings.Repeat("a", 501)
		require.Error(t, d.Validate())
	})

	t.Run("invalid max retries", func(t *testing.T) {
		d := validWebhookCreate()
		neg := -1
		d.MaxRetries = &neg
		require.Error(t, d.Validate())
	})

	t.Run("invalid timeout", func(t *testing.T) {
		d := validWebhookCreate()
		tooLong := 200
		d.TimeoutSeconds = &tooLong
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validWebhookCreate()
		bad := "deleted"
		d.Status = &bad
		require.Error(t, d.Validate())
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func validWebhookUpdate() WebhookEndpointUpdateRequestDTO {
	return WebhookEndpointUpdateRequestDTO{
		URL:          "https://example.com/hook",
		SubscribeAll: true,
		Description:  "Test hook",
	}
}

func TestWebhookEndpointUpdateRequestDTO_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, validWebhookUpdate().Validate())
	})

	t.Run("valid with optional fields", func(t *testing.T) {
		d := validWebhookUpdate()
		retries := 5
		timeout := 60
		status := shared.StatusActive
		d.MaxRetries = &retries
		d.TimeoutSeconds = &timeout
		d.Status = &status
		assert.NoError(t, d.Validate())
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

	t.Run("http url rejected", func(t *testing.T) {
		d := validWebhookUpdate()
		d.URL = "http://example.com/hook"
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validWebhookUpdate()
		bad := "deleted"
		d.Status = &bad
		require.Error(t, d.Validate())
	})
}

// ---------------------------------------------------------------------------
// Update status
// ---------------------------------------------------------------------------

func TestWebhookEndpointUpdateStatusRequestDTO_Validate(t *testing.T) {
	t.Run("valid active", func(t *testing.T) {
		assert.NoError(t, WebhookEndpointUpdateStatusRequestDTO{Status: shared.StatusActive}.Validate())
	})

	t.Run("valid inactive", func(t *testing.T) {
		assert.NoError(t, WebhookEndpointUpdateStatusRequestDTO{Status: shared.StatusInactive}.Validate())
	})

	t.Run("missing status", func(t *testing.T) {
		require.Error(t, WebhookEndpointUpdateStatusRequestDTO{}.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		require.Error(t, WebhookEndpointUpdateStatusRequestDTO{Status: "deleted"}.Validate())
	})
}

// ---------------------------------------------------------------------------
// Filter
// ---------------------------------------------------------------------------

func TestWebhookEndpointFilterDTO_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		filter := WebhookEndpointFilterDTO{
			Status:               []string{shared.StatusActive},
			PaginationRequestDTO: PaginationRequestDTO{Page: 1, Limit: 10},
		}
		assert.NoError(t, filter.Validate())
	})
}
