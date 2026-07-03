//go:build integration

package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
)

// PKCE end-to-end binding (RFC 7636).
//
// This exercises the exact functions the two OAuth endpoints use: `/authorize`
// stores the S256 challenge a client derives from its verifier, and `/token`
// validates the presented verifier against that stored challenge. Testing them
// together guards the cross-endpoint invariant that a token cannot be minted
// without proving possession of the original verifier — the whole point of PKCE
// for public clients. No database is required.
func TestIntegration_PKCE_ChallengeVerifierBinding(t *testing.T) {
	// A client generates a high-entropy verifier and derives the challenge it
	// sends to /authorize.
	verifier, err := crypto.GenerateRandomString(64)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(verifier), 43, "verifier must be 43-128 chars per RFC 7636")

	challenge := crypto.ComputeS256Challenge(verifier)
	require.NotEmpty(t, challenge)
	require.NotEqual(t, verifier, challenge, "challenge must be the SHA-256 of the verifier, not the verifier itself")

	t.Run("correct verifier validates against the stored challenge", func(t *testing.T) {
		assert.NoError(t, crypto.ValidatePKCEChallenge(verifier, challenge, "S256"))
	})

	t.Run("wrong verifier is rejected", func(t *testing.T) {
		other, err := crypto.GenerateRandomString(64)
		require.NoError(t, err)
		assert.Error(t, crypto.ValidatePKCEChallenge(other, challenge, "S256"),
			"a token request with a verifier that does not match the authorize-time challenge must fail")
	})

	t.Run("tampered stored challenge is rejected", func(t *testing.T) {
		tampered := challenge[:len(challenge)-1] + "A"
		assert.Error(t, crypto.ValidatePKCEChallenge(verifier, tampered, "S256"))
	})

	t.Run("plain method is not accepted where S256 is expected", func(t *testing.T) {
		// A downgrade to "plain" (verifier == challenge) must not validate against
		// an S256 challenge.
		assert.Error(t, crypto.ValidatePKCEChallenge(verifier, challenge, "plain"))
	})
}
