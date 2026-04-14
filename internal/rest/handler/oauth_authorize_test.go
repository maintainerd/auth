package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewOAuthAuthorizeHandler
// ---------------------------------------------------------------------------

func TestNewOAuthAuthorizeHandler(t *testing.T) {
	h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})
	assert.NotNil(t, h)
}

// ---------------------------------------------------------------------------
// Authorize
// ---------------------------------------------------------------------------

func TestOAuthAuthorizeHandler_Authorize_NoUser(t *testing.T) {
	h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})
	r := httptest.NewRequest(http.MethodGet, "/oauth/authorize", nil)
	w := httptest.NewRecorder()

	h.Authorize(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOAuthAuthorizeHandler_Authorize_ValidationError(t *testing.T) {
	h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})
	// Missing required query params.
	r := httptest.NewRequest(http.MethodGet, "/oauth/authorize?response_type=token", nil)
	r = withUser(r)
	w := httptest.NewRecorder()

	h.Authorize(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// validAuthorizeQuery returns a valid query string for the authorize endpoint.
func validAuthorizeQuery() string {
	return "response_type=code&client_id=myapp" +
		"&redirect_uri=https://app.example.com/cb" +
		"&scope=openid&state=abc" +
		"&code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" +
		"&code_challenge_method=S256"
}

func TestOAuthAuthorizeHandler_Authorize_ServiceOAuthError(t *testing.T) {
	svc := &mockOAuthAuthorizeService{
		authorizeFn: func(_ context.Context, _ dto.OAuthAuthorizeRequestDTO, _ int64) (*dto.OAuthAuthorizeResult, *apperror.OAuthError) {
			return nil, apperror.NewOAuthInvalidRequest("bad client")
		},
	}
	h := NewOAuthAuthorizeHandler(svc)
	r := httptest.NewRequest(http.MethodGet,
		"/oauth/authorize?"+validAuthorizeQuery(), nil)
	r = withUser(r)
	w := httptest.NewRecorder()

	h.Authorize(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "invalid_request", body["error"])
	assert.Contains(t, body["error_description"], "bad client")
}

func TestOAuthAuthorizeHandler_Authorize_ConsentRequired(t *testing.T) {
	challengeID := uuid.New().String()
	svc := &mockOAuthAuthorizeService{
		authorizeFn: func(_ context.Context, _ dto.OAuthAuthorizeRequestDTO, _ int64) (*dto.OAuthAuthorizeResult, *apperror.OAuthError) {
			return &dto.OAuthAuthorizeResult{ConsentChallenge: challengeID}, nil
		},
	}
	h := NewOAuthAuthorizeHandler(svc)
	r := httptest.NewRequest(http.MethodGet,
		"/oauth/authorize?"+validAuthorizeQuery(), nil)
	r = withUser(r)
	w := httptest.NewRecorder()

	h.Authorize(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Contains(t, string(body["message"]), "Consent required")

	var data dto.OAuthConsentRequiredResponseDTO
	require.NoError(t, json.Unmarshal(body["data"], &data))
	assert.Equal(t, challengeID, data.ConsentChallenge)
}

func TestOAuthAuthorizeHandler_Authorize_Success(t *testing.T) {
	svc := &mockOAuthAuthorizeService{
		authorizeFn: func(_ context.Context, _ dto.OAuthAuthorizeRequestDTO, _ int64) (*dto.OAuthAuthorizeResult, *apperror.OAuthError) {
			return &dto.OAuthAuthorizeResult{RedirectURI: "https://app.example.com/cb?code=xyz&state=abc"}, nil
		},
	}
	h := NewOAuthAuthorizeHandler(svc)
	r := httptest.NewRequest(http.MethodGet,
		"/oauth/authorize?"+validAuthorizeQuery(), nil)
	r = withUser(r)
	w := httptest.NewRecorder()

	h.Authorize(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Contains(t, string(body["message"]), "Authorization successful")

	var data dto.OAuthAuthorizeResponseDTO
	require.NoError(t, json.Unmarshal(body["data"], &data))
	assert.Equal(t, "https://app.example.com/cb?code=xyz&state=abc", data.RedirectURI)
}

func TestOAuthAuthorizeHandler_Authorize_PassesQueryParams(t *testing.T) {
	var captured dto.OAuthAuthorizeRequestDTO
	svc := &mockOAuthAuthorizeService{
		authorizeFn: func(_ context.Context, req dto.OAuthAuthorizeRequestDTO, _ int64) (*dto.OAuthAuthorizeResult, *apperror.OAuthError) {
			captured = req
			return &dto.OAuthAuthorizeResult{RedirectURI: "https://app.example.com/cb?code=x"}, nil
		},
	}
	h := NewOAuthAuthorizeHandler(svc)
	r := httptest.NewRequest(http.MethodGet,
		"/oauth/authorize?response_type=code&client_id=myapp&redirect_uri=https://app.example.com/cb"+
			"&scope=openid+profile&state=s1&nonce=n1"+
			"&code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"+
			"&code_challenge_method=S256", nil)
	r = withUser(r)
	w := httptest.NewRecorder()

	h.Authorize(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "code", captured.ResponseType)
	assert.Equal(t, "myapp", captured.ClientID)
	assert.Equal(t, "https://app.example.com/cb", captured.RedirectURI)
	assert.Equal(t, "openid profile", captured.Scope)
	assert.Equal(t, "s1", captured.State)
	assert.Equal(t, "n1", captured.Nonce)
	assert.Equal(t, "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", captured.CodeChallenge)
	assert.Equal(t, "S256", captured.CodeChallengeMethod)
}

// ---------------------------------------------------------------------------
// GetConsentChallenge
// ---------------------------------------------------------------------------

func TestOAuthAuthorizeHandler_GetConsentChallenge_NoUser(t *testing.T) {
	h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})
	r := httptest.NewRequest(http.MethodGet, "/oauth/consent/"+testResourceUUID.String(), nil)
	r = withChiParam(r, "challenge_id", testResourceUUID.String())
	w := httptest.NewRecorder()

	h.GetConsentChallenge(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOAuthAuthorizeHandler_GetConsentChallenge_InvalidUUID(t *testing.T) {
	h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})
	r := httptest.NewRequest(http.MethodGet, "/oauth/consent/not-a-uuid", nil)
	r = withUser(r)
	r = withChiParam(r, "challenge_id", "not-a-uuid")
	w := httptest.NewRecorder()

	h.GetConsentChallenge(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthAuthorizeHandler_GetConsentChallenge_ServiceError(t *testing.T) {
	svc := &mockOAuthAuthorizeService{
		getConsentChallengeFn: func(_ context.Context, _ uuid.UUID, _ int64) (*dto.OAuthConsentChallengeResponseDTO, error) {
			return nil, errNotFound
		},
	}
	h := NewOAuthAuthorizeHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/oauth/consent/"+testResourceUUID.String(), nil)
	r = withUser(r)
	r = withChiParam(r, "challenge_id", testResourceUUID.String())
	w := httptest.NewRecorder()

	h.GetConsentChallenge(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOAuthAuthorizeHandler_GetConsentChallenge_Forbidden(t *testing.T) {
	svc := &mockOAuthAuthorizeService{
		getConsentChallengeFn: func(_ context.Context, _ uuid.UUID, _ int64) (*dto.OAuthConsentChallengeResponseDTO, error) {
			return nil, errForbidden
		},
	}
	h := NewOAuthAuthorizeHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/oauth/consent/"+testResourceUUID.String(), nil)
	r = withUser(r)
	r = withChiParam(r, "challenge_id", testResourceUUID.String())
	w := httptest.NewRecorder()

	h.GetConsentChallenge(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestOAuthAuthorizeHandler_GetConsentChallenge_Success(t *testing.T) {
	challengeUUID := testResourceUUID
	resp := &dto.OAuthConsentChallengeResponseDTO{
		ChallengeID: challengeUUID.String(),
		ClientName:  "My App",
		ClientUUID:  uuid.New().String(),
		Scopes:      []string{"openid", "profile"},
		RedirectURI: "https://app.example.com/cb",
		ExpiresAt:   1700000000,
	}
	svc := &mockOAuthAuthorizeService{
		getConsentChallengeFn: func(_ context.Context, _ uuid.UUID, _ int64) (*dto.OAuthConsentChallengeResponseDTO, error) {
			return resp, nil
		},
	}
	h := NewOAuthAuthorizeHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/oauth/consent/"+challengeUUID.String(), nil)
	r = withUser(r)
	r = withChiParam(r, "challenge_id", challengeUUID.String())
	w := httptest.NewRecorder()

	h.GetConsentChallenge(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Contains(t, string(body["message"]), "Consent challenge retrieved")
}

// ---------------------------------------------------------------------------
// HandleConsent
// ---------------------------------------------------------------------------

func TestOAuthAuthorizeHandler_HandleConsent_NoUser(t *testing.T) {
	h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})
	r := jsonReq(t, http.MethodPost, "/oauth/consent", dto.OAuthConsentDecisionDTO{
		ChallengeID: testResourceUUID.String(),
		Approved:    true,
	})
	w := httptest.NewRecorder()

	h.HandleConsent(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOAuthAuthorizeHandler_HandleConsent_InvalidJSON(t *testing.T) {
	h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})
	r := badJSONReq(t, http.MethodPost, "/oauth/consent")
	r = withUser(r)
	w := httptest.NewRecorder()

	h.HandleConsent(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthAuthorizeHandler_HandleConsent_ValidationError(t *testing.T) {
	h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})
	// Empty challenge_id should fail validation.
	r := jsonReq(t, http.MethodPost, "/oauth/consent", dto.OAuthConsentDecisionDTO{
		ChallengeID: "",
		Approved:    true,
	})
	r = withUser(r)
	w := httptest.NewRecorder()

	h.HandleConsent(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthAuthorizeHandler_HandleConsent_InvalidChallengeUUID(t *testing.T) {
	h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})
	r := jsonReq(t, http.MethodPost, "/oauth/consent", dto.OAuthConsentDecisionDTO{
		ChallengeID: "not-a-uuid",
		Approved:    true,
	})
	r = withUser(r)
	w := httptest.NewRecorder()

	h.HandleConsent(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthAuthorizeHandler_HandleConsent_ServiceOAuthError(t *testing.T) {
	svc := &mockOAuthAuthorizeService{
		handleConsentFn: func(_ context.Context, _ dto.OAuthConsentDecisionDTO, _ int64) (*dto.OAuthConsentDecisionResult, *apperror.OAuthError) {
			return nil, apperror.NewOAuthAccessDenied("user denied")
		},
	}
	h := NewOAuthAuthorizeHandler(svc)
	r := jsonReq(t, http.MethodPost, "/oauth/consent", dto.OAuthConsentDecisionDTO{
		ChallengeID: testResourceUUID.String(),
		Approved:    false,
	})
	r = withUser(r)
	w := httptest.NewRecorder()

	h.HandleConsent(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "access_denied", body["error"])
}

func TestOAuthAuthorizeHandler_HandleConsent_Success(t *testing.T) {
	svc := &mockOAuthAuthorizeService{
		handleConsentFn: func(_ context.Context, _ dto.OAuthConsentDecisionDTO, _ int64) (*dto.OAuthConsentDecisionResult, *apperror.OAuthError) {
			return &dto.OAuthConsentDecisionResult{
				RedirectURI: "https://app.example.com/cb?code=xyz&state=abc",
			}, nil
		},
	}
	h := NewOAuthAuthorizeHandler(svc)
	r := jsonReq(t, http.MethodPost, "/oauth/consent", dto.OAuthConsentDecisionDTO{
		ChallengeID: testResourceUUID.String(),
		Approved:    true,
	})
	r = withUser(r)
	w := httptest.NewRecorder()

	h.HandleConsent(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Contains(t, string(body["message"]), "Consent processed")

	var data dto.OAuthConsentDecisionResponseDTO
	require.NoError(t, json.Unmarshal(body["data"], &data))
	assert.Equal(t, "https://app.example.com/cb?code=xyz&state=abc", data.RedirectURI)
}
