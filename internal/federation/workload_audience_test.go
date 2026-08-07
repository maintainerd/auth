//nolint:staticcheck
package federation

import "testing"

// The workload-identity exchange is KEYLESS — it authenticates a workload by its
// upstream assertion, not a client secret. Honouring a caller-supplied audience
// there let anyone who could satisfy the trust policy mint a token addressed to
// any resource server that trusts this issuer, not just the one the federation
// was registered for.
func TestResolveWorkloadAudience(t *testing.T) {
	const registered = "https://api.example.com"

	t.Run("defaults to the registered audience when none is requested", func(t *testing.T) {
		got, err := resolveWorkloadAudience("", registered)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != registered {
			t.Fatalf("expected %q, got %q", registered, got)
		}
	})

	t.Run("blank and whitespace are treated as absent", func(t *testing.T) {
		got, err := resolveWorkloadAudience("   ", registered)
		if err != nil || got != registered {
			t.Fatalf("expected the registered audience, got %q err=%v", got, err)
		}
	})

	t.Run("the registered audience may be requested explicitly", func(t *testing.T) {
		got, err := resolveWorkloadAudience(registered, registered)
		if err != nil || got != registered {
			t.Fatalf("expected %q, got %q err=%v", registered, got, err)
		}
	})

	// The whole point: an unregistered target is refused outright.
	t.Run("an unregistered audience is refused with invalid_target", func(t *testing.T) {
		got, err := resolveWorkloadAudience("https://victim.example.com", registered)
		//nolint:staticcheck
		if err == nil {
			t.Fatal("an unregistered audience must be refused")
		}
		if err.Code != "invalid_target" {
			t.Fatalf("RFC 8693 §2.2.2 requires invalid_target, got %q", err.Code)
		}
		if got != "" {
			t.Fatalf("no audience may be returned on refusal, got %q", got)
		}
	})

	// Refusing rather than downgrading matters: silently issuing a token for a
	// different audience than the caller named is its own hazard.
	t.Run("a mismatch is never silently downgraded to the registered value", func(t *testing.T) {
		got, _ := resolveWorkloadAudience("https://victim.example.com", registered)
		if got == registered {
			t.Fatal("a mismatched request must not fall back to the registered audience")
		}
	})
}
