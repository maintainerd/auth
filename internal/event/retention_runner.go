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
	retentionInterval            = 6 * time.Hour
	outboxRetentionDays          = 7
	deliveryHistoryRetentionDays = 90
)

// DeliveryHistoryPurger purges delivery-history records by age and by subject.
// Implemented by the webhook delivery-history repository; declared here to keep
// the event package decoupled from the webhook package.
type DeliveryHistoryPurger interface {
	DeleteOlderThan(cutoff time.Time) (int64, error)
	DeleteBySubjectUUID(subjectUUID uuid.UUID) (int64, error)
}

// RetentionRunner periodically purges old events from the outbox and delivery history.
type RetentionRunner struct {
	outboxRepo  OutboxRepository
	historyRepo DeliveryHistoryPurger
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// NewRetentionRunner creates a new retention runner. historyRepo may be nil to
// disable delivery-history purging.
func NewRetentionRunner(outboxRepo OutboxRepository, historyRepo DeliveryHistoryPurger) *RetentionRunner {
	return &RetentionRunner{
		outboxRepo:  outboxRepo,
		historyRepo: historyRepo,
		stopCh:      make(chan struct{}),
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

	if r.historyRepo != nil {
		historyCutoff := time.Now().AddDate(0, 0, -deliveryHistoryRetentionDays)
		hDeleted, hErr := r.historyRepo.DeleteOlderThan(historyCutoff)
		if hErr != nil {
			slog.Error("retention: delivery history purge failed", "err", hErr)
			span.RecordError(hErr)
		} else {
			slog.Info("retention: delivery history purged", "deleted", hDeleted, "cutoff", historyCutoff)
			span.SetAttributes(attribute.Int64("delivery_history.deleted", hDeleted))
		}
	}

	span.SetStatus(codes.Ok, "")
}

// EraseSubject purges all outbox and delivery-history rows referencing the
// given subject UUID. Called when a user is hard-deleted (right-to-erasure).
func (r *RetentionRunner) EraseSubject(ctx context.Context, subjectUUID uuid.UUID) {
	// Purge delivery history first — it resolves the subject through the outbox
	// table, so the outbox rows must still exist at this point.
	if r.historyRepo != nil {
		hDeleted, hErr := r.historyRepo.DeleteBySubjectUUID(subjectUUID)
		if hErr != nil {
			slog.Error("retention: subject erasure (delivery history) failed", "subject_uuid", subjectUUID, "err", hErr)
		} else if hDeleted > 0 {
			slog.Info("retention: subject erased from delivery history", "subject_uuid", subjectUUID, "deleted", hDeleted)
		}
	}

	deleted, err := r.outboxRepo.DeleteBySubjectUUID(subjectUUID)
	if err != nil {
		slog.Error("retention: subject erasure (outbox) failed", "subject_uuid", subjectUUID, "err", err)
	} else if deleted > 0 {
		slog.Info("retention: subject erased from outbox", "subject_uuid", subjectUUID, "deleted", deleted)
	}
}

// Shutdown stops the retention loop gracefully.
func (r *RetentionRunner) Shutdown() {
	close(r.stopCh)
	r.wg.Wait()
	slog.Info("retention runner: stopped")
}
