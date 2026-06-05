package webhook

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
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
	createFn        func(ctx context.Context, tenantID int64, url, secret string, events []string, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error)
	updateFn        func(ctx context.Context, tenantID int64, weUUID uuid.UUID, url, secret string, events []string, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error)
	updateStatusFn  func(ctx context.Context, tenantID int64, weUUID uuid.UUID, status string) (*WebhookEndpointServiceDataResult, error)
	deleteFn        func(ctx context.Context, tenantID int64, weUUID uuid.UUID) (*WebhookEndpointServiceDataResult, error)
}

func (m *testWebhookEndpointService) GetAll(ctx context.Context, tenantID int64, status []string, page, limit int, sortBy, sortOrder string) (*WebhookEndpointServiceListResult, error) {
	return m.getAllFn(ctx, tenantID, status, page, limit, sortBy, sortOrder)
}
func (m *testWebhookEndpointService) GetByUUID(ctx context.Context, tenantID int64, weUUID uuid.UUID) (*WebhookEndpointServiceDataResult, error) {
	return m.getByUUIDFn(ctx, tenantID, weUUID)
}
func (m *testWebhookEndpointService) Create(ctx context.Context, tenantID int64, url, secret string, events []string, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error) {
	return m.createFn(ctx, tenantID, url, secret, events, maxRetries, timeoutSeconds, description, status)
}
func (m *testWebhookEndpointService) Update(ctx context.Context, tenantID int64, weUUID uuid.UUID, url, secret string, events []string, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error) {
	return m.updateFn(ctx, tenantID, weUUID, url, secret, events, maxRetries, timeoutSeconds, description, status)
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
			createFn: func(ctx context.Context, tenantID int64, url, secret string, events []string, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error) { return &weResult, nil },
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
}
