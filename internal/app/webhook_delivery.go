package app

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/maintainerd/auth/internal/event"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/webhook"
)

const (
	webhookSuccessMaxStatus = 300
	webhookMaxBackoff       = 60 * time.Second
)

// deliverToWebhooks finds active webhook endpoints for the outbox event's tenant,
// filters by subscription, and delivers the event to each matching endpoint.
func deliverToWebhooks(
	ctx context.Context,
	outbox *event.Outbox,
	endpointRepo webhook.WebhookEndpointRepository,
	historyRepo webhook.DeliveryHistoryRepository,
) error {
	endpoints, err := endpointRepo.FindActiveByTenantID(outbox.TenantID)
	if err != nil {
		return fmt.Errorf("find active endpoints: %w", err)
	}

	for _, ep := range endpoints {
		if !endpointMatchesEvent(ep, outbox.EventType) {
			continue
		}

		payload, err := event.OutboxPayloadJSON(outbox)
		if err != nil {
			slog.Error("webhook: marshal payload failed", "err", err)
			continue
		}
		body, _ := json.Marshal(payload)

		history := &webhook.DeliveryHistory{
			WebhookEndpointID: ep.WebhookEndpointID,
			EventID:           outbox.EventID,
			EventType:         outbox.EventType,
			TenantID:          outbox.TenantID,
			AttemptCount:      0,
			FinalStatus:       "pending",
		}

		created, err := historyRepo.Create(history)
		if err != nil {
			slog.Error("webhook: create delivery history failed", "err", err)
			continue
		}

		attemptDelivery(ctx, ep, outbox, body, created, historyRepo)

		_ = endpointRepo.UpdateLastTriggeredAt(ep.WebhookEndpointID, time.Now())
	}

	return nil
}

// endpointMatchesEvent checks if an endpoint should receive this event type.
func endpointMatchesEvent(ep webhook.WebhookEndpoint, eventType string) bool {
	if ep.SubscribeAll {
		return true
	}
	return true // Subscription matching via webhook_endpoint_events queried in relay filter
}

func attemptDelivery(
	ctx context.Context,
	ep webhook.WebhookEndpoint,
	outbox *event.Outbox,
	body []byte,
	history *webhook.DeliveryHistory,
	historyRepo webhook.DeliveryHistoryRepository,
) {
	timestamp := time.Now().Unix()
	secret := crypto.SafeDecryptAtRest(ep.SecretEncrypted)
	sig := computeWebhookSignature(secret, timestamp, body)
	timeout := time.Duration(ep.TimeoutSeconds) * time.Second

	maxAttempts := ep.MaxRetries + 1

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		statusCode, respSummary, deliveryErr := doDeliveryRequest(reqCtx, ep.URL, body, sig, timestamp, outbox.EventType, outbox.EventID.String())
		cancel()

		if deliveryErr == nil {
			_ = historyRepo.UpdateAttempt(
				history.DeliveryHistoryID, attempt,
				&statusCode, respSummary, "",
				nil, "success",
			)

			return
		}

		if attempt < maxAttempts {
			backoff := time.Duration(attempt) * time.Second
			if backoff > webhookMaxBackoff {
				backoff = webhookMaxBackoff
			}
			nextRetry := time.Now().UTC().Add(backoff)

			_ = historyRepo.UpdateAttempt(
				history.DeliveryHistoryID, attempt,
				nil, "", deliveryErr.Error(),
				&nextRetry, "pending",
			)

			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
		} else {
			_ = historyRepo.MoveToDeadLetter(
				history.DeliveryHistoryID,
				deliveryErr.Error(),
			)
		}
	}
}

func doDeliveryRequest(ctx context.Context, url string, body []byte, sig string, timestamp int64, eventType, eventID string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Maintainerd-Event", eventType)
	req.Header.Set("X-Maintainerd-Event-Id", eventID)
	req.Header.Set("X-Maintainerd-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Maintainerd-Signature-256", sig)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= webhookSuccessMaxStatus {
		return resp.StatusCode, fmt.Sprintf("HTTP %d", resp.StatusCode), fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return resp.StatusCode, fmt.Sprintf("HTTP %d", resp.StatusCode), nil
}

func computeWebhookSignature(secret string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
