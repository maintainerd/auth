package oauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
