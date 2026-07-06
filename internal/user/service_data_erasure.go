package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	// erasureRetentionDays is the GDPR Article 17(3) maximum: a request is
	// scheduled 30 days out by default. Compliance teams can schedule earlier.
	erasureRetentionDays = 30
	// erasureBatchSize bounds how many due requests the worker processes per tick.
	erasureBatchSize = 10
)

// UserAnonymizer is the narrow capability the erasure worker needs from the
// user service: the canonical multi-table anonymization cascade.
type UserAnonymizer interface {
	AnonymizeUser(ctx context.Context, userID int64) error
}

// DataErasureRequestResult is the service-layer view of an erasure request.
type DataErasureRequestResult struct {
	UUID        uuid.UUID
	Status      string
	Reason      string
	ScheduledAt time.Time
	CreatedAt   time.Time
}

// RequestErasureInput carries the resolved parameters for creating a request.
type RequestErasureInput struct {
	TenantID           int64
	UserID             int64
	RequestedByUserID  *int64
	RequestedByAdminID *int64
	Reason             string
}

// DataErasureService manages GDPR Article 17 erasure requests: it records the
// request lifecycle and drives the background anonymization worker.
type DataErasureService interface {
	// RequestErasure records an erasure request for a user, scheduled 30 days
	// out per GDPR Art.17(3). If a pending/in-progress request already exists for
	// the user it is returned unchanged (idempotent).
	RequestErasure(ctx context.Context, in RequestErasureInput) (*DataErasureRequestResult, error)
	// ProcessPendingErasureRequests anonymizes users whose scheduled_at has
	// passed and that are not under legal hold. Invoked by the background worker.
	ProcessPendingErasureRequests(ctx context.Context) error
}

type dataErasureService struct {
	repo       DataErasureRequestRepository
	anonymizer UserAnonymizer
}

// NewDataErasureService creates a new DataErasureService.
func NewDataErasureService(repo DataErasureRequestRepository, anonymizer UserAnonymizer) DataErasureService {
	return &dataErasureService{repo: repo, anonymizer: anonymizer}
}

func toErasureResult(r *DataErasureRequest) *DataErasureRequestResult {
	return &DataErasureRequestResult{
		UUID:        r.DataErasureRequestUUID,
		Status:      r.Status,
		Reason:      r.Reason,
		ScheduledAt: r.ScheduledAt,
		CreatedAt:   r.CreatedAt,
	}
}

// RequestErasure implements DataErasureService.
func (s *dataErasureService) RequestErasure(ctx context.Context, in RequestErasureInput) (*DataErasureRequestResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "dataErasure.request")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", in.TenantID), attribute.Int64("user.id", in.UserID))

	// Idempotency: an existing pending/in-progress request is returned as-is.
	existing, err := s.repo.FindActiveByUserID(in.UserID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "lookup failed")
		return nil, apperror.NewInternal("failed to check existing erasure requests", err)
	}
	if existing != nil {
		span.SetStatus(codes.Ok, "existing")
		return toErasureResult(existing), nil
	}

	req := &DataErasureRequest{
		TenantID:           in.TenantID,
		UserID:             in.UserID,
		RequestedByUserID:  in.RequestedByUserID,
		RequestedByAdminID: in.RequestedByAdminID,
		Status:             "pending",
		Reason:             in.Reason,
		ScheduledAt:        time.Now().Add(erasureRetentionDays * 24 * time.Hour),
	}
	created, err := s.repo.Create(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toErasureResult(created), nil
}

// ProcessPendingErasureRequests implements DataErasureService. It is a named job
// in the background worker, distinct from the DELETE-expired cleanup jobs
// because erasure is a complex multi-table anonymization, not a simple DELETE.
func (s *dataErasureService) ProcessPendingErasureRequests(ctx context.Context) error {
	now := time.Now()
	due, err := s.repo.FindDueForProcessing(now, erasureBatchSize)
	if err != nil {
		return err
	}

	for i := range due {
		req := due[i]
		if e := s.repo.MarkInProgress(req.DataErasureRequestID, time.Now()); e != nil {
			slog.Warn("erasure worker: mark in_progress failed", "request_id", req.DataErasureRequestID, "err", e)
			continue
		}
		if e := s.anonymizer.AnonymizeUser(ctx, req.UserID); e != nil {
			slog.Error("erasure worker: anonymize failed", "request_id", req.DataErasureRequestID, "user_id", req.UserID, "err", e)
			// Revert to pending so the request is retried next tick. 'failed' is
			// not an allowed status per chk_data_erasure_requests_status.
			if re := s.repo.MarkPending(req.DataErasureRequestID); re != nil {
				slog.Warn("erasure worker: revert to pending failed", "request_id", req.DataErasureRequestID, "err", re)
			}
			continue
		}
		if e := s.repo.MarkCompleted(req.DataErasureRequestID, time.Now()); e != nil {
			slog.Warn("erasure worker: mark completed failed", "request_id", req.DataErasureRequestID, "err", e)
			continue
		}
		slog.Info("erasure worker: user anonymized", "request_id", req.DataErasureRequestID, "user_id", req.UserID)
	}
	return nil
}

// StartDataErasureWorker runs ProcessPendingErasureRequests on an interval until
// ctx is cancelled. It is a background worker distinct from the ephemeral-row
// cleanup runner.
func StartDataErasureWorker(ctx context.Context, svc DataErasureService, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := svc.ProcessPendingErasureRequests(ctx); err != nil {
				slog.Warn("erasure worker: process pending failed", "err", err)
			}
		}
	}
}
