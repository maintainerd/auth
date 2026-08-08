package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// newTestFS is a minimal SPA dist: an HTML shell and one hashed asset.
func newTestFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":        {Data: []byte("<!doctype html><title>app</title>")},
		"assets/app-abc.js": {Data: []byte("console.log(1)")},
	}
}

func do(t *testing.T, h http.Handler, method, target string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Result()
}

func TestConfigJSPayloads(t *testing.T) {
	c := string(consoleConfigJS("https://identity.example.com"))
	if !strings.Contains(c, `VITE_AUTH_IDENTITY_BASE_URL: "https://identity.example.com"`) {
		t.Errorf("console config missing identity url: %s", c)
	}
	if !strings.Contains(c, `VITE_AUTH_API_BASE_URL: ""`) || !strings.Contains(c, `VITE_AUTH_PUBLIC_API_BASE_URL: ""`) {
		t.Errorf("console config should default APIs to same-origin (empty): %s", c)
	}
	i := string(identityConfigJS())
	if !strings.Contains(i, `VITE_AUTH_API_BASE_URL: ""`) {
		t.Errorf("identity config should default API to same-origin (empty): %s", i)
	}
}

func TestBuildSPA_ConfigJS(t *testing.T) {
	h := buildSPA(newTestFS(), []byte(`window.__ENV__ = {};`), nil)
	resp := do(t, h, http.MethodGet, "/config.js")
	body, _ := io.ReadAll(resp.Body)
	if string(body) != `window.__ENV__ = {};` {
		t.Errorf("unexpected config.js body: %q", body)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("config.js Cache-Control = %q, want no-store", got)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/javascript") {
		t.Errorf("config.js Content-Type = %q", ct)
	}
}

func TestBuildSPA_IndexAndSecurityHeaders(t *testing.T) {
	h := buildSPA(newTestFS(), nil, nil)
	resp := do(t, h, http.MethodGet, "/")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("index status = %d", resp.StatusCode)
	}
	if csp := resp.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'self'") {
		t.Errorf("missing/invalid CSP on index: %q", csp)
	}
	for _, kv := range [][2]string{
		{"X-Frame-Options", "DENY"},
		{"X-Content-Type-Options", "nosniff"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	} {
		if got := resp.Header.Get(kv[0]); got != kv[1] {
			t.Errorf("%s = %q, want %q", kv[0], got, kv[1])
		}
	}
}

func TestBuildSPA_AssetImmutableCache(t *testing.T) {
	h := buildSPA(newTestFS(), nil, nil)
	resp := do(t, h, http.MethodGet, "/assets/app-abc.js")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("asset status = %d", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("asset Cache-Control = %q, want immutable", cc)
	}
}

func TestBuildSPA_ClientRouteFallsBackToIndex(t *testing.T) {
	h := buildSPA(newTestFS(), nil, nil)
	resp := do(t, h, http.MethodGet, "/tenants/42/settings") // not a real file
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SPA fallback status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<title>app</title>") {
		t.Errorf("SPA fallback did not serve index.html: %q", body)
	}
}

func TestBuildSPA_APIMountAndStrip(t *testing.T) {
	var sawControl, sawData string
	control := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { sawControl = r.URL.Path })
	data := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { sawData = r.URL.Path })

	h := buildSPA(newTestFS(), nil, []mount{
		{pattern: "/api/", handler: control},
		{pattern: "/public-api/", strip: "/public-api", handler: data},
	})

	do(t, h, http.MethodGet, "/api/v1/tenants")
	if sawControl != "/api/v1/tenants" {
		t.Errorf("control handler saw %q, want /api/v1/tenants", sawControl)
	}

	// The console reaches the data plane at /public-api/...; the prefix must be
	// stripped so the router sees its normal /api/v1 path.
	do(t, h, http.MethodGet, "/public-api/api/v1/token")
	if sawData != "/api/v1/token" {
		t.Errorf("data handler saw %q, want /api/v1/token (stripped)", sawData)
	}
}
