package iam

import "testing"

func TestEvaluate(t *testing.T) {
	docs := []PolicyDocument{{
		Version: "v1",
		Statement: []PolicyStatement{
			{Effect: "allow", Action: []string{"serviceB:*"}, Resource: []string{"serviceB:*"}},
			{Effect: "deny", Action: []string{"serviceB:delete"}, Resource: []string{"*"}},
		},
	}}

	tests := []struct {
		name    string
		docs    []PolicyDocument
		req     AuthzRequest
		allowed bool
		reason  string
	}{
		{"default deny", docs, AuthzRequest{Action: "serviceC:invoke", Resource: "serviceC:grpc"}, false, decisionNoMatching},
		{"explicit deny wins", docs, AuthzRequest{Action: "serviceB:delete", Resource: "serviceB:grpc"}, false, decisionExplicitDeny},
		{"action wildcard allow", docs, AuthzRequest{Action: "serviceB:invoke", Resource: "serviceB:grpc"}, true, decisionMatchedAllow},
		{"resource wildcard allow", []PolicyDocument{{Version: "v1", Statement: []PolicyStatement{{Effect: "allow", Action: []string{"serviceB:invoke"}, Resource: []string{"*"}}}}}, AuthzRequest{Action: "serviceB:invoke", Resource: "anything"}, true, decisionMatchedAllow},
		{"global wildcard allow", []PolicyDocument{{Version: "v1", Statement: []PolicyStatement{{Effect: "allow", Action: []string{"*"}, Resource: []string{"*"}}}}}, AuthzRequest{Action: "x:y", Resource: "z:q"}, true, decisionMatchedAllow},
		// Fail CLOSED, do not skip. Skipping is asymmetric: dropping an allow is
		// safe, dropping a deny silently removes a guardrail.
		{"unsupported version is refused, not ignored", []PolicyDocument{{Version: "v2", Statement: []PolicyStatement{{Effect: "allow", Action: []string{"*"}, Resource: []string{"*"}}}}}, AuthzRequest{Action: "x:y", Resource: "z:q"}, false, decisionUnsupportedVersion},
		{"missing action", docs, AuthzRequest{Resource: "serviceB:grpc"}, false, decisionInvalidAction},
		{"mid segment glob", []PolicyDocument{{Version: "v1", Statement: []PolicyStatement{{Effect: "allow", Action: []string{"svc:*:read"}, Resource: []string{"res:*"}}}}}, AuthzRequest{Action: "svc:thing:read", Resource: "res:item"}, true, decisionMatchedAllow},
		{"unrecognised effect is refused, not ignored", []PolicyDocument{{Version: "v1", Statement: []PolicyStatement{{Effect: "wat", Action: []string{"*"}, Resource: []string{"*"}}}}}, AuthzRequest{Action: "x:y", Resource: "z:q"}, false, decisionUnknownEffect},
		{"empty pattern ignored", []PolicyDocument{{Version: "v1", Statement: []PolicyStatement{{Effect: "allow", Action: []string{""}, Resource: []string{"*"}}}}}, AuthzRequest{Action: "x:y", Resource: "z:q"}, false, decisionNoMatching},
		{"prefix mismatch glob denied", []PolicyDocument{{Version: "v1", Statement: []PolicyStatement{{Effect: "allow", Action: []string{"svc:*"}, Resource: []string{"res:*"}}}}}, AuthzRequest{Action: "other:read", Resource: "res:item"}, false, decisionNoMatching},
		{"leading literal must start value", []PolicyDocument{{Version: "v1", Statement: []PolicyStatement{{Effect: "allow", Action: []string{"svc*"}, Resource: []string{"*"}}}}}, AuthzRequest{Action: "other-svc-read", Resource: "res:item"}, false, decisionNoMatching},
		{"suffix mismatch glob denied", []PolicyDocument{{Version: "v1", Statement: []PolicyStatement{{Effect: "allow", Action: []string{"*:read"}, Resource: []string{"res:*"}}}}}, AuthzRequest{Action: "svc:write", Resource: "res:item"}, false, decisionNoMatching},
		{"middle segment absent denied", []PolicyDocument{{Version: "v1", Statement: []PolicyStatement{{Effect: "allow", Action: []string{"svc:*:read"}, Resource: []string{"*"}}}}}, AuthzRequest{Action: "svc:read", Resource: "res:item"}, false, decisionNoMatching},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(tt.docs, tt.req)
			if got.Allowed != tt.allowed || got.Reason != tt.reason {
				t.Fatalf("Evaluate() = %+v, want allowed=%v reason=%q", got, tt.allowed, tt.reason)
			}
		})
	}
}

// MRN resources are matched segment-aware, legacy flat strings keep the
// original glob, and the two worlds never match each other — except through the
// universal "*" statement resource, which bridges both.
func TestEvaluate_MRNResources(t *testing.T) {
	allow := func(action, resource string) []PolicyDocument {
		return []PolicyDocument{{Version: "v1", Statement: []PolicyStatement{
			{Effect: "allow", Action: []string{action}, Resource: []string{resource}},
		}}}
	}

	tests := []struct {
		name    string
		docs    []PolicyDocument
		req     AuthzRequest
		allowed bool
		reason  string
	}{
		// The type wall: a legacy grant written before MRNs existed must not
		// silently start matching "mrn:..." resources it never contemplated —
		// "storage:*" would glob-match "mrn:storage:..." otherwise. And vice versa.
		{"legacy pattern never matches an MRN resource", allow("storage:read", "storage:*"),
			AuthzRequest{Action: "storage:read", Resource: "mrn:storage:acme:billing:bucket/invoices"}, false, decisionNoMatching},
		{"legacy mrn-prefixed glob never matches an MRN resource", allow("storage:read", "mrn*"),
			AuthzRequest{Action: "storage:read", Resource: "mrn:storage:acme:billing:bucket/invoices"}, false, decisionNoMatching},
		{"MRN pattern never matches a legacy resource", allow("storage:read", "mrn:storage:*:*:*"),
			AuthzRequest{Action: "storage:read", Resource: "storage:grpc"}, false, decisionNoMatching},

		// The universal "*" bridges both worlds — this is what keeps the
		// setup-granted control policy (Resource: ["*"]) authorizing MRN requests.
		{"universal star matches an MRN resource", allow("*", "*"),
			AuthzRequest{Action: "storage:read", Resource: "mrn:storage:acme:billing:bucket/invoices"}, true, decisionMatchedAllow},
		{"universal star still matches a legacy resource", allow("*", "*"),
			AuthzRequest{Action: "storage:read", Resource: "storage:grpc"}, true, decisionMatchedAllow},

		// Tenant-segment isolation: a wildcard never spans the colon boundary.
		{"MRN grant matches its own tenant", allow("storage:read", "mrn:storage:tenant-a:*:*"),
			AuthzRequest{Action: "storage:read", Resource: "mrn:storage:tenant-a:proj:bucket/x"}, true, decisionMatchedAllow},
		{"MRN grant never matches another tenant", allow("storage:read", "mrn:storage:tenant-a:*:*"),
			AuthzRequest{Action: "storage:read", Resource: "mrn:storage:tenant-b:proj:bucket/x"}, false, decisionNoMatching},
		{"MRN tenant literal never prefix-bleeds", allow("storage:read", "mrn:storage:acme:*:*"),
			AuthzRequest{Action: "storage:read", Resource: "mrn:storage:acmecorp:proj:bucket/x"}, false, decisionNoMatching},

		// Resource-path prefix matching.
		{"path prefix wildcard matches deep path", allow("storage:read", "mrn:storage:acme:billing:bucket/invoices/*"),
			AuthzRequest{Action: "storage:read", Resource: "mrn:storage:acme:billing:bucket/invoices/object/2026/q3.pdf"}, true, decisionMatchedAllow},
		{"path prefix wildcard rejects sibling path", allow("storage:read", "mrn:storage:acme:billing:bucket/invoices/*"),
			AuthzRequest{Action: "storage:read", Resource: "mrn:storage:acme:billing:bucket/receipts/r1"}, false, decisionNoMatching},

		// Empty pattern segments are a scope boundary, not a wildcard: a
		// platform-scoped grant speaks only for platform-scoped resources.
		{"platform-scoped pattern matches platform-scoped resource", allow("core:read", "mrn:core:::agent/*"),
			AuthzRequest{Action: "core:read", Resource: "mrn:core:::agent/agent-1"}, true, decisionMatchedAllow},
		{"platform-scoped pattern rejects tenant-scoped resource", allow("core:read", "mrn:core:::agent/*"),
			AuthzRequest{Action: "core:read", Resource: "mrn:core:acme::agent/agent-1"}, false, decisionNoMatching},

		// A requested resource claiming the MRN scheme but failing to parse is
		// refused outright — it must never fall through to legacy matching, even
		// against a statement that would have matched it as a flat string.
		{"malformed MRN resource is refused, not glob-matched", allow("storage:read", "mrn*"),
			AuthzRequest{Action: "storage:read", Resource: "mrn:storage:acme"}, false, decisionMalformedResource},
		{"malformed MRN resource is refused even against universal star", allow("*", "*"),
			AuthzRequest{Action: "storage:read", Resource: "mrn:storage:acme:proj:/leading/slash"}, false, decisionMalformedResource},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(tt.docs, tt.req)
			if got.Allowed != tt.allowed || got.Reason != tt.reason {
				t.Fatalf("Evaluate() = %+v, want allowed=%v reason=%q", got, tt.allowed, tt.reason)
			}
		})
	}
}

// Explicit deny wins inside the MRN world exactly as it does for flat strings.
func TestEvaluate_MRNExplicitDenyWins(t *testing.T) {
	docs := []PolicyDocument{{Version: "v1", Statement: []PolicyStatement{
		{Effect: "allow", Action: []string{"storage:*"}, Resource: []string{"mrn:storage:acme:*:*"}},
		{Effect: "deny", Action: []string{"storage:*"}, Resource: []string{"mrn:storage:acme:billing:bucket/secret/*"}},
	}}}

	got := Evaluate(docs, AuthzRequest{Action: "storage:read", Resource: "mrn:storage:acme:billing:bucket/secret/key.pem"})
	if got.Allowed || got.Reason != decisionExplicitDeny {
		t.Fatalf("Evaluate() = %+v, want explicit deny", got)
	}

	got = Evaluate(docs, AuthzRequest{Action: "storage:read", Resource: "mrn:storage:acme:billing:bucket/public/logo.png"})
	if !got.Allowed {
		t.Fatalf("Evaluate() = %+v, want allow outside the denied prefix", got)
	}
}

// A statement resource that claims the MRN scheme but cannot be parsed refuses
// the whole decision: it may have been intended as a deny, and dropping a deny
// silently removes a guardrail (the same asymmetry as unsupported versions).
func TestEvaluate_MalformedStatementMRNRefusesDecision(t *testing.T) {
	docs := []PolicyDocument{{Version: "v1", Statement: []PolicyStatement{
		{Effect: "allow", Action: []string{"*"}, Resource: []string{"*"}},
		{Effect: "deny", Action: []string{"storage:delete"}, Resource: []string{"mrn:storage:acme:bucket/*"}}, // 4 parts — malformed
	}}}
	got := Evaluate(docs, AuthzRequest{Action: "storage:delete", Resource: "mrn:storage:acme:billing:bucket/x"})
	if got.Allowed || got.Reason != decisionMalformedStatement {
		t.Fatalf("Evaluate() = %+v, want refusal with %q", got, decisionMalformedStatement)
	}
}

// Regression lock: the flat-string policies that exist in production today —
// the setup-granted control policy (Resource: ["*"]) and service policies like
// ["auth:*", "account:profile"] — must keep evaluating byte-for-byte
// identically for legacy requests.
func TestEvaluate_LegacyFlatBehaviorUnchanged(t *testing.T) {
	controlPolicy := []PolicyDocument{{Version: "v1", Statement: []PolicyStatement{
		{Effect: "allow", Action: []string{"*"}, Resource: []string{"*"}},
	}}}
	servicePolicy := []PolicyDocument{{Version: "v1", Statement: []PolicyStatement{
		{Effect: "allow", Action: []string{"user:*"}, Resource: []string{"auth:*", "account:profile"}},
	}}}

	tests := []struct {
		name    string
		docs    []PolicyDocument
		req     AuthzRequest
		allowed bool
		reason  string
	}{
		{"control policy allows anything legacy", controlPolicy, AuthzRequest{Action: "tenant:delete", Resource: "tenant:42"}, true, decisionMatchedAllow},
		{"service glob matches its prefix", servicePolicy, AuthzRequest{Action: "user:read", Resource: "auth:login"}, true, decisionMatchedAllow},
		{"service literal matches exactly", servicePolicy, AuthzRequest{Action: "user:read", Resource: "account:profile"}, true, decisionMatchedAllow},
		{"service literal rejects near-miss", servicePolicy, AuthzRequest{Action: "user:read", Resource: "account:profile2"}, false, decisionNoMatching},
		{"foreign resource still default-denied", servicePolicy, AuthzRequest{Action: "user:read", Resource: "billing:invoice"}, false, decisionNoMatching},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(tt.docs, tt.req)
			if got.Allowed != tt.allowed || got.Reason != tt.reason {
				t.Fatalf("Evaluate() = %+v, want allowed=%v reason=%q", got, tt.allowed, tt.reason)
			}
		})
	}
}

// The asymmetry that makes silent-skip unsafe: a deny in a document the evaluator
// does not understand must never be dropped while sibling allows still apply.
func TestEvaluate_DroppedDenyCannotWidenAccess(t *testing.T) {
	t.Run("an unsupported-version document refuses the whole decision", func(t *testing.T) {
		docs := []PolicyDocument{
			{Version: "v1", Statement: []PolicyStatement{
				{Effect: "allow", Action: []string{"*"}, Resource: []string{"*"}},
			}},
			// Written by a newer release, carrying a deny this build cannot read.
			{Version: "v2", Statement: []PolicyStatement{
				{Effect: "deny", Action: []string{"tenant:delete"}, Resource: []string{"*"}},
			}},
		}
		got := Evaluate(docs, AuthzRequest{Action: "tenant:delete", Resource: "tenant:1"})
		if got.Allowed {
			t.Fatal("an unreadable document must not be skipped past a matching allow")
		}
	})

	t.Run("an unrecognised effect refuses the whole decision", func(t *testing.T) {
		docs := []PolicyDocument{{Version: "v1", Statement: []PolicyStatement{
			{Effect: "allow", Action: []string{"*"}, Resource: []string{"*"}},
			{Effect: "Deny ", Action: []string{"tenant:delete"}, Resource: []string{"*"}},
		}}}
		got := Evaluate(docs, AuthzRequest{Action: "tenant:delete", Resource: "tenant:1"})
		if got.Allowed {
			t.Fatal("a statement whose effect is not understood must not be treated as absent")
		}
	})
}
