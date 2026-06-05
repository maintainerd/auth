package idp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

type testIDPService struct {
	getFn            func(ctx context.Context, filter IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error)
	getByUUIDFn      func(ctx context.Context, idpUUID uuid.UUID, tenantID int64) (*IdentityProviderServiceDataResult, error)
	createFn         func(ctx context.Context, name, displayName, provider, providerType string, config datatypes.JSON, pStatus string, tenantUUID string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
	updateFn         func(ctx context.Context, idpUUID uuid.UUID, name, displayName, provider, providerType string, config datatypes.JSON, pStatus string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
	setStatusByUUIDFn func(ctx context.Context, idpUUID uuid.UUID, pStatus string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
	deleteByUUIDFn    func(ctx context.Context, idpUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
}

func (m *testIDPService) Get(ctx context.Context, filter IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error) {
	return m.getFn(ctx, filter)
}
func (m *testIDPService) GetByUUID(ctx context.Context, idpUUID uuid.UUID, tenantID int64) (*IdentityProviderServiceDataResult, error) {
	return m.getByUUIDFn(ctx, idpUUID, tenantID)
}
func (m *testIDPService) Create(ctx context.Context, name, displayName, provider, providerType string, config datatypes.JSON, pStatus string, tenantUUID string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	return m.createFn(ctx, name, displayName, provider, providerType, config, pStatus, tenantUUID, tenantID, actorUserUUID)
}
func (m *testIDPService) Update(ctx context.Context, idpUUID uuid.UUID, name, displayName, provider, providerType string, config datatypes.JSON, pStatus string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	return m.updateFn(ctx, idpUUID, name, displayName, provider, providerType, config, pStatus, tenantID, actorUserUUID)
}
func (m *testIDPService) SetStatusByUUID(ctx context.Context, idpUUID uuid.UUID, pStatus string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	return m.setStatusByUUIDFn(ctx, idpUUID, pStatus, tenantID, actorUserUUID)
}
func (m *testIDPService) DeleteByUUID(ctx context.Context, idpUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	return m.deleteByUUIDFn(ctx, idpUUID, tenantID, actorUserUUID)
}

type mockIDPTenantResolver struct {
	getByUUIDFn func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

func (m *mockIDPTenantResolver) GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(ctx, tenantUUID)
	}
	return &TenantServiceDataResult{TenantID: 1, TenantUUID: tenantUUID}, nil
}

func TestIdentityProviderGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	idpUUID := uuid.New()
	actorUUID := uuid.New()
	now := time.Now()
	resolver := &mockIDPTenantResolver{getByUUIDFn: func(ctx context.Context, id uuid.UUID) (*TenantServiceDataResult, error) {
		return &TenantServiceDataResult{TenantID: 1, TenantUUID: id}, nil
	}}
	idpResult := IdentityProviderServiceDataResult{
		IdentityProviderUUID: idpUUID,
		Name:                 "google",
		DisplayName:          "Google",
		Provider:             "google",
		ProviderType:         "social",
		Identifier:           "google-oidc",
		Status:               "active",
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	t.Run("list success", func(t *testing.T) {
		svc := &testIDPService{
			getFn: func(ctx context.Context, filter IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error) {
				return &IdentityProviderServiceGetResult{Data: []IdentityProviderServiceDataResult{idpResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewIdentityProviderGRPCHandler(resolver, svc)
		res, err := h.ListIdentityProviders(ctx, &authv1.ListIdentityProvidersRequest{TenantUuid: tenantUUID.String(), Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.IdentityProviders) != 1 {
			t.Fatalf("expected 1 provider, got %d", len(res.IdentityProviders))
		}
		if res.IdentityProviders[0].Name != "google" {
			t.Errorf("expected name google, got %s", res.IdentityProviders[0].Name)
		}
	})

	t.Run("get success", func(t *testing.T) {
		svc := &testIDPService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*IdentityProviderServiceDataResult, error) {
				return &idpResult, nil
			},
		}
		h := NewIdentityProviderGRPCHandler(resolver, svc)
		res, err := h.GetIdentityProvider(ctx, &authv1.GetIdentityProviderRequest{TenantUuid: tenantUUID.String(), IdentityProviderUuid: idpUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.IdentityProvider.Name != "google" {
			t.Errorf("expected name google, got %s", res.IdentityProvider.Name)
		}
	})

	t.Run("create success", func(t *testing.T) {
		svc := &testIDPService{
			createFn: func(ctx context.Context, name, displayName, provider, providerType string, config datatypes.JSON, pStatus string, tUUID string, tenantID int64, actorUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
				return &idpResult, nil
			},
		}
		h := NewIdentityProviderGRPCHandler(resolver, svc)
		res, err := h.CreateIdentityProvider(ctx, &authv1.CreateIdentityProviderRequest{
			TenantUuid:    tenantUUID.String(),
			Name:          "google",
			DisplayName:   "Google",
			Provider:      "google",
			ProviderType:  "social",
			Status:        "active",
			ActorUserUuid: actorUUID.String(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.IdentityProvider.DisplayName != "Google" {
			t.Errorf("expected DisplayName Google, got %s", res.IdentityProvider.DisplayName)
		}
	})

	t.Run("update success", func(t *testing.T) {
		svc := &testIDPService{
			updateFn: func(ctx context.Context, id uuid.UUID, name, displayName, provider, providerType string, config datatypes.JSON, pStatus string, tenantID int64, actorUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
				return &idpResult, nil
			},
		}
		h := NewIdentityProviderGRPCHandler(resolver, svc)
		res, err := h.UpdateIdentityProvider(ctx, &authv1.UpdateIdentityProviderRequest{
			TenantUuid:            tenantUUID.String(),
			IdentityProviderUuid:  idpUUID.String(),
			Name:                  "google",
			DisplayName:           "Google",
			Provider:              "google",
			ProviderType:          "social",
			Status:                "active",
			ActorUserUuid:         actorUUID.String(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.IdentityProvider.Provider != "google" {
			t.Errorf("expected provider google, got %s", res.IdentityProvider.Provider)
		}
	})

	t.Run("setStatus success", func(t *testing.T) {
		svc := &testIDPService{
			setStatusByUUIDFn: func(ctx context.Context, id uuid.UUID, pStatus string, tenantID int64, actorUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
				return &idpResult, nil
			},
		}
		h := NewIdentityProviderGRPCHandler(resolver, svc)
		res, err := h.SetIdentityProviderStatus(ctx, &authv1.SetIdentityProviderStatusRequest{
			TenantUuid:           tenantUUID.String(),
			IdentityProviderUuid: idpUUID.String(),
			Status:               "active",
			ActorUserUuid:        actorUUID.String(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.IdentityProvider.Status != "active" {
			t.Errorf("expected status active, got %s", res.IdentityProvider.Status)
		}
	})

	t.Run("delete success", func(t *testing.T) {
		svc := &testIDPService{
			deleteByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64, actorUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
				return &idpResult, nil
			},
		}
		h := NewIdentityProviderGRPCHandler(resolver, svc)
		res, err := h.DeleteIdentityProvider(ctx, &authv1.DeleteIdentityProviderRequest{
			TenantUuid:           tenantUUID.String(),
			IdentityProviderUuid: idpUUID.String(),
			ActorUserUuid:        actorUUID.String(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.IdentityProvider.Name != "google" {
			t.Errorf("expected name google, got %s", res.IdentityProvider.Name)
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		svc := &testIDPService{
			getFn: func(ctx context.Context, filter IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error) {
				return &IdentityProviderServiceGetResult{}, nil
			},
		}
		h := NewIdentityProviderGRPCHandler(resolver, svc)
		_, err := h.ListIdentityProviders(ctx, &authv1.ListIdentityProvidersRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
		_, err = h.GetIdentityProvider(ctx, &authv1.GetIdentityProviderRequest{TenantUuid: tenantUUID.String(), IdentityProviderUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("service errors", func(t *testing.T) {
		svcErr := errors.New("db error")
		svc := &testIDPService{
			getFn: func(ctx context.Context, filter IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error) {
				return nil, svcErr
			},
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*IdentityProviderServiceDataResult, error) {
				return nil, svcErr
			},
			createFn: func(ctx context.Context, name, displayName, provider, providerType string, config datatypes.JSON, pStatus string, tUUID string, tenantID int64, actorUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
				return nil, svcErr
			},
			updateFn: func(ctx context.Context, id uuid.UUID, name, displayName, provider, providerType string, config datatypes.JSON, pStatus string, tenantID int64, actorUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
				return nil, svcErr
			},
			setStatusByUUIDFn: func(ctx context.Context, id uuid.UUID, pStatus string, tenantID int64, actorUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
				return nil, svcErr
			},
			deleteByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64, actorUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
				return nil, svcErr
			},
		}
		h := NewIdentityProviderGRPCHandler(resolver, svc)
		_, err := h.ListIdentityProviders(ctx, &authv1.ListIdentityProvidersRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
		_, err = h.GetIdentityProvider(ctx, &authv1.GetIdentityProviderRequest{TenantUuid: tenantUUID.String(), IdentityProviderUuid: idpUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
		_, err = h.CreateIdentityProvider(ctx, &authv1.CreateIdentityProviderRequest{TenantUuid: tenantUUID.String(), Name: "google", DisplayName: "Google", Provider: "google", ProviderType: "social", Status: "active"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
		_, err = h.UpdateIdentityProvider(ctx, &authv1.UpdateIdentityProviderRequest{TenantUuid: tenantUUID.String(), IdentityProviderUuid: idpUUID.String(), Name: "google", DisplayName: "Google", Provider: "google", ProviderType: "social", Status: "active"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
		_, err = h.SetIdentityProviderStatus(ctx, &authv1.SetIdentityProviderStatusRequest{TenantUuid: tenantUUID.String(), IdentityProviderUuid: idpUUID.String(), Status: "active"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
		_, err = h.DeleteIdentityProvider(ctx, &authv1.DeleteIdentityProviderRequest{TenantUuid: tenantUUID.String(), IdentityProviderUuid: idpUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("tenant not found", func(t *testing.T) {
		badResolver := &mockIDPTenantResolver{getByUUIDFn: func(ctx context.Context, id uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, apperror.NewNotFound("tenant not found")
		}}
		svc := &testIDPService{}
		h := NewIdentityProviderGRPCHandler(badResolver, svc)
		_, err := h.ListIdentityProviders(ctx, &authv1.ListIdentityProvidersRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.NotFound {
			t.Errorf("expected NotFound, got %v", code)
		}
	})
}
