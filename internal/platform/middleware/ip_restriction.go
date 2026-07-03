package middleware

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type IPRestriction struct {
	Type      string
	IPAddress string
}

type TenantIPRestrictionReader interface {
	GetActiveIPRestrictions(ctx context.Context, tenantID int64) ([]IPRestriction, error)
}

const ipRestrictionCacheTTL = 5 * time.Second

func TenantIPRestrictionMiddleware(reader TenantIPRestrictionReader) func(http.Handler) http.Handler {
	cache := &ipRestrictionCache{
		ttl:     ipRestrictionCacheTTL,
		entries: map[int64]ipRestrictionCacheEntry{},
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if reader == nil {
				next.ServeHTTP(w, r)
				return
			}
			auth := AuthFromRequest(r)
			if auth == nil || auth.Tenant == nil {
				next.ServeHTTP(w, r)
				return
			}

			rules := cache.get(r.Context(), reader, auth.Tenant.TenantID)
			if len(rules) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := extractClientIP(r)
			if !ipAllowed(clientIP, rules) {
				resp.Error(w, http.StatusForbidden, "Access denied by IP restriction policy")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func ipAllowed(clientIP string, rules []IPRestriction) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return true
	}

	var hasDeny, hasAllow bool
	for _, rule := range rules {
		match := matchIP(ip, rule.IPAddress)
		switch rule.Type {
		case "deny":
			hasDeny = true
			if match {
				return false
			}
		case "allow":
			hasAllow = true
		}
	}

	if hasDeny && !hasAllow {
		return true
	}

	if hasAllow {
		for _, rule := range rules {
			if rule.Type == "allow" && matchIP(ip, rule.IPAddress) {
				return true
			}
		}
		return false
	}

	return true
}

func matchIP(clientIP net.IP, ruleAddr string) bool {
	_, cidr, err := net.ParseCIDR(ruleAddr)
	if err == nil {
		return cidr.Contains(clientIP)
	}
	ruleIP := net.ParseIP(ruleAddr)
	if ruleIP == nil {
		return false
	}
	return clientIP.Equal(ruleIP)
}

type ipRestrictionCacheEntry struct {
	rules     []IPRestriction
	expiresAt time.Time
}

type ipRestrictionCache struct {
	ttl     time.Duration
	mu      sync.Mutex
	entries map[int64]ipRestrictionCacheEntry
}

func (c *ipRestrictionCache) get(ctx context.Context, reader TenantIPRestrictionReader, tenantID int64) []IPRestriction {
	now := time.Now()
	c.mu.Lock()
	entry, ok := c.entries[tenantID]
	if ok && now.Before(entry.expiresAt) {
		c.mu.Unlock()
		return entry.rules
	}
	c.mu.Unlock()

	rules, err := reader.GetActiveIPRestrictions(ctx, tenantID)
	if err != nil {
		return nil
	}

	c.mu.Lock()
	c.entries[tenantID] = ipRestrictionCacheEntry{
		rules:     rules,
		expiresAt: now.Add(c.ttl),
	}
	c.mu.Unlock()

	return rules
}
