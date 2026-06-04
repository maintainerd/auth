package iam

import "strings"

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
}

const (
	policyVersionV1       = "v1"
	decisionExplicitDeny  = "explicit deny"
	decisionMatchedAllow  = "matched allow"
	decisionNoMatching    = "no matching allow"
	decisionInvalidAction = "missing action or resource"
)

// Evaluate applies AWS-style identity-policy semantics over the supplied
// principal-scoped documents: default deny, explicit deny wins, and allow only
// when both action and resource match a v1 statement.
func Evaluate(docs []PolicyDocument, req AuthzRequest) Decision {
	if strings.TrimSpace(req.Action) == "" || strings.TrimSpace(req.Resource) == "" {
		return Decision{Allowed: false, Reason: decisionInvalidAction}
	}

	allowed := false
	for _, doc := range docs {
		if doc.Version != policyVersionV1 {
			continue
		}
		for _, stmt := range doc.Statement {
			if !statementMatches(stmt, req) {
				continue
			}
			if strings.EqualFold(stmt.Effect, "deny") {
				return Decision{Allowed: false, Reason: decisionExplicitDeny}
			}
			if strings.EqualFold(stmt.Effect, "allow") {
				allowed = true
			}
		}
	}

	if allowed {
		return Decision{Allowed: true, Reason: decisionMatchedAllow}
	}
	return Decision{Allowed: false, Reason: decisionNoMatching}
}

func statementMatches(stmt PolicyStatement, req AuthzRequest) bool {
	return anyPatternMatches(stmt.Action, req.Action) && anyPatternMatches(stmt.Resource, req.Resource)
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
