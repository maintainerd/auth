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

func TestRedactString_DoesNotRedactFreeTextKeyword(t *testing.T) {
	input := "user updated email preferences"
	got := RedactString(&input)
	require.NotNil(t, got)
	assert.Equal(t, input, *got)
}

func TestRedactString_RedactsEmailAndBearerValues(t *testing.T) {
	input := "failed login for jane@example.com with Bearer abc.def.ghi"
	got := RedactString(&input)
	require.NotNil(t, got)
	assert.Equal(t, "failed login for [REDACTED] with Bearer [REDACTED]", *got)
}

func TestRedactString_NoChangeReturnsSamePointer(t *testing.T) {
	input := "no sensitive data here"
	got := RedactString(&input)
	assert.Same(t, &input, got)
}

func TestRedactString_NilPointer(t *testing.T) {
	got := RedactString(nil)
	assert.Nil(t, got)
}

func TestRedactString_EmptyString(t *testing.T) {
	input := ""
	got := RedactString(&input)
	assert.Same(t, &input, got)
}

func TestRedactJSON_Empty(t *testing.T) {
	assert.Empty(t, RedactJSON(nil))
	assert.Empty(t, RedactJSON([]byte{}))
}

func TestRedactJSON_NonJSON(t *testing.T) {
	raw := []byte("not-json")
	assert.Equal(t, raw, RedactJSON(raw))
}

func TestRedactJSON_RedactsPII(t *testing.T) {
	raw := []byte(`{"email":"test@x.com","name":"Alice","token":"s3cret"}`)
	got := RedactJSON(raw)
	var m map[string]any
	require.NoError(t, json.Unmarshal(got, &m))
	assert.Equal(t, redacted, m["email"])
	assert.Equal(t, "Alice", m["name"])
	assert.Equal(t, redacted, m["token"])
}

func TestRedactJSON_NestedMap(t *testing.T) {
	raw := []byte(`{"user":{"email":"nested@x.com","id":"42"}}`)
	got := RedactJSON(raw)
	var m map[string]any
	require.NoError(t, json.Unmarshal(got, &m))
	user := m["user"].(map[string]any)
	assert.Equal(t, redacted, user["email"])
	assert.Equal(t, "42", user["id"])
}

func TestRedactJSON_ArrayOfMaps(t *testing.T) {
	raw := []byte(`{"items":[{"email":"a@b.com"},{"id":"x","password":"pw"}]}`)
	got := RedactJSON(raw)
	var m map[string]any
	require.NoError(t, json.Unmarshal(got, &m))
	items := m["items"].([]any)
	assert.Equal(t, redacted, items[0].(map[string]any)["email"])
	assert.Equal(t, redacted, items[1].(map[string]any)["password"])
	assert.Equal(t, "x", items[1].(map[string]any)["id"])
}

func TestPIIRedact_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewPIIRedactHandler(inner)
	grouped := h.WithGroup("db").WithAttrs([]slog.Attr{slog.String("password", "s3cret")})
	log := slog.New(grouped)
	log.Info("ns")
	m := loggedFields(t, &buf)
	db, ok := m["db"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, redacted, db["password"])
}

func TestPIIRedact_HandleEmptyAttrs(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)
	log.Info("no-attrs")
	m := loggedFields(t, &buf)
	assert.Equal(t, "no-attrs", m["msg"])
}

func TestRedactString_JWTPatternMatched(t *testing.T) {
	input := "token=eyJhbGciOiJSUzI1NiIs.eyJzdWIiOiIxMjM0NTY3ODkw.IiwibmFtZSI6IkpvaG4_"
	got := RedactString(&input)
	assert.Equal(t, "token=[REDACTED]", *got)
}

func TestIsPIIKey(t *testing.T) {
	assert.True(t, isPIIKey("email"))
	assert.True(t, isPIIKey("EMAIL"))
	assert.True(t, isPIIKey("password"))
	assert.False(t, isPIIKey("user_id"))
	assert.False(t, isPIIKey("request_id"))
}
