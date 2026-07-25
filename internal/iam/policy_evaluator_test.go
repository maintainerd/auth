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
