package util

import (
	"fmt"
	"net/url"
	"time"

	"github.com/maintainerd/auth/internal/crypto"
)

// GenerateSignedURL generates a signed URL with custom params and expiration.
func GenerateSignedURL(baseURL string, params map[string]string, ttl time.Duration) (string, error) {
	expires := time.Now().Add(ttl).Unix()

	// Build query params
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	values.Set("expires", fmt.Sprintf("%d", expires))

	// Compute signature using crypto package
	sig := crypto.ComputeSignature(values)
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
