package idp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
)

type mockFederationService struct {
	exchangeExternalTokenFn      func(req FederationTokenRequestDTO) (*LoginResponseDTO, error)
	exchangeOAuth2CodeFn         func(req FederationOAuth2CallbackDTO) (*LoginResponseDTO, error)
	homeRealmDiscoveryFn         func(tenantID int64, email string) (*HRDResponseDTO, error)
	homeRealmDiscoveryByClientFn func(clientID string, email string) (*HRDResponseDTO, error)
	resolveBrokerProviderFn      func(idpIdentifier string) (*BrokerProviderInfo, error)
	resolveBrokerUserFn          func(idpID int64, code, verifier, nonce, redirectURI string, clientID int64) (*BrokerResolvedUser, error)
	getUserIdentitiesFn          func(userID int64) ([]IdentityDTO, error)
	linkIdentityFn               func(userID int64, req LinkIdentityRequestDTO) (*IdentityDTO, error)
	unlinkIdentityFn             func(userID int64, identityUUID string) error
	adminUnlinkIdentityFn        func(tenantID int64, actorUserID int64, userUUID uuid.UUID, identityUUID string) error
}

func (m *mockFederationService) ExchangeExternalToken(_ context.Context, req FederationTokenRequestDTO) (*LoginResponseDTO, error) {
	if m.exchangeExternalTokenFn != nil {
		return m.exchangeExternalTokenFn(req)
	}
	return &LoginResponseDTO{AccessToken: "at"}, nil
}
func (m *mockFederationService) ExchangeOAuth2Code(_ context.Context, req FederationOAuth2CallbackDTO) (*LoginResponseDTO, error) {
	if m.exchangeOAuth2CodeFn != nil {
		return m.exchangeOAuth2CodeFn(req)
	}
	return &LoginResponseDTO{AccessToken: "at"}, nil
}
func (m *mockFederationService) HomeRealmDiscovery(_ context.Context, tenantID int64, email string) (*HRDResponseDTO, error) {
	if m.homeRealmDiscoveryFn != nil {
		return m.homeRealmDiscoveryFn(tenantID, email)
	}
	return &HRDResponseDTO{ProviderIdentifier: "idp-1"}, nil
}
func (m *mockFederationService) HomeRealmDiscoveryByClient(_ context.Context, clientID string, email string) (*HRDResponseDTO, error) {
	if m.homeRealmDiscoveryByClientFn != nil {
		return m.homeRealmDiscoveryByClientFn(clientID, email)
	}
	return &HRDResponseDTO{ProviderIdentifier: "idp-1"}, nil
}
func (m *mockFederationService) ResolveBrokerProvider(_ context.Context, idpIdentifier string) (*BrokerProviderInfo, error) {
	if m.resolveBrokerProviderFn != nil {
		return m.resolveBrokerProviderFn(idpIdentifier)
	}
	return &BrokerProviderInfo{AuthorizationEndpoint: "https://idp.example.com/authorize", ClientID: "upstream-client"}, nil
}
func (m *mockFederationService) ResolveBrokerUser(_ context.Context, idpID int64, code, verifier, nonce, redirectURI string, clientID int64) (*BrokerResolvedUser, error) {
	if m.resolveBrokerUserFn != nil {
		return m.resolveBrokerUserFn(idpID, code, verifier, nonce, redirectURI, clientID)
	}
	return &BrokerResolvedUser{UserID: 1, IdentitySub: "sub-123"}, nil
}
func (m *mockFederationService) GetUserIdentities(_ context.Context, userID int64) ([]IdentityDTO, error) {
	if m.getUserIdentitiesFn != nil {
		return m.getUserIdentitiesFn(userID)
	}
	return nil, nil
}
func (m *mockFederationService) LinkIdentity(_ context.Context, userID int64, req LinkIdentityRequestDTO) (*IdentityDTO, error) {
	if m.linkIdentityFn != nil {
		return m.linkIdentityFn(userID, req)
	}
	return &IdentityDTO{Provider: "google"}, nil
}
func (m *mockFederationService) UnlinkIdentity(_ context.Context, userID int64, identityUUID string) error {
	if m.unlinkIdentityFn != nil {
		return m.unlinkIdentityFn(userID, identityUUID)
	}
	return nil
}
func (m *mockFederationService) AdminUnlinkIdentity(_ context.Context, tenantID int64, actorUserID int64, userUUID uuid.UUID, identityUUID string) error {
	if m.adminUnlinkIdentityFn != nil {
		return m.adminUnlinkIdentityFn(tenantID, actorUserID, userUUID, identityUUID)
	}
	return nil
}

func (m *mockFederationService) InitiateSAMLSSO(_ context.Context, _ SAMLInitiateInput) (*SAMLInitiateResult, error) {
	return &SAMLInitiateResult{RedirectURL: "https://idp.example.com/saml/sso"}, nil
}
func (m *mockFederationService) HandleSAMLResponse(_ context.Context, _ *http.Request, _ string) (*SAMLCallbackResult, error) {
	return &SAMLCallbackResult{RedirectURI: "https://app.example.com/callback?code=abc", Code: "abc"}, nil
}
func (m *mockFederationService) ExchangeSAMLCode(_ context.Context, _ string) (*LoginResponseDTO, error) {
	return &LoginResponseDTO{AccessToken: "at"}, nil
}
func (m *mockFederationService) SAMLMetadata(_ context.Context, _ string) ([]byte, error) {
	return []byte(`<?xml version="1.0"?><EntityDescriptor/>`), nil
}

func (m *mockFederationService) TestConnection(_ context.Context, _ TestConnectionRequestDTO) (*TestConnectionResultDTO, error) {
	return &TestConnectionResultDTO{Success: true}, nil
}

func TestFederationHandler_ExchangeExternalToken(t *testing.T) {
	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPost, "/federation/token")
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).ExchangeExternalToken(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing provider_identifier returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/federation/token", map[string]string{"external_token": "tok", "client_id": "app"})
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).ExchangeExternalToken(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockFederationService{
			exchangeExternalTokenFn: func(FederationTokenRequestDTO) (*LoginResponseDTO, error) {
				return nil, apperror.NewUnauthorized("invalid token")
			},
		}
		r := jsonReq(t, http.MethodPost, "/federation/token", map[string]string{
			"provider_identifier": "idp-1", "external_token": "tok", "client_id": "app",
		})
		w := httptest.NewRecorder()
		NewFederationHandler(svc).ExchangeExternalToken(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/federation/token", map[string]string{
			"provider_identifier": "idp-1", "external_token": "tok", "client_id": "app",
		})
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).ExchangeExternalToken(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestFederationHandler_ExchangeOAuth2Code(t *testing.T) {
	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPost, "/federation/oauth2/callback")
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).ExchangeOAuth2Code(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing provider_identifier returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/federation/oauth2/callback", map[string]string{"code": "c", "redirect_uri": "https://x.com", "client_id": "app"})
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).ExchangeOAuth2Code(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockFederationService{
			exchangeOAuth2CodeFn: func(FederationOAuth2CallbackDTO) (*LoginResponseDTO, error) {
				return nil, errors.New("exchange failed")
			},
		}
		r := jsonReq(t, http.MethodPost, "/federation/oauth2/callback", map[string]string{
			"provider_identifier": "idp-1", "code": "c", "redirect_uri": "https://x.com", "client_id": "app",
		})
		w := httptest.NewRecorder()
		NewFederationHandler(svc).ExchangeOAuth2Code(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/federation/oauth2/callback", map[string]string{
			"provider_identifier": "idp-1", "code": "c", "redirect_uri": "https://x.com", "client_id": "app",
		})
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).ExchangeOAuth2Code(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestFederationHandler_HomeRealmDiscovery(t *testing.T) {
	t.Run("missing email returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/federation/hrd?tenant_id=1", nil)
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).HomeRealmDiscovery(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing client_id and tenant_id returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/federation/hrd?email=user@example.com", nil)
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).HomeRealmDiscovery(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("client_id success returns 200", func(t *testing.T) {
		var capturedClientID string
		svc := &mockFederationService{
			homeRealmDiscoveryByClientFn: func(clientID, _ string) (*HRDResponseDTO, error) {
				capturedClientID = clientID
				return &HRDResponseDTO{ProviderIdentifier: "idp-1"}, nil
			},
		}
		r := httptest.NewRequest(http.MethodGet, "/federation/hrd?email=user@example.com&client_id=app", nil)
		w := httptest.NewRecorder()
		NewFederationHandler(svc).HomeRealmDiscovery(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "app", capturedClientID)
	})

	t.Run("invalid tenant_id returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/federation/hrd?email=user@example.com&tenant_id=bad", nil)
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).HomeRealmDiscovery(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockFederationService{
			homeRealmDiscoveryFn: func(int64, string) (*HRDResponseDTO, error) {
				return nil, errors.New("not found")
			},
		}
		r := httptest.NewRequest(http.MethodGet, "/federation/hrd?email=user@example.com&tenant_id=1", nil)
		w := httptest.NewRecorder()
		NewFederationHandler(svc).HomeRealmDiscovery(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/federation/hrd?email=user@example.com&tenant_id=1", nil)
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).HomeRealmDiscovery(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestFederationHandler_GetIdentities(t *testing.T) {
	t.Run("no user returns 401", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/account/identities", nil)
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).GetIdentities(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockFederationService{
			getUserIdentitiesFn: func(int64) ([]IdentityDTO, error) {
				return nil, errors.New("db error")
			},
		}
		r := withTenantAndUser(httptest.NewRequest(http.MethodGet, "/account/identities", nil))
		w := httptest.NewRecorder()
		NewFederationHandler(svc).GetIdentities(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := withTenantAndUser(httptest.NewRequest(http.MethodGet, "/account/identities", nil))
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).GetIdentities(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestFederationHandler_LinkIdentity(t *testing.T) {
	t.Run("no user returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/account/identities/link", map[string]string{"provider_identifier": "idp-1", "external_token": "tok"})
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).LinkIdentity(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withTenantAndUser(badJSONReq(t, http.MethodPost, "/account/identities/link"))
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).LinkIdentity(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing fields returns 400", func(t *testing.T) {
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/account/identities/link", map[string]string{}))
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).LinkIdentity(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockFederationService{
			linkIdentityFn: func(int64, LinkIdentityRequestDTO) (*IdentityDTO, error) {
				return nil, errors.New("link failed")
			},
		}
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/account/identities/link", map[string]string{
			"provider_identifier": "idp-1", "external_token": "tok",
		}))
		w := httptest.NewRecorder()
		NewFederationHandler(svc).LinkIdentity(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/account/identities/link", map[string]string{
			"provider_identifier": "idp-1", "external_token": "tok",
		}))
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).LinkIdentity(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestFederationHandler_UnlinkIdentity(t *testing.T) {
	t.Run("no user returns 401", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "identity_uuid", "id-1")
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).UnlinkIdentity(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockFederationService{
			unlinkIdentityFn: func(int64, string) error { return errors.New("not found") },
		}
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "identity_uuid", "id-1"))
		w := httptest.NewRecorder()
		NewFederationHandler(svc).UnlinkIdentity(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "identity_uuid", "id-1"))
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).UnlinkIdentity(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
