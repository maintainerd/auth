package event

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	relayPollInterval   = 5 * time.Second
	relayBatchSize      = 50
	relayMaxConcurrency = 4
)

// WebhookDeliveryFunc delivers a single outbox event to subscribed webhook endpoints.
// The implementation lives in the webhook package to avoid circular imports.
type WebhookDeliveryFunc func(ctx context.Context, outbox *Outbox) error

// BrokerDeliveryFunc delivers a single outbox event to the RabbitMQ broker.
type BrokerDeliveryFunc func(ctx context.Context, outbox *Outbox) error

// Relay reads unpublished outbox rows and dispatches them to webhook and broker delivery.
type Relay struct {
	outboxRepo     OutboxRepository
	deliverWebhook WebhookDeliveryFunc
	deliverBroker  BrokerDeliveryFunc

	wg     sync.WaitGroup
	stopCh chan struct{}
}

// NewRelay creates a new outbox relay.
func NewRelay(
	outboxRepo OutboxRepository,
	deliverWebhook WebhookDeliveryFunc,
	deliverBroker BrokerDeliveryFunc,
) *Relay {
	r := &Relay{
		outboxRepo:     outboxRepo,
		deliverWebhook: deliverWebhook,
		deliverBroker:  deliverBroker,
		stopCh:         make(chan struct{}),
	}

	r.wg.Add(1)
	go r.runLoop()

	return r
}

// runLoop runs the relay loop and restarts it if a panic escapes processing,
// so a single bad batch never permanently kills the relay.
func (r *Relay) runLoop() {
	defer r.wg.Done()
	for {
		select {
		case <-r.stopCh:
			return
		default:
		}
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					slog.Error("relay: loop panicked, restarting", "panic", rec)
				}
			}()
			r.loop()
		}()
	}
}

func (r *Relay) loop() {
	ticker := time.NewTicker(relayPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.processBatch(context.Background())
		}
	}
}

func (r *Relay) processBatch(ctx context.Context) {
	// ClaimUnpublished uses FOR UPDATE SKIP LOCKED so concurrent relay replicas
	// never process the same outbox row.
	rows, err := r.outboxRepo.ClaimUnpublished(relayBatchSize)
	if err != nil {
		slog.Error("relay: claim unpublished failed", "err", err)
		return
	}
	if len(rows) == 0 {
		return
	}

	sem := make(chan struct{}, relayMaxConcurrency)
	var wg sync.WaitGroup

	for i := range rows {
		row := rows[i]
		sem <- struct{}{}
		wg.Add(1)

		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			r.deliverOne(ctx, &row)
		}()
	}

	wg.Wait()
}

func (r *Relay) deliverOne(ctx context.Context, row *Outbox) {
	// The two arms are INDEPENDENT and each runs at most once to completion. A row
	// carries per-arm state (webhook_delivered_at / broker_published_at), so an
	// arm already done is skipped on re-claim. This is what prevents a broker
	// outage from re-running the webhook arm and fanning out duplicate deliveries
	// (and growing delivery_history) every claim while the broker is down.
	webhookDone := row.WebhookDeliveredAt != nil || r.deliverWebhook == nil
	brokerDone := row.BrokerPublishedAt != nil || r.deliverBroker == nil

	if !webhookDone {
		if err := r.deliverWebhook(ctx, row); err != nil {
			slog.Error("relay: webhook delivery failed",
				"event_type", row.EventType, "tenant_id", row.TenantID, "outbox_id", row.OutboxID, "err", err)
		} else if mErr := r.outboxRepo.MarkWebhookDelivered(row.OutboxID); mErr != nil {
			slog.Error("relay: mark webhook delivered failed", "outbox_id", row.OutboxID, "err", mErr)
		} else {
			webhookDone = true
		}
	}

	if !brokerDone {
		if err := r.deliverBroker(ctx, row); err != nil {
			slog.Error("relay: broker delivery failed",
				"event_type", row.EventType, "tenant_id", row.TenantID, "outbox_id", row.OutboxID, "err", err)
		} else if mErr := r.outboxRepo.MarkBrokerPublished(row.OutboxID); mErr != nil {
			slog.Error("relay: mark broker published failed", "outbox_id", row.OutboxID, "err", mErr)
		} else {
			brokerDone = true
		}
	}

	// The row is fully published only when BOTH arms are done. Otherwise it stays
	// unpublished; its claim expires and a later poll re-runs ONLY the arm that is
	// still incomplete (at-least-once, per-arm). Per-endpoint HTTP outcomes are
	// owned by delivery_history + the BackgroundRetrier, not the outbox row — the
	// webhook arm reports success once the fan-out is durably recorded.
	if webhookDone && brokerDone {
		if err := r.outboxRepo.MarkPublished(row.OutboxID); err != nil {
			slog.Error("relay: mark published failed", "outbox_id", row.OutboxID, "err", err)
		}
	}
}

// Shutdown stops the relay loop and waits for in-flight deliveries.
func (r *Relay) Shutdown() {
	close(r.stopCh)
	r.wg.Wait()
}
