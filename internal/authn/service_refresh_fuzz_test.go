package authn

import (
	"strings"
	"testing"
)

// FuzzLogSafe verifies the log-injection barrier: for ANY input (including values
// that originate from an unverified token), the sanitized result never contains
// CR/LF, so a crafted value cannot forge or split log lines (CWE-117).
func FuzzLogSafe(f *testing.F) {
	for _, seed := range []string{
		"", "abc123", "line1\nline2", "carriage\rreturn", "\r\n\r\n",
		"tab\tok", "unicode-π", "\x00\x1b[31m", "a\nb\rc",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if out := logSafe(s); strings.ContainsAny(out, "\r\n") {
			t.Fatalf("logSafe(%q) left CR/LF in output: %q", s, out)
		}
	})
}
