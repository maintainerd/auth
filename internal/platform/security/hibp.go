package security

import (
	"context"
	"crypto/sha1" // #nosec G505 -- HIBP API requires SHA-1 k-anonymity hashes
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// HIBP HTTP client with reasonable timeouts.
var hibpClient = &http.Client{Timeout: 5 * time.Second}

const hibpRangeURL = "https://api.pwnedpasswords.com/range/"

// CheckHIBPPassword performs a k-anonymity HIBP v3 check. It returns true when
// the password appears in the breach database beyond the threshold count. The
// function is fail-open: network or API errors are logged and return false
// (password is treated as not breached) so the authentication flow is not
// blocked by an external dependency outage.
func CheckHIBPPassword(ctx context.Context, password []byte) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	_, span := otel.Tracer("security").Start(ctx, "security.hibp_check")
	defer span.End()

	sum := sha1.Sum(password) // #nosec G401 -- HIBP API requires SHA-1
	hexHash := strings.ToUpper(fmt.Sprintf("%x", sum[:]))
	prefix := hexHash[:5]
	suffix := hexHash[5:]

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hibpRangeURL+prefix, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "hibp request failed")
		return hibpUnavailable(span, "request build failed", err)
	}
	req.Header.Set("Add-Padding", "true")

	resp, err := hibpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "hibp request failed")
		return hibpUnavailable(span, "request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		span.SetAttributes(attribute.Int("hibp.status_code", resp.StatusCode))
		span.SetStatus(codes.Error, "hibp returned non-200")
		return hibpUnavailable(span, "non-200 response",
			fmt.Errorf("status %d", resp.StatusCode))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		span.RecordError(err)
		return hibpUnavailable(span, "response read failed", err)
	}

	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if len(line) < 36 {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.EqualFold(parts[0], suffix) {
			span.SetAttributes(attribute.String("hibp.result", "found"))
			span.SetStatus(codes.Ok, "")
			return true
		}
	}

	span.SetStatus(codes.Ok, "")
	return false
}

// hibpUnavailable records that the breach check could not run and returns "not
// breached".
//
// Failing OPEN is the deliberate choice: HIBP is a third-party dependency on the
// login and password-set hot path, and failing closed would turn their outage into
// ours. But an unobservable fail-open is indistinguishable from a working check —
// an egress firewall change or a DNS block would silently disable breach checking
// for every tenant with no signal at all. The span alone was not enough: every
// production caller reaches this through the non-context wrapper, so the span is a
// detached root nobody queries.
func hibpUnavailable(span trace.Span, reason string, err error) bool {
	span.SetAttributes(attribute.Bool("hibp.failed_open", true))
	slog.Warn("breach check unavailable — password accepted WITHOUT a HaveIBeenPwned check",
		"reason", reason, "error", err)
	return false
}
