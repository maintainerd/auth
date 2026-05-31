package main

import (
	"log/slog"
	"testing"
)

// TestParseSlogLevel documents the accepted LOG_LEVEL values and guards the
// safe info-level fallback used during logger setup.
func TestParseSlogLevel(t *testing.T) {
	tests := map[string]slog.Level{
		"":          slog.LevelInfo,
		"debug":     slog.LevelDebug,
		" DEBUG ":   slog.LevelDebug,
		"warn":      slog.LevelWarn,
		"warning":   slog.LevelWarn,
		"error":     slog.LevelError,
		"something": slog.LevelInfo,
	}

	for input, want := range tests {
		if got := parseSlogLevel(input); got != want {
			t.Fatalf("parseSlogLevel(%q) = %v, want %v", input, got, want)
		}
	}
}
