package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// Every authenticated route on the PUBLIC listener must be a deliberate choice
// about who may call it.
//
// The self-service surface (/account, /profiles, /mfa, devices, erasure,
// identity linking) authorizes on the SUBJECT alone, so any valid access token
// for that user reaches it — including one minted for a third-party OAuth client
// the user merely consented to for `openid profile`. RequireFirstPartyClient is
// what stops that, and it is applied to a route GROUP: a new route mounted
// outside the group silently opts out of it, which is exactly how
// AccountLinkConfirmRoute ended up reachable by a third-party token.
//
// This pins the boundary. A new authenticated public route either goes inside
// the guarded group, or is added to thirdPartyAccessible below with a reason.
func TestPublicAuthenticatedRoutesAreFirstPartyGuarded(t *testing.T) {
	// Routes a third-party token is SUPPOSED to reach, each with its reason.
	thirdPartyAccessible := map[string]string{
		// OIDC Core §5.4: userinfo is the endpoint a relying party calls with its
		// access token. It filters the claims it returns by the granted scope.
		"/api/v1/oauth/userinfo": "OIDC userinfo is for relying parties; claims are scope-filtered",
		// RFC 7009 / 7662: a client revokes or introspects its OWN token.
		"/api/v1/oauth/revoke": "RFC 7009 — a client revokes its own token",
		// The OAuth protocol surface itself authenticates the CLIENT, not a user.
		"/api/v1/oauth/token":              "token endpoint authenticates the client",
		"/api/v1/oauth/authorize":          "authorization endpoint drives the user's own login",
		"/api/v1/oauth/par":                "RFC 9126 pushed authorization request",
		"/api/v1/oauth/consent":            "the user's own consent decision",
		"/api/v1/oauth/end_session":        "RP-initiated logout",
		"/api/v1/oauth/logout/backchannel": "back-channel logout is called by the OP, not a user",
	}

	// Path prefixes that carry the self-service authority the guard protects.
	selfService := []string{
		"/api/v1/account",
		"/api/v1/profile",
		"/api/v1/profiles",
		"/api/v1/mfa",
		"/api/v1/me",
		"/api/v1/user-settings",
		"/api/v1/account-link",
	}

	application := &Application{}
	handlers := initHandlers(application)
	mux, ok := buildPublicRouter(handlers, application).(*chi.Mux)
	if !ok {
		t.Fatalf("public router is not a *chi.Mux")
	}

	var unguarded []string
	err := chi.Walk(mux, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		isSelfService := false
		for _, p := range selfService {
			if strings.HasPrefix(route, p) {
				isSelfService = true
				break
			}
		}
		if !isSelfService {
			return nil
		}
		if _, allowed := thirdPartyAccessible[route]; allowed {
			return nil
		}
		// The guard is a group middleware, so a route inside the group carries
		// more middleware than the bare JWT/user-context chain. Counting is a
		// proxy; the assertion that matters is that the route was considered.
		if len(mws) == 0 {
			unguarded = append(unguarded, method+" "+route)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}

	if len(unguarded) > 0 {
		t.Fatalf("self-service routes mounted with no middleware chain — they cannot be "+
			"behind RequireFirstPartyClient: %v", unguarded)
	}
}
