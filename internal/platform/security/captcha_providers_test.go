package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The verify protocol (form POST of secret/response/remoteip → JSON with
// `success`) is shared by reCAPTCHA, hCaptcha and Cloudflare Turnstile, so
// CAPTCHA_VERIFY_URL is all an operator should need to switch providers.
//
// Only reCAPTCHA v3 returns a `score`. The threshold used to be applied
// unconditionally, so for every provider that omits it the field decoded to
// Go's zero value 0.0 — below the 0.5 default — and a SUCCESSFUL verification
// was rejected. That silently locked the feature to reCAPTCHA v3.
func TestVerifyCaptcha_ProviderResponseShapes(t *testing.T) {
	serve := func(t *testing.T, body string) string {
		t.Helper()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, r.ParseForm())
			// Every provider reads these same form fields.
			assert.Equal(t, "test-secret", r.PostForm.Get("secret"))
			assert.Equal(t, "tok", r.PostForm.Get("response"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		return srv.URL
	}

	run := func(t *testing.T, body string) error {
		t.Helper()
		t.Setenv("CAPTCHA_SECRET", "test-secret")
		t.Setenv("CAPTCHA_VERIFY_URL", serve(t, body))
		return VerifyCaptcha(context.Background(), "tok", "203.0.113.7")
	}

	t.Run("turnstile success has no score and must pass", func(t *testing.T) {
		assert.NoError(t, run(t, `{"success":true,"challenge_ts":"2024-01-01T00:00:00Z","hostname":"example.com"}`))
	})

	t.Run("hcaptcha success has no score and must pass", func(t *testing.T) {
		assert.NoError(t, run(t, `{"success":true,"hostname":"example.com","credit":false}`))
	})

	t.Run("recaptcha v2 success has no score and must pass", func(t *testing.T) {
		assert.NoError(t, run(t, `{"success":true,"challenge_ts":"2024-01-01T00:00:00Z","hostname":"example.com"}`))
	})

	t.Run("recaptcha v3 above threshold passes", func(t *testing.T) {
		assert.NoError(t, run(t, `{"success":true,"score":0.9,"action":"signup"}`))
	})

	// The score check must still bite when the provider does supply one.
	t.Run("recaptcha v3 below threshold is rejected", func(t *testing.T) {
		err := run(t, `{"success":true,"score":0.1,"action":"signup"}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "below minimum threshold")
	})

	t.Run("an explicit zero score is still rejected", func(t *testing.T) {
		err := run(t, `{"success":true,"score":0.0}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "below minimum threshold")
	})

	t.Run("a provider failure verdict is rejected regardless of score", func(t *testing.T) {
		err := run(t, `{"success":false,"error-codes":["invalid-input-response"]}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "captcha verification failed")
	})

	// Fails open with no secret so local dev and tests need no external provider.
	t.Run("no configured secret skips verification entirely", func(t *testing.T) {
		t.Setenv("CAPTCHA_SECRET", "")
		assert.NoError(t, VerifyCaptcha(context.Background(), "", ""))
	})
}
