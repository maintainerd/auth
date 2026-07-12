package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserTrustedDeviceService struct {
	listDevicesFn  func(ctx context.Context, userID int64) ([]UserTrustedDevice, error)
	deleteDeviceFn func(ctx context.Context, deviceUUID string) error
}

func (m *mockUserTrustedDeviceService) ListDevices(ctx context.Context, userID int64) ([]UserTrustedDevice, error) {
	if m.listDevicesFn != nil {
		return m.listDevicesFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockUserTrustedDeviceService) DeleteDevice(ctx context.Context, deviceUUID string) error {
	if m.deleteDeviceFn != nil {
		return m.deleteDeviceFn(ctx, deviceUUID)
	}
	return nil
}

func TestUserTrustedDeviceHandler_ListMyDevices_NoAuth(t *testing.T) {
	h := NewUserTrustedDeviceHandler(&mockUserTrustedDeviceService{}, nil, nil)
	r := httptest.NewRequest(http.MethodGet, "/me/devices", nil)
	w := httptest.NewRecorder()
	h.ListMyDevices(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserTrustedDeviceHandler_ListMyDevices_ServiceError(t *testing.T) {
	svc := &mockUserTrustedDeviceService{
		listDevicesFn: func(ctx context.Context, userID int64) ([]UserTrustedDevice, error) {
			return nil, assert.AnError
		},
	}
	h := NewUserTrustedDeviceHandler(svc, nil, nil)
	r := withTenantAndUser(httptest.NewRequest(http.MethodGet, "/me/devices", nil))
	w := httptest.NewRecorder()
	h.ListMyDevices(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUserTrustedDeviceHandler_ListMyDevices_Success(t *testing.T) {
	svc := &mockUserTrustedDeviceService{
		listDevicesFn: func(ctx context.Context, userID int64) ([]UserTrustedDevice, error) {
			return []UserTrustedDevice{}, nil
		},
	}
	h := NewUserTrustedDeviceHandler(svc, nil, nil)
	r := withTenantAndUser(httptest.NewRequest(http.MethodGet, "/me/devices", nil))
	w := httptest.NewRecorder()
	h.ListMyDevices(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserTrustedDeviceHandler_DeleteMyDevice_NoAuth(t *testing.T) {
	h := NewUserTrustedDeviceHandler(&mockUserTrustedDeviceService{}, nil, nil)
	r := httptest.NewRequest(http.MethodDelete, "/me/devices/123", nil)
	w := httptest.NewRecorder()
	h.DeleteMyDevice(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserTrustedDeviceHandler_DeleteMyDevice_MissingUUID(t *testing.T) {
	h := NewUserTrustedDeviceHandler(&mockUserTrustedDeviceService{}, nil, nil)
	r := withTenantAndUser(httptest.NewRequest(http.MethodDelete, "/me/devices/123", nil))
	r = withChiParam(r, "device_uuid", "")
	w := httptest.NewRecorder()
	h.DeleteMyDevice(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserTrustedDeviceHandler_DeleteMyDevice_NotOwned(t *testing.T) {
	// Ownership guard (IDOR): revoking a device UUID the user does not own is a
	// 404, not a delete.
	svc := &mockUserTrustedDeviceService{
		listDevicesFn: func(ctx context.Context, userID int64) ([]UserTrustedDevice, error) {
			return []UserTrustedDevice{{UserTrustedDeviceUUID: uuid.New()}}, nil
		},
	}
	h := NewUserTrustedDeviceHandler(svc, nil, nil)
	r := withTenantAndUser(httptest.NewRequest(http.MethodDelete, "/me/devices/123", nil))
	r = withChiParam(r, "device_uuid", uuid.New().String())
	w := httptest.NewRecorder()
	h.DeleteMyDevice(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserTrustedDeviceHandler_DeleteMyDevice_ServiceError(t *testing.T) {
	devUUID := uuid.New()
	svc := &mockUserTrustedDeviceService{
		listDevicesFn: func(ctx context.Context, userID int64) ([]UserTrustedDevice, error) {
			return []UserTrustedDevice{{UserTrustedDeviceUUID: devUUID}}, nil
		},
		deleteDeviceFn: func(ctx context.Context, deviceUUID string) error {
			return assert.AnError
		},
	}
	h := NewUserTrustedDeviceHandler(svc, nil, nil)
	r := withTenantAndUser(httptest.NewRequest(http.MethodDelete, "/me/devices/123", nil))
	r = withChiParam(r, "device_uuid", devUUID.String())
	w := httptest.NewRecorder()
	h.DeleteMyDevice(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUserTrustedDeviceHandler_DeleteMyDevice_Success(t *testing.T) {
	devUUID := uuid.New()
	svc := &mockUserTrustedDeviceService{
		listDevicesFn: func(ctx context.Context, userID int64) ([]UserTrustedDevice, error) {
			return []UserTrustedDevice{{UserTrustedDeviceUUID: devUUID}}, nil
		},
	}
	h := NewUserTrustedDeviceHandler(svc, nil, nil)
	r := withTenantAndUser(httptest.NewRequest(http.MethodDelete, "/me/devices/123", nil))
	r = withChiParam(r, "device_uuid", devUUID.String())
	w := httptest.NewRecorder()
	h.DeleteMyDevice(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserTrustedDeviceHandler_GetUserDevices_MissingUUID(t *testing.T) {
	h := NewUserTrustedDeviceHandler(&mockUserTrustedDeviceService{}, nil, nil)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/users/123/devices", nil))
	r = withChiParam(r, "user_uuid", "")
	w := httptest.NewRecorder()
	h.GetUserDevices(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserTrustedDeviceHandler_GetUserDevices_InvalidUUID(t *testing.T) {
	h := NewUserTrustedDeviceHandler(&mockUserTrustedDeviceService{}, nil, nil)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/users/123/devices", nil))
	r = withChiParam(r, "user_uuid", "not-a-uuid")
	w := httptest.NewRecorder()
	h.GetUserDevices(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserTrustedDeviceHandler_GetUserDevices_UserNotFound(t *testing.T) {
	h := NewUserTrustedDeviceHandler(&mockUserTrustedDeviceService{}, nil, &mockUserRepo{
		findByUUIDFn: func(id any, p ...string) (*User, error) {
			return nil, nil
		},
	})
	uu := uuid.New().String()
	r := withTenant(httptest.NewRequest(http.MethodGet, "/users/"+uu+"/devices", nil))
	r = withChiParam(r, "user_uuid", uu)
	w := httptest.NewRecorder()
	h.GetUserDevices(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserTrustedDeviceHandler_GetUserDevices_WrongTenant(t *testing.T) {
	// Tenant isolation: a user that belongs to another tenant is a 404, so an
	// admin cannot read across tenants by guessing a user UUID.
	testUser := &User{UserID: 42, UserIdentities: []UserIdentity{{TenantID: tenantID + 999}}}
	h := NewUserTrustedDeviceHandler(&mockUserTrustedDeviceService{}, nil, &mockUserRepo{
		findByUUIDFn: func(id any, p ...string) (*User, error) {
			return testUser, nil
		},
	})
	uu := uuid.New().String()
	r := withTenant(httptest.NewRequest(http.MethodGet, "/users/"+uu+"/devices", nil))
	r = withChiParam(r, "user_uuid", uu)
	w := httptest.NewRecorder()
	h.GetUserDevices(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserTrustedDeviceHandler_GetUserDevices_ServiceError(t *testing.T) {
	testUser := &User{UserID: 42, UserIdentities: []UserIdentity{{TenantID: tenantID}}}
	svc := &mockUserTrustedDeviceService{
		listDevicesFn: func(ctx context.Context, userID int64) ([]UserTrustedDevice, error) {
			return nil, assert.AnError
		},
	}
	h := NewUserTrustedDeviceHandler(svc, nil, &mockUserRepo{
		findByUUIDFn: func(id any, p ...string) (*User, error) {
			return testUser, nil
		},
	})
	uu := uuid.New().String()
	r := withTenant(httptest.NewRequest(http.MethodGet, "/users/"+uu+"/devices", nil))
	r = withChiParam(r, "user_uuid", uu)
	w := httptest.NewRecorder()
	h.GetUserDevices(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUserTrustedDeviceHandler_GetUserDevices_Success(t *testing.T) {
	testUser := &User{UserID: 42, UserIdentities: []UserIdentity{{TenantID: tenantID}}}
	svc := &mockUserTrustedDeviceService{
		listDevicesFn: func(ctx context.Context, userID int64) ([]UserTrustedDevice, error) {
			return []UserTrustedDevice{}, nil
		},
	}
	h := NewUserTrustedDeviceHandler(svc, nil, &mockUserRepo{
		findByUUIDFn: func(id any, p ...string) (*User, error) {
			return testUser, nil
		},
	})
	uu := uuid.New().String()
	r := withTenantAndUser(httptest.NewRequest(http.MethodGet, "/users/"+uu+"/devices", nil))
	r = withChiParam(r, "user_uuid", uu)
	w := httptest.NewRecorder()
	h.GetUserDevices(w, r)
	require.Equal(t, http.StatusOK, w.Code)
}
