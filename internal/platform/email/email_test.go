package email

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startMockSMTP(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		_, _ = fmt.Fprintf(conn, "220 mock SMTP\r\n")
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			cmd := strings.ToUpper(line)
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				_, _ = fmt.Fprintf(conn, "250-mock Hello\r\n250 OK\r\n")
			case strings.HasPrefix(cmd, "MAIL"):
				_, _ = fmt.Fprintf(conn, "250 OK\r\n")
			case strings.HasPrefix(cmd, "RCPT"):
				_, _ = fmt.Fprintf(conn, "250 OK\r\n")
			case strings.HasPrefix(cmd, "DATA"):
				_, _ = fmt.Fprintf(conn, "354 Go ahead\r\n")
				for scanner.Scan() {
					if scanner.Text() == "." {
						break
					}
				}
				_, _ = fmt.Fprintf(conn, "250 OK\r\n")
			case strings.HasPrefix(cmd, "QUIT"):
				_, _ = fmt.Fprintf(conn, "221 Bye\r\n")
				return
			default:
				_, _ = fmt.Fprintf(conn, "250 OK\r\n")
			}
		}
	}()

	return port
}

func TestSMTPProvider_Send_Success(t *testing.T) {
	port := startMockSMTP(t)
	cfg := ProviderConfig{Provider: "smtp", Host: "127.0.0.1", Port: port}
	p, err := NewProvider(context.Background(), cfg)
	require.NoError(t, err)

	err = p.Send(context.Background(), SendParams{
		To: "user@example.com", From: "noreply@example.com", Subject: "Hello", BodyHTML: "<p>Hello</p>",
	})
	assert.NoError(t, err)
}

func TestSMTPProvider_Send_WithCustomFrom(t *testing.T) {
	port := startMockSMTP(t)
	cfg := ProviderConfig{Provider: "smtp", Host: "127.0.0.1", Port: port}
	p, err := NewProvider(context.Background(), cfg)
	require.NoError(t, err)

	err = p.Send(context.Background(), SendParams{
		To: "user@example.com", From: "custom@example.com", Subject: "Hello", BodyHTML: "<p>Hello</p>",
	})
	assert.NoError(t, err)
}

func TestSMTPProvider_Send_PlainText(t *testing.T) {
	port := startMockSMTP(t)
	cfg := ProviderConfig{Provider: "smtp", Host: "127.0.0.1", Port: port}
	p, err := NewProvider(context.Background(), cfg)
	require.NoError(t, err)

	err = p.Send(context.Background(), SendParams{
		To: "user@example.com", From: "noreply@example.com", Subject: "Plain + HTML", BodyHTML: "<p>Hello</p>", BodyPlain: "Hello",
	})
	assert.NoError(t, err)
}

func TestSMTPProvider_Send_Unreachable(t *testing.T) {
	cfg := ProviderConfig{Provider: "smtp", Host: "127.0.0.1", Port: 1}
	p, err := NewProvider(context.Background(), cfg)
	require.NoError(t, err)

	err = p.Send(context.Background(), SendParams{
		To: "user@example.com", Subject: "Hello", BodyHTML: "<p>Hello</p>",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "smtp:")
}

func TestSMTPProvider_Send_BadHost(t *testing.T) {
	cfg := ProviderConfig{Provider: "smtp", Host: "this-host-does-not-exist.invalid", Port: 587}
	p, err := NewProvider(context.Background(), cfg)
	require.NoError(t, err)

	err = p.Send(context.Background(), SendParams{
		To: "user@example.com", Subject: "Bad host", BodyHTML: "<p>body</p>",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "smtp:")
}

func TestNewProvider_SMTP(t *testing.T) {
	cfg := ProviderConfig{Provider: "smtp", Host: "smtp.example.com", Port: 587, Username: "user", Password: "pass"}
	p, err := NewProvider(context.Background(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestNewProvider_EmptyProviderDefaultsToSMTP(t *testing.T) {
	cfg := ProviderConfig{Provider: ""}
	p, err := NewProvider(context.Background(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestNewProvider_UnknownProvider(t *testing.T) {
	cfg := ProviderConfig{Provider: "unknown"}
	_, err := NewProvider(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported provider")
}

// The SaaS API providers (SES/SendGrid/Mailgun/Postmark/Resend) were removed:
// maintainerd delivers over SMTP only, so every provider is reached via its SMTP
// relay. NewProvider now rejects any non-smtp provider.
func TestNewProvider_SaaSProvidersRejected(t *testing.T) {
	for _, provider := range []string{"ses", "sendgrid", "mailgun", "postmark", "resend"} {
		_, err := NewProvider(context.Background(), ProviderConfig{Provider: provider})
		require.Error(t, err, provider)
		assert.Contains(t, err.Error(), "unsupported provider", provider)
	}
}
