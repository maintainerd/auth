package resthandler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRequest(t *testing.T, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	r := httptest.NewRequest(http.MethodPost, "/setup", &buf)
	r.Header.Set("Content-Type", "application/json")
	return r
}

// ---------------------------------------------------------------------------
// GetSetupStatus
// ---------------------------------------------------------------------------

func TestSetupHandler_GetSetupStatus_ServiceError(t *testing.T) {
	svc := &mockSetupService{
		getSetupStatusFn: func() (*dto.SetupStatusResponseDto, error) { return nil, assert.AnError },
	}
	h := NewSetupHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/setup/status", nil)
	w := httptest.NewRecorder()
	h.GetSetupStatus(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSetupHandler_GetSetupStatus_Success(t *testing.T) {
	svc := &mockSetupService{
		getSetupStatusFn: func() (*dto.SetupStatusResponseDto, error) {
			return &dto.SetupStatusResponseDto{}, nil
		},
	}
	h := NewSetupHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/setup/status", nil)
	w := httptest.NewRecorder()
	h.GetSetupStatus(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// CreateTenant
// ---------------------------------------------------------------------------

func TestSetupHandler_CreateTenant_InvalidBody(t *testing.T) {
	h := NewSetupHandler(&mockSetupService{})
	r := httptest.NewRequest(http.MethodPost, "/setup/tenant",
		bytes.NewBufferString(`not-json`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateTenant(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetupHandler_CreateTenant_ValidationError(t *testing.T) {
	h := NewSetupHandler(&mockSetupService{})
	r := setupRequest(t, map[string]string{}) // missing required fields
	w := httptest.NewRecorder()
	h.CreateTenant(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetupHandler_CreateTenant_ServiceError(t *testing.T) {
	svc := &mockSetupService{
		createTenantFn: func(req dto.CreateTenantRequestDto) (*dto.CreateTenantResponseDto, error) {
			return nil, assert.AnError
		},
	}
	h := NewSetupHandler(svc)
	r := setupRequest(t, map[string]any{
		"name":       "Test Tenant",
		"identifier": "test-tenant",
	})
	w := httptest.NewRecorder()
	h.CreateTenant(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// CreateAdmin
// ---------------------------------------------------------------------------

func TestSetupHandler_CreateAdmin_InvalidBody(t *testing.T) {
	h := NewSetupHandler(&mockSetupService{})
	r := httptest.NewRequest(http.MethodPost, "/setup/admin",
		bytes.NewBufferString(`bad`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateAdmin(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetupHandler_CreateAdmin_ValidationError(t *testing.T) {
	h := NewSetupHandler(&mockSetupService{})
	r := setupRequest(t, map[string]string{})
	w := httptest.NewRecorder()
	h.CreateAdmin(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetupHandler_CreateAdmin_ServiceError(t *testing.T) {
	svc := &mockSetupService{
		createAdminFn: func(req dto.CreateAdminRequestDto) (*dto.CreateAdminResponseDto, error) {
			return nil, assert.AnError
		},
	}
	h := NewSetupHandler(svc)
	r := setupRequest(t, map[string]string{
		"username": "admin", "password": "Admin@1234", "email": "admin@test.com",
	})
	w := httptest.NewRecorder()
	h.CreateAdmin(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// CreateProfile
// ---------------------------------------------------------------------------

func TestSetupHandler_CreateProfile_ServiceError(t *testing.T) {
	svc := &mockSetupService{
		createProfileFn: func(req dto.CreateProfileRequestDto) (*dto.CreateProfileResponseDto, error) {
			return nil, assert.AnError
		},
	}
	h := NewSetupHandler(svc)
	r := setupRequest(t, map[string]string{"first_name": "John", "last_name": "Doe"})
	w := httptest.NewRecorder()
	h.CreateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
