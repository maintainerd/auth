package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	inner := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(NewPIIRedactHandler(inner))
}

func loggedFields(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	return m
}

func TestPIIRedact_KnownFields(t *testing.T) {
	fields := []string{
		"email", "password", "phone", "token", "access_token",
		"id_token", "refresh_token", "api_key", "secret", "client_secret",
		"authorization", "cookie",
	}
	for _, field := range fields {
		t.Run(field, func(t *testing.T) {
			var buf bytes.Buffer
			log := newTestLogger(&buf)
			log.Info("test", field, "sensitive-value")
			m := loggedFields(t, &buf)
			assert.Equal(t, redacted, m[field], "field %q should be redacted", field)
		})
	}
}

func TestPIIRedact_CaseInsensitive(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)
	log.Info("test", "EMAIL", "foo@example.com", "Password", "hunter2")
	m := loggedFields(t, &buf)
	assert.Equal(t, redacted, m["EMAIL"])
	assert.Equal(t, redacted, m["Password"])
}

func TestPIIRedact_SafeFieldsPassThrough(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)
	log.Info("test", "user_id", "123", "request_id", "abc", "status", 200)
	m := loggedFields(t, &buf)
	assert.Equal(t, "123", m["user_id"])
	assert.Equal(t, "abc", m["request_id"])
}

func TestPIIRedact_GroupRecursion(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)
	log.Info("test", slog.Group("user", "id", "42", "email", "x@x.com", "name", "Alice"))
	m := loggedFields(t, &buf)
	user, ok := m["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "42", user["id"])
	assert.Equal(t, redacted, user["email"])
	assert.Equal(t, "Alice", user["name"])
}

func TestPIIRedact_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewPIIRedactHandler(inner)
	log := slog.New(h.WithAttrs([]slog.Attr{
		slog.String("email", "test@example.com"),
		slog.String("request_id", "xyz"),
	}))
	log.Info("msg")
	m := loggedFields(t, &buf)
	assert.Equal(t, redacted, m["email"])
	assert.Equal(t, "xyz", m["request_id"])
}

func TestPIIRedact_Enabled(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	h := NewPIIRedactHandler(inner)
	assert.False(t, h.Enabled(context.Background(), slog.LevelDebug))
	assert.False(t, h.Enabled(context.Background(), slog.LevelInfo))
	assert.True(t, h.Enabled(context.Background(), slog.LevelWarn))
}
