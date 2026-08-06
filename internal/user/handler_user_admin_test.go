package user

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureAuditLogger records the serialized change payloads so a test can assert
// what does — and does not — reach the management audit log.
type captureAuditLogger struct{ changes []string }

func (c *captureAuditLogger) Log(_ context.Context, e auditlog.LogEntry) error {
	c.changes = append(c.changes, e.Changes)
	return nil
}

// ---------------------------------------------------------------------------
// PUT /users/{user_uuid}/password
// ---------------------------------------------------------------------------

func TestUserHandler_SetUserPassword(t *testing.T) {
	body := map[string]any{"password": adminSetPlaintext}

	t.Run("no tenant returns 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).SetUserPassword(w, jsonReq(t, http.MethodPut, "/", body))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("no acting user returns 401", func(t *testing.T) {
		r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/", body), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).SetUserPassword(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withTenantAndActor(withChiParam(jsonReq(t, http.MethodPut, "/", body), "user_uuid", "bad"), 7)
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).SetUserPassword(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing password returns 400", func(t *testing.T) {
		r := withTenantAndActor(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{}), "user_uuid", testResourceUUID.String()), 7)
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).SetUserPassword(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("a policy rejection surfaces as 400, not 500", func(t *testing.T) {
		svc := &mockUserService{setPasswordFn: func(uuid.UUID, int64, string, bool, uuid.UUID) error {
			return apperror.NewValidation("password was used recently and cannot be reused")
		}}
		r := withTenantAndActor(withChiParam(jsonReq(t, http.MethodPut, "/", body), "user_uuid", testResourceUUID.String()), 7)
		w := httptest.NewRecorder()
		NewUserHandler(svc).SetUserPassword(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockUserService{setPasswordFn: func(uuid.UUID, int64, string, bool, uuid.UUID) error {
			return errors.New("boom")
		}}
		r := withTenantAndActor(withChiParam(jsonReq(t, http.MethodPut, "/", body), "user_uuid", testResourceUUID.String()), 7)
		w := httptest.NewRecorder()
		NewUserHandler(svc).SetUserPassword(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success passes the temporary flag through", func(t *testing.T) {
		var gotTemporary bool
		var gotActor uuid.UUID
		svc := &mockUserService{setPasswordFn: func(_ uuid.UUID, _ int64, _ string, temporary bool, actor uuid.UUID) error {
			gotTemporary, gotActor = temporary, actor
			return nil
		}}
		r := withTenantAndActor(withChiParam(
			jsonReq(t, http.MethodPut, "/", map[string]any{"password": adminSetPlaintext, "temporary": true}),
			"user_uuid", testResourceUUID.String()), 7)
		w := httptest.NewRecorder()
		NewUserHandler(svc).SetUserPassword(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, gotTemporary)
		// The actor comes from the verified auth context, never the request body.
		assert.Equal(t, testUserUUID, gotActor)
	})

	// The plaintext must never reach the management audit log, which is readable
	// by anyone holding the audit read permission.
	t.Run("the audit trail never records the password", func(t *testing.T) {
		logger := &captureAuditLogger{}
		h := NewUserHandler(&mockUserService{})
		h.SetAuditLogger(logger)
		r := withTenantAndActor(withChiParam(jsonReq(t, http.MethodPut, "/", body), "user_uuid", testResourceUUID.String()), 7)
		h.SetUserPassword(httptest.NewRecorder(), r)
		require.NotEmpty(t, logger.changes)
		assert.NotContains(t, logger.changes[0], adminSetPlaintext)
	})
}

// ---------------------------------------------------------------------------
// POST /users/{user_uuid}/identities
// ---------------------------------------------------------------------------

func TestUserHandler_LinkUserIdentity(t *testing.T) {
	idpUUID := uuid.New()
	body := map[string]any{"identity_provider_id": idpUUID.String(), "sub": "google-oauth2|1234"}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withUserNoTenant(jsonReq(t, http.MethodPost, "/", body))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).LinkUserIdentity(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid user UUID returns 400", func(t *testing.T) {
		r := withTenantAndActor(withChiParam(jsonReq(t, http.MethodPost, "/", body), "user_uuid", "bad"), 7)
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).LinkUserIdentity(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("non-UUID provider returns 400", func(t *testing.T) {
		r := withTenantAndActor(withChiParam(
			jsonReq(t, http.MethodPost, "/", map[string]any{"identity_provider_id": "nope", "sub": "s"}),
			"user_uuid", testResourceUUID.String()), 7)
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).LinkUserIdentity(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty sub returns 400", func(t *testing.T) {
		r := withTenantAndActor(withChiParam(
			jsonReq(t, http.MethodPost, "/", map[string]any{"identity_provider_id": idpUUID.String(), "sub": ""}),
			"user_uuid", testResourceUUID.String()), 7)
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).LinkUserIdentity(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("an already-linked sub surfaces as 409", func(t *testing.T) {
		svc := &mockUserService{adminLinkIdentityFn: func(uuid.UUID, int64, uuid.UUID, string, uuid.UUID) (*UserIdentityServiceDataResult, error) {
			return nil, apperror.NewConflict("this identity is already linked to another user")
		}}
		r := withTenantAndActor(withChiParam(jsonReq(t, http.MethodPost, "/", body), "user_uuid", testResourceUUID.String()), 7)
		w := httptest.NewRecorder()
		NewUserHandler(svc).LinkUserIdentity(w, r)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("success returns 200 with the linked identity", func(t *testing.T) {
		var gotSub string
		var gotProvider uuid.UUID
		svc := &mockUserService{adminLinkIdentityFn: func(_ uuid.UUID, _ int64, providerUUID uuid.UUID, sub string, _ uuid.UUID) (*UserIdentityServiceDataResult, error) {
			gotSub, gotProvider = sub, providerUUID
			return &UserIdentityServiceDataResult{UserIdentityUUID: uuid.New(), Provider: "google", Sub: sub}, nil
		}}
		r := withTenantAndActor(withChiParam(jsonReq(t, http.MethodPost, "/", body), "user_uuid", testResourceUUID.String()), 7)
		w := httptest.NewRecorder()
		NewUserHandler(svc).LinkUserIdentity(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "google-oauth2|1234", gotSub)
		assert.Equal(t, idpUUID, gotProvider)
	})
}
