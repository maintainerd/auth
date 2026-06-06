package webhook

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testWebhookTenant struct {
	getFn func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

func (m *testWebhookTenant) GetByUUID(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
	if m.getFn != nil { return m.getFn(ctx, tuuid) }
	return &TenantServiceDataResult{TenantID: 1, TenantUUID: tuuid}, nil
}

type testWebhookEndpointService struct {
	getAllFn        func(ctx context.Context, tenantID int64, status []string, page, limit int, sortBy, sortOrder string) (*WebhookEndpointServiceListResult, error)
	getByUUIDFn     func(ctx context.Context, tenantID int64, weUUID uuid.UUID) (*WebhookEndpointServiceDataResult, error)
	createFn        func(ctx context.Context, tenantID int64, url string, subscribeAll bool, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error)
	updateFn        func(ctx context.Context, tenantID int64, weUUID uuid.UUID, url string, rotateSecret bool, subscribeAll bool, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error)
	updateStatusFn  func(ctx context.Context, tenantID int64, weUUID uuid.UUID, status string) (*WebhookEndpointServiceDataResult, error)
	deleteFn        func(ctx context.Context, tenantID int64, weUUID uuid.UUID) (*WebhookEndpointServiceDataResult, error)
}

func (m *testWebhookEndpointService) GetAll(ctx context.Context, tenantID int64, status []string, page, limit int, sortBy, sortOrder string) (*WebhookEndpointServiceListResult, error) {
	return m.getAllFn(ctx, tenantID, status, page, limit, sortBy, sortOrder)
}
func (m *testWebhookEndpointService) GetByUUID(ctx context.Context, tenantID int64, weUUID uuid.UUID) (*WebhookEndpointServiceDataResult, error) {
	return m.getByUUIDFn(ctx, tenantID, weUUID)
}
func (m *testWebhookEndpointService) Create(ctx context.Context, tenantID int64, url string, subscribeAll bool, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error) {
	return m.createFn(ctx, tenantID, url, subscribeAll, maxRetries, timeoutSeconds, description, status)
}
func (m *testWebhookEndpointService) Update(ctx context.Context, tenantID int64, weUUID uuid.UUID, url string, rotateSecret bool, subscribeAll bool, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error) {
	return m.updateFn(ctx, tenantID, weUUID, url, rotateSecret, subscribeAll, maxRetries, timeoutSeconds, description, status)
}
func (m *testWebhookEndpointService) UpdateStatus(ctx context.Context, tenantID int64, weUUID uuid.UUID, status string) (*WebhookEndpointServiceDataResult, error) {
	return m.updateStatusFn(ctx, tenantID, weUUID, status)
}
func (m *testWebhookEndpointService) Delete(ctx context.Context, tenantID int64, weUUID uuid.UUID) (*WebhookEndpointServiceDataResult, error) {
	return m.deleteFn(ctx, tenantID, weUUID)
}

func TestWebhookEndpointGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	weUUID := uuid.New()
	resolver := &testWebhookTenant{}
	weResult := WebhookEndpointServiceDataResult{WebhookEndpointUUID: weUUID, URL: "https://example.com/hook", Description: "Test", Status: "active"}

	t.Run("list success", func(t *testing.T) {
		svc := &testWebhookEndpointService{
			getAllFn: func(ctx context.Context, tenantID int64, status []string, page, limit int, sortBy, sortOrder string) (*WebhookEndpointServiceListResult, error) {
				return &WebhookEndpointServiceListResult{Data: []WebhookEndpointServiceDataResult{weResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewWebhookEndpointGRPCHandler(resolver, svc)
		res, err := h.ListWebhookEndpoints(ctx, &authv1.ListWebhookEndpointsRequest{TenantUuid: tenantUUID.String()})
		if err != nil { t.Fatal(err) }
		if len(res.Endpoints) != 1 { t.Fatalf("expected 1, got %d", len(res.Endpoints)) }
	})

	t.Run("get success", func(t *testing.T) {
		svc := &testWebhookEndpointService{
			getByUUIDFn: func(ctx context.Context, tenantID int64, id uuid.UUID) (*WebhookEndpointServiceDataResult, error) { return &weResult, nil },
		}
		h := NewWebhookEndpointGRPCHandler(resolver, svc)
		_, err := h.GetWebhookEndpoint(ctx, &authv1.GetWebhookEndpointRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: weUUID.String()})
		if err != nil { t.Fatal(err) }
	})

	t.Run("create success", func(t *testing.T) {
		svc := &testWebhookEndpointService{
			createFn: func(ctx context.Context, tenantID int64, url string, subscribeAll bool, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error) { return &weResult, nil },
		}
		h := NewWebhookEndpointGRPCHandler(resolver, svc)
		_, err := h.CreateWebhookEndpoint(ctx, &authv1.CreateWebhookEndpointRequest{TenantUuid: tenantUUID.String(), Url: "https://h", Description: "Test", Status: "active"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("service error", func(t *testing.T) {
		svc := &testWebhookEndpointService{
			getAllFn: func(ctx context.Context, tenantID int64, status []string, page, limit int, sortBy, sortOrder string) (*WebhookEndpointServiceListResult, error) { return nil, errors.New("db") },
		}
		h := NewWebhookEndpointGRPCHandler(resolver, svc)
		_, err := h.ListWebhookEndpoints(ctx, &authv1.ListWebhookEndpointsRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("update success", func(t *testing.T) {
		svc := &testWebhookEndpointService{
			updateFn: func(ctx context.Context, tenantID int64, weUUID uuid.UUID, url string, rotateSecret bool, subscribeAll bool, maxRetries, timeoutSeconds *int, description, rStatus string) (*WebhookEndpointServiceDataResult, error) { return &weResult, nil },
		}
		h := NewWebhookEndpointGRPCHandler(resolver, svc)
		_, err := h.UpdateWebhookEndpoint(ctx, &authv1.UpdateWebhookEndpointRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: weUUID.String(), Url: "https://h"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("update service error", func(t *testing.T) {
		svc := &testWebhookEndpointService{
			updateFn: func(ctx context.Context, tenantID int64, weUUID uuid.UUID, url string, rotateSecret bool, subscribeAll bool, maxRetries, timeoutSeconds *int, description, rStatus string) (*WebhookEndpointServiceDataResult, error) { return nil, errors.New("db") },
		}
		h := NewWebhookEndpointGRPCHandler(resolver, svc)
		_, err := h.UpdateWebhookEndpoint(ctx, &authv1.UpdateWebhookEndpointRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: weUUID.String(), Url: "https://h"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("setStatus success", func(t *testing.T) {
		svc := &testWebhookEndpointService{
			updateStatusFn: func(ctx context.Context, tenantID int64, weUUID uuid.UUID, rStatus string) (*WebhookEndpointServiceDataResult, error) { return &weResult, nil },
		}
		h := NewWebhookEndpointGRPCHandler(resolver, svc)
		_, err := h.SetWebhookEndpointStatus(ctx, &authv1.SetWebhookEndpointStatusRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: weUUID.String(), Status: "inactive"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("setStatus service error", func(t *testing.T) {
		svc := &testWebhookEndpointService{
			updateStatusFn: func(ctx context.Context, tenantID int64, weUUID uuid.UUID, rStatus string) (*WebhookEndpointServiceDataResult, error) { return nil, errors.New("db") },
		}
		h := NewWebhookEndpointGRPCHandler(resolver, svc)
		_, err := h.SetWebhookEndpointStatus(ctx, &authv1.SetWebhookEndpointStatusRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: weUUID.String(), Status: "inactive"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("delete success", func(t *testing.T) {
		svc := &testWebhookEndpointService{
			deleteFn: func(ctx context.Context, tenantID int64, weUUID uuid.UUID) (*WebhookEndpointServiceDataResult, error) { return &weResult, nil },
		}
		h := NewWebhookEndpointGRPCHandler(resolver, svc)
		_, err := h.DeleteWebhookEndpoint(ctx, &authv1.DeleteWebhookEndpointRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: weUUID.String()})
		if err != nil { t.Fatal(err) }
	})

	t.Run("delete service error", func(t *testing.T) {
		svc := &testWebhookEndpointService{
			deleteFn: func(ctx context.Context, tenantID int64, weUUID uuid.UUID) (*WebhookEndpointServiceDataResult, error) { return nil, errors.New("db") },
		}
		h := NewWebhookEndpointGRPCHandler(resolver, svc)
		_, err := h.DeleteWebhookEndpoint(ctx, &authv1.DeleteWebhookEndpointRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: weUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("get service error", func(t *testing.T) {
		svc := &testWebhookEndpointService{
			getByUUIDFn: func(ctx context.Context, tenantID int64, id uuid.UUID) (*WebhookEndpointServiceDataResult, error) { return nil, errors.New("db") },
		}
		h := NewWebhookEndpointGRPCHandler(resolver, svc)
		_, err := h.GetWebhookEndpoint(ctx, &authv1.GetWebhookEndpointRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: weUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("create service error", func(t *testing.T) {
		svc := &testWebhookEndpointService{
			createFn: func(ctx context.Context, tenantID int64, url string, subscribeAll bool, maxRetries, timeoutSeconds *int, description, rStatus string) (*WebhookEndpointServiceDataResult, error) { return nil, errors.New("db") },
		}
		h := NewWebhookEndpointGRPCHandler(resolver, svc)
		_, err := h.CreateWebhookEndpoint(ctx, &authv1.CreateWebhookEndpointRequest{TenantUuid: tenantUUID.String(), Url: "https://h"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("tenant invalid UUID", func(t *testing.T) {
		svc := &testWebhookEndpointService{}
		h := NewWebhookEndpointGRPCHandler(resolver, svc)
		_, err := h.ListWebhookEndpoints(ctx, &authv1.ListWebhookEndpointsRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("tenant empty UUID", func(t *testing.T) {
		svc := &testWebhookEndpointService{}
		h := NewWebhookEndpointGRPCHandler(resolver, svc)
		_, err := h.ListWebhookEndpoints(ctx, &authv1.ListWebhookEndpointsRequest{TenantUuid: ""})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("tenant resolver error", func(t *testing.T) {
		errResolver := &testWebhookTenant{getFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) { return nil, errors.New("tenant") }}
		svc := &testWebhookEndpointService{}
		h := NewWebhookEndpointGRPCHandler(errResolver, svc)
		_, err := h.ListWebhookEndpoints(ctx, &authv1.ListWebhookEndpointsRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("parseUUID empty", func(t *testing.T) {
		_, err := parseUUID("", "test")
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("paginationDTO nil", func(t *testing.T) {
		dto := paginationDTO(nil)
		if dto.Page != 1 || dto.Limit != pagination.DefaultPageSize { t.Errorf("expected defaults, got %+v", dto) }
	})

	t.Run("intPtr nil", func(t *testing.T) {
		v := intPtr(nil)
		if v != nil { t.Error("expected nil") }
	})

	t.Run("webhookProto nil", func(t *testing.T) {
		v := webhookProto(nil)
		if v != nil { t.Error("expected nil") }
	})

	t.Run("webhookProto with LastTriggeredAt", func(t *testing.T) {
		now := time.Now()
		r := &WebhookEndpointServiceDataResult{WebhookEndpointUUID: weUUID, URL: "https://x", LastTriggeredAt: &now, CreatedAt: now, UpdatedAt: now}
		v := webhookProto(r)
		if v == nil { t.Error("expected non-nil") }
		if v.LastTriggeredAt == nil { t.Error("expected non-nil LastTriggeredAt") }
	})

	t.Run("get tenant resolver error", func(t *testing.T) {
		errResolver := &testWebhookTenant{getFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) { return nil, errors.New("tenant") }}
		h := NewWebhookEndpointGRPCHandler(errResolver, &testWebhookEndpointService{})
		_, err := h.GetWebhookEndpoint(ctx, &authv1.GetWebhookEndpointRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: weUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("create tenant resolver error", func(t *testing.T) {
		errResolver := &testWebhookTenant{getFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) { return nil, errors.New("tenant") }}
		h := NewWebhookEndpointGRPCHandler(errResolver, &testWebhookEndpointService{})
		_, err := h.CreateWebhookEndpoint(ctx, &authv1.CreateWebhookEndpointRequest{TenantUuid: tenantUUID.String(), Url: "https://h"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("update tenant resolver error", func(t *testing.T) {
		errResolver := &testWebhookTenant{getFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) { return nil, errors.New("tenant") }}
		h := NewWebhookEndpointGRPCHandler(errResolver, &testWebhookEndpointService{})
		_, err := h.UpdateWebhookEndpoint(ctx, &authv1.UpdateWebhookEndpointRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: weUUID.String(), Url: "https://h"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("setStatus tenant resolver error", func(t *testing.T) {
		errResolver := &testWebhookTenant{getFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) { return nil, errors.New("tenant") }}
		h := NewWebhookEndpointGRPCHandler(errResolver, &testWebhookEndpointService{})
		_, err := h.SetWebhookEndpointStatus(ctx, &authv1.SetWebhookEndpointStatusRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: weUUID.String(), Status: "inactive"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("delete tenant resolver error", func(t *testing.T) {
		errResolver := &testWebhookTenant{getFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) { return nil, errors.New("tenant") }}
		h := NewWebhookEndpointGRPCHandler(errResolver, &testWebhookEndpointService{})
		_, err := h.DeleteWebhookEndpoint(ctx, &authv1.DeleteWebhookEndpointRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: weUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("invalid webhook UUID update", func(t *testing.T) {
		h := NewWebhookEndpointGRPCHandler(resolver, &testWebhookEndpointService{})
		_, err := h.UpdateWebhookEndpoint(ctx, &authv1.UpdateWebhookEndpointRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: "bad", Url: "https://h"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("invalid webhook UUID setStatus", func(t *testing.T) {
		h := NewWebhookEndpointGRPCHandler(resolver, &testWebhookEndpointService{})
		_, err := h.SetWebhookEndpointStatus(ctx, &authv1.SetWebhookEndpointStatusRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: "bad", Status: "inactive"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("invalid webhook UUID delete", func(t *testing.T) {
		h := NewWebhookEndpointGRPCHandler(resolver, &testWebhookEndpointService{})
		_, err := h.DeleteWebhookEndpoint(ctx, &authv1.DeleteWebhookEndpointRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("paginationDTO zero", func(t *testing.T) {
		dto := paginationDTO(&authv1.Pagination{Page: 0, Limit: 0})
		if dto.Page != 1 || dto.Limit != pagination.DefaultPageSize { t.Errorf("expected defaults, got %+v", dto) }
	})

	t.Run("intPtr non-nil", func(t *testing.T) {
		v32 := int32(5)
		v := intPtr(&v32)
		if v == nil || *v != 5 { t.Error("expected non-nil with value 5") }
	})

	t.Run("get webhook tenant resolver error", func(t *testing.T) {
		errResolver := &testWebhookTenant{getFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) { return nil, errors.New("tenant") }}
		h := NewWebhookEndpointGRPCHandler(errResolver, &testWebhookEndpointService{})
		_, err := h.GetWebhookEndpoint(ctx, &authv1.GetWebhookEndpointRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: weUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("get invalid webhook UUID", func(t *testing.T) {
		h := NewWebhookEndpointGRPCHandler(resolver, &testWebhookEndpointService{})
		_, err := h.GetWebhookEndpoint(ctx, &authv1.GetWebhookEndpointRequest{TenantUuid: tenantUUID.String(), WebhookEndpointUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})
}
