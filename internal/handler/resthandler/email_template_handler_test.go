package resthandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestEmailTemplateHandler_GetAll_NoTenant(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := httptest.NewRequest(http.MethodGet, "/email-templates", nil)
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestEmailTemplateHandler_GetAll_ServiceError(t *testing.T) {
	svc := &mockEmailTemplateService{
		getAllFn: func(tid int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*service.EmailTemplateServiceListResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewEmailTemplateHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/email-templates?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestEmailTemplateHandler_GetAll_Success(t *testing.T) {
	svc := &mockEmailTemplateService{
		getAllFn: func(tid int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*service.EmailTemplateServiceListResult, error) {
			return &service.EmailTemplateServiceListResult{}, nil
		},
	}
	h := NewEmailTemplateHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/email-templates?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEmailTemplateHandler_GetByUUID_NoTenant(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := withChiParam(httptest.NewRequest(http.MethodGet, "/email-templates/"+testResourceUUID.String(), nil), "email_template_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestEmailTemplateHandler_GetByUUID_InvalidUUID(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/email-templates/bad", nil), "email_template_uuid", "bad"))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailTemplateHandler_GetByUUID_NotFound(t *testing.T) {
	svc := &mockEmailTemplateService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.EmailTemplateServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewEmailTemplateHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/email-templates/"+testResourceUUID.String(), nil), "email_template_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestEmailTemplateHandler_Create_NoTenant(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := jsonReq(t, http.MethodPost, "/email-templates", map[string]string{"name": "t"})
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestEmailTemplateHandler_Create_ServiceError(t *testing.T) {
	svc := &mockEmailTemplateService{
		createFn: func(tid int64, name, subject, bodyHTML string, bodyPlain *string, status string, isDefault bool) (*service.EmailTemplateServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewEmailTemplateHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPost, "/email-templates", map[string]any{
		"name": "tmpl1", "subject": "Hello", "body_html": "<p>hi</p>", "status": "active",
	}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailTemplateHandler_Delete_InvalidUUID(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/email-templates/bad", nil), "email_template_uuid", "bad"))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailTemplateHandler_Delete_Success(t *testing.T) {
	svc := &mockEmailTemplateService{
		deleteFn: func(id uuid.UUID, tid int64) (*service.EmailTemplateServiceDataResult, error) {
			return &service.EmailTemplateServiceDataResult{Name: "tmpl1"}, nil
		},
	}
	h := NewEmailTemplateHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/email-templates/"+testResourceUUID.String(), nil), "email_template_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
