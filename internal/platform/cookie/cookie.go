package cookie

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/shared"
)

// cookieSecure returns whether cookies should set Secure=true.
// Defaults true; can be disabled for local dev via COOKIE_SECURE=false.
func cookieSecure() bool {
	return config.CookieSecure
}

// cookieSameSite maps the COOKIE_SAMESITE config value to an http.SameSite constant.
func cookieSameSite() http.SameSite {
	switch strings.ToLower(config.CookieSameSite) {
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteStrictMode
	}
}

func secureForCookieName(name string) bool {
	if strings.HasPrefix(name, "__Host-") || strings.HasPrefix(name, "__Secure-") {
		return true
	}
	return cookieSecure()
}

func sameSiteString(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	case "strict":
		return http.SameSiteStrictMode
	default:
		return cookieSameSite()
	}
}

func sameSiteForCookie(secure bool, override string) http.SameSite {
	sameSite := sameSiteString(override)
	if sameSite == http.SameSiteNoneMode && !secure {
		return http.SameSiteLaxMode
	}
	return sameSite
}

func setAuthCookie(w http.ResponseWriter, name, value, path string, maxAge int, opts authCookieOptions) {
	secure := secureForCookieName(name)
	if opts.Secure != nil && !strings.HasPrefix(name, "__Host-") && !strings.HasPrefix(name, "__Secure-") {
		secure = *opts.Secure
	}
	httpOnly := true
	if opts.HTTPOnly != nil {
		httpOnly = *opts.HTTPOnly
	}
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- cookie attributes set per-name via helpers
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		Secure:   secure,
		SameSite: sameSiteForCookie(secure, opts.SameSite),
	})
}

// Auth cookies always use the __Host-/__Secure- prefixes. These prefixes are a
// browser-enforced hardening (host-only, Secure-required) and are honored on
// localhost, which browsers treat as a secure context. secureForCookieName
// forces Secure=true for any prefixed name regardless of COOKIE_SECURE.
func accessTokenCookieName() string { return "__Host-access_token" }

func idTokenCookieName() string { return "__Host-id_token" }

func refreshTokenCookieName() string { return "__Secure-refresh_token" }

// refreshTokenCookiePath scopes the refresh-token cookie to the refresh
// endpoint only, so it is never sent on ordinary API requests. It must match the
// mounted route path (POST /api/v1/refresh-token).
const refreshTokenCookiePath = "/api/v1/refresh-token"

type authCookieOptions struct {
	Secure        *bool
	HTTPOnly      *bool
	SameSite      string
	RefreshMaxAge int
}

// SetAuthCookies sets authentication tokens as secure HTTP-only cookies.
//
// Cookie naming conventions:
//   - __Host-access_token  / __Host-id_token  — __Host- prefix (Secure, Path=/, no Domain)
//   - __Secure-refresh_token                  — __Secure- prefix (Secure, narrow path)
//
// The __Host- prefix prevents subdomain fixation attacks; __Secure- is used for
// the refresh token because it has a non-root path (/api/v1/refresh-token) so it
// is only sent to the refresh endpoint.
func SetAuthCookies(w http.ResponseWriter, authResponse interface{}) {
	var accessToken, idToken, refreshToken string
	var expiresIn int64 = shared.DefaultAccessTokenExpiresIn
	opts := authCookieOptions{RefreshMaxAge: 7 * 24 * 60 * 60}

	if response, ok := authResponse.(map[string]interface{}); ok {
		if at, exists := response["access_token"]; exists {
			if s, ok := at.(string); ok {
				accessToken = s
			}
		}
		if it, exists := response["id_token"]; exists {
			if s, ok := it.(string); ok {
				idToken = s
			}
		}
		if rt, exists := response["refresh_token"]; exists {
			if s, ok := rt.(string); ok {
				refreshToken = s
			}
		}
		if ei, exists := response["expires_in"]; exists {
			if v, ok := ei.(int64); ok {
				expiresIn = v
			}
		}
	} else {
		v := reflect.ValueOf(authResponse)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() == reflect.Struct {
			if f := v.FieldByName("AccessToken"); f.IsValid() && f.Kind() == reflect.String {
				accessToken = f.String()
			}
			if f := v.FieldByName("IDToken"); f.IsValid() && f.Kind() == reflect.String {
				idToken = f.String()
			}
			if f := v.FieldByName("RefreshToken"); f.IsValid() && f.Kind() == reflect.String {
				refreshToken = f.String()
			}
			if f := v.FieldByName("ExpiresIn"); f.IsValid() && f.Kind() == reflect.Int64 {
				expiresIn = f.Int()
			}
			if f := v.FieldByName("CookieSecure"); f.IsValid() && !f.IsNil() {
				b := f.Elem().Bool()
				opts.Secure = &b
			}
			if f := v.FieldByName("CookieHTTPOnly"); f.IsValid() && !f.IsNil() {
				b := f.Elem().Bool()
				opts.HTTPOnly = &b
			}
			if f := v.FieldByName("CookieSameSite"); f.IsValid() && f.Kind() == reflect.String {
				opts.SameSite = f.String()
			}
			if f := v.FieldByName("RefreshTokenMaxAge"); f.IsValid() && f.Kind() == reflect.Int && f.Int() > 0 {
				opts.RefreshMaxAge = int(f.Int())
			}
		}
	}

	if accessToken != "" {
		setAuthCookie(w, accessTokenCookieName(), accessToken, "/", int(expiresIn), opts)
	}

	if idToken != "" {
		setAuthCookie(w, idTokenCookieName(), idToken, "/", shared.DefaultAccessTokenExpiresIn, opts)
	}

	if refreshToken != "" {
		setAuthCookie(w, refreshTokenCookieName(), refreshToken, refreshTokenCookiePath, opts.RefreshMaxAge, opts)
	}
}

// ClearAuthCookies clears all authentication-related cookies.
func ClearAuthCookies(w http.ResponseWriter) {
	for _, name := range []string{accessTokenCookieName(), "access_token"} {
		setAuthCookie(w, name, "", "/", -1, authCookieOptions{})
	}
	for _, name := range []string{idTokenCookieName(), "id_token"} {
		setAuthCookie(w, name, "", "/", -1, authCookieOptions{})
	}
	for _, name := range []string{refreshTokenCookieName(), "refresh_token"} {
		setAuthCookie(w, name, "", refreshTokenCookiePath, -1, authCookieOptions{})
	}
}
