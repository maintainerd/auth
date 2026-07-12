package cookie

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// authResponseStruct mimics a DTO with the fields SetAuthCookies inspects via reflection.
type authResponseStruct struct {
	AccessToken             string
	IDToken                 string
	RefreshToken            string
	ExpiresIn               int64
	AccessTokenCookieMaxAge int64
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

	at := findCookie(t, rr, "__Host-access_token")
	require.NotNil(t, at)
	assert.Equal(t, "at-value", at.Value)
	assert.Equal(t, 1800, at.MaxAge)
	assert.True(t, at.HttpOnly)
	assert.Equal(t, "/", at.Path)

	it := findCookie(t, rr, "__Host-id_token")
	require.NotNil(t, it)
	assert.Equal(t, "it-value", it.Value)
	assert.Equal(t, 1800, it.MaxAge)

	rt := findCookie(t, rr, "__Secure-refresh_token")
	require.NotNil(t, rt)
	assert.Equal(t, "rt-value", rt.Value)
	assert.Equal(t, 7*24*60*60, rt.MaxAge)
	assert.Equal(t, "/api/v1/refresh-token", rt.Path)
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

	at := findCookie(t, rr, "__Host-access_token")
	require.NotNil(t, at)
	assert.Equal(t, "at-struct", at.Value)
	assert.Equal(t, 900, at.MaxAge)

	it := findCookie(t, rr, "__Host-id_token")
	require.NotNil(t, it)
	assert.Equal(t, "it-struct", it.Value)

	rt := findCookie(t, rr, "__Secure-refresh_token")
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

	at := findCookie(t, rr, "__Host-access_token")
	require.NotNil(t, at)
	assert.Equal(t, "at-ptr", at.Value)
}

func TestSetAuthCookies_AccessTokenCookieMaxAge(t *testing.T) {
	rr := httptest.NewRecorder()
	data := &authResponseStruct{
		AccessToken:             "at-idle",
		IDToken:                 "it-idle",
		RefreshToken:            "rt-idle",
		ExpiresIn:               900,
		AccessTokenCookieMaxAge: 1800,
	}
	SetAuthCookies(rr, data)

	at := findCookie(t, rr, "__Host-access_token")
	require.NotNil(t, at)
	assert.Equal(t, 1800, at.MaxAge)

	it := findCookie(t, rr, "__Host-id_token")
	require.NotNil(t, it)
	assert.Equal(t, 1800, it.MaxAge)
}

func TestSetAuthCookies_EmptyTokensNotSet(t *testing.T) {
	rr := httptest.NewRecorder()
	SetAuthCookies(rr, map[string]interface{}{})

	assert.Nil(t, findCookie(t, rr, "__Host-access_token"))
	assert.Nil(t, findCookie(t, rr, "__Host-id_token"))
	assert.Nil(t, findCookie(t, rr, "__Secure-refresh_token"))
}

func TestAuthCookies_ForceSecureForPrefixedNames(t *testing.T) {
	oldSecure := config.CookieSecure
	oldSameSite := config.CookieSameSite
	t.Cleanup(func() {
		config.CookieSecure = oldSecure
		config.CookieSameSite = oldSameSite
	})
	config.CookieSecure = false
	config.CookieSameSite = "none"

	rr := httptest.NewRecorder()
	SetAuthCookies(rr, map[string]interface{}{
		"access_token":  "at",
		"id_token":      "it",
		"refresh_token": "rt",
	})

	for _, name := range []string{"__Host-access_token", "__Host-id_token", "__Secure-refresh_token"} {
		c := findCookie(t, rr, name)
		require.NotNil(t, c)
		assert.True(t, c.Secure)
		assert.Equal(t, http.SameSiteNoneMode, c.SameSite)
	}
}

func TestClearAuthCookies(t *testing.T) {
	rr := httptest.NewRecorder()
	ClearAuthCookies(rr)

	names := []string{"__Host-access_token", "__Host-id_token", "__Secure-refresh_token"}
	for _, name := range names {
		c := findCookie(t, rr, name)
		require.NotNil(t, c, "cookie %s should be present in clear response", name)
		assert.Equal(t, "", c.Value)
		assert.Equal(t, -1, c.MaxAge, "MaxAge -1 signals deletion for %s", name)
		assert.True(t, c.HttpOnly)
	}

	rt := findCookie(t, rr, "__Secure-refresh_token")
	require.NotNil(t, rt)
	assert.Equal(t, "/api/v1/refresh-token", rt.Path)
}

func TestCookieSecure(t *testing.T) {
	oldSecure := config.CookieSecure
	t.Cleanup(func() { config.CookieSecure = oldSecure })

	config.CookieSecure = true
	assert.True(t, cookieSecure())

	config.CookieSecure = false
	assert.False(t, cookieSecure())
}

func TestCookieSameSite(t *testing.T) {
	oldSameSite := config.CookieSameSite
	t.Cleanup(func() { config.CookieSameSite = oldSameSite })

	config.CookieSameSite = "strict"
	assert.Equal(t, http.SameSiteStrictMode, cookieSameSite())

	config.CookieSameSite = "lax"
	assert.Equal(t, http.SameSiteLaxMode, cookieSameSite())

	config.CookieSameSite = "none"
	assert.Equal(t, http.SameSiteNoneMode, cookieSameSite())

	config.CookieSameSite = "unknown"
	assert.Equal(t, http.SameSiteStrictMode, cookieSameSite())
}

func TestSecureForCookieName(t *testing.T) {
	oldSecure := config.CookieSecure
	t.Cleanup(func() { config.CookieSecure = oldSecure })

	config.CookieSecure = false

	assert.True(t, secureForCookieName("__Host-access_token"))
	assert.True(t, secureForCookieName("__Secure-refresh_token"))
	assert.False(t, secureForCookieName("plain_cookie"))
}

func TestSameSiteForCookie(t *testing.T) {
	oldSecure := config.CookieSecure
	oldSameSite := config.CookieSameSite
	t.Cleanup(func() {
		config.CookieSecure = oldSecure
		config.CookieSameSite = oldSameSite
	})

	// Empty override → falls back to the global COOKIE_SAMESITE config.
	config.CookieSameSite = "strict"
	assert.Equal(t, http.SameSiteStrictMode, sameSiteForCookie(true, ""))
	assert.Equal(t, http.SameSiteStrictMode, sameSiteForCookie(false, ""))

	config.CookieSameSite = "lax"
	assert.Equal(t, http.SameSiteLaxMode, sameSiteForCookie(true, ""))
	assert.Equal(t, http.SameSiteLaxMode, sameSiteForCookie(false, ""))

	config.CookieSameSite = "none"
	assert.Equal(t, http.SameSiteNoneMode, sameSiteForCookie(true, ""))
	assert.Equal(t, http.SameSiteLaxMode, sameSiteForCookie(false, ""))

	// Explicit per-policy override wins over the global config.
	config.CookieSameSite = "strict"
	assert.Equal(t, http.SameSiteLaxMode, sameSiteForCookie(true, "lax"))
	assert.Equal(t, http.SameSiteNoneMode, sameSiteForCookie(true, "none"))
	// None downgrades to Lax when the cookie is not Secure (browser requirement).
	assert.Equal(t, http.SameSiteLaxMode, sameSiteForCookie(false, "none"))
}

func TestSetTrustedDeviceCookie(t *testing.T) {
	rr := httptest.NewRecorder()
	SetTrustedDeviceCookie(rr, "td-secret", 3600)

	c := findCookie(t, rr, "__Host-mfa_trusted_device")
	require.NotNil(t, c)
	assert.Equal(t, "td-secret", c.Value)
	assert.Equal(t, 3600, c.MaxAge)
	assert.Equal(t, "/", c.Path)
	assert.True(t, c.HttpOnly, "trusted-device cookie must be httpOnly")
	assert.True(t, c.Secure, "__Host- prefix forces Secure")
}

func TestSetTrustedDeviceCookie_DefaultsMaxAgeWhenNonPositive(t *testing.T) {
	rr := httptest.NewRecorder()
	SetTrustedDeviceCookie(rr, "td-secret", 0)

	c := findCookie(t, rr, "__Host-mfa_trusted_device")
	require.NotNil(t, c)
	assert.Equal(t, trustedDeviceDefaultMaxAge, c.MaxAge)
}

func TestTrustedDeviceValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", nil)
	req.AddCookie(&http.Cookie{Name: "__Host-mfa_trusted_device", Value: "from-cookie"})
	assert.Equal(t, "from-cookie", TrustedDeviceValue(req))

	// Absent cookie → empty string, never an error.
	bare := httptest.NewRequest(http.MethodPost, "/api/v1/login", nil)
	assert.Equal(t, "", TrustedDeviceValue(bare))
}

func TestClearTrustedDeviceCookie(t *testing.T) {
	rr := httptest.NewRecorder()
	ClearTrustedDeviceCookie(rr)

	c := findCookie(t, rr, "__Host-mfa_trusted_device")
	require.NotNil(t, c)
	assert.Equal(t, "", c.Value)
	assert.Equal(t, -1, c.MaxAge, "MaxAge -1 signals deletion")
}

// Device trust must survive an ordinary logout, so ClearAuthCookies must not
// touch the trusted-device cookie.
func TestClearAuthCookies_LeavesTrustedDeviceCookie(t *testing.T) {
	rr := httptest.NewRecorder()
	ClearAuthCookies(rr)
	assert.Nil(t, findCookie(t, rr, "__Host-mfa_trusted_device"))
}

func TestSetDeviceIDCookie(t *testing.T) {
	rr := httptest.NewRecorder()
	SetDeviceIDCookie(rr, "dev-123")

	c := findCookie(t, rr, "__Host-device_id")
	require.NotNil(t, c)
	assert.Equal(t, "dev-123", c.Value)
	assert.Equal(t, deviceIDMaxAge, c.MaxAge)
	assert.Equal(t, "/", c.Path)
	assert.True(t, c.HttpOnly)
	assert.True(t, c.Secure)
}

func TestDeviceIDValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", nil)
	req.AddCookie(&http.Cookie{Name: "__Host-device_id", Value: "dev-123"})
	assert.Equal(t, "dev-123", DeviceIDValue(req))

	bare := httptest.NewRequest(http.MethodPost, "/api/v1/login", nil)
	assert.Equal(t, "", DeviceIDValue(bare))
}

func TestClearDeviceIDCookie(t *testing.T) {
	rr := httptest.NewRecorder()
	ClearDeviceIDCookie(rr)

	c := findCookie(t, rr, "__Host-device_id")
	require.NotNil(t, c)
	assert.Equal(t, "", c.Value)
	assert.Equal(t, -1, c.MaxAge)
}
