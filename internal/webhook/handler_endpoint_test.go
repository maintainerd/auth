package webhook

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func webhookResult() *WebhookEndpointServiceDataResult {
	return &WebhookEndpointServiceDataResult{
		WebhookEndpointUUID: uuid.New(),
		TenantID:            testTenantID,
		URL:                 "https://example.com/hook",
		SubscribeAll:        true,
		MaxRetries:          3,
		TimeoutSeconds:      30,
		Status:              "active",
	}
}

// ---------------------------------------------------------------------------
// GetAll
// ---------------------------------------------------------------------------

func TestWebhookEndpointHandler_GetAll_NoTenant(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	w := httptest.NewRecorder()
	h.GetAll(w, httptest.NewRequest(http.MethodGet, "/webhook-endpoints", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWebhookEndpointHandler_GetAll_ValidationError(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	// Invalid sort_order triggers DTO validation error
	r := withTenant(httptest.NewRequest(http.MethodGet, "/webhook-endpoints?sort_order=invalid", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookEndpointHandler_GetAll_ServiceError(t *testing.T) {
	svc := &mockWebhookEndpointService{
		getAllFn: func(_ int64, _ []string, _, _ int, _, _ string) (*WebhookEndpointServiceListResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewWebhookEndpointHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/webhook-endpoints?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestWebhookEndpointHandler_GetAll_Success(t *testing.T) {
	svc := &mockWebhookEndpointService{
		getAllFn: func(_ int64, _ []string, _, _ int, _, _ string) (*WebhookEndpointServiceListResult, error) {
			return &WebhookEndpointServiceListResult{
				Data:       []WebhookEndpointServiceDataResult{*webhookResult()},
				Total:      1,
				Page:       1,
				Limit:      10,
				TotalPages: 1,
			}, nil
		},
	}
	h := NewWebhookEndpointHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/webhook-endpoints?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWebhookEndpointHandler_GetAll_WithStatusFilter(t *testing.T) {
	svc := &mockWebhookEndpointService{
		getAllFn: func(_ int64, status []string, _, _ int, _, _ string) (*WebhookEndpointServiceListResult, error) {
			assert.Equal(t, []string{"active"}, status)
			return &WebhookEndpointServiceListResult{}, nil
		},
	}
	h := NewWebhookEndpointHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/webhook-endpoints?page=1&limit=10&status=active", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestWebhookEndpointHandler_Get_NoTenant(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	w := httptest.NewRecorder()
	h.Get(w, httptest.NewRequest(http.MethodGet, "/webhook-endpoints/abc", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWebhookEndpointHandler_Get_InvalidUUID(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/webhook-endpoints/not-a-uuid", nil))
	r = withChiParam(r, "webhook_endpoint_uuid", "not-a-uuid")
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookEndpointHandler_Get_ServiceError(t *testing.T) {
	svc := &mockWebhookEndpointService{
		getByUUIDFn: func(_ int64, _ uuid.UUID) (*WebhookEndpointServiceDataResult, error) {
			return nil, errNotFound
		},
	}
	h := NewWebhookEndpointHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/webhook-endpoints/"+testResourceUUID.String(), nil))
	r = withChiParam(r, "webhook_endpoint_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWebhookEndpointHandler_Get_Success(t *testing.T) {
	svc := &mockWebhookEndpointService{
		getByUUIDFn: func(_ int64, _ uuid.UUID) (*WebhookEndpointServiceDataResult, error) {
			return webhookResult(), nil
		},
	}
	h := NewWebhookEndpointHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/webhook-endpoints/"+testResourceUUID.String(), nil))
	r = withChiParam(r, "webhook_endpoint_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWebhookEndpointHandler_Get_SuccessWithLastTriggered(t *testing.T) {
	now := time.Now()
	result := webhookResult()
	result.LastTriggeredAt = &now
	svc := &mockWebhookEndpointService{
		getByUUIDFn: func(_ int64, _ uuid.UUID) (*WebhookEndpointServiceDataResult, error) {
			return result, nil
		},
	}
	h := NewWebhookEndpointHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/webhook-endpoints/"+testResourceUUID.String(), nil))
	r = withChiParam(r, "webhook_endpoint_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), now.Format("2006-01-02"))
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestWebhookEndpointHandler_Create_NoTenant(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/webhook-endpoints", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWebhookEndpointHandler_Create_BadJSON(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	w := httptest.NewRecorder()
	h.Create(w, withTenant(badJSONReq(t, http.MethodPost, "/webhook-endpoints")))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookEndpointHandler_Create_ValidationError(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	body := map[string]any{"description": "missing url & events"}
	w := httptest.NewRecorder()
	h.Create(w, withTenant(jsonReq(t, http.MethodPost, "/webhook-endpoints", body)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookEndpointHandler_Create_ServiceError(t *testing.T) {
	svc := &mockWebhookEndpointService{
		createFn: func(_ int64, _ string, _ bool, _, _ *int, _, _ string) (*WebhookEndpointServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewWebhookEndpointHandler(svc)
	body := map[string]any{"url": "https://example.com/hook", "subscribe_all": true}
	w := httptest.NewRecorder()
	h.Create(w, withTenant(jsonReq(t, http.MethodPost, "/webhook-endpoints", body)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestWebhookEndpointHandler_Create_Success(t *testing.T) {
	svc := &mockWebhookEndpointService{
		createFn: func(_ int64, _ string, _ bool, _, _ *int, _, _ string) (*WebhookEndpointServiceDataResult, error) {
			return webhookResult(), nil
		},
	}
	h := NewWebhookEndpointHandler(svc)
	body := map[string]any{"url": "https://example.com/hook", "subscribe_all": true}
	w := httptest.NewRecorder()
	h.Create(w, withTenant(jsonReq(t, http.MethodPost, "/webhook-endpoints", body)))
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestWebhookEndpointHandler_Create_WithExplicitStatus(t *testing.T) {
	svc := &mockWebhookEndpointService{
		createFn: func(_ int64, _ string, _ bool, _, _ *int, _, status string) (*WebhookEndpointServiceDataResult, error) {
			assert.Equal(t, "inactive", status)
			return webhookResult(), nil
		},
	}
	h := NewWebhookEndpointHandler(svc)
	body := map[string]any{"url": "https://example.com/hook", "events": []string{"user.created"}, "status": "inactive"}
	w := httptest.NewRecorder()
	h.Create(w, withTenant(jsonReq(t, http.MethodPost, "/webhook-endpoints", body)))
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestWebhookEndpointHandler_Update_NoTenant(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	w := httptest.NewRecorder()
	h.Update(w, httptest.NewRequest(http.MethodPut, "/webhook-endpoints/abc", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWebhookEndpointHandler_Update_InvalidUUID(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	r := withTenant(httptest.NewRequest(http.MethodPut, "/webhook-endpoints/bad", nil))
	r = withChiParam(r, "webhook_endpoint_uuid", "bad")
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookEndpointHandler_Update_BadJSON(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	r := withTenant(badJSONReq(t, http.MethodPut, "/webhook-endpoints/"+testResourceUUID.String()))
	r = withChiParam(r, "webhook_endpoint_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookEndpointHandler_Update_ValidationError(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	body := map[string]any{"description": "missing url & events"}
	r := withTenant(jsonReq(t, http.MethodPut, "/webhook-endpoints/"+testResourceUUID.String(), body))
	r = withChiParam(r, "webhook_endpoint_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookEndpointHandler_Update_ServiceError(t *testing.T) {
	svc := &mockWebhookEndpointService{
		updateFn: func(_ int64, _ uuid.UUID, _ string, _ bool, _ bool, _, _ *int, _, _ string) (*WebhookEndpointServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewWebhookEndpointHandler(svc)
	body := map[string]any{"url": "https://example.com/hook", "subscribe_all": true}
	r := withTenant(jsonReq(t, http.MethodPut, "/webhook-endpoints/"+testResourceUUID.String(), body))
	r = withChiParam(r, "webhook_endpoint_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestWebhookEndpointHandler_Update_Success(t *testing.T) {
	svc := &mockWebhookEndpointService{
		updateFn: func(_ int64, _ uuid.UUID, _ string, _ bool, _ bool, _, _ *int, _, _ string) (*WebhookEndpointServiceDataResult, error) {
			return webhookResult(), nil
		},
	}
	h := NewWebhookEndpointHandler(svc)
	body := map[string]any{"url": "https://example.com/hook", "subscribe_all": true}
	r := withTenant(jsonReq(t, http.MethodPut, "/webhook-endpoints/"+testResourceUUID.String(), body))
	r = withChiParam(r, "webhook_endpoint_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWebhookEndpointHandler_Update_WithExplicitStatus(t *testing.T) {
	svc := &mockWebhookEndpointService{
		updateFn: func(_ int64, _ uuid.UUID, _ string, _ bool, _ bool, _, _ *int, _, status string) (*WebhookEndpointServiceDataResult, error) {
			assert.Equal(t, "inactive", status)
			return webhookResult(), nil
		},
	}
	h := NewWebhookEndpointHandler(svc)
	body := map[string]any{"url": "https://example.com/hook", "events": []string{"user.created"}, "status": "inactive"}
	r := withTenant(jsonReq(t, http.MethodPut, "/webhook-endpoints/"+testResourceUUID.String(), body))
	r = withChiParam(r, "webhook_endpoint_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestWebhookEndpointHandler_Delete_NoTenant(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	w := httptest.NewRecorder()
	h.Delete(w, httptest.NewRequest(http.MethodDelete, "/webhook-endpoints/abc", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWebhookEndpointHandler_Delete_InvalidUUID(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	r := withTenant(httptest.NewRequest(http.MethodDelete, "/webhook-endpoints/bad", nil))
	r = withChiParam(r, "webhook_endpoint_uuid", "bad")
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookEndpointHandler_Delete_ServiceError(t *testing.T) {
	svc := &mockWebhookEndpointService{
		deleteFn: func(_ int64, _ uuid.UUID) (*WebhookEndpointServiceDataResult, error) {
			return nil, errNotFound
		},
	}
	h := NewWebhookEndpointHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodDelete, "/webhook-endpoints/"+testResourceUUID.String(), nil))
	r = withChiParam(r, "webhook_endpoint_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWebhookEndpointHandler_Delete_Success(t *testing.T) {
	svc := &mockWebhookEndpointService{
		deleteFn: func(_ int64, _ uuid.UUID) (*WebhookEndpointServiceDataResult, error) {
			return webhookResult(), nil
		},
	}
	h := NewWebhookEndpointHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodDelete, "/webhook-endpoints/"+testResourceUUID.String(), nil))
	r = withChiParam(r, "webhook_endpoint_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// UpdateStatus
// ---------------------------------------------------------------------------

func TestWebhookEndpointHandler_UpdateStatus_NoTenant(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	w := httptest.NewRecorder()
	h.UpdateStatus(w, httptest.NewRequest(http.MethodPatch, "/webhook-endpoints/abc/status", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWebhookEndpointHandler_UpdateStatus_InvalidUUID(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	r := withTenant(httptest.NewRequest(http.MethodPatch, "/webhook-endpoints/bad/status", nil))
	r = withChiParam(r, "webhook_endpoint_uuid", "bad")
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookEndpointHandler_UpdateStatus_BadJSON(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	r := withTenant(badJSONReq(t, http.MethodPatch, "/webhook-endpoints/"+testResourceUUID.String()+"/status"))
	r = withChiParam(r, "webhook_endpoint_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookEndpointHandler_UpdateStatus_ValidationError(t *testing.T) {
	h := NewWebhookEndpointHandler(&mockWebhookEndpointService{})
	body := map[string]any{"status": "bogus"}
	r := withTenant(jsonReq(t, http.MethodPatch, "/webhook-endpoints/"+testResourceUUID.String()+"/status", body))
	r = withChiParam(r, "webhook_endpoint_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookEndpointHandler_UpdateStatus_ServiceError(t *testing.T) {
	svc := &mockWebhookEndpointService{
		updateStatusFn: func(_ int64, _ uuid.UUID, _ string) (*WebhookEndpointServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewWebhookEndpointHandler(svc)
	body := map[string]any{"status": "active"}
	r := withTenant(jsonReq(t, http.MethodPatch, "/webhook-endpoints/"+testResourceUUID.String()+"/status", body))
	r = withChiParam(r, "webhook_endpoint_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestWebhookEndpointHandler_UpdateStatus_Success(t *testing.T) {
	svc := &mockWebhookEndpointService{
		updateStatusFn: func(_ int64, _ uuid.UUID, _ string) (*WebhookEndpointServiceDataResult, error) {
			return webhookResult(), nil
		},
	}
	h := NewWebhookEndpointHandler(svc)
	body := map[string]any{"status": "active"}
	r := withTenant(jsonReq(t, http.MethodPatch, "/webhook-endpoints/"+testResourceUUID.String()+"/status", body))
	r = withChiParam(r, "webhook_endpoint_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
