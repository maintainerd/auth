package invite

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authctx"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/stretchr/testify/require"
)

const testTenantID int64 = 1

var (
	testTenantUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testUserUUID   = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

type mockInviteService struct {
	sendInviteFn   func(int64, string, int64, *string, *string) (*Invite, error)
	resendInviteFn func(uuid.UUID, int64) (*Invite, error)
	listInvitesFn  func(int64) ([]Invite, error)
	revokeInviteFn func(uuid.UUID, int64) error
	getByTokenFn   func(string) (*Invite, error)
}

func (m *mockInviteService) SendInvite(_ context.Context, tenantID int64, email string, userID int64, registrationFlowUUID *string, callbackURL *string) (*Invite, error) {
	if m.sendInviteFn != nil {
		return m.sendInviteFn(tenantID, email, userID, registrationFlowUUID, callbackURL)
	}
	return nil, nil
}

func (m *mockInviteService) ResendInvite(_ context.Context, inviteUUID uuid.UUID, tenantID int64) (*Invite, error) {
	if m.resendInviteFn != nil {
		return m.resendInviteFn(inviteUUID, tenantID)
	}
	return nil, nil
}

func (m *mockInviteService) ListInvites(_ context.Context, tenantID int64) ([]Invite, error) {
	if m.listInvitesFn != nil {
		return m.listInvitesFn(tenantID)
	}
	return nil, nil
}

func (m *mockInviteService) RevokeInvite(_ context.Context, inviteUUID uuid.UUID, tenantID int64) error {
	if m.revokeInviteFn != nil {
		return m.revokeInviteFn(inviteUUID, tenantID)
	}
	return nil
}

func (m *mockInviteService) GetByToken(_ context.Context, inviteToken string) (*Invite, error) {
	if m.getByTokenFn != nil {
		return m.getByTokenFn(inviteToken)
	}
	return nil, nil
}

func (m *mockInviteService) GetByUUID(_ context.Context, inviteUUID uuid.UUID, tenantID int64) (*Invite, error) {
	return nil, nil
}

func withTenant(r *http.Request) *http.Request {
	tenant := &Tenant{TenantID: testTenantID, TenantUUID: testTenantUUID}
	return middleware.WithAuthContext(r, &authctx.AuthContext{Tenant: tenant})
}

func withTenantAndUser(r *http.Request) *http.Request {
	tenant := &Tenant{TenantID: testTenantID, TenantUUID: testTenantUUID}
	user := &User{UserID: 2, UserUUID: testUserUUID}
	return middleware.WithAuthContext(r, &authctx.AuthContext{Tenant: tenant, User: user})
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
