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

func smsResult() service.SmsTemplateServiceDataResult {
	return service.SmsTemplateServiceDataResult{SmsTemplateUUID: testResourceUUID, Name: "tmpl", Message: "Hi", Status: "active"}
}

func TestSMSTemplateHandler_GetAll(t *testing.T) {
	svc := &mockSmsTemplateService{getAllFn: func(_ int64, _ *string, _ []string, _, _ *bool, _, _ int, _, _ string) (*service.SmsTemplateServiceListResult, error) {
		return &service.SmsTemplateServiceListResult{Data: []service.SmsTemplateServiceDataResult{smsResult()}}, nil
	}}
	h := NewSMSTemplateHandler(svc)
	w := httptest.NewRecorder()
	h.GetAll(w, withTenant(jsonReq(t, http.MethodGet, "/sms-templates?page=1&limit=10", nil)))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSMSTemplateHandler_GetAll_Error(t *testing.T) {
	svc := &mockSmsTemplateService{getAllFn: func(_ int64, _ *string, _ []string, _, _ *bool, _, _ int, _, _ string) (*service.SmsTemplateServiceListResult, error) {
		return nil, errors.New("db")
	}}
	h := NewSMSTemplateHandler(svc)
	w := httptest.NewRecorder()
	h.GetAll(w, withTenant(jsonReq(t, http.MethodGet, "/sms-templates?page=1&limit=10", nil)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSMSTemplateHandler_Get(t *testing.T) {
	res := smsResult()
	svc := &mockSmsTemplateService{getByUUIDFn: func(_ uuid.UUID, _ int64) (*service.SmsTemplateServiceDataResult, error) { return &res, nil }}
	h := NewSMSTemplateHandler(svc)
	w := httptest.NewRecorder()
	h.Get(w, withChiParam(withTenant(jsonReq(t, http.MethodGet, "/sms-templates/id", nil)), "sms_template_uuid", testResourceUUID.String()))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSMSTemplateHandler_Get_BadUUID(t *testing.T) {
	h := NewSMSTemplateHandler(&mockSmsTemplateService{})
	w := httptest.NewRecorder()
	h.Get(w, withChiParam(withTenant(jsonReq(t, http.MethodGet, "/sms-templates/bad", nil)), "sms_template_uuid", "not-a-uuid"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSMSTemplateHandler_Get_NotFound(t *testing.T) {
	svc := &mockSmsTemplateService{getByUUIDFn: func(_ uuid.UUID, _ int64) (*service.SmsTemplateServiceDataResult, error) {
		return nil, errors.New("not found")
	}}
	h := NewSMSTemplateHandler(svc)
	w := httptest.NewRecorder()
	h.Get(w, withChiParam(withTenant(jsonReq(t, http.MethodGet, "/sms-templates/id", nil)), "sms_template_uuid", testResourceUUID.String()))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSMSTemplateHandler_Create(t *testing.T) {
	res := smsResult()
	svc := &mockSmsTemplateService{createFn: func(_ int64, _ string, _ *string, _ string, _ *string, _ string) (*service.SmsTemplateServiceDataResult, error) {
		return &res, nil
	}}
	h := NewSMSTemplateHandler(svc)
	body := map[string]any{"name": "tmpl1", "message": "Hello"}
	w := httptest.NewRecorder()
	h.Create(w, withTenant(jsonReq(t, http.MethodPost, "/sms-templates", body)))
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestSMSTemplateHandler_Create_Error(t *testing.T) {
	svc := &mockSmsTemplateService{createFn: func(_ int64, _ string, _ *string, _ string, _ *string, _ string) (*service.SmsTemplateServiceDataResult, error) {
		return nil, errors.New("fail")
	}}
	h := NewSMSTemplateHandler(svc)
	body := map[string]any{"name": "tmpl1", "message": "Hello"}
	w := httptest.NewRecorder()
	h.Create(w, withTenant(jsonReq(t, http.MethodPost, "/sms-templates", body)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSMSTemplateHandler_Update(t *testing.T) {
	res := smsResult()
	svc := &mockSmsTemplateService{updateFn: func(_ uuid.UUID, _ int64, _ string, _ *string, _ string, _ *string, _ string) (*service.SmsTemplateServiceDataResult, error) {
		return &res, nil
	}}
	h := NewSMSTemplateHandler(svc)
	body := map[string]any{"name": "upd", "message": "Hi"}
	r := withChiParam(withTenant(jsonReq(t, http.MethodPut, "/sms-templates/id", body)), "sms_template_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSMSTemplateHandler_Delete(t *testing.T) {
	res := smsResult()
	svc := &mockSmsTemplateService{deleteFn: func(_ uuid.UUID, _ int64) (*service.SmsTemplateServiceDataResult, error) { return &res, nil }}
	h := NewSMSTemplateHandler(svc)
	r := withChiParam(withTenant(jsonReq(t, http.MethodDelete, "/sms-templates/id", nil)), "sms_template_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSMSTemplateHandler_UpdateStatus(t *testing.T) {
	res := smsResult()
	svc := &mockSmsTemplateService{updateStatusFn: func(_ uuid.UUID, _ int64, _ string) (*service.SmsTemplateServiceDataResult, error) { return &res, nil }}
	h := NewSMSTemplateHandler(svc)
	r := withChiParam(withTenant(jsonReq(t, http.MethodPatch, "/sms-templates/id/status", map[string]any{"status": "inactive"})), "sms_template_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
