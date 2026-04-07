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

func smsResult() service.SMSTemplateServiceDataResult {
	return service.SMSTemplateServiceDataResult{SMSTemplateUUID: testResourceUUID, Name: "tmpl", Message: "Hi", Status: "active"}
}

func TestSMSTemplateHandler_GetAll(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/sms-templates", nil)
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).GetAll(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/sms-templates?page=1&limit=10&sort_order=bad", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).GetAll(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockSMSTemplateService{getAllFn: func(_ int64, _ *string, _ []string, _, _ *bool, _, _ int, _, _ string) (*service.SMSTemplateServiceListResult, error) {
			return nil, errors.New("db")
		}}
		r := jsonReq(t, http.MethodGet, "/sms-templates?page=1&limit=10", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(svc).GetAll(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success with all filters and rows covers filter branches", func(t *testing.T) {
		svc := &mockSMSTemplateService{getAllFn: func(_ int64, _ *string, _ []string, _, _ *bool, _, _ int, _, _ string) (*service.SMSTemplateServiceListResult, error) {
			return &service.SMSTemplateServiceListResult{Data: []service.SMSTemplateServiceDataResult{smsResult()}}, nil
		}}
		r := jsonReq(t, http.MethodGet, "/sms-templates?page=1&limit=10&status=active&is_default=true&is_system=false", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(svc).GetAll(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSMSTemplateHandler_Get(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).Get(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withChiParam(withTenant(jsonReq(t, http.MethodGet, "/", nil)), "sms_template_uuid", "not-a-uuid")
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).Get(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		svc := &mockSMSTemplateService{getByUUIDFn: func(_ uuid.UUID, _ int64) (*service.SMSTemplateServiceDataResult, error) {
			return nil, errors.New("not found")
		}}
		r := withChiParam(withTenant(jsonReq(t, http.MethodGet, "/", nil)), "sms_template_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		res := smsResult()
		svc := &mockSMSTemplateService{getByUUIDFn: func(_ uuid.UUID, _ int64) (*service.SMSTemplateServiceDataResult, error) { return &res, nil }}
		r := withChiParam(withTenant(jsonReq(t, http.MethodGet, "/", nil)), "sms_template_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSMSTemplateHandler_Create(t *testing.T) {
	validBody := map[string]any{"name": "tmpl1", "message": "Hello"}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/sms-templates", validBody)
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).Create(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPost, "/sms-templates")
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/sms-templates", map[string]any{"name": ""})
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockSMSTemplateService{createFn: func(_ int64, _ string, _ *string, _ string, _ *string, _ string) (*service.SMSTemplateServiceDataResult, error) {
			return nil, errors.New("fail")
		}}
		r := jsonReq(t, http.MethodPost, "/sms-templates", validBody)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success with explicit status covers status != nil branch", func(t *testing.T) {
		res := smsResult()
		svc := &mockSMSTemplateService{createFn: func(_ int64, _ string, _ *string, _ string, _ *string, _ string) (*service.SMSTemplateServiceDataResult, error) {
			return &res, nil
		}}
		body := map[string]any{"name": "tmpl1", "message": "Hello", "status": "inactive"}
		r := jsonReq(t, http.MethodPost, "/sms-templates", body)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("success returns 201", func(t *testing.T) {
		res := smsResult()
		svc := &mockSMSTemplateService{createFn: func(_ int64, _ string, _ *string, _ string, _ *string, _ string) (*service.SMSTemplateServiceDataResult, error) {
			return &res, nil
		}}
		r := jsonReq(t, http.MethodPost, "/sms-templates", validBody)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestSMSTemplateHandler_Update(t *testing.T) {
	validBody := map[string]any{"name": "upd", "message": "Hi"}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", validBody)
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).Update(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withChiParam(withTenant(jsonReq(t, http.MethodPut, "/", validBody)), "sms_template_uuid", "bad-uuid")
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withChiParam(withTenant(badJSONReq(t, http.MethodPut, "/")), "sms_template_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withChiParam(withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{"name": ""})), "sms_template_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockSMSTemplateService{updateFn: func(_ uuid.UUID, _ int64, _ string, _ *string, _ string, _ *string, _ string) (*service.SMSTemplateServiceDataResult, error) {
			return nil, errors.New("update error")
		}}
		r := withChiParam(withTenant(jsonReq(t, http.MethodPut, "/", validBody)), "sms_template_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success with explicit status covers status != nil branch", func(t *testing.T) {
		res := smsResult()
		svc := &mockSMSTemplateService{updateFn: func(_ uuid.UUID, _ int64, _ string, _ *string, _ string, _ *string, _ string) (*service.SMSTemplateServiceDataResult, error) {
			return &res, nil
		}}
		body := map[string]any{"name": "upd", "message": "Hi", "status": "inactive"}
		r := withChiParam(withTenant(jsonReq(t, http.MethodPut, "/", body)), "sms_template_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		res := smsResult()
		svc := &mockSMSTemplateService{updateFn: func(_ uuid.UUID, _ int64, _ string, _ *string, _ string, _ *string, _ string) (*service.SMSTemplateServiceDataResult, error) {
			return &res, nil
		}}
		r := withChiParam(withTenant(jsonReq(t, http.MethodPut, "/", validBody)), "sms_template_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSMSTemplateHandler_Delete(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).Delete(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withChiParam(withTenant(jsonReq(t, http.MethodDelete, "/", nil)), "sms_template_uuid", "bad-uuid")
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).Delete(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockSMSTemplateService{deleteFn: func(_ uuid.UUID, _ int64) (*service.SMSTemplateServiceDataResult, error) {
			return nil, errors.New("delete error")
		}}
		r := withChiParam(withTenant(jsonReq(t, http.MethodDelete, "/", nil)), "sms_template_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		res := smsResult()
		svc := &mockSMSTemplateService{deleteFn: func(_ uuid.UUID, _ int64) (*service.SMSTemplateServiceDataResult, error) { return &res, nil }}
		r := withChiParam(withTenant(jsonReq(t, http.MethodDelete, "/", nil)), "sms_template_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSMSTemplateHandler_UpdateStatus(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).UpdateStatus(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withChiParam(withTenant(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})), "sms_template_uuid", "bad-uuid")
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).UpdateStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withChiParam(withTenant(badJSONReq(t, http.MethodPatch, "/")), "sms_template_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).UpdateStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withChiParam(withTenant(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "invalid"})), "sms_template_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(&mockSMSTemplateService{}).UpdateStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockSMSTemplateService{updateStatusFn: func(_ uuid.UUID, _ int64, _ string) (*service.SMSTemplateServiceDataResult, error) {
			return nil, errors.New("status error")
		}}
		r := withChiParam(withTenant(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})), "sms_template_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(svc).UpdateStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		res := smsResult()
		svc := &mockSMSTemplateService{updateStatusFn: func(_ uuid.UUID, _ int64, _ string) (*service.SMSTemplateServiceDataResult, error) { return &res, nil }}
		r := withChiParam(withTenant(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "inactive"})), "sms_template_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewSMSTemplateHandler(svc).UpdateStatus(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
