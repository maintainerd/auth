package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNeutralizeLogString(t *testing.T) {
	t.Run("strips CR and LF", func(t *testing.T) {
		got := neutralizeLogString("GET /login\r\nInjected: fake-line")
		if strings.ContainsAny(got, "\r\n") {
			t.Fatalf("CR/LF not stripped: %q", got)
		}
		if got != "GET /loginInjected: fake-line" {
			t.Fatalf("unexpected result: %q", got)
		}
	})

	t.Run("strips C0/C1 control characters", func(t *testing.T) {
		// The C1 control (U+009B) is built at runtime so the source file holds
		// no literal control character (staticcheck ST1018).
		in := "val\x00\x07\x1b[31m" + string(rune(0x9b))
		if got := neutralizeLogString(in); got != "val[31m" {
			t.Fatalf("control chars not stripped: %q", got)
		}
	})

	t.Run("preserves tab and printable/unicode text", func(t *testing.T) {
		in := "col1\tcol2 café 名前 🚀"
		if got := neutralizeLogString(in); got != in {
			t.Fatalf("legitimate text altered: %q", got)
		}
	})

	t.Run("returns input unchanged when already safe", func(t *testing.T) {
		in := "nothing to strip"
		if got := neutralizeLogString(in); got != in {
			t.Fatalf("safe string altered: %q", got)
		}
	})
}

// TestPIIRedactHandler_NeutralizesInjection proves the neutralisation is applied
// end-to-end by the handler to both attribute values and the record message.
func TestPIIRedactHandler_NeutralizesInjection(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewPIIRedactHandler(slog.NewJSONHandler(&buf, nil)))

	logger.Info("request\r\nFAKE line", "path", "/a\r\nlevel=ERROR msg=pwned")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("log output is not a single JSON record: %v\n%s", err, buf.String())
	}
	if msg, _ := rec["msg"].(string); strings.ContainsAny(msg, "\r\n") {
		t.Fatalf("message not neutralised: %q", msg)
	}
	if path, _ := rec["path"].(string); strings.ContainsAny(path, "\r\n") {
		t.Fatalf("attribute value not neutralised: %q", path)
	}
	// A forged newline would have produced a second JSON object; expect one line.
	if lines := bytes.Count(bytes.TrimSpace(buf.Bytes()), []byte("\n")); lines != 0 {
		t.Fatalf("expected a single log line, found %d extra newlines", lines)
	}
}
