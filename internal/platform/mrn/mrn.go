// Package mrn parses and matches maintainerd resource names (MRNs), the
// platform-wide resource identifier evaluated by the IAM policy engine:
//
//	mrn:<service>:<tenant>:<project>:<resource-path>
//
// Exactly five colon-separated head parts. The resource-path is the fifth part
// and may itself contain colons and slashes — parsing splits with a limit of 5
// so those are never re-split. Empty tenant and project segments are
// meaningful, not missing: an empty project means tenant-scoped, an empty
// tenant means platform-scoped.
//
// Matching is deliberately SEGMENT-AWARE rather than a flat glob. A flat glob
// lets a wildcard run across the colon boundaries, so "mrn:storage:acme:*"
// would match "mrn:storage:acmecorp:x:y" — a grant written for tenant "acme"
// silently reaching into tenant "acmecorp". Here a wildcard is confined to the
// single segment it was written in, which is what makes an MRN pattern safe to
// use as a tenant-isolation boundary.
package mrn

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Prefix is the scheme prefix every MRN starts with. Any string carrying it is
// committed to the MRN grammar: it either parses or is rejected, never quietly
// treated as a legacy flat resource string.
const Prefix = "mrn:"

// MRN is a parsed maintainerd resource name identifying one concrete resource.
// It carries no wildcards; patterns are the separate Pattern type so a resource
// can never be accidentally interpreted as a grant over other resources.
type MRN struct {
	Service      string // owning service, e.g. "storage" — required
	Tenant       string // tenant slug — empty means platform-scoped
	Project      string // project slug — empty means tenant-scoped
	ResourcePath string // service-defined path, e.g. "bucket/invoices/object/2026/q3.pdf"
}

// Pattern is a parsed MRN pattern as written in a policy statement resource.
// Service/Tenant/Project are a literal, "*", or "" (tenant/project only);
// ResourcePath is a literal, a prefix ending in "*", or a bare "*".
type Pattern struct {
	Service      string
	Tenant       string
	Project      string
	ResourcePath string
}

// IsMRN reports whether s claims the MRN scheme. It is a cheap prefix check
// only — a true result does NOT mean s is valid, it means s must be held to
// the MRN grammar (Parse/ParsePattern) instead of legacy flat-string matching.
func IsMRN(s string) bool {
	return strings.HasPrefix(s, Prefix)
}

// Parse parses s as a concrete resource MRN.
//
// Validation is strict and fail-closed on purpose: this is the identifier
// authorization decisions are keyed on, so anything ambiguous (wrong part
// count, wildcard characters, unprintable bytes) is rejected rather than
// normalized. A permissive parser here would let two visually different
// strings authorize as the same resource.
func Parse(s string) (MRN, error) {
	parts, err := splitMRN(s)
	if err != nil {
		return MRN{}, err
	}
	m := MRN{Service: parts[1], Tenant: parts[2], Project: parts[3], ResourcePath: parts[4]}
	if m.Service == "" {
		return MRN{}, fmt.Errorf("mrn %q: service segment is required", s)
	}
	if !validSegment(m.Service) {
		return MRN{}, fmt.Errorf("mrn %q: service segment must contain only lowercase letters, digits, and hyphens", s)
	}
	if !validSegment(m.Tenant) {
		return MRN{}, fmt.Errorf("mrn %q: tenant segment must contain only lowercase letters, digits, and hyphens", s)
	}
	if !validSegment(m.Project) {
		return MRN{}, fmt.Errorf("mrn %q: project segment must contain only lowercase letters, digits, and hyphens", s)
	}
	if err := validResourcePath(m.ResourcePath); err != nil {
		return MRN{}, fmt.Errorf("mrn %q: %w", s, err)
	}
	return m, nil
}

// String renders the MRN back to its canonical form. For any m produced by
// Parse, Parse(m.String()) round-trips to an identical value.
func (m MRN) String() string {
	return Prefix + m.Service + ":" + m.Tenant + ":" + m.Project + ":" + m.ResourcePath
}

// ParsePattern parses s as an MRN pattern from a policy statement.
//
// Wildcard placement is restricted to shapes whose meaning is unambiguous:
// a segment is matched by "*" or by exact equality, and the resource-path glob
// is a literal, a trailing-"*" prefix, or a bare "*". Mid-path wildcards
// ("bucket/*/object") are rejected here — at write/validation time — rather
// than accepted and silently mis-matched at evaluation time, where the miss
// would be invisible until it either denied legitimate access or granted more
// than the author intended.
func ParsePattern(s string) (Pattern, error) {
	parts, err := splitMRN(s)
	if err != nil {
		return Pattern{}, err
	}
	p := Pattern{Service: parts[1], Tenant: parts[2], Project: parts[3], ResourcePath: parts[4]}
	if p.Service == "" {
		// A concrete MRN can never have an empty service, so an empty service
		// pattern segment could match nothing — it is always an authoring mistake.
		return Pattern{}, fmt.Errorf("mrn pattern %q: service segment is required (a literal or *)", s)
	}
	if p.Service != "*" && !validSegment(p.Service) {
		return Pattern{}, fmt.Errorf("mrn pattern %q: service segment must be a literal (lowercase letters, digits, hyphens) or *", s)
	}
	if p.Tenant != "*" && !validSegment(p.Tenant) {
		return Pattern{}, fmt.Errorf("mrn pattern %q: tenant segment must be a literal (lowercase letters, digits, hyphens), *, or empty", s)
	}
	if p.Project != "*" && !validSegment(p.Project) {
		return Pattern{}, fmt.Errorf("mrn pattern %q: project segment must be a literal (lowercase letters, digits, hyphens), *, or empty", s)
	}
	if err := validPathPattern(p.ResourcePath); err != nil {
		return Pattern{}, fmt.Errorf("mrn pattern %q: %w", s, err)
	}
	return p, nil
}

// MatchPattern reports whether the MRN pattern matches the concrete MRN
// resource. It returns an error — never a silent false — when either side is
// not valid for its role, so a caller can distinguish "does not match" from
// "cannot be evaluated" and fail closed on the latter.
//
// Segment semantics:
//   - "*" matches anything, INCLUDING an empty segment.
//   - a literal matches only itself.
//   - an EMPTY pattern segment matches only an EMPTY resource segment. Scope
//     is a boundary, not a wildcard: a platform-scoped pattern
//     (mrn:core:::agent/*) speaks only for platform-scoped resources and must
//     never leak into some tenant's identically-named resources — that would
//     turn "narrower scope" into "broader grant".
//
// A wildcard never spans a colon boundary; that is the entire point versus a
// naive glob (see the package comment).
func MatchPattern(pattern, resource string) (bool, error) {
	p, err := ParsePattern(pattern)
	if err != nil {
		return false, err
	}
	r, err := Parse(resource)
	if err != nil {
		return false, err
	}
	if !segmentMatches(p.Service, r.Service) {
		return false, nil
	}
	if !segmentMatches(p.Tenant, r.Tenant) {
		return false, nil
	}
	if !segmentMatches(p.Project, r.Project) {
		return false, nil
	}
	return pathMatches(p.ResourcePath, r.ResourcePath), nil
}

// splitMRN enforces the shared head shape: the "mrn:" prefix and exactly five
// parts, split with limit 5 so colons inside the resource-path are preserved.
func splitMRN(s string) ([]string, error) {
	if !strings.HasPrefix(s, Prefix) {
		return nil, fmt.Errorf("%q does not start with %q", s, Prefix)
	}
	parts := strings.SplitN(s, ":", 5)
	if len(parts) != 5 {
		return nil, fmt.Errorf("%q must have exactly 5 colon-separated parts: mrn:<service>:<tenant>:<project>:<resource-path>", s)
	}
	return parts, nil
}

// validSegment reports whether s is a valid (possibly empty) name segment:
// lowercase letters, digits, and hyphens only. The charset is deliberately
// narrow — no uppercase, no dots, no percent-escapes — so there is exactly one
// spelling of every segment and no case-folding or normalization ambiguity for
// an attacker to exploit.
func validSegment(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			continue
		}
		return false
	}
	return true
}

// validResourcePath validates the fifth part of a concrete MRN: required,
// printable, valid UTF-8, no leading "/" (a leading slash would create two
// spellings — "bucket/x" and "/bucket/x" — of what a human reads as the same
// resource, and prefix-based grants would silently cover only one of them),
// and no "*" at all — a concrete resource that LOOKS like a pattern is
// rejected rather than matched literally, so a crafted resource string can
// never masquerade as, or be confused with, a grant.
func validResourcePath(path string) error {
	if path == "" {
		return errors.New("resource-path is required")
	}
	if strings.ContainsRune(path, '*') {
		return errors.New("resource-path of a concrete resource must not contain *")
	}
	if strings.HasPrefix(path, "/") {
		return errors.New("resource-path must not start with /")
	}
	if !utf8.ValidString(path) {
		return errors.New("resource-path must be valid UTF-8")
	}
	for _, r := range path {
		if !unicode.IsPrint(r) {
			return errors.New("resource-path must contain only printable characters")
		}
	}
	return nil
}

// validPathPattern validates a pattern's resource-path glob: the v1 wildcard
// restriction — "*" may only be the whole path or its single final character —
// plus everything validResourcePath requires of the literal part.
func validPathPattern(path string) error {
	if path == "*" {
		return nil
	}
	literal := path
	if strings.ContainsRune(path, '*') {
		if strings.Count(path, "*") > 1 || !strings.HasSuffix(path, "*") {
			return errors.New("resource-path may use * only as a bare * or a single trailing wildcard (mid-path wildcards are not supported)")
		}
		literal = strings.TrimSuffix(path, "*")
	}
	return validResourcePath(literal)
}

// segmentMatches applies the segment rules documented on MatchPattern.
func segmentMatches(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	// Covers literals AND the empty-pattern-matches-only-empty scope boundary.
	return pattern == value
}

// pathMatches applies the resource-path glob: bare "*", trailing-"*" prefix,
// or exact literal. ParsePattern has already rejected every other shape.
func pathMatches(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(value, pattern[:len(pattern)-1])
	}
	return pattern == value
}
