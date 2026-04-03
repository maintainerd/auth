package util

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// authResponseStruct mimics a DTO with the fields SetAuthCookies inspects via reflection.
type authResponseStruct struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
	ExpiresIn    int64
}

// findCookie returns the first cookie with the given name from a ResponseRecorder.
func findCookie(t *testing.T, rr *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, c := range rr.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestSetAuthCookies_FromMap(t *testing.T) {
	rr := httptest.NewRecorder()
	data := map[string]interface{}{
		"access_token":  "at-value",
		"id_token":      "it-value",
		"refresh_token": "rt-value",
		"expires_in":    int64(1800),
	}
	SetAuthCookies(rr, data)

	at := findCookie(t, rr, "access_token")
	require.NotNil(t, at)
	assert.Equal(t, "at-value", at.Value)
	assert.Equal(t, 1800, at.MaxAge)
	assert.True(t, at.HttpOnly)
	assert.True(t, at.Secure)
	assert.Equal(t, "/", at.Path)

	it := findCookie(t, rr, "id_token")
	require.NotNil(t, it)
	assert.Equal(t, "it-value", it.Value)
	assert.Equal(t, 3600, it.MaxAge)

	rt := findCookie(t, rr, "refresh_token")
	require.NotNil(t, rt)
	assert.Equal(t, "rt-value", rt.Value)
	assert.Equal(t, 7*24*60*60, rt.MaxAge)
	assert.Equal(t, "/auth/refresh", rt.Path)
}

func TestSetAuthCookies_FromStruct(t *testing.T) {
	rr := httptest.NewRecorder()
	data := authResponseStruct{
		AccessToken:  "at-struct",
		IDToken:      "it-struct",
		RefreshToken: "rt-struct",
		ExpiresIn:    900,
	}
	SetAuthCookies(rr, data)

	at := findCookie(t, rr, "access_token")
	require.NotNil(t, at)
	assert.Equal(t, "at-struct", at.Value)
	assert.Equal(t, 900, at.MaxAge)

	it := findCookie(t, rr, "id_token")
	require.NotNil(t, it)
	assert.Equal(t, "it-struct", it.Value)

	rt := findCookie(t, rr, "refresh_token")
	require.NotNil(t, rt)
	assert.Equal(t, "rt-struct", rt.Value)
}

func TestSetAuthCookies_FromStructPtr(t *testing.T) {
	rr := httptest.NewRecorder()
	data := &authResponseStruct{
		AccessToken:  "at-ptr",
		IDToken:      "it-ptr",
		RefreshToken: "rt-ptr",
		ExpiresIn:    600,
	}
	SetAuthCookies(rr, data)

	at := findCookie(t, rr, "access_token")
	require.NotNil(t, at)
	assert.Equal(t, "at-ptr", at.Value)
}

func TestSetAuthCookies_EmptyTokensNotSet(t *testing.T) {
	rr := httptest.NewRecorder()
	SetAuthCookies(rr, map[string]interface{}{})

	assert.Nil(t, findCookie(t, rr, "access_token"))
	assert.Nil(t, findCookie(t, rr, "id_token"))
	assert.Nil(t, findCookie(t, rr, "refresh_token"))
}

func TestClearAuthCookies(t *testing.T) {
	rr := httptest.NewRecorder()
	ClearAuthCookies(rr)

	names := []string{"access_token", "id_token", "refresh_token"}
	for _, name := range names {
		c := findCookie(t, rr, name)
		require.NotNil(t, c, "cookie %s should be present in clear response", name)
		assert.Equal(t, "", c.Value)
		assert.Equal(t, -1, c.MaxAge, "MaxAge -1 signals deletion for %s", name)
		assert.True(t, c.HttpOnly)
		assert.True(t, c.Secure)
	}

	rt := findCookie(t, rr, "refresh_token")
	require.NotNil(t, rt)
	assert.Equal(t, "/auth/refresh", rt.Path)
}

