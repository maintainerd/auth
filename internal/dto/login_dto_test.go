package dto

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/signedurl"
)

func TestLoginRequestDto_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dto     LoginRequestDto
		wantErr bool
	}{
		{
			name:    "valid credentials",
			dto:     LoginRequestDto{Username: "admin", Password: "secret"},
			wantErr: false,
		},
		{
			name:    "missing username",
			dto:     LoginRequestDto{Username: "", Password: "secret"},
			wantErr: true,
		},
		{
			name:    "missing password",
			dto:     LoginRequestDto{Username: "admin", Password: ""},
			wantErr: true,
		},
		{
			name:    "both fields missing",
			dto:     LoginRequestDto{},
			wantErr: true,
		},
		{
			name:    "username too long",
			dto:     LoginRequestDto{Username: string(make([]byte, 256)), Password: "secret"},
			wantErr: true,
		},
		{
			name:    "password too long",
			dto:     LoginRequestDto{Username: "admin", Password: string(make([]byte, 129))},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := tc.dto
			err := d.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoginRequestDto_Sanitize(t *testing.T) {
	d := LoginRequestDto{
		Username: "  admin  ",
		Password: "secret",
	}
	err := d.Validate()
	require.NoError(t, err)
	// SanitizeInput strips HTML/script tags — plain text should remain usable
	assert.NotEmpty(t, d.Username)
}

func TestLoginQueryDto_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dto     LoginQueryDto
		wantErr bool
	}{
		{
			name:    "valid query",
			dto:     LoginQueryDto{ClientID: "client-1", ProviderID: "provider-1"},
			wantErr: false,
		},
		{
			name:    "missing client id",
			dto:     LoginQueryDto{ClientID: "", ProviderID: "provider-1"},
			wantErr: true,
		},
		{
			name:    "missing provider id",
			dto:     LoginQueryDto{ClientID: "client-1", ProviderID: ""},
			wantErr: true,
		},
		{
			name:    "both missing",
			dto:     LoginQueryDto{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := tc.dto
			err := d.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoginQueryDto_ValidateSignedURL(t *testing.T) {
	q := &LoginQueryDto{ClientID: "c1", ProviderID: "p1"}

	t.Run("missing signed url params returns error", func(t *testing.T) {
		values := url.Values{}
		err := q.ValidateSignedURL(values)
		require.Error(t, err)
	})

	t.Run("valid signed url returns nil", func(t *testing.T) {
		t.Setenv("HMAC_SECRET_KEY", "test-secret-key")
		raw, _ := signedurl.GenerateSignedURL("https://example.com", map[string]string{"client_id": "c1"}, time.Minute)
		parsed, _ := url.Parse(raw)
		err := q.ValidateSignedURL(parsed.Query())
		assert.NoError(t, err)
	})
}

func TestLoginResponseDto_Fields(t *testing.T) {
	resp := LoginResponseDto{
		AccessToken:  "access",
		IDToken:      "id",
		RefreshToken: "refresh",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		IssuedAt:     1700000000,
	}
	assert.Equal(t, "access", resp.AccessToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Equal(t, int64(3600), resp.ExpiresIn)
}
