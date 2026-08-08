package logging

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
)

// piiFields is the set of attribute keys whose values are always redacted.
// All entries are lower-cased; matching is case-insensitive.
var piiFields = map[string]bool{
	"email":           true,
	"email_address":   true,
	"password":        true,
	"password_hash":   true,
	"hashed_password": true,
	"phone":           true,
	"phone_number":    true,
	"mobile":          true,
	"token":           true,
	"access_token":    true,
	"id_token":        true,
	"refresh_token":   true,
	"api_key":         true,
	"api_secret":      true,
	"secret":          true,
	"client_secret":   true,
	"authorization":   true,
	"cookie":          true,
	"ssn":             true,
	"credit_card":     true,
	"card_number":     true,
	"cvv":             true,
}

const redacted = "[REDACTED]"

var redactStringPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`),
	regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+\b`),
	regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`),
}

// PIIRedactHandler wraps a slog.Handler and replaces values of known PII
// attribute keys with [REDACTED] before forwarding to the inner handler.
type PIIRedactHandler struct {
	inner slog.Handler
}

// NewPIIRedactHandler wraps inner with PII redaction.
func NewPIIRedactHandler(inner slog.Handler) *PIIRedactHandler {
	return &PIIRedactHandler{inner: inner}
}

func (h *PIIRedactHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *PIIRedactHandler) Handle(ctx context.Context, r slog.Record) error {
	sanitised := slog.NewRecord(r.Time, r.Level, neutralizeLogString(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		sanitised.AddAttrs(redactAttr(a))
		return true
	})
	return h.inner.Handle(ctx, sanitised)
}

func (h *PIIRedactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	sanitised := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		sanitised[i] = redactAttr(a)
	}
	return &PIIRedactHandler{inner: h.inner.WithAttrs(sanitised)}
}

func (h *PIIRedactHandler) WithGroup(name string) slog.Handler {
	return &PIIRedactHandler{inner: h.inner.WithGroup(name)}
}

func redactAttr(a slog.Attr) slog.Attr {
	if isPIIKey(a.Key) {
		return slog.String(a.Key, redacted)
	}
	if a.Value.Kind() == slog.KindGroup {
		sub := a.Value.Group()
		cleaned := make([]any, 0, len(sub))
		for _, sa := range sub {
			cleaned = append(cleaned, redactAttr(sa))
		}
		return slog.Group(a.Key, cleaned...)
	}
	// Neutralise CR/LF and control characters in string values so untrusted
	// input cannot forge log lines (CWE-117). Non-string values are unaffected.
	if a.Value.Kind() == slog.KindString {
		s := a.Value.String()
		if cleaned := neutralizeLogString(s); cleaned != s {
			return slog.String(a.Key, cleaned)
		}
	}
	return a
}

func isPIIKey(key string) bool {
	return piiFields[strings.ToLower(key)]
}

// RedactJSON walks a JSON object and replaces values whose keys match the PII
// field set with [REDACTED]. Non-object JSON (arrays, primitives, malformed
// input) is returned unchanged so callers never get an error path.
func RedactJSON(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return raw
	}
	redactMap(obj)
	out, err := json.Marshal(obj)
	if err != nil {
		return raw
	}
	return out
}

func redactMap(m map[string]any) {
	for k, v := range m {
		if isPIIKey(k) {
			m[k] = redacted
			continue
		}
		switch nested := v.(type) {
		case map[string]any:
			redactMap(nested)
		case []any:
			for _, elem := range nested {
				if nestedMap, ok := elem.(map[string]any); ok {
					redactMap(nestedMap)
				}
			}
		}
	}
}

func RedactString(s *string) *string {
	if s == nil || *s == "" {
		return s
	}
	result := *s
	for _, pattern := range redactStringPatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			if strings.HasPrefix(strings.ToLower(match), "bearer ") {
				return "Bearer " + redacted
			}
			return redacted
		})
	}
	if result != *s {
		return &result
	}
	return s
}
