package logging

import "strings"

// neutralizeLogString removes characters that let untrusted input forge or
// break log records (CWE-117, Log Injection / Log Forging). Carriage returns
// and line feeds are stripped so a value can never introduce a fake log line,
// and other ASCII control characters are dropped so terminal escape sequences
// cannot be smuggled through log viewers.
//
// It is applied centrally by PIIRedactHandler to every string attribute value
// and to the record message, so no individual call site has to remember to
// sanitise user-provided data before logging it.
func neutralizeLogString(s string) string {
	if !strings.ContainsFunc(s, isUnsafeLogRune) {
		return s
	}
	return strings.Map(func(r rune) rune {
		if isUnsafeLogRune(r) {
			return -1
		}
		return r
	}, s)
}

// isUnsafeLogRune reports whether r must not appear verbatim in a log record.
// It covers CR, LF, and the C0/C1 control ranges while preserving tab (\t),
// which is legitimate whitespace, and all printable/Unicode text.
func isUnsafeLogRune(r rune) bool {
	if r == '\t' {
		return false
	}
	return r < 0x20 || (r >= 0x7f && r <= 0x9f)
}
