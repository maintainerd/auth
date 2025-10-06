package util

import (
	"net/http"
	"reflect"
)

// SetAuthCookies sets authentication tokens as secure HTTP-only cookies
func SetAuthCookies(w http.ResponseWriter, authResponse interface{}) {
	// Extract token values using type assertion or reflection
	var accessToken, idToken, refreshToken string
	var expiresIn int64 = 3600 // default

	// Try map[string]interface{} first (for backward compatibility)
	if response, ok := authResponse.(map[string]interface{}); ok {
		if at, exists := response["access_token"]; exists {
			if atStr, ok := at.(string); ok {
				accessToken = atStr
			}
		}
		if it, exists := response["id_token"]; exists {
			if itStr, ok := it.(string); ok {
				idToken = itStr
			}
		}
		if rt, exists := response["refresh_token"]; exists {
			if rtStr, ok := rt.(string); ok {
				refreshToken = rtStr
			}
		}
		if ei, exists := response["expires_in"]; exists {
			if eiInt, ok := ei.(int64); ok {
				expiresIn = eiInt
			}
		}
	} else {
		// Use reflection for struct types (like AuthResponseDto)
		v := reflect.ValueOf(authResponse)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() == reflect.Struct {
			// Get AccessToken field
			if field := v.FieldByName("AccessToken"); field.IsValid() && field.Kind() == reflect.String {
				accessToken = field.String()
			}
			// Get IDToken field
			if field := v.FieldByName("IDToken"); field.IsValid() && field.Kind() == reflect.String {
				idToken = field.String()
			}
			// Get RefreshToken field
			if field := v.FieldByName("RefreshToken"); field.IsValid() && field.Kind() == reflect.String {
				refreshToken = field.String()
			}
			// Get ExpiresIn field
			if field := v.FieldByName("ExpiresIn"); field.IsValid() && field.Kind() == reflect.Int64 {
				expiresIn = field.Int()
			}
		}
	}

	// Set access token cookie (short-lived)
	if accessToken != "" {
		accessTokenCookie := &http.Cookie{
			Name:     "access_token",
			Value:    accessToken,
			Path:     "/",
			MaxAge:   int(expiresIn),
			HttpOnly: true,
			Secure:   true, // Only send over HTTPS in production
			SameSite: http.SameSiteStrictMode,
		}
		http.SetCookie(w, accessTokenCookie)
	}

	// Set ID token cookie (short-lived)
	if idToken != "" {
		idTokenCookie := &http.Cookie{
			Name:     "id_token",
			Value:    idToken,
			Path:     "/",
			MaxAge:   3600, // 1 hour
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
		}
		http.SetCookie(w, idTokenCookie)
	}

	// Set refresh token cookie (long-lived, more secure)
	if refreshToken != "" {
		refreshTokenCookie := &http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			Path:     "/auth/refresh", // Restrict to refresh endpoint only
			MaxAge:   7 * 24 * 60 * 60, // 7 days
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
		}
		http.SetCookie(w, refreshTokenCookie)
	}
}

// ClearAuthCookies clears all authentication-related cookies
func ClearAuthCookies(w http.ResponseWriter) {
	// Clear access token cookie
	clearCookie := &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, clearCookie)

	// Clear ID token cookie
	clearCookie = &http.Cookie{
		Name:     "id_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, clearCookie)

	// Clear refresh token cookie
	clearCookie = &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/auth/refresh",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, clearCookie)
}
