package iam

import (
	"context"
	"testing"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

type mockTenantResolver struct {
	getByUUIDFn func(uuid.UUID) (*tenant.TenantServiceDataResult, error)
}

func (m mockTenantResolver) GetByUUID(_ context.Context, tenantUUID uuid.UUID) (*tenant.TenantServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(tenantUUID)
	}
	return &tenant.TenantServiceDataResult{TenantID: 99, TenantUUID: tenantUUID}, nil
}

func TestIAMGRPCHelpers(t *testing.T) {
	id := uuid.New()
	parsed, err := iamParseUUID(id.String(), "Test UUID")
	require.NoError(t, err)
	assert.Equal(t, id, parsed)
	_, err = iamParseUUID("", "Test UUID")
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = iamParseUUID("bad", "Test UUID")
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	defaultPage := iamPaginationDTO(nil)
	assert.Equal(t, 1, defaultPage.Page)
	pagination := iamPaginationDTO(&authv1.Pagination{SortBy: "created_at", SortOrder: SortOrderDesc})
	assert.Equal(t, 1, pagination.Page)
	assert.Equal(t, "created_at", pagination.SortBy)
	assert.Equal(t, int32(3), iamPageProto(12, 2, 5, 3).TotalPages)
	assert.Nil(t, iamOptionalString(""))
	require.NotNil(t, iamOptionalString("value"))

	assert.Nil(t, serviceProto(nil))
	assert.Nil(t, apiProto(nil))
	assert.NotNil(t, serviceProto(&ServiceServiceDataResult{ServiceUUID: id}))
	assert.NotNil(t, apiProto(&APIServiceDataResult{APIUUID: id, Service: &ServiceServiceDataResult{ServiceUUID: id}}))
}

func accessTenantResolver(t *testing.T, tenantUUID uuid.UUID) mockTenantResolver {
	t.Helper()
	return mockTenantResolver{getByUUIDFn: func(id uuid.UUID) (*tenant.TenantServiceDataResult, error) {
		assert.Equal(t, tenantUUID, id)
		return &tenant.TenantServiceDataResult{TenantID: 77, TenantUUID: id}, nil
	}}
}

func TestAccessGRPCHelpers(t *testing.T) {
	assert.Nil(t, permissionProto(nil))
	assert.Nil(t, policyProto(nil))
	assert.Nil(t, policyServiceProto(nil))
	assert.Nil(t, roleProto(nil))
	assert.Empty(t, policyDocumentProto(nil).AsMap())
	assert.Empty(t, policyDocumentProto(datatypes.JSON(`bad`)).AsMap())
	_, err := policyDocumentJSON(nil)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	ids, err := parseIAMUUIDs([]string{testResourceUUID.String()}, "Permission UUID")
	require.NoError(t, err)
	assert.Equal(t, testResourceUUID, ids[0])
}
