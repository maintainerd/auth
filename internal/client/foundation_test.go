package client

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/stretchr/testify/assert"
)

type mockAuthEventService struct{}

func (m *mockAuthEventService) Log(_ context.Context, _ authevent.AuthEventInput) {}
func (m *mockAuthEventService) FindPaginated(_ context.Context, _ authevent.AuthEventRepositoryGetFilter) (*authevent.PaginationResult[authevent.AuthEventServiceDataResult], error) {
	return nil, nil
}
func (m *mockAuthEventService) FindByUUID(_ context.Context, _ int64, _ uuid.UUID) (*authevent.AuthEventServiceDataResult, error) {
	return nil, nil
}
func (m *mockAuthEventService) CountByEventType(_ context.Context, _ string, _ int64) (int64, error) {
	return 0, nil
}
func (m *mockAuthEventService) DeleteOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}
func (m *mockAuthEventService) Shutdown() {}

func TestCoalesceAuthEventService(t *testing.T) {
	t.Run("returns provided service when not nil", func(t *testing.T) {
		svc := &mockAuthEventService{}
		result := coalesceAuthEventService(svc)
		assert.Same(t, svc, result)
	})
	t.Run("returns NoopService when nil", func(t *testing.T) {
		result := coalesceAuthEventService(nil)
		assert.NotNil(t, result)
	})
}
