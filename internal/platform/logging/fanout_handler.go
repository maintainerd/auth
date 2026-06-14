package logging

import (
	"context"
	"log/slog"
)

// FanoutHandler dispatches each log record to every wrapped handler, so the
// same (already PII-redacted) record can go to multiple sinks at once — e.g.
// stdout JSON for local/container logs and an OTLP bridge for OpenTelemetry.
//
// A single level gate is applied here so all sinks observe the same filtered
// stream regardless of each handler's own level configuration.
type FanoutHandler struct {
	level    slog.Leveler
	handlers []slog.Handler
}

// NewFanoutHandler returns a handler that forwards to all of handlers, gated at
// level (nil means slog.LevelInfo).
func NewFanoutHandler(level slog.Leveler, handlers ...slog.Handler) *FanoutHandler {
	return &FanoutHandler{level: level, handlers: handlers}
}

func (h *FanoutHandler) minLevel() slog.Level {
	if h.level != nil {
		return h.level.Level()
	}
	return slog.LevelInfo
}

func (h *FanoutHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.minLevel()
}

func (h *FanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, inner := range h.handlers {
		// Clone so handlers that mutate/retain the record don't interfere.
		if err := inner.Handle(ctx, r.Clone()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (h *FanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, len(h.handlers))
	for i, inner := range h.handlers {
		next[i] = inner.WithAttrs(attrs)
	}
	return &FanoutHandler{level: h.level, handlers: next}
}

func (h *FanoutHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, len(h.handlers))
	for i, inner := range h.handlers {
		next[i] = inner.WithGroup(name)
	}
	return &FanoutHandler{level: h.level, handlers: next}
}
