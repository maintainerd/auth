package client

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

type RedirectURIMatch struct {
	URI  string
	Type string
}

// MatchClientRedirectURI resolves a presented redirect_uri against the client's
// registered set.
//
// Matching is EXACT per the OAuth 2.0 Security BCP §4.1.3 — no wildcards, no
// prefix or subdomain matching — with one spec-mandated exception: a loopback
// redirect ignores the port (RFC 8252 §7.3). A native app cannot reserve a port,
// so it binds an ephemeral one at runtime; requiring an exact match would make
// loopback redirects unusable and push developers toward registering a fixed
// port, which RFC 8252 §7.3 explicitly warns against.
//
// It fails closed: no registered URIs means no match.
func MatchClientRedirectURI(uris []RedirectURIMatch, candidate string) error {
	if err := security.ValidateRedirectURI(candidate); err != nil {
		return fmt.Errorf("forbidden scheme: %w", err)
	}
	if len(uris) == 0 {
		return fmt.Errorf("no redirect URIs registered for this client")
	}
	for _, uri := range uris {
		if uri.Type != shared.ClientURITypeRedirect {
			continue
		}
		if uri.URI == candidate {
			return nil
		}
		if loopbackRedirectsMatch(uri.URI, candidate) {
			return nil
		}
	}
	return fmt.Errorf("redirect_uri does not match any registered redirect URIs")
}

// loopbackRedirectsMatch implements the RFC 8252 §7.3 loopback exception: two
// http redirects to 127.0.0.1 or [::1] match when everything but the port is
// identical. Only loopback qualifies — the exception exists because the app owns
// the entire host, which is not true of any routable address.
func loopbackRedirectsMatch(registered, candidate string) bool {
	reg, err := url.Parse(registered)
	if err != nil {
		return false
	}
	cand, err := url.Parse(candidate)
	if err != nil {
		return false
	}
	// http only: the loopback interface is not reachable by a network attacker, so
	// TLS is not required there — but this must never relax https elsewhere.
	if reg.Scheme != "http" || cand.Scheme != "http" {
		return false
	}
	if !isLoopbackHost(reg.Hostname()) || !isLoopbackHost(cand.Hostname()) {
		return false
	}
	// Everything except the port must be identical, so a loopback registration
	// cannot become a wildcard for arbitrary paths.
	return reg.Hostname() == cand.Hostname() &&
		reg.EscapedPath() == cand.EscapedPath() &&
		reg.RawQuery == cand.RawQuery
}

func isLoopbackHost(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// ValidateRegisteredRedirectURI checks a redirect URI at REGISTRATION time, where
// the rules can be strict because a developer is choosing the value.
//
// security.ValidateRedirectURI is only a denylist of code-executing schemes; it
// permits values that are invalid as an OAuth redirect: relative URIs, missing
// hosts, fragments (forbidden by OIDC Core §3.1.2.1), embedded credentials, and
// plain http to a routable host.
//
// clientType decides how strict the scheme rule is:
//   - mobile: a private-use reverse-domain scheme (com.example.app:/oauth) and
//     http loopback are both legitimate per RFC 8252 §7.
//   - everything else: https, except http on loopback for local development.
func ValidateRegisteredRedirectURI(clientType, raw string) error {
	if err := security.ValidateRedirectURI(raw); err != nil {
		return err
	}

	trimmed := strings.TrimSpace(raw)
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("redirect_uri is not a valid URI")
	}
	if !parsed.IsAbs() {
		return fmt.Errorf("redirect_uri must be absolute and include a scheme")
	}
	// OIDC Core §3.1.2.1: the redirect URI must not contain a fragment; the
	// authorization response appends its own.
	if parsed.Fragment != "" || strings.Contains(trimmed, "#") {
		return fmt.Errorf("redirect_uri must not contain a fragment")
	}
	if parsed.User != nil {
		return fmt.Errorf("redirect_uri must not contain embedded credentials")
	}

	scheme := strings.ToLower(parsed.Scheme)
	switch scheme {
	case "https":
		if parsed.Host == "" {
			return fmt.Errorf("redirect_uri must include a host")
		}
		return nil
	case "http":
		// Loopback only. Plain http to a routable host would expose the
		// authorization code on the wire.
		if !isLoopbackHost(parsed.Hostname()) {
			return fmt.Errorf("redirect_uri must use https (http is only allowed for 127.0.0.1 and [::1])")
		}
		return nil
	default:
		// A private-use scheme is how a native app receives the response. Require a
		// dotted reverse-domain scheme per RFC 8252 §7.1 so it is collision
		// resistant, and reject it for client types that run in a browser.
		if clientType != shared.ClientTypeMobile {
			return fmt.Errorf("redirect_uri scheme %q is only allowed for mobile clients", scheme)
		}
		if !strings.Contains(scheme, ".") {
			return fmt.Errorf("a custom redirect_uri scheme must be a reverse-domain name (e.g. com.example.app)")
		}
		return nil
	}
}
