package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// DPoPBindingValidator verifies a DPoP proof against the access token it is
// presented with. It returns nil when the proof is valid AND its key thumbprint
// matches the token's cnf.jkt binding.
//
// It is injected rather than imported so this package does not depend on the dpop
// package (the dependency runs the other way for the token endpoint).
type DPoPBindingValidator func(
	ctx context.Context,
	proofHeader string,
	method string,
	requestURL string,
	accessToken string,
	cnfJKT string,
) error

var (
	dpopBindingValidator DPoPBindingValidator
	dpopRequestURL       func(*http.Request) string
)

// ConfigureDPoPBinding installs the resource-side DPoP enforcement used by
// JWTAuthMiddleware. Call once at startup, before serving.
//
// Until this is called, a sender-constrained token cannot be used at all: the
// enforcement below fails closed rather than degrading to bearer semantics, since
// silently accepting a bound token without its proof is exactly the theft scenario
// DPoP exists to prevent.
func ConfigureDPoPBinding(validator DPoPBindingValidator, requestURL func(*http.Request) string) {
	dpopBindingValidator = validator
	dpopRequestURL = requestURL
}

// errDPoPRequired is returned when a sender-constrained token is presented without
// a matching proof.
var errDPoPRequired = errors.New("this access token is bound to a DPoP key and must be presented with a valid DPoP proof")

// tokenConfirmationThumbprint reads the cnf.jkt confirmation claim (RFC 9449 §6.1),
// which is present only on sender-constrained tokens.
func tokenConfirmationThumbprint(rawClaims map[string]any) string {
	cnf, ok := rawClaims["cnf"].(map[string]any)
	if !ok {
		return ""
	}
	jkt, _ := cnf["jkt"].(string)
	return strings.TrimSpace(jkt)
}

// enforceDPoPBinding implements RFC 9449 §7.1 on the resource side.
//
// The gap this closes: tokens were issued with a cnf.jkt binding, but nothing ever
// checked it. A bound token presented as `Authorization: Bearer <token>` was
// accepted like any other, so stealing one was enough — the binding was decorative.
// Conversely `Authorization: DPoP <token>`, the scheme RFC 9449 requires, was not
// recognized at all, so a correctly behaving client could not authenticate.
//
// A token with no cnf claim is unconstrained and keeps plain bearer semantics.
func enforceDPoPBinding(r *http.Request, scheme, token string, rawClaims map[string]any) error {
	jkt := tokenConfirmationThumbprint(rawClaims)
	if jkt == "" {
		return nil
	}

	// §7.1: a sender-constrained token MUST be presented with the DPoP scheme.
	// Accepting it under Bearer (or from a cookie, which carries no scheme) would
	// let a stolen token be replayed without the key.
	if !strings.EqualFold(scheme, "dpop") {
		return errDPoPRequired
	}

	proof := r.Header.Get("DPoP")
	if strings.TrimSpace(proof) == "" {
		return errDPoPRequired
	}
	if dpopBindingValidator == nil {
		// Fail closed: with no validator wired there is no way to check the binding,
		// and accepting the token anyway would silently reduce it to a bearer token.
		return errDPoPRequired
	}

	requestURL := r.URL.String()
	if dpopRequestURL != nil {
		requestURL = dpopRequestURL(r)
	}

	return dpopBindingValidator(r.Context(), proof, r.Method, requestURL, token, jkt)
}

// IsSenderConstrainedToken reports whether an access token is bound to a DPoP key.
//
// It exists for transports that cannot verify a proof at all. DPoP is defined over
// HTTP (htu/htm bind a proof to a method and URL), so a bound token presented over
// a non-HTTP transport such as gRPC can never be proven. Accepting it there would
// silently downgrade it to a bearer token and reopen the theft path, so callers must
// reject it instead.
func IsSenderConstrainedToken(rawClaims map[string]any) bool {
	return tokenConfirmationThumbprint(rawClaims) != ""
}
