package iam

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
)

func TestPolicyGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	tenantUUID := uuid.New()
	policyUUID := uuid.New()
	serviceUUID := uuid.New()
	document := validPolicyDocumentStruct(t)
	documentJSON := datatypes.JSON(`{"statement":[{"action":["user:read"],"effect":"allow","resource":["user:*"]}],"version":"v1"}`)
	description := "Read users"
	policy := PolicyServiceDataResult{PolicyUUID: policyUUID, Name: "user:read", Description: &description, Document: documentJSON, Version: "v1", Status: shared.StatusActive, CreatedAt: now, UpdatedAt: now}
	service := PolicyServiceServiceDataResult{ServiceUUID: serviceUUID, Name: "auth", DisplayName: "Auth Service", Description: "Authentication service", Version: "v1", Status: shared.StatusActive}
	tenantResolver := accessTenantResolver(t, tenantUUID)

	t.Run("success", func(t *testing.T) {
		svc := &mockPolicyService{
			getFn: func(f PolicyServiceGetFilter) (*PolicyServiceGetResult, error) {
				assert.Equal(t, int64(77), f.TenantID)
				assert.Equal(t, serviceUUID, *f.ServiceID)
				return &PolicyServiceGetResult{Data: []PolicyServiceDataResult{policy}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
			getByUUIDFn: func(id uuid.UUID, tenantID int64) (*PolicyServiceDataResult, error) {
				assert.Equal(t, policyUUID, id)
				return &policy, nil
			},
			getServicesByPolicyUUIDFn: func(id uuid.UUID, tenantID int64, f PolicyServiceServicesFilter) (*PolicyServiceServicesResult, error) {
				assert.Equal(t, policyUUID, id)
				return &PolicyServiceServicesResult{Data: []PolicyServiceServiceDataResult{service}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
			createFn: func(tenantID int64, name string, desc *string, doc datatypes.JSON, version string, policyStatus string, isSystem bool) (*PolicyServiceDataResult, error) {
				assert.JSONEq(t, string(documentJSON), string(doc))
				assert.False(t, isSystem)
				return &policy, nil
			},
			updateFn: func(id uuid.UUID, tenantID int64, name string, desc *string, doc datatypes.JSON, version string, policyStatus string) (*PolicyServiceDataResult, error) {
				assert.Equal(t, policyUUID, id)
				return &policy, nil
			},
			setStatusByUUIDFn: func(id uuid.UUID, tenantID int64, policyStatus string) (*PolicyServiceDataResult, error) {
				updated := policy
				updated.Status = policyStatus
				return &updated, nil
			},
			deleteByUUIDFn: func(id uuid.UUID, tenantID int64) (*PolicyServiceDataResult, error) {
				return &policy, nil
			},
		}
		h := NewPolicyGRPCHandler(tenantResolver, svc)

		list, err := h.ListPolicies(ctx, &authv1.ListPoliciesRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String(), Status: []string{shared.StatusActive}, Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		require.NoError(t, err)
		assert.Equal(t, "user:read", list.Policies[0].Name)
		got, err := h.GetPolicy(ctx, &authv1.GetPolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, "v1", got.Policy.Document.AsMap()["version"])
		services, err := h.ListPolicyServices(ctx, &authv1.ListPolicyServicesRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String(), Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		require.NoError(t, err)
		assert.Equal(t, "auth", services.Services[0].Name)
		created, err := h.CreatePolicy(ctx, validCreatePolicyRequest(tenantUUID, document))
		require.NoError(t, err)
		assert.Equal(t, "user:read", created.Policy.Name)
		updated, err := h.UpdatePolicy(ctx, validUpdatePolicyRequest(tenantUUID, policyUUID, document))
		require.NoError(t, err)
		assert.Equal(t, policyUUID.String(), updated.Policy.PolicyUuid)
		statusRes, err := h.SetPolicyStatus(ctx, &authv1.SetPolicyStatusRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String(), Status: shared.StatusInactive})
		require.NoError(t, err)
		assert.Equal(t, shared.StatusInactive, statusRes.Policy.Status)
		deleted, err := h.DeletePolicy(ctx, &authv1.DeletePolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, policyUUID.String(), deleted.Policy.PolicyUuid)
	})

	t.Run("validation and service errors", func(t *testing.T) {
		h := NewPolicyGRPCHandler(mockTenantResolver{}, &mockPolicyService{})
		_, err := h.ListPolicies(ctx, &authv1.ListPoliciesRequest{TenantUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListPolicies(ctx, &authv1.ListPoliciesRequest{TenantUuid: tenantUUID.String(), ServiceUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListPolicies(ctx, &authv1.ListPoliciesRequest{TenantUuid: tenantUUID.String(), Status: []string{"bad"}})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.GetPolicy(ctx, &authv1.GetPolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListPolicyServices(ctx, &authv1.ListPolicyServicesRequest{TenantUuid: tenantUUID.String(), PolicyUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListPolicyServices(ctx, &authv1.ListPolicyServicesRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String(), Name: string(make([]byte, 151))})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.CreatePolicy(ctx, &authv1.CreatePolicyRequest{TenantUuid: tenantUUID.String(), Name: "BAD", Document: document})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.CreatePolicy(ctx, &authv1.CreatePolicyRequest{TenantUuid: tenantUUID.String(), Name: "user:read"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantCreate := validCreatePolicyRequest(tenantUUID, document)
		badTenantCreate.TenantUuid = "bad"
		_, err = h.CreatePolicy(ctx, badTenantCreate)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdatePolicy(ctx, &authv1.UpdatePolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String(), Name: "BAD", Document: document})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdatePolicy(ctx, &authv1.UpdatePolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: "bad", Name: "user:read", Document: document})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantUpdate := validUpdatePolicyRequest(tenantUUID, policyUUID, document)
		badTenantUpdate.TenantUuid = "bad"
		_, err = h.UpdatePolicy(ctx, badTenantUpdate)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		nilDocumentUpdate := validUpdatePolicyRequest(tenantUUID, policyUUID, document)
		nilDocumentUpdate.Document = nil
		_, err = h.UpdatePolicy(ctx, nilDocumentUpdate)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetPolicyStatus(ctx, &authv1.SetPolicyStatusRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String(), Status: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetPolicyStatus(ctx, &authv1.SetPolicyStatusRequest{TenantUuid: "bad", PolicyUuid: policyUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeletePolicy(ctx, &authv1.DeletePolicyRequest{TenantUuid: "bad", PolicyUuid: policyUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeletePolicy(ctx, &authv1.DeletePolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		serviceErr := errors.New("db")
		h = NewPolicyGRPCHandler(tenantResolver, &mockPolicyService{
			getFn:       func(PolicyServiceGetFilter) (*PolicyServiceGetResult, error) { return nil, serviceErr },
			getByUUIDFn: func(uuid.UUID, int64) (*PolicyServiceDataResult, error) { return nil, serviceErr },
			getServicesByPolicyUUIDFn: func(uuid.UUID, int64, PolicyServiceServicesFilter) (*PolicyServiceServicesResult, error) {
				return nil, serviceErr
			},
			createFn: func(int64, string, *string, datatypes.JSON, string, string, bool) (*PolicyServiceDataResult, error) {
				return nil, serviceErr
			},
			updateFn: func(uuid.UUID, int64, string, *string, datatypes.JSON, string, string) (*PolicyServiceDataResult, error) {
				return nil, serviceErr
			},
			setStatusByUUIDFn: func(uuid.UUID, int64, string) (*PolicyServiceDataResult, error) { return nil, serviceErr },
			deleteByUUIDFn:    func(uuid.UUID, int64) (*PolicyServiceDataResult, error) { return nil, serviceErr },
		})
		_, err = h.ListPolicies(ctx, &authv1.ListPoliciesRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.GetPolicy(ctx, &authv1.GetPolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.ListPolicyServices(ctx, &authv1.ListPolicyServicesRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.CreatePolicy(ctx, validCreatePolicyRequest(tenantUUID, document))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.UpdatePolicy(ctx, validUpdatePolicyRequest(tenantUUID, policyUUID, document))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.SetPolicyStatus(ctx, &authv1.SetPolicyStatusRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.DeletePolicy(ctx, &authv1.DeletePolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func validPolicyDocumentStruct(t *testing.T) *structpb.Struct {
	t.Helper()
	doc, err := structpb.NewStruct(map[string]any{
		"version": "v1",
		"statement": []any{
			map[string]any{"effect": "allow", "action": []any{"user:read"}, "resource": []any{"user:*"}},
		},
	})
	require.NoError(t, err)
	return doc
}

func validCreatePolicyRequest(tenantUUID uuid.UUID, document *structpb.Struct) *authv1.CreatePolicyRequest {
	description := "Read users"
	return &authv1.CreatePolicyRequest{TenantUuid: tenantUUID.String(), Name: "user:read", Description: &description, Document: document, Version: "v1", Status: shared.StatusActive}
}

func validUpdatePolicyRequest(tenantUUID uuid.UUID, policyUUID uuid.UUID, document *structpb.Struct) *authv1.UpdatePolicyRequest {
	description := "Read users"
	return &authv1.UpdatePolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String(), Name: "user:read", Description: &description, Document: document, Version: "v1", Status: shared.StatusActive}
}
