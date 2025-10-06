package util

import (
	"encoding/json"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

func Success(w http.ResponseWriter, data interface{}, message string) {
	writeJSON(w, http.StatusOK, response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

func Created(w http.ResponseWriter, data interface{}, message string) {
	writeJSON(w, http.StatusCreated, response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

func Error(w http.ResponseWriter, status int, err string, details ...any) {
	resp := response{
		Success: false,
		Error:   err,
	}
	if len(details) > 0 {
		resp.Details = details[0]
	}
	writeJSON(w, status, resp)
}

func ValidationError(w http.ResponseWriter, err error) {
	if ve, ok := err.(validation.Errors); ok {
		Error(w, http.StatusBadRequest, "Validation failed", ve)
		return
	}
	Error(w, http.StatusBadRequest, "Validation failed", err.Error())
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// AuthSuccess handles authentication response with optional cookie delivery
// Based on X-Token-Delivery header: "cookie" or "body" (default)
func AuthSuccess(w http.ResponseWriter, r *http.Request, authResponse interface{}, message string) {
	tokenDelivery := r.Header.Get("X-Token-Delivery")

	if tokenDelivery == "cookie" {
		// Set tokens as secure HTTP-only cookies
		setAuthCookies(w, authResponse)
	}

	// Always return the full token response in body (same as default)
	writeJSON(w, http.StatusOK, response{
		Success: true,
		Data:    authResponse,
		Message: message,
	})
}

// AuthCreated handles authentication response for registration with optional cookie delivery
func AuthCreated(w http.ResponseWriter, r *http.Request, authResponse interface{}, message string) {
	tokenDelivery := r.Header.Get("X-Token-Delivery")

	if tokenDelivery == "cookie" {
		// Set tokens as secure HTTP-only cookies
		setAuthCookies(w, authResponse)
	}

	// Always return the full token response in body (same as default)
	writeJSON(w, http.StatusCreated, response{
		Success: true,
		Data:    authResponse,
		Message: message,
	})
}

// setAuthCookies sets authentication tokens as secure HTTP-only cookies using interface{}
func setAuthCookies(w http.ResponseWriter, authResponse interface{}) {
	// Extract token values using type assertion or reflection
	var accessToken, idToken, refreshToken string
	var expiresIn int64 = 3600 // default

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
			Path:     "/auth/refresh",  // Restrict to refresh endpoint only
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
