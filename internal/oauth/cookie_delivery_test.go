package oauth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// jsonHandler returns a handler that writes the given status + body.
func jsonHandler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func hasAuthCookie(res *http.Response) bool {
	for _, c := range res.Cookies() {
		if c.Name == "__Host-access_token" {
			return true
		}
	}
	return false
}

func TestDeliverAuthCookies(t *testing.T) {
	const tokenBody = `{"access_token":"at","id_token":"it","refresh_token":"rt","expires_in":3600,"token_type":"Bearer"}`

	t.Run("no header — passthrough, no cookies", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
		rec := httptest.NewRecorder()
		deliverAuthCookies(rec, req, jsonHandler(http.StatusOK, tokenBody))

		res := rec.Result()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", res.StatusCode)
		}
		if rec.Body.String() != tokenBody {
			t.Fatalf("body not passed through: %q", rec.Body.String())
		}
		if hasAuthCookie(res) {
			t.Fatal("auth cookie set without X-Token-Delivery header")
		}
	})

	t.Run("cookie delivery on 2xx raw token response", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
		req.Header.Set("X-Token-Delivery", "cookie")
		rec := httptest.NewRecorder()
		deliverAuthCookies(rec, req, jsonHandler(http.StatusOK, tokenBody))

		res := rec.Result()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", res.StatusCode)
		}
		if rec.Body.String() != tokenBody {
			t.Fatalf("body not passed through: %q", rec.Body.String())
		}
		if !hasAuthCookie(res) {
			t.Fatal("expected __Host-access_token cookie to be set")
		}
	})

	t.Run("cookie delivery on wrapped {data:{...}} envelope", func(t *testing.T) {
		body := `{"success":true,"message":"ok","data":` + tokenBody + `}`
		req := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
		req.Header.Set("X-Token-Delivery", "cookie")
		rec := httptest.NewRecorder()
		deliverAuthCookies(rec, req, jsonHandler(http.StatusOK, body))

		if !hasAuthCookie(rec.Result()) {
			t.Fatal("expected cookie to be set from data envelope")
		}
	})

	t.Run("no cookies on error status", func(t *testing.T) {
		errBody := `{"error":"invalid_grant"}`
		req := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
		req.Header.Set("X-Token-Delivery", "cookie")
		rec := httptest.NewRecorder()
		deliverAuthCookies(rec, req, jsonHandler(http.StatusBadRequest, errBody))

		res := rec.Result()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", res.StatusCode)
		}
		if rec.Body.String() != errBody {
			t.Fatalf("error body not passed through: %q", rec.Body.String())
		}
		if hasAuthCookie(res) {
			t.Fatal("auth cookie must not be set on a non-2xx response")
		}
	})

	t.Run("cookie header but no access_token in body — no cookies", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
		req.Header.Set("X-Token-Delivery", "cookie")
		rec := httptest.NewRecorder()
		deliverAuthCookies(rec, req, jsonHandler(http.StatusOK, `{"token_type":"Bearer"}`))

		if hasAuthCookie(rec.Result()) {
			t.Fatal("no cookie should be set without an access_token")
		}
		if !strings.Contains(rec.Body.String(), "Bearer") {
			t.Fatal("body not passed through")
		}
	})
}
