package webhook

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDispatcher(t *testing.T) {
	t.Run("success starts workers", func(t *testing.T) {
		dispatcher := NewDispatcher(&mockWebhookEndpointRepo{})

		require.NotNil(t, dispatcher)
		assert.NotNil(t, dispatcher.jobs)

		dispatcher.Shutdown()
	})
}

func TestDispatcher_Dispatch(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		dispatcher := &Dispatcher{repo: &mockWebhookEndpointRepo{
			findActiveByTenantIDFn: func(tenantID int64) ([]WebhookEndpoint, error) {
				assert.Equal(t, int64(1), tenantID)
				return nil, assert.AnError
			},
		}}

		dispatcher.Dispatch(context.Background(), testAuthEvent())
	})

	t.Run("queues matching endpoints and skips nonmatching", func(t *testing.T) {
		dispatcher := &Dispatcher{
			repo: &mockWebhookEndpointRepo{
				findActiveByTenantIDFn: func(_ int64) ([]WebhookEndpoint, error) {
					return []WebhookEndpoint{
						{WebhookEndpointID: 1, SubscribeAll: true, Status: shared.StatusActive},
						{WebhookEndpointID: 2, SubscribeAll: false, Status: shared.StatusActive},
					}, nil
				},
			},
			jobs: make(chan webhookDelivery, 1),
		}

		dispatcher.Dispatch(context.Background(), testAuthEvent())

		job := <-dispatcher.jobs
		assert.Equal(t, int64(1), job.ep.WebhookEndpointID)
		assert.Empty(t, dispatcher.jobs)
	})
}

func TestDispatcher_Enqueue(t *testing.T) {
	t.Run("closed dispatcher drops job", func(t *testing.T) {
		dispatcher := &Dispatcher{jobs: make(chan webhookDelivery, 1), closed: true}

		dispatcher.enqueue(webhookDelivery{event: testAuthEvent()})

		assert.Empty(t, dispatcher.jobs)
	})

	t.Run("full queue drops job", func(t *testing.T) {
		dispatcher := &Dispatcher{jobs: make(chan webhookDelivery, 1)}
		dispatcher.jobs <- webhookDelivery{event: testAuthEvent()}

		dispatcher.enqueue(webhookDelivery{event: testAuthEvent(), ep: WebhookEndpoint{WebhookEndpointID: 2}})

		assert.Len(t, dispatcher.jobs, 1)
	})
}

func TestDispatcher_Worker(t *testing.T) {
	t.Run("delivers queued jobs", func(t *testing.T) {
		withDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNoContent,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}, nil
		}))
		encrypted, err := crypto.EncryptAtRest("webhook-secret")
		require.NoError(t, err)
		updated := make(chan int64, 1)
		dispatcher := &Dispatcher{
			repo: &mockWebhookEndpointRepo{
				updateLastTriggeredFn: func(webhookEndpointID int64, _ time.Time) error {
					updated <- webhookEndpointID
					return nil
				},
			},
			jobs: make(chan webhookDelivery, 1),
		}
		dispatcher.wg.Add(1)
		go dispatcher.worker()

		dispatcher.jobs <- webhookDelivery{
			ctx:   context.Background(),
			event: testAuthEvent(),
			ep: WebhookEndpoint{
				WebhookEndpointID: 7,
				URL:               "https://93.184.216.34/hook",
				SecretEncrypted:   encrypted,
				TimeoutSeconds:    1,
			},
		}
		close(dispatcher.jobs)
		dispatcher.wg.Wait()

		assert.Equal(t, int64(7), <-updated)
	})
}

func TestDispatcher_Shutdown(t *testing.T) {
	t.Run("idempotent", func(t *testing.T) {
		dispatcher := NewDispatcher(&mockWebhookEndpointRepo{})

		dispatcher.Shutdown()
		dispatcher.Shutdown()

		assert.True(t, dispatcher.closed)
	})
}
