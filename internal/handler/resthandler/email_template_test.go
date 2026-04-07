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

// ---------------------------------------------------------------------------
// GetAll — missing branches
// ---------------------------------------------------------------------------

func TestEmailTemplateHandler_GetAll_ValidationError(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/email-templates?status=bad_status", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailTemplateHandler_GetAll_WithFiltersAndRows(t *testing.T) {
	// Covers status/is_default/is_system query param branches and toEmailTemplateListResponseDTO loop body
	svc := &mockEmailTemplateService{
		getAllFn: func(tid int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*service.EmailTemplateServiceListResult, error) {
			return &service.EmailTemplateServiceListResult{
				Data: []service.EmailTemplateServiceDataResult{{Name: "t1"}},
			}, nil
		},
	}
	h := NewEmailTemplateHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/email-templates?page=1&limit=10&status=active&is_default=true&is_system=false", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Get — missing success branch
// ---------------------------------------------------------------------------

func TestEmailTemplateHandler_Get_Success(t *testing.T) {
	svc := &mockEmailTemplateService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.EmailTemplateServiceDataResult, error) {
			return &service.EmailTemplateServiceDataResult{Name: "tmpl1"}, nil
		},
	}
	h := NewEmailTemplateHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/email-templates/"+testResourceUUID.String(), nil), "email_template_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Create — missing branches
// ---------------------------------------------------------------------------

func TestEmailTemplateHandler_Create_BadJSON(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := withTenant(badJSONReq(t, http.MethodPost, "/email-templates"))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailTemplateHandler_Create_ValidationError(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := withTenant(jsonReq(t, http.MethodPost, "/email-templates", map[string]any{}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailTemplateHandler_Create_WithCustomStatus(t *testing.T) {
	// Covers the req.Status != nil branch (custom status path)
	svc := &mockEmailTemplateService{
		createFn: func(tid int64, name, subject, bodyHTML string, bodyPlain *string, status string, isDefault bool) (*service.EmailTemplateServiceDataResult, error) {
			return &service.EmailTemplateServiceDataResult{Name: name, Status: status}, nil
		},
	}
	h := NewEmailTemplateHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPost, "/email-templates", map[string]any{
		"name": "tmpl1", "subject": "Hello", "body_html": "<p>hi</p>", "status": "inactive",
	}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestEmailTemplateHandler_Create_Success(t *testing.T) {
	svc := &mockEmailTemplateService{
		createFn: func(tid int64, name, subject, bodyHTML string, bodyPlain *string, status string, isDefault bool) (*service.EmailTemplateServiceDataResult, error) {
			return &service.EmailTemplateServiceDataResult{Name: name}, nil
		},
	}
	h := NewEmailTemplateHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPost, "/email-templates", map[string]any{
		"name": "tmpl1", "subject": "Hello", "body_html": "<p>hi</p>",
	}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ---------------------------------------------------------------------------
// Update — full coverage (0%)
// ---------------------------------------------------------------------------

func TestEmailTemplateHandler_Update_NoTenant(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"name": "n"}), "email_template_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestEmailTemplateHandler_Update_InvalidUUID(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"name": "n"}), "email_template_uuid", "bad"))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailTemplateHandler_Update_BadJSON(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := withTenant(withChiParam(badJSONReq(t, http.MethodPut, "/"), "email_template_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailTemplateHandler_Update_ValidationError(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{}), "email_template_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailTemplateHandler_Update_ServiceError(t *testing.T) {
	svc := &mockEmailTemplateService{
		updateFn: func(id uuid.UUID, tid int64, name, subject, bodyHTML string, bodyPlain *string, status string) (*service.EmailTemplateServiceDataResult, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewEmailTemplateHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{
		"name": "tmpl1", "subject": "Hello", "body_html": "<p>hi</p>",
	}), "email_template_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailTemplateHandler_Update_WithCustomStatus(t *testing.T) {
	// Covers the req.Status != nil branch
	svc := &mockEmailTemplateService{
		updateFn: func(id uuid.UUID, tid int64, name, subject, bodyHTML string, bodyPlain *string, status string) (*service.EmailTemplateServiceDataResult, error) {
			return &service.EmailTemplateServiceDataResult{Name: name, Status: status}, nil
		},
	}
	h := NewEmailTemplateHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{
		"name": "tmpl1", "subject": "Hello", "body_html": "<p>hi</p>", "status": "inactive",
	}), "email_template_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEmailTemplateHandler_Update_Success(t *testing.T) {
	svc := &mockEmailTemplateService{
		updateFn: func(id uuid.UUID, tid int64, name, subject, bodyHTML string, bodyPlain *string, status string) (*service.EmailTemplateServiceDataResult, error) {
			return &service.EmailTemplateServiceDataResult{Name: name}, nil
		},
	}
	h := NewEmailTemplateHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{
		"name": "tmpl1", "subject": "Hello", "body_html": "<p>hi</p>",
	}), "email_template_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Delete — missing branches
// ---------------------------------------------------------------------------

func TestEmailTemplateHandler_Delete_NoTenant(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "email_template_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestEmailTemplateHandler_Delete_ServiceError(t *testing.T) {
	svc := &mockEmailTemplateService{
		deleteFn: func(id uuid.UUID, tid int64) (*service.EmailTemplateServiceDataResult, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewEmailTemplateHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "email_template_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// UpdateStatus — full coverage (0%)
// ---------------------------------------------------------------------------

func TestEmailTemplateHandler_UpdateStatus_NoTenant(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "email_template_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestEmailTemplateHandler_UpdateStatus_InvalidUUID(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "email_template_uuid", "bad"))
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailTemplateHandler_UpdateStatus_BadJSON(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := withTenant(withChiParam(badJSONReq(t, http.MethodPatch, "/"), "email_template_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailTemplateHandler_UpdateStatus_ValidationError(t *testing.T) {
	h := NewEmailTemplateHandler(&mockEmailTemplateService{})
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "invalid"}), "email_template_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailTemplateHandler_UpdateStatus_ServiceError(t *testing.T) {
	svc := &mockEmailTemplateService{
		updateStatusFn: func(id uuid.UUID, tid int64, status string) (*service.EmailTemplateServiceDataResult, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewEmailTemplateHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "email_template_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailTemplateHandler_UpdateStatus_Success(t *testing.T) {
	svc := &mockEmailTemplateService{
		updateStatusFn: func(id uuid.UUID, tid int64, status string) (*service.EmailTemplateServiceDataResult, error) {
			return &service.EmailTemplateServiceDataResult{Status: status}, nil
		},
	}
	h := NewEmailTemplateHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "email_template_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
