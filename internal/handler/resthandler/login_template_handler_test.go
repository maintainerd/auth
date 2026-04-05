package resthandler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

func loginTmplResult() service.LoginTemplateServiceDataResult {
	return service.LoginTemplateServiceDataResult{LoginTemplateUUID: testResourceUUID, Name: "tmpl", Template: "<html/>", Status: "active"}
}

func TestLoginTemplateHandler_GetAll(t *testing.T) {
	svc := &mockLoginTemplateService{getAllFn: func(_ int64, _ *string, _ []string, _ *string, _, _ *bool, _, _ int, _, _ string) (*service.LoginTemplateServiceListResult, error) {
		return &service.LoginTemplateServiceListResult{Data: []service.LoginTemplateServiceDataResult{loginTmplResult()}}, nil
	}}
	h := NewLoginTemplateHandler(svc)
	w := httptest.NewRecorder()
	h.GetAll(w, withTenant(jsonReq(t, http.MethodGet, "/login-templates", nil)))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginTemplateHandler_GetAll_Error(t *testing.T) {
	svc := &mockLoginTemplateService{getAllFn: func(_ int64, _ *string, _ []string, _ *string, _, _ *bool, _, _ int, _, _ string) (*service.LoginTemplateServiceListResult, error) {
		return nil, errors.New("db")
	}}
	h := NewLoginTemplateHandler(svc)
	w := httptest.NewRecorder()
	h.GetAll(w, withTenant(jsonReq(t, http.MethodGet, "/login-templates", nil)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestLoginTemplateHandler_Get(t *testing.T) {
	res := loginTmplResult()
	svc := &mockLoginTemplateService{getByUUIDFn: func(_ uuid.UUID, _ int64) (*service.LoginTemplateServiceDataResult, error) { return &res, nil }}
	h := NewLoginTemplateHandler(svc)
	r := withChiParam(withTenant(jsonReq(t, http.MethodGet, "/login-templates/id", nil)), "login_template_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginTemplateHandler_Get_BadUUID(t *testing.T) {
	h := NewLoginTemplateHandler(&mockLoginTemplateService{})
	r := withChiParam(withTenant(jsonReq(t, http.MethodGet, "/login-templates/bad", nil)), "login_template_uuid", "not-a-uuid")
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginTemplateHandler_Get_NotFound(t *testing.T) {
	svc := &mockLoginTemplateService{getByUUIDFn: func(_ uuid.UUID, _ int64) (*service.LoginTemplateServiceDataResult, error) {
		return nil, errors.New("not found")
	}}
	h := NewLoginTemplateHandler(svc)
	r := withChiParam(withTenant(jsonReq(t, http.MethodGet, "/login-templates/id", nil)), "login_template_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLoginTemplateHandler_Create(t *testing.T) {
	res := loginTmplResult()
	svc := &mockLoginTemplateService{createFn: func(_ int64, _ string, _ *string, _ string, _ map[string]any, _ string) (*service.LoginTemplateServiceDataResult, error) {
		return &res, nil
	}}
	h := NewLoginTemplateHandler(svc)
	body := map[string]any{"name": "tmpl1", "template": "modern"}
	w := httptest.NewRecorder()
	h.Create(w, withTenant(jsonReq(t, http.MethodPost, "/login-templates", body)))
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestLoginTemplateHandler_Create_Error(t *testing.T) {
	svc := &mockLoginTemplateService{createFn: func(_ int64, _ string, _ *string, _ string, _ map[string]any, _ string) (*service.LoginTemplateServiceDataResult, error) {
		return nil, errors.New("fail")
	}}
	h := NewLoginTemplateHandler(svc)
	body := map[string]any{"name": "tmpl1", "template": "modern"}
	w := httptest.NewRecorder()
	h.Create(w, withTenant(jsonReq(t, http.MethodPost, "/login-templates", body)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginTemplateHandler_Update(t *testing.T) {
	res := loginTmplResult()
	svc := &mockLoginTemplateService{updateFn: func(_ uuid.UUID, _ int64, _ string, _ *string, _ string, _ map[string]any, _ string) (*service.LoginTemplateServiceDataResult, error) {
		return &res, nil
	}}
	h := NewLoginTemplateHandler(svc)
	body := map[string]any{"name": "upd", "template": "modern"}
	r := withChiParam(withTenant(jsonReq(t, http.MethodPut, "/login-templates/id", body)), "login_template_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginTemplateHandler_Delete(t *testing.T) {
	res := loginTmplResult()
	svc := &mockLoginTemplateService{deleteFn: func(_ uuid.UUID, _ int64) (*service.LoginTemplateServiceDataResult, error) { return &res, nil }}
	h := NewLoginTemplateHandler(svc)
	r := withChiParam(withTenant(jsonReq(t, http.MethodDelete, "/login-templates/id", nil)), "login_template_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginTemplateHandler_UpdateStatus(t *testing.T) {
	res := loginTmplResult()
	svc := &mockLoginTemplateService{updateStatusFn: func(_ uuid.UUID, _ int64, _ string) (*service.LoginTemplateServiceDataResult, error) {
		return &res, nil
	}}
	h := NewLoginTemplateHandler(svc)
	r := withChiParam(withTenant(jsonReq(t, http.MethodPatch, "/login-templates/id/status", map[string]any{"status": "inactive"})), "login_template_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
