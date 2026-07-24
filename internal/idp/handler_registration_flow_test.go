package idp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// fullFlowResult is a service result with every field populated, so success
// tests can assert the handler projects each one into the right DTO shape.
func fullFlowResult(flowUUID uuid.UUID) *RegistrationFlowServiceDataResult {
	clientUUID := uuid.New()
	return &RegistrationFlowServiceDataResult{
		RegistrationFlowUUID: flowUUID,
		Name:                 "partner-signup",
		Description:          "Partner self-service signup",
		Status:               shared.StatusActive,
		ClientUUID:           &clientUUID,
		ClientName:           "partner-portal",
		ClientDisplayName:    "Partner Portal",
		ClientIdentifier:     "partner-portal-client",
		ClientStatus:         shared.StatusActive,
		VerificationRequired: true,
		RequiredFields:       datatypes.JSON([]byte(`["email","fullname"]`)),
		IsSystem:             false,
	}
}

// ---------------------------------------------------------------------------
// Get (list)
// ---------------------------------------------------------------------------

func TestRegistrationFlowHandler_Get(t *testing.T) {
	// 1. auth
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/registration_flows", nil)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Get(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// 3. query parameter validation
	t.Run("invalid sort_order returns 400", func(t *testing.T) {
		r := withTenant(jsonReq(t, http.MethodGet, "/registration_flows?sort_order=bad", nil))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Get(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid client_id returns 400", func(t *testing.T) {
		r := withTenant(jsonReq(t, http.MethodGet, "/registration_flows?client_id=not-a-uuid", nil))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Get(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid status value returns 400", func(t *testing.T) {
		r := withTenant(jsonReq(t, http.MethodGet, "/registration_flows?status=bogus", nil))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Get(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// 5. primary service error
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			getFn: func(RegistrationFlowServiceGetFilter) (*RegistrationFlowServiceListResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := withTenant(jsonReq(t, http.MethodGet, "/registration_flows?page=1&limit=10", nil))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	// 6. success + full body
	t.Run("success returns 200 with the lean list projection", func(t *testing.T) {
		flowUUID := uuid.New()
		svc := &mockRegistrationFlowService{
			getFn: func(f RegistrationFlowServiceGetFilter) (*RegistrationFlowServiceListResult, error) {
				assert.Equal(t, tenantID, f.TenantID)
				return &RegistrationFlowServiceListResult{
					Data:  []RegistrationFlowServiceDataResult{*fullFlowResult(flowUUID)},
					Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		r := withTenant(jsonReq(t, http.MethodGet, "/registration_flows?page=1&limit=10", nil))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Get(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		body := decodeEnvelopeDataMap(t, w)
		assert.Equal(t, float64(1), body["total"])
		assert.Equal(t, float64(1), body["total_pages"])
		rows, ok := body["rows"].([]any)
		require.True(t, ok)
		require.Len(t, rows, 1)
		row := rows[0].(map[string]any)
		assert.Equal(t, flowUUID.String(), row["registration_flow_id"])
		assert.Equal(t, "partner-signup", row["name"])
		assert.Equal(t, shared.StatusActive, row["status"])
		assert.Equal(t, true, row["verification_required"])
		assert.Equal(t, false, row["is_system"])
		// The list shape is lean: required_fields and the nested client summary
		// are detail-only.
		assert.NotContains(t, row, "required_fields")
		assert.NotContains(t, row, "client")
	})

	t.Run("success with all filters parsed onto the service filter", func(t *testing.T) {
		clientUUID := uuid.New()
		var got RegistrationFlowServiceGetFilter
		svc := &mockRegistrationFlowService{
			getFn: func(f RegistrationFlowServiceGetFilter) (*RegistrationFlowServiceListResult, error) {
				got = f
				return &RegistrationFlowServiceListResult{Page: 1, Limit: 10}, nil
			},
		}
		url := "/registration_flows?page=1&limit=10&name=partner" +
			"&search=partner&status=active,%20inactive&is_system=true&client_id=" + clientUUID.String()
		r := withTenant(jsonReq(t, http.MethodGet, url, nil))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Get(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, got.Name)
		assert.Equal(t, "partner", *got.Name)
		require.NotNil(t, got.Search)
		assert.Equal(t, "partner", *got.Search)
		// Status is a comma-separated multi-select, whitespace tolerated.
		assert.Equal(t, []string{shared.StatusActive, shared.StatusInactive}, got.Status)
		require.NotNil(t, got.IsSystem)
		assert.True(t, *got.IsSystem)
		require.NotNil(t, got.ClientUUID)
		assert.Equal(t, clientUUID, *got.ClientUUID)
	})

	t.Run("is_system=1 parses as true and is_system=false as false", func(t *testing.T) {
		for _, tc := range []struct {
			raw  string
			want bool
		}{{"1", true}, {"true", true}, {"false", false}, {"0", false}} {
			t.Run("is_system="+tc.raw, func(t *testing.T) {
				var got RegistrationFlowServiceGetFilter
				svc := &mockRegistrationFlowService{
					getFn: func(f RegistrationFlowServiceGetFilter) (*RegistrationFlowServiceListResult, error) {
						got = f
						return &RegistrationFlowServiceListResult{}, nil
					},
				}
				r := withTenant(jsonReq(t, http.MethodGet, "/registration_flows?is_system="+tc.raw, nil))
				w := httptest.NewRecorder()
				NewRegistrationFlowHandler(svc).Get(w, r)
				assert.Equal(t, http.StatusOK, w.Code)
				require.NotNil(t, got.IsSystem)
				assert.Equal(t, tc.want, *got.IsSystem)
			})
		}
	})

	t.Run("omitted is_system leaves the filter nil", func(t *testing.T) {
		var got RegistrationFlowServiceGetFilter
		svc := &mockRegistrationFlowService{
			getFn: func(f RegistrationFlowServiceGetFilter) (*RegistrationFlowServiceListResult, error) {
				got = f
				return &RegistrationFlowServiceListResult{}, nil
			},
		}
		r := withTenant(jsonReq(t, http.MethodGet, "/registration_flows", nil))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Nil(t, got.IsSystem)
		assert.Nil(t, got.Status)
	})

	t.Run("success with empty result set returns an empty rows array", func(t *testing.T) {
		r := withTenant(jsonReq(t, http.MethodGet, "/registration_flows?page=1&limit=10", nil))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		body := decodeEnvelopeDataMap(t, w)
		rows, ok := body["rows"].([]any)
		require.True(t, ok)
		assert.Empty(t, rows)
	})
}

// ---------------------------------------------------------------------------
// GetByUUID
// ---------------------------------------------------------------------------

func TestRegistrationFlowHandler_GetByUUID(t *testing.T) {
	flowUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodGet, "/", nil), "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetByUUID(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("missing uuid param returns 400", func(t *testing.T) {
		r := withTenant(jsonReq(t, http.MethodGet, "/", nil))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetByUUID(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(jsonReq(t, http.MethodGet, "/", nil), "registration_flow_uuid", "bad"))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetByUUID(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			getByUUIDFn: func(uuid.UUID, int64) (*RegistrationFlowServiceDataResult, error) {
				return nil, errNotFound
			},
		}
		r := withTenant(withChiParam(jsonReq(t, http.MethodGet, "/", nil), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).GetByUUID(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			getByUUIDFn: func(uuid.UUID, int64) (*RegistrationFlowServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := withTenant(withChiParam(jsonReq(t, http.MethodGet, "/", nil), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).GetByUUID(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200 with the detail projection", func(t *testing.T) {
		want := fullFlowResult(flowUUID)
		svc := &mockRegistrationFlowService{
			getByUUIDFn: func(id uuid.UUID, tid int64) (*RegistrationFlowServiceDataResult, error) {
				assert.Equal(t, flowUUID, id)
				assert.Equal(t, tenantID, tid)
				return want, nil
			},
		}
		r := withTenant(withChiParam(jsonReq(t, http.MethodGet, "/", nil), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).GetByUUID(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		// Snapshot the body so it can be read twice: typed, then untyped.
		rawBody := append([]byte(nil), w.Body.Bytes()...)
		got := decodeEnvelopeData[RegistrationFlowDetailResponseDTO](t, w)
		assert.Equal(t, flowUUID.String(), got.RegistrationFlowUUID)
		assert.Equal(t, want.Name, got.Name)
		assert.Equal(t, want.Status, got.Status)
		assert.True(t, got.VerificationRequired)
		assert.False(t, got.IsSystem)
		assert.JSONEq(t, `["email","fullname"]`, string(got.RequiredFields))
		require.NotNil(t, got.ClientUUID)
		assert.Equal(t, want.ClientUUID.String(), *got.ClientUUID)
		// The detail shape resolves the client so an operator sees a name, not a UUID.
		require.NotNil(t, got.Client)
		assert.Equal(t, want.ClientUUID.String(), got.Client.ClientUUID)
		assert.Equal(t, "partner-portal", got.Client.Name)
		assert.Equal(t, "Partner Portal", got.Client.DisplayName)
		assert.Equal(t, "partner-portal-client", got.Client.Identifier)
		assert.Equal(t, shared.StatusActive, got.Client.Status)

		// required_fields IS part of the detail shape (unlike the list shape).
		var env struct {
			Data map[string]any `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rawBody, &env))
		assert.Contains(t, env.Data, "required_fields")
		assert.Contains(t, env.Data, "client")
	})

	t.Run("success without a client omits the nested client summary", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			getByUUIDFn: func(id uuid.UUID, _ int64) (*RegistrationFlowServiceDataResult, error) {
				return &RegistrationFlowServiceDataResult{
					RegistrationFlowUUID: id,
					Name:                 "orphan",
					RequiredFields:       datatypes.JSON([]byte(`[]`)),
				}, nil
			},
		}
		r := withTenant(withChiParam(jsonReq(t, http.MethodGet, "/", nil), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).GetByUUID(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		body := decodeEnvelopeDataMap(t, w)
		assert.NotContains(t, body, "client")
		assert.NotContains(t, body, "client_id")
	})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestRegistrationFlowHandler_Create(t *testing.T) {
	clientUUID := uuid.New()
	roleUUID := uuid.New()
	validBody := map[string]any{
		"name":        "partner-signup",
		"description": "Partner self-service signup",
		"client_id":   clientUUID.String(),
	}

	// 1. auth
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/registration_flows", validBody)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Create(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("no user returns 401", func(t *testing.T) {
		// A nil audit actor used to fall through here and create the flow with a
		// nil created_by; the handler now refuses.
		called := false
		svc := &mockRegistrationFlowService{
			createFn: func(RegistrationFlowCreateInput) (*RegistrationFlowServiceDataResult, error) {
				called = true
				return &RegistrationFlowServiceDataResult{}, nil
			},
		}
		r := withTenant(jsonReq(t, http.MethodPost, "/registration_flows", validBody))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, called, "service must not be called without an authenticated actor")
	})

	// 4. body parsing
	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withTenantAndUser(badJSONReq(t, http.MethodPost, "/registration_flows"))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// 5. DTO validation
	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/registration_flows", map[string]any{"name": ""}))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// 8. service errors mapped by type
	t.Run("service conflict returns 409", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			createFn: func(RegistrationFlowCreateInput) (*RegistrationFlowServiceDataResult, error) {
				return nil, apperror.NewConflict("registration flow with this name already exists")
			},
		}
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/registration_flows", validBody))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("service forbidden returns 403", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			createFn: func(RegistrationFlowCreateInput) (*RegistrationFlowServiceDataResult, error) {
				return nil, apperror.NewForbidden("system roles cannot be assigned to a registration flow")
			},
		}
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/registration_flows", validBody))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("service validation error returns 400", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			createFn: func(RegistrationFlowCreateInput) (*RegistrationFlowServiceDataResult, error) {
				return nil, errValidation
			},
		}
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/registration_flows", validBody))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			createFn: func(RegistrationFlowCreateInput) (*RegistrationFlowServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/registration_flows", validBody))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	// 9. success
	t.Run("success returns 201 with the detail body and defaults applied", func(t *testing.T) {
		flowUUID := uuid.New()
		var got RegistrationFlowCreateInput
		svc := &mockRegistrationFlowService{
			createFn: func(in RegistrationFlowCreateInput) (*RegistrationFlowServiceDataResult, error) {
				got = in
				return fullFlowResult(flowUUID), nil
			},
		}
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/registration_flows", validBody))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Create(w, r)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		// Input mapping: tenant + actor come from context, status defaults to active,
		// verification_required defaults to false, and no identifier is accepted.
		assert.Equal(t, tenantID, got.TenantID)
		assert.Equal(t, testUserUUID, got.ActorUserUUID)
		assert.Equal(t, "partner-signup", got.Name)
		assert.Equal(t, shared.StatusActive, got.Status)
		assert.Equal(t, clientUUID, got.ClientUUID)
		assert.False(t, got.VerificationRequired)
		assert.Nil(t, got.RoleUUIDs)
		assert.Nil(t, got.RequiredFields)

		body := decodeEnvelopeData[RegistrationFlowDetailResponseDTO](t, w)
		assert.Equal(t, flowUUID.String(), body.RegistrationFlowUUID)
		assert.True(t, body.VerificationRequired)
		assert.False(t, body.IsSystem)
		assert.JSONEq(t, `["email","fullname"]`, string(body.RequiredFields))
		require.NotNil(t, body.Client)
	})

	t.Run("success with explicit status, roles, verification and required_fields", func(t *testing.T) {
		var got RegistrationFlowCreateInput
		svc := &mockRegistrationFlowService{
			createFn: func(in RegistrationFlowCreateInput) (*RegistrationFlowServiceDataResult, error) {
				got = in
				return fullFlowResult(uuid.New()), nil
			},
		}
		body := map[string]any{
			"name":                  "partner-signup",
			"description":           "desc",
			"client_id":             clientUUID.String(),
			"status":                shared.StatusInactive,
			"role_ids":              []string{roleUUID.String()},
			"verification_required": true,
			"required_fields":       []string{"email"},
		}
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/registration_flows", body))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Create(w, r)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, shared.StatusInactive, got.Status)
		assert.Equal(t, []uuid.UUID{roleUUID}, got.RoleUUIDs)
		assert.True(t, got.VerificationRequired)
		require.NotNil(t, got.RequiredFields)
		assert.Equal(t, []string{"email"}, *got.RequiredFields)
	})

	t.Run("a stray identifier key in the body is ignored; name is the selector", func(t *testing.T) {
		var got RegistrationFlowCreateInput
		svc := &mockRegistrationFlowService{
			createFn: func(in RegistrationFlowCreateInput) (*RegistrationFlowServiceDataResult, error) {
				got = in
				return fullFlowResult(uuid.New()), nil
			},
		}
		// There is no separate identifier any more — the name IS the public
		// selector. A leftover "identifier" key from an older client must not
		// influence anything.
		body := map[string]any{
			"name":       "partner-signup",
			"client_id":  clientUUID.String(),
			"identifier": "attacker-chosen-identifier",
		}
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/registration_flows", body))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Create(w, r)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "partner-signup", got.Name)
	})

	t.Run("audit entry records the actor and the created flow", func(t *testing.T) {
		flowUUID := uuid.New()
		svc := &mockRegistrationFlowService{
			createFn: func(RegistrationFlowCreateInput) (*RegistrationFlowServiceDataResult, error) {
				return fullFlowResult(flowUUID), nil
			},
		}
		logger := &captureAuditLogger{}
		h := NewRegistrationFlowHandler(svc)
		h.SetAuditLogger(logger)

		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/registration_flows", validBody))
		w := httptest.NewRecorder()
		h.Create(w, r)

		assert.Equal(t, http.StatusCreated, w.Code)
		require.Len(t, logger.entries, 1)
		entry := logger.entries[0]
		assert.Equal(t, "registration_flow.create", entry.Action)
		assert.Equal(t, "registration_flow", entry.ResourceType)
		assert.Equal(t, flowUUID.String(), entry.ResourceID)
		require.NotNil(t, entry.ActorUserID)
		assert.Equal(t, testActorUserID, *entry.ActorUserID)
		assert.Equal(t, tenantID, entry.TenantID)
		assert.Equal(t, "success", entry.Outcome)
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestRegistrationFlowHandler_Update(t *testing.T) {
	flowUUID := uuid.New()
	validBody := map[string]any{"name": "renamed"}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Update(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("no user returns 401", func(t *testing.T) {
		called := false
		svc := &mockRegistrationFlowService{
			updateFn: func(RegistrationFlowUpdateInput) (*RegistrationFlowServiceDataResult, error) {
				called = true
				return &RegistrationFlowServiceDataResult{}, nil
			},
		}
		r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, called)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "registration_flow_uuid", "bad"))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPut, "/"), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"name": ""}), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("system flow returns 400 from the service guard", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			updateFn: func(RegistrationFlowUpdateInput) (*RegistrationFlowServiceDataResult, error) {
				return nil, apperror.NewValidation("system registration flow is not allowed to be updated")
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			updateFn: func(RegistrationFlowUpdateInput) (*RegistrationFlowServiceDataResult, error) {
				return nil, errNotFound
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			updateFn: func(RegistrationFlowUpdateInput) (*RegistrationFlowServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	// The data-loss regression: a PUT carrying only "name" must leave every other
	// field nil so the service treats them as unchanged.
	t.Run("omitted fields arrive nil (omitted means unchanged)", func(t *testing.T) {
		var got RegistrationFlowUpdateInput
		svc := &mockRegistrationFlowService{
			updateFn: func(in RegistrationFlowUpdateInput) (*RegistrationFlowServiceDataResult, error) {
				got = in
				return fullFlowResult(flowUUID), nil
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"name": "renamed"}), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Update(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, flowUUID, got.RegistrationFlowUUID)
		assert.Equal(t, tenantID, got.TenantID)
		assert.Equal(t, testUserUUID, got.ActorUserUUID)
		require.NotNil(t, got.Name)
		assert.Equal(t, "renamed", *got.Name)
		assert.Nil(t, got.Description, "description omitted → unchanged")
		assert.Nil(t, got.Status, "status omitted → must not be re-activated")
		assert.Nil(t, got.VerificationRequired, "verification_required omitted → must not be downgraded")
		assert.Nil(t, got.RequiredFields, "required_fields omitted → must not be wiped")
		assert.Nil(t, got.RoleUUIDs, "role_ids omitted → membership untouched")
	})

	t.Run("explicitly provided fields are forwarded", func(t *testing.T) {
		roleUUID := uuid.New()
		var got RegistrationFlowUpdateInput
		svc := &mockRegistrationFlowService{
			updateFn: func(in RegistrationFlowUpdateInput) (*RegistrationFlowServiceDataResult, error) {
				got = in
				return fullFlowResult(flowUUID), nil
			},
		}
		body := map[string]any{
			"name":                  "renamed",
			"description":           "new desc",
			"status":                shared.StatusInactive,
			"verification_required": false,
			"required_fields":       []string{"email", "phone"},
			"role_ids":              []string{roleUUID.String()},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", body), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Update(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, got.Description)
		assert.Equal(t, "new desc", *got.Description)
		require.NotNil(t, got.Status)
		assert.Equal(t, shared.StatusInactive, *got.Status)
		require.NotNil(t, got.VerificationRequired)
		assert.False(t, *got.VerificationRequired, "an explicit false must be distinguishable from omitted")
		require.NotNil(t, got.RequiredFields)
		assert.Equal(t, []string{"email", "phone"}, *got.RequiredFields)
		assert.Equal(t, []uuid.UUID{roleUUID}, got.RoleUUIDs)
	})

	t.Run("empty role_ids array clears membership (non-nil, empty)", func(t *testing.T) {
		var got RegistrationFlowUpdateInput
		svc := &mockRegistrationFlowService{
			updateFn: func(in RegistrationFlowUpdateInput) (*RegistrationFlowServiceDataResult, error) {
				got = in
				return fullFlowResult(flowUUID), nil
			},
		}
		body := map[string]any{"role_ids": []string{}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", body), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Update(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, got.RoleUUIDs, "an empty array must NOT collapse to nil (that would mean 'unchanged')")
		assert.Empty(t, got.RoleUUIDs)
	})

	t.Run("success returns 200 with the detail body", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			updateFn: func(RegistrationFlowUpdateInput) (*RegistrationFlowServiceDataResult, error) {
				return fullFlowResult(flowUUID), nil
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Update(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		body := decodeEnvelopeData[RegistrationFlowDetailResponseDTO](t, w)
		assert.Equal(t, flowUUID.String(), body.RegistrationFlowUUID)
		assert.True(t, body.VerificationRequired)
		assert.False(t, body.IsSystem)
		assert.JSONEq(t, `["email","fullname"]`, string(body.RequiredFields))
	})
}

// ---------------------------------------------------------------------------
// SetStatus
// ---------------------------------------------------------------------------

func TestRegistrationFlowHandler_SetStatus(t *testing.T) {
	flowUUID := uuid.New()
	validBody := map[string]any{"status": shared.StatusInactive}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPatch, "/", validBody), "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("no user returns 401", func(t *testing.T) {
		called := false
		svc := &mockRegistrationFlowService{
			setStatusFn: func(uuid.UUID, int64, uuid.UUID, string) (*RegistrationFlowServiceDataResult, error) {
				called = true
				return &RegistrationFlowServiceDataResult{}, nil
			},
		}
		r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/", validBody), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).SetStatus(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, called)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", validBody), "registration_flow_uuid", "bad"))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPatch, "/"), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid status returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "bogus"}), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing status returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{}), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			setStatusFn: func(uuid.UUID, int64, uuid.UUID, string) (*RegistrationFlowServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", validBody), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).SetStatus(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200 with the updated status and the actor forwarded", func(t *testing.T) {
		result := fullFlowResult(flowUUID)
		result.Status = shared.StatusInactive
		var gotStatus string
		var gotActor uuid.UUID
		svc := &mockRegistrationFlowService{
			setStatusFn: func(id uuid.UUID, tid int64, actor uuid.UUID, status string) (*RegistrationFlowServiceDataResult, error) {
				assert.Equal(t, flowUUID, id)
				assert.Equal(t, tenantID, tid)
				gotActor, gotStatus = actor, status
				return result, nil
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", validBody), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).SetStatus(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, shared.StatusInactive, gotStatus)
		assert.Equal(t, testUserUUID, gotActor)

		body := decodeEnvelopeData[RegistrationFlowDetailResponseDTO](t, w)
		assert.Equal(t, shared.StatusInactive, body.Status)
		assert.False(t, body.IsSystem)
	})
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestRegistrationFlowHandler_Delete(t *testing.T) {
	flowUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodDelete, "/", nil), "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Delete(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("no user returns 401", func(t *testing.T) {
		called := false
		svc := &mockRegistrationFlowService{
			deleteFn: func(uuid.UUID, int64, uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
				called = true
				return &RegistrationFlowServiceDataResult{}, nil
			},
		}
		r := withTenant(withChiParam(jsonReq(t, http.MethodDelete, "/", nil), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, called)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodDelete, "/", nil), "registration_flow_uuid", "bad-uuid"))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Delete(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// business rules enforced by the service, surfaced as status codes here
	t.Run("system flow returns 400", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			deleteFn: func(uuid.UUID, int64, uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
				return nil, apperror.NewValidation("system registration flow is not allowed to be deleted")
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodDelete, "/", nil), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("pending invites returns 409", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			deleteFn: func(uuid.UUID, int64, uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
				return nil, apperror.NewConflict("cannot delete registration flow that is referenced by pending invites")
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodDelete, "/", nil), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			deleteFn: func(uuid.UUID, int64, uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
				return nil, errNotFound
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodDelete, "/", nil), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			deleteFn: func(uuid.UUID, int64, uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodDelete, "/", nil), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200 with the deleted flow body", func(t *testing.T) {
		var gotActor uuid.UUID
		svc := &mockRegistrationFlowService{
			deleteFn: func(id uuid.UUID, tid int64, actor uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
				assert.Equal(t, flowUUID, id)
				assert.Equal(t, tenantID, tid)
				gotActor = actor
				return fullFlowResult(flowUUID), nil
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodDelete, "/", nil), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Delete(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, testUserUUID, gotActor)
		body := decodeEnvelopeData[RegistrationFlowDetailResponseDTO](t, w)
		assert.Equal(t, flowUUID.String(), body.RegistrationFlowUUID)
		assert.False(t, body.IsSystem)
	})
}

// ---------------------------------------------------------------------------
// AssignRoles
// ---------------------------------------------------------------------------

func TestRegistrationFlowHandler_AssignRoles(t *testing.T) {
	flowUUID := uuid.New()
	roleUUID := uuid.New()
	validBody := map[string]any{"role_uuids": []string{roleUUID.String()}}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("no user returns 401", func(t *testing.T) {
		called := false
		svc := &mockRegistrationFlowService{
			assignRolesFn: func(uuid.UUID, int64, uuid.UUID, []uuid.UUID) ([]RegistrationFlowRoleServiceDataResult, error) {
				called = true
				return nil, nil
			},
		}
		r := withTenant(withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).AssignRoles(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, called)
	})

	t.Run("missing uuid param returns 400", func(t *testing.T) {
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/", validBody))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "registration_flow_uuid", "bad-uuid"))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPost, "/"), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty role_uuids returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{}}), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("malformed role uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{"nope"}}), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("system role rejection returns 403", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			assignRolesFn: func(uuid.UUID, int64, uuid.UUID, []uuid.UUID) ([]RegistrationFlowRoleServiceDataResult, error) {
				return nil, apperror.NewForbidden("system roles cannot be assigned to a registration flow")
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).AssignRoles(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("un-possessed role rejection returns 400", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			assignRolesFn: func(uuid.UUID, int64, uuid.UUID, []uuid.UUID) ([]RegistrationFlowRoleServiceDataResult, error) {
				return nil, apperror.NewValidation("you cannot grant roles you do not possess")
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			assignRolesFn: func(uuid.UUID, int64, uuid.UUID, []uuid.UUID) ([]RegistrationFlowRoleServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).AssignRoles(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200 with the assigned roles", func(t *testing.T) {
		var gotActor uuid.UUID
		var gotRoles []uuid.UUID
		svc := &mockRegistrationFlowService{
			assignRolesFn: func(id uuid.UUID, tid int64, actor uuid.UUID, roles []uuid.UUID) ([]RegistrationFlowRoleServiceDataResult, error) {
				assert.Equal(t, flowUUID, id)
				assert.Equal(t, tenantID, tid)
				gotActor, gotRoles = actor, roles
				return []RegistrationFlowRoleServiceDataResult{{
					RegistrationFlowRoleUUID: uuid.New(),
					RegistrationFlowUUID:     flowUUID,
					RoleUUID:                 roleUUID,
					RoleName:                 "partner-user",
					RoleDescription:          "Partner user",
					RoleStatus:               shared.StatusActive,
					RoleIsDefault:            true,
					RoleIsSystem:             false,
				}}, nil
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).AssignRoles(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, testUserUUID, gotActor)
		assert.Equal(t, []uuid.UUID{roleUUID}, gotRoles)

		roles := decodeEnvelopeData[[]RoleResponseDTO](t, w)
		require.Len(t, roles, 1)
		assert.Equal(t, roleUUID, roles[0].RoleUUID)
		assert.Equal(t, "partner-user", roles[0].Name)
		assert.Equal(t, "Partner user", roles[0].Description)
		assert.Equal(t, shared.StatusActive, roles[0].Status)
		assert.True(t, roles[0].IsDefault)
		assert.False(t, roles[0].IsSystem)
	})

	t.Run("all roles already assigned returns 200 with an empty array", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			assignRolesFn: func(uuid.UUID, int64, uuid.UUID, []uuid.UUID) ([]RegistrationFlowRoleServiceDataResult, error) {
				return []RegistrationFlowRoleServiceDataResult{}, nil
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).AssignRoles(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
		assert.JSONEq(t, `[]`, string(env.Data))
	})
}

// ---------------------------------------------------------------------------
// GetRoles
// ---------------------------------------------------------------------------

func TestRegistrationFlowHandler_GetRoles(t *testing.T) {
	flowUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodGet, "/", nil), "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("missing uuid param returns 400", func(t *testing.T) {
		r := withTenant(jsonReq(t, http.MethodGet, "/", nil))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(jsonReq(t, http.MethodGet, "/", nil), "registration_flow_uuid", "bad-uuid"))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid pagination returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(jsonReq(t, http.MethodGet, "/?sort_order=bad", nil), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("flow not found returns 404", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			getRolesFn: func(uuid.UUID, int64, int, int) (*RegistrationFlowRoleServiceListResult, error) {
				return nil, errNotFound
			},
		}
		r := withTenant(withChiParam(jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).GetRoles(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			getRolesFn: func(uuid.UUID, int64, int, int) (*RegistrationFlowRoleServiceListResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := withTenant(withChiParam(jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).GetRoles(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200 with the paginated roles", func(t *testing.T) {
		roleUUID := uuid.New()
		svc := &mockRegistrationFlowService{
			getRolesFn: func(id uuid.UUID, tid int64, page, limit int) (*RegistrationFlowRoleServiceListResult, error) {
				assert.Equal(t, flowUUID, id)
				assert.Equal(t, tenantID, tid)
				assert.Equal(t, 1, page)
				assert.Equal(t, 10, limit)
				return &RegistrationFlowRoleServiceListResult{
					Data: []RegistrationFlowRoleServiceDataResult{{
						RoleUUID:     roleUUID,
						RoleName:     "partner-user",
						RoleStatus:   shared.StatusActive,
						RoleIsSystem: false,
					}},
					Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		r := withTenant(withChiParam(jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).GetRoles(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		body := decodeEnvelopeData[PaginatedResponseDTO[RoleResponseDTO]](t, w)
		assert.Equal(t, int64(1), body.Total)
		assert.Equal(t, 1, body.TotalPages)
		require.Len(t, body.Rows, 1)
		assert.Equal(t, roleUUID, body.Rows[0].RoleUUID)
		assert.Equal(t, "partner-user", body.Rows[0].Name)
		assert.False(t, body.Rows[0].IsSystem)
	})

	t.Run("success with no roles returns an empty rows array", func(t *testing.T) {
		r := withTenant(withChiParam(jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil), "registration_flow_uuid", flowUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		body := decodeEnvelopeData[PaginatedResponseDTO[RoleResponseDTO]](t, w)
		assert.Empty(t, body.Rows)
	})
}

// ---------------------------------------------------------------------------
// RemoveRole
// ---------------------------------------------------------------------------

func TestRegistrationFlowHandler_RemoveRole(t *testing.T) {
	flowUUID := uuid.New()
	roleUUID := uuid.New()

	withBothParams := func(r *http.Request) *http.Request {
		return withChiParam(withChiParam(r, "registration_flow_uuid", flowUUID.String()), "role_uuid", roleUUID.String())
	}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withBothParams(jsonReq(t, http.MethodDelete, "/", nil))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("no user returns 401", func(t *testing.T) {
		called := false
		svc := &mockRegistrationFlowService{
			removeRoleFn: func(uuid.UUID, int64, uuid.UUID, uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
				called = true
				return &RegistrationFlowServiceDataResult{}, nil
			},
		}
		r := withTenant(withBothParams(jsonReq(t, http.MethodDelete, "/", nil)))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).RemoveRole(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, called)
	})

	t.Run("missing both uuid params returns 400", func(t *testing.T) {
		r := withTenantAndUser(jsonReq(t, http.MethodDelete, "/", nil))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid registration flow uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodDelete, "/", nil), "registration_flow_uuid", "bad"), "role_uuid", roleUUID.String()))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid role uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodDelete, "/", nil), "registration_flow_uuid", flowUUID.String()), "role_uuid", "bad"))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("system flow returns 400", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			removeRoleFn: func(uuid.UUID, int64, uuid.UUID, uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
				return nil, apperror.NewValidation("system registration flow is not allowed to be modified")
			},
		}
		r := withTenantAndUser(withBothParams(jsonReq(t, http.MethodDelete, "/", nil)))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).RemoveRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("role not found returns 404", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			removeRoleFn: func(uuid.UUID, int64, uuid.UUID, uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
				return nil, errNotFound
			},
		}
		r := withTenantAndUser(withBothParams(jsonReq(t, http.MethodDelete, "/", nil)))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).RemoveRole(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			removeRoleFn: func(uuid.UUID, int64, uuid.UUID, uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := withTenantAndUser(withBothParams(jsonReq(t, http.MethodDelete, "/", nil)))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).RemoveRole(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	// RemoveRole returns the parent flow now, so the client can re-render without
	// a follow-up GET — assert the body is that flow, not an empty envelope.
	t.Run("success returns 200 with the parent flow", func(t *testing.T) {
		var gotFlow, gotActor, gotRole uuid.UUID
		svc := &mockRegistrationFlowService{
			removeRoleFn: func(id uuid.UUID, tid int64, actor, role uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
				gotFlow, gotActor, gotRole = id, actor, role
				assert.Equal(t, tenantID, tid)
				return fullFlowResult(flowUUID), nil
			},
		}
		r := withTenantAndUser(withBothParams(jsonReq(t, http.MethodDelete, "/", nil)))
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).RemoveRole(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, flowUUID, gotFlow)
		assert.Equal(t, testUserUUID, gotActor)
		assert.Equal(t, roleUUID, gotRole)

		body := decodeEnvelopeData[RegistrationFlowDetailResponseDTO](t, w)
		assert.Equal(t, flowUUID.String(), body.RegistrationFlowUUID)
		assert.False(t, body.IsSystem)
		assert.JSONEq(t, `["email","fullname"]`, string(body.RequiredFields))
	})
}

// ---------------------------------------------------------------------------
// DTO projection helpers
// ---------------------------------------------------------------------------

func TestClientUUIDString(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, clientUUIDString(nil))
	})

	t.Run("non-nil returns the string form", func(t *testing.T) {
		id := uuid.New()
		got := clientUUIDString(&id)
		require.NotNil(t, got)
		assert.Equal(t, id.String(), *got)
	})
}

func TestParseUUIDList(t *testing.T) {
	valid := uuid.New()

	t.Run("nil input stays nil (field omitted)", func(t *testing.T) {
		assert.Nil(t, parseUUIDList(nil))
	})

	t.Run("empty input returns an empty non-nil slice (field cleared)", func(t *testing.T) {
		got := parseUUIDList([]string{})
		require.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("parses valid uuids", func(t *testing.T) {
		assert.Equal(t, []uuid.UUID{valid}, parseUUIDList([]string{valid.String()}))
	})

	t.Run("skips unparseable entries", func(t *testing.T) {
		assert.Equal(t, []uuid.UUID{valid}, parseUUIDList([]string{valid.String(), "nope"}))
	})
}

func TestToRegistrationFlowListResponseDTO_LeanShape(t *testing.T) {
	flowUUID := uuid.New()
	dto := toRegistrationFlowListResponseDTO(*fullFlowResult(flowUUID))

	raw, err := json.Marshal(dto)
	require.NoError(t, err)
	var asMap map[string]any
	require.NoError(t, json.Unmarshal(raw, &asMap))

	assert.Equal(t, flowUUID.String(), asMap["registration_flow_id"])
	assert.Contains(t, asMap, "name")
	assert.Contains(t, asMap, "verification_required")
	assert.Contains(t, asMap, "is_system")
	// The two detail-only members must be absent from the list projection, and
	// there is no separate identifier any more — the name is the selector.
	assert.NotContains(t, asMap, "identifier")
	assert.NotContains(t, asMap, "required_fields")
	assert.NotContains(t, asMap, "client")
}

func TestToRegistrationFlowDetailResponseDTO(t *testing.T) {
	t.Run("with client resolves the nested summary", func(t *testing.T) {
		flowUUID := uuid.New()
		in := fullFlowResult(flowUUID)
		dto := toRegistrationFlowDetailResponseDTO(*in)
		require.NotNil(t, dto.Client)
		assert.Equal(t, in.ClientUUID.String(), dto.Client.ClientUUID)
		assert.Equal(t, in.ClientName, dto.Client.Name)
		assert.Equal(t, in.ClientDisplayName, dto.Client.DisplayName)
		assert.Equal(t, in.ClientIdentifier, dto.Client.Identifier)
		assert.Equal(t, in.ClientStatus, dto.Client.Status)
	})

	t.Run("without client leaves both client members nil", func(t *testing.T) {
		dto := toRegistrationFlowDetailResponseDTO(RegistrationFlowServiceDataResult{
			RegistrationFlowUUID: uuid.New(),
			RequiredFields:       datatypes.JSON([]byte(`[]`)),
		})
		assert.Nil(t, dto.Client)
		assert.Nil(t, dto.ClientUUID)
	})
}

func TestToRegistrationFlowRoleResponseDTOList(t *testing.T) {
	t.Run("empty input returns an empty slice", func(t *testing.T) {
		assert.Empty(t, toRegistrationFlowRoleResponseDTOList(nil))
	})

	t.Run("maps every field", func(t *testing.T) {
		roleUUID := uuid.New()
		out := toRegistrationFlowRoleResponseDTOList([]RegistrationFlowRoleServiceDataResult{{
			RoleUUID:        roleUUID,
			RoleName:        "partner-user",
			RoleDescription: "desc",
			RoleStatus:      shared.StatusActive,
			RoleIsDefault:   true,
			RoleIsSystem:    true,
		}})
		require.Len(t, out, 1)
		assert.Equal(t, roleUUID, out[0].RoleUUID)
		assert.Equal(t, "partner-user", out[0].Name)
		assert.Equal(t, "desc", out[0].Description)
		assert.Equal(t, shared.StatusActive, out[0].Status)
		assert.True(t, out[0].IsDefault)
		assert.True(t, out[0].IsSystem)
	})
}
