package webhook

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func withDefaultTransport(t *testing.T, rt http.RoundTripper) {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = rt
	t.Cleanup(func() {
		http.DefaultTransport = original
	})
}

func testAuthEvent() *authevent.AuthEvent {
	return &authevent.AuthEvent{
		TenantID:  1,
		EventType: authevent.AuthEventTypeLoginSuccess,
		Category:  authevent.AuthEventCategoryAuthn,
		Severity:  authevent.AuthEventSeverityInfo,
		Result:    authevent.AuthEventResultSuccess,
		IPAddress: "203.0.113.10",
		CreatedAt: time.Now().UTC(),
	}
}

func TestDoRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		withDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, []byte(`{"ok":true}`), body)
			assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
			assert.Equal(t, "user.login.success", req.Header.Get("X-Maintainerd-Event"))
			assert.Equal(t, "delivery-1", req.Header.Get("X-Maintainerd-Delivery"))
			assert.Equal(t, "1710000000", req.Header.Get("X-Maintainerd-Timestamp"))
			assert.Equal(t, "sha256=test", req.Header.Get("X-Maintainerd-Signature-256"))
			return &http.Response{
				StatusCode: http.StatusNoContent,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}, nil
		}))

		err := doRequest(context.Background(), "https://93.184.216.34/hook", []byte(`{"ok":true}`), "sha256=test", 1710000000, "delivery-1", "user.login.success")

		require.NoError(t, err)
	})

	t.Run("validation error", func(t *testing.T) {
		err := doRequest(context.Background(), "http://example.com/hook", []byte(`{}`), "sig", 1, "delivery", "event")

		require.ErrorContains(t, err, "https")
	})

	t.Run("transport error", func(t *testing.T) {
		withDefaultTransport(t, roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, assert.AnError
		}))

		err := doRequest(context.Background(), "https://93.184.216.34/hook", []byte(`{}`), "sig", 1, "delivery", "event")

		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("unexpected status", func(t *testing.T) {
		withDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}, nil
		}))

		err := doRequest(context.Background(), "https://93.184.216.34/hook", []byte(`{}`), "sig", 1, "delivery", "event")

		require.ErrorContains(t, err, "unexpected status 500")
	})

	t.Run("unsafe redirect error", func(t *testing.T) {
		withDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusFound,
				Header:     http.Header{"Location": []string{"https://127.0.0.1/hook"}},
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}, nil
		}))

		err := doRequest(context.Background(), "https://93.184.216.34/hook", []byte(`{}`), "sig", 1, "delivery", "event")

		require.ErrorContains(t, err, "not allowed")
	})
}

func TestDispatcher_Deliver(t *testing.T) {
	t.Run("marshal payload error stops delivery", func(t *testing.T) {
		called := false
		dispatcher := &Dispatcher{repo: &mockWebhookEndpointRepo{
			updateLastTriggeredFn: func(_ int64, _ time.Time) error {
				called = true
				return nil
			},
		}}
		event := testAuthEvent()
		event.Metadata = []byte(`{bad json`)

		dispatcher.deliver(context.Background(), WebhookEndpoint{
			WebhookEndpointID: 1,
			URL:               "https://93.184.216.34/hook",
			MaxRetries:        0,
			TimeoutSeconds:    1,
		}, event)

		assert.False(t, called)
	})

	t.Run("success updates last triggered", func(t *testing.T) {
		withDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
			assert.NotEmpty(t, req.Header.Get("X-Maintainerd-Delivery"))
			assert.NotEmpty(t, req.Header.Get("X-Maintainerd-Timestamp"))
			assert.Contains(t, req.Header.Get("X-Maintainerd-Signature-256"), "sha256=")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}, nil
		}))
		encrypted, err := crypto.EncryptAtRest("webhook-secret")
		require.NoError(t, err)
		var updatedID int64
		var updatedAt time.Time
		dispatcher := &Dispatcher{repo: &mockWebhookEndpointRepo{
			updateLastTriggeredFn: func(webhookEndpointID int64, t time.Time) error {
				updatedID = webhookEndpointID
				updatedAt = t
				return nil
			},
		}}

		dispatcher.deliver(context.Background(), WebhookEndpoint{
			WebhookEndpointID: 1,
			URL:               "https://93.184.216.34/hook",
			SecretEncrypted:   encrypted,
			MaxRetries:        0,
			TimeoutSeconds:    1,
		}, testAuthEvent())

		assert.Equal(t, int64(1), updatedID)
		assert.WithinDuration(t, time.Now(), updatedAt, time.Second)
	})

	t.Run("context cancellation stops retry before update", func(t *testing.T) {
		withDefaultTransport(t, roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, assert.AnError
		}))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		updated := false
		dispatcher := &Dispatcher{repo: &mockWebhookEndpointRepo{
			updateLastTriggeredFn: func(_ int64, _ time.Time) error {
				updated = true
				return nil
			},
		}}

		dispatcher.deliver(ctx, WebhookEndpoint{
			WebhookEndpointID: 1,
			URL:               "https://93.184.216.34/hook",
			MaxRetries:        1,
			TimeoutSeconds:    1,
		}, testAuthEvent())

		assert.False(t, updated)
	})

	t.Run("update last triggered error is swallowed", func(t *testing.T) {
		withDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}, nil
		}))
		dispatcher := &Dispatcher{repo: &mockWebhookEndpointRepo{
			updateLastTriggeredFn: func(_ int64, _ time.Time) error {
				return assert.AnError
			},
		}}

		dispatcher.deliver(context.Background(), WebhookEndpoint{
			WebhookEndpointID: 1,
			URL:               "https://93.184.216.34/hook",
			MaxRetries:        0,
			TimeoutSeconds:    1,
		}, testAuthEvent())
	})

	t.Run("retries with capped backoff then updates", func(t *testing.T) {
		attempts := 0
		withDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}, nil
		}))
		originalSleep := webhookBackoffSleep
		var sleeps []time.Duration
		webhookBackoffSleep = func(d time.Duration) <-chan time.Time {
			sleeps = append(sleeps, d)
			ch := make(chan time.Time, 1)
			ch <- time.Now()
			return ch
		}
		t.Cleanup(func() {
			webhookBackoffSleep = originalSleep
		})
		updated := false
		dispatcher := &Dispatcher{repo: &mockWebhookEndpointRepo{
			updateLastTriggeredFn: func(_ int64, _ time.Time) error {
				updated = true
				return nil
			},
		}}

		dispatcher.deliver(context.Background(), WebhookEndpoint{
			WebhookEndpointID: 1,
			URL:               "https://93.184.216.34/hook",
			MaxRetries:        8,
			TimeoutSeconds:    1,
		}, testAuthEvent())

		assert.Equal(t, 9, attempts)
		assert.Contains(t, sleeps, webhookMaxBackoff)
		assert.True(t, updated)
	})
}
