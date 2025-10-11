/*
Package util provides signed URL utilities for secure link generation and validation.

This module implements cryptographically signed URLs for secure operations like:
- Email verification links
- Password reset links  
- Invite registration links
- Time-limited access URLs

SECURITY FEATURES:
- HMAC-SHA256 signature generation and validation
- Configurable expiration times
- Tamper-proof URL parameters
- Frontend/API URL conversion

COMPLIANCE:
- SOC2 CC6.1 (Logical Access Controls)
- ISO27001 A.13.2.1 (Information Transfer)

USAGE:
	// Generate a signed URL
	params := map[string]string{"invite_token": "abc123"}
	signedURL, err := util.GenerateSignedURL(baseURL, params, 24*time.Hour)

	// Validate a signed URL
	values := url.Values{...}
	params, err := util.ValidateSignedURL(values)

	// Convert API URL to frontend URL
	frontendURL, err := util.ConvertToFrontendURL(apiURL, frontendBase)
*/
package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ============================================================================
// CONFIGURATION
// ============================================================================

// Secret key for HMAC signing (should be loaded from config in production)
// TODO: Load from environment/config for production security
var secretKey = []byte("super-secret-key")

// ============================================================================
// PUBLIC API
// ============================================================================

// GenerateSignedURL generates a signed URL with custom params and expiration
// Complies with SOC2 CC6.1 and ISO27001 A.13.2.1
func GenerateSignedURL(baseURL string, params map[string]string, ttl time.Duration) (string, error) {
	expires := time.Now().Add(ttl).Unix()

	// Build query params
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	values.Set("expires", fmt.Sprintf("%d", expires))

	// Compute signature
	sig := ComputeSignature(values)
	values.Set("sig", sig)

	return fmt.Sprintf("%s?%s", baseURL, values.Encode()), nil
}

// ValidateSignedURL validates a signed URL and returns the query params if valid
// Complies with SOC2 CC6.1 and ISO27001 A.13.2.1
func ValidateSignedURL(values url.Values) (map[string]string, error) {
	expires := values.Get("expires")
	sig := values.Get("sig")

	if expires == "" || sig == "" {
		return nil, fmt.Errorf("missing required parameters")
	}

	// Expiration check
	exp, err := strconv.ParseInt(expires, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid expires param")
	}
	if time.Now().Unix() > exp {
		return nil, fmt.Errorf("link expired")
	}

	// Recompute expected signature
	expected := cloneValues(values)
	expected.Del("sig") // remove sig before recomputing

	expectedSig := ComputeSignature(expected)
	if !hmac.Equal([]byte(expectedSig), []byte(sig)) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Return clean params (excluding sig)
	params := map[string]string{}
	for k, v := range expected {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	return params, nil
}

// ConvertToFrontendURL converts an API signed URL to a frontend URL
// Preserves all query parameters including the signature
func ConvertToFrontendURL(apiSignedURL, frontendBaseURL string) (string, error) {
	parsedURL, err := url.Parse(apiSignedURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse API signed URL: %w", err)
	}

	// Extract query parameters
	query := parsedURL.RawQuery

	// Construct frontend URL with the same path and query
	frontendURL := fmt.Sprintf("%s?%s", frontendBaseURL, query)
	return frontendURL, nil
}

// ============================================================================
// INTERNAL UTILITIES
// ============================================================================

// ComputeSignature generates HMAC signature for url.Values
// Used internally by signed URL functions
func ComputeSignature(values url.Values) string {
	// Sort keys for deterministic signing
	keys := make([]string, 0, len(values))
	for k := range values {
		if k == "sig" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build canonical string
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(values.Get(k))
		sb.WriteString("&")
	}
	data := strings.TrimRight(sb.String(), "&")

	// Compute HMAC SHA256
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(data))
	return base64.URLEncoding.EncodeToString(mac.Sum(nil))
}

// cloneValues safely clones url.Values to avoid mutation
func cloneValues(v url.Values) url.Values {
	clone := make(url.Values, len(v))
	for key, vals := range v {
		copied := make([]string, len(vals))
		copy(copied, vals)
		clone[key] = copied
	}
	return clone
}
