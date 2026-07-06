package federation

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// GetAll
// ---------------------------------------------------------------------------

func TestWIFHandler_GetAll_NoTenant(t *testing.T) {
	h := NewWorkloadIdentityFederationHandler(&mockWIFService{})
	w := httptest.NewRecorder()
	h.GetAll(w, httptest.NewRequest(http.MethodGet, "/workload-identity-federations", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWIFHandler_GetAll_ValidationError(t *testing.T) {
	h := NewWorkloadIdentityFederationHandler(&mockWIFService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/workload-identity-federations?sort_order=invalid", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWIFHandler_GetAll_ServiceError(t *testing.T) {
	svc := &mockWIFService{
		getAllFn: func(int64, int, int, string, string) (*WorkloadIdentityFederationServiceListResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewWorkloadIdentityFederationHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/workload-identity-federations?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestWIFHandler_GetAll_Success(t *testing.T) {
	svc := &mockWIFService{
		getAllFn: func(int64, int, int, string, string) (*WorkloadIdentityFederationServiceListResult, error) {
			return &WorkloadIdentityFederationServiceListResult{
				Data:       []WorkloadIdentityFederationServiceDataResult{*wifResult()},
				Total:      1,
				Page:       1,
				Limit:      10,
				TotalPages: 1,
			}, nil
		},
	}
	h := NewWorkloadIdentityFederationHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/workload-identity-federations?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "github-actions")
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestWIFHandler_Get_NoTenant(t *testing.T) {
	h := NewWorkloadIdentityFederationHandler(&mockWIFService{})
	w := httptest.NewRecorder()
	h.Get(w, httptest.NewRequest(http.MethodGet, "/workload-identity-federations/abc", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWIFHandler_Get_InvalidUUID(t *testing.T) {
	h := NewWorkloadIdentityFederationHandler(&mockWIFService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/workload-identity-federations/not-a-uuid", nil))
	r = withChiParam(r, "workload_identity_federation_uuid", "not-a-uuid")
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWIFHandler_Get_ServiceError(t *testing.T) {
	svc := &mockWIFService{
		getFn: func(int64, uuid.UUID) (*WorkloadIdentityFederationServiceDataResult, error) {
			return nil, errNotFound
		},
	}
	h := NewWorkloadIdentityFederationHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/workload-identity-federations/"+testResourceUUID.String(), nil))
	r = withChiParam(r, "workload_identity_federation_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWIFHandler_Get_Success(t *testing.T) {
	svc := &mockWIFService{
		getFn: func(_ int64, id uuid.UUID) (*WorkloadIdentityFederationServiceDataResult, error) {
			assert.Equal(t, testResourceUUID, id)
			return wifResult(), nil
		},
	}
	h := NewWorkloadIdentityFederationHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/workload-identity-federations/"+testResourceUUID.String(), nil))
	r = withChiParam(r, "workload_identity_federation_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestWIFHandler_Create_NoTenant(t *testing.T) {
	h := NewWorkloadIdentityFederationHandler(&mockWIFService{})
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/workload-identity-federations", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWIFHandler_Create_BadJSON(t *testing.T) {
	h := NewWorkloadIdentityFederationHandler(&mockWIFService{})
	w := httptest.NewRecorder()
	h.Create(w, withTenant(badJSONReq(t, http.MethodPost, "/workload-identity-federations")))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWIFHandler_Create_ValidationError(t *testing.T) {
	h := NewWorkloadIdentityFederationHandler(&mockWIFService{})
	// Missing required fields (name, issuer_url, audience, subject_pattern).
	body := map[string]any{"client_uuid": testClientUUID.String()}
	w := httptest.NewRecorder()
	h.Create(w, withTenant(jsonReq(t, http.MethodPost, "/workload-identity-federations", body)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWIFHandler_Create_ValidationError_NonHTTPSIssuer(t *testing.T) {
	h := NewWorkloadIdentityFederationHandler(&mockWIFService{})
	body := validCreateBody()
	body["issuer_url"] = "http://insecure.example.com"
	w := httptest.NewRecorder()
	h.Create(w, withTenant(jsonReq(t, http.MethodPost, "/workload-identity-federations", body)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWIFHandler_Create_ServiceError(t *testing.T) {
	svc := &mockWIFService{
		createFn: func(int64, WorkloadIdentityFederationCreateInput) (*WorkloadIdentityFederationServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewWorkloadIdentityFederationHandler(svc)
	w := httptest.NewRecorder()
	h.Create(w, withTenant(jsonReq(t, http.MethodPost, "/workload-identity-federations", validCreateBody())))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestWIFHandler_Create_Success(t *testing.T) {
	svc := &mockWIFService{
		createFn: func(tid int64, in WorkloadIdentityFederationCreateInput) (*WorkloadIdentityFederationServiceDataResult, error) {
			assert.Equal(t, testTenantID, tid)
			assert.Equal(t, testClientUUID, in.ClientUUID)
			assert.Equal(t, "github-actions", in.Name)
			// Actor user id is propagated from the auth context for created_by.
			if assert.NotNil(t, in.ActorUserID) {
				assert.Equal(t, testUserID, *in.ActorUserID)
			}
			return wifResult(), nil
		},
	}
	h := NewWorkloadIdentityFederationHandler(svc)
	w := httptest.NewRecorder()
	h.Create(w, withTenant(jsonReq(t, http.MethodPost, "/workload-identity-federations", validCreateBody())))
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestWIFHandler_Update_NoTenant(t *testing.T) {
	h := NewWorkloadIdentityFederationHandler(&mockWIFService{})
	w := httptest.NewRecorder()
	h.Update(w, httptest.NewRequest(http.MethodPut, "/workload-identity-federations/abc", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWIFHandler_Update_InvalidUUID(t *testing.T) {
	h := NewWorkloadIdentityFederationHandler(&mockWIFService{})
	r := withTenant(httptest.NewRequest(http.MethodPut, "/workload-identity-federations/bad", nil))
	r = withChiParam(r, "workload_identity_federation_uuid", "bad")
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWIFHandler_Update_BadJSON(t *testing.T) {
	h := NewWorkloadIdentityFederationHandler(&mockWIFService{})
	r := withTenant(badJSONReq(t, http.MethodPut, "/workload-identity-federations/"+testResourceUUID.String()))
	r = withChiParam(r, "workload_identity_federation_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWIFHandler_Update_ValidationError(t *testing.T) {
	h := NewWorkloadIdentityFederationHandler(&mockWIFService{})
	body := map[string]any{"description": "missing required fields"}
	r := withTenant(jsonReq(t, http.MethodPut, "/workload-identity-federations/"+testResourceUUID.String(), body))
	r = withChiParam(r, "workload_identity_federation_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWIFHandler_Update_ServiceError(t *testing.T) {
	svc := &mockWIFService{
		updateFn: func(int64, uuid.UUID, WorkloadIdentityFederationUpdateInput) (*WorkloadIdentityFederationServiceDataResult, error) {
			return nil, errNotFound
		},
	}
	h := NewWorkloadIdentityFederationHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPut, "/workload-identity-federations/"+testResourceUUID.String(), validUpdateBody()))
	r = withChiParam(r, "workload_identity_federation_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWIFHandler_Update_Success(t *testing.T) {
	svc := &mockWIFService{
		updateFn: func(tid int64, id uuid.UUID, in WorkloadIdentityFederationUpdateInput) (*WorkloadIdentityFederationServiceDataResult, error) {
			assert.Equal(t, testResourceUUID, id)
			assert.Equal(t, "github-actions", in.Name)
			return wifResult(), nil
		},
	}
	h := NewWorkloadIdentityFederationHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPut, "/workload-identity-federations/"+testResourceUUID.String(), validUpdateBody()))
	r = withChiParam(r, "workload_identity_federation_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestWIFHandler_Delete_NoTenant(t *testing.T) {
	h := NewWorkloadIdentityFederationHandler(&mockWIFService{})
	w := httptest.NewRecorder()
	h.Delete(w, httptest.NewRequest(http.MethodDelete, "/workload-identity-federations/abc", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWIFHandler_Delete_InvalidUUID(t *testing.T) {
	h := NewWorkloadIdentityFederationHandler(&mockWIFService{})
	r := withTenant(httptest.NewRequest(http.MethodDelete, "/workload-identity-federations/bad", nil))
	r = withChiParam(r, "workload_identity_federation_uuid", "bad")
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWIFHandler_Delete_ServiceError(t *testing.T) {
	svc := &mockWIFService{
		deleteFn: func(int64, uuid.UUID) (*WorkloadIdentityFederationServiceDataResult, error) {
			return nil, errNotFound
		},
	}
	h := NewWorkloadIdentityFederationHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodDelete, "/workload-identity-federations/"+testResourceUUID.String(), nil))
	r = withChiParam(r, "workload_identity_federation_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWIFHandler_Delete_Success(t *testing.T) {
	svc := &mockWIFService{
		deleteFn: func(_ int64, id uuid.UUID) (*WorkloadIdentityFederationServiceDataResult, error) {
			assert.Equal(t, testResourceUUID, id)
			return wifResult(), nil
		},
	}
	h := NewWorkloadIdentityFederationHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodDelete, "/workload-identity-federations/"+testResourceUUID.String(), nil))
	r = withChiParam(r, "workload_identity_federation_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
