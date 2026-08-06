package shared

import (
	"net"
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// SameRegistrableDomain reports whether two hosts belong to the same
// registrable domain (eTLD+1) — the unit a browser uses to decide whether a
// cookie may be shared.
//
// It answers "would the browser treat these as the same site?", which is why
// comparison is on eTLD+1 and not on an exact host match: console.example.com
// and api.example.com genuinely are the same site and genuinely do share
// cookies, while example.com.evil.net does not, despite the prefix.
//
// Both arguments may be bare hosts or full URLs; scheme, port, path, userinfo,
// case and a trailing dot are all normalized away before comparison, because
// none of them affect which site a cookie belongs to and every one of them has
// been a source of comparison bugs.
//
// It is deliberately strict about what it will call "the same":
//
//   - An empty or unparseable host on either side is never a match. This
//     function gates a privilege boundary, so "I could not tell" must mean no.
//   - An IP address is compared literally. Public-suffix logic is meaningless
//     for an IP, and treating 10.0.0.1 and 10.0.0.2 as one site because they
//     share "10.0.0" would be badly wrong.
//   - A host that IS a public suffix (a bare "com", or "local") has no
//     registrable domain and matches nothing, so a misconfiguration cannot
//     accidentally make every .com client first-party.
//
// publicsuffix.EffectiveTLDPlusOne handles the multi-label suffixes an
// eyeballed "last two labels" rule gets wrong — co.uk, com.au, github.io —
// where that shortcut would make two unrelated tenants on foo.co.uk and
// bar.co.uk look like one site.
func SameRegistrableDomain(a string, b string) bool {
	hostA := registrableDomain(a)
	if hostA == "" {
		return false
	}
	return hostA == registrableDomain(b)
}

// registrableDomain normalizes a host or URL to its eTLD+1, or "" when it has
// none or cannot be determined.
func registrableDomain(value string) string {
	host := normalizeSiteHost(value)
	if host == "" {
		return ""
	}

	// An IP has no registrable domain; compare it as itself.
	if ip := net.ParseIP(host); ip != nil {
		return host
	}

	// A host that IS an ICANN-managed public suffix ("com", "co.uk") has no
	// registrable domain. Returning it would make every client under that suffix
	// share a "site", so it matches nothing.
	if suffix, icann := publicsuffix.PublicSuffix(host); icann && suffix == host {
		return ""
	}

	// A single-label host that is not an ICANN suffix — "localhost", a Docker
	// service name, a bare ".local" label — has no registrable domain either, but
	// it is a legitimate same-site value in development, so compare it literally.
	// Two such hosts match only when they are the same string, which is the same
	// answer an exact comparison would give.
	if !strings.Contains(host, ".") {
		return host
	}

	etldPlusOne, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		// The host is itself a public suffix, or is otherwise not a registrable
		// name. Either way there is no site to share, so it matches nothing.
		return ""
	}
	return etldPlusOne
}

// normalizeSiteHost extracts a comparable hostname from a bare host or a full
// URL.
//
// Distinct from this package's normalizeHost, which is lenient by design (it
// salvages a host from a configured base URL). This one gates a privilege
// boundary, so a value it cannot read cleanly is rejected rather than trimmed
// into something that compares equal to a host the caller never named.
func normalizeSiteHost(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return ""
	}

	// url.Parse only populates Host when a scheme is present; a bare
	// "console.example.com" parses as a path. Both shapes are stored in
	// clients.domain, so both have to work.
	if strings.Contains(v, "://") {
		parsed, err := url.Parse(v)
		if err != nil || parsed.Host == "" {
			return ""
		}
		v = parsed.Host
	}

	// Strip a port. SplitHostPort errors when there is none, which is fine.
	if host, _, err := net.SplitHostPort(v); err == nil {
		v = host
	}

	// A trailing dot denotes the DNS root and does not change the site.
	v = strings.TrimSuffix(v, ".")

	// Hostnames are case-insensitive; cookies are matched case-insensitively.
	v = strings.ToLower(v)

	// Reject anything still carrying URL structure: a value like
	// "example.com/path" or "user@example.com" is not a host, and silently
	// keeping the prefix would compare the wrong string.
	if strings.ContainsAny(v, "/@?#") {
		return ""
	}
	return v
}
