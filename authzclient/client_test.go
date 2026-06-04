package authzclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maintainerd/auth/internal/iam"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestNewRefreshCanAndClose(t *testing.T) {
	tokenCalls := 0
	bundleCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/oauth/token":
			tokenCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token"}`))
		case "/api/v1/services/me/policy-bundle":
			bundleCalls++
			if r.Header.Get("Authorization") != "Bearer token" {
				t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
			}
			w.Header().Set("ETag", `"v1"`)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"service":"serviceA","version":"v1","policies":[{"version":"v1","statement":[{"effect":"allow","action":["serviceB:invoke"],"resource":["serviceB:grpc"]}]}],"generated_at":"2026-06-04T00:00:00Z"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := New(Config{AuthServerURL: server.URL, ClientID: "clientA", ClientSecret: "secret", PollInterval: time.Hour, HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	if tokenCalls != 1 || bundleCalls != 1 {
		t.Fatalf("tokenCalls=%d bundleCalls=%d", tokenCalls, bundleCalls)
	}
	if !client.Can("serviceB:invoke", "serviceB:grpc") {
		t.Fatal("Can() = false")
	}
	if !client.CanPrincipal("serviceA", "serviceB:invoke", "serviceB:grpc") {
		t.Fatal("CanPrincipal() = false")
	}
}

func TestRefresh_NotModifiedKeepsBundle(t *testing.T) {
	notModified := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/oauth/token":
			_, _ = w.Write([]byte(`{"access_token":"token"}`))
		case "/api/v1/services/me/policy-bundle":
			if notModified {
				if r.Header.Get("If-None-Match") != `"v1"` {
					t.Fatalf("if-none-match = %q", r.Header.Get("If-None-Match"))
				}
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("ETag", `"v1"`)
			_, _ = w.Write([]byte(`{"data":{"service":"serviceA","version":"v1","policies":[{"version":"v1","statement":[{"effect":"allow","action":["*"],"resource":["*"]}]}],"generated_at":"2026-06-04T00:00:00Z"}}`))
		}
	}))
	defer server.Close()

	client, err := New(Config{AuthServerURL: server.URL, ClientID: "clientA", PollInterval: time.Hour})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()
	notModified = true
	if err := client.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if !client.Can("anything", "anything") {
		t.Fatal("cached bundle was not retained")
	}
}

func TestNewValidationAndDecodeBundle(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("New() error = nil for missing URL")
	}
	if _, err := New(Config{AuthServerURL: "https://auth.example.com"}); err == nil {
		t.Fatal("New() error = nil for missing client id")
	}
	if _, err := New(Config{AuthServerURL: "http://[::1", ClientID: "clientA"}); err == nil {
		t.Fatal("New() error = nil when initial refresh fails")
	}
	if _, err := DecodeBundle([]byte(`{"service":"s","version":"v1"}`)); err != nil {
		t.Fatalf("DecodeBundle() error = %v", err)
	}
	if _, err := DecodeBundle([]byte(`{bad`)); err == nil {
		t.Fatal("DecodeBundle() bad json error = nil")
	}
}

func TestNewWithWebhookListenBranchAndPollTick(t *testing.T) {
	var bundleCalls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/oauth/token":
			_, _ = w.Write([]byte(`{"access_token":"token"}`))
		case "/api/v1/services/me/policy-bundle":
			bundleCalls.Add(1)
			_, _ = w.Write([]byte(`{"data":{"service":"serviceA","version":"v1","policies":[],"generated_at":"2026-06-04T00:00:00Z"}}`))
		}
	}))
	defer server.Close()

	c, err := New(Config{AuthServerURL: server.URL, ClientID: "clientA", PollInterval: time.Millisecond, WebhookListen: "bad listen address"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	c.Close()
	if bundleCalls.Load() < 2 {
		t.Fatalf("bundleCalls = %d", bundleCalls.Load())
	}
}

func TestCan_EmptyCachesDeny(t *testing.T) {
	c := &Client{bundles: map[string]*iam.PolicyBundle{}}
	if c.Can("a", "r") {
		t.Fatal("Can() = true for missing bundle")
	}
	if c.CanPrincipal("serviceA", "a", "r") {
		t.Fatal("CanPrincipal() = true for missing bundle")
	}
}

func TestRefresh_ErrorPaths(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
		path string
	}{
		{"token status error", ``, http.StatusUnauthorized, "/api/v1/oauth/token"},
		{"token missing access token", `{}`, http.StatusOK, "/api/v1/oauth/token"},
		{"token malformed json", `{bad`, http.StatusOK, "/api/v1/oauth/token"},
		{"bundle status error", `{"access_token":"token"}`, http.StatusForbidden, "/api/v1/services/me/policy-bundle"},
		{"bundle malformed json", `{"access_token":"token"}`, http.StatusOK, "/api/v1/services/me/policy-bundle"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/oauth/token" {
					if tt.path == r.URL.Path {
						w.WriteHeader(tt.code)
						_, _ = w.Write([]byte(tt.body))
						return
					}
					_, _ = w.Write([]byte(`{"access_token":"token"}`))
					return
				}
				if tt.path == r.URL.Path {
					w.WriteHeader(tt.code)
					_, _ = w.Write([]byte(`{bad`))
				}
			}))
			defer server.Close()
			c := &Client{cfg: Config{AuthServerURL: server.URL, ClientID: "clientA"}, httpClient: server.Client(), bundles: map[string]*iam.PolicyBundle{}}
			if err := c.Refresh(context.Background()); err == nil {
				t.Fatal("Refresh() error = nil")
			}
		})
	}
}

func TestRefresh_RequestBuildErrors(t *testing.T) {
	c := &Client{cfg: Config{AuthServerURL: "http://[::1", ClientID: "clientA"}, httpClient: http.DefaultClient, bundles: map[string]*iam.PolicyBundle{}}
	if err := c.Refresh(context.Background()); err == nil {
		t.Fatal("Refresh() token request build error = nil")
	}
	c.token = "token"
	if err := c.Refresh(context.Background()); err == nil {
		t.Fatal("Refresh() bundle request build error = nil")
	}
}

func TestRefresh_HTTPClientErrors(t *testing.T) {
	failing := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network down")
	})}
	c := &Client{cfg: Config{AuthServerURL: "https://auth.example.com", ClientID: "clientA"}, httpClient: failing, bundles: map[string]*iam.PolicyBundle{}}
	if err := c.Refresh(context.Background()); err == nil {
		t.Fatal("Refresh() token transport error = nil")
	}

	c.token = "token"
	if err := c.Refresh(context.Background()); err == nil {
		t.Fatal("Refresh() bundle transport error = nil")
	}
}

func TestWebhookHandlerTriggersRefresh(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/oauth/token":
			_, _ = w.Write([]byte(`{"access_token":"token"}`))
		case "/api/v1/services/me/policy-bundle":
			calls++
			_, _ = w.Write([]byte(`{"data":{"service":"serviceA","version":"v1","policies":[],"generated_at":"2026-06-04T00:00:00Z"}}`))
		}
	}))
	defer server.Close()

	c := &Client{cfg: Config{AuthServerURL: server.URL, ClientID: "clientA"}, httpClient: server.Client(), bundles: map[string]*iam.PolicyBundle{}}
	w := httptest.NewRecorder()
	c.webhookHandler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"event_type":"iam.policy.updated"}`)))
	if w.Code != http.StatusNoContent || calls != 1 {
		t.Fatalf("status=%d calls=%d", w.Code, calls)
	}
}
