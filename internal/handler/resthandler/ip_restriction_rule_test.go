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

func ipRuleResult() service.IPRestrictionRuleServiceDataResult {
	return service.IPRestrictionRuleServiceDataResult{IPRestrictionRuleUUID: testResourceUUID, Description: "rule", Type: "allow", IPAddress: "1.2.3.4", Status: "active"}
}

func TestIPRestrictionRuleHandler_GetAll(t *testing.T) {
	svc := &mockIPRestrictionRuleService{getAllFn: func(_ int64, _ *string, _ []string, _, _ *string, _, _ int, _, _ string) (*service.IPRestrictionRuleServiceListResult, error) {
		return &service.IPRestrictionRuleServiceListResult{Data: []service.IPRestrictionRuleServiceDataResult{ipRuleResult()}}, nil
	}}
	h := NewIPRestrictionRuleHandler(svc)
	w := httptest.NewRecorder()
	h.GetAll(w, withTenant(jsonReq(t, http.MethodGet, "/ip-rules?page=1&limit=10", nil)))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRestrictionRuleHandler_GetAll_NoTenant(t *testing.T) {
	w := httptest.NewRecorder()
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).GetAll(w, jsonReq(t, http.MethodGet, "/ip-rules", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIPRestrictionRuleHandler_GetAll_ValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).GetAll(w, withTenant(jsonReq(t, http.MethodGet, "/ip-rules?sort_order=invalid", nil)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIPRestrictionRuleHandler_GetAll_WithStatusFilter(t *testing.T) {
	// Covers the status != "" branch (line 51-53)
	svc := &mockIPRestrictionRuleService{getAllFn: func(_ int64, _ *string, _ []string, _, _ *string, _, _ int, _, _ string) (*service.IPRestrictionRuleServiceListResult, error) {
		return &service.IPRestrictionRuleServiceListResult{}, nil
	}}
	w := httptest.NewRecorder()
	NewIPRestrictionRuleHandler(svc).GetAll(w, withTenant(jsonReq(t, http.MethodGet, "/ip-rules?page=1&limit=10&status=active", nil)))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRestrictionRuleHandler_GetAll_Error(t *testing.T) {
	svc := &mockIPRestrictionRuleService{getAllFn: func(_ int64, _ *string, _ []string, _, _ *string, _, _ int, _, _ string) (*service.IPRestrictionRuleServiceListResult, error) {
		return nil, errors.New("db")
	}}
	h := NewIPRestrictionRuleHandler(svc)
	w := httptest.NewRecorder()
	h.GetAll(w, withTenant(jsonReq(t, http.MethodGet, "/ip-rules?page=1&limit=10", nil)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestIPRestrictionRuleHandler_Get_NoTenant(t *testing.T) {
	w := httptest.NewRecorder()
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).Get(w, jsonReq(t, http.MethodGet, "/ip-rules/id", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIPRestrictionRuleHandler_Get(t *testing.T) {
	res := ipRuleResult()
	svc := &mockIPRestrictionRuleService{getByUUIDFn: func(_ int64, _ uuid.UUID) (*service.IPRestrictionRuleServiceDataResult, error) { return &res, nil }}
	h := NewIPRestrictionRuleHandler(svc)
	r := withChiParam(withTenant(jsonReq(t, http.MethodGet, "/ip-rules/id", nil)), "ip_restriction_rule_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRestrictionRuleHandler_Get_BadUUID(t *testing.T) {
	h := NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{})
	r := withChiParam(withTenant(jsonReq(t, http.MethodGet, "/ip-rules/bad", nil)), "ip_restriction_rule_uuid", "not-a-uuid")
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIPRestrictionRuleHandler_Get_NotFound(t *testing.T) {
	svc := &mockIPRestrictionRuleService{getByUUIDFn: func(_ int64, _ uuid.UUID) (*service.IPRestrictionRuleServiceDataResult, error) {
		return nil, errors.New("not found")
	}}
	h := NewIPRestrictionRuleHandler(svc)
	r := withChiParam(withTenant(jsonReq(t, http.MethodGet, "/ip-rules/id", nil)), "ip_restriction_rule_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestIPRestrictionRuleHandler_Create(t *testing.T) {
	res := ipRuleResult()
	svc := &mockIPRestrictionRuleService{createFn: func(_ int64, _, _, _, _ string, _ int64) (*service.IPRestrictionRuleServiceDataResult, error) {
		return &res, nil
	}}
	h := NewIPRestrictionRuleHandler(svc)
	body := map[string]any{"type": "allow", "ip_address": "1.2.3.4", "description": "rule"}
	w := httptest.NewRecorder()
	h.Create(w, withTenantAndUser(jsonReq(t, http.MethodPost, "/ip-rules", body)))
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestIPRestrictionRuleHandler_Create_NoTenant(t *testing.T) {
	w := httptest.NewRecorder()
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).Create(w, withUser(jsonReq(t, http.MethodPost, "/ip-rules", nil)))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIPRestrictionRuleHandler_Create_NoUser(t *testing.T) {
	w := httptest.NewRecorder()
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).Create(w, withTenant(jsonReq(t, http.MethodPost, "/ip-rules", nil)))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIPRestrictionRuleHandler_Create_BadJSON(t *testing.T) {
	w := httptest.NewRecorder()
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).Create(w, withTenantAndUser(badJSONReq(t, http.MethodPost, "/ip-rules")))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIPRestrictionRuleHandler_Create_ValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).Create(w, withTenantAndUser(jsonReq(t, http.MethodPost, "/ip-rules", map[string]any{})))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIPRestrictionRuleHandler_Create_CustomStatus(t *testing.T) {
	// Covers the req.Status != nil branch (line 158-160)
	res := ipRuleResult()
	svc := &mockIPRestrictionRuleService{createFn: func(_ int64, _, _, _, _ string, _ int64) (*service.IPRestrictionRuleServiceDataResult, error) {
		return &res, nil
	}}
	body := map[string]any{"type": "allow", "ip_address": "1.2.3.4", "description": "rule", "status": "inactive"}
	w := httptest.NewRecorder()
	NewIPRestrictionRuleHandler(svc).Create(w, withTenantAndUser(jsonReq(t, http.MethodPost, "/ip-rules", body)))
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestIPRestrictionRuleHandler_Create_Error(t *testing.T) {
	svc := &mockIPRestrictionRuleService{createFn: func(_ int64, _, _, _, _ string, _ int64) (*service.IPRestrictionRuleServiceDataResult, error) {
		return nil, errors.New("fail")
	}}
	h := NewIPRestrictionRuleHandler(svc)
	body := map[string]any{"type": "allow", "ip_address": "1.2.3.4", "description": "rule"}
	w := httptest.NewRecorder()
	h.Create(w, withTenantAndUser(jsonReq(t, http.MethodPost, "/ip-rules", body)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIPRestrictionRuleHandler_Update(t *testing.T) {
	res := ipRuleResult()
	svc := &mockIPRestrictionRuleService{updateFn: func(_ int64, _ uuid.UUID, _, _, _, _ string, _ int64) (*service.IPRestrictionRuleServiceDataResult, error) {
		return &res, nil
	}}
	body := map[string]any{"type": "deny", "ip_address": "1.2.3.4", "description": "upd"}
	r := withChiParam(withTenantAndUser(jsonReq(t, http.MethodPut, "/ip-rules/id", body)), "ip_restriction_rule_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	NewIPRestrictionRuleHandler(svc).Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRestrictionRuleHandler_Update_NoTenant(t *testing.T) {
	w := httptest.NewRecorder()
	r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/", nil), "ip_restriction_rule_uuid", testResourceUUID.String()))
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).Update(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIPRestrictionRuleHandler_Update_NoUser(t *testing.T) {
	w := httptest.NewRecorder()
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/", nil), "ip_restriction_rule_uuid", testResourceUUID.String()))
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).Update(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIPRestrictionRuleHandler_Update_InvalidUUID(t *testing.T) {
	w := httptest.NewRecorder()
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", nil), "ip_restriction_rule_uuid", "bad"))
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIPRestrictionRuleHandler_Update_BadJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPut, "/"), "ip_restriction_rule_uuid", testResourceUUID.String()))
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIPRestrictionRuleHandler_Update_ValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{}), "ip_restriction_rule_uuid", testResourceUUID.String()))
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIPRestrictionRuleHandler_Update_CustomStatus(t *testing.T) {
	// Covers the req.Status != nil branch (lines 222-224)
	res := ipRuleResult()
	svc := &mockIPRestrictionRuleService{updateFn: func(_ int64, _ uuid.UUID, _, _, _, _ string, _ int64) (*service.IPRestrictionRuleServiceDataResult, error) {
		return &res, nil
	}}
	body := map[string]any{"type": "allow", "ip_address": "1.2.3.4", "status": "inactive"}
	w := httptest.NewRecorder()
	r := withChiParam(withTenantAndUser(jsonReq(t, http.MethodPut, "/", body)), "ip_restriction_rule_uuid", testResourceUUID.String())
	NewIPRestrictionRuleHandler(svc).Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRestrictionRuleHandler_Update_ServiceError(t *testing.T) {
	svc := &mockIPRestrictionRuleService{updateFn: func(_ int64, _ uuid.UUID, _, _, _, _ string, _ int64) (*service.IPRestrictionRuleServiceDataResult, error) {
		return nil, errors.New("fail")
	}}
	body := map[string]any{"type": "allow", "ip_address": "1.2.3.4"}
	w := httptest.NewRecorder()
	r := withChiParam(withTenantAndUser(jsonReq(t, http.MethodPut, "/", body)), "ip_restriction_rule_uuid", testResourceUUID.String())
	NewIPRestrictionRuleHandler(svc).Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIPRestrictionRuleHandler_Delete(t *testing.T) {
	res := ipRuleResult()
	svc := &mockIPRestrictionRuleService{deleteFn: func(_ int64, _ uuid.UUID) (*service.IPRestrictionRuleServiceDataResult, error) { return &res, nil }}
	r := withChiParam(withTenant(jsonReq(t, http.MethodDelete, "/ip-rules/id", nil)), "ip_restriction_rule_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	NewIPRestrictionRuleHandler(svc).Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRestrictionRuleHandler_Delete_NoTenant(t *testing.T) {
	w := httptest.NewRecorder()
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).Delete(w,
		withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "ip_restriction_rule_uuid", testResourceUUID.String()))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIPRestrictionRuleHandler_Delete_InvalidUUID(t *testing.T) {
	w := httptest.NewRecorder()
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).Delete(w,
		withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "ip_restriction_rule_uuid", "bad")))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIPRestrictionRuleHandler_Delete_ServiceError(t *testing.T) {
	svc := &mockIPRestrictionRuleService{deleteFn: func(_ int64, _ uuid.UUID) (*service.IPRestrictionRuleServiceDataResult, error) {
		return nil, errors.New("fail")
	}}
	w := httptest.NewRecorder()
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "ip_restriction_rule_uuid", testResourceUUID.String()))
	NewIPRestrictionRuleHandler(svc).Delete(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIPRestrictionRuleHandler_UpdateStatus(t *testing.T) {
	res := ipRuleResult()
	svc := &mockIPRestrictionRuleService{updateStatusFn: func(_ int64, _ uuid.UUID, _ string, _ int64) (*service.IPRestrictionRuleServiceDataResult, error) {
		return &res, nil
	}}
	r := withChiParam(withTenantAndUser(jsonReq(t, http.MethodPatch, "/ip-rules/id/status", map[string]any{"status": "inactive"})), "ip_restriction_rule_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	NewIPRestrictionRuleHandler(svc).UpdateStatus(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRestrictionRuleHandler_UpdateStatus_NoTenant(t *testing.T) {
	w := httptest.NewRecorder()
	r := withUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "ip_restriction_rule_uuid", testResourceUUID.String()))
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).UpdateStatus(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIPRestrictionRuleHandler_UpdateStatus_NoUser(t *testing.T) {
	w := httptest.NewRecorder()
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "ip_restriction_rule_uuid", testResourceUUID.String()))
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).UpdateStatus(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIPRestrictionRuleHandler_UpdateStatus_InvalidUUID(t *testing.T) {
	w := httptest.NewRecorder()
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "ip_restriction_rule_uuid", "bad"))
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIPRestrictionRuleHandler_UpdateStatus_BadJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPatch, "/"), "ip_restriction_rule_uuid", testResourceUUID.String()))
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIPRestrictionRuleHandler_UpdateStatus_ValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "invalid"}), "ip_restriction_rule_uuid", testResourceUUID.String()))
	NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}).UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIPRestrictionRuleHandler_UpdateStatus_ServiceError(t *testing.T) {
	svc := &mockIPRestrictionRuleService{updateStatusFn: func(_ int64, _ uuid.UUID, _ string, _ int64) (*service.IPRestrictionRuleServiceDataResult, error) {
		return nil, errors.New("fail")
	}}
	w := httptest.NewRecorder()
	r := withChiParam(withTenantAndUser(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})), "ip_restriction_rule_uuid", testResourceUUID.String())
	NewIPRestrictionRuleHandler(svc).UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
