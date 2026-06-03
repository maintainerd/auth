package authn

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginResponseDTO_JSONContract(t *testing.T) {
	sessionID := "session-1"
	challengeToken := "challenge-1"
	dto := LoginResponseDTO{
		AccessToken:       "access",
		IDToken:           "id",
		RefreshToken:      "refresh",
		ExpiresIn:         3600,
		TokenType:         "Bearer",
		IssuedAt:          100,
		SessionID:         &sessionID,
		MFARequired:       true,
		MFAChallengeToken: &challengeToken,
		MFAAllowedMethods: []string{"totp"},
	}

	body, err := json.Marshal(dto)

	require.NoError(t, err)
	assert.Contains(t, string(body), `"access_token":"access"`)
	assert.Contains(t, string(body), `"mfa_required":true`)
	assert.Contains(t, string(body), `"session_id":"session-1"`)
}

func TestRegisterResponseDTO_JSONContract(t *testing.T) {
	dto := RegisterResponseDTO{
		AccessToken:  "access",
		IDToken:      "id",
		RefreshToken: "refresh",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		IssuedAt:     100,
	}

	body, err := json.Marshal(dto)

	require.NoError(t, err)
	assert.Contains(t, string(body), `"access_token":"access"`)
	assert.Contains(t, string(body), `"refresh_token":"refresh"`)
}
