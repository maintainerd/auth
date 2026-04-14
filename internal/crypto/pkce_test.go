package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePKCEChallenge(t *testing.T) {
	// Generate a valid verifier and challenge pair.
	verifier := strings.Repeat("abcdefghijklmnop", 3) // 48 chars, valid charset
	challenge := ComputeS256Challenge(verifier)

	cases := []struct {
		name        string
		verifier    string
		challenge   string
		method      string
		expectErr   bool
		errContains string
	}{
		{
			name:      "valid S256",
			verifier:  verifier,
			challenge: challenge,
			method:    "S256",
			expectErr: false,
		},
		{
			name:        "unsupported method plain",
			verifier:    verifier,
			challenge:   challenge,
			method:      "plain",
			expectErr:   true,
			errContains: "unsupported code_challenge_method",
		},
		{
			name:        "unsupported method empty",
			verifier:    verifier,
			challenge:   challenge,
			method:      "",
			expectErr:   true,
			errContains: "unsupported code_challenge_method",
		},
		{
			name:        "verifier too short",
			verifier:    strings.Repeat("a", PKCEMinVerifierLength-1),
			challenge:   challenge,
			method:      "S256",
			expectErr:   true,
			errContains: "code_verifier length must be between",
		},
		{
			name:        "verifier too long",
			verifier:    strings.Repeat("a", PKCEMaxVerifierLength+1),
			challenge:   challenge,
			method:      "S256",
			expectErr:   true,
			errContains: "code_verifier length must be between",
		},
		{
			name:      "verifier exact min length valid",
			verifier:  strings.Repeat("a", PKCEMinVerifierLength),
			challenge: ComputeS256Challenge(strings.Repeat("a", PKCEMinVerifierLength)),
			method:    "S256",
			expectErr: false,
		},
		{
			name:      "verifier exact max length valid",
			verifier:  strings.Repeat("b", PKCEMaxVerifierLength),
			challenge: ComputeS256Challenge(strings.Repeat("b", PKCEMaxVerifierLength)),
			method:    "S256",
			expectErr: false,
		},
		{
			name:        "verifier invalid characters",
			verifier:    strings.Repeat("a", PKCEMinVerifierLength-1) + "!",
			challenge:   challenge,
			method:      "S256",
			expectErr:   true,
			errContains: "code_verifier contains invalid characters",
		},
		{
			name:        "verifier with space",
			verifier:    strings.Repeat("a", PKCEMinVerifierLength-1) + " ",
			challenge:   challenge,
			method:      "S256",
			expectErr:   true,
			errContains: "code_verifier contains invalid characters",
		},
		{
			name:        "verifier does not match challenge",
			verifier:    strings.Repeat("c", PKCEMinVerifierLength),
			challenge:   challenge,
			method:      "S256",
			expectErr:   true,
			errContains: "code_verifier does not match code_challenge",
		},
		{
			name:      "valid verifier with special chars",
			verifier:  strings.Repeat("a-._~Z9", 7)[:PKCEMinVerifierLength],
			challenge: ComputeS256Challenge(strings.Repeat("a-._~Z9", 7)[:PKCEMinVerifierLength]),
			method:    "S256",
			expectErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePKCEChallenge(tc.verifier, tc.challenge, tc.method)
			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestComputeS256Challenge(t *testing.T) {
	// Known test vector: the S256 challenge of "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	// should produce "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	// (from RFC 7636 Appendix B)
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	expected := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	assert.Equal(t, expected, ComputeS256Challenge(verifier))
}

func TestHashAuthorizationCode(t *testing.T) {
	code := "test-authorization-code"
	hash := HashAuthorizationCode(code)
	assert.NotEmpty(t, hash)
	// Same input must produce same hash.
	assert.Equal(t, hash, HashAuthorizationCode(code))
	// Different inputs produce different hashes.
	assert.NotEqual(t, hash, HashAuthorizationCode("different-code"))
}

func TestHashRefreshToken(t *testing.T) {
	token := "test-refresh-token"
	hash := HashRefreshToken(token)
	assert.NotEmpty(t, hash)
	assert.Equal(t, hash, HashRefreshToken(token))
	assert.NotEqual(t, hash, HashRefreshToken("different-token"))
}
