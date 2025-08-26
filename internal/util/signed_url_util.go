package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

var secretKey = []byte("super-secret-key") // load from env/config in real apps

// GenerateSignedURL generates a signed URL with custom params and expiration.
func GenerateSignedURL(baseURL string, params map[string]string, ttl time.Duration) (string, error) {
	expires := time.Now().Add(ttl).Unix()

	// Build query params
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	values.Set("expires", fmt.Sprintf("%d", expires))

	// Compute signature
	sig := computeSignature(values)
	values.Set("sig", sig)

	return fmt.Sprintf("%s?%s", baseURL, values.Encode()), nil
}

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

// ValidateSignedURL validates a signed URL and returns the query params if valid.
func ValidateSignedURL(values url.Values) (map[string]string, error) {
	expires := values.Get("expires")
	sig := values.Get("sig")

	if expires == "" || sig == "" {
		return nil, fmt.Errorf("missing required parameters")
	}

	// Expiration check
	exp, err := parseInt64(expires)
	if err != nil {
		return nil, fmt.Errorf("invalid expires param")
	}
	if time.Now().Unix() > exp {
		return nil, fmt.Errorf("link expired")
	}

	// Recompute expected signature
	expected := cloneValues(values) // ✅ works everywhere
	expected.Del("sig")             // remove sig before recomputing

	expectedSig := computeSignature(expected)
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

// Helpers
func computeSignature(values url.Values) string {
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

func parseInt64(s string) (int64, error) {
	var n int64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// cloneValues is a safe clone for url.Values (works even if it's just a map[string][]string)
func cloneValues(v url.Values) url.Values {
	clone := make(url.Values, len(v))
	for key, vals := range v {
		copied := make([]string, len(vals))
		copy(copied, vals)
		clone[key] = copied
	}
	return clone
}
