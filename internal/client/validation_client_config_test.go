package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func validJWKS() string {
	return `{"keys":[{"kty":"RSA","kid":"a","n":"0vx7","e":"AQAB","use":"sig","alg":"RS256"}]}`
}

func TestValidateClientConfig_AcceptsOrdinaryConfig(t *testing.T) {
	for _, config := range []datatypes.JSON{
		nil,
		datatypes.JSON(``),
		datatypes.JSON(`{}`),
		datatypes.JSON(`{"grant_types":["authorization_code"],"custom":{"team":"platform"}}`),
		datatypes.JSON(`{"jwks":` + validJWKS() + `}`),
		datatypes.JSON(`{"jwks_uri":"https://app.example.com/.well-known/jwks.json"}`),
	} {
		assert.NoError(t, validateClientConfig(config), string(config))
	}
}

// A malformed blob used to be stored verbatim, which silently discarded every
// setting the runtime reads from a column.
func TestValidateClientConfig_RejectsNonObject(t *testing.T) {
	for _, config := range []datatypes.JSON{
		datatypes.JSON(`{"grant_types":`),
		datatypes.JSON(`["authorization_code"]`),
		datatypes.JSON(`"a string"`),
	} {
		err := validateClientConfig(config)
		require.Error(t, err, string(config))
		assert.Contains(t, err.Error(), "must be a JSON object")
	}
}

// RFC 7591 §2 — with both set, which key source actually verifies an assertion
// depends on lookup order rather than on what the operator intended.
func TestValidateClientConfig_RejectsBothKeySources(t *testing.T) {
	err := validateClientConfig(datatypes.JSON(
		`{"jwks":` + validJWKS() + `,"jwks_uri":"https://app.example.com/jwks.json"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not both be set")
}

func TestValidateClientConfig_JWKSShape(t *testing.T) {
	cases := map[string]string{
		"JWK Set object":         `{"jwks":"not-an-object"}`,
		"non-empty \"keys\"":     `{"jwks":{"keys":[]}}`,
		"must be a JWK object":   `{"jwks":{"keys":["nope"]}}`,
		"must declare a \"kty\"": `{"jwks":{"keys":[{"kid":"a"}]}}`,
	}
	for want, config := range cases {
		err := validateClientConfig(datatypes.JSON(config))
		require.Error(t, err, config)
		assert.Contains(t, err.Error(), want, config)
	}
}

// A JWKS is a PUBLIC key set. A private component means the operator pasted the
// client's signing key into the authorization server — storing it would be a
// credential leak, and it is never needed to verify an assertion.
func TestValidateClientConfig_RejectsPrivateKeyComponents(t *testing.T) {
	for _, component := range []string{"d", "p", "q", "dp", "dq", "qi", "k"} {
		config := `{"jwks":{"keys":[{"kty":"RSA","n":"0vx7","e":"AQAB","` + component + `":"secret"}]}}`
		err := validateClientConfig(datatypes.JSON(config))
		require.Error(t, err, component)
		assert.Contains(t, err.Error(), "private key component")
	}
}

// The keys served from this URL decide whether a client assertion is accepted, so
// an unauthenticated or tamperable fetch would let an attacker substitute keys.
func TestValidateClientConfig_JWKSURIRules(t *testing.T) {
	cases := map[string]string{
		"must use https":              `{"jwks_uri":"http://app.example.com/jwks.json"}`,
		"must be an absolute URL":     `{"jwks_uri":"/jwks.json"}`,
		"must not contain a fragment": `{"jwks_uri":"https://app.example.com/jwks.json#k"}`,
	}
	for want, config := range cases {
		err := validateClientConfig(datatypes.JSON(config))
		require.Error(t, err, config)
		assert.Contains(t, err.Error(), want, config)
	}
}

func TestValidateClientConfig_ThumbprintShape(t *testing.T) {
	// 43 base64url characters = a 32-byte SHA-256 digest (RFC 8705 §3.1).
	valid := `{"mtls_bound_cert_thumbprint":"` + "A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8S9t0U1v" + `"}`
	assert.NoError(t, validateClientConfig(datatypes.JSON(valid)))

	for _, config := range []string{
		`{"mtls_bound_cert_thumbprint":"too-short"}`,
		`{"mtls_bound_cert_thumbprint":""}`,
		`{"mtls_bound_cert_thumbprint":12345}`,
		// Standard base64 padding/alphabet is not base64url.
		`{"mtls_bound_cert_thumbprint":"A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8S9t0U1+"}`,
	} {
		err := validateClientConfig(datatypes.JSON(config))
		require.Error(t, err, config)
		assert.Contains(t, err.Error(), "mtls_bound_cert_thumbprint")
	}
}

// A mapper landing on a reserved claim would let the client restate its own
// identity, audience or permissions inside the token it receives. The issuer
// strips these at runtime; rejecting on write tells the operator instead of
// leaving them to assume it worked.
func TestValidateClientConfig_RejectsReservedClaimMappers(t *testing.T) {
	for _, claim := range []string{"sub", "aud", "permissions", "tenant_id", "SCOPE"} {
		err := validateClientConfig(datatypes.JSON(`{"claim_mappers":{"` + claim + `":"x"}}`))
		require.Error(t, err, claim)
		assert.Contains(t, err.Error(), "reserved claim")
	}

	assert.NoError(t, validateClientConfig(datatypes.JSON(`{"claim_mappers":{"department":"metadata.dept"}}`)))
}

func TestValidateClientConfig_RejectsNonObjectMappings(t *testing.T) {
	for _, key := range []string{"claim_mappers", "scope_claim_mappings"} {
		err := validateClientConfig(datatypes.JSON(`{"` + key + `":["nope"]}`))
		require.Error(t, err, key)
		assert.Contains(t, err.Error(), key)
	}
}

// The runtime reads these from columns, not from config; without the mapping a
// private_key_jwt client could be saved and then reject every assertion.
func TestApplyConfigToClientColumns_MirrorsAdvancedSettings(t *testing.T) {
	c := &Client{}
	applyConfigToClientColumns(c, datatypes.JSON(`{
		"jwks": `+validJWKS()+`,
		"mtls_bound_cert_thumbprint": "A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8S9t0U1v",
		"scope_claim_mappings": {"profile": ["name"]},
		"claim_mappers": {"department": "metadata.dept"}
	}`))

	assert.NotNil(t, c.JWKS)
	assert.Nil(t, c.JWKSUri, "an inline JWKS and a jwks_uri must never both be stored")
	require.NotNil(t, c.MTLSBoundCertThumbprint)
	assert.Equal(t, "A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8S9t0U1v", *c.MTLSBoundCertThumbprint)
	assert.NotNil(t, c.ScopeClaimMappings)
	assert.NotNil(t, c.ClaimMappers)
}

func TestApplyConfigToClientColumns_MirrorsJWKSURI(t *testing.T) {
	c := &Client{}
	applyConfigToClientColumns(c, datatypes.JSON(`{"jwks_uri":"https://app.example.com/jwks.json"}`))
	require.NotNil(t, c.JWKSUri)
	assert.Equal(t, "https://app.example.com/jwks.json", *c.JWKSUri)
	assert.Nil(t, c.JWKS)
}

// Removing the keys in the console must actually revoke them, otherwise a
// rotated-out key keeps authenticating.
func TestApplyConfigToClientColumns_AbsentAdvancedSettingsAreCleared(t *testing.T) {
	uri := "https://app.example.com/jwks.json"
	thumb := "A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8S9t0U1v"
	c := &Client{
		JWKS:                    datatypes.JSON(validJWKS()),
		JWKSUri:                 &uri,
		MTLSBoundCertThumbprint: &thumb,
		ScopeClaimMappings:      datatypes.JSON(`{"profile":["name"]}`),
		ClaimMappers:            datatypes.JSON(`{"department":"metadata.dept"}`),
	}

	applyConfigToClientColumns(c, datatypes.JSON(`{"grant_types":["authorization_code"]}`))

	assert.Nil(t, c.JWKS)
	assert.Nil(t, c.JWKSUri)
	assert.Nil(t, c.MTLSBoundCertThumbprint)
	assert.Nil(t, c.ScopeClaimMappings)
	assert.Nil(t, c.ClaimMappers)
}

// An empty object is how the console represents "cleared"; storing `{}` would read
// as "configured but empty" to every consumer.
func TestApplyConfigToClientColumns_EmptyAdvancedObjectsClear(t *testing.T) {
	c := &Client{
		JWKS:         datatypes.JSON(validJWKS()),
		ClaimMappers: datatypes.JSON(`{"department":"metadata.dept"}`),
	}
	applyConfigToClientColumns(c, datatypes.JSON(`{"jwks":{},"claim_mappers":{}}`))
	assert.Nil(t, c.JWKS)
	assert.Nil(t, c.ClaimMappers)
}
