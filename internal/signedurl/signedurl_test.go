package signedurl

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testHMACSecret = "test-hmac-secret-key-for-unit-tests"

func setHMACSecret(t *testing.T) {
	t.Helper()
	t.Setenv("HMAC_SECRET_KEY", testHMACSecret)
}

func TestGenerateSignedURL(t *testing.T) {
	setHMACSecret(t)

	params := map[string]string{"token": "abc123", "email": "user@example.com"}
	signed, err := GenerateSignedURL("https://example.com/verify", params, time.Hour)
	require.NoError(t, err)
	assert.Contains(t, signed, "https://example.com/verify?")
	assert.Contains(t, signed, "sig=")
	assert.Contains(t, signed, "expires=")
	assert.Contains(t, signed, "token=abc123")
	assert.Contains(t, signed, "email=user%40example.com")
}

func TestValidateSignedURL_Valid(t *testing.T) {
	setHMACSecret(t)

	params := map[string]string{"invite_token": "xyz789"}
	signed, err := GenerateSignedURL("https://api.example.com/invite", params, time.Hour)
	require.NoError(t, err)

	parsed, err := url.Parse(signed)
	require.NoError(t, err)

	result, err := ValidateSignedURL(parsed.Query())
	require.NoError(t, err)
	assert.Equal(t, "xyz789", result["invite_token"])
}

func TestValidateSignedURL_Expired(t *testing.T) {
	setHMACSecret(t)

	params := map[string]string{"token": "old"}
	signed, err := GenerateSignedURL("https://example.com/reset", params, -time.Minute)
	require.NoError(t, err)

	parsed, err := url.Parse(signed)
	require.NoError(t, err)

	_, err = ValidateSignedURL(parsed.Query())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestValidateSignedURL_TamperedSignature(t *testing.T) {
	setHMACSecret(t)

	params := map[string]string{"token": "real"}
	signed, err := GenerateSignedURL("https://example.com/verify", params, time.Hour)
	require.NoError(t, err)

	// Replace sig with garbage
	tampered := strings.Replace(signed, "sig=", "sig=TAMPERED", 1)
	parsed, err := url.Parse(tampered)
	require.NoError(t, err)

	_, err = ValidateSignedURL(parsed.Query())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature")
}

func TestValidateSignedURL_TamperedParam(t *testing.T) {
	setHMACSecret(t)

	params := map[string]string{"email": "original@example.com"}
	signed, err := GenerateSignedURL("https://example.com/verify", params, time.Hour)
	require.NoError(t, err)

	tampered := strings.Replace(signed, "original%40example.com", "attacker%40evil.com", 1)
	parsed, err := url.Parse(tampered)
	require.NoError(t, err)

	_, err = ValidateSignedURL(parsed.Query())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature")
}

func TestValidateSignedURL_MissingParams(t *testing.T) {
	setHMACSecret(t)

	_, err := ValidateSignedURL(url.Values{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required parameters")
}

func TestConvertToFrontendURL(t *testing.T) {
	setHMACSecret(t)

	apiURL := "https://api.example.com/verify?token=abc&expires=9999999999&sig=somesig"
	frontendURL, err := ConvertToFrontendURL(apiURL, "https://app.example.com/verify")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(frontendURL, "https://app.example.com/verify?"), "frontend URL should start with the base")
	assert.Contains(t, frontendURL, "token=abc")
	assert.Contains(t, frontendURL, "sig=somesig")
}

func TestConvertToFrontendURL_InvalidAPIURL(t *testing.T) {
	_, err := ConvertToFrontendURL("://bad-url", "https://app.example.com/verify")
	require.Error(t, err)
}

func TestComputeSignature_Deterministic(t *testing.T) {
	setHMACSecret(t)

	values := url.Values{"token": {"abc"}, "expires": {"9999"}}
	sig1, err := computeSignature(values)
	require.NoError(t, err)
	sig2, err := computeSignature(values)
	require.NoError(t, err)
	assert.Equal(t, sig1, sig2)
}

func TestComputeSignature_DifferentInputDifferentSig(t *testing.T) {
	setHMACSecret(t)

	v1 := url.Values{"token": {"abc"}, "expires": {"9999"}}
	v2 := url.Values{"token": {"xyz"}, "expires": {"9999"}}
	sig1, err := computeSignature(v1)
	require.NoError(t, err)
	sig2, err := computeSignature(v2)
	require.NoError(t, err)
	assert.NotEqual(t, sig1, sig2)
}

func TestComputeSignature_IgnoresSigKey(t *testing.T) {
	setHMACSecret(t)

	withSig := url.Values{"token": {"abc"}, "expires": {"9999"}, "sig": {"noise"}}
	withoutSig := url.Values{"token": {"abc"}, "expires": {"9999"}}
	sig1, err := computeSignature(withSig)
	require.NoError(t, err)
	sig2, err := computeSignature(withoutSig)
	require.NoError(t, err)
	assert.Equal(t, sig1, sig2)
}

// ---------------------------------------------------------------------------
// hmacSecretKey — error path
// ---------------------------------------------------------------------------

func TestHMACSecretKey_ErrorWhenMissing(t *testing.T) {
	t.Setenv("HMAC_SECRET_KEY", "")
	_, err := hmacSecretKey()
	assert.Error(t, err)
}

func TestComputeSignature_ErrorWhenNoHMAC(t *testing.T) {
	t.Setenv("HMAC_SECRET_KEY", "")
	_, err := computeSignature(url.Values{"key": {"val"}})
	assert.Error(t, err)
}

func TestGenerateSignedURL_ErrorWhenNoHMAC(t *testing.T) {
	t.Setenv("HMAC_SECRET_KEY", "")
	_, err := GenerateSignedURL("https://example.com", map[string]string{"k": "v"}, time.Hour)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compute signature")
}

func TestValidateSignedURL_ErrorWhenNoHMAC(t *testing.T) {
	// First generate a valid URL, then clear the key before validating.
	setHMACSecret(t)
	signed, err := GenerateSignedURL("https://example.com/reset", map[string]string{"token": "abc"}, time.Hour)
	require.NoError(t, err)

	parsed, err := url.Parse(signed)
	require.NoError(t, err)

	t.Setenv("HMAC_SECRET_KEY", "")
	_, err = ValidateSignedURL(parsed.Query())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compute signature")
}

// ---------------------------------------------------------------------------
// ValidateSignedURL — invalid expires param
// ---------------------------------------------------------------------------

func TestValidateSignedURL_InvalidExpires(t *testing.T) {
	setHMACSecret(t)

	values := url.Values{
		"token":   {"abc"},
		"expires": {"not-a-number"},
		"sig":     {"fakesig"},
	}
	_, err := ValidateSignedURL(values)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid expires param")
}
