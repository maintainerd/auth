package webhook

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// SafeDeliveryClient is the shared HTTP client for outbound webhook delivery,
// hardened against SSRF via DNS-rebinding.
//
// The problem it solves: validating a URL's resolved IP and then letting the
// HTTP client re-resolve and dial independently leaves a TOCTOU window — an
// attacker's DNS can answer with a public IP at validation time and a private /
// metadata IP at dial time. This client closes that window by resolving the host
// exactly ONCE inside DialContext, validating every resolved IP against the
// blocklist, and dialing the pinned IP literal — so the address validated is the
// address connected to. TLS still verifies against the original hostname (the
// Transport sets ServerName from the request host, not the dialed IP), so
// certificate validation is unaffected. Redirects reuse this transport, so every
// hop is pinned and validated; CheckRedirect additionally re-enforces https.
var SafeDeliveryClient = &http.Client{
	Transport: &http.Transport{
		DialContext:           pinnedSafeDialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
	CheckRedirect: func(r *http.Request, _ []*http.Request) error {
		return ValidateDeliveryURL(r.Context(), r.URL.String())
	},
}

var safeDialer = &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}

// pinnedSafeDialContext resolves the destination host once, rejects if ANY
// resolved address is unsafe (so an attacker cannot smuggle a private IP in a
// multi-answer response), then dials the pinned safe IP literal.
func pinnedSafeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	if literal := net.ParseIP(host); literal != nil {
		ips = []net.IP{literal}
	} else {
		resolved, rerr := net.DefaultResolver.LookupIPAddr(ctx, host)
		if rerr != nil {
			return nil, fmt.Errorf("resolve %s: %w", host, rerr)
		}
		for _, a := range resolved {
			ips = append(ips, a.IP)
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses for host %q", host)
	}
	for _, ip := range ips {
		if isUnsafeWebhookIP(ip) {
			return nil, fmt.Errorf("destination resolves to a disallowed address")
		}
	}
	// All resolved addresses passed; pin the first and dial the IP literal so no
	// second resolution can occur between validation and connect.
	return safeDialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
}
