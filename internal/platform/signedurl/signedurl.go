/*
Package signedurl provides signed URL utilities for secure link generation and validation.

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
package signedurl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// CONFIGURATION
// ============================================================================

// Signer signs and validates URL query parameters with a preloaded HMAC key.
type Signer struct {
	secret []byte
}

var (
	defaultSignerMu sync.RWMutex
	defaultSigner   *Signer
)

// NewSigner returns a signed URL helper using the supplied HMAC key.
func NewSigner(secret []byte) (*Signer, error) {
	if len(secret) == 0 {
		return nil, fmt.Errorf("HMAC secret key is required")
	}
	copied := make([]byte, len(secret))
	copy(copied, secret)
	return &Signer{secret: copied}, nil
}

// Configure installs the process-wide signer used by the package-level helpers.
func Configure(secret []byte) error {
	signer, err := NewSigner(secret)
	if err != nil {
		return err
	}

	defaultSignerMu.Lock()
	defaultSigner = signer
	defaultSignerMu.Unlock()
	return nil
}

func configuredSigner() (*Signer, error) {
	defaultSignerMu.RLock()
	signer := defaultSigner
	defaultSignerMu.RUnlock()
	if signer == nil {
		return nil, fmt.Errorf("signed URL signer is not configured")
	}
	return signer, nil
}

func resetDefaultSignerForTest() {
	defaultSignerMu.Lock()
	defaultSigner = nil
	defaultSignerMu.Unlock()
}

// ============================================================================
// PUBLIC API
// ============================================================================

// GenerateSignedURL generates a signed URL with custom params and expiration
// Complies with SOC2 CC6.1 and ISO27001 A.13.2.1
var GenerateSignedURL = generateSignedURL

func generateSignedURL(baseURL string, params map[string]string, ttl time.Duration) (string, error) {
	signer, err := configuredSigner()
	if err != nil {
		return "", err
	}
	return signer.GenerateSignedURL(baseURL, params, ttl)
}

// GenerateSignedURL generates a signed URL with custom params and expiration.
func (s *Signer) GenerateSignedURL(baseURL string, params map[string]string, ttl time.Duration) (string, error) {
	expires := time.Now().Add(ttl).Unix()

	// Build query params
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	values.Set("expires", fmt.Sprintf("%d", expires))

	// Compute signature
	sig, err := s.computeSignature(values)
	if err != nil {
		return "", fmt.Errorf("failed to compute signature: %w", err)
	}
	values.Set("sig", sig)

	return fmt.Sprintf("%s?%s", baseURL, values.Encode()), nil
}

// ValidateSignedURL validates a signed URL and returns the query params if valid
// Complies with SOC2 CC6.1 and ISO27001 A.13.2.1
func ValidateSignedURL(values url.Values) (map[string]string, error) {
	signer, err := configuredSigner()
	if err != nil {
		return nil, err
	}
	return signer.ValidateSignedURL(values)
}

// ValidateSignedURL validates a signed URL and returns the query params if valid.
func (s *Signer) ValidateSignedURL(values url.Values) (map[string]string, error) {
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

	expectedSig, err := s.computeSignature(expected)
	if err != nil {
		return nil, fmt.Errorf("failed to compute signature: %w", err)
	}
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
var ConvertToFrontendURL = convertToFrontendURL

func convertToFrontendURL(apiSignedURL, frontendBaseURL string) (string, error) {
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

// computeSignature generates HMAC signature for url.Values.
// Used internally by signed URL functions.
func computeSignature(values url.Values) (string, error) {
	signer, err := configuredSigner()
	if err != nil {
		return "", err
	}
	return signer.computeSignature(values)
}

func (s *Signer) computeSignature(values url.Values) (string, error) {
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
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(data))
	return base64.URLEncoding.EncodeToString(mac.Sum(nil)), nil
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
