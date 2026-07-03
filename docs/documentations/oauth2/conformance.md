# OIDC / OAuth2 Conformance

How to validate that the OAuth2/OIDC surface is spec-compliant before release (checklist item **J6**). Run against a test deployment; not part of CI.

## Discovery is the contract

Fetch discovery and confirm every advertised capability is real and every implemented endpoint is advertised:

```bash
curl -fsS https://public-api.auth.maintainerd.local/.well-known/openid-configuration | jq
```

Cross-check `issuer`, `authorization_endpoint`, `token_endpoint`, `userinfo_endpoint`, `jwks_uri`, `end_session_endpoint`, `introspection_endpoint`, `revocation_endpoint`, `registration_endpoint`, `device_authorization_endpoint`, `pushed_authorization_request_endpoint`, and the `*_supported` arrays (`response_types`, `grant_types`, `code_challenge_methods_supported` must include `S256`, `id_token_signing_alg_values_supported` must be RS256/PS256 only — never `none`).

## Automated suite

Use the OpenID Foundation conformance suite (https://gitlab.com/openid/conformance-suite) against a test deployment:

1. Register a test client (authorization_code + PKCE, a registered `redirect_uri`).
2. Run the **Basic OP** and **Config OP** profiles.
3. Capture the pass report and attach it to the release notes (K7 sign-off).

## Manual spot-checks (must all hold)

- [ ] `/token` requires `code_verifier` when a `code_challenge` was sent; wrong verifier → `invalid_grant`.
- [ ] Authorization codes are single-use and short-lived; replay → `invalid_grant` + issued tokens revoked.
- [ ] `alg=none` and HS/RS confusion tokens are rejected at verification; unknown `kid` → rejected.
- [ ] `redirect_uri` is exact-match against registered URIs; mismatch → `invalid_request`, no redirect.
- [ ] `state` is required and echoed; `nonce` required + echoed when `id_token` requested.
- [ ] Refresh-token rotation: replaying a rotated token revokes the whole family.
- [ ] `/introspect` and `/revoke` require client auth; revoked tokens introspect as `active:false`.
- [ ] Error responses use the RFC 6749 error codes (`invalid_request`, `invalid_client`, `invalid_grant`, `unauthorized_client`, `unsupported_grant_type`, `invalid_scope`).
- [ ] `userinfo` returns claims consistent with the granted scopes and the id_token `sub`.

## JWKS & key rotation

- [ ] `jwks_uri` serves current public keys; each token's `kid` resolves in JWKS.
- [ ] After a signing-key rotation, both the retiring and new `kid` validate during the overlap window (see the Key Rotation section of the operator runbook).
