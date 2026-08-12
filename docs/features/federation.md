# Federation & External Identity Providers

> Brokering upstream OIDC, generic-OAuth2, and SAML 2.0 identity providers into maintainerd-auth: per-tenant provider config, external-token exchange, JIT provisioning, identity linking, attribute extraction, and home-realm discovery.

| | |
|---|---|
| **Status** | Implemented. OIDC + generic-OAuth2 + SAML 2.0 (incl. SP-initiated Single Logout) are all live. |
| **Code** | `internal/idp` (external-IdP brokering — this doc). Sibling package `internal/federation` is a **separate** subsystem: [Workload Identity Federation](./federation.md). |
| **Endpoints** | `POST /federation/token`, `POST /federation/oauth2/callback`, `GET /federation/hrd`, `GET/POST /federation/saml/*`, `GET/POST/DELETE /account/identities*`, `GET/POST/PUT/DELETE /identity_providers*`, `POST /identity_providers/test` |
| **Storage** | `identity_providers`, `identity_provider_email_domains`, `identity_provider_allowed_audiences`, `client_identity_providers`, `user_identities` (migrations 016–018, 023, 030) |
| **Config** | Env: `APP_PUBLIC_HOSTNAME`, at-rest encryption key, HMAC secret key. Per-provider row + `config` JSONB (see [Configuration](#configuration)) |

## Overview

Federation lets a tenant delegate authentication to an external identity provider and receive a first-party maintainerd session in return. An operator registers an `IdentityProvider` row (Cognito, Auth0, Google, Microsoft/Entra, GitLab, LinkedIn, GitHub, Facebook, X/Twitter, another maintainerd instance, or a generic SAML 2.0 IdP), connects it to one or more OAuth clients, and end users then sign in — or link an additional identity — through that provider.

Three protocol families are handled by one service (`federationService` in `internal/idp/service_federation.go`), differing only in how the upstream identity is obtained:

| Family | Providers | Identity source | Verification |
|--------|-----------|-----------------|--------------|
| **OIDC** | cognito, auth0, google, microsoft, linkedin, gitlab, external maintainerd (enterprise) | upstream `id_token` (JWT) | `go-oidc` discovery + JWKS signature + issuer/aud/nonce |
| **OAuth2-only** | github, facebook, twitter | userinfo endpoint (no `id_token` issued) | PKCE + single-use broker session; profile fetched server-side |
| **SAML 2.0** | saml (enterprise) | signed SAML assertion | `crewjam/saml` XML-signature against the IdP certificate |

The provider's differences (no-OIDC quirks, provider-specific scopes, claim normalization, second-fetch email augmentation, token auth style) are centralized in one registry — `profileFor()` in `internal/idp/provider_profiles.go` — so adding a provider is a single edit rather than scattered conditionals.

A note on direction: this feature is maintainerd **consuming** external IdPs. The reverse (maintainerd acting as an OpenID Provider / authorization server for third-party apps) is the OAuth authorization-code server in `internal/oauth` — see [OAuth Authorization](./oauth2-oidc.md).

## How it works

There are five entry paths into federation. All converge on the shared provision/link core (`provisionUser` → `user_identities`), then either mint first-party tokens or return a resolved user to a caller that mints them.

### 1. Direct OIDC token exchange — `POST /federation/token`

For frontends that already hold an upstream `id_token` and want a maintainerd session.

1. Look up the active `IdentityProvider` by `provider_identifier`; require `status = active`.
2. Verify the `external_token` via OIDC discovery against the provider's `issuer` + `provider_client_id` (`validateOIDCToken`).
3. Extract claims → `IdentityMetadata` using the provider's `attribute_mapping`.
4. Resolve the OAuth client via the `client_identity_providers` connection (must be an **enabled** connection to *this* provider — no fallback to the tenant default).
5. Find the existing `user_identities` row by `(tenant, provider, sub)`, or JIT-provision (see [Provisioning](#provisioning--jit)).
6. Mint first-party access + ID + refresh tokens (`generateTokens`) and open a session.

### 2. Generic OAuth2 code callback — `POST /federation/oauth2/callback`

For OAuth2 providers where the client received an authorization `code`:

1. Look up provider; decrypt its client secret (fails **closed** — an undecryptable secret aborts, never POSTs ciphertext upstream).
2. Exchange the code at the token endpoint (`resolveTokenEndpoint`: explicit `token_endpoint` → OIDC discovery → issuer-based default).
3. Fetch the profile from the userinfo endpoint; unmarshal to claims (response body capped at 1 MiB).
4. Extract metadata, resolve the client, provision/link, mint tokens (same core as path 1).

### 3. Brokered browser sign-in (redirect flow)

The full redirect login is driven by the OAuth authorization server in `internal/oauth`, which calls two `FederationService` methods:

1. `ResolveBrokerProvider(idpIdentifier)` returns the upstream authorize endpoint, `client_id`, and resolved scopes (`brokerScopesOrDefault` — OIDC providers get their scopes/openid default; OAuth2-only providers get their required provider-specific scopes verbatim). No secrets returned.
2. The browser is redirected to the upstream provider with PKCE (`S256`) + `state` + `nonce`. LinkedIn is the exception — it rejects PKCE alongside a client secret, so its leg omits `code_verifier`.
3. On return to `/api/v1/oauth/callback/{idp}`, `ResolveBrokerUser(...)` runs `exchangeUpstreamCode`: redeem the code, then either validate the returned `id_token` (issuer + audience + nonce) or, for OAuth2-only providers, fetch userinfo. It provisions/links the user and opens a broker session (recording the upstream `sid` as `idp_session_id` for back-channel logout), returning the resolved user — **without** minting tokens. `internal/oauth` mints its own authorization code from that user.

### 4. SAML 2.0 SP-initiated SSO

| Step | Endpoint | Action |
|------|----------|--------|
| Initiate | `GET /federation/saml/initiate` | Build an `AuthnRequest`, validate `redirect_uri` against the client's registered URIs, sign a `RelayState` (HMAC-SHA256, purpose=`sso`, carries provider/client/redirect/tenant/`AuthnRequest.ID`/nonce), redirect to the IdP. |
| ACS | `POST /federation/saml/acs/{provider_identifier}` | Verify RelayState (+ purpose + single-use), parse & signature-verify the Response, bind `InResponseTo` to the exact request ID (`AllowIDPInitiated=false`), extract attributes, provision/link, mint tokens, store them under a **single-use 5-min exchange code**, redirect back with `?code=`. |
| Exchange | `POST /federation/saml/exchange` | Redeem the one-time code for the full `LoginResponseDTO` (tokens never ride in the URL). |
| Metadata | `GET /federation/saml/metadata/{provider_identifier}` | SP metadata XML for the IdP to import. |

SAML SP-initiated and IdP-honoring **Single Logout** is also implemented — `GET/POST /federation/saml/logout` (SP-initiated) and `GET/POST /federation/saml/slo/{provider_identifier}` (consumes the IdP `LogoutResponse`, honors IdP-initiated `LogoutRequest`). Both directions require a valid IdP XML signature. See `service_saml_slo.go` / `saml_slo.go`.

### 5. Direct token federation for API access (resource-server)

`internal/platform/middleware/multi_issuer_middleware.go` lets a resource-server request present an **external IdP token directly** as its bearer credential. When the token's `iss` is not first-party, the middleware looks up the IdP by issuer and accepts it only if `status = active` **and** `allow_token_federation = true`, validates the token audience against `identity_provider_allowed_audiences`, then resolves/JIT-provisions the principal (`ResolveFederatedPrincipal`, `token_use` must be `id`). This is distinct from the login flows above — no maintainerd token is issued; the external token authorizes the call in place.

### Provisioning & JIT

`provisionUser` (`service_federation.go`) is the single enforcement point for every path above:

1. **Email-collision check first.** If the upstream email matches an existing tenant user, provisioning stops and returns `errEmailCollision` — the email is a *discovery hint*, never proof of ownership, so identities are **never silently merged**. The broker flow turns the collision into an account-link confirmation request ([Account linking](#account-linking)); direct-token/SAML flows return `409 Conflict`.
2. **JIT gate.** A genuinely new user is created only if `allow_jit_provisioning = true` for that provider; otherwise `401`.
3. **Registration policy.** New-account creation also honors the tenant registration policy: `blocked_email_domains` is a hard block, and a closed directory (`self_registration_enabled = false`) refuses social/federated creation. (The self-signup *allowlist* is deliberately not applied — federation is a separate admin-configured trust path.)
4. A new `User` (status active) is created with a stable **built-in system identity** (keyed to the tenant's system IdP, not the provider string) plus the external `user_identities` row (`provisioning_source = "jit"`), and the tenant default role is assigned.

**Email-verified trust** (`resolveFederatedEmailVerified`): an explicit upstream `email_verified` is honored verbatim; when omitted, the email is trusted only for `enterprise` and `saml` provider types (admin-configured single-org trust anchors), never for social providers. For SAML specifically, an asserted email is treated as verified only when its domain is on the provider's configured `email_domains` allow-list (`HandleSAMLResponse` / `samlEmailDomainAllowed`).

### Account linking

An authenticated user can attach or remove additional provider identities (`user_identities` rows):

- `GET /account/identities` — list built-in + external identities.
- `POST /account/identities/link` — attach from a pasted upstream `id_token` (OIDC).
- `POST /account/identities/link/start` + `/link/callback` — OAuth2 redirect linking; the start request is bound to the caller's `user_id` (CSRF defense), stored single-use with PKCE + nonce (10-min TTL), and the callback exchanges the code server-side.
- `DELETE /account/identities/{identity_uuid}` — unlink (the built-in system identity can never be unlinked).
- `AdminUnlinkIdentity` — a tenant admin unlinks another user's identity, strictly tenant-scoped (cross-tenant target reported as `NotFound`).

Linking never issues a session and refuses an identity already claimed by a different account (`attachResolvedIdentity`).

### Home-realm discovery — `GET /federation/hrd`

Given an email + `client_id`, resolve the tenant from the client, then return the provider whose `email_domains` list owns the email's domain (single indexed lookup on `identity_provider_email_domains`), falling back to the tenant's default (maintainerd) IdP. The public surface accepts only `client_id`; `tenant_id` is an internal-router fallback.

## Implementation

| Concern | File | Notes |
|---------|------|-------|
| Service interface + all flows | `internal/idp/service_federation.go` | `FederationService`; token exchange, provisioning, HRD, token validation, `generateTokens` |
| Broker code exchange | `internal/idp/service_federation.go:1076` | `exchangeUpstreamCode` (shared by sign-in + linking; nonce checked when the provider echoes it) |
| JIT + collision | `internal/idp/service_federation.go:1499` | `provisionUser`; `resolveFederatedEmailVerified:1482` |
| OIDC token verify | `internal/idp/service_federation.go:1337` | `validateOIDCToken`; Azure multi-tenant `{tenantid}` re-validation `:1388`; slash-tolerant discovery `:1433`; 1-min clock-skew leeway `:45` |
| Federated principal (token federation) | `internal/idp/service_federated_principal.go` | `ResolveFederatedPrincipal`, audience validation |
| Provider registry / quirks | `internal/idp/provider_profiles.go` | `profileFor`; GitHub/Facebook/Twitter claim normalization + email augmentation; LinkedIn `client_secret_post` |
| Identity link store + flow | `internal/idp/service_identity_link.go` | `StartIdentityLink` / `CompleteIdentityLink`; single-use `IdentityLinkRequestStore` |
| SAML SSO | `internal/idp/service_saml.go` | initiate / ACS / exchange / metadata |
| SAML SLO | `internal/idp/service_saml_slo.go`, `saml_slo.go` | SP-initiated + IdP-honoring logout |
| SAML SP/RelayState | `internal/idp/saml_provider.go` | HMAC-signed RelayState (`signRelayState` `:51`), SP URLs, attribute extraction |
| Provider CRUD | `internal/idp/service_provider.go`, `handler_provider.go` | create/update/status/delete; `callback_url` derived from `APP_PUBLIC_HOSTNAME` (`idpCallbackURL`) |
| Secret encryption | `internal/idp/encryption.go`, `model_provider.go` | write-only secret (`***REDACTED***` sentinel); `DecryptedProviderClientSecretStrict` fails closed |
| SSRF-safe HTTP | `internal/idp/http_client.go` | blocks RFC-1918/loopback/link-local/metadata; dials the validated IP, not the hostname |
| Handlers / routes | `internal/idp/handler_federation.go`, `routes.go` | public `/federation/*`; authenticated `/account/identities/*`; step-up-gated `/identity_providers/*` |
| Test connection | `internal/idp/service_federation.go:2132` | OIDC discovery + JWKS probe over the SSRF-safe client |
| Provider by issuer (token federation) | `internal/platform/middleware/multi_issuer_middleware.go:140` | `allow_token_federation` gate + audience allow-list |
| Broker driver (redirect login) | `internal/oauth/service_broker.go`, `handler_callback.go` | calls `ResolveBrokerProvider` / `ResolveBrokerUser` |

**Provider constants** (`internal/shared/constants.go`): providers `maintainerd, cognito, auth0, google, facebook, github, microsoft, linkedin, twitter, gitlab, saml`; types `system, social, enterprise, saml`.

**Storage**

| Table | Migration | Purpose |
|-------|-----------|---------|
| `identity_providers` | 016 | Provider row: `issuer`, `provider_client_id`, `provider_client_secret_encrypted`, `allow_jit_provisioning`, `allow_registration`, `allow_token_federation`, `config` JSONB, `certificate_expires_at`, `status`, `is_default`, `is_system`. Unique `identifier`; issuer unique **per tenant**; one `is_system` provider per tenant. |
| `identity_provider_email_domains` | 017 | Domain → provider map for HRD (one domain per IdP per tenant). |
| `identity_provider_allowed_audiences` | 018 | Allowed token audiences for direct token federation. |
| `client_identity_providers` | 023 | Client ↔ provider connections (`enabled`, `is_default`, `display_order`). |
| `user_identities` | 030 | Per-user identities (built-in + external); unique `(tenant, provider, sub)`; carries `metadata`, `jit_provisioned_at`, `provisioning_source`. |

The `config` JSONB holds the polymorphic fields only — `OIDCProviderConfig` (`scopes`, `attribute_mapping`, `userinfo_endpoint`, `authorization_endpoint`, `token_endpoint`) or `SAMLProviderConfig` (`entity_id`, `sso_url`, `slo_url`, `certificate`, `name_id_format`, `attribute_mapping`). Security-critical/queried fields (issuer, client id/secret, the `allow_*` flags) are promoted to dedicated columns.

> The `internal/idp` package also contains **registration flows** (`/registration_flows`, `service_registration_flow.go`) — self-signup form definitions, adjacent to but not part of federation.

## Configuration

**Environment / global**

| Variable | Used for |
|----------|----------|
| `APP_PUBLIC_HOSTNAME` | Base for SAML SP entity/ACS/metadata/SLO URLs (`samlSPURLs`) and the OAuth broker callback URL (`idpCallbackURL`, `buildBrokerCallbackURL`). Must match what upstream providers are configured to redirect to. |
| At-rest encryption key | `crypto.EncryptAtRest` / `DecryptAtRest` for `provider_client_secret_encrypted`. |
| HMAC secret key (`config.HMACSecretKey`) | Signs/verifies the SAML `RelayState`. |

**Per-provider (row + `config` JSONB), set via the step-up-gated IdP CRUD API:**

| Setting | Effect |
|---------|--------|
| `provider`, `provider_type` | Selects the protocol family and provider quirks. |
| `issuer`, `provider_client_id`, `provider_client_secret` | Upstream OIDC/OAuth2 credentials. Secret is **write-only** (never returned; blank/`***REDACTED***` on update preserves the stored value). |
| `allow_jit_provisioning` | Whether an unknown upstream user may create a new account. |
| `allow_registration` | Registration allowance flag on the provider row. |
| `allow_token_federation` | Whether external tokens from this issuer are accepted directly for API access (multi-issuer middleware). |
| `allowed_audiences` | Audience allow-list enforced for direct token federation. |
| `email_domains` | HRD routing + (SAML) the allow-list that makes an asserted email verified. |
| `config.scopes` / `attribute_mapping` / endpoints | OAuth scopes, claim→field mapping, and explicit endpoints (else derived via discovery). |
| `config.entity_id/sso_url/slo_url/certificate/name_id_format` | SAML trust config (`certificate` populates `certificate_expires_at`). |
| `status` (`active`/`inactive`), `is_default` | Only active providers participate in any flow; the default is HRD's fallback. |

## Security considerations

- **SSRF protection.** All outbound IdP calls (discovery, code exchange, userinfo, JWKS, test-connection) go through `idpHTTPClient`, which resolves the target and refuses loopback, link-local/cloud-metadata (`169.254.169.254`), RFC-1918, CGN, multicast, and reserved ranges, then dials the **validated IP** (not the hostname) to defeat DNS rebinding. Responses are capped at 1 MiB.
- **Secret handling.** Upstream client secrets are encrypted at rest and never returned by any read/list endpoint. Broker/exchange paths use `DecryptedProviderClientSecretStrict`, which fails closed rather than POSTing an undecryptable ciphertext blob upstream.
- **Token verification.** OIDC `id_token`s are verified via `go-oidc` (signature over provider JWKS + issuer + audience/`client_id`). Azure AD multi-tenant `{tenantid}` template issuers are re-validated manually against the token's own `tid` GUID. A bounded 1-minute clock-skew leeway is applied without disabling expiry.
- **Replay / CSRF defenses.** PKCE `S256` on broker legs (except LinkedIn, which rejects it with a client secret); `nonce` compared when the provider echoes it; OAuth link `state` is single-use and bound to the initiating user; SAML `RelayState` is HMAC-signed, purpose-pinned (`sso` vs `slo`), 15-min TTL, and single-use, with the Response bound to the issued `AuthnRequest.ID` and `AllowIDPInitiated=false`.
- **Open-redirect prevention.** Every redirect target (`redirect_uri`) is validated by exact-match against the client's registered URIs (shared `MatchClientRedirectURI`) before any authorization code / one-time SAML code is appended.
- **No silent account merges.** An upstream email that collides with an existing account is always surfaced for explicit, re-authenticated confirmation via account linking — never merged automatically. The built-in system identity can never be unlinked.
- **Privileged mutations are step-up gated.** All `/identity_providers` create/update/status/delete require `RequireStepUp` — an IdP row is an authentication trust anchor, so a stolen non-elevated session must not be able to repoint an issuer or flip a provider active. External identities carry a `sub` unique per issuer; a subject already owned by a different provider in the tenant is refused rather than guessed.
- **Issued tokens** are maintainerd's standard RS256 JWTs (access + ID + refresh) with a session; federation does not introduce a separate token format.

## Related

- [OAuth Authorization](./oauth2-oidc.md) — the authorization-code server that drives brokered redirect login and mints first-party tokens.
- [Workload Identity Federation](./federation.md) — the separate `internal/federation` subsystem for machine/workload token exchange.
- [Account Linking](./federation.md) — the confirmation-request service invoked on an email collision.
- [Sessions](./sessions.md) · [JWT & Tokens](./cryptography-and-keys.md) — session policy and the RS256 tokens federation issues.
</content>
</invoke>
