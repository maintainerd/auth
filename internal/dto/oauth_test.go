package dto

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

	t.Run("missing client_id", func(t *testing.T) {
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
		require.Error(t, err)
		assert.Contains(t, err.Error(), "code_challenge")
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
		require.Error(t, err)
		assert.Contains(t, err.Error(), "code_challenge_method")
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

func TestOAuthTokenRequestDTO_Validate(t *testing.T) {
	t.Run("valid authorization_code", func(t *testing.T) {
		r := OAuthTokenRequestDTO{GrantType: "authorization_code"}
		require.NoError(t, r.Validate())
	})

	t.Run("valid refresh_token", func(t *testing.T) {
		r := OAuthTokenRequestDTO{GrantType: "refresh_token"}
		require.NoError(t, r.Validate())
	})

	t.Run("valid client_credentials", func(t *testing.T) {
		r := OAuthTokenRequestDTO{GrantType: "client_credentials"}
		require.NoError(t, r.Validate())
	})

	t.Run("missing grant_type", func(t *testing.T) {
		r := OAuthTokenRequestDTO{GrantType: ""}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "grant_type")
	})

	t.Run("invalid grant_type", func(t *testing.T) {
		r := OAuthTokenRequestDTO{GrantType: "implicit"}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "grant_type")
	})
}

func TestOAuthRevokeRequestDTO_Validate(t *testing.T) {
	t.Run("valid with token only", func(t *testing.T) {
		r := OAuthRevokeRequestDTO{Token: "some-token"}
		require.NoError(t, r.Validate())
	})

	t.Run("valid with hint", func(t *testing.T) {
		r := OAuthRevokeRequestDTO{Token: "t", TokenTypeHint: "refresh_token"}
		require.NoError(t, r.Validate())
	})

	t.Run("missing token", func(t *testing.T) {
		r := OAuthRevokeRequestDTO{Token: ""}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token")
	})

	t.Run("invalid hint", func(t *testing.T) {
		r := OAuthRevokeRequestDTO{Token: "t", TokenTypeHint: "bad"}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token_type_hint")
	})
}

func TestOAuthIntrospectRequestDTO_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := OAuthIntrospectRequestDTO{Token: "some-token"}
		require.NoError(t, r.Validate())
	})

	t.Run("valid with hint", func(t *testing.T) {
		r := OAuthIntrospectRequestDTO{Token: "t", TokenTypeHint: "access_token"}
		require.NoError(t, r.Validate())
	})

	t.Run("missing token", func(t *testing.T) {
		r := OAuthIntrospectRequestDTO{Token: ""}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token")
	})

	t.Run("invalid hint", func(t *testing.T) {
		r := OAuthIntrospectRequestDTO{Token: "t", TokenTypeHint: "bad"}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token_type_hint")
	})
}
