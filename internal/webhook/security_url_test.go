package webhook

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWebhookURL(t *testing.T) {
	t.Run("success public ip with resolve", func(t *testing.T) {
		require.NoError(t, validateWebhookURL(context.Background(), "https://93.184.216.34/hook", true))
	})

	t.Run("success hostname without resolve", func(t *testing.T) {
		require.NoError(t, validateWebhookURL(context.Background(), "https://example.com/hook", false))
	})

	t.Run("success hostname with resolve", func(t *testing.T) {
		require.NoError(t, validateWebhookURL(context.Background(), "https://example.com/hook", true))
	})

	t.Run("invalid url", func(t *testing.T) {
		require.ErrorContains(t, validateWebhookURL(context.Background(), "://bad", false), "valid")
	})

	t.Run("missing host", func(t *testing.T) {
		require.ErrorContains(t, validateWebhookURL(context.Background(), "https:///hook", false), "valid")
	})

	t.Run("empty parsed hostname", func(t *testing.T) {
		require.ErrorContains(t, validateWebhookURL(context.Background(), "https://:443/hook", false), "host is required")
	})

	t.Run("requires https", func(t *testing.T) {
		require.ErrorContains(t, validateWebhookURL(context.Background(), "http://example.com/hook", false), "https")
	})

	t.Run("rejects unsafe ip host", func(t *testing.T) {
		require.ErrorContains(t, validateWebhookURL(context.Background(), "https://127.0.0.1/hook", true), "not allowed")
	})

	t.Run("resolve error", func(t *testing.T) {
		err := validateWebhookURL(context.Background(), "https://nonexistent.invalid/hook", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resolve webhook URL host")
	})

	t.Run("resolved private host rejected", func(t *testing.T) {
		err := validateWebhookURL(context.Background(), "https://localhost/hook", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "private address")
	})
}

func TestIsUnsafeWebhookIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "loopback", ip: "127.0.0.1", want: true},
		{name: "private", ip: "10.0.0.1", want: true},
		{name: "link local unicast", ip: "169.254.1.1", want: true},
		{name: "link local multicast", ip: "224.0.0.1", want: true},
		{name: "unspecified", ip: "0.0.0.0", want: true},
		{name: "multicast", ip: "ff02::1", want: true},
		{name: "public", ip: "93.184.216.34", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isUnsafeWebhookIP(net.ParseIP(tt.ip)))
		})
	}
}
