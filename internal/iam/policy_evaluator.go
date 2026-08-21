package iam

import (
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/platform/mrn"
)

// Decision is the PDP verdict for a policy evaluation request.
type Decision struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
}

// AuthzRequest is the normalized authorization question evaluated by the PDP.
type AuthzRequest struct {
	Principal string `json:"principal"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	TenantID  int64  `json:"tenant_id,omitempty"`
}

const (
	policyVersionV1            = "v1"
	decisionExplicitDeny       = "explicit deny"
	decisionUnsupportedVersion = "policy document version is not supported"
	decisionUnknownEffect      = "policy statement has an unrecognised effect"
	decisionMatchedAllow       = "matched allow"
	decisionNoMatching         = "no matching allow"
	decisionInvalidAction      = "missing action or resource"
	decisionMalformedResource  = "requested resource claims the MRN scheme but is not a valid MRN"
	decisionMalformedStatement = "a policy statement resource claims the MRN scheme but is not a valid MRN pattern"
)

// Evaluate applies AWS-style identity-policy semantics over the supplied
// principal-scoped documents: default deny, explicit deny wins, and allow only
// when both action and resource match a v1 statement.
func Evaluate(docs []PolicyDocument, req AuthzRequest) Decision {
	if strings.TrimSpace(req.Action) == "" || strings.TrimSpace(req.Resource) == "" {
		return Decision{Allowed: false, Reason: decisionInvalidAction}
	}

	// A requested resource that claims the MRN scheme but cannot be parsed is
	// refused BEFORE any matching. Falling through to legacy glob matching would
	// let a malformed string like "mrn:storage:acme" be matched by a legacy
	// pattern (e.g. "mrn*") under semantics its author never contemplated — the
	// "mrn:" prefix is a commitment to the MRN grammar, so the only safe answers
	// are "valid MRN" or "denied".
	if mrn.IsMRN(strings.TrimSpace(req.Resource)) {
		if _, err := mrn.Parse(strings.TrimSpace(req.Resource)); err != nil {
			return Decision{Allowed: false, Reason: decisionMalformedResource}
		}
	}

	allowed := false
	for _, doc := range docs {
		// An unrecognised document version is refused, not skipped. Skipping is
		// asymmetric: dropping an allow is safe, but dropping a DENY silently
		// removes a guardrail while sibling allows still apply. A document written
		// as "v2" or "1" must never be able to widen access by being ignored.
		if doc.Version != policyVersionV1 {
			return Decision{Allowed: false, Reason: decisionUnsupportedVersion}
		}
		for _, stmt := range doc.Statement {
			matched, err := statementMatches(stmt, req)
			if err != nil {
				// Same asymmetry as the version/effect refusals above: a statement
				// resource that claims the MRN scheme but cannot be parsed may have
				// been INTENDED as a deny, so skipping it would silently remove a
				// guardrail. Write-time validation rejects these, so hitting this
				// path means the store holds a document this build cannot honor —
				// refuse the whole decision.
				return Decision{Allowed: false, Reason: decisionMalformedStatement}
			}
			if !matched {
				continue
			}
			switch {
			case strings.EqualFold(stmt.Effect, "deny"):
				return Decision{Allowed: false, Reason: decisionExplicitDeny}
			case strings.EqualFold(stmt.Effect, "allow"):
				allowed = true
			default:
				// Same reasoning: an effect that is neither allow nor deny may have
				// been INTENDED as a deny, so it must not be discarded.
				return Decision{Allowed: false, Reason: decisionUnknownEffect}
			}
		}
	}

	if allowed {
		return Decision{Allowed: true, Reason: decisionMatchedAllow}
	}
	return Decision{Allowed: false, Reason: decisionNoMatching}
}

func statementMatches(stmt PolicyStatement, req AuthzRequest) (bool, error) {
	// Action matching is deliberately untouched by MRN support: actions are flat
	// "service:verb" strings in both worlds and keep the original glob semantics.
	if !anyPatternMatches(stmt.Action, req.Action) {
		return false, nil
	}
	return anyResourceMatches(stmt.Resource, req.Resource)
}

func anyResourceMatches(patterns []string, resource string) (bool, error) {
	for _, pattern := range patterns {
		matched, err := resourceMatches(pattern, resource)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

// resourceMatches is the dual-mode resource matcher with a strict type wall
// between the two resource worlds.
//
//   - MRN statement pattern vs MRN requested resource → segment-aware
//     mrn.MatchPattern, so wildcards can never span a colon boundary.
//   - legacy flat pattern vs legacy flat resource → the original glob,
//     byte-for-byte unchanged.
//   - the universal "*" statement resource matches EVERYTHING in both worlds.
//     This is required, not a convenience: the setup-granted control policy
//     (Resource: ["*"]) predates MRNs and must keep authorizing the control
//     plane once it starts asking about MRN resources.
//   - any other cross-world pairing → NO match, by design. A legacy "auth:*"
//     grant was written before MRNs existed; letting it glob-match
//     "mrn:auth:..." would silently widen it onto resources it never
//     contemplated. Symmetrically, an MRN pattern is a scoped, segment-aware
//     grant and must not be reinterpreted as a flat glob over legacy strings.
//     Cross-world access is granted by writing a resource in that world's own
//     grammar, never by coincidence of spelling.
func resourceMatches(pattern, resource string) (bool, error) {
	pattern = strings.TrimSpace(pattern)
	resource = strings.TrimSpace(resource)
	if pattern == "" || resource == "" {
		return false, nil
	}
	if pattern == "*" {
		return true, nil
	}
	patternIsMRN := mrn.IsMRN(pattern)
	resourceIsMRN := mrn.IsMRN(resource)
	switch {
	case patternIsMRN && resourceIsMRN:
		return mrn.MatchPattern(pattern, resource)
	case !patternIsMRN && !resourceIsMRN:
		return wildcardMatch(pattern, resource), nil
	default:
		return false, nil
	}
}

func anyPatternMatches(patterns []string, value string) bool {
	for _, pattern := range patterns {
		if wildcardMatch(pattern, value) {
			return true
		}
	}
	return false
}

func wildcardMatch(pattern, value string) bool {
	pattern = strings.TrimSpace(pattern)
	value = strings.TrimSpace(value)
	if pattern == "" || value == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == value
	}

	parts := strings.Split(pattern, "*")
	pos := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		idx := strings.Index(value[pos:], part)
		if idx < 0 {
			return false
		}
		if i == 0 && !strings.HasPrefix(pattern, "*") && idx != 0 {
			return false
		}
		pos += idx + len(part)
	}
	last := parts[len(parts)-1]
	return strings.HasSuffix(pattern, "*") || last == "" || strings.HasSuffix(value, last)
}
