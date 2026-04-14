package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewOAuthUserInfoHandler
// ---------------------------------------------------------------------------

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
	r = middleware.WithAuthContext(r, &middleware.AuthContext{})
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
	r = middleware.WithAuthContext(r, &middleware.AuthContext{
		User: &model.User{
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

	var resp dto.OAuthUserInfoResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	assert.Equal(t, userUUID.String(), resp.Sub)
	assert.Equal(t, "user@example.com", resp.Email)
	assert.True(t, resp.EmailVerified)
	assert.Equal(t, "+1234567890", resp.Phone)
	assert.False(t, resp.PhoneVerified)
	assert.Equal(t, "Jane Doe", resp.Name)
	assert.Equal(t, now.Unix(), resp.UpdatedAt)
	assert.Empty(t, resp.Picture)
}

func TestOAuthUserInfoHandler_UserInfo_WithProfilePicture(t *testing.T) {
	profileURL := "https://cdn.example.com/avatar.jpg"

	h := NewOAuthUserInfoHandler()
	r := httptest.NewRequest(http.MethodGet, "/oauth/userinfo", nil)
	r = middleware.WithAuthContext(r, &middleware.AuthContext{
		User: &model.User{
			UserUUID:        testUserUUID,
			Email:           "pic@example.com",
			IsEmailVerified: false,
			Fullname:        "Pic User",
			UpdatedAt:       time.Now(),
			Profile: &model.Profile{
				ProfileURL: &profileURL,
			},
		},
	})
	w := httptest.NewRecorder()

	h.UserInfo(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.OAuthUserInfoResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "https://cdn.example.com/avatar.jpg", resp.Picture)
}

func TestOAuthUserInfoHandler_UserInfo_NilProfileURL(t *testing.T) {
	h := NewOAuthUserInfoHandler()
	r := httptest.NewRequest(http.MethodGet, "/oauth/userinfo", nil)
	r = middleware.WithAuthContext(r, &middleware.AuthContext{
		User: &model.User{
			UserUUID:  testUserUUID,
			Fullname:  "No Pic",
			UpdatedAt: time.Now(),
			Profile: &model.Profile{
				ProfileURL: nil,
			},
		},
	})
	w := httptest.NewRecorder()

	h.UserInfo(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.OAuthUserInfoResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Empty(t, resp.Picture)
}

func TestOAuthUserInfoHandler_UserInfo_NoProfile(t *testing.T) {
	h := NewOAuthUserInfoHandler()
	r := httptest.NewRequest(http.MethodGet, "/oauth/userinfo", nil)
	r = middleware.WithAuthContext(r, &middleware.AuthContext{
		User: &model.User{
			UserUUID:  testUserUUID,
			Fullname:  "No Profile",
			UpdatedAt: time.Now(),
			Profile:   nil,
		},
	})
	w := httptest.NewRecorder()

	h.UserInfo(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.OAuthUserInfoResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Empty(t, resp.Picture)
}
