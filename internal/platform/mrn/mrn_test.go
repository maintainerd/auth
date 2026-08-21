package mrn

import "testing"

func TestIsMRN(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"mrn:storage:acme:billing-app:bucket/invoices", true},
		{"mrn:", true},          // claims the scheme, even though it cannot Parse
		{"mrn:not-valid", true}, // same: IsMRN is a commitment check, not validation
		{"auth:*", false},
		{"*", false},
		{"", false},
		{"MRN:storage:acme:p:x", false}, // scheme is lowercase only
		{" mrn:storage:acme:p:x", false},
	}
	for _, tt := range tests {
		if got := IsMRN(tt.in); got != tt.want {
			t.Errorf("IsMRN(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    MRN
		wantErr bool
	}{
		{
			"full mrn with slashes in path",
			"mrn:storage:acme:billing-app:bucket/invoices/object/2026/q3.pdf",
			MRN{Service: "storage", Tenant: "acme", Project: "billing-app", ResourcePath: "bucket/invoices/object/2026/q3.pdf"},
			false,
		},
		{
			"empty project is tenant-scoped",
			"mrn:core:acme::role/admin",
			MRN{Service: "core", Tenant: "acme", Project: "", ResourcePath: "role/admin"},
			false,
		},
		{
			"empty tenant and project is platform-scoped",
			"mrn:core:::agent/agent-1",
			MRN{Service: "core", Tenant: "", Project: "", ResourcePath: "agent/agent-1"},
			false,
		},
		{
			// The whole point of SplitN limit 5: colons in the resource-path are
			// part of the path, never new segments.
			"colons in resource-path are not re-split",
			"mrn:core:acme:proj:object:v2:latest",
			MRN{Service: "core", Tenant: "acme", Project: "proj", ResourcePath: "object:v2:latest"},
			false,
		},
		{"missing prefix", "storage:acme:billing:bucket/x", MRN{}, true},
		{"prefix only", "mrn:", MRN{}, true},
		{"four parts", "mrn:storage:acme:bucket/x", MRN{}, true},
		{"three parts", "mrn:storage:acme", MRN{}, true},
		{"empty service", "mrn::acme:proj:bucket/x", MRN{}, true},
		{"uppercase service", "mrn:Storage:acme:proj:bucket/x", MRN{}, true},
		{"underscore in service", "mrn:my_svc:acme:proj:bucket/x", MRN{}, true},
		{"wildcard in service", "mrn:*:acme:proj:bucket/x", MRN{}, true},
		{"uppercase tenant", "mrn:storage:Acme:proj:bucket/x", MRN{}, true},
		{"wildcard tenant is not a resource", "mrn:storage:*:proj:bucket/x", MRN{}, true},
		{"dot in project", "mrn:storage:acme:billing.app:bucket/x", MRN{}, true},
		{"empty resource-path", "mrn:storage:acme:proj:", MRN{}, true},
		{"leading slash in resource-path", "mrn:storage:acme:proj:/bucket/x", MRN{}, true},
		{"control char in resource-path", "mrn:storage:acme:proj:bucket/\x00x", MRN{}, true},
		{"invalid utf-8 in resource-path", "mrn:storage:acme:proj:bucket/\xff", MRN{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) = %+v, want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Fatalf("Parse(%q) = %+v, want %+v", tt.in, got, tt.want)
			}
			if got.String() != tt.in {
				t.Fatalf("String() = %q does not round-trip %q", got.String(), tt.in)
			}
		})
	}
}

func TestParsePattern(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Pattern
		wantErr bool
	}{
		{"all literals", "mrn:storage:acme:billing:bucket/invoices", Pattern{"storage", "acme", "billing", "bucket/invoices"}, false},
		{"wildcard segments and bare path wildcard", "mrn:*:*:*:*", Pattern{"*", "*", "*", "*"}, false},
		{"trailing path wildcard", "mrn:storage:acme:*:bucket/*", Pattern{"storage", "acme", "*", "bucket/*"}, false},
		{"empty tenant and project pattern", "mrn:core:::agent/*", Pattern{"core", "", "", "agent/*"}, false},
		{"trailing wildcard without slash", "mrn:storage:acme:proj:bucket*", Pattern{"storage", "acme", "proj", "bucket*"}, false},
		{"missing prefix", "storage:acme:*:bucket/*", Pattern{}, true},
		{"four parts", "mrn:storage:acme:*", Pattern{}, true},
		{"empty service", "mrn::acme:proj:bucket/*", Pattern{}, true},
		{"uppercase tenant literal", "mrn:storage:Acme:proj:bucket/*", Pattern{}, true},
		{"mid-path wildcard", "mrn:storage:acme:proj:bucket/*/object", Pattern{}, true},
		{"embedded wildcard", "mrn:storage:acme:proj:bu*cket", Pattern{}, true},
		{"double wildcard", "mrn:storage:acme:proj:bucket/**", Pattern{}, true},
		{"leading wildcard path", "mrn:storage:acme:proj:*suffix", Pattern{}, true},
		{"empty resource-path", "mrn:storage:acme:proj:", Pattern{}, true},
		{"leading slash path", "mrn:storage:acme:proj:/bucket/*", Pattern{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePattern(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParsePattern(%q) = %+v, want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePattern(%q) error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Fatalf("ParsePattern(%q) = %+v, want %+v", tt.in, got, tt.want)
			}
		})
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		resource string
		want     bool
		wantErr  bool
	}{
		// Exact literal matching.
		{"exact match", "mrn:storage:acme:billing:bucket/invoices", "mrn:storage:acme:billing:bucket/invoices", true, false},
		{"literal path mismatch", "mrn:storage:acme:billing:bucket/invoices", "mrn:storage:acme:billing:bucket/other", false, false},
		{"service mismatch", "mrn:storage:acme:billing:bucket/x", "mrn:core:acme:billing:bucket/x", false, false},

		// Segment wildcards match anything in THAT segment, including empty.
		{"service wildcard", "mrn:*:acme:billing:bucket/x", "mrn:storage:acme:billing:bucket/x", true, false},
		{"tenant wildcard matches a tenant", "mrn:storage:*:billing:bucket/x", "mrn:storage:acme:billing:bucket/x", true, false},
		{"tenant wildcard matches empty tenant", "mrn:core:*:*:agent/agent-1", "mrn:core:::agent/agent-1", true, false},
		{"project wildcard matches empty project", "mrn:core:acme:*:role/admin", "mrn:core:acme::role/admin", true, false},

		// A literal segment matches only itself — never a prefix. This is the
		// segment-aware guarantee versus naive glob: a wildcard in the NEXT
		// segment must not let "acme" bleed into "acmecorp".
		{"tenant literal does not prefix-match", "mrn:storage:acme:*:*", "mrn:storage:acmecorp:x:y", false, false},
		{"tenant literal does not suffix-match", "mrn:storage:acme:*:*", "mrn:storage:notacme:x:y", false, false},

		// Empty pattern segments are a scope BOUNDARY, not a wildcard.
		{"empty tenant pattern matches only empty tenant", "mrn:core:::agent/*", "mrn:core:::agent/agent-1", true, false},
		{"platform-scoped pattern rejects tenant-scoped resource", "mrn:core:::agent/*", "mrn:core:acme::agent/agent-1", false, false},
		{"tenant-scoped pattern rejects project-scoped resource", "mrn:core:acme::role/*", "mrn:core:acme:billing:role/admin", false, false},

		// Resource-path glob: bare "*" and trailing prefix wildcard.
		{"bare path wildcard", "mrn:storage:acme:billing:*", "mrn:storage:acme:billing:bucket/invoices/object/2026/q3.pdf", true, false},
		{"path prefix wildcard matches deep path", "mrn:storage:acme:billing:bucket/invoices/*", "mrn:storage:acme:billing:bucket/invoices/object/2026/q3.pdf", true, false},
		{"path prefix wildcard rejects sibling", "mrn:storage:acme:billing:bucket/invoices/*", "mrn:storage:acme:billing:bucket/receipts/r1", false, false},
		{"path prefix without slash matches string prefix", "mrn:storage:acme:billing:bucket*", "mrn:storage:acme:billing:buckets/x", true, false},

		// Malformed inputs are errors, never a silent non-match.
		{"malformed pattern (too few parts)", "mrn:storage:acme:*", "mrn:storage:acme:x:y", false, true},
		{"malformed pattern (mid-path wildcard)", "mrn:storage:acme:x:a/*/b", "mrn:storage:acme:x:a/1/b", false, true},
		{"malformed resource", "mrn:storage:acme:*:*", "mrn:storage:acme", false, true},
		{"resource with wildcard is malformed", "mrn:storage:acme:*:*", "mrn:storage:acme:x:*", false, true},
		{"legacy string as resource is malformed", "mrn:storage:acme:*:*", "auth:login", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MatchPattern(tt.pattern, tt.resource)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("MatchPattern(%q, %q) = %v, want error", tt.pattern, tt.resource, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("MatchPattern(%q, %q) error: %v", tt.pattern, tt.resource, err)
			}
			if got != tt.want {
				t.Fatalf("MatchPattern(%q, %q) = %v, want %v", tt.pattern, tt.resource, got, tt.want)
			}
		})
	}
}
