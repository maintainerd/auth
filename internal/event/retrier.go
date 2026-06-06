package event

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	retryPollInterval = 30 * time.Second
)

// DeliveryHistoryRetrier supplies the pending delivery records that are due for
// retry. State transitions (success / reschedule / dead-letter) are owned by the
// delivery function, so this interface is read-only.
type DeliveryHistoryRetrier interface {
	FindPendingRetries() ([]DeliveryRetryRecord, error)
}

// DeliveryRetryRecord carries everything the delivery function needs to
// re-attempt one pending webhook delivery.
type DeliveryRetryRecord struct {
	DeliveryHistoryID   int64
	DeliveryHistoryUUID string
	WebhookEndpointID   int64
	EventID             string
	EventType           string
	TenantID            int64
	AttemptCount        int
	URL                 string
	SecretEncrypted     string
	TimeoutSeconds      int
	MaxRetries          int
}

// BackgroundRetrier polls for pending retries and re-attempts delivery.
type BackgroundRetrier struct {
	deliveryFn func(ctx context.Context, record DeliveryRetryRecord) error
	retryRepo  DeliveryHistoryRetrier
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewBackgroundRetrier creates a new background retrier.
func NewBackgroundRetrier(
	retryRepo DeliveryHistoryRetrier,
	deliveryFn func(ctx context.Context, record DeliveryRetryRecord) error,
) *BackgroundRetrier {
	r := &BackgroundRetrier{
		deliveryFn: deliveryFn,
		retryRepo:  retryRepo,
		stopCh:     make(chan struct{}),
	}
	r.wg.Add(1)
	go r.runLoop()
	return r
}

// runLoop runs the retry loop and restarts it if a panic escapes processing,
// so a single bad batch never permanently kills durable retries.
func (r *BackgroundRetrier) runLoop() {
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
					slog.Error("retrier: loop panicked, restarting", "panic", rec)
				}
			}()
			r.loop()
		}()
	}
}

func (r *BackgroundRetrier) loop() {
	ticker := time.NewTicker(retryPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.processRetries(context.Background())
		}
	}
}

func (r *BackgroundRetrier) processRetries(ctx context.Context) {
	_, span := otel.Tracer("retrier").Start(ctx, "retrier.process")
	defer span.End()

	records, err := r.retryRepo.FindPendingRetries()
	if err != nil {
		slog.Error("retrier: find pending failed", "err", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "find pending failed")
		return
	}
	if len(records) == 0 {
		return
	}

	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup

	for _, rec := range records {
		sem <- struct{}{}
		wg.Add(1)
		go func(rec DeliveryRetryRecord) {
			defer wg.Done()
			defer func() { <-sem }()
			r.retryOne(ctx, rec)
		}(rec)
	}

	wg.Wait()
	span.SetAttributes(attribute.Int("retried", len(records)))
	span.SetStatus(codes.Ok, "")
}

// retryOne invokes the delivery function, which performs the attempt and owns
// all state transitions (success / reschedule with jittered backoff /
// dead-letter + endpoint quarantine). The retrier only schedules the work.
func (r *BackgroundRetrier) retryOne(ctx context.Context, record DeliveryRetryRecord) {
	if err := r.deliveryFn(ctx, record); err != nil {
		slog.Warn("retrier: delivery attempt error (will be retried)",
			"delivery_history_id", record.DeliveryHistoryID,
			"event_type", record.EventType,
			"err", err,
		)
	}
}

// Shutdown stops the background retrier.
func (r *BackgroundRetrier) Shutdown() {
	close(r.stopCh)
	r.wg.Wait()
}
