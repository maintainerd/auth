package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The console's status multi-select sends its choices comma-joined. Appending
// the raw string produced `status IN ('active,inactive')`, which matches
// nothing — so picking two statuses looked like "no webhooks exist". And the
// listing's "Search webhooks by URL..." box sent ?url= that nothing read, so
// typing in it did nothing at all.
func TestWebhookEndpointHandler_GetAll_FilterParsing(t *testing.T) {
	call := func(t *testing.T, query string) ([]string, *string) {
		t.Helper()
		var gotStatus []string
		var gotURL *string
		svc := &mockWebhookEndpointService{
			getAllFn: func(_ int64, status []string, url *string, _, _ int, _, _ string) (*WebhookEndpointServiceListResult, error) {
				gotStatus, gotURL = status, url
				return &WebhookEndpointServiceListResult{}, nil
			},
		}
		r := withTenant(httptest.NewRequest(http.MethodGet, "/?"+query, nil))
		w := httptest.NewRecorder()
		NewWebhookEndpointHandler(svc).GetAll(w, r)
		require.Equal(t, http.StatusOK, w.Code)
		return gotStatus, gotURL
	}

	t.Run("a single status is passed through", func(t *testing.T) {
		status, _ := call(t, "status=active")
		assert.Equal(t, []string{"active"}, status)
	})

	t.Run("comma-joined statuses are split into separate values", func(t *testing.T) {
		status, _ := call(t, "status=active,inactive")
		assert.Equal(t, []string{"active", "inactive"}, status,
			"a joined string would become IN ('active,inactive') and match nothing")
	})

	t.Run("no status filter leaves the list unfiltered", func(t *testing.T) {
		status, _ := call(t, "")
		assert.Empty(t, status)
	})

	t.Run("the url search term reaches the service", func(t *testing.T) {
		_, url := call(t, "url=hooks.example")
		require.NotNil(t, url, "the search box sends ?url=; it must not be dropped")
		assert.Equal(t, "hooks.example", *url)
	})

	t.Run("an empty url search is not treated as a filter", func(t *testing.T) {
		_, url := call(t, "url=")
		assert.Nil(t, url)
	})

	_ = context.Background
}
