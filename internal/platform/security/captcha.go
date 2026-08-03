package security

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultCaptchaVerifyURL = "https://www.google.com/recaptcha/api/siteverify"
const defaultCaptchaMinScore = 0.5

type captchaVerifyResponse struct {
	Success bool `json:"success"`
	// Score is a POINTER because "no score" and "score 0.0" are different
	// answers. Only reCAPTCHA v3 scores a request; reCAPTCHA v2, hCaptcha's free
	// tier and Cloudflare Turnstile all omit the field on an otherwise
	// successful verification.
	Score      *float64 `json:"score"`
	Action     string   `json:"action"`
	ErrorCodes []string `json:"error-codes"`
}

func captchaMinScore() float64 {
	v := strings.TrimSpace(os.Getenv("CAPTCHA_MIN_SCORE"))
	if v == "" {
		return defaultCaptchaMinScore
	}
	score, err := strconv.ParseFloat(v, 64)
	if err != nil || score < 0 || score > 1 {
		slog.Warn("invalid CAPTCHA_MIN_SCORE, using default", "value", v, "default", defaultCaptchaMinScore)
		return defaultCaptchaMinScore
	}
	return score
}

// VerifyCaptcha validates the browser-provided CAPTCHA response when a provider
// secret is configured. Without CAPTCHA_SECRET it fails open so local dev and
// tests do not require an external provider.
func VerifyCaptcha(ctx context.Context, responseToken, remoteIP string) error {
	secret := strings.TrimSpace(os.Getenv("CAPTCHA_SECRET"))
	if secret == "" {
		return nil
	}
	responseToken = strings.TrimSpace(responseToken)
	if responseToken == "" {
		return fmt.Errorf("captcha token is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	verifyURL := strings.TrimSpace(os.Getenv("CAPTCHA_VERIFY_URL"))
	if verifyURL == "" {
		verifyURL = defaultCaptchaVerifyURL
	}

	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", responseToken)
	if strings.TrimSpace(remoteIP) != "" {
		form.Set("remoteip", strings.TrimSpace(remoteIP))
	}

	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("captcha provider returned status %d", resp.StatusCode)
	}

	var body captchaVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}
	if !body.Success {
		return fmt.Errorf("captcha verification failed")
	}
	// Apply the risk threshold only when the provider actually scored the
	// request. As a plain float64 this defaulted to 0.0 for every provider that
	// does not score, which is below the 0.5 default — so a successful Turnstile,
	// hCaptcha or reCAPTCHA-v2 verification was rejected anyway, and the endpoint
	// was effectively locked to reCAPTCHA v3. Every provider's own pass/fail
	// verdict is `success`, which is checked above.
	if body.Score != nil {
		minScore := captchaMinScore()
		if *body.Score < minScore {
			return fmt.Errorf("captcha score %.2f is below minimum threshold %.2f", *body.Score, minScore)
		}
	}
	return nil
}
