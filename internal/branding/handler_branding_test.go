package branding

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestBrandingHandler_List_NoTenant(t *testing.T) {
	h := NewBrandingHandler(&mockBrandingService{})
	r := httptest.NewRequest(http.MethodGet, "/branding", nil)
	w := httptest.NewRecorder()
	h.List(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBrandingHandler_List_ServiceError(t *testing.T) {
	svc := &mockBrandingService{
		listFn: func(_ int64) ([]*BrandingServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewBrandingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/branding", nil))
	w := httptest.NewRecorder()
	h.List(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestBrandingHandler_List_Success(t *testing.T) {
	svc := &mockBrandingService{
		listFn: func(_ int64) ([]*BrandingServiceDataResult, error) {
			return []*BrandingServiceDataResult{{BrandingUUID: uuid.New(), CompanyName: "Acme"}}, nil
		},
	}
	h := NewBrandingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/branding", nil))
	w := httptest.NewRecorder()
	h.List(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBrandingHandler_Update_NoTenant(t *testing.T) {
	h := NewBrandingHandler(&mockBrandingService{})
	r := httptest.NewRequest(http.MethodPut, "/branding", nil)
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBrandingHandler_Update_BadUUID(t *testing.T) {
	h := NewBrandingHandler(&mockBrandingService{})
	r := withChiParam(withTenant(jsonReq(t, http.MethodPut, "/branding/bad", map[string]any{"company_name": "Acme"})), "branding_uuid", "not-a-uuid")
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBrandingHandler_Update_BadJSON(t *testing.T) {
	h := NewBrandingHandler(&mockBrandingService{})
	r := withChiParam(withTenant(badJSONReq(t, http.MethodPut, "/branding")), "branding_uuid", uuid.New().String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBrandingHandler_Update_ValidationError(t *testing.T) {
	h := NewBrandingHandler(&mockBrandingService{})
	body := map[string]any{"company_name": string(make([]byte, 256))}
	r := withChiParam(withTenant(jsonReq(t, http.MethodPut, "/branding", body)), "branding_uuid", uuid.New().String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBrandingHandler_Update_ServiceError(t *testing.T) {
	svc := &mockBrandingService{
		updateByUUIDFn: func(_ uuid.UUID, _ int64, _ string, _ string, _ string, _ string, _ datatypes.JSON, _ string, _ string, _ string) (*BrandingServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewBrandingHandler(svc)
	body := map[string]any{"company_name": "Acme"}
	r := withChiParam(withTenant(jsonReq(t, http.MethodPut, "/branding", body)), "branding_uuid", uuid.New().String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestBrandingHandler_Update_Success(t *testing.T) {
	svc := &mockBrandingService{
		updateByUUIDFn: func(_ uuid.UUID, _ int64, _ string, _ string, _ string, _ string, _ datatypes.JSON, _ string, _ string, _ string) (*BrandingServiceDataResult, error) {
			return &BrandingServiceDataResult{BrandingUUID: uuid.New(), CompanyName: "Acme"}, nil
		},
	}
	h := NewBrandingHandler(svc)
	body := map[string]any{"company_name": "Acme"}
	r := withChiParam(withTenant(jsonReq(t, http.MethodPut, "/branding", body)), "branding_uuid", uuid.New().String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
