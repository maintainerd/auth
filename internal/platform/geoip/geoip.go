// Package geoip resolves IP addresses to a coarse, human-readable location for
// display in device/session lists. It uses a local MaxMind GeoLite2 database so
// no client IP is ever sent to a third party (a GDPR/privacy requirement for an
// IdP). When no database is configured, it degrades to a no-op resolver.
package geoip

import (
	"net"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

// Resolver maps an IP to a short "City, Country" label.
type Resolver interface {
	// Lookup returns a location label for ip and true when resolvable. It returns
	// ("", false) for private/loopback/unknown IPs or when no database is loaded.
	Lookup(ip string) (string, bool)
	// Close releases any underlying database handle.
	Close() error
}

// noopResolver is used when no GeoIP database is configured.
type noopResolver struct{}

func (noopResolver) Lookup(string) (string, bool) { return "", false }
func (noopResolver) Close() error                 { return nil }

// NewNoop returns a resolver that never resolves — the safe default.
func NewNoop() Resolver { return noopResolver{} }

type maxmindResolver struct {
	db *geoip2.Reader
}

// New opens the GeoLite2/GeoIP2 City database at dbPath. An empty path yields a
// no-op resolver (feature disabled). A non-empty but unreadable path returns a
// no-op resolver together with the open error so the caller can log it without
// failing startup.
func New(dbPath string) (Resolver, error) {
	p := strings.TrimSpace(dbPath)
	if p == "" {
		return NewNoop(), nil
	}
	db, err := geoip2.Open(p)
	if err != nil {
		return NewNoop(), err
	}
	return &maxmindResolver{db: db}, nil
}

func (r *maxmindResolver) Close() error { return r.db.Close() }

func (r *maxmindResolver) Lookup(ipStr string) (string, bool) {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	// Private/loopback/link-local IPs have no meaningful geolocation.
	if ip == nil || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return "", false
	}
	rec, err := r.db.City(ip)
	if err != nil {
		return "", false
	}
	city := rec.City.Names["en"]
	country := rec.Country.Names["en"]
	switch {
	case city != "" && country != "":
		return city + ", " + country, true
	case country != "":
		return country, true
	default:
		return "", false
	}
}
