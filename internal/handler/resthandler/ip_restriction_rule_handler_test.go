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

func ipRuleResult() service.IpRestrictionRuleServiceDataResult {
	return service.IpRestrictionRuleServiceDataResult{IpRestrictionRuleUUID: testResourceUUID, Description: "rule", Type: "allow", IpAddress: "1.2.3.4", Status: "active"}
}

func TestIPRestrictionRuleHandler_GetAll(t *testing.T) {
	svc := &mockIpRestrictionRuleService{getAllFn: func(_ int64, _ *string, _ []string, _, _ *string, _, _ int, _, _ string) (*service.IpRestrictionRuleServiceListResult, error) {
		return &service.IpRestrictionRuleServiceListResult{Data: []service.IpRestrictionRuleServiceDataResult{ipRuleResult()}}, nil
	}}
	h := NewIPRestrictionRuleHandler(svc)
	w := httptest.NewRecorder()
	h.GetAll(w, withTenant(jsonReq(t, http.MethodGet, "/ip-rules?page=1&limit=10", nil)))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRestrictionRuleHandler_GetAll_Error(t *testing.T) {
	svc := &mockIpRestrictionRuleService{getAllFn: func(_ int64, _ *string, _ []string, _, _ *string, _, _ int, _, _ string) (*service.IpRestrictionRuleServiceListResult, error) {
		return nil, errors.New("db")
	}}
	h := NewIPRestrictionRuleHandler(svc)
	w := httptest.NewRecorder()
	h.GetAll(w, withTenant(jsonReq(t, http.MethodGet, "/ip-rules?page=1&limit=10", nil)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestIPRestrictionRuleHandler_Get(t *testing.T) {
	res := ipRuleResult()
	svc := &mockIpRestrictionRuleService{getByUUIDFn: func(_ int64, _ uuid.UUID) (*service.IpRestrictionRuleServiceDataResult, error) { return &res, nil }}
	h := NewIPRestrictionRuleHandler(svc)
	r := withChiParam(withTenant(jsonReq(t, http.MethodGet, "/ip-rules/id", nil)), "ip_restriction_rule_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRestrictionRuleHandler_Get_BadUUID(t *testing.T) {
	h := NewIPRestrictionRuleHandler(&mockIpRestrictionRuleService{})
	r := withChiParam(withTenant(jsonReq(t, http.MethodGet, "/ip-rules/bad", nil)), "ip_restriction_rule_uuid", "not-a-uuid")
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIPRestrictionRuleHandler_Get_NotFound(t *testing.T) {
	svc := &mockIpRestrictionRuleService{getByUUIDFn: func(_ int64, _ uuid.UUID) (*service.IpRestrictionRuleServiceDataResult, error) {
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
	svc := &mockIpRestrictionRuleService{createFn: func(_ int64, _, _, _, _ string, _ int64) (*service.IpRestrictionRuleServiceDataResult, error) {
		return &res, nil
	}}
	h := NewIPRestrictionRuleHandler(svc)
	body := map[string]any{"type": "allow", "ip_address": "1.2.3.4", "description": "rule"}
	w := httptest.NewRecorder()
	h.Create(w, withTenantAndUser(jsonReq(t, http.MethodPost, "/ip-rules", body)))
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestIPRestrictionRuleHandler_Create_Error(t *testing.T) {
	svc := &mockIpRestrictionRuleService{createFn: func(_ int64, _, _, _, _ string, _ int64) (*service.IpRestrictionRuleServiceDataResult, error) {
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
	svc := &mockIpRestrictionRuleService{updateFn: func(_ int64, _ uuid.UUID, _, _, _, _ string, _ int64) (*service.IpRestrictionRuleServiceDataResult, error) {
		return &res, nil
	}}
	h := NewIPRestrictionRuleHandler(svc)
	body := map[string]any{"type": "deny", "ip_address": "1.2.3.4", "description": "upd"}
	r := withChiParam(withTenantAndUser(jsonReq(t, http.MethodPut, "/ip-rules/id", body)), "ip_restriction_rule_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRestrictionRuleHandler_Delete(t *testing.T) {
	res := ipRuleResult()
	svc := &mockIpRestrictionRuleService{deleteFn: func(_ int64, _ uuid.UUID) (*service.IpRestrictionRuleServiceDataResult, error) { return &res, nil }}
	h := NewIPRestrictionRuleHandler(svc)
	r := withChiParam(withTenant(jsonReq(t, http.MethodDelete, "/ip-rules/id", nil)), "ip_restriction_rule_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRestrictionRuleHandler_UpdateStatus(t *testing.T) {
	res := ipRuleResult()
	svc := &mockIpRestrictionRuleService{updateStatusFn: func(_ int64, _ uuid.UUID, _ string, _ int64) (*service.IpRestrictionRuleServiceDataResult, error) {
		return &res, nil
	}}
	h := NewIPRestrictionRuleHandler(svc)
	r := withChiParam(withTenantAndUser(jsonReq(t, http.MethodPatch, "/ip-rules/id/status", map[string]any{"status": "inactive"})), "ip_restriction_rule_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
