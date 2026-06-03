package oauth

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestTokenRemainingTTL(t *testing.T) {
	future := time.Now().Add(time.Hour).Unix()
	past := time.Now().Add(-time.Hour).Unix()

	assert.Positive(t, tokenRemainingTTL(float64(future)))
	assert.Positive(t, tokenRemainingTTL(int64(future)))
	assert.Positive(t, tokenRemainingTTL(int(future)))
	assert.Positive(t, tokenRemainingTTL(json.Number("123456789000")))
	assert.Zero(t, tokenRemainingTTL(json.Number("not-a-number")))
	assert.Zero(t, tokenRemainingTTL("not-supported"))
	assert.Negative(t, tokenRemainingTTL(float64(past)))
}

func TestBuildUserProfile(t *testing.T) {
	t.Run("uses profile fields", func(t *testing.T) {
		user := &User{
			Email:           "jane@example.com",
			IsEmailVerified: true,
			Phone:           "+15551234567",
			IsPhoneVerified: true,
			Fullname:        "Fallback Name",
			Profile: &Profile{
				FirstName:  "Jane",
				LastName:   ptr.Ptr("Doe"),
				ProfileURL: ptr.Ptr("https://example.com/jane.png"),
			},
		}

		profile := buildUserProfile(user)

		assert.Equal(t, "jane@example.com", profile.Email)
		assert.True(t, profile.EmailVerified)
		assert.Equal(t, "+15551234567", profile.Phone)
		assert.True(t, profile.PhoneVerified)
		assert.Equal(t, "Jane", profile.FirstName)
		assert.Equal(t, "Doe", profile.LastName)
		assert.Equal(t, "Jane Doe", profile.Name)
		assert.Equal(t, "https://example.com/jane.png", profile.Picture)
	})

	t.Run("falls back to fullname", func(t *testing.T) {
		profile := buildUserProfile(&User{Fullname: "Fallback Name", Profile: &Profile{}})

		assert.Equal(t, "Fallback Name", profile.Name)
	})

	t.Run("builds name from first name only", func(t *testing.T) {
		profile := buildUserProfile(&User{Profile: &Profile{FirstName: "Jane"}})

		assert.Equal(t, "Jane", profile.Name)
	})
}

func TestBuildIDTokenParams(t *testing.T) {
	t.Run("empty scope returns nil", func(t *testing.T) {
		assert.Nil(t, buildIDTokenParams("", &Client{}))
		assert.Nil(t, buildIDTokenParams("   ", &Client{}))
	})

	t.Run("includes mappings and extra claims", func(t *testing.T) {
		mappings := datatypes.JSON([]byte(`{"profile":["name","picture"]}`))
		extraClaims := datatypes.JSON([]byte(`{"tier":"gold"}`))
		params := buildIDTokenParams("openid profile", &Client{
			ScopeClaimMappings: mappings,
			ClaimMappers:       extraClaims,
		})

		require.NotNil(t, params)
		assert.Equal(t, []string{"openid", "profile"}, params.RequestedScopes)
		assert.Equal(t, []string{jwt.AMRPassword}, params.AMR)
		assert.Equal(t, jwt.ACRLevel1, params.ACR)
		assert.Equal(t, map[string][]string{"profile": {"name", "picture"}}, params.ScopeClaimMappings)
		assert.Equal(t, map[string]any{"tier": "gold"}, params.ExtraClaims)
	})

	t.Run("ignores invalid mapping JSON", func(t *testing.T) {
		invalid := datatypes.JSON([]byte(`{`))
		params := buildIDTokenParams("openid", &Client{
			ScopeClaimMappings: invalid,
			ClaimMappers:       invalid,
		})

		require.NotNil(t, params)
		assert.Nil(t, params.ScopeClaimMappings)
		assert.Nil(t, params.ExtraClaims)
	})
}

func TestParseScopes(t *testing.T) {
	assert.Nil(t, parseScopes(""))
	assert.Nil(t, parseScopes("   "))
	assert.Equal(t, []string{"openid", "email"}, parseScopes(" openid   email "))
}

func TestAMRClaimValues(t *testing.T) {
	assert.Nil(t, amrClaimValues("pwd"))
	assert.Equal(t, []string{"pwd", "mfa"}, amrClaimValues([]any{"pwd", "", 123, "mfa"}))
}
