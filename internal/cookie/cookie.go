package cookie

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/maintainerd/auth/internal/config"
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

// SetAuthCookies sets authentication tokens as secure HTTP-only cookies.
//
// Cookie naming conventions:
//   - __Host-access_token  / __Host-id_token  — __Host- prefix (Secure, Path=/, no Domain)
//   - __Secure-refresh_token                  — __Secure- prefix (Secure, narrow path)
//
// The __Host- prefix prevents subdomain fixation attacks; __Secure- is used for
// the refresh token because it has a non-root path (/auth/refresh).
func SetAuthCookies(w http.ResponseWriter, authResponse interface{}) {
	var accessToken, idToken, refreshToken string
	var expiresIn int64 = 3600

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
		}
	}

	secure := cookieSecure()
	sameSite := cookieSameSite()

	if accessToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "__Host-access_token",
			Value:    accessToken,
			Path:     "/",
			MaxAge:   int(expiresIn),
			HttpOnly: true,
			Secure:   secure,
			SameSite: sameSite,
		})
	}

	if idToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "__Host-id_token",
			Value:    idToken,
			Path:     "/",
			MaxAge:   3600,
			HttpOnly: true,
			Secure:   secure,
			SameSite: sameSite,
		})
	}

	if refreshToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "__Secure-refresh_token",
			Value:    refreshToken,
			Path:     "/auth/refresh",
			MaxAge:   7 * 24 * 60 * 60,
			HttpOnly: true,
			Secure:   secure,
			SameSite: sameSite,
		})
	}
}

// ClearAuthCookies clears all authentication-related cookies.
func ClearAuthCookies(w http.ResponseWriter) {
	secure := cookieSecure()
	sameSite := cookieSameSite()

	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-id_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "__Secure-refresh_token",
		Value:    "",
		Path:     "/auth/refresh",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
}
