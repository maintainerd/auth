package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authctx"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOAuthUserInfoHandler(t *testing.T) {
	h := NewOAuthUserInfoHandler()
	assert.NotNil(t, h)
}

// ---------------------------------------------------------------------------
// UserInfo
// ---------------------------------------------------------------------------

func TestOAuthUserInfoHandler_UserInfo_NoUser(t *testing.T) {
	h := NewOAuthUserInfoHandler()
	r := httptest.NewRequest(http.MethodGet, "/oauth/userinfo", nil)
	// Inject empty auth context (no user).
	r = middleware.WithAuthContext(r, &authctx.AuthContext{})
	w := httptest.NewRecorder()

	h.UserInfo(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "invalid_token", body["error"])
}

func TestOAuthUserInfoHandler_UserInfo_NilAuthContext(t *testing.T) {
	h := NewOAuthUserInfoHandler()
	r := httptest.NewRequest(http.MethodGet, "/oauth/userinfo", nil)
	// No auth context at all — middleware.AuthFromRequest returns zero-value.
	w := httptest.NewRecorder()

	h.UserInfo(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOAuthUserInfoHandler_UserInfo_Success(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	userUUID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	h := NewOAuthUserInfoHandler()
	r := httptest.NewRequest(http.MethodGet, "/oauth/userinfo", nil)
	r = middleware.WithJWTClaims(r, &middleware.JWTClaims{
		Sub:   "identity-sub-123",
		Scope: "openid email profile phone",
	})
	r = middleware.WithAuthContext(r, &authctx.AuthContext{
		User: &User{
			UserUUID:        userUUID,
			Email:           "user@example.com",
			IsEmailVerified: true,
			Phone:           "+1234567890",
			IsPhoneVerified: false,
			Fullname:        "Jane Doe",
			UpdatedAt:       now,
		},
	})
	w := httptest.NewRecorder()

	h.UserInfo(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))

	var resp OAuthUserInfoResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	assert.Equal(t, "identity-sub-123", resp.Sub)
	assert.Equal(t, "user@example.com", resp.Email)
	assert.True(t, resp.EmailVerified)
	assert.Equal(t, "+1234567890", resp.Phone)
	assert.False(t, resp.PhoneVerified)
	assert.Equal(t, "Jane Doe", resp.Name)
	assert.Equal(t, now.Unix(), resp.UpdatedAt)
	assert.Empty(t, resp.Picture)
}

func TestOAuthUserInfoHandler_UserInfo_OpenIDOnly(t *testing.T) {
	userUUID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	h := NewOAuthUserInfoHandler()
	r := httptest.NewRequest(http.MethodGet, "/oauth/userinfo", nil)
	r = middleware.WithJWTClaims(r, &middleware.JWTClaims{
		Sub:   "identity-sub-123",
		Scope: "openid",
	})
	r = middleware.WithAuthContext(r, &authctx.AuthContext{
		User: &User{
			UserUUID:        userUUID,
			Email:           "user@example.com",
			IsEmailVerified: true,
			Phone:           "+1234567890",
			Fullname:        "Jane Doe",
			UpdatedAt:       time.Now(),
		},
	})
	w := httptest.NewRecorder()

	h.UserInfo(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp OAuthUserInfoResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "identity-sub-123", resp.Sub)
	assert.Empty(t, resp.Email)
	assert.Empty(t, resp.Phone)
	assert.Empty(t, resp.Name)
	assert.Zero(t, resp.UpdatedAt)
}

func TestOAuthUserInfoHandler_UserInfo_NilProfileURL(t *testing.T) {
	h := NewOAuthUserInfoHandler()
	r := httptest.NewRequest(http.MethodGet, "/oauth/userinfo", nil)
	r = middleware.WithJWTClaims(r, &middleware.JWTClaims{Scope: "profile"})
	r = middleware.WithAuthContext(r, &authctx.AuthContext{
		User: &User{
			UserUUID:  testUserUUID,
			Fullname:  "No Pic",
			UpdatedAt: time.Now(),
			Profile: &Profile{
				ProfileURL: nil,
			},
		},
	})
	w := httptest.NewRecorder()

	h.UserInfo(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp OAuthUserInfoResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Empty(t, resp.Picture)
}

func TestOAuthUserInfoHandler_UserInfo_NoProfile(t *testing.T) {
	h := NewOAuthUserInfoHandler()
	r := httptest.NewRequest(http.MethodGet, "/oauth/userinfo", nil)
	r = middleware.WithJWTClaims(r, &middleware.JWTClaims{Scope: "profile"})
	r = middleware.WithAuthContext(r, &authctx.AuthContext{
		User: &User{
			UserUUID:  testUserUUID,
			Fullname:  "No Profile",
			UpdatedAt: time.Now(),
			Profile:   nil,
		},
	})
	w := httptest.NewRecorder()

	h.UserInfo(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp OAuthUserInfoResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Empty(t, resp.Picture)
}

func TestComposeUserDisplayName(t *testing.T) {
	t.Run("nil user", func(t *testing.T) {
		assert.Empty(t, composeUserDisplayName(nil))
	})

	t.Run("profile display name takes priority", func(t *testing.T) {
		dn := "Display Name"
		u := &User{Fullname: "Legacy", Profile: &Profile{DisplayName: &dn}}
		assert.Equal(t, "Display Name", composeUserDisplayName(u))
	})

	t.Run("display name whitespace only falls through", func(t *testing.T) {
		dn := "   "
		u := &User{Fullname: "Legacy", Profile: &Profile{DisplayName: &dn}}
		assert.Equal(t, "Legacy", composeUserDisplayName(u))
	})

	t.Run("first name + last name fallback", func(t *testing.T) {
		ln := "Doe"
		u := &User{Fullname: "Legacy", Profile: &Profile{FirstName: "John", LastName: &ln}}
		assert.Equal(t, "John Doe", composeUserDisplayName(u))
	})

	t.Run("first name only", func(t *testing.T) {
		u := &User{Fullname: "Legacy", Profile: &Profile{FirstName: "John"}}
		assert.Equal(t, "John", composeUserDisplayName(u))
	})

	t.Run("display name empty string falls through", func(t *testing.T) {
		dn := ""
		ln := "Doe"
		u := &User{Fullname: "Legacy", Profile: &Profile{DisplayName: &dn, FirstName: "John", LastName: &ln}}
		assert.Equal(t, "John Doe", composeUserDisplayName(u))
	})

	t.Run("no profile falls back to fullname", func(t *testing.T) {
		u := &User{Fullname: "Legacy User"}
		assert.Equal(t, "Legacy User", composeUserDisplayName(u))
	})
}
