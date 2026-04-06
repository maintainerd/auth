package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	SecurityHeadersMiddleware(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", rr.Header().Get("X-XSS-Protection"))
	assert.NotEmpty(t, rr.Header().Get("Content-Security-Policy"))
	assert.NotEmpty(t, rr.Header().Get("Referrer-Policy"))
	assert.NotEmpty(t, rr.Header().Get("Permissions-Policy"))
}

func TestSecurityContextMiddleware(t *testing.T) {
	var gotClientIP, gotUserAgent, gotRequestID string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClientIP, _ = r.Context().Value(ClientIPKey).(string)
		gotUserAgent, _ = r.Context().Value(UserAgentKey).(string)
		gotRequestID, _ = r.Context().Value(RequestIDKey).(string)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.RemoteAddr = "192.168.1.10:12345"
	rr := httptest.NewRecorder()
	SecurityContextMiddleware(next).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "192.168.1.10", gotClientIP)
	assert.Equal(t, "TestAgent/1.0", gotUserAgent)
	assert.NotEmpty(t, gotRequestID)
	assert.NotEmpty(t, rr.Header().Get("X-Request-ID"))
}

func TestRequestSizeLimitMiddleware(t *testing.T) {
	t.Run("within limit → passes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rr := httptest.NewRecorder()
		RequestSizeLimitMiddleware(1024)(okHandler()).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestTimeoutMiddleware(t *testing.T) {
	t.Run("context has deadline", func(t *testing.T) {
		var hasDeadline bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, hasDeadline = r.Context().Deadline()
			w.WriteHeader(http.StatusOK)
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		TimeoutMiddleware(5*time.Second)(next).ServeHTTP(rr, req)
		assert.True(t, hasDeadline)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestIPWhitelistMiddleware(t *testing.T) {
	cases := []struct {
		name       string
		cidrs      []string
		remoteAddr string
		wantStatus int
	}{
		{
			name:       "empty whitelist → passes all",
			cidrs:      []string{},
			remoteAddr: "10.0.0.1:1234",
			wantStatus: http.StatusOK,
		},
		{
			name:       "IP in whitelist → passes",
			cidrs:      []string{"10.0.0.0/24"},
			remoteAddr: "10.0.0.5:1234",
			wantStatus: http.StatusOK,
		},
		{
			name:       "IP not in whitelist → 403",
			cidrs:      []string{"10.0.0.0/24"},
			remoteAddr: "192.168.1.1:1234",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "invalid CIDR ignored → no effective whitelist → passes",
			cidrs:      []string{"not-a-cidr"},
			remoteAddr: "1.2.3.4:5678",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remoteAddr
			rr := httptest.NewRecorder()
			IPWhitelistMiddleware(tc.cidrs)(okHandler()).ServeHTTP(rr, req)
			assert.Equal(t, tc.wantStatus, rr.Code)
		})
	}
}

func TestExtractClientIP(t *testing.T) {
	cases := []struct {
		name       string
		xff        string
		xri        string
		cfip       string
		remoteAddr string
		want       string
	}{
		{
			name:       "X-Forwarded-For first valid IP",
			xff:        "203.0.113.5, 10.0.0.1",
			remoteAddr: "127.0.0.1:1234",
			want:       "203.0.113.5",
		},
		{
			name:       "X-Real-IP when no XFF",
			xri:        "203.0.113.10",
			remoteAddr: "127.0.0.1:1234",
			want:       "203.0.113.10",
		},
		{
			name:       "CF-Connecting-IP when no XFF or XRI",
			cfip:       "203.0.113.20",
			remoteAddr: "127.0.0.1:1234",
			want:       "203.0.113.20",
		},
		{
			name:       "fallback to RemoteAddr",
			remoteAddr: "10.0.0.5:9999",
			want:       "10.0.0.5",
		},
		{
			name:       "XFF with invalid entry skipped",
			xff:        "not-an-ip, 198.51.100.1",
			remoteAddr: "127.0.0.1:1234",
			want:       "198.51.100.1",
		},
		{
			name:       "RemoteAddr with no port → SplitHostPort fails → raw value returned",
			remoteAddr: "192.168.1.1",
			want:       "192.168.1.1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			if tc.xri != "" {
				req.Header.Set("X-Real-IP", tc.xri)
			}
			if tc.cfip != "" {
				req.Header.Set("CF-Connecting-IP", tc.cfip)
			}
			req.RemoteAddr = tc.remoteAddr
			assert.Equal(t, tc.want, extractClientIP(req))
		})
	}
}

func TestSecurityHeadersMiddleware_Production(t *testing.T) {
	t.Setenv("ENV", "production")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	SecurityHeadersMiddleware(okHandler()).ServeHTTP(rr, req)
	assert.NotEmpty(t, rr.Header().Get("Strict-Transport-Security"))
}
