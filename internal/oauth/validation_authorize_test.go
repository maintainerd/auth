package oauth

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthAuthorizeRequestDTO_Validate(t *testing.T) {
	validReq := func() OAuthAuthorizeRequestDTO {
		return OAuthAuthorizeRequestDTO{
			ResponseType:        "code",
			ClientID:            "my-client",
			RedirectURI:         "https://example.com/callback",
			Scope:               "openid profile",
			State:               "abc",
			CodeChallenge:       strings.Repeat("A", 43),
			CodeChallengeMethod: "S256",
		}
	}

	t.Run("valid request", func(t *testing.T) {
		r := validReq()
		require.NoError(t, r.Validate())
	})

	t.Run("missing response_type", func(t *testing.T) {
		r := validReq()
		r.ResponseType = ""
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "response_type")
	})

	t.Run("invalid response_type", func(t *testing.T) {
		r := validReq()
		r.ResponseType = "token"
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "response_type")
	})

	t.Run("missing client context", func(t *testing.T) {
		r := validReq()
		r.ClientID = ""
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "client_id")
	})

	t.Run("missing redirect_uri", func(t *testing.T) {
		r := validReq()
		r.RedirectURI = ""
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "redirect_uri")
	})

	t.Run("missing code_challenge", func(t *testing.T) {
		r := validReq()
		r.CodeChallenge = ""
		err := r.Validate()
		require.NoError(t, err)
	})

	t.Run("code_challenge too short", func(t *testing.T) {
		r := validReq()
		r.CodeChallenge = strings.Repeat("A", 42)
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "code_challenge")
	})

	t.Run("code_challenge too long", func(t *testing.T) {
		r := validReq()
		r.CodeChallenge = strings.Repeat("A", 129)
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "code_challenge")
	})

	t.Run("missing code_challenge_method", func(t *testing.T) {
		r := validReq()
		r.CodeChallengeMethod = ""
		err := r.Validate()
		require.NoError(t, err)
	})

	t.Run("invalid code_challenge_method", func(t *testing.T) {
		r := validReq()
		r.CodeChallengeMethod = "plain"
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "code_challenge_method")
	})

	t.Run("state too long", func(t *testing.T) {
		r := validReq()
		r.State = strings.Repeat("x", 513)
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "state")
	})

	t.Run("scope too long", func(t *testing.T) {
		r := validReq()
		r.Scope = strings.Repeat("x", 1025)
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "scope")
	})

	t.Run("nonce too long", func(t *testing.T) {
		r := validReq()
		r.Nonce = strings.Repeat("x", 513)
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nonce")
	})

	t.Run("optional fields empty is valid", func(t *testing.T) {
		r := validReq()
		r.Scope = ""
		r.State = ""
		r.Nonce = ""
		require.NoError(t, r.Validate())
	})
}

func TestOAuthConsentDecisionDTO_Validate(t *testing.T) {
	validID := uuid.New().String()

	t.Run("valid", func(t *testing.T) {
		r := OAuthConsentDecisionDTO{ChallengeID: validID, Approved: true}
		require.NoError(t, r.Validate())
	})

	t.Run("missing challenge_id", func(t *testing.T) {
		r := OAuthConsentDecisionDTO{ChallengeID: "", Approved: true}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "challenge_id")
	})

	t.Run("invalid UUID", func(t *testing.T) {
		r := OAuthConsentDecisionDTO{ChallengeID: "not-a-uuid", Approved: true}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "challenge_id")
	})

	t.Run("approved false is valid", func(t *testing.T) {
		r := OAuthConsentDecisionDTO{ChallengeID: validID, Approved: false}
		require.NoError(t, r.Validate())
	})
}

func TestOAuthAuthorizeRequestDTO_Validate_MaxLengths(t *testing.T) {
	validReq := OAuthAuthorizeRequestDTO{
		ResponseType:        "code",
		ClientID:            "my-client",
		RedirectURI:         "https://example.com/callback",
		Scope:               "openid",
		CodeChallenge:       strings.Repeat("A", 43),
		CodeChallengeMethod: "S256",
	}

	t.Run("client_id too long", func(t *testing.T) {
		r := validReq
		r.ClientID = strings.Repeat("x", 300)
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "client_id")
	})

	t.Run("redirect_uri too long", func(t *testing.T) {
		r := validReq
		r.RedirectURI = "https://x.com/" + strings.Repeat("y", 3000)
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "redirect_uri")
	})
}
