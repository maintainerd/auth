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
	retryPollInterval      = 30 * time.Second
	retryBatchSize         = 50
	quarantineThreshold    = 10 // consecutive failures before quarantining endpoint
)

// DeliveryHistoryRetrier provides the interface for retrieving and updating
// pending delivery records.
type DeliveryHistoryRetrier interface {
	FindPendingRetries() ([]DeliveryRetryRecord, error)
	UpdateAttempt(deliveryHistoryID int64, attemptCount int, responseStatus *int, responseSummary string, errorReason string, nextRetry *time.Time, finalStatus string) error
	MoveToDeadLetter(deliveryHistoryID int64, errorReason string) error
}

// DeliveryRetryRecord is a minimal record for the retrier.
type DeliveryRetryRecord struct {
	DeliveryHistoryID int64
	WebhookEndpointID int64
	EventID           string
	EventType         string
	TenantID          int64
	AttemptCount      int
	URL               string
	SecretEncrypted   string
	TimeoutSeconds    int
	MaxRetries        int
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
	go func() {
		defer func() { recover() }()
		r.loop()
	}()
	return r
}

func (r *BackgroundRetrier) loop() {
	defer r.wg.Done()
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

func (r *BackgroundRetrier) retryOne(ctx context.Context, record DeliveryRetryRecord) {
	attempt := record.AttemptCount + 1
	maxAttempts := record.MaxRetries + 1

	err := r.deliveryFn(ctx, record)
	if err == nil {
		_ = r.retryRepo.UpdateAttempt(
			record.DeliveryHistoryID, attempt,
			nil, "", "",
			nil, "success",
		)
		return
	}

	if attempt >= maxAttempts {
		_ = r.retryRepo.MoveToDeadLetter(record.DeliveryHistoryID, err.Error())
		slog.Warn("retrier: delivery moved to dead-letter",
			"delivery_history_id", record.DeliveryHistoryID,
			"event_type", record.EventType,
			"attempt", attempt,
			"err", err,
		)
		return
	}

	backoff := time.Duration(attempt) * time.Second
	if backoff > 60*time.Second {
		backoff = 60 * time.Second
	}
	nextRetry := time.Now().UTC().Add(backoff)

	_ = r.retryRepo.UpdateAttempt(
		record.DeliveryHistoryID, attempt,
		nil, "", err.Error(),
		&nextRetry, "pending",
	)
	slog.Warn("retrier: delivery retry scheduled",
		"delivery_history_id", record.DeliveryHistoryID,
		"event_type", record.EventType,
		"attempt", attempt,
		"next_retry", nextRetry,
	)
}

// Shutdown stops the background retrier.
func (r *BackgroundRetrier) Shutdown() {
	close(r.stopCh)
	r.wg.Wait()
}
