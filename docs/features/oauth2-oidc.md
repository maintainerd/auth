# OAuth 2.0 & OpenID Connect

> A from-scratch (no third-party OAuth library) multi-tenant OAuth 2.1 / OpenID Connect Provider: authorization-code+PKCE, refresh, client-credentials, device, token-exchange and CIBA grants, plus discovery, JWKS, revocation, introspection, UserInfo, PAR, DCR and RP/back-channel logout.

| | |
|---|---|
| **Status** | Implemented. A few surfaces are deliberately narrowed (introspection & DCR are control-plane only; `request`/JAR is refused; mTLS client auth is registry-accepted but unimplemented). See notes inline. |
| **Code** | `internal/oauth` (handlers, services, repositories, models, routes); token minting/verification in `internal/platform/jwt`; DPoP in `internal/platform/dpop`; per-tenant policy in `internal/secpolicy` |
| **Endpoints** | `/.well-known/openid-configuration`, `/.well-known/oauth-authorization-server`, `/.well-known/jwks.json` (root); `/api/v1/oauth/{authorize,token,revoke,userinfo,par,device_authorization,ciba,end_session,callback/{idp},consent,...}` (public :8081); `/api/v1/oauth/{introspect,signing-keys,register}` (internal :8080) |
| **Storage** | `oauth_authorization_codes`, `oauth_refresh_tokens`, `oauth_consent_grants`, `oauth_consent_challenges`, `oauth_par_requests`, `oauth_device_codes`, `oauth_ciba_requests`, `oauth_broker_sessions`, `oauth_authorize_requests`, `oauth_token_revocations`, `oauth_token_exchanges`, `oauth_dpop_nonces`, `signing_keys` (migrations `061`–`073`); OAuth columns on `clients` (`019`) |
| **Config** | `APP_PUBLIC_HOSTNAME` (the issuer, and the base for every advertised endpoint); per-tenant `security_settings` (session/token/MFA policy); per-client columns on `clients` |

## Overview

`internal/oauth` is a self-authored OAuth 2.0 / OIDC authorization server (no `ory/fosite` or similar). It is **API-only**: the authorization endpoint never renders HTML — it returns JSON (`login_required`, `consent_required`, a `redirect_uri`, or a `consent_challenge`) that the hosted identity SPA consumes. All external artifacts are anchored to the deployment's public hostname.

The server is split across two HTTP planes (`routes.go`):

- **Public plane (:8081)** — everything a relying party or user-agent touches: `/authorize`, `/token`, `/revoke`, `/userinfo`, `/par`, `/device_authorization`, `/ciba`, `/end_session`, `/callback/{idp}`, consent, connections, back-channel logout. Public endpoints are mounted under `/api/v1/oauth`; discovery + JWKS are at the router root.
- **Internal control plane (:8080, VPN-only)** — `/introspect` (RFC 7662), the signing-key lifecycle (`/signing-keys...`), and Dynamic Client Registration (`/register`, RFC 7591/7592). Mounting DCR and introspection here (not publicly) is a deliberate blast-radius decision — see `routes.go:156` and `handler_discovery.go:38`.

Grants supported (`handler_discovery.go:50`): `authorization_code` (+PKCE), `refresh_token`, `client_credentials`, `urn:ietf:params:oauth:grant-type:device_code` (RFC 8628), `urn:ietf:params:oauth:grant-type:token-exchange` (RFC 8693), `urn:openid:params:grant-type:ciba` (OIDC CIBA). Implicit and ROPC are **not** implemented (OAuth 2.1).

## How it works

### Endpoint → RFC map

| Endpoint | Method | Plane | RFC / spec | Handler |
|---|---|---|---|---|
| `/api/v1/oauth/authorize` | GET | public | RFC 6749 §4.1.1 | `handler_authorize.go:45` |
| `/api/v1/oauth/token` | POST | public | RFC 6749 §4.1.3/§6/§4.4 (dispatch by `grant_type`) | `routes.go:111`, `handler_token.go:41` |
| `/api/v1/oauth/revoke` | POST | public | RFC 7009 | `handler_token.go:142` |
| `/api/v1/oauth/introspect` | POST | **internal** | RFC 7662 | `handler_token.go:180` |
| `/api/v1/oauth/userinfo` | GET | public (JWT) | OIDC Core §5.3 | `handler_userinfo.go:21` |
| `/api/v1/oauth/consent/{challenge_id}` / `/consent` | GET / POST | public (JWT) | (API for the identity SPA) | `handler_authorize.go:184`/`:208` |
| `/api/v1/oauth/authorize/continue` | POST | public (JWT) | signup continuation | `handler_authorize.go:241` |
| `/api/v1/oauth/par` | POST | public | RFC 9126 | `handler_par.go:20` |
| `/api/v1/oauth/device_authorization` | POST | public | RFC 8628 §3.1 | `handler_device.go:21` |
| `/api/v1/oauth/device` / `/device/deny` | POST | public (JWT) | RFC 8628 §3.3 | `handler_device.go:49`/`:107` |
| `/api/v1/oauth/ciba` | POST | public | OIDC CIBA §7 | `handler_ciba.go:21` |
| `/api/v1/oauth/ciba/approve` / `/deny` | POST | public (JWT) | OIDC CIBA | `handler_ciba.go:76`/`:106` |
| `/api/v1/oauth/end_session` | GET/POST | public (JWT) | OIDC RP-Initiated Logout | `handler_session.go:22` |
| `/api/v1/oauth/logout/backchannel` | POST | public | OIDC Back-Channel Logout §2.5 | `handler_session.go:70` |
| `/api/v1/oauth/callback/{idp_identifier}` | GET | public | broker (upstream IdP return leg) | `service_broker.go` |
| `/api/v1/oauth/register` / `/register/{client_id}` | POST / GET | **internal** | RFC 7591 §3 / RFC 7592 §2.1 | `handler_register.go:29`/`:57` |
| `/api/v1/oauth/signing-keys[...]` | GET/POST | **internal** | key lifecycle | `handler_signing_key.go` |
| `/.well-known/openid-configuration` | GET | public | OIDC Discovery 1.0 | `handler_discovery.go:28` |
| `/.well-known/oauth-authorization-server` | GET | public | RFC 8414 | `handler_discovery.go:85` |
| `/.well-known/jwks.json` | GET | public | RFC 7517 | `handler_discovery.go:144` |

### Authorization Code + PKCE (the primary interactive flow)

1. **`GET /authorize`** is session-aware (`routes.go:73`, `OptionalUserContextMiddleware`). The DTO is parsed (`handler_authorize.go:46`) and validated. If a `request_uri` (PAR) is present, the pushed request **replaces** all query params wholesale (`handler_authorize.go:74`); if a `request` (JAR) object is present it is **refused** (`request_not_supported`) rather than silently ignored.
2. If `idp_hint` is set, the broker leg to an upstream IdP starts instead (`handler_authorize.go:90`, `service_broker.go`).
3. **No session** → the server validates the client + `redirect_uri` (via `PrepareAuthorize`) and returns `login_required` (with an opaque `request_id`) so the identity SPA can render login and re-drive the request; `screen_hint=signup` additionally sets a browser-binding cookie for `POST /authorize/continue` (`handler_authorize.go:113`).
4. **With session** → `oauthAuthorizeService.Authorize` (`service_authorize.go:234`) runs the checks below and either returns a `consent_challenge` or a fully-built `redirect_uri` carrying `code` + `state`.

`Authorize` validation order (`service_authorize.go:243`–`:395`):

| # | Check | Evidence |
|---|---|---|
| 1 | Client resolved + `status = active` | `:243`/`:250` |
| 2 | **Tenant binding**: request-host tenant (Origin/Host) is authoritative; hard-fails on mismatch, no system-tenant fallback | `:267` |
| 3 | PKCE required per effective token policy (`require_pkce`, default true) | `:277`, `pkce_policy.go` |
| 4 | MFA/step-up: `RequiredACR="2"` demands a fresh `acr=2` token within the step-up TTL | `:286` |
| 5 | `acr_values` / `max_age` enforced against the session (fails closed) | `:313`, `:411` |
| 6 | Client has `authorization_code` grant | `:319` |
| 7 | `response_type` enabled for client | `:325` |
| 8 | Requested `scope` allowed for client | `:330`, `foundation.go:146` |
| 9 | `redirect_uri` **exact match** against registered URIs | `:336`, `validateRedirectURI:799` |
| 10 | `state` **required** (CSRF, RFC 6749 §10.12) | `:342` |
| 11 | `nonce` required if `response_type` contains `id_token` | `:348` |
| 12 | Consent needed? → issue `consent_challenge` (or `consent_required` if `prompt=none`); else issue code | `:354`/`:361`/`:379` |

The authorization code is a 32-byte random string; only its `HashAuthorizationCode` is stored (`issueAuthorizationCode:910`). TTL **10 min**, single-use, and it carries `user_session_uuid` so tokens minted from it get a `sid` claim (`service_authorize.go:29`, migration `061`).

5. **`POST /token` (grant `authorization_code`)** — `exchangeAuthorizationCode` (`service_token.go:130`):
   - Authenticate the client (`authenticateOAuthClient`).
   - Load code by hash; **reuse** (already `Used`) → log critical `TokenReuse`, deny the JTI of the access token that code minted **and** revoke the user/client refresh tokens, return `invalid_grant` (`:166`).
   - Expiry, client binding, tenant binding, scope-allow, exact `redirect_uri` re-match.
   - **PKCE**: a code carrying a `code_challenge` requires a matching `code_verifier` (S256 only); a **public** client that somehow has a code with no challenge is rejected (PKCE mandatory, RFC 9700) (`:242`).
   - Mark code used, resolve `sub` from `user_identities` (read-only — OAuth never creates identities, `:1062`), resolve requested audience (RFC 8707, `api_audience.go`), mint tokens.

### Refresh (`refresh_token`)

`exchangeRefreshToken` (`service_token.go:317`). Tokens are opaque 32-byte randoms stored as SHA-256 hashes with **family tracking** (`family_id`, migration `062`).

- **Reuse of an already-revoked token** (`:351`):
  - **In-window** (within `refresh_token_reuse_interval_seconds`, default **10s**, of revocation) → returns `invalid_grant` **without** revoking the family. This tolerates a client's concurrent/retried refresh of a just-rotated token without nuking the session.
  - **Out-of-window** (or interval `0`) → logs critical `TokenReuse` and **revokes the entire family** (`RevokeByFamily`), returns `invalid_grant`. *(Note: the code returns `invalid_grant` on in-window duplicates — it does not replay a cached token response.)*
- Expiry, client binding, tenant binding.
- **DPoP binding** (RFC 9449 §5): if the stored token has a `dpop_jkt`, the caller must present a proof over the **same** key (constant-time compare); a mismatch is treated as theft and revokes the family (`:394`).
- **Rotation** (default on, `rotate_refresh_tokens=true`): in one DB transaction, revoke the presented token and mint a new one **in the same family** carrying the same `dpop_jkt` and `user_session_uuid`; a narrower requested `scope` must be a subset of the grant. When rotation is disabled, the incoming refresh token is reused and only access/ID tokens are re-minted (`:428`–`:505`).
- A refresh token is only minted at authorization time when `offline_access` scope was granted (`generateTokens:1014`, RFC 6749 §1.5).

### Client Credentials

`exchangeClientCredentials` (`service_token.go:536`): confidential/m2m only; access token only (no refresh, no ID token). `sub`/`aud` derive from the client identifier (or the service name); RFC 8707 lets an m2m client name a registered API as `aud` (`resolveRequestedAudience`), and inherited + direct permissions are resolved into a `permissions` claim (`:591`).

### Token minting

`generateTokens` (`service_token.go:937`) → `internal/platform/jwt`:

- **Access token** — JWT (RFC 9068 profile), **RS256 only**, `kid`-selected key. `aud` = the requested API (RFC 8707) or the client; `iss` = `TokenIssuer` (below). Real session `acr`/`amr`/`auth_time` are carried from the session row (never hardcoded pwd/acr=1); `sid` stamps the browser session so a single-session logout is possible. `token_type` becomes `"DPoP"` and a `cnf.jkt` is bound when a DPoP proof was presented; otherwise `"Bearer"`.
- **ID token** — JWT RS256; `at_hash` binds it to the access token, `nonce` echoed, `auth_time`/`acr`/`amr`/`sid` set. Operator `claim_mappers`/`scope_claim_mappings` are merged but reserved names are stripped (`jwt.SanitizeClientClaimMappers`, `service_token.go:1188`).
- Response headers: `Cache-Control: no-store`, `Pragma: no-cache` (`handler_token.go:277`).

**Issuer** (`internal/platform/jwt/issuer.go:24`): `iss` = `APP_PUBLIC_HOSTNAME` (the authorization server), falling back to the client domain only when that env var is unset. It is **not** a legacy `ISSUER_URL`. This matches the `issuer` in discovery so spec-compliant RPs accept the token (OIDC Core §3.1.3.7).

### Revocation & Introspection

- **`POST /revoke`** (RFC 7009, `service_token.go:640`): client-authenticated; tries refresh-token revoke first (client-bound), then denies the access-token JTI in the Redis denylist for its remaining TTL (`revokeAccessToken:683`, handles `token_type` `DPoP` too). Always 200.
- **`POST /introspect`** (RFC 7662, internal plane, `service_token.go:753`): client-authenticated. **Tenant-scoped** — a caller may only introspect its own tokens or tokens belonging to a client in its own tenant; anything else is reported `active:false` (not an error), since all tenants share one signing key (`callerMayIntrospect:853`). Validates JWTs and falls back to refresh-token lookup.

### Other grants (brief)

- **Device (RFC 8628)** — `POST /device_authorization` mints `device_code` + `user_code` (TTL **15 min**, `service_device.go:27`); user approves at `POST /device`; client polls `POST /token` with `grant_type=...:device_code` (`handler_device.go`, `service_device.go`).
- **Token Exchange (RFC 8693)** — `POST /token` with `grant_type=...:token-exchange` (`handler_token_exchange.go:21`). Also front-doors **workload identity federation** (§3.21): a keyless external OIDC JWT (`subject_token_type=...:jwt`, no client creds) is tried against configured federations first; `audience`/`resource` are collapsed to one target before minting; otherwise it falls through to the standard client-authenticated exchange.
- **CIBA (OIDC)** — `POST /ciba` returns `auth_req_id`; user approves via `POST /ciba/approve`; client polls `POST /token` with `grant_type=urn:openid:params:grant-type:ciba`. Poll delivery mode only (`handler_discovery.go:120`).
- **PAR (RFC 9126)** — `POST /par` mints a `request_uri`; `/authorize` consumes it once, and the on-the-wire `client_id` must match the pushed one (`handler_authorize.go:157`). PAR is refused if the service isn't wired (fails closed).
- **DCR (RFC 7591/7592)** — internal plane, behind `client:create`/`client:read` permissions. Only `authorization_code`/`refresh_token`/`client_credentials` grants are registerable (`service_register.go:37`); ≥1 redirect URI required; dangerous redirect schemes rejected at registration; tenant is the authenticated caller's.
- **RP-Initiated & Back-Channel Logout** — `/end_session` clears auth cookies and honors `post_logout_redirect_uri`; `/logout/backchannel` consumes a logout token with replay protection (`service_session.go`).

## Implementation

- **Routing**: `routes.go` — `OAuthPublicRoute` (:8081), `OAuthDiscoveryRoute` (root), `OAuthInternalRouteWithRegistration` (:8080). The internal mount **panics** if any internal handler is nil, and the thin `OAuthInternalRoute`/`...WithKeys` wrappers were deleted so a half-wired control plane won't compile (`routes.go:190`–`:288`).
- **Token endpoint dispatch**: `routes.go:111` branches on `grant_type` to the exchange/device/ciba handlers, defaulting to `deliverAuthCookies(... tokenHandler.Token)` (which additionally sets httpOnly session cookies for the admin console).
- **Handlers**: `handler_authorize.go`, `handler_token.go`, `handler_token_exchange.go`, `handler_device.go`, `handler_ciba.go`, `handler_par.go`, `handler_register.go`, `handler_session.go`, `handler_userinfo.go`, `handler_discovery.go`, `handler_signing_key.go`, `handler_consent.go`, `handler_connections.go`, `handler_callback.go`, `handler_introspect_grpc.go` (gRPC :50051 introspection).
- **Services**: `service_authorize.go`, `service_token.go`, `service_token_exchange.go`, `service_device.go`, `service_ciba.go`, `service_par.go`, `service_register.go`, `service_session.go`, `service_backchannel_logout.go`, `service_broker.go`, `service_consent.go`, `service_signing_key.go`, `service_workload_federation.go`, `service_security_policy.go`, `service_token_code_index.go`.
- **Client auth** (`authentication.go`): `client_secret_basic`, `client_secret_post`, `none` (public clients only — `TokenAuthMethodNone` is rejected for confidential/m2m at redemption, `:43`), `private_key_jwt` (RFC 7523; RS256/384/512; JWKS or `jwks_uri`; ≥2048-bit RSA; single-use `jti` replay guard; ≤5-min assertion lifetime; `aud` must be the AS, exact-match), `client_secret_jwt` (HS256/384/512). `tls_client_auth`/`self_signed_tls_client_auth` are registry-accepted but **explicitly refused at runtime** (unimplemented, `:54`).
- **Constants** (`foundation.go`): grant types, auth methods, and the sessionless subject-type labels (`device`/`ciba`/`exchange`). Scope authorization (`validateClientAllowedScopes:146`): an empty client allowlist means the **baseline OIDC scopes only** (`openid profile email phone address offline_access`), never "all"; unknown scopes are rejected.
- **Discovery** (`handler_discovery.go`) is code-driven off the same constants (grant types, `code_challenge_methods=[S256]`, `id_token_signing_alg_values=[RS256]`, `token_endpoint_auth_signing_alg_values` derived from what client-assertion auth actually accepts). `request_parameter_supported=false`, `request_uri_parameter_supported=true`. `introspection_endpoint` and `registration_endpoint` are **omitted** because those routes are control-plane only (`:38`, `types.go:207`).
- **JWKS** (`handler_discovery.go:144`) publishes the **union** of DB-backed `signing_keys` and the in-memory key store (deduped by `kid`), so a token signed by the in-memory rotation runner never carries a `kid` absent from JWKS.

### Storage

| Table | Migration | Purpose / key columns |
|---|---|---|
| `oauth_authorization_codes` | `061` | `code_hash` (unique), PKCE `code_challenge`/`method` (CHECK `S256`), `scope[]`, `nonce`, `user_session_uuid`, `used`, `expires_at` |
| `oauth_refresh_tokens` | `062` | `token_hash` (unique), `family_id`, `scope[]`, `is_revoked`/`revoked_at` (CHECK paired), `acr`/`amr`, `dpop_jkt`, `user_session_uuid`, `expires_at` |
| `oauth_consent_grants` | `063` | one row per (user, client) with granted `scopes` |
| `oauth_consent_challenges` | `064` | pending consent (10-min TTL) |
| `oauth_par_requests` | `065` | pushed request objects, single-use |
| `oauth_device_codes` | `066` | device/user codes |
| `oauth_ciba_requests` | `067` | CIBA auth requests |
| `oauth_broker_sessions` | `068` | upstream-IdP broker state (single-use `idp_state`) |
| `oauth_authorize_requests` | `069` | pending `/authorize` (login/signup continuation) |
| `oauth_token_revocations` | `070` | access-token JTI revocation records |
| `oauth_token_exchanges` | `071` | RFC 8693 exchange records |
| `oauth_dpop_nonces` | `072` | DPoP server-nonce store (ephemeral) |
| `signing_keys` | `073` | RS/ES/EdDSA key rows; `private_key_encrypted` (BYTEA, KEK-wrapped), `status` (active/retired/compromised) |

OAuth columns live on `clients` (migration `019`): `token_endpoint_auth_method`, `grant_types[]`, `response_types[]`, `access_token_ttl`, `refresh_token_ttl`, `require_consent`, `allowed_scopes`, `require_pkce`, `required_acr`, `jwks`/`jwks_uri`, `claim_mappers`, `scope_claim_mappings`, DPoP-required flag, and the previous-secret grace columns.

Access-token JTI revocation is a Redis denylist (`cache.JTIDenylister`), keyed to each token's remaining TTL — not a DB scan.

## Configuration

| Setting | Where | Effect / default |
|---|---|---|
| `APP_PUBLIC_HOSTNAME` | env (`config.go:156`) | The `iss` claim and the base URL for every advertised endpoint and client-assertion audience. Client-assertion auth **fails closed** if unset. |
| `access_token_ttl_minutes` | tenant `security_settings.session` | Access token TTL, default **15 min** (`secpolicy/defaults_setting.go:42`) |
| `refresh_token_ttl_days` | tenant `security_settings.session` | Refresh token TTL, default **30 days** (`:43`; hard fallback `jwt.RefreshTokenTTL` = 7 days when policy is 0) |
| `rotate_refresh_tokens` | tenant `security_settings.session` | Rotate-on-use, default **true** (`:47`) |
| `refresh_token_reuse_interval_seconds` | tenant `security_settings.session` | In-window duplicate-use grace before family revocation, default **10s** (`:48`) |
| `require_pkce` | tenant `security_settings.token` / per-client | PKCE required, default **true** (`:59`) |
| `signing_algorithm` | tenant `security_settings.token` | ID-token alg, default **RS256** (`:58`) |
| `required_acr` | tenant / per-client | `"2"` enforces MFA/step-up at `/authorize` |
| Per-client overrides | `clients` columns | `access_token_ttl`, `refresh_token_ttl`, `grant_types`, `response_types`, `allowed_scopes`, `require_consent`, `require_pkce`, `required_acr`, `token_endpoint_auth_method`, `jwks`/`jwks_uri`, claim mappers |

A per-tenant `security_settings` row supplies session/token/MFA policy; effective values resolve as tenant policy ∩ client overrides (`service_security_policy.go:15`, `secpolicy/validation_setting.go:467`). The `/token` endpoint is rate-limited when a limiter is wired (`routes.go:132`).

## Security considerations

- **PKCE** S256-only; mandatory for public clients and by default for all (`pkce_policy.go`, `validation_token.go`, RFC 9700). `plain` is rejected.
- **Exact redirect-URI matching** (`validateRedirectURI:799` → `client.MatchClientRedirectURI`); no wildcards. `state` is echoed via encoded query params so a `state` containing `&`/`#` cannot re-partition the callback (`buildAuthCodeRedirect:953`).
- **Authorization-code single use**: replay denies the minted access token's JTI and revokes the family's refresh tokens, and logs a critical event (`service_token.go:166`, `service_token_code_index.go`).
- **Refresh rotation with family revocation** on out-of-window reuse or DPoP-key mismatch; a bounded in-window grace tolerates client retries without revoking the family (`service_token.go:349`).
- **DPoP (RFC 9449)**: any presented proof is validated (no silent bypass), access tokens get `token_type:DPoP` + `cnf.jkt`, refresh tokens are key-bound, and clients with `dpop_required` face the §8 server-nonce gate (`handler_token.go:69`, `:112`).
- **Client-assertion hardening (RFC 7523)**: `aud` must exactly equal the AS issuer/endpoints (no prefix match), lifetime ≤5 min, `iss`/`sub` must equal `client_id`, single-use `jti`, ≥2048-bit RSA (`authentication.go:206`).
- **Tenant isolation**: `/authorize` binds `client_id` to the request-host tenant (server-derived, hard-fail); introspection refuses cross-tenant tokens despite the shared signing key (`callerMayIntrospect:853`).
- **JWTs are RS256 only**; `alg=none` and HS/RS confusion are rejected at verification; unknown `kid` fails (`internal/platform/jwt`, `conformance.md`). ID-token/claim-mapper reserved names are stripped so an operator mapper can't forge `sub`/`exp`/`permissions`.
- **Scope floor, not blanket**: an empty client allowlist grants only baseline OIDC scopes; unknown scopes are `invalid_scope` (`foundation.go:146`).
- **Least disclosure**: refresh tokens and auth codes are stored hashed; introspection returns `active:false` (never a reason) for invalid/foreign tokens; discovery advertises only publicly reachable endpoints.
- **Control-plane containment**: DCR and introspection are :8080-only (VPN); a nil internal handler panics rather than silently shipping an unreachable surface (`routes.go:214`).
- **Not implemented / narrowed** (honest gaps): JAR `request` objects (`request_not_supported`), mTLS client auth (`tls_client_auth`/`self_signed_tls_client_auth` refused at runtime), CIBA `ping`/`push` delivery (poll only), a public introspection/registration endpoint.

## Related

- `./cryptography-and-keys.md` — DPoP proof validation, nonce gate, sender-constrained tokens (`internal/platform/dpop`)
- `./cryptography-and-keys.md` — RS256 minting/verification, `kid` rotation, the `signing_keys` lifecycle & JWKS
- `./clients.md` — the `clients` model, auth methods, redirect URIs, per-client OAuth policy
- `./security-settings.md` — per-tenant `security_settings` (session/token/MFA) and effective-policy resolution
- `./federation.md` — the broker leg (`idp_hint`, `/callback/{idp}`) and workload identity federation
