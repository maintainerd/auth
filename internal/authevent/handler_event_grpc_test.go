package authevent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testAutheventTenant struct {
	getFn func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

func (m *testAutheventTenant) GetByUUID(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
	if m.getFn != nil { return m.getFn(ctx, tuuid) }
	return &TenantServiceDataResult{TenantID: 1, TenantUUID: tuuid}, nil
}

type testAuthEventService struct {
	findFn    func(ctx context.Context, filter AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error)
	findByUUIDFn func(ctx context.Context, tenantID int64, eventUUID uuid.UUID) (*AuthEventServiceDataResult, error)
	countFn   func(ctx context.Context, eventType string, tenantID int64) (int64, error)
}

func (m *testAuthEventService) Log(ctx context.Context, input AuthEventInput) {}
func (m *testAuthEventService) FindPaginated(ctx context.Context, filter AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error) {
	return m.findFn(ctx, filter)
}
func (m *testAuthEventService) FindByUUID(ctx context.Context, tenantID int64, eventUUID uuid.UUID) (*AuthEventServiceDataResult, error) {
	return m.findByUUIDFn(ctx, tenantID, eventUUID)
}
func (m *testAuthEventService) CountByEventType(ctx context.Context, eventType string, tenantID int64) (int64, error) {
	return m.countFn(ctx, eventType, tenantID)
}
func (m *testAuthEventService) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) { return 0, nil }
func (m *testAuthEventService) Shutdown() {}

func TestAuthEventGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	eventUUID := uuid.New()
	resolver := &testAutheventTenant{}
	ev := AuthEventServiceDataResult{AuthEventUUID: eventUUID, Category: "auth", EventType: "login", Severity: "info", Result: "success"}

	t.Run("list success", func(t *testing.T) {
		svc := &testAuthEventService{
			findFn: func(ctx context.Context, filter AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error) {
				return &PaginationResult[AuthEventServiceDataResult]{Data: []AuthEventServiceDataResult{ev}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewAuthEventGRPCHandler(resolver, svc)
		res, err := h.ListAuthEvents(ctx, &authv1.ListAuthEventsRequest{TenantUuid: tenantUUID.String()})
		if err != nil { t.Fatal(err) }
		if len(res.Events) != 1 { t.Fatalf("expected 1, got %d", len(res.Events)) }
	})

	t.Run("get success", func(t *testing.T) {
		svc := &testAuthEventService{
			findByUUIDFn: func(ctx context.Context, tenantID int64, id uuid.UUID) (*AuthEventServiceDataResult, error) { return &ev, nil },
		}
		h := NewAuthEventGRPCHandler(resolver, svc)
		_, err := h.GetAuthEvent(ctx, &authv1.GetAuthEventRequest{TenantUuid: tenantUUID.String(), AuthEventUuid: eventUUID.String()})
		if err != nil { t.Fatal(err) }
	})

	t.Run("count success", func(t *testing.T) {
		svc := &testAuthEventService{
			countFn: func(ctx context.Context, eventType string, tenantID int64) (int64, error) { return 42, nil },
		}
		h := NewAuthEventGRPCHandler(resolver, svc)
		res, err := h.CountAuthEventsByType(ctx, &authv1.CountAuthEventsByTypeRequest{TenantUuid: tenantUUID.String(), EventType: "login"})
		if err != nil { t.Fatal(err) }
		if res.Count != 42 { t.Errorf("expected 42, got %d", res.Count) }
	})

	t.Run("service error", func(t *testing.T) {
		svc := &testAuthEventService{
			findFn: func(ctx context.Context, filter AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error) { return nil, errors.New("db") },
		}
		h := NewAuthEventGRPCHandler(resolver, svc)
		_, err := h.ListAuthEvents(ctx, &authv1.ListAuthEventsRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})
}
