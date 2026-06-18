package authevent

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authctx"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/middleware"
)

const testTenantID int64 = 1

var (
	errNotFound      = apperror.NewNotFoundWithReason("not found")
	testTenantUUID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testResourceUUID = uuid.MustParse("00000000-0000-0000-0000-000000000099")
)

type mockAuthEventService struct {
	logFn              func(ctx context.Context, input AuthEventInput)
	findPaginatedFn    func(ctx context.Context, filter AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error)
	findByUUIDFn       func(ctx context.Context, tenantID int64, eventUUID uuid.UUID) (*AuthEventServiceDataResult, error)
	countByEventTypeFn func(ctx context.Context, eventType string, tenantID int64) (int64, error)
	deleteOlderThanFn  func(ctx context.Context, cutoff time.Time) (int64, error)
	exportFn           func(ctx context.Context, filter AuthEventRepositoryGetFilter, format string) (*AuthEventExport, error)
}

func (m *mockAuthEventService) Log(ctx context.Context, input AuthEventInput) {
	if m.logFn != nil {
		m.logFn(ctx, input)
	}
}

func (m *mockAuthEventService) FindPaginated(ctx context.Context, filter AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(ctx, filter)
	}
	return &PaginationResult[AuthEventServiceDataResult]{}, nil
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

func (m *mockAuthEventService) Export(ctx context.Context, filter AuthEventRepositoryGetFilter, format string) (*AuthEventExport, error) {
	if m.exportFn != nil {
		return m.exportFn(ctx, filter, format)
	}
	return &AuthEventExport{Format: "json", ContentType: "application/json", Filename: "auth-events.json", Data: []byte("[]")}, nil
}

func (m *mockAuthEventService) Shutdown() {}

func withTenant(r *http.Request) *http.Request {
	tenant := &authctx.AuthTenant{TenantID: testTenantID, TenantUUID: testTenantUUID}
	return middleware.WithAuthContext(r, &authctx.AuthContext{Tenant: tenant})
}

func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		rctx = chi.NewRouteContext()
	}
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}
