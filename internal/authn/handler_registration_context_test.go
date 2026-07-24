package authn

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRegistrationContextService struct {
	result *RegistrationContextResult
	err    error
	gotCtx struct {
		clientID *string
		tenantID *string
		flowName string
	}
}

func (s *stubRegistrationContextService) Get(_ context.Context, clientID, tenantID *string, flowName string) (*RegistrationContextResult, error) {
	s.gotCtx.clientID, s.gotCtx.tenantID, s.gotCtx.flowName = clientID, tenantID, flowName
	return s.result, s.err
}

func contextRequest(url string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, url, nil)
	r.Header.Set("User-Agent", "Mozilla/5.0 (test)")
	return withSecurityCtx(r)
}

func decodeContextBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	return envelope.Data
}

func TestRegistrationContextHandler_GetPublic(t *testing.T) {
	okResult := func() *RegistrationContextResult {
		return &RegistrationContextResult{
			RegistrationFlow:     "partner-signup",
			RequiredFields:       []string{"fullname"},
			VerificationRequired: true,
		}
	}

	// Surface contract (CLAUDE.md): the public port requires client_id and rejects
	// tenant_id. Asserted here so this endpoint cannot drift from /register.
	t.Run("client_id is required", func(t *testing.T) {
		svc := &stubRegistrationContextService{result: okResult()}
		w := httptest.NewRecorder()
		NewRegistrationContextHandler(svc).GetPublic(w, contextRequest("/registration_context"))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("tenant_id is rejected", func(t *testing.T) {
		svc := &stubRegistrationContextService{result: okResult()}
		w := httptest.NewRecorder()
		NewRegistrationContextHandler(svc).GetPublic(w,
			contextRequest("/registration_context?client_id=c1&tenant_id=t1"))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("malformed registration_flow is rejected before any lookup", func(t *testing.T) {
		svc := &stubRegistrationContextService{result: okResult()}
		w := httptest.NewRecorder()
		NewRegistrationContextHandler(svc).GetPublic(w,
			contextRequest("/registration_context?client_id=c1&registration_flow=Not%20A%20Slug"))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Empty(t, svc.gotCtx.flowName, "a value that cannot match any stored name must not reach the service")
	})

	t.Run("over-long registration_flow is rejected", func(t *testing.T) {
		long := ""
		for range 101 {
			long += "a"
		}
		svc := &stubRegistrationContextService{result: okResult()}
		w := httptest.NewRecorder()
		NewRegistrationContextHandler(svc).GetPublic(w,
			contextRequest("/registration_context?client_id=c1&registration_flow="+long))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service not-found becomes 404", func(t *testing.T) {
		svc := &stubRegistrationContextService{
			err: apperror.NewNotFoundWithReason("registration flow not found for this client"),
		}
		w := httptest.NewRecorder()
		NewRegistrationContextHandler(svc).GetPublic(w,
			contextRequest("/registration_context?client_id=c1&registration_flow=partner-signup"))

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("service internal error does not leak the detail", func(t *testing.T) {
		svc := &stubRegistrationContextService{
			err: apperror.NewInternal("failed to load registration flow", errors.New("pq: boom")),
		}
		w := httptest.NewRecorder()
		NewRegistrationContextHandler(svc).GetPublic(w,
			contextRequest("/registration_context?client_id=c1&registration_flow=partner-signup"))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.NotContains(t, w.Body.String(), "pq:")
	})

	t.Run("success returns the full body and forwards the parsed params", func(t *testing.T) {
		svc := &stubRegistrationContextService{result: okResult()}
		w := httptest.NewRecorder()
		NewRegistrationContextHandler(svc).GetPublic(w,
			contextRequest("/registration_context?client_id=c1&registration_flow=partner-signup"))

		assert.Equal(t, http.StatusOK, w.Code)

		require.NotNil(t, svc.gotCtx.clientID)
		assert.Equal(t, "c1", *svc.gotCtx.clientID)
		assert.Nil(t, svc.gotCtx.tenantID, "tenant_id is never honoured on the public surface")
		assert.Equal(t, "partner-signup", svc.gotCtx.flowName)

		body := decodeContextBody(t, w)
		assert.Equal(t, "partner-signup", body["registration_flow"])
		assert.Equal(t, []any{"fullname"}, body["required_fields"])
		assert.Equal(t, true, body["verification_required"])
	})

	// Status is the operator's kill switch for a published link, so a cached copy
	// is exactly the window in which a revoked link keeps looking valid.
	t.Run("response is never cached", func(t *testing.T) {
		svc := &stubRegistrationContextService{result: okResult()}
		w := httptest.NewRecorder()
		NewRegistrationContextHandler(svc).GetPublic(w,
			contextRequest("/registration_context?client_id=c1&registration_flow=partner-signup"))

		assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	})

	// The body must describe the form and nothing else. Roles above all: the flow
	// name is guessable by design, so publishing the granted roles would hand an
	// attacker a ranked target list.
	t.Run("body exposes only presentation fields", func(t *testing.T) {
		svc := &stubRegistrationContextService{result: okResult()}
		w := httptest.NewRecorder()
		NewRegistrationContextHandler(svc).GetPublic(w,
			contextRequest("/registration_context?client_id=c1&registration_flow=partner-signup"))

		body := decodeContextBody(t, w)
		assert.ElementsMatch(t,
			[]string{"registration_flow", "required_fields", "verification_required"},
			keysOf(body),
			"any new field here is a deliberate disclosure decision",
		)
		for _, forbidden := range []string{
			"roles", "role_ids", "description", "is_system", "status",
			"client_id", "tenant_id", "registration_flow_id", "created_at", "updated_at",
		} {
			assert.NotContains(t, body, forbidden)
		}
	})

	t.Run("no flow in the query is a valid baseline request", func(t *testing.T) {
		svc := &stubRegistrationContextService{result: &RegistrationContextResult{RequiredFields: []string{}}}
		w := httptest.NewRecorder()
		NewRegistrationContextHandler(svc).GetPublic(w,
			contextRequest("/registration_context?client_id=c1"))

		assert.Equal(t, http.StatusOK, w.Code)
		body := decodeContextBody(t, w)
		assert.Equal(t, "", body["registration_flow"])
		assert.Equal(t, []any{}, body["required_fields"], "must serialize as [] rather than null")
	})
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestRegistrationContextQueryDTO_Validate(t *testing.T) {
	valid := []string{"partner", "partner-signup", "partner_signup", "flow2026", "a1-b2_c3"}
	for _, name := range valid {
		t.Run("accepts "+name, func(t *testing.T) {
			q := RegistrationContextQueryDTO{ClientID: "c1", RegistrationFlow: name}
			assert.NoError(t, q.Validate())
		})
	}

	invalid := map[string]string{
		"spaces":            "partner signup",
		"uppercase":         "PartnerSignup",
		"colon":             "partner:signup",
		"leading hyphen":    "-partner",
		"trailing hyphen":   "partner-",
		"double separator":  "partner--signup",
		"leading separator": "_partner",
	}
	for label, name := range invalid {
		t.Run("rejects "+label, func(t *testing.T) {
			q := RegistrationContextQueryDTO{ClientID: "c1", RegistrationFlow: name}
			assert.Error(t, q.Validate())
		})
	}

	t.Run("an absent flow is valid — it means baseline signup", func(t *testing.T) {
		q := RegistrationContextQueryDTO{ClientID: "c1"}
		assert.NoError(t, q.Validate())
	})
}
