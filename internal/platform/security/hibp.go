package security

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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

	sum := sha1.Sum(password)
	hexHash := strings.ToUpper(fmt.Sprintf("%x", sum[:]))
	prefix := hexHash[:5]
	suffix := hexHash[5:]

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hibpRangeURL+prefix, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "hibp request failed")
		return false
	}
	req.Header.Set("Add-Padding", "true")

	resp, err := hibpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "hibp request failed")
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		span.SetAttributes(attribute.Int("hibp.status_code", resp.StatusCode))
		span.SetStatus(codes.Error, "hibp returned non-200")
		return false
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		span.RecordError(err)
		return false
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
