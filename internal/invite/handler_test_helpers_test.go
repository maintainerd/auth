package invite

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/stretchr/testify/require"
)

const testTenantID int64 = 1

var (
	testTenantUUID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testUserUUID     = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testResourceUUID = uuid.MustParse("00000000-0000-0000-0000-000000000099")
)

type mockInviteService struct {
	sendInviteFn func(int64, string, int64, []string) (*model.Invite, error)
}

func (m *mockInviteService) SendInvite(_ context.Context, tenantID int64, email string, userID int64, roleUUIDs []string) (*model.Invite, error) {
	if m.sendInviteFn != nil {
		return m.sendInviteFn(tenantID, email, userID, roleUUIDs)
	}
	return nil, nil
}

func withTenant(r *http.Request) *http.Request {
	tenant := &model.Tenant{TenantID: testTenantID, TenantUUID: testTenantUUID}
	return middleware.WithAuthContext(r, &middleware.AuthContext{Tenant: tenant})
}

func withTenantAndUser(r *http.Request) *http.Request {
	tenant := &model.Tenant{TenantID: testTenantID, TenantUUID: testTenantUUID}
	user := &model.User{UserID: 2, UserUUID: testUserUUID}
	return middleware.WithAuthContext(r, &middleware.AuthContext{Tenant: tenant, User: user})
}

func jsonReq(t *testing.T, method, url string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	r := httptest.NewRequest(method, url, &buf)
	r.Header.Set("Content-Type", "application/json")
	return r
}
