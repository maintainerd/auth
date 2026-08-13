package idp

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	jwtlib "github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

const (
	sloTestHostname   = "https://auth.slo.test"
	sloTestProviderID = "idp-slo"
	sloTestEntityID   = "https://idp.slo.test/entity"
	sloTestIdPSLOURL  = "https://idp.slo.test/slo"
	sloTestNameID     = "nameid-abc"
)

// sloTestFixture wires a federationService with just the collaborators the SLO
// paths touch, plus the throwaway IdP that signs the inbound messages.
type sloTestFixture struct {
	svc      *federationService
	idp      *IdentityProvider
	signer   *samlTestIdP
	sessions *mockSessionService
	sloURL   string
}

func newSLOTestFixture(t *testing.T, mutate ...func(*sloTestFixture)) *sloTestFixture {
	t.Helper()

	prevHost := config.AppPublicHostname
	config.AppPublicHostname = sloTestHostname
	t.Cleanup(func() { config.AppPublicHostname = prevHost })

	signer := newSAMLTestIdP(t)
	cfg, err := json.Marshal(SAMLProviderConfig{
		EntityID:    sloTestEntityID,
		SSOURL:      "https://idp.slo.test/sso",
		SLOURL:      sloTestIdPSLOURL,
		Certificate: signer.certPEM,
	})
	require.NoError(t, err)

	idp := &IdentityProvider{
		IdentityProviderID: 9,
		TenantID:           tenantID,
		Provider:           "saml",
		ProviderType:       "saml",
		Identifier:         sloTestProviderID,
		Status:             "active",
		Config:             datatypes.JSON(cfg),
	}

	sessions := &mockSessionService{}
	fixture := &sloTestFixture{
		idp:      idp,
		signer:   signer,
		sessions: sessions,
		sloURL:   sloTestHostname + "/federation/saml/slo/" + sloTestProviderID,
	}
	fixture.svc = &federationService{
		idpRepo: &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) { return idp, nil },
		},
		userIdentityRepo: &mockFederationUserIdentityRepo{
			findByTenantProviderAndSubFn: func(_ int64, _, sub string) (*UserIdentity, error) {
				if sub != sloTestNameID {
					return nil, nil
				}
				return &UserIdentity{UserIdentityID: 3, UserID: 77, Sub: sloTestNameID}, nil
			},
		},
		userRepo: &mockUserRepo{
			findByIDFn: func(any, ...string) (*User, error) { return &User{UserID: 77}, nil },
		},
		clientRepo:       &mockClientRepo{},
		authEventService: &mockAuthEventService{},
		sessionService:   sessions,
		samlStore:        newFakeSAMLStore(),
	}
	for _, m := range mutate {
		m(fixture)
	}
	return fixture
}

// stubIDTokenHint makes the id_token_hint validator return the given claims.
func stubIDTokenHint(t *testing.T, claims jwtgo.MapClaims, err error) {
	t.Helper()
	prev := idpValidateIDTokenHint
	idpValidateIDTokenHint = func(context.Context, string) (jwtgo.MapClaims, error) {
		return claims, err
	}
	t.Cleanup(func() { idpValidateIDTokenHint = prev })
}

func validIDTokenClaims() jwtgo.MapClaims {
	return jwtgo.MapClaims{"sub": sloTestNameID, "token_type": jwtlib.TokenTypeID}
}

// deflateSAMLRedirect encodes a message for the HTTP-Redirect binding.
func deflateSAMLRedirect(t *testing.T, raw []byte) string {
	t.Helper()
	var compressed bytes.Buffer
	w, err := flate.NewWriter(&compressed, 9)
	require.NoError(t, err)
	_, err = w.Write(raw)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return base64.StdEncoding.EncodeToString(compressed.Bytes())
}

// encodeSAMLPost encodes a message for the HTTP-POST binding (base64, no DEFLATE).
func encodeSAMLPost(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
}

func marshalSAMLElement(t *testing.T, el *etree.Element) []byte {
	t.Helper()
	doc := etree.NewDocument()
	doc.SetRoot(el)
	out, err := doc.WriteToBytes()
	require.NoError(t, err)
	return out
}

func TestInitiateSAMLLogout(t *testing.T) {
	t.Run("redirects to the IdP SLO endpoint and ends local sessions first", func(t *testing.T) {
		f := newSLOTestFixture(t)
		stubIDTokenHint(t, validIDTokenClaims(), nil)

		result, err := f.svc.InitiateSAMLLogout(context.Background(), SAMLLogoutInitiateInput{
			ProviderIdentifier: sloTestProviderID,
			IDTokenHint:        "hint",
		})
		require.NoError(t, err)

		parsed, err := url.Parse(result.RedirectURL)
		require.NoError(t, err)
		assert.Equal(t, "idp.slo.test", parsed.Host)
		assert.Equal(t, "/slo", parsed.Path)
		assert.NotEmpty(t, parsed.Query().Get("SAMLRequest"))

		rs, err := verifyRelayStateForPurpose(parsed.Query().Get("RelayState"), samlRelayPurposeSLO)
		require.NoError(t, err)
		assert.Equal(t, sloTestProviderID, rs.ProviderIdentifier)
		assert.NotEmpty(t, rs.RequestID)

		assert.Equal(t, []int64{77}, f.sessions.revokedAllUserIDs,
			"sessions must be gone before the browser leaves for the IdP — it may never come back")
	})

	t.Run("provider without slo_url is refused", func(t *testing.T) {
		f := newSLOTestFixture(t, func(f *sloTestFixture) {
			cfg, err := json.Marshal(SAMLProviderConfig{
				EntityID:    sloTestEntityID,
				SSOURL:      "https://idp.slo.test/sso",
				Certificate: f.signer.certPEM,
			})
			require.NoError(t, err)
			f.idp.Config = datatypes.JSON(cfg)
		})
		stubIDTokenHint(t, validIDTokenClaims(), nil)

		_, err := f.svc.InitiateSAMLLogout(context.Background(), SAMLLogoutInitiateInput{
			ProviderIdentifier: sloTestProviderID,
			IDTokenHint:        "hint",
		})
		require.Error(t, err)
		assert.Empty(t, f.sessions.revokedAllUserIDs)
	})

	t.Run("an access token is not an id_token_hint", func(t *testing.T) {
		f := newSLOTestFixture(t)
		// Every token this server mints is signed with the same key, so only the
		// token_type claim separates an access token from an ID token.
		stubIDTokenHint(t, jwtgo.MapClaims{"sub": sloTestNameID, "token_type": jwtlib.TokenTypeAccess}, nil)

		_, err := f.svc.InitiateSAMLLogout(context.Background(), SAMLLogoutInitiateInput{
			ProviderIdentifier: sloTestProviderID,
			IDTokenHint:        "hint",
		})
		require.Error(t, err)
		assert.Empty(t, f.sessions.revokedAllUserIDs)
	})

	t.Run("subject that belongs to no identity of this provider is refused", func(t *testing.T) {
		f := newSLOTestFixture(t)
		stubIDTokenHint(t, jwtgo.MapClaims{"sub": "someone-else", "token_type": jwtlib.TokenTypeID}, nil)

		_, err := f.svc.InitiateSAMLLogout(context.Background(), SAMLLogoutInitiateInput{
			ProviderIdentifier: sloTestProviderID,
			IDTokenHint:        "hint",
		})
		require.Error(t, err)
		assert.Empty(t, f.sessions.revokedAllUserIDs)
	})

	t.Run("missing session service fails closed", func(t *testing.T) {
		f := newSLOTestFixture(t, func(f *sloTestFixture) { f.svc.sessionService = nil })
		stubIDTokenHint(t, validIDTokenClaims(), nil)

		_, err := f.svc.InitiateSAMLLogout(context.Background(), SAMLLogoutInitiateInput{
			ProviderIdentifier: sloTestProviderID,
			IDTokenHint:        "hint",
		})
		require.Error(t, err, "never report a logout we could not actually perform")
	})

	t.Run("unregistered post_logout_redirect_uri is refused", func(t *testing.T) {
		f := newSLOTestFixture(t, func(f *sloTestFixture) {
			f.svc.clientRepo = &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
					return &Client{ClientID: 5}, nil
				},
				findRedirectURIsFn: func(int64) ([]ClientURI, error) {
					return []ClientURI{{ClientID: 5, URI: "https://app.test/bye", Type: shared.ClientURITypeLogout}}, nil
				},
			}
		})
		stubIDTokenHint(t, validIDTokenClaims(), nil)

		_, err := f.svc.InitiateSAMLLogout(context.Background(), SAMLLogoutInitiateInput{
			ProviderIdentifier:    sloTestProviderID,
			ClientID:              "app",
			IDTokenHint:           "hint",
			PostLogoutRedirectURI: "https://evil.test/steal",
		})
		require.Error(t, err, "the SLO endpoint redirects here unchecked, so it must be validated now")
		assert.Empty(t, f.sessions.revokedAllUserIDs)
	})

	t.Run("registered logout_uri is sealed into the relay state", func(t *testing.T) {
		f := newSLOTestFixture(t, func(f *sloTestFixture) {
			f.svc.clientRepo = &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
					return &Client{ClientID: 5}, nil
				},
				findRedirectURIsFn: func(int64) ([]ClientURI, error) {
					return []ClientURI{
						{ClientID: 5, URI: "https://app.test/cb", Type: shared.ClientURITypeRedirect},
						{ClientID: 5, URI: "https://app.test/bye", Type: shared.ClientURITypeLogout},
					}, nil
				},
			}
		})
		stubIDTokenHint(t, validIDTokenClaims(), nil)

		result, err := f.svc.InitiateSAMLLogout(context.Background(), SAMLLogoutInitiateInput{
			ProviderIdentifier:    sloTestProviderID,
			ClientID:              "app",
			IDTokenHint:           "hint",
			PostLogoutRedirectURI: "https://app.test/bye",
		})
		require.NoError(t, err)

		parsed, err := url.Parse(result.RedirectURL)
		require.NoError(t, err)
		rs, err := verifyRelayStateForPurpose(parsed.Query().Get("RelayState"), samlRelayPurposeSLO)
		require.NoError(t, err)
		assert.Equal(t, "https://app.test/bye", rs.RedirectURI)
	})

	t.Run("a redirect registered only as a redirect_uri does not qualify", func(t *testing.T) {
		f := newSLOTestFixture(t, func(f *sloTestFixture) {
			f.svc.clientRepo = &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
					return &Client{ClientID: 5}, nil
				},
				findRedirectURIsFn: func(int64) ([]ClientURI, error) {
					return []ClientURI{{ClientID: 5, URI: "https://app.test/cb", Type: shared.ClientURITypeRedirect}}, nil
				},
			}
		})
		stubIDTokenHint(t, validIDTokenClaims(), nil)

		_, err := f.svc.InitiateSAMLLogout(context.Background(), SAMLLogoutInitiateInput{
			ProviderIdentifier:    sloTestProviderID,
			ClientID:              "app",
			IDTokenHint:           "hint",
			PostLogoutRedirectURI: "https://app.test/cb",
		})
		require.Error(t, err)
	})
}

func TestHandleSAMLSingleLogout_IDPInitiated(t *testing.T) {
	newSignedRequest := func(t *testing.T, f *sloTestFixture, id, nameID string) *http.Request {
		t.Helper()
		el := samlTestLogoutRequestXML(id, sloTestEntityID, f.sloURL, nameID, time.Now())
		signed := f.signer.signEnveloped(t, el)
		form := url.Values{"SAMLRequest": {encodeSAMLPost(signed)}}
		req := httptest.NewRequest(http.MethodPost, "/federation/saml/slo/"+sloTestProviderID, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req
	}

	t.Run("signed LogoutRequest ends the subject's sessions and is answered", func(t *testing.T) {
		f := newSLOTestFixture(t)
		result, err := f.svc.HandleSAMLSingleLogout(context.Background(), newSignedRequest(t, f, "req-1", sloTestNameID), sloTestProviderID)
		require.NoError(t, err)
		assert.True(t, result.LoggedOut)
		assert.Equal(t, []int64{77}, f.sessions.revokedAllUserIDs)

		parsed, err := url.Parse(result.RedirectURL)
		require.NoError(t, err)
		assert.Equal(t, "idp.slo.test", parsed.Host)
		assert.NotEmpty(t, parsed.Query().Get("SAMLResponse"))
	})

	t.Run("unsigned LogoutRequest revokes nothing", func(t *testing.T) {
		f := newSLOTestFixture(t)
		el := samlTestLogoutRequestXML("req-2", sloTestEntityID, f.sloURL, sloTestNameID, time.Now())
		form := url.Values{"SAMLRequest": {encodeSAMLPost(marshalSAMLElement(t, el))}}
		req := httptest.NewRequest(http.MethodPost, "/federation/saml/slo/"+sloTestProviderID, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		_, err := f.svc.HandleSAMLSingleLogout(context.Background(), req, sloTestProviderID)
		require.Error(t, err, "the endpoint is unauthenticated — the signature is the whole trust decision")
		assert.Empty(t, f.sessions.revokedAllUserIDs)
	})

	t.Run("LogoutRequest signed by another key revokes nothing", func(t *testing.T) {
		f := newSLOTestFixture(t)
		attacker := newSAMLTestIdP(t)
		el := samlTestLogoutRequestXML("req-3", sloTestEntityID, f.sloURL, sloTestNameID, time.Now())
		form := url.Values{"SAMLRequest": {encodeSAMLPost(attacker.signEnveloped(t, el))}}
		req := httptest.NewRequest(http.MethodPost, "/federation/saml/slo/"+sloTestProviderID, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		_, err := f.svc.HandleSAMLSingleLogout(context.Background(), req, sloTestProviderID)
		require.Error(t, err)
		assert.Empty(t, f.sessions.revokedAllUserIDs)
	})

	t.Run("replayed LogoutRequest is rejected", func(t *testing.T) {
		f := newSLOTestFixture(t)
		_, err := f.svc.HandleSAMLSingleLogout(context.Background(), newSignedRequest(t, f, "req-4", sloTestNameID), sloTestProviderID)
		require.NoError(t, err)

		_, err = f.svc.HandleSAMLSingleLogout(context.Background(), newSignedRequest(t, f, "req-4", sloTestNameID), sloTestProviderID)
		require.Error(t, err, "a captured LogoutRequest must not be replayable to keep killing sessions")
		assert.Len(t, f.sessions.revokedAllUserIDs, 1)
	})

	t.Run("unknown subject is answered without leaking that it is unknown", func(t *testing.T) {
		f := newSLOTestFixture(t)
		result, err := f.svc.HandleSAMLSingleLogout(context.Background(), newSignedRequest(t, f, "req-5", "who-is-this"), sloTestProviderID)
		require.NoError(t, err)
		assert.False(t, result.LoggedOut)
		assert.NotEmpty(t, result.RedirectURL)
		assert.Empty(t, f.sessions.revokedAllUserIDs)
	})

	t.Run("redirect-binding LogoutRequest with an enveloped signature is accepted", func(t *testing.T) {
		// Not every IdP uses the detached query signature on the redirect binding;
		// some deflate an XML-signed document instead.
		f := newSLOTestFixture(t)
		el := samlTestLogoutRequestXML("req-7", sloTestEntityID, f.sloURL, sloTestNameID, time.Now())
		signed := f.signer.signEnveloped(t, el)
		req := httptest.NewRequest(http.MethodGet,
			"/federation/saml/slo/"+sloTestProviderID+"?SAMLRequest="+url.QueryEscape(deflateSAMLRedirect(t, signed)), nil)

		result, err := f.svc.HandleSAMLSingleLogout(context.Background(), req, sloTestProviderID)
		require.NoError(t, err)
		assert.True(t, result.LoggedOut)
	})

	t.Run("redirect-binding LogoutRequest with no signature at all is rejected", func(t *testing.T) {
		f := newSLOTestFixture(t)
		el := samlTestLogoutRequestXML("req-8", sloTestEntityID, f.sloURL, sloTestNameID, time.Now())
		req := httptest.NewRequest(http.MethodGet,
			"/federation/saml/slo/"+sloTestProviderID+"?SAMLRequest="+url.QueryEscape(deflateSAMLRedirect(t, marshalSAMLElement(t, el))), nil)

		_, err := f.svc.HandleSAMLSingleLogout(context.Background(), req, sloTestProviderID)
		require.Error(t, err)
		assert.Empty(t, f.sessions.revokedAllUserIDs)
	})

	t.Run("redirect-binding LogoutRequest is accepted", func(t *testing.T) {
		f := newSLOTestFixture(t)
		el := samlTestLogoutRequestXML("req-6", sloTestEntityID, f.sloURL, sloTestNameID, time.Now())
		query := f.signer.signRedirectQuery(t, "SAMLRequest", string(marshalSAMLElement(t, el)), "")
		req := httptest.NewRequest(http.MethodGet, "/federation/saml/slo/"+sloTestProviderID+"?"+query, nil)

		result, err := f.svc.HandleSAMLSingleLogout(context.Background(), req, sloTestProviderID)
		require.NoError(t, err)
		assert.True(t, result.LoggedOut)
	})
}

func TestHandleSAMLSingleLogout_SPInitiatedRoundTrip(t *testing.T) {
	// Each leg starts its own logout: the RelayState nonce is single-use, so a
	// rejected attempt still consumes it and cannot be reused by the next case.
	startLogout := func(t *testing.T) (*sloTestFixture, string, *samlRelayState) {
		t.Helper()
		f := newSLOTestFixture(t, func(f *sloTestFixture) {
			f.svc.clientRepo = &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) {
					return &Client{ClientID: 5}, nil
				},
				findRedirectURIsFn: func(int64) ([]ClientURI, error) {
					return []ClientURI{{ClientID: 5, URI: "https://app.test/bye", Type: shared.ClientURITypeLogout}}, nil
				},
			}
		})
		stubIDTokenHint(t, validIDTokenClaims(), nil)

		initiated, err := f.svc.InitiateSAMLLogout(context.Background(), SAMLLogoutInitiateInput{
			ProviderIdentifier:    sloTestProviderID,
			ClientID:              "app",
			IDTokenHint:           "hint",
			PostLogoutRedirectURI: "https://app.test/bye",
		})
		require.NoError(t, err)

		parsed, err := url.Parse(initiated.RedirectURL)
		require.NoError(t, err)
		relayState := parsed.Query().Get("RelayState")
		rs, err := verifyRelayStateForPurpose(relayState, samlRelayPurposeSLO)
		require.NoError(t, err)
		return f, relayState, rs
	}

	newResponse := func(t *testing.T, f *sloTestFixture, inResponseTo, relay string) *http.Request {
		t.Helper()
		el := samlTestLogoutResponseXML("res-1", sloTestEntityID, f.sloURL, inResponseTo, samlStatusSuccess, time.Now())
		query := f.signer.signRedirectQuery(t, "SAMLResponse", string(marshalSAMLElement(t, el)), relay)
		return httptest.NewRequest(http.MethodGet, "/federation/saml/slo/"+sloTestProviderID+"?"+query, nil)
	}

	t.Run("matching response lands on the validated post-logout page", func(t *testing.T) {
		f, relayState, rs := startLogout(t)
		result, err := f.svc.HandleSAMLSingleLogout(context.Background(), newResponse(t, f, rs.RequestID, relayState), sloTestProviderID)
		require.NoError(t, err)
		assert.Equal(t, "https://app.test/bye", result.RedirectURL)
		assert.True(t, result.LoggedOut)

		_, err = f.svc.HandleSAMLSingleLogout(context.Background(), newResponse(t, f, rs.RequestID, relayState), sloTestProviderID)
		require.Error(t, err, "the RelayState nonce is single-use")
	})

	t.Run("response answering another request is rejected", func(t *testing.T) {
		f, relayState, _ := startLogout(t)
		_, err := f.svc.HandleSAMLSingleLogout(context.Background(), newResponse(t, f, "some-other-request", relayState), sloTestProviderID)
		require.Error(t, err, "InResponseTo is what ties the answer to the logout we started")
	})

	t.Run("relay state from an SSO flow is refused here", func(t *testing.T) {
		f, _, rs := startLogout(t)
		ssoRelay, err := newSAMLRelayState(sloTestProviderID, "app", "https://app.test/cb", rs.RequestID)
		require.NoError(t, err)
		_, err = f.svc.HandleSAMLSingleLogout(context.Background(), newResponse(t, f, rs.RequestID, ssoRelay), sloTestProviderID)
		require.Error(t, err)
	})
}

// SP metadata is how an IdP learns where to send a logout. Before SLO existed
// the descriptor carried no SingleLogoutService at all, so an admin who filled
// in slo_url still had no way to make the IdP talk to us.
func TestSAMLMetadataAdvertisesSingleLogout(t *testing.T) {
	t.Run("published when the provider has an slo_url", func(t *testing.T) {
		f := newSLOTestFixture(t)
		xmlBytes, err := f.svc.SAMLMetadata(context.Background(), sloTestProviderID)
		require.NoError(t, err)
		assert.Contains(t, string(xmlBytes), "SingleLogoutService")
		assert.Contains(t, string(xmlBytes), f.sloURL)
	})

	t.Run("omitted when the provider has none", func(t *testing.T) {
		f := newSLOTestFixture(t, func(f *sloTestFixture) {
			cfg, err := json.Marshal(SAMLProviderConfig{
				EntityID:    sloTestEntityID,
				SSOURL:      "https://idp.slo.test/sso",
				Certificate: f.signer.certPEM,
			})
			require.NoError(t, err)
			f.idp.Config = datatypes.JSON(cfg)
		})
		xmlBytes, err := f.svc.SAMLMetadata(context.Background(), sloTestProviderID)
		require.NoError(t, err)
		assert.NotContains(t, string(xmlBytes), "SingleLogoutService",
			"advertising an endpoint we cannot complete invites logouts that never finish")
	})
}

func TestSAMLLogoutHandlers(t *testing.T) {
	t.Run("initiate redirects the browser to the IdP", func(t *testing.T) {
		svc := &mockFederationService{}
		req := httptest.NewRequest(http.MethodGet, "/federation/saml/logout?provider_identifier=p&id_token_hint=hint", nil)
		w := httptest.NewRecorder()
		NewFederationHandler(svc).InitiateSAMLLogout(w, req)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "https://idp.example.com/saml/slo?SAMLRequest=abc", w.Header().Get("Location"))
	})

	t.Run("initiate without an id_token_hint is refused", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/federation/saml/logout?provider_identifier=p", nil)
		w := httptest.NewRecorder()
		NewFederationHandler(&mockFederationService{}).InitiateSAMLLogout(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("slo endpoint answers with a redirect when one is produced", func(t *testing.T) {
		svc := &mockFederationService{
			handleSAMLSingleLogoutFn: func(string) (*SAMLSingleLogoutResult, error) {
				return &SAMLSingleLogoutResult{RedirectURL: "https://idp.example.com/slo?SAMLResponse=x", LoggedOut: true}, nil
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/federation/saml/slo/p", nil)
		req = withChiParam(req, "provider_identifier", "p")
		w := httptest.NewRecorder()
		NewFederationHandler(svc).SAMLSingleLogout(w, req)
		assert.Equal(t, http.StatusFound, w.Code)
	})

	t.Run("slo endpoint surfaces a rejected message as an error", func(t *testing.T) {
		svc := &mockFederationService{
			handleSAMLSingleLogoutFn: func(string) (*SAMLSingleLogoutResult, error) {
				return nil, apperror.NewUnauthorized("SAML logout message rejected")
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/federation/saml/slo/p", nil)
		req = withChiParam(req, "provider_identifier", "p")
		w := httptest.NewRecorder()
		NewFederationHandler(svc).SAMLSingleLogout(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
