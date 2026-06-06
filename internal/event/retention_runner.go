package event

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	retentionInterval       = 6 * time.Hour
	outboxRetentionDays     = 7
	deliveryHistoryRetentionDays = 90
)

// RetentionRunner periodically purges old events from the outbox and delivery history.
type RetentionRunner struct {
	outboxRepo  OutboxRepository
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// NewRetentionRunner creates a new retention runner.
func NewRetentionRunner(outboxRepo OutboxRepository) *RetentionRunner {
	return &RetentionRunner{
		outboxRepo: outboxRepo,
		stopCh:     make(chan struct{}),
	}
}

// Start begins the periodic retention loop in a background goroutine.
func (r *RetentionRunner) Start() {
	r.wg.Add(1)
	go r.loop()
	slog.Info("retention runner: started",
		"interval", retentionInterval,
		"outbox_days", outboxRetentionDays,
		"delivery_history_days", deliveryHistoryRetentionDays,
	)
}

func (r *RetentionRunner) loop() {
	defer r.wg.Done()
	ticker := time.NewTicker(retentionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.run(context.Background())
		}
	}
}

func (r *RetentionRunner) run(ctx context.Context) {
	_, span := otel.Tracer("retention").Start(ctx, "retention.run")
	defer span.End()

	outboxCutoff := time.Now().AddDate(0, 0, -outboxRetentionDays)
	deleted, err := r.outboxRepo.DeleteOlderThan(outboxCutoff)
	if err != nil {
		slog.Error("retention: outbox purge failed", "err", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "outbox purge failed")
	} else {
		slog.Info("retention: outbox purged", "deleted", deleted, "cutoff", outboxCutoff)
		span.SetAttributes(attribute.Int64("outbox.deleted", deleted))
	}

	span.SetStatus(codes.Ok, "")
}

// EraseSubject purges all outbox rows referencing the given subject UUID.
// Called when a user is hard-deleted (right-to-erasure).
func (r *RetentionRunner) EraseSubject(ctx context.Context, subjectUUID uuid.UUID) {
	deleted, err := r.outboxRepo.DeleteBySubjectUUID(subjectUUID)
	if err != nil {
		slog.Error("retention: subject erasure failed",
			"subject_uuid", subjectUUID,
			"err", err,
		)
		return
	}
	if deleted > 0 {
		slog.Info("retention: subject erased from outbox",
			"subject_uuid", subjectUUID,
			"deleted", deleted,
		)
	}
}

// Shutdown stops the retention loop gracefully.
func (r *RetentionRunner) Shutdown() {
	close(r.stopCh)
	r.wg.Wait()
	slog.Info("retention runner: stopped")
}
