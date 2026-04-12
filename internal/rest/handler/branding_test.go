package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestBrandingHandler_Get_NoTenant(t *testing.T) {
	h := NewBrandingHandler(&mockBrandingService{})
	r := httptest.NewRequest(http.MethodGet, "/branding", nil)
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBrandingHandler_Get_ServiceError(t *testing.T) {
	svc := &mockBrandingService{
		getFn: func(_ int64) (*service.BrandingServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewBrandingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/branding", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestBrandingHandler_Get_Success(t *testing.T) {
	svc := &mockBrandingService{
		getFn: func(_ int64) (*service.BrandingServiceDataResult, error) {
			return &service.BrandingServiceDataResult{BrandingUUID: uuid.New(), CompanyName: "Acme"}, nil
		},
	}
	h := NewBrandingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/branding", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBrandingHandler_Update_NoTenant(t *testing.T) {
	h := NewBrandingHandler(&mockBrandingService{})
	r := httptest.NewRequest(http.MethodPut, "/branding", nil)
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBrandingHandler_Update_BadJSON(t *testing.T) {
	h := NewBrandingHandler(&mockBrandingService{})
	r := withTenant(badJSONReq(t, http.MethodPut, "/branding"))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBrandingHandler_Update_ValidationError(t *testing.T) {
	h := NewBrandingHandler(&mockBrandingService{})
	body := map[string]any{"company_name": string(make([]byte, 256))}
	r := withTenant(jsonReq(t, http.MethodPut, "/branding", body))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBrandingHandler_Update_ServiceError(t *testing.T) {
	svc := &mockBrandingService{
		updateFn: func(_ int64, _, _, _, _, _, _, _, _, _, _, _ string) (*service.BrandingServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewBrandingHandler(svc)
	body := map[string]any{"company_name": "Acme"}
	r := withTenant(jsonReq(t, http.MethodPut, "/branding", body))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestBrandingHandler_Update_Success(t *testing.T) {
	svc := &mockBrandingService{
		updateFn: func(_ int64, _, _, _, _, _, _, _, _, _, _, _ string) (*service.BrandingServiceDataResult, error) {
			return &service.BrandingServiceDataResult{BrandingUUID: uuid.New(), CompanyName: "Acme"}, nil
		},
	}
	h := NewBrandingHandler(svc)
	body := map[string]any{"company_name": "Acme"}
	r := withTenant(jsonReq(t, http.MethodPut, "/branding", body))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
