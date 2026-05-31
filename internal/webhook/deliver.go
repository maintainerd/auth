package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/crypto"
)

const (
	// webhookSuccessMaxStatus is the highest HTTP status code still
	// considered a successful delivery (exclusive).
	webhookSuccessMaxStatus = 300

	// webhookMaxBackoff caps the exponential backoff between retries.
	webhookMaxBackoff = 60 * time.Second
)

func (d *Dispatcher) deliver(ctx context.Context, ep WebhookEndpoint, event *authevent.AuthEvent) {
	payload := buildPayload(event)
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("webhook: marshal payload", "err", err)
		return
	}

	timestamp := time.Now().Unix()
	secret := crypto.SafeDecryptAtRest(ep.SecretEncrypted)
	sig := computeSignature(secret, timestamp, body)
	deliveryID := uuid.New().String()
	timeout := time.Duration(ep.TimeoutSeconds) * time.Second

	maxAttempts := ep.MaxRetries + 1
	backoff := time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		err = doRequest(reqCtx, ep.URL, body, sig, timestamp, deliveryID, event.EventType)
		cancel()

		if err == nil {
			break
		}

		slog.Warn("webhook: delivery failed",
			"url", ep.URL,
			"attempt", attempt,
			"max_attempts", maxAttempts,
			"err", err,
		)
		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				backoff *= 2
				if backoff > webhookMaxBackoff {
					backoff = webhookMaxBackoff
				}
			}
		}
	}

	now := time.Now()
	if updErr := d.repo.UpdateLastTriggeredAt(ep.WebhookEndpointID, now); updErr != nil {
		slog.Error("webhook: update last_triggered_at", "webhook_endpoint_id", ep.WebhookEndpointID, "err", updErr)
	}
}

func doRequest(ctx context.Context, url string, body []byte, sig string, timestamp int64, deliveryID, eventType string) error {
	if err := validateWebhookURL(ctx, url, true); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Maintainerd-Event", eventType)
	req.Header.Set("X-Maintainerd-Delivery", deliveryID)
	req.Header.Set("X-Maintainerd-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Maintainerd-Signature-256", sig)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			return validateWebhookURL(req.Context(), req.URL.String(), true)
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= webhookSuccessMaxStatus {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}
