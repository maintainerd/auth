package handler

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
		getSetupStatusFn: func() (*dto.SetupStatusResponseDTO, error) { return nil, assert.AnError },
	}
	h := NewSetupHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/setup/status", nil)
	w := httptest.NewRecorder()
	h.GetSetupStatus(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSetupHandler_GetSetupStatus_Success(t *testing.T) {
	svc := &mockSetupService{
		getSetupStatusFn: func() (*dto.SetupStatusResponseDTO, error) {
			return &dto.SetupStatusResponseDTO{}, nil
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
		createTenantFn: func(req dto.CreateTenantRequestDTO) (*dto.CreateTenantResponseDTO, error) {
			return nil, errValidation
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
		createAdminFn: func(req dto.CreateAdminRequestDTO) (*dto.CreateAdminResponseDTO, error) {
			return nil, errValidation
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
		createProfileFn: func(req dto.CreateProfileRequestDTO) (*dto.CreateProfileResponseDTO, error) {
			return nil, errValidation
		},
	}
	h := NewSetupHandler(svc)
	r := setupRequest(t, map[string]string{"first_name": "John", "last_name": "Doe"})
	w := httptest.NewRecorder()
	h.CreateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── CreateTenant: success path and corrected service-error path ───────────────

// The existing CreateTenant_ServiceError test uses "identifier" instead of
// "display_name", so Validate() fails before the service is called and the
// createTenantFn mock is never exercised. Fix that with a properly valid body.

func TestSetupHandler_CreateTenant_ServiceError_Valid(t *testing.T) {
	svc := &mockSetupService{
		createTenantFn: func(req dto.CreateTenantRequestDTO) (*dto.CreateTenantResponseDTO, error) {
			return nil, errValidation
		},
	}
	h := NewSetupHandler(svc)
	r := setupRequest(t, map[string]any{
		"name": "Test Tenant", "display_name": "Test Tenant Display",
	})
	w := httptest.NewRecorder()
	h.CreateTenant(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetupHandler_CreateTenant_Success(t *testing.T) {
	svc := &mockSetupService{
		createTenantFn: func(req dto.CreateTenantRequestDTO) (*dto.CreateTenantResponseDTO, error) {
			return &dto.CreateTenantResponseDTO{}, nil
		},
	}
	h := NewSetupHandler(svc)
	r := setupRequest(t, map[string]any{
		"name": "Test Tenant", "display_name": "Test Tenant Display",
	})
	w := httptest.NewRecorder()
	h.CreateTenant(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ── CreateAdmin: success path and corrected service-error path ────────────────

// The existing CreateAdmin_ServiceError test omits "fullname" so Validate()
// fails before the service is called. Fix with a complete valid body.

func TestSetupHandler_CreateAdmin_ServiceError_Valid(t *testing.T) {
	svc := &mockSetupService{
		createAdminFn: func(req dto.CreateAdminRequestDTO) (*dto.CreateAdminResponseDTO, error) {
			return nil, errValidation
		},
	}
	h := NewSetupHandler(svc)
	r := setupRequest(t, map[string]string{
		"username": "admin", "fullname": "Admin User",
		"password": "Admin@1234", "email": "admin@example.com",
	})
	w := httptest.NewRecorder()
	h.CreateAdmin(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetupHandler_CreateAdmin_Success(t *testing.T) {
	svc := &mockSetupService{
		createAdminFn: func(req dto.CreateAdminRequestDTO) (*dto.CreateAdminResponseDTO, error) {
			return &dto.CreateAdminResponseDTO{}, nil
		},
	}
	h := NewSetupHandler(svc)
	r := setupRequest(t, map[string]string{
		"username": "admin", "fullname": "Admin User",
		"password": "Admin@1234", "email": "admin@example.com",
	})
	w := httptest.NewRecorder()
	h.CreateAdmin(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ── CreateProfile: all missing branches ──────────────────────────────────────

func TestSetupHandler_CreateProfile_BadJSON(t *testing.T) {
	h := NewSetupHandler(&mockSetupService{})
	r := httptest.NewRequest(http.MethodPost, "/setup/profile",
		bytes.NewBufferString(`{bad}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetupHandler_CreateProfile_ValidationError(t *testing.T) {
	h := NewSetupHandler(&mockSetupService{})
	// empty body → first_name is required → Validate() fails
	r := setupRequest(t, map[string]string{})
	w := httptest.NewRecorder()
	h.CreateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ProfileAlreadyExists: service returns an error containing "profile already exists"
// → covers the strings.Contains branch (lines 103-106).
func TestSetupHandler_CreateProfile_AlreadyExists(t *testing.T) {
	svc := &mockSetupService{
		createProfileFn: func(req dto.CreateProfileRequestDTO) (*dto.CreateProfileResponseDTO, error) {
			return nil, errValidation
		},
	}
	h := NewSetupHandler(svc)
	r := setupRequest(t, map[string]string{"first_name": "John"})
	w := httptest.NewRecorder()
	h.CreateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetupHandler_CreateProfile_Success(t *testing.T) {
	svc := &mockSetupService{
		createProfileFn: func(req dto.CreateProfileRequestDTO) (*dto.CreateProfileResponseDTO, error) {
			return &dto.CreateProfileResponseDTO{}, nil
		},
	}
	h := NewSetupHandler(svc)
	r := setupRequest(t, map[string]string{"first_name": "John"})
	w := httptest.NewRecorder()
	h.CreateProfile(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}
