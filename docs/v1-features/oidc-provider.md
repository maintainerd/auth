# OIDC Provider Compatibility

## What this is

This feature makes maintainerd/auth act as a **standards-compliant OpenID Connect Provider**.
It is not needed for the auth-console or auth-identity frontends — those are first-party consumers
that use the direct REST API.

This becomes relevant when a **third-party application** wants to use maintainerd/auth as
its identity provider. Examples:

- A customer's internal app wants "Login with Maintainerd Auth"
- A mobile SDK (iOS/Android) uses OIDC auto-discovery to configure itself
- Another microservice needs to validate JWTs without being hardcoded to this service's key URL
- An enterprise SSO integration expects to consume a standard OIDC metadata document

Without this, those consumers must hardcode endpoint URLs and manually distribute the public
signing key — which breaks every time the key rotates or the URL changes.

## How OIDC Discovery works

As defined in OpenID Connect Discovery 1.0 (RFC 5785):

```
GET /.well-known/openid-configuration
```

Returns a JSON document that tells any OIDC client everything it needs:

```json
{
  "issuer": "https://tenant-name.auth.maintainerd.com",
  "authorization_endpoint": "https://tenant-name.auth.maintainerd.com/auth/authorize",
  "token_endpoint": "https://tenant-name.auth.maintainerd.com/auth/token",
  "userinfo_endpoint": "https://tenant-name.auth.maintainerd.com/auth/me",
  "end_session_endpoint": "https://tenant-name.auth.maintainerd.com/auth/logout",
  "jwks_uri": "https://tenant-name.auth.maintainerd.com/.well-known/jwks.json",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token"],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["RS256"],
  "scopes_supported": ["openid", "profile", "email"]
}
```

The JWKS endpoint exposes the RSA public key used for JWT signing:

```
GET /.well-known/jwks.json
```

```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "key-id",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

Any resource server or third-party SDK can call `jwks.json` to fetch the public key and
validate JWTs locally — without making a call to maintainerd/auth on every request.

## Checklist

### Phase A — JWKS Endpoint (prerequisite, useful standalone)

Even without full OIDC discovery, the JWKS endpoint is immediately useful: any downstream
service that validates our JWTs benefits from it.

- [ ] Create `GET /.well-known/jwks.json` on port **8081** (public, no auth)
  - Read the RSA public key from the same source as `internal/jwt` package
  - Format as RFC 7517 JWK Set (`{ "keys": [ { ... } ] }`)
  - Include: `kty`, `use`, `alg`, `kid`, `n`, `e` fields
  - Do **not** include `d`, `p`, `q` or any private key material
- [ ] Add `WellKnownHandler` in `internal/rest/handler/well_known.go`
  - `GetJWKS(w, r)` method
- [ ] Add route in `internal/rest/route/well_known.go`
  - Register under `/.well-known/jwks.json` with no auth middleware
  - Register on public server only (port 8081)
- [ ] Set `Cache-Control: public, max-age=3600` response header
- [ ] Cache response in Redis (`well_known:jwks`, TTL 1 hour)
  - Invalidate on JWT key rotation (admin-triggered endpoint or config reload)
- [ ] Unit tests for `GetJWKS` (table-driven, 100% statement coverage)
  - Case: public key loads successfully
  - Case: key parse failure → 500 with no key material leaked in response

### Phase B — OIDC Discovery Endpoint

- [ ] Create `GET /.well-known/openid-configuration` on port **8081** (public, no auth)
  - All URLs in the response must use the same base URL as the request host
    (use `r.Host` or a configured `ISSUER_URL` env var — not hardcoded)
  - Must be valid per OpenID Connect Discovery 1.0 spec
  - `issuer` value must exactly match the `iss` claim in JWTs issued by `internal/jwt`
- [ ] Add `GetOpenIDConfiguration(w, r)` to `WellKnownHandler`
- [ ] Add `ISSUER_URL` environment variable to config
  - Example: `https://tenant-name.auth.maintainerd.com`
  - Used in both the OIDC document and the `iss` JWT claim
  - Document in `docs/contributing/environment-variables.md`
  - Document in `docs/deployment/environment-variables.md`
- [ ] Verify that `internal/jwt` package uses `ISSUER_URL` as the `iss` claim
  - If it currently hardcodes the issuer, update it to read from config
- [ ] Set `Cache-Control: public, max-age=300` response header
- [ ] Cache response in Redis (`well_known:openid_configuration`, TTL 5 minutes)
- [ ] Unit tests for `GetOpenIDConfiguration`
  - Case: issuer URL set in config → correct response
  - Case: all required fields present (issuer, authorization_endpoint, token_endpoint,
    jwks_uri, response_types_supported, subject_types_supported,
    id_token_signing_alg_values_supported)

### Phase C — Authorization Code Flow (full OIDC)

This is what's needed to actually issue OIDC authorization codes to third-party apps.
This is significantly more work and should be scoped as its own feature after Phase A and B.

- [ ] `GET /auth/authorize` — OIDC authorization endpoint (redirect-based)
  - Accepts `response_type=code`, `client_id`, `redirect_uri`, `scope`, `state`, `nonce`
  - Validates `redirect_uri` against registered URIs on the `clients` table
  - Issues short-lived authorization code (opaque, stored in Redis, TTL 60s)
- [ ] `POST /auth/token` — token endpoint
  - Accepts `grant_type=authorization_code`, exchanges code for `id_token` + `access_token`
  - Accepts `grant_type=refresh_token`
  - `id_token` must include: `iss`, `sub`, `aud`, `exp`, `iat`, `nonce`
- [ ] `GET /auth/me` (UserInfo endpoint)
  - Returns profile claims for the authenticated user (`name`, `email`, `picture`)
  - Protected by Bearer token (the `access_token` from the token endpoint)
- [ ] `POST /auth/logout` (end session endpoint)
  - Accepts `id_token_hint`, optionally `post_logout_redirect_uri`
  - Revokes the session
- [ ] Dynamic or static client registration
  - For now: static — clients pre-registered via admin API
  - Each client record needs `redirect_uris []string` and `allowed_scopes []string`
- [ ] PKCE support (`code_challenge` + `code_challenge_method=S256`) — required for public clients (SPAs, mobile)

## Implementation order

1. Phase A (JWKS) — small, standalone, immediately useful for JWT validation by downstream services
2. Phase B (OIDC discovery) — builds on A, unblocks SDK auto-configuration
3. Phase C (Authorization Code Flow) — full OIDC, own feature cycle

## Do not implement in v1

- Implicit flow (deprecated, insecure — RFC 9700)
- Device Authorization Grant (for CLIs/TVs — future)
- Client Credentials Grant (machine-to-machine — could be useful but separate feature)
- OIDC Dynamic Client Registration — admin API is sufficient for v1
