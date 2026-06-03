package oauth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthDomainModels(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		model     any
		before    func() (uuid.UUID, error)
	}{
		{
			name: "authorization code", tableName: "oauth_authorization_codes", model: OAuthAuthorizationCode{},
			before: func() (uuid.UUID, error) {
				m := &OAuthAuthorizationCode{}
				err := m.BeforeCreate(nil)
				return m.OAuthAuthorizationCodeUUID, err
			},
		},
		{
			name: "ciba request", tableName: "oauth_ciba_requests", model: OAuthCIBARequest{},
			before: func() (uuid.UUID, error) {
				m := &OAuthCIBARequest{}
				err := m.BeforeCreate(nil)
				return m.OAuthCIBARequestUUID, err
			},
		},
		{
			name: "consent challenge", tableName: "oauth_consent_challenges", model: OAuthConsentChallenge{},
			before: func() (uuid.UUID, error) {
				m := &OAuthConsentChallenge{}
				err := m.BeforeCreate(nil)
				return m.OAuthConsentChallengeUUID, err
			},
		},
		{
			name: "consent grant", tableName: "oauth_consent_grants", model: OAuthConsentGrant{},
			before: func() (uuid.UUID, error) {
				m := &OAuthConsentGrant{}
				err := m.BeforeCreate(nil)
				return m.OAuthConsentGrantUUID, err
			},
		},
		{
			name: "device code", tableName: "oauth_device_codes", model: OAuthDeviceCode{},
			before: func() (uuid.UUID, error) {
				m := &OAuthDeviceCode{}
				err := m.BeforeCreate(nil)
				return m.OAuthDeviceCodeUUID, err
			},
		},
		{
			name: "par request", tableName: "oauth_par_requests", model: OAuthPARRequest{},
			before: func() (uuid.UUID, error) {
				m := &OAuthPARRequest{}
				err := m.BeforeCreate(nil)
				return m.OAuthPARRequestUUID, err
			},
		},
		{
			name: "refresh token", tableName: "oauth_refresh_tokens", model: OAuthRefreshToken{},
			before: func() (uuid.UUID, error) {
				m := &OAuthRefreshToken{}
				err := m.BeforeCreate(nil)
				return m.OAuthRefreshTokenUUID, err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch model := tt.model.(type) {
			case OAuthAuthorizationCode:
				assert.Equal(t, tt.tableName, model.TableName())
			case OAuthCIBARequest:
				assert.Equal(t, tt.tableName, model.TableName())
			case OAuthConsentChallenge:
				assert.Equal(t, tt.tableName, model.TableName())
			case OAuthConsentGrant:
				assert.Equal(t, tt.tableName, model.TableName())
			case OAuthDeviceCode:
				assert.Equal(t, tt.tableName, model.TableName())
			case OAuthPARRequest:
				assert.Equal(t, tt.tableName, model.TableName())
			case OAuthRefreshToken:
				assert.Equal(t, tt.tableName, model.TableName())
			}
			id, err := tt.before()
			require.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, id)
		})
	}
}

func TestOAuthDomainModelsBeforeCreateKeepsExistingUUID(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000abc")

	authCode := &OAuthAuthorizationCode{OAuthAuthorizationCodeUUID: id}
	require.NoError(t, authCode.BeforeCreate(nil))
	assert.Equal(t, id, authCode.OAuthAuthorizationCodeUUID)

	ciba := &OAuthCIBARequest{OAuthCIBARequestUUID: id}
	require.NoError(t, ciba.BeforeCreate(nil))
	assert.Equal(t, id, ciba.OAuthCIBARequestUUID)

	challenge := &OAuthConsentChallenge{OAuthConsentChallengeUUID: id}
	require.NoError(t, challenge.BeforeCreate(nil))
	assert.Equal(t, id, challenge.OAuthConsentChallengeUUID)

	grant := &OAuthConsentGrant{OAuthConsentGrantUUID: id}
	require.NoError(t, grant.BeforeCreate(nil))
	assert.Equal(t, id, grant.OAuthConsentGrantUUID)

	device := &OAuthDeviceCode{OAuthDeviceCodeUUID: id}
	require.NoError(t, device.BeforeCreate(nil))
	assert.Equal(t, id, device.OAuthDeviceCodeUUID)

	par := &OAuthPARRequest{OAuthPARRequestUUID: id}
	require.NoError(t, par.BeforeCreate(nil))
	assert.Equal(t, id, par.OAuthPARRequestUUID)

	refresh := &OAuthRefreshToken{OAuthRefreshTokenUUID: id}
	require.NoError(t, refresh.BeforeCreate(nil))
	assert.Equal(t, id, refresh.OAuthRefreshTokenUUID)
}

func TestOAuthProjectionTableNames(t *testing.T) {
	assert.Equal(t, "tenants", Tenant{}.TableName())
	assert.Equal(t, "identity_providers", IdentityProvider{}.TableName())
	assert.Equal(t, "clients", Client{}.TableName())
	assert.Equal(t, "client_uris", ClientURI{}.TableName())
	assert.Equal(t, "user_identities", UserIdentity{}.TableName())
}

func TestOAuthModelExpiryAndActiveStates(t *testing.T) {
	past := time.Now().Add(-time.Second)
	future := time.Now().Add(time.Second)

	assert.True(t, (&OAuthAuthorizationCode{ExpiresAt: past}).IsExpired())
	assert.False(t, (&OAuthAuthorizationCode{ExpiresAt: future}).IsExpired())
	assert.True(t, (&OAuthCIBARequest{ExpiresAt: past}).IsExpired())
	assert.False(t, (&OAuthDeviceCode{ExpiresAt: future}).IsExpired())
	assert.True(t, (&OAuthPARRequest{ExpiresAt: past}).IsExpired())

	assert.True(t, (&OAuthRefreshToken{ExpiresAt: future}).IsActive())
	assert.False(t, (&OAuthRefreshToken{ExpiresAt: past}).IsActive())
	assert.False(t, (&OAuthRefreshToken{ExpiresAt: future, IsRevoked: true}).IsActive())
}
