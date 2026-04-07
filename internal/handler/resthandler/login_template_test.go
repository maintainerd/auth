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

func validLoginTemplateBody() map[string]any {
	return map[string]any{"name": "tmpl1", "template": "modern"}
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

func TestLoginTemplateHandler_GetAll_NoTenant(t *testing.T) {
	w := httptest.NewRecorder()
	NewLoginTemplateHandler(&mockLoginTemplateService{}).GetAll(w, jsonReq(t, http.MethodGet, "/login-templates", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginTemplateHandler_GetAll_ValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	NewLoginTemplateHandler(&mockLoginTemplateService{}).GetAll(w, withTenant(jsonReq(t, http.MethodGet, "/login-templates?sort_order=invalid", nil)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginTemplateHandler_GetAll_WithFilters(t *testing.T) {
	// Covers status != "" (line 59-61), is_default != "" (65-68), is_system != "" (71-74) branches.
	svc := &mockLoginTemplateService{getAllFn: func(_ int64, _ *string, _ []string, _ *string, _, _ *bool, _, _ int, _, _ string) (*service.LoginTemplateServiceListResult, error) {
		return &service.LoginTemplateServiceListResult{}, nil
	}}
	w := httptest.NewRecorder()
	NewLoginTemplateHandler(svc).GetAll(w, withTenant(jsonReq(t, http.MethodGet,
		"/login-templates?page=1&limit=10&status=active&is_default=true&is_system=false", nil)))
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

func TestLoginTemplateHandler_Get_NoTenant(t *testing.T) {
	w := httptest.NewRecorder()
	NewLoginTemplateHandler(&mockLoginTemplateService{}).Get(w, jsonReq(t, http.MethodGet, "/login-templates/id", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
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

func TestLoginTemplateHandler_Create_NoTenant(t *testing.T) {
	w := httptest.NewRecorder()
	NewLoginTemplateHandler(&mockLoginTemplateService{}).Create(w, withUser(jsonReq(t, http.MethodPost, "/", validLoginTemplateBody())))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginTemplateHandler_Create_BadJSON(t *testing.T) {
	w := httptest.NewRecorder()
	NewLoginTemplateHandler(&mockLoginTemplateService{}).Create(w, withTenantAndUser(badJSONReq(t, http.MethodPost, "/")))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginTemplateHandler_Create_ValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	NewLoginTemplateHandler(&mockLoginTemplateService{}).Create(w, withTenantAndUser(jsonReq(t, http.MethodPost, "/", map[string]any{})))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginTemplateHandler_Create_CustomStatus(t *testing.T) {
	// Covers req.Status != nil branch (lines 173-175).
	svc := &mockLoginTemplateService{createFn: func(_ int64, _ string, _ *string, _ string, _ map[string]any, _ string) (*service.LoginTemplateServiceDataResult, error) {
		return &service.LoginTemplateServiceDataResult{Name: "tmpl1"}, nil
	}}
	body := map[string]any{"name": "tmpl1", "template": "modern", "status": "inactive"}
	w := httptest.NewRecorder()
	NewLoginTemplateHandler(svc).Create(w, withTenantAndUser(jsonReq(t, http.MethodPost, "/", body)))
	assert.Equal(t, http.StatusCreated, w.Code)
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

func TestLoginTemplateHandler_Update_NoTenant(t *testing.T) {
	w := httptest.NewRecorder()
	r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/", validLoginTemplateBody()), "login_template_uuid", testResourceUUID.String()))
	NewLoginTemplateHandler(&mockLoginTemplateService{}).Update(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginTemplateHandler_Update_InvalidUUID(t *testing.T) {
	w := httptest.NewRecorder()
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", validLoginTemplateBody()), "login_template_uuid", "bad"))
	NewLoginTemplateHandler(&mockLoginTemplateService{}).Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginTemplateHandler_Update_BadJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPut, "/"), "login_template_uuid", testResourceUUID.String()))
	NewLoginTemplateHandler(&mockLoginTemplateService{}).Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginTemplateHandler_Update_ValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{}), "login_template_uuid", testResourceUUID.String()))
	NewLoginTemplateHandler(&mockLoginTemplateService{}).Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginTemplateHandler_Update_ServiceError(t *testing.T) {
	svc := &mockLoginTemplateService{updateFn: func(_ uuid.UUID, _ int64, _ string, _ *string, _ string, _ map[string]any, _ string) (*service.LoginTemplateServiceDataResult, error) {
		return nil, errors.New("fail")
	}}
	w := httptest.NewRecorder()
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", validLoginTemplateBody()), "login_template_uuid", testResourceUUID.String()))
	NewLoginTemplateHandler(svc).Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginTemplateHandler_Update_CustomStatus(t *testing.T) {
	// Covers req.Status != nil branch (lines 236-238).
	svc := &mockLoginTemplateService{updateFn: func(_ uuid.UUID, _ int64, _ string, _ *string, _ string, _ map[string]any, _ string) (*service.LoginTemplateServiceDataResult, error) {
		return &service.LoginTemplateServiceDataResult{Name: "tmpl1"}, nil
	}}
	body := map[string]any{"name": "tmpl1", "template": "modern", "status": "inactive"}
	w := httptest.NewRecorder()
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", body), "login_template_uuid", testResourceUUID.String()))
	NewLoginTemplateHandler(svc).Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
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

func TestLoginTemplateHandler_Delete_NoTenant(t *testing.T) {
	w := httptest.NewRecorder()
	NewLoginTemplateHandler(&mockLoginTemplateService{}).Delete(w,
		withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "login_template_uuid", testResourceUUID.String()))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginTemplateHandler_Delete_InvalidUUID(t *testing.T) {
	w := httptest.NewRecorder()
	NewLoginTemplateHandler(&mockLoginTemplateService{}).Delete(w,
		withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "login_template_uuid", "bad")))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginTemplateHandler_Delete_ServiceError(t *testing.T) {
	svc := &mockLoginTemplateService{deleteFn: func(_ uuid.UUID, _ int64) (*service.LoginTemplateServiceDataResult, error) {
		return nil, errors.New("fail")
	}}
	w := httptest.NewRecorder()
	NewLoginTemplateHandler(svc).Delete(w,
		withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "login_template_uuid", testResourceUUID.String())))
	assert.Equal(t, http.StatusBadRequest, w.Code)
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

func TestLoginTemplateHandler_UpdateStatus_NoTenant(t *testing.T) {
	w := httptest.NewRecorder()
	r := withUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "login_template_uuid", testResourceUUID.String()))
	NewLoginTemplateHandler(&mockLoginTemplateService{}).UpdateStatus(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginTemplateHandler_UpdateStatus_InvalidUUID(t *testing.T) {
	w := httptest.NewRecorder()
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "login_template_uuid", "bad"))
	NewLoginTemplateHandler(&mockLoginTemplateService{}).UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginTemplateHandler_UpdateStatus_BadJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPatch, "/"), "login_template_uuid", testResourceUUID.String()))
	NewLoginTemplateHandler(&mockLoginTemplateService{}).UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginTemplateHandler_UpdateStatus_ValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "invalid"}), "login_template_uuid", testResourceUUID.String()))
	NewLoginTemplateHandler(&mockLoginTemplateService{}).UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginTemplateHandler_UpdateStatus_ServiceError(t *testing.T) {
	svc := &mockLoginTemplateService{updateStatusFn: func(_ uuid.UUID, _ int64, _ string) (*service.LoginTemplateServiceDataResult, error) {
		return nil, errors.New("fail")
	}}
	w := httptest.NewRecorder()
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "login_template_uuid", testResourceUUID.String()))
	NewLoginTemplateHandler(svc).UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
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
