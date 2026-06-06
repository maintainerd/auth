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
	var webhookErr, brokerErr error

	if r.deliverWebhook != nil {
		webhookErr = r.deliverWebhook(ctx, row)
		if webhookErr != nil {
			slog.Error("relay: webhook delivery failed",
				"event_type", row.EventType,
				"tenant_id", row.TenantID,
				"outbox_id", row.OutboxID,
				"err", webhookErr,
			)
		}
	}

	if r.deliverBroker != nil {
		brokerErr = r.deliverBroker(ctx, row)
		if brokerErr != nil {
			slog.Error("relay: broker delivery failed",
				"event_type", row.EventType,
				"tenant_id", row.TenantID,
				"outbox_id", row.OutboxID,
				"err", brokerErr,
			)
		}
	}

	// Mark published only when the hand-off to BOTH channels succeeded. A
	// hand-off failure leaves the row unpublished; its claim expires and a relay
	// worker re-claims it on a later poll (at-least-once). Per-endpoint HTTP
	// outcomes are NOT a hand-off failure — those are durably recorded in
	// delivery_history and retried by the BackgroundRetrier, so a failed POST
	// does not block the outbox row from being marked published.
	if webhookErr != nil || brokerErr != nil {
		return
	}

	if err := r.outboxRepo.MarkPublished(row.OutboxID); err != nil {
		slog.Error("relay: mark published failed",
			"outbox_id", row.OutboxID,
			"err", err,
		)
	}
}

// Shutdown stops the relay loop and waits for in-flight deliveries.
func (r *Relay) Shutdown() {
	close(r.stopCh)
	r.wg.Wait()
}
