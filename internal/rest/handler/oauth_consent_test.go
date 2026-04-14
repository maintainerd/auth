package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewOAuthConsentHandler
// ---------------------------------------------------------------------------

func TestNewOAuthConsentHandler(t *testing.T) {
	h := NewOAuthConsentHandler(&mockOAuthConsentService{})
	assert.NotNil(t, h)
}

// ---------------------------------------------------------------------------
// ListGrants
// ---------------------------------------------------------------------------

func TestOAuthConsentHandler_ListGrants_NoUser(t *testing.T) {
	h := NewOAuthConsentHandler(&mockOAuthConsentService{})
	r := httptest.NewRequest(http.MethodGet, "/oauth/consent/grants", nil)
	w := httptest.NewRecorder()

	h.ListGrants(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOAuthConsentHandler_ListGrants_ServiceError(t *testing.T) {
	svc := &mockOAuthConsentService{
		listGrantsFn: func(_ context.Context, _ int64) ([]dto.OAuthConsentGrantResponseDTO, error) {
			return nil, errNotFound
		},
	}
	h := NewOAuthConsentHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/oauth/consent/grants", nil)
	r = withUser(r)
	w := httptest.NewRecorder()

	h.ListGrants(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOAuthConsentHandler_ListGrants_Success(t *testing.T) {
	grants := []dto.OAuthConsentGrantResponseDTO{
		{
			ConsentGrantUUID: uuid.New().String(),
			ClientName:       "App A",
			ClientUUID:       uuid.New().String(),
			Scopes:           []string{"openid", "profile"},
			GrantedAt:        "2024-01-01T00:00:00Z",
			UpdatedAt:        "2024-01-01T00:00:00Z",
		},
	}
	svc := &mockOAuthConsentService{
		listGrantsFn: func(_ context.Context, _ int64) ([]dto.OAuthConsentGrantResponseDTO, error) {
			return grants, nil
		},
	}
	h := NewOAuthConsentHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/oauth/consent/grants", nil)
	r = withUser(r)
	w := httptest.NewRecorder()

	h.ListGrants(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Contains(t, string(body["message"]), "Consent grants retrieved")
}

func TestOAuthConsentHandler_ListGrants_Empty(t *testing.T) {
	svc := &mockOAuthConsentService{
		listGrantsFn: func(_ context.Context, _ int64) ([]dto.OAuthConsentGrantResponseDTO, error) {
			return []dto.OAuthConsentGrantResponseDTO{}, nil
		},
	}
	h := NewOAuthConsentHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/oauth/consent/grants", nil)
	r = withUser(r)
	w := httptest.NewRecorder()

	h.ListGrants(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// RevokeGrant
// ---------------------------------------------------------------------------

func TestOAuthConsentHandler_RevokeGrant_NoUser(t *testing.T) {
	h := NewOAuthConsentHandler(&mockOAuthConsentService{})
	r := httptest.NewRequest(http.MethodDelete, "/oauth/consent/grants/"+testResourceUUID.String(), nil)
	r = withChiParam(r, "grant_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()

	h.RevokeGrant(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOAuthConsentHandler_RevokeGrant_InvalidUUID(t *testing.T) {
	h := NewOAuthConsentHandler(&mockOAuthConsentService{})
	r := httptest.NewRequest(http.MethodDelete, "/oauth/consent/grants/not-a-uuid", nil)
	r = withUser(r)
	r = withChiParam(r, "grant_uuid", "not-a-uuid")
	w := httptest.NewRecorder()

	h.RevokeGrant(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthConsentHandler_RevokeGrant_ServiceError(t *testing.T) {
	svc := &mockOAuthConsentService{
		revokeGrantFn: func(_ context.Context, _ uuid.UUID, _ int64) error {
			return errNotFound
		},
	}
	h := NewOAuthConsentHandler(svc)
	r := httptest.NewRequest(http.MethodDelete, "/oauth/consent/grants/"+testResourceUUID.String(), nil)
	r = withUser(r)
	r = withChiParam(r, "grant_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()

	h.RevokeGrant(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOAuthConsentHandler_RevokeGrant_Forbidden(t *testing.T) {
	svc := &mockOAuthConsentService{
		revokeGrantFn: func(_ context.Context, _ uuid.UUID, _ int64) error {
			return errForbidden
		},
	}
	h := NewOAuthConsentHandler(svc)
	r := httptest.NewRequest(http.MethodDelete, "/oauth/consent/grants/"+testResourceUUID.String(), nil)
	r = withUser(r)
	r = withChiParam(r, "grant_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()

	h.RevokeGrant(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestOAuthConsentHandler_RevokeGrant_Success(t *testing.T) {
	svc := &mockOAuthConsentService{
		revokeGrantFn: func(_ context.Context, _ uuid.UUID, _ int64) error {
			return nil
		},
	}
	h := NewOAuthConsentHandler(svc)
	r := httptest.NewRequest(http.MethodDelete, "/oauth/consent/grants/"+testResourceUUID.String(), nil)
	r = withUser(r)
	r = withChiParam(r, "grant_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()

	h.RevokeGrant(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Contains(t, string(body["message"]), "Consent grant revoked")
}
