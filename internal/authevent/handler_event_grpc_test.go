package authevent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

type testAutheventTenant struct {
	getFn func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

func (m *testAutheventTenant) GetByUUID(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
	if m.getFn != nil {
		return m.getFn(ctx, tuuid)
	}
	return &TenantServiceDataResult{TenantID: 1, TenantUUID: tuuid}, nil
}

type testAuthEventService struct {
	findFn       func(ctx context.Context, filter AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error)
	findByUUIDFn func(ctx context.Context, tenantID int64, eventUUID uuid.UUID) (*AuthEventServiceDataResult, error)
	countFn      func(ctx context.Context, eventType string, tenantID int64) (int64, error)
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
func (m *testAuthEventService) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	return 0, nil
}
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
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Events) != 1 {
			t.Fatalf("expected 1, got %d", len(res.Events))
		}
	})

	t.Run("get success", func(t *testing.T) {
		svc := &testAuthEventService{
			findByUUIDFn: func(ctx context.Context, tenantID int64, id uuid.UUID) (*AuthEventServiceDataResult, error) {
				return &ev, nil
			},
		}
		h := NewAuthEventGRPCHandler(resolver, svc)
		_, err := h.GetAuthEvent(ctx, &authv1.GetAuthEventRequest{TenantUuid: tenantUUID.String(), AuthEventUuid: eventUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("count success", func(t *testing.T) {
		svc := &testAuthEventService{
			countFn: func(ctx context.Context, eventType string, tenantID int64) (int64, error) { return 42, nil },
		}
		h := NewAuthEventGRPCHandler(resolver, svc)
		res, err := h.CountAuthEventsByType(ctx, &authv1.CountAuthEventsByTypeRequest{TenantUuid: tenantUUID.String(), EventType: "login"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Count != 42 {
			t.Errorf("expected 42, got %d", res.Count)
		}
	})

	t.Run("service error", func(t *testing.T) {
		svc := &testAuthEventService{
			findFn: func(ctx context.Context, filter AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error) {
				return nil, errors.New("db")
			},
		}
		h := NewAuthEventGRPCHandler(resolver, svc)
		_, err := h.ListAuthEvents(ctx, &authv1.ListAuthEventsRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("list nil pagination", func(t *testing.T) {
		svc := &testAuthEventService{
			findFn: func(ctx context.Context, filter AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error) {
				return &PaginationResult[AuthEventServiceDataResult]{Data: []AuthEventServiceDataResult{ev}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewAuthEventGRPCHandler(resolver, svc)
		res, err := h.ListAuthEvents(ctx, &authv1.ListAuthEventsRequest{TenantUuid: tenantUUID.String(), Pagination: nil})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Events) != 1 {
			t.Fatalf("expected 1, got %d", len(res.Events))
		}
	})

	t.Run("list tenant resolver error", func(t *testing.T) {
		errResolver := &testAutheventTenant{getFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant err")
		}}
		svc := &testAuthEventService{}
		h := NewAuthEventGRPCHandler(errResolver, svc)
		_, err := h.ListAuthEvents(ctx, &authv1.ListAuthEventsRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("get invalid auth event UUID", func(t *testing.T) {
		svc := &testAuthEventService{}
		h := NewAuthEventGRPCHandler(resolver, svc)
		_, err := h.GetAuthEvent(ctx, &authv1.GetAuthEventRequest{TenantUuid: tenantUUID.String(), AuthEventUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("get service error", func(t *testing.T) {
		svc := &testAuthEventService{
			findByUUIDFn: func(ctx context.Context, tenantID int64, id uuid.UUID) (*AuthEventServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewAuthEventGRPCHandler(resolver, svc)
		_, err := h.GetAuthEvent(ctx, &authv1.GetAuthEventRequest{TenantUuid: tenantUUID.String(), AuthEventUuid: eventUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("count empty tenant UUID", func(t *testing.T) {
		svc := &testAuthEventService{}
		h := NewAuthEventGRPCHandler(resolver, svc)
		_, err := h.CountAuthEventsByType(ctx, &authv1.CountAuthEventsByTypeRequest{TenantUuid: "", EventType: "login"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("count service error", func(t *testing.T) {
		svc := &testAuthEventService{
			countFn: func(ctx context.Context, eventType string, tenantID int64) (int64, error) { return 0, errors.New("db") },
		}
		h := NewAuthEventGRPCHandler(resolver, svc)
		_, err := h.CountAuthEventsByType(ctx, &authv1.CountAuthEventsByTypeRequest{TenantUuid: tenantUUID.String(), EventType: "login"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("pagintationDTO nil", func(t *testing.T) {
		dto := paginationDTO(nil)
		if dto.Page != 1 || dto.Limit != pagination.DefaultPageSize {
			t.Errorf("expected defaults, got Page=%d Limit=%d", dto.Page, dto.Limit)
		}
	})

	t.Run("pageProto zero", func(t *testing.T) {
		proto := paginationDTO(&authv1.Pagination{Page: 0, Limit: 0})
		if proto.Page != 1 {
			t.Errorf("expected page=1, got %d", proto.Page)
		}
		if proto.Limit != pagination.DefaultPageSize {
			t.Errorf("expected default limit, got %d", proto.Limit)
		}
	})

	t.Run("eventProto nil", func(t *testing.T) {
		v := eventProto(nil)
		if v != nil {
			t.Error("expected nil")
		}
	})

	t.Run("parseUUID empty", func(t *testing.T) {
		_, err := parseUUID("", "test")
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("eventProto with metadata", func(t *testing.T) {
		r := &AuthEventServiceDataResult{AuthEventUUID: eventUUID, Metadata: datatypes.JSON(`{"key":"val"}`), CreatedAt: time.Now()}
		v := eventProto(r)
		if v == nil {
			t.Fatal("expected non-nil")
		}
		if v.Metadata == nil {
			t.Error("expected non-nil metadata")
		}
	})

	t.Run("strPtr non-empty", func(t *testing.T) {
		v := strPtr("hello")
		if v == nil || *v != "hello" {
			t.Error("expected non-nil")
		}
	})

	t.Run("get tenant resolver error", func(t *testing.T) {
		errResolver := &testAutheventTenant{getFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		svc := &testAuthEventService{}
		h := NewAuthEventGRPCHandler(errResolver, svc)
		_, err := h.GetAuthEvent(ctx, &authv1.GetAuthEventRequest{TenantUuid: tenantUUID.String(), AuthEventUuid: eventUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})
}
