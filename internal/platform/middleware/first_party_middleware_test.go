package middleware

import (
	"context"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubFirstPartyResolver struct {
	firstParty map[string]bool
	calls      int
}

func (s *stubFirstPartyResolver) IsFirstPartyClient(_ context.Context, id string) bool {
	s.calls++
	return s.firstParty[id]
}

// The self-service API authorizes on the SUBJECT alone, so any valid access
// token for that user reaches it. A token minted for a third-party OAuth client
// — one the user consented to for `openid profile` — must not be able to change
// their email, rotate their password, revoke their sessions, or strip their MFA.
func TestRequireFirstPartyClient(t *testing.T) {
	guarded := func(r *stubFirstPartyResolver) http.Handler {
		return RequireFirstPartyClient(r)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	}

	call := func(h http.Handler, mutate func(*http.Request)) int {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/account", nil)
		if mutate != nil {
			mutate(r)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}

	// A nil resolver cannot distinguish first- from third-party, so it must not
	// silently allow everything through.
	t.Run("fails closed with no resolver", func(t *testing.T) {
		h := RequireFirstPartyClient(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		if code := call(h, nil); code != http.StatusForbidden {
			t.Fatalf("expected 403 with no resolver, got %d", code)
		}
	})

	// INVERTED. This used to assert that cookie-borne requests pass through
	// unchecked, on the premise that only the hosted first-party app holds the
	// session cookie. That premise is false: a cookie is just a request header a
	// server-side caller can set, so the guard was bypassable verbatim by moving
	// the same third-party token from Authorization into access_token — which
	// returned the full account, the GDPR PII export, and session revocation.
	t.Run("an absent token falls through to the JWT middleware", func(t *testing.T) {
		res := &stubFirstPartyResolver{firstParty: map[string]bool{}}
		if code := call(guarded(res), nil); code != http.StatusOK {
			t.Fatalf("expected pass-through when there is no token at all, got %d", code)
		}
		if res.calls != 0 {
			t.Fatal("with no token there is no client to resolve")
		}
	})

	// An unparseable token is the per-route JWT middleware's job to reject with
	// the canonical 401; this guard must not pre-empt it with a 403.
	t.Run("an invalid token defers to the JWT middleware", func(t *testing.T) {
		res := &stubFirstPartyResolver{firstParty: map[string]bool{}}
		code := call(guarded(res), func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer not-a-real-token")
		})
		if code != http.StatusOK {
			t.Fatalf("expected pass-through to the next handler, got %d", code)
		}
	})
}

// The regression test that matters: a REAL, correctly-signed access token minted
// for a third-party client must be refused, and the equivalent first-party token
// must pass. The boundary cases above prove the guard's plumbing; this proves the
// actual security property.
func TestRequireFirstPartyClientRefusesThirdPartyTokens(t *testing.T) {
	initTestJWTKeys(t)

	const firstParty = "auth-identity-client"
	const thirdParty = "some-partner-app"

	mint := func(t *testing.T, clientID string) string {
		t.Helper()
		// aud must equal client_id: the guard requires both and rejects a mismatch,
		// which is what stops a token with a doctored private claim slipping past.
		tok, err := jwt.GenerateAccessToken(
			uuid.New().String(), "openid profile",
			"https://auth.example.com", clientID, clientID, "provider-1",
		)
		if err != nil {
			t.Fatalf("mint token: %v", err)
		}
		return tok
	}

	resolver := &stubFirstPartyResolver{firstParty: map[string]bool{firstParty: true}}
	handler := RequireFirstPartyClient(resolver)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func(t *testing.T, method, path, token string) int {
		t.Helper()
		r := httptest.NewRequest(method, path, nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code
	}

	// A user consenting to sign in with a partner app must not thereby hand that
	// app their email, password, sessions or MFA.
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/account"},
		{http.MethodPut, "/api/v1/account/password"},
		{http.MethodPut, "/api/v1/account/username"},
		{http.MethodDelete, "/api/v1/account/sessions"},
		{http.MethodPost, "/api/v1/mfa/reset"},
	} {
		t.Run("third-party refused: "+tc.method+" "+tc.path, func(t *testing.T) {
			if code := call(t, tc.method, tc.path, mint(t, thirdParty)); code != http.StatusForbidden {
				t.Fatalf("expected 403 for a third-party token, got %d", code)
			}
		})
	}

	t.Run("first-party token still passes", func(t *testing.T) {
		if code := call(t, http.MethodGet, "/api/v1/account", mint(t, firstParty)); code != http.StatusOK {
			t.Fatalf("expected 200 for a first-party token, got %d", code)
		}
	})

	// An unknown client is not first-party, so it must be refused rather than
	// defaulting to allowed.
	t.Run("an unknown client is refused", func(t *testing.T) {
		if code := call(t, http.MethodGet, "/api/v1/account", mint(t, "never-registered")); code != http.StatusForbidden {
			t.Fatalf("expected 403 for an unresolvable client, got %d", code)
		}
	})
}

// The bypass that mattered: the SAME third-party token must be refused whether
// it arrives in the Authorization header or in a cookie. Transport is not a
// trust boundary.
func TestRequireFirstPartyClientChecksCookieBorneTokens(t *testing.T) {
	initTestJWTKeys(t)

	const firstParty = "auth-identity-client"
	const thirdParty = "some-partner-app"

	mint := func(t *testing.T, clientID string) string {
		t.Helper()
		tok, err := jwt.GenerateAccessToken(
			uuid.New().String(), "openid profile",
			"https://auth.example.com", clientID, clientID, "provider-1",
		)
		if err != nil {
			t.Fatalf("mint token: %v", err)
		}
		return tok
	}

	resolver := &stubFirstPartyResolver{firstParty: map[string]bool{firstParty: true}}
	handler := RequireFirstPartyClient(resolver)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	viaCookie := func(t *testing.T, name, token string) int {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/account", nil)
		r.AddCookie(&http.Cookie{Name: name, Value: token})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code
	}

	for _, cookieName := range []string{"access_token", "__Host-access_token", "__Secure-access_token"} {
		t.Run("third-party refused via "+cookieName, func(t *testing.T) {
			if code := viaCookie(t, cookieName, mint(t, thirdParty)); code != http.StatusForbidden {
				t.Fatalf("a third-party token in %s must be refused, got %d", cookieName, code)
			}
		})
	}

	// The hosted login app authenticates by cookie and must keep working.
	t.Run("first-party cookie session still passes", func(t *testing.T) {
		if code := viaCookie(t, "access_token", mint(t, firstParty)); code != http.StatusOK {
			t.Fatalf("expected 200 for a first-party cookie session, got %d", code)
		}
	})
}
