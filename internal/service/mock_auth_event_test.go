package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/repository"
)

// mockAuthEventService is a test double for AuthEventService.
type mockAuthEventService struct {
	logFn              func(ctx context.Context, input AuthEventInput)
	findPaginatedFn    func(ctx context.Context, filter repository.AuthEventRepositoryGetFilter) (*repository.PaginationResult[AuthEventServiceDataResult], error)
	findByUUIDFn       func(ctx context.Context, tenantID int64, eventUUID uuid.UUID) (*AuthEventServiceDataResult, error)
	countByEventTypeFn func(ctx context.Context, eventType string, tenantID int64) (int64, error)
	deleteOlderThanFn  func(ctx context.Context, cutoff time.Time) (int64, error)
}

func (m *mockAuthEventService) Log(ctx context.Context, input AuthEventInput) {
	if m.logFn != nil {
		m.logFn(ctx, input)
	}
}

func (m *mockAuthEventService) FindPaginated(ctx context.Context, filter repository.AuthEventRepositoryGetFilter) (*repository.PaginationResult[AuthEventServiceDataResult], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(ctx, filter)
	}
	return &repository.PaginationResult[AuthEventServiceDataResult]{}, nil
}

func (m *mockAuthEventService) FindByUUID(ctx context.Context, tenantID int64, eventUUID uuid.UUID) (*AuthEventServiceDataResult, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(ctx, tenantID, eventUUID)
	}
	return nil, nil
}

func (m *mockAuthEventService) CountByEventType(ctx context.Context, eventType string, tenantID int64) (int64, error) {
	if m.countByEventTypeFn != nil {
		return m.countByEventTypeFn(ctx, eventType, tenantID)
	}
	return 0, nil
}

func (m *mockAuthEventService) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	if m.deleteOlderThanFn != nil {
		return m.deleteOlderThanFn(ctx, cutoff)
	}
	return 0, nil
}
