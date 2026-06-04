package authzclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/maintainerd/auth/internal/iam"
)

type Config struct {
	AuthServerURL string
	ClientID      string
	ClientSecret  string
	PollInterval  time.Duration
	WebhookListen string
	HTTPClient    *http.Client
}

type Client struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.RWMutex
	etag       string
	token      string
	bundle     *iam.PolicyBundle
	bundles    map[string]*iam.PolicyBundle
	stop       chan struct{}
	done       chan struct{}
}

func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.AuthServerURL) == "" {
		return nil, errors.New("auth server url is required")
	}
	if strings.TrimSpace(cfg.ClientID) == "" {
		return nil, errors.New("client id is required")
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 30 * time.Second
	}
	c := &Client{cfg: cfg, httpClient: httpClient, bundles: map[string]*iam.PolicyBundle{}, stop: make(chan struct{}), done: make(chan struct{})}
	if err := c.Refresh(context.Background()); err != nil {
		return nil, err
	}
	go c.poll()
	if cfg.WebhookListen != "" {
		go c.listenWebhook()
	}
	return c, nil
}

func (c *Client) Close() {
	close(c.stop)
	<-c.done
}

func (c *Client) Can(action, resource string) bool {
	c.mu.RLock()
	bundle := c.bundle
	c.mu.RUnlock()
	if bundle == nil {
		return false
	}
	return iam.Evaluate(bundle.Policies, iam.AuthzRequest{Principal: bundle.Service, Action: action, Resource: resource}).Allowed
}

func (c *Client) CanPrincipal(principal, action, resource string) bool {
	c.mu.RLock()
	bundle := c.bundles[principal]
	c.mu.RUnlock()
	if bundle == nil {
		return false
	}
	return iam.Evaluate(bundle.Policies, iam.AuthzRequest{Principal: principal, Action: action, Resource: resource}).Allowed
}

func (c *Client) Refresh(ctx context.Context) error {
	if err := c.ensureToken(ctx); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.cfg.AuthServerURL, "/")+"/api/v1/services/me/policy-bundle", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	c.mu.RLock()
	etag := c.etag
	c.mu.RUnlock()
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode == http.StatusNotModified {
		return nil
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("policy bundle fetch failed")
	}
	var envelope struct {
		Data iam.PolicyBundle `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return err
	}
	c.mu.Lock()
	c.etag = res.Header.Get("ETag")
	c.bundle = &envelope.Data
	c.bundles[envelope.Data.Service] = &envelope.Data
	c.mu.Unlock()
	return nil
}

func (c *Client) ensureToken(ctx context.Context) error {
	c.mu.RLock()
	hasToken := c.token != ""
	c.mu.RUnlock()
	if hasToken {
		return nil
	}
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.cfg.AuthServerURL, "/")+"/api/v1/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return errors.New("token request failed")
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&token); err != nil {
		return err
	}
	if token.AccessToken == "" {
		return errors.New("token response missing access_token")
	}
	c.mu.Lock()
	c.token = token.AccessToken
	c.mu.Unlock()
	return nil
}

func (c *Client) poll() {
	defer close(c.done)
	ticker := time.NewTicker(c.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = c.Refresh(context.Background())
		case <-c.stop:
			return
		}
	}
}

func (c *Client) listenWebhook() {
	_ = http.ListenAndServe(c.cfg.WebhookListen, c.webhookHandler())
}

func (c *Client) webhookHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = c.Refresh(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
}

func DecodeBundle(body []byte) (*iam.PolicyBundle, error) {
	var bundle iam.PolicyBundle
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&bundle); err != nil {
		return nil, err
	}
	return &bundle, nil
}
