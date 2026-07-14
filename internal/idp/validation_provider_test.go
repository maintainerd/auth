package idp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

// validProviderConfigJSON returns a config JSON that contains ONLY allow-listed
// keys for an OIDC provider. Unlike validOAuth2ConfigJSON (which still carries the
// legacy issuer/client_id/client_secret keys promoted to columns), this fixture
// passes the strict config allow-list. It deliberately carries NO endpoint keys:
// OIDC providers derive endpoints from discovery, so endpoint overrides are
// rejected (an override could point the token endpoint at an attacker host and
// leak the upstream client secret).
func validProviderConfigJSON() datatypes.JSON {
	return datatypes.JSON(`{"scopes":["openid","email"]}`)
}

func validIDPCreate() IdentityProviderCreateRequestDTO {
	return IdentityProviderCreateRequestDTO{
		Name:                 "my-idp",
		DisplayName:          "My Identity Provider",
		Provider:             shared.IDPProviderGoogle,
		ProviderType:         shared.IDPTypeIdentity,
		Issuer:               "https://accounts.google.com",
		ProviderClientID:     "test-client",
		ProviderClientSecret: "test-secret",
		Config:               validProviderConfigJSON(),
		Status:               shared.StatusActive,
	}
}

func TestIdentityProviderCreateRequestDto_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, validIDPCreate().Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := validIDPCreate()
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("name too short", func(t *testing.T) {
		d := validIDPCreate()
		d.Name = "ab"
		require.Error(t, d.Validate())
	})

	t.Run("display_name too short", func(t *testing.T) {
		d := validIDPCreate()
		d.DisplayName = "a"
		require.Error(t, d.Validate())
	})

	t.Run("invalid provider", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = "yahoo"
		require.Error(t, d.Validate())
	})

	t.Run("enterprise provider_type is valid", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeEnterprise
		require.NoError(t, d.Validate())
	})

	t.Run("config is optional", func(t *testing.T) {
		d := validIDPCreate()
		d.Config = nil
		require.NoError(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validIDPCreate()
		d.Status = "unknown"
		require.Error(t, d.Validate())
	})

	t.Run("active social missing client_id is invalid", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeSocial
		d.ProviderClientID = ""
		require.Error(t, d.Validate())
	})

	t.Run("active social missing issuer is invalid", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeSocial
		d.Issuer = ""
		require.Error(t, d.Validate())
	})

	t.Run("active external missing client_secret is invalid on create", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeSocial
		d.ProviderClientSecret = ""
		require.Error(t, d.Validate())
	})

	t.Run("inactive external missing client_secret is allowed (draft)", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeSocial
		d.Status = shared.StatusInactive
		d.ProviderClientSecret = ""
		require.NoError(t, d.Validate())
	})

	t.Run("active social with non-url issuer is invalid", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeSocial
		d.Issuer = "not-a-url"
		require.Error(t, d.Validate())
	})

	t.Run("enterprise OIDC with issuer only is valid", func(t *testing.T) {
		// gitlab is a variable-domain OIDC provider (self-managed): its issuer host
		// is not restricted, and endpoints are obtained via OIDC discovery.
		d := validIDPCreate()
		d.Provider = shared.IDPProviderGitLab
		d.ProviderType = shared.IDPTypeEnterprise
		d.Issuer = "https://idp.example.com"
		d.ProviderClientID = "abc"
		d.Config = datatypes.JSON(`{"scopes":["openid","email"]}`)
		require.NoError(t, d.Validate())
	})

	t.Run("enterprise OIDC with endpoint override is rejected", func(t *testing.T) {
		// OIDC providers must derive endpoints from discovery; an operator-supplied
		// token_endpoint could be pointed at an attacker host and leak the secret.
		d := validIDPCreate()
		d.Provider = shared.IDPProviderGitLab
		d.ProviderType = shared.IDPTypeEnterprise
		d.Issuer = "https://idp.example.com"
		d.ProviderClientID = "abc"
		d.Config = datatypes.JSON(`{"authorization_endpoint":"https://idp.example.com/authorize","token_endpoint":"https://idp.example.com/token"}`)
		require.Error(t, d.Validate())
	})

	t.Run("invalid email domain is rejected", func(t *testing.T) {
		d := validIDPCreate()
		d.EmailDomains = []string{"bad domain!"}
		require.Error(t, d.Validate())
	})

	t.Run("valid email domains pass", func(t *testing.T) {
		d := validIDPCreate()
		d.EmailDomains = []string{"example.com", "sub.example.org"}
		require.NoError(t, d.Validate())
	})

	t.Run("inactive social without creds is allowed (draft)", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeSocial
		d.Status = shared.StatusInactive
		d.Issuer = ""
		d.ProviderClientID = ""
		require.NoError(t, d.Validate())
	})

	t.Run("system provider type skips external creds rule", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderMaintainerd
		d.ProviderType = shared.IDPTypeSystem
		d.Issuer = ""
		d.ProviderClientID = ""
		d.Config = datatypes.JSON(`{}`)
		require.NoError(t, d.Validate())
	})
}

func TestIdentityProviderUpdateRequestDto_Validate(t *testing.T) {
	d := IdentityProviderUpdateRequestDTO{
		Name:         "my-idp",
		DisplayName:  "My Identity Provider",
		Provider:     shared.IDPProviderMaintainerd,
		ProviderType: shared.IDPTypeSystem,
		Config:       validOAuth2ConfigJSON(),
		Status:       shared.StatusInactive,
	}
	assert.NoError(t, d.Validate())

	d.Name = ""
	require.Error(t, d.Validate())
}

func TestIdentityProviderStatusUpdateDto_Validate(t *testing.T) {
	assert.NoError(t, IdentityProviderStatusUpdateDTO{Status: shared.StatusActive}.Validate())
	require.Error(t, IdentityProviderStatusUpdateDTO{Status: "bad"}.Validate())
	require.Error(t, IdentityProviderStatusUpdateDTO{Status: ""}.Validate())
}

func TestIdentityProviderFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := IdentityProviderFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid provider in list", func(t *testing.T) {
		f := IdentityProviderFilterDTO{
			PaginationRequestDTO: validPagination(),
			Provider:             []string{"yahoo"},
		}
		require.Error(t, f.Validate())
	})

	t.Run("enterprise provider_type is valid", func(t *testing.T) {
		pt := "enterprise"
		f := IdentityProviderFilterDTO{PaginationRequestDTO: validPagination(), ProviderType: &pt}
		require.NoError(t, f.Validate())
	})

	t.Run("invalid status in list", func(t *testing.T) {
		f := IdentityProviderFilterDTO{
			PaginationRequestDTO: validPagination(),
			Status:               []string{"bad"},
		}
		require.Error(t, f.Validate())
	})
}

func TestIdentityProviderCreateValidation_TokenFederation(t *testing.T) {
	t.Run("token federation enabled with issuer and audiences passes", func(t *testing.T) {
		d := validIDPCreate()
		d.AllowTokenFederation = true
		d.AllowedAudiences = []string{"app-1"}
		assert.NoError(t, d.Validate())
	})

	t.Run("token federation enabled requires audiences", func(t *testing.T) {
		d := validIDPCreate()
		d.AllowTokenFederation = true
		d.AllowedAudiences = nil
		require.Error(t, d.Validate())
	})

	t.Run("token federation enabled requires issuer", func(t *testing.T) {
		d := validIDPCreate()
		d.AllowTokenFederation = true
		d.AllowedAudiences = []string{"app-1"}
		d.Issuer = ""
		require.Error(t, d.Validate())
	})

	t.Run("token federation off with no audiences is ok", func(t *testing.T) {
		d := validIDPCreate()
		d.AllowTokenFederation = false
		d.AllowedAudiences = nil
		assert.NoError(t, d.Validate())
	})
}

func TestIdentityProviderUpdateValidation_TokenFederation(t *testing.T) {
	t.Run("token federation enabled with audiences passes", func(t *testing.T) {
		d := validIDPUpdate()
		d.AllowTokenFederation = true
		d.AllowedAudiences = []string{"app-1"}
		assert.NoError(t, d.Validate())
	})

	t.Run("token federation enabled requires audiences", func(t *testing.T) {
		d := validIDPUpdate()
		d.AllowTokenFederation = true
		d.AllowedAudiences = nil
		require.Error(t, d.Validate())
	})
}

// Secret is write-only on update: an active external provider may be updated
// without resupplying the client secret (blank keeps the stored value).
func TestIdentityProviderUpdateValidation_SecretOptional(t *testing.T) {
	d := validIDPUpdate()
	d.ProviderClientSecret = ""
	require.NoError(t, d.Validate())
}

func validIDPUpdate() IdentityProviderUpdateRequestDTO {
	return IdentityProviderUpdateRequestDTO{
		Name:             "my-idp",
		DisplayName:      "My Identity Provider",
		Provider:         shared.IDPProviderGoogle,
		ProviderType:     shared.IDPTypeSocial,
		Issuer:           "https://accounts.google.com",
		ProviderClientID: "test-client",
		Config:           validProviderConfigJSON(),
		Status:           shared.StatusActive,
	}
}

// B3 — name slug format rule (^[a-z0-9-]+$).
func TestIdentityProviderValidation_NameSlugFormat(t *testing.T) {
	t.Run("create rejects uppercase letters", func(t *testing.T) {
		d := validIDPCreate()
		d.Name = "MyIdp"
		require.Error(t, d.Validate())
	})

	t.Run("create rejects underscores and spaces", func(t *testing.T) {
		d := validIDPCreate()
		d.Name = "my_idp name"
		require.Error(t, d.Validate())
	})

	t.Run("create rejects colon (allowed for roles, not idp)", func(t *testing.T) {
		d := validIDPCreate()
		d.Name = "my:idp"
		require.Error(t, d.Validate())
	})

	t.Run("create accepts lowercase, digits and hyphens", func(t *testing.T) {
		d := validIDPCreate()
		d.Name = "my-idp-2"
		require.NoError(t, d.Validate())
	})

	t.Run("update rejects invalid name chars", func(t *testing.T) {
		d := validIDPUpdate()
		d.Name = "Bad Name"
		require.Error(t, d.Validate())
	})
}

// B4 — OIDC config endpoint URL validation.
// Endpoint URL validity is exercised against github, an OAuth2-only provider
// whose explicit endpoints are legitimately present (and host-bound). OIDC
// providers may not carry endpoint keys at all (see the endpoint-override tests),
// so the URL rule can only be exercised on an OAuth2-only provider.
func TestIdentityProviderValidation_OIDCEndpointURLs(t *testing.T) {
	// githubEndpoints returns an active github DTO with the three official
	// endpoints; individual sub-tests overwrite one endpoint to a bad value.
	githubDTO := func() IdentityProviderCreateRequestDTO {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderGitHub
		d.ProviderType = shared.IDPTypeSocial
		d.Issuer = ""
		d.ProviderClientID = "cid"
		d.ProviderClientSecret = "sec"
		d.Config = datatypes.JSON(`{"authorization_endpoint":"https://github.com/login/oauth/authorize","token_endpoint":"https://github.com/login/oauth/access_token","userinfo_endpoint":"https://api.github.com/user"}`)
		return d
	}

	t.Run("create rejects non-url authorization_endpoint", func(t *testing.T) {
		d := githubDTO()
		d.Config = datatypes.JSON(`{"authorization_endpoint":"not-a-url","token_endpoint":"https://github.com/login/oauth/access_token","userinfo_endpoint":"https://api.github.com/user"}`)
		require.Error(t, d.Validate())
	})

	t.Run("create rejects non-url token_endpoint", func(t *testing.T) {
		d := githubDTO()
		d.Config = datatypes.JSON(`{"authorization_endpoint":"https://github.com/login/oauth/authorize","token_endpoint":"not-a-url","userinfo_endpoint":"https://api.github.com/user"}`)
		require.Error(t, d.Validate())
	})

	t.Run("create rejects non-url userinfo_endpoint", func(t *testing.T) {
		d := githubDTO()
		d.Config = datatypes.JSON(`{"authorization_endpoint":"https://github.com/login/oauth/authorize","token_endpoint":"https://github.com/login/oauth/access_token","userinfo_endpoint":"not-a-url"}`)
		require.Error(t, d.Validate())
	})

	t.Run("create accepts valid oauth2-only endpoints", func(t *testing.T) {
		require.NoError(t, githubDTO().Validate())
	})

	t.Run("create allows OIDC without endpoints (issuer discovery)", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeEnterprise
		d.Config = datatypes.JSON(`{"scopes":["openid"]}`)
		require.NoError(t, d.Validate())
	})

	// FIX D: OIDC providers must derive endpoints from discovery, so any endpoint
	// override is rejected outright (regardless of https/host validity).
	t.Run("create rejects OIDC authorization_endpoint override", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeEnterprise
		d.Config = datatypes.JSON(`{"authorization_endpoint":"https://idp.example.com/authorize"}`)
		require.Error(t, d.Validate())
	})

	t.Run("create rejects OIDC token_endpoint override", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeEnterprise
		d.Config = datatypes.JSON(`{"token_endpoint":"https://evil.com/token"}`)
		require.Error(t, d.Validate())
	})

	t.Run("create rejects OIDC userinfo_endpoint override", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeEnterprise
		d.Config = datatypes.JSON(`{"userinfo_endpoint":"https://idp.example.com/userinfo"}`)
		require.Error(t, d.Validate())
	})

	t.Run("update rejects OIDC endpoint override", func(t *testing.T) {
		d := validIDPUpdate()
		d.ProviderType = shared.IDPTypeEnterprise
		d.Config = datatypes.JSON(`{"token_endpoint":"https://idp.example.com/token"}`)
		require.Error(t, d.Validate())
	})
}

// B5 — provider / provider_type consistency.
func TestIdentityProviderValidation_ProviderTypeConsistency(t *testing.T) {
	t.Run("create rejects maintainerd with non-system type", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderMaintainerd
		d.ProviderType = shared.IDPTypeSocial
		require.Error(t, d.Validate())
	})

	t.Run("create rejects system type with non-maintainerd provider", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderGoogle
		d.ProviderType = shared.IDPTypeSystem
		require.Error(t, d.Validate())
	})

	t.Run("create rejects saml provider with non-saml type", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderSAML
		d.ProviderType = shared.IDPTypeEnterprise
		require.Error(t, d.Validate())
	})

	t.Run("create accepts maintainerd with system type", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderMaintainerd
		d.ProviderType = shared.IDPTypeSystem
		d.Issuer = ""
		d.ProviderClientID = ""
		d.Config = datatypes.JSON(`{}`)
		require.NoError(t, d.Validate())
	})

	// External Maintainerd registered as a federated OIDC IdP: provider is
	// "maintainerd" but the type is "enterprise", not "system".
	t.Run("create accepts maintainerd with enterprise type", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderMaintainerd
		d.ProviderType = shared.IDPTypeEnterprise
		require.NoError(t, d.Validate())
	})

	t.Run("create rejects maintainerd with social type", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderMaintainerd
		d.ProviderType = shared.IDPTypeSocial
		require.Error(t, d.Validate())
	})

	t.Run("create rejects maintainerd with saml type", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderMaintainerd
		d.ProviderType = shared.IDPTypeSAML
		require.Error(t, d.Validate())
	})

	t.Run("update accepts maintainerd with enterprise type", func(t *testing.T) {
		d := validIDPUpdate()
		d.Provider = shared.IDPProviderMaintainerd
		d.ProviderType = shared.IDPTypeEnterprise
		require.NoError(t, d.Validate())
	})

	t.Run("update rejects maintainerd with non-system/enterprise type", func(t *testing.T) {
		d := validIDPUpdate()
		d.Provider = shared.IDPProviderMaintainerd
		d.ProviderType = shared.IDPTypeSocial
		require.Error(t, d.Validate())
	})
}

// selfSignedCertPEM generates a valid self-signed X.509 certificate in PEM form
// so SAML config validation (which parses the certificate) can be exercised
// against a real cert without hardcoding a fixture.
func selfSignedCertPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-idp"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

// Provider-aware config validation: OAuth2-only vs OIDC issuer/endpoints, config
// allow-list, scope-token format, attribute-mapping targets and SAML config.
// Mirrors the frontend spec.
func TestIdentityProviderValidation_ProviderAwareConfig(t *testing.T) {
	t.Run("oauth2-only github active without endpoints is rejected", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderGitHub
		d.ProviderType = shared.IDPTypeSocial
		d.Issuer = ""
		d.Config = datatypes.JSON(`{"scopes":["read:user"]}`)
		require.Error(t, d.Validate())
	})

	t.Run("oauth2-only github active with all three endpoints is accepted", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderGitHub
		d.ProviderType = shared.IDPTypeSocial
		d.Issuer = ""
		d.Config = datatypes.JSON(`{"authorization_endpoint":"https://github.com/login/oauth/authorize","token_endpoint":"https://github.com/login/oauth/access_token","userinfo_endpoint":"https://api.github.com/user"}`)
		require.NoError(t, d.Validate())
	})

	t.Run("oauth2-only github inactive without endpoints is allowed (draft)", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderGitHub
		d.ProviderType = shared.IDPTypeSocial
		d.Status = shared.StatusInactive
		d.Issuer = ""
		d.Config = datatypes.JSON(`{"scopes":["read:user"]}`)
		require.NoError(t, d.Validate())
	})

	t.Run("oidc google active still requires issuer", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderGoogle
		d.ProviderType = shared.IDPTypeSocial
		d.Issuer = ""
		require.Error(t, d.Validate())
	})

	t.Run("unknown config key on cognito is rejected", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderCognito
		d.ProviderType = shared.IDPTypeEnterprise
		d.Config = datatypes.JSON(`{"region":"x"}`)
		require.Error(t, d.Validate())
	})

	t.Run("bad scope token is rejected", func(t *testing.T) {
		d := validIDPCreate()
		d.Config = datatypes.JSON(`{"scopes":["openid","bad scope!"]}`)
		require.Error(t, d.Validate())
	})

	t.Run("valid scopes are accepted", func(t *testing.T) {
		d := validIDPCreate()
		d.Config = datatypes.JSON(`{"scopes":["openid","email","profile","https://www.googleapis.com/auth/userinfo.email"]}`)
		require.NoError(t, d.Validate())
	})

	t.Run("attribute_mapping with unknown target is rejected", func(t *testing.T) {
		d := validIDPCreate()
		d.Config = datatypes.JSON(`{"attribute_mapping":{"not_a_target":"email"}}`)
		require.Error(t, d.Validate())
	})

	t.Run("attribute_mapping with known target is accepted", func(t *testing.T) {
		d := validIDPCreate()
		d.Config = datatypes.JSON(`{"attribute_mapping":{"email":"mail","name":"display_name"}}`)
		require.NoError(t, d.Validate())
	})

	t.Run("update oauth2-only twitter active without endpoints is rejected", func(t *testing.T) {
		d := validIDPUpdate()
		d.Provider = shared.IDPProviderTwitter
		d.ProviderType = shared.IDPTypeSocial
		d.Issuer = ""
		d.Config = datatypes.JSON(`{"scopes":["tweet.read"]}`)
		require.Error(t, d.Validate())
	})

	t.Run("saml unknown config key is rejected", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderSAML
		d.ProviderType = shared.IDPTypeSAML
		d.Issuer = ""
		d.ProviderClientID = ""
		d.ProviderClientSecret = ""
		d.Config = datatypes.JSON(`{"entity_id":"urn:idp","sso_url":"https://idp.example.com/sso","certificate":"x","bogus":"y"}`)
		require.Error(t, d.Validate())
	})

	t.Run("saml bad certificate is rejected", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderSAML
		d.ProviderType = shared.IDPTypeSAML
		d.Issuer = ""
		d.ProviderClientID = ""
		d.ProviderClientSecret = ""
		d.Config = datatypes.JSON(`{"entity_id":"urn:idp","sso_url":"https://idp.example.com/sso","certificate":"not-a-pem-cert"}`)
		require.Error(t, d.Validate())
	})

	t.Run("saml active with valid config is accepted", func(t *testing.T) {
		cert, _ := json.Marshal(selfSignedCertPEM(t))
		d := validIDPCreate()
		d.Provider = shared.IDPProviderSAML
		d.ProviderType = shared.IDPTypeSAML
		d.Issuer = ""
		d.ProviderClientID = ""
		d.ProviderClientSecret = ""
		d.Config = datatypes.JSON(`{"entity_id":"urn:idp","sso_url":"https://idp.example.com/sso","attribute_mapping":{"mail":"email"},"certificate":` + string(cert) + `}`)
		require.NoError(t, d.Validate())
	})

	t.Run("saml attribute_mapping with unknown target VALUE is rejected", func(t *testing.T) {
		// SAML maps samlAttr->target, so the VALUE must be a known target.
		d := validIDPCreate()
		d.Provider = shared.IDPProviderSAML
		d.ProviderType = shared.IDPTypeSAML
		d.Issuer = ""
		d.ProviderClientID = ""
		d.ProviderClientSecret = ""
		d.Config = datatypes.JSON(`{"entity_id":"urn:idp","sso_url":"https://idp.example.com/sso","certificate":"x","attribute_mapping":{"mail":"not_a_target"}}`)
		require.Error(t, d.Validate())
	})

	t.Run("saml active missing entity_id is rejected", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderSAML
		d.ProviderType = shared.IDPTypeSAML
		d.Issuer = ""
		d.ProviderClientID = ""
		d.ProviderClientSecret = ""
		d.Config = datatypes.JSON(`{"sso_url":"https://idp.example.com/sso","certificate":"x"}`)
		require.Error(t, d.Validate())
	})
}

// ---------------------------------------------------------------------------
// F4: external endpoint URLs must be https (http allowed only for localhost).
// ---------------------------------------------------------------------------

func TestRequireHTTPSURL(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty is ok (required-ness handled elsewhere)", "", false},
		{"https accepted", "https://idp.example.com/authorize", false},
		{"http rejected", "http://idp.example.com/authorize", true},
		{"http localhost accepted", "http://localhost:8080/authorize", false},
		{"http 127.0.0.1 accepted", "http://127.0.0.1:8080/authorize", false},
		{"http localhost no port accepted", "http://localhost/token", false},
		{"non-url rejected", "not-a-url", true},
		{"ftp scheme rejected", "ftp://idp.example.com", true},
		{"http subdomain of localhost rejected", "http://localhost.evil.com/cb", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := requireHTTPSURL(tc.value)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Per-provider host binding: a well-known provider's issuer/endpoints must live
// on its official domain(s), so an operator can't point a `google`/`github`/etc.
// provider at an attacker host (which would leak the client secret or federate
// against a fake IdP). Variable-domain providers (auth0/gitlab/maintainerd/saml)
// are intentionally unrestricted.
func TestIdentityProviderValidation_ProviderHostBinding(t *testing.T) {
	githubDTO := func() IdentityProviderCreateRequestDTO {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderGitHub
		d.ProviderType = shared.IDPTypeSocial
		d.Issuer = "" // OAuth2-only: no issuer
		d.ProviderClientID = "cid"
		d.ProviderClientSecret = "sec"
		d.Config = datatypes.JSON(`{"authorization_endpoint":"https://github.com/login/oauth/authorize","token_endpoint":"https://github.com/login/oauth/access_token","userinfo_endpoint":"https://api.github.com/user"}`)
		return d
	}

	t.Run("google issuer must be accounts.google.com", func(t *testing.T) {
		d := validIDPCreate()
		d.Issuer = "https://evil.com"
		require.Error(t, d.Validate())
	})

	t.Run("google official issuer accepted", func(t *testing.T) {
		d := validIDPCreate()
		d.Issuer = "https://accounts.google.com"
		require.NoError(t, d.Validate())
	})

	t.Run("github official endpoints accepted", func(t *testing.T) {
		require.NoError(t, githubDTO().Validate())
	})

	t.Run("github token_endpoint on foreign host rejected", func(t *testing.T) {
		d := githubDTO()
		d.Config = datatypes.JSON(`{"authorization_endpoint":"https://github.com/login/oauth/authorize","token_endpoint":"https://evil.com/token","userinfo_endpoint":"https://api.github.com/user"}`)
		require.Error(t, d.Validate())
	})

	t.Run("github endpoint with embedded credentials rejected", func(t *testing.T) {
		d := githubDTO()
		d.Config = datatypes.JSON(`{"authorization_endpoint":"https://github.com/login/oauth/authorize","token_endpoint":"https://api.github.com@evil.com/token","userinfo_endpoint":"https://api.github.com/user"}`)
		require.Error(t, d.Validate())
	})

	t.Run("github lookalike host rejected", func(t *testing.T) {
		d := githubDTO()
		d.Config = datatypes.JSON(`{"authorization_endpoint":"https://github.com.evil.com/authorize","token_endpoint":"https://github.com/login/oauth/access_token","userinfo_endpoint":"https://api.github.com/user"}`)
		require.Error(t, d.Validate())
	})

	t.Run("cognito regional issuer accepted", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderCognito
		d.ProviderType = shared.IDPTypeEnterprise
		d.Issuer = "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_AbC123"
		require.NoError(t, d.Validate())
	})

	t.Run("cognito bogus issuer rejected", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderCognito
		d.ProviderType = shared.IDPTypeEnterprise
		d.Issuer = "https://evil.com/us-east-1_AbC123"
		require.Error(t, d.Validate())
	})

	t.Run("variable-domain gitlab issuer any host accepted", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderGitLab
		d.ProviderType = shared.IDPTypeEnterprise
		d.Issuer = "https://gitlab.mycorp.example"
		require.NoError(t, d.Validate())
	})
}

func TestIdentityProviderValidate_HTTPSRule(t *testing.T) {
	t.Run("http issuer rejected", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeSocial
		d.Issuer = "http://accounts.google.com"
		require.Error(t, d.Validate())
	})

	t.Run("https issuer accepted", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeSocial
		d.Issuer = "https://accounts.google.com"
		require.NoError(t, d.Validate())
	})

	t.Run("http localhost issuer accepted", func(t *testing.T) {
		// gitlab: variable-domain, so only the https rule applies (localhost ok).
		d := validIDPCreate()
		d.Provider = shared.IDPProviderGitLab
		d.ProviderType = shared.IDPTypeSocial
		d.Issuer = "http://localhost:9000"
		require.NoError(t, d.Validate())
	})

	t.Run("http oauth2 endpoint rejected", func(t *testing.T) {
		// github is OAuth2-only, so endpoints are legitimately present and the https
		// rule applies to them (http on a non-localhost host is rejected).
		d := validIDPCreate()
		d.Provider = shared.IDPProviderGitHub
		d.ProviderType = shared.IDPTypeSocial
		d.Issuer = ""
		d.ProviderClientID = "abc"
		d.ProviderClientSecret = "sec"
		d.Config = datatypes.JSON(`{"authorization_endpoint":"http://github.com/login/oauth/authorize","token_endpoint":"https://github.com/login/oauth/access_token","userinfo_endpoint":"https://api.github.com/user"}`)
		require.Error(t, d.Validate())
	})

	t.Run("http saml sso_url rejected", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = shared.IDPProviderSAML
		d.ProviderType = shared.IDPTypeSAML
		d.Issuer = ""
		d.ProviderClientID = ""
		d.ProviderClientSecret = ""
		d.Config = datatypes.JSON(`{"entity_id":"urn:idp","sso_url":"http://idp.example.com/sso","certificate":"x"}`)
		require.Error(t, d.Validate())
	})

	t.Run("https saml sso_url accepted (structurally)", func(t *testing.T) {
		// Uses an inactive provider so the certificate/entity requirements do not
		// fire; this isolates the sso_url https rule.
		d := validIDPCreate()
		d.Provider = shared.IDPProviderSAML
		d.ProviderType = shared.IDPTypeSAML
		d.Status = shared.StatusInactive
		d.Issuer = ""
		d.ProviderClientID = ""
		d.ProviderClientSecret = ""
		d.Config = datatypes.JSON(`{"sso_url":"https://idp.example.com/sso"}`)
		require.NoError(t, d.Validate())
	})
}
