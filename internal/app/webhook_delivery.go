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
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/event"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/webhook"
)

const (
	webhookSuccessMaxStatus = 300
	webhookMaxBackoff       = 60 * time.Second
	// quarantineThreshold is the number of consecutive dead-lettered deliveries
	// (counted per endpoint, reset on any success) before an endpoint is
	// auto-quarantined so a dead endpoint is not retried for every event forever.
	quarantineThreshold = 10
	// maxInlineDeliveryConcurrency bounds the parallel first-attempt fan-out for a
	// single outbox event so a slow/hostile endpoint set cannot serialize the
	// relay and stall other tenants' events.
	maxInlineDeliveryConcurrency = 8
)

// deliverToWebhooks is the relay hand-off for one outbox event. It creates a
// durable delivery_history row for each subscribed endpoint and performs the
// first delivery attempt. Per-endpoint retries thereafter are owned by the
// BackgroundRetrier (driven by delivery_history.next_retry_time).
//
// It returns an error only when the hand-off itself cannot proceed (the endpoint
// list cannot be loaded), so the relay leaves the outbox row unpublished for
// re-claim. A successful return means the event was durably fanned out — the
// relay may then mark it published even if individual HTTP attempts failed,
// because recovery now lives in delivery_history, not the outbox row.
func deliverToWebhooks(
	ctx context.Context,
	outbox *event.Outbox,
	endpointRepo webhook.WebhookEndpointRepository,
	historyRepo webhook.DeliveryHistoryRepository,
	endpointEventRepo webhook.WebhookEndpointEventRepository,
) error {
	endpoints, err := endpointRepo.FindActiveByTenantID(outbox.TenantID)
	if err != nil {
		return fmt.Errorf("find active endpoints: %w", err)
	}

	body, err := buildDeliveryBody(outbox)
	if err != nil {
		// A thin payload that cannot be marshalled is a poison message: log and
		// return nil so the relay marks it published rather than wedging the
		// queue retrying an un-deliverable row forever.
		slog.Error("webhook: marshal payload failed (dropping poison event)",
			"event_id", outbox.EventID, "event_type", outbox.EventType, "err", err)
		return nil
	}

	// Create the durable delivery_history row for each subscribed endpoint first
	// (fast, O(n) DB writes), collecting the work items.
	type deliveryJob struct {
		ep      webhook.WebhookEndpoint
		history *webhook.DeliveryHistory
	}
	jobs := make([]deliveryJob, 0, len(endpoints))
	for i := range endpoints {
		ep := endpoints[i]
		if !endpointMatchesEvent(ep, endpointEventRepo, outbox.EventType) {
			continue
		}
		created, err := historyRepo.Create(&webhook.DeliveryHistory{
			WebhookEndpointID: ep.WebhookEndpointID,
			EventID:           outbox.EventID,
			EventType:         outbox.EventType,
			TenantID:          outbox.TenantID,
			AttemptCount:      0,
			FinalStatus:       "pending",
		})
		if err != nil {
			slog.Error("webhook: create delivery history failed",
				"endpoint_id", ep.WebhookEndpointID, "err", err)
			continue
		}
		jobs = append(jobs, deliveryJob{ep: ep, history: created})
	}

	// Run the first attempts with BOUNDED concurrency. A sequential loop lets one
	// slow or hostile endpoint (up to TimeoutSeconds each) serialize the whole
	// fan-out and hold a scarce relay slot for sum(timeouts) — a cross-tenant
	// noisy-neighbor stall. Bounding it caps the relay hold near a single timeout.
	sem := make(chan struct{}, maxInlineDeliveryConcurrency)
	var wg sync.WaitGroup
	for i := range jobs {
		j := jobs[i]
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			attemptOnce(ctx, &j.ep, outbox.EventID, outbox.EventType, body, j.history, historyRepo, endpointRepo)
			_ = endpointRepo.UpdateLastTriggeredAt(j.ep.WebhookEndpointID, time.Now())
		}()
	}
	wg.Wait()

	return nil
}

// endpointMatchesEvent reports whether an endpoint should receive this event type.
func endpointMatchesEvent(ep webhook.WebhookEndpoint, repo webhook.WebhookEndpointEventRepository, eventType string) bool {
	if ep.SubscribeAll {
		return true
	}
	if repo == nil {
		return false
	}
	matched, err := repo.ExistsByEndpointAndEventKey(ep.WebhookEndpointID, eventType)
	if err != nil {
		slog.Warn("webhook: subscription lookup failed", "endpoint_id", ep.WebhookEndpointID, "err", err)
		return false
	}
	return matched
}

// attemptOnce performs a single delivery attempt and records the outcome. It is
// the sole place delivery state transitions happen and is shared by the relay's
// first attempt and the BackgroundRetrier's subsequent attempts:
//   - success           -> history=success, endpoint failure counter reset
//   - retryable failure  -> history=pending + jittered next_retry_time
//   - attempts exhausted -> history=dead_letter, endpoint counter++ (+ quarantine)
func attemptOnce(
	ctx context.Context,
	ep *webhook.WebhookEndpoint,
	eventID uuid.UUID,
	eventType string,
	body []byte,
	history *webhook.DeliveryHistory,
	historyRepo webhook.DeliveryHistoryRepository,
	endpointRepo webhook.WebhookEndpointRepository,
) {
	attempt := history.AttemptCount + 1
	maxAttempts := ep.MaxRetries + 1

	timestamp := time.Now().Unix()
	secret := crypto.SafeDecryptAtRest(ep.SecretEncrypted)
	sig := computeWebhookSignature(secret, timestamp, body)
	timeout := time.Duration(ep.TimeoutSeconds) * time.Second

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	statusCode, summary, deliveryErr := doDeliveryRequest(
		reqCtx, ep.URL, body, sig, timestamp, eventType, eventID.String(),
		history.DeliveryHistoryUUID.String(), attempt,
	)
	cancel()

	if deliveryErr == nil {
		_ = historyRepo.UpdateAttempt(history.DeliveryHistoryID, attempt, &statusCode, summary, "", nil, "success")
		_ = endpointRepo.ResetConsecutiveFailures(ep.WebhookEndpointID)
		return
	}

	if attempt < maxAttempts {
		next := time.Now().UTC().Add(jitteredBackoff(attempt))
		_ = historyRepo.UpdateAttempt(history.DeliveryHistoryID, attempt, nil, summary, deliveryErr.Error(), &next, "pending")
		return
	}

	// Attempts exhausted: dead-letter and account the failure against the endpoint.
	_ = historyRepo.MoveToDeadLetter(history.DeliveryHistoryID, deliveryErr.Error())
	failures, err := endpointRepo.IncrementConsecutiveFailures(ep.WebhookEndpointID)
	if err != nil {
		slog.Warn("webhook: increment failure counter failed", "endpoint_id", ep.WebhookEndpointID, "err", err)
		return
	}
	if failures >= quarantineThreshold {
		if qErr := endpointRepo.Quarantine(ep.WebhookEndpointID); qErr != nil {
			slog.Error("webhook: quarantine failed", "endpoint_id", ep.WebhookEndpointID, "err", qErr)
			return
		}
		// A quarantined endpoint becomes inactive, and the retrier skips inactive
		// endpoints — so any still-pending deliveries for it would be orphaned
		// (never retried, never dead-lettered) until purge. Dead-letter them now so
		// their terminal state is explicit and observable.
		if n, dErr := historyRepo.DeadLetterPendingByEndpoint(ep.WebhookEndpointID, "endpoint quarantined after sustained delivery failures"); dErr != nil {
			slog.Error("webhook: dead-letter pending on quarantine failed", "endpoint_id", ep.WebhookEndpointID, "err", dErr)
		} else if n > 0 {
			slog.Warn("webhook: dead-lettered orphaned pending deliveries for quarantined endpoint",
				"endpoint_id", ep.WebhookEndpointID, "count", n)
		}
		slog.Warn("webhook: endpoint quarantined after sustained failures",
			"endpoint_id", ep.WebhookEndpointID, "consecutive_failures", failures)
	}
}

func doDeliveryRequest(
	ctx context.Context,
	url string,
	body []byte,
	sig string,
	timestamp int64,
	eventType, eventID, deliveryID string,
	attempt int,
) (int, string, error) {
	// SSRF defense at delivery time (and on each redirect hop) — closes the
	// DNS-rebinding window that registration-time validation cannot cover.
	if err := webhook.ValidateDeliveryURL(ctx, url); err != nil {
		return 0, "", fmt.Errorf("destination rejected: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Maintainerd-Event", eventType)
	req.Header.Set("X-Maintainerd-Event-Id", eventID)
	req.Header.Set("X-Maintainerd-Delivery", deliveryID)
	req.Header.Set("X-Maintainerd-Attempt", strconv.Itoa(attempt))
	req.Header.Set("X-Maintainerd-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Maintainerd-Signature-256", sig)

	// Shared SSRF-hardened client: pins the resolved IP at dial time (closes the
	// DNS-rebinding TOCTOU) and re-validates https on every redirect hop.
	resp, err := webhook.SafeDeliveryClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= webhookSuccessMaxStatus {
		return resp.StatusCode, fmt.Sprintf("HTTP %d", resp.StatusCode), fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return resp.StatusCode, fmt.Sprintf("HTTP %d", resp.StatusCode), nil
}

// jitteredBackoff returns an exponential backoff with full jitter (random in
// [base/2, base]) capped at webhookMaxBackoff, to avoid retry thundering herds.
func jitteredBackoff(attempt int) time.Duration {
	exp := attempt - 1
	if exp > 6 {
		exp = 6 // cap the shift; 2^6 = 64 already exceeds the 60s cap
	}
	base := time.Second * time.Duration(1<<uint(exp))
	if base > webhookMaxBackoff {
		base = webhookMaxBackoff
	}
	half := int64(base / 2)
	if half <= 0 {
		return base
	}
	// Jitter for retry backoff is timing variance, not a security primitive, so a
	// non-cryptographic PRNG is appropriate here.
	return time.Duration(half) + time.Duration(rand.Int64N(half+1)) // #nosec G404 -- non-security retry-backoff jitter
}

func computeWebhookSignature(secret string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func buildDeliveryBody(outbox *event.Outbox) ([]byte, error) {
	return json.Marshal(event.OutboxPayload(outbox))
}

// newRetryDeliveryFn builds the BackgroundRetrier's delivery function. It
// reconstructs the event body from the outbox, rebuilds endpoint context from
// the retry record, and runs the shared attemptOnce (which owns all state
// transitions). If the outbox event is no longer available (purged), the
// pending delivery is dead-lettered rather than retried forever.
func newRetryDeliveryFn(
	outboxRepo event.OutboxRepository,
	endpointRepo webhook.WebhookEndpointRepository,
	historyRepo webhook.DeliveryHistoryRepository,
) func(ctx context.Context, rec event.DeliveryRetryRecord) error {
	return func(ctx context.Context, rec event.DeliveryRetryRecord) error {
		eventID, err := uuid.Parse(rec.EventID)
		if err != nil {
			_ = historyRepo.MoveToDeadLetter(rec.DeliveryHistoryID, "invalid event id")
			return nil
		}
		outbox, err := outboxRepo.FindByEventID(eventID)
		if err != nil {
			return err // transient; retrier will pick it up again
		}
		if outbox == nil {
			_ = historyRepo.MoveToDeadLetter(rec.DeliveryHistoryID, "outbox event no longer available")
			return nil
		}
		body, err := buildDeliveryBody(outbox)
		if err != nil {
			_ = historyRepo.MoveToDeadLetter(rec.DeliveryHistoryID, "payload marshal failed")
			return nil
		}

		deliveryUUID, _ := uuid.Parse(rec.DeliveryHistoryUUID)
		ep := &webhook.WebhookEndpoint{
			WebhookEndpointID: rec.WebhookEndpointID,
			URL:               rec.URL,
			SecretEncrypted:   rec.SecretEncrypted,
			TimeoutSeconds:    rec.TimeoutSeconds,
			MaxRetries:        rec.MaxRetries,
		}
		history := &webhook.DeliveryHistory{
			DeliveryHistoryID:   rec.DeliveryHistoryID,
			DeliveryHistoryUUID: deliveryUUID,
			AttemptCount:        rec.AttemptCount,
		}
		attemptOnce(ctx, ep, eventID, rec.EventType, body, history, historyRepo, endpointRepo)
		return nil
	}
}

// newReplayFn builds the function backing the webhook replay API. It loads the
// original event from the outbox, records a fresh (is_replay) delivery history
// row, and runs the shared attemptOnce delivery path.
func newReplayFn(
	outboxRepo event.OutboxRepository,
	historyRepo webhook.DeliveryHistoryRepository,
	endpointRepo webhook.WebhookEndpointRepository,
) func(ctx context.Context, ep webhook.WebhookEndpoint, eventID uuid.UUID, isReplay bool) error {
	return func(ctx context.Context, ep webhook.WebhookEndpoint, eventID uuid.UUID, isReplay bool) error {
		outbox, err := outboxRepo.FindByEventID(eventID)
		if err != nil {
			return err
		}
		if outbox == nil {
			return fmt.Errorf("event %s not found", eventID)
		}
		// Tenant isolation: only replay an event that belongs to the (tenant-scoped)
		// endpoint's tenant. eventID comes from the request body, so without this an
		// attacker could replay another tenant's outbox payload to their own endpoint.
		if outbox.TenantID != ep.TenantID {
			return fmt.Errorf("event %s not found", eventID)
		}
		body, err := buildDeliveryBody(outbox)
		if err != nil {
			return err
		}
		history, err := historyRepo.Create(&webhook.DeliveryHistory{
			WebhookEndpointID: ep.WebhookEndpointID,
			EventID:           outbox.EventID,
			EventType:         outbox.EventType,
			TenantID:          outbox.TenantID,
			AttemptCount:      0,
			FinalStatus:       "pending",
			IsReplay:          isReplay,
		})
		if err != nil {
			return err
		}
		attemptOnce(ctx, &ep, outbox.EventID, outbox.EventType, body, history, historyRepo, endpointRepo)
		return nil
	}
}

// newBrokerDeliverFn builds the relay's broker (RabbitMQ) delivery arm. It only
// publishes an event when the tenant has an enabled event_route for that type,
// and no-ops when the publisher is disabled (no AMQP config) — so the broker
// channel is structurally complete and activates once an AMQP publishFunc is
// injected into the RabbitMQPublisher.
func newBrokerDeliverFn(
	publisher *event.RabbitMQPublisher,
	eventTypeRepo event.EventTypeRepository,
	eventRouteRepo event.EventRouteRepository,
) func(ctx context.Context, outbox *event.Outbox) error {
	return func(ctx context.Context, outbox *event.Outbox) error {
		if !publisher.IsEnabled() {
			return nil
		}
		et, err := eventTypeRepo.FindByKeyAndTenantID(outbox.EventType, outbox.TenantID)
		if err != nil {
			return err
		}
		if et == nil {
			return nil
		}
		route, err := eventRouteRepo.FindByTenantIDAndEventTypeID(outbox.TenantID, et.EventTypeID)
		if err != nil {
			return err
		}
		if route == nil || !route.Enabled {
			return nil // no enabled broker route for this tenant + event type
		}
		return publisher.Publish(ctx, outbox)
	}
}

// deliveryRetrierAdapter implements event.DeliveryHistoryRetrier by joining
// pending delivery_history rows with their owning endpoint.
type deliveryRetrierAdapter struct {
	historyRepo  webhook.DeliveryHistoryRepository
	endpointRepo webhook.WebhookEndpointRepository
}

func (a *deliveryRetrierAdapter) FindPendingRetries() ([]event.DeliveryRetryRecord, error) {
	histories, err := a.historyRepo.FindPendingRetries()
	if err != nil {
		return nil, err
	}
	records := make([]event.DeliveryRetryRecord, 0, len(histories))
	for _, h := range histories {
		ep, err := a.endpointRepo.FindByID(h.WebhookEndpointID)
		if err != nil || ep == nil {
			continue
		}
		// Skip endpoints that are no longer active (e.g. quarantined/disabled).
		if ep.Status != "active" {
			continue
		}
		records = append(records, event.DeliveryRetryRecord{
			DeliveryHistoryID:   h.DeliveryHistoryID,
			DeliveryHistoryUUID: h.DeliveryHistoryUUID.String(),
			WebhookEndpointID:   h.WebhookEndpointID,
			EventID:             h.EventID.String(),
			EventType:           h.EventType,
			TenantID:            h.TenantID,
			AttemptCount:        h.AttemptCount,
			URL:                 ep.URL,
			SecretEncrypted:     ep.SecretEncrypted,
			TimeoutSeconds:      ep.TimeoutSeconds,
			MaxRetries:          ep.MaxRetries,
		})
	}
	return records, nil
}
