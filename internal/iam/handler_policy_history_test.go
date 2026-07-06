package iam

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func policyHistoryEntry() *PolicyHistoryEntryResult {
	return &PolicyHistoryEntryResult{
		UUID:          uuid.New(),
		VersionNumber: 1,
		Name:          "storage-writer",
		Document:      datatypes.JSON([]byte(`{"version":"v1","statement":[]}`)),
		PolicyVersion: "v1",
		SnapshotAt:    time.Now(),
	}
}

// ---------------------------------------------------------------------------
// ListHistory — GET /policies/{policy_uuid}/history
// ---------------------------------------------------------------------------

func TestPolicyHistoryHandler_ListHistory_NoTenant(t *testing.T) {
	h := NewPolicyHistoryHandler(&mockPolicyService{})
	w := httptest.NewRecorder()
	h.ListHistory(w, httptest.NewRequest(http.MethodGet, "/policies/x/history", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPolicyHistoryHandler_ListHistory_InvalidUUID(t *testing.T) {
	h := NewPolicyHistoryHandler(&mockPolicyService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/policies/bad/history", nil))
	r = withChiParam(r, "policy_uuid", "not-a-uuid")
	w := httptest.NewRecorder()
	h.ListHistory(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHistoryHandler_ListHistory_ServiceError(t *testing.T) {
	svc := &mockPolicyService{getHistoryFn: func(uuid.UUID, int64, int, int) (*PolicyHistoryListResult, error) {
		return nil, apperror.NewNotFoundWithReason("policy not found")
	}}
	h := NewPolicyHistoryHandler(svc)
	id := uuid.New()
	r := withTenant(httptest.NewRequest(http.MethodGet, "/policies/"+id.String()+"/history", nil))
	r = withChiParam(r, "policy_uuid", id.String())
	w := httptest.NewRecorder()
	h.ListHistory(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPolicyHistoryHandler_ListHistory_Success(t *testing.T) {
	svc := &mockPolicyService{getHistoryFn: func(_ uuid.UUID, _ int64, page, limit int) (*PolicyHistoryListResult, error) {
		assert.Equal(t, 2, page)
		assert.Equal(t, 5, limit)
		return &PolicyHistoryListResult{
			Data:       []PolicyHistoryEntryResult{*policyHistoryEntry()},
			Total:      1,
			Page:       2,
			Limit:      5,
			TotalPages: 1,
		}, nil
	}}
	h := NewPolicyHistoryHandler(svc)
	id := uuid.New()
	r := withTenant(httptest.NewRequest(http.MethodGet, "/policies/"+id.String()+"/history?page=2&limit=5", nil))
	r = withChiParam(r, "policy_uuid", id.String())
	w := httptest.NewRecorder()
	h.ListHistory(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "storage-writer")
}

// ---------------------------------------------------------------------------
// GetHistoryVersion — GET /policies/{policy_uuid}/history/{version_number}
// ---------------------------------------------------------------------------

func TestPolicyHistoryHandler_GetVersion_NoTenant(t *testing.T) {
	h := NewPolicyHistoryHandler(&mockPolicyService{})
	w := httptest.NewRecorder()
	h.GetHistoryVersion(w, httptest.NewRequest(http.MethodGet, "/policies/x/history/1", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPolicyHistoryHandler_GetVersion_InvalidUUID(t *testing.T) {
	h := NewPolicyHistoryHandler(&mockPolicyService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/policies/bad/history/1", nil))
	r = withChiParam(r, "policy_uuid", "not-a-uuid")
	r = withChiParam(r, "version_number", "1")
	w := httptest.NewRecorder()
	h.GetHistoryVersion(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHistoryHandler_GetVersion_InvalidVersion(t *testing.T) {
	h := NewPolicyHistoryHandler(&mockPolicyService{})
	id := uuid.New()
	r := withTenant(httptest.NewRequest(http.MethodGet, "/policies/"+id.String()+"/history/abc", nil))
	r = withChiParam(r, "policy_uuid", id.String())
	r = withChiParam(r, "version_number", "abc")
	w := httptest.NewRecorder()
	h.GetHistoryVersion(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHistoryHandler_GetVersion_NotFound(t *testing.T) {
	svc := &mockPolicyService{getHistoryVersionFn: func(uuid.UUID, int64, int) (*PolicyHistoryEntryResult, error) {
		return nil, apperror.NewNotFoundWithReason("policy version not found")
	}}
	h := NewPolicyHistoryHandler(svc)
	id := uuid.New()
	r := withTenant(httptest.NewRequest(http.MethodGet, "/policies/"+id.String()+"/history/9", nil))
	r = withChiParam(r, "policy_uuid", id.String())
	r = withChiParam(r, "version_number", "9")
	w := httptest.NewRecorder()
	h.GetHistoryVersion(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPolicyHistoryHandler_GetVersion_Success(t *testing.T) {
	svc := &mockPolicyService{getHistoryVersionFn: func(_ uuid.UUID, _ int64, version int) (*PolicyHistoryEntryResult, error) {
		assert.Equal(t, 3, version)
		return policyHistoryEntry(), nil
	}}
	h := NewPolicyHistoryHandler(svc)
	id := uuid.New()
	r := withTenant(httptest.NewRequest(http.MethodGet, "/policies/"+id.String()+"/history/3", nil))
	r = withChiParam(r, "policy_uuid", id.String())
	r = withChiParam(r, "version_number", "3")
	w := httptest.NewRecorder()
	h.GetHistoryVersion(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	// The detail response includes the full document for diff/rollback.
	assert.Contains(t, w.Body.String(), "statement")
}
