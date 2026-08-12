# Clients

> A **client** is a downstream OAuth2/OIDC relying-party application (SPA, traditional web, mobile/native, or M2M) registered under a tenant, describing how that application authenticates to and obtains tokens from this authorization server.

| | |
|---|---|
| **Status** | Implemented. mTLS client-auth methods (`tls_client_auth`, `self_signed_tls_client_auth`) are accepted by the column CHECK constraint but rejected at write time — no certificate-binding implementation exists behind them. |
| **Code** | `internal/client` (model, service, handlers, validation, config mapping, CORS/redirect matching) |
| **Endpoints** | REST `/clients/*` (control plane :8080, JWT + permission gated) and public `/client`, `/client/console`; a parallel gRPC `ClientService` surface |
| **Storage** | `clients`, `client_uris`, `client_apis`, `client_permissions`, `client_identity_providers`, `client_roles` (migrations `019`–`024`) |
| **Config** | Per-client OAuth/security settings live in first-class columns, written through the free-form `config` JSONB blob and mirrored into columns; deployment-level `AppPublicHostname`, `CORS_ALLOWED_ORIGINS` |

## Overview

A client row represents a **downstream** application that talks to this authorization server — the same concept as an OAuth 2.0 client, a Cognito App Client, or an Auth0 Application. It is **not** where an external provider's own credentials are stored: upstream federation credentials (Cognito/Auth0/Google `client_id`/`secret`) live on `identity_providers` and are enabled per client through the `client_identity_providers` join table (`internal/client/model_client.go:43`). The OAuth columns on a client always describe how *this* app authenticates to *our* token endpoint, regardless of provider.

Each client belongs to exactly one tenant (`tenant_id`, `ON DELETE CASCADE`). Its login options come entirely from its identity-provider connections; a client is not tied to a single IDP (the legacy `IdentityProvider`/`IdentityProviderID` fields are an unpersisted read projection — `model_client.go:66`).

### Client types

`client_type` (`internal/shared/constants.go:42`, DB CHECK `chk_clients_client_type`):

| Type | Class | Secret | Default auth method | Notes |
|------|-------|--------|---------------------|-------|
| `traditional` | Confidential | Yes | `client_secret_basic` | Server-rendered web app |
| `spa` | **Public** | No | `none` (+ PKCE) | Browser single-page app |
| `mobile` | **Public** | No | `none` (+ PKCE) | Native app; custom & loopback redirect schemes |
| `m2m` | Confidential | Yes | `client_secret_basic` | Machine-to-machine; only type bindable to a service |

`IsPublicClientType` (`internal/client/client_matrix.go:27`) = `spa` or `mobile`. Public clients are structurally incapable of keeping a secret (their code is readable), so the DB forbids a stored secret on them (`chk_clients_public_has_no_secret`) and forces PKCE + exact redirect matching instead.

## How it works

### Creation (`clientService.Create`, `internal/client/service_client.go:574`)

1. Resolve the initial identity provider (the named IDP for the tenant, else the tenant's built-in `maintainerd` system provider) — `service_client.go:779`.
2. Resolve + authorize the actor against the tenant (`ValidateTenantAccess`); reject a duplicate client name within the tenant.
3. Mint a globally-unique OAuth `client_id` (`identifier`): 12 symbols (~71 bits) from a 62-symbol alphabet, retried against active/inactive/soft-deleted rows (`generateUniqueClientIdentifier`, `service_client.go:349`).
4. If `serviceUUID` is set, resolve the service binding — same tenant, `active`, `m2m` only (`resolveServiceBinding`, `internal/client/service_binding.go:40`).
5. **Confidential clients only**: generate a 64-symbol secret, bcrypt-hash it into `secret_hash`, keep the plaintext for the one-time response. Public clients get no secret and default `token_endpoint_auth_method` to `none`.
6. Register the client `domain` as a valid token issuer before any token can be minted (`registerIssuer`, `service_client.go:646`).
7. `applyConfigToClientColumns` mirrors the OAuth/security settings from the `config` blob into first-class columns (see [Config mapping](#config-mapping)).
8. If the resolved method is `client_secret_jwt`, additionally store an AES-encrypted (reversible) copy of the secret in `secret_encrypted` — that method HMACs the client assertion with the plaintext, so the server must hold it. No other method stores `secret_encrypted`.
9. `ValidateClientOAuthMatrix` enforces the cross-field rules (see [Security considerations](#security-considerations)).
10. Persist, attach the built-in identity-provider connection (default + enabled), emit `client.created`, write an audit event.
11. Return the client plus a **one-time** `credentials` block (`client_id` + plaintext `client_secret`). The secret is never stored in plaintext and cannot be retrieved again.

### Config mapping (`internal/client/config_mapping.go`)

The admin console persists OAuth/security settings inside the free-form `config` JSONB, but the authorization, token-issuance, login, and session paths read them from **dedicated columns**. `applyConfigToClientColumns` (`config_mapping.go:61`) copies them across on every write:

| `config` key (aliases) | Column | Semantics |
|------------------------|--------|-----------|
| `grant_types` | `grant_types` | leave-unchanged if absent (NOT NULL, DB default) |
| `response_types` | `response_types` | leave-unchanged if absent |
| `allowed_scopes` | `allowed_scopes` | leave-unchanged if absent; empty = every scope |
| `token_endpoint_auth_method` | `token_endpoint_auth_method` | ignored unless in the implemented allowlist |
| `require_consent` (`consent_required`) | `require_consent` | leave-unchanged if absent |
| `require_pkce` (`pkce_required`) | `require_pkce` | leave-unchanged if absent |
| `access_token_lifetime` (`access_token_ttl`) | `access_token_ttl` | **cleared to NULL when absent** (inherit tenant) |
| `refresh_token_lifetime` (`refresh_token_ttl`) | `refresh_token_ttl` | cleared to NULL when absent |
| `required_acr` | `required_acr` | cleared to NULL when absent / out of `{1,2}` |
| `session_idle_timeout` (`_seconds`) | `session_idle_timeout` | cleared to NULL when absent |
| `session_absolute_timeout` (`_seconds`) | `session_absolute_timeout` | cleared to NULL when absent |
| `jwks` / `jwks_uri` | `jwks` / `jwks_uri` | mutually exclusive; inline JWKS wins |
| `mtls_bound_cert_thumbprint` | `mtls_bound_cert_thumbprint` | base64url SHA-256, 43 chars |
| `scope_claim_mappings` | `scope_claim_mappings` | `{scope: [claims]}` |
| `claim_mappers` | `claim_mappers` | `{claim: value}`; reserved claims rejected |

Keys with **no column** (`cors_enabled`, `refresh_token_rotation`, `multi_resource_refresh_token`, operator metadata) pass through the blob untouched.

Reads return the **effective** config, not the raw blob: `effectiveClientConfig` (`config_mapping.go:304`) strips every mirrored key and re-emits the canonical spelling from the authoritative column, so `GET /clients/{uuid}/config` can never show a stale value the runtime rejected, and a round-trip save cannot silently revert a column changed by the seeder/gRPC/SQL.

### Secret rotation (`clientService.RotateSecret`, `service_client.go:847`)

1. Grace period capped at **168h (7 days)** — enforced in the service so both REST and gRPC transports are covered (`maxSecretGracePeriodHours`, `validation_client.go:224`).
2. Refuses system clients (would break the seeded console/login UI) and public clients (no secret to rotate).
3. Current `secret_hash`/`secret_encrypted` move to `previous_secret_*`. With `grace_period_hours > 0`, `previous_secret_expires_at` is set; with `0`, the previous secret is dropped immediately.
4. New secret hashed into `secret_hash`; `secret_encrypted` written only if the client is on `client_secret_jwt`, else cleared.
5. Emit `client.secret_rotated`, write a `Warn`-severity audit event. Returns the new plaintext once.

During the grace window the token endpoint accepts either secret: `internal/oauth/foundation.go:104` checks the previous hash while `previous_secret_expires_at` is still in the future.

## Implementation

### Key files (`internal/client/`)

| File | Role |
|------|------|
| `model_client.go` | `Client` GORM model + field groups; `DefaultConnectedIdentityProvider` resolution |
| `types.go` | Request/response DTOs (`ClientCreateRequestDTO`, `ClientResponseDTO`, `ClientPublicResponseDTO`, …) |
| `validation_client.go` | Per-field DTO validation (name, domain pattern, config size, grace cap) |
| `validation_client_config.go` | Advanced `config` shape validation (JWKS, mTLS thumbprint, claim mappers) |
| `config_mapping.go` | `applyConfigToClientColumns` / `effectiveClientConfig` blob↔column mirroring |
| `client_matrix.go` | `ValidateClientOAuthMatrix` cross-field rules + IDP-connection usability invariants |
| `service_client.go` | `clientService` — create/update/rotate/status/delete, URIs, connections, APIs, roles |
| `service_binding.go` | `resolveServiceBinding` — m2m→service binding rules |
| `redirect_match.go` | `MatchClientRedirectURI` (exact + loopback), `ValidateRegisteredRedirectURI` |
| `cors_origins.go` | `CORSOriginResolver` — per-tenant `cors_origin_uri` allow-list |
| `handler_client.go` / `handler_client_grpc.go` | REST + gRPC handlers |
| `routes.go` | Route table + permission/step-up middleware |

### REST endpoints (`internal/client/routes.go`)

Public (unauthenticated), `ClientPublicRoute`:

| Method | Path | Handler |
|--------|------|---------|
| GET | `/client` | `GetPublic` — resolve a client by OAuth `client_id` |
| GET | `/client/console` | `GetPublicConsole` — the tenant's seeded auth-console client |

Managed, `ClientRoute` — all require `JWTAuthMiddleware` + `UserContextMiddleware`; each carries a `client:*` permission, and every mutating route additionally requires `RequireStepUp`:

| Method | Path | Permission |
|--------|------|-----------|
| GET | `/clients` | `client:read` |
| GET | `/clients/{uuid}` | `client:read` |
| POST | `/clients` | `client:create` |
| PUT | `/clients/{uuid}` | `client:update` (step-up) |
| PUT | `/clients/{uuid}/status` | `client:update` (step-up) |
| DELETE | `/clients/{uuid}` | `client:delete` (step-up) |
| POST | `/clients/{uuid}/rotate-secret` | `client:secret:rotate` (step-up) |
| GET | `/clients/{uuid}/config` | `client:config:read` |
| GET/POST/PUT/DELETE | `/clients/{uuid}/uris[...]` | `client:uri:{read,create,update,delete}` |
| GET/POST/PUT/DELETE | `/clients/{uuid}/identity_providers[...]` | `client:identity_provider:{read,create,update,delete}` |
| GET/POST/DELETE | `/clients/{uuid}/apis[...]` | `client:api:{read,create,delete}` |
| GET/POST/DELETE | `/clients/{uuid}/apis/{api}/permissions[...]` | `client:api:permission:{read,create,delete}` |
| GET/POST/DELETE | `/clients/{uuid}/roles[...]` | `client:role:{read,create,delete}` |

There is **deliberately no** `GET /{uuid}/secret`: secrets are bcrypt-hashed at rest and unrecoverable. The gRPC `GetClientSecret` returns `Unimplemented` for the same reason (`handler_client_grpc.go:97`); rotation is the only way to obtain a secret after creation.

### gRPC surface (`internal/client/handler_client_grpc.go`)

`ClientGRPCHandler` mirrors the REST surface: `ListClients`, `GetClient`, `CreateClient` (replay-guarded — the response carries a one-time secret), `UpdateClient`, `SetClientStatus`, `DeleteClient`, `RotateClientSecret`, `GetClientConfig`, and the URI/API/permission sub-resources. `RotateSecret`'s grace cap is enforced in the shared service, so gRPC cannot bypass it.

### Storage

`clients` (migration `019_create_clients_table.go`), field groups: identity & ownership, descriptive, secret storage, config & lifecycle, OAuth core, token lifetime, security overrides, advanced client auth, claims, audit. Notable constraints:

- `uq_clients_identifier` — global unique OAuth `client_id` (resolved without a tenant predicate).
- `uq_clients_tenant_default` — at most one `is_default` client per tenant.
- `uq_clients_tenant_name` — unique client name per tenant.
- `chk_clients_public_has_no_secret`, `chk_clients_none_auth_is_public_only` — public-client secret/auth invariants.
- `chk_clients_token_ttl_order` — refresh TTL ≥ access TTL; `chk_clients_session_timeout_order` — absolute ≥ idle.
- `chk_clients_grant_types` — subset of `{authorization_code, client_credentials, refresh_token, device_code (urn), ciba (urn), token-exchange (urn)}`.
- `chk_clients_response_types` — subset of `{code, token, id_token}`.

`client_uris` (`020`): `type ∈ {redirect_uri, origin_uri, logout_uri, login_uri, cors_origin_uri}` (`chk_client_uris_type`), unique per `(client_id, type, uri)` among live rows. `client_identity_providers` (`023`): `(client_id, identity_provider_id)` unique, at most one `is_default` per client. `client_apis`/`client_permissions`/`client_roles` (`021`/`022`/`024`) back API-scope and role grants.

## Configuration

Client behavior is per-client and stored in the DB, not env vars. Governing inputs:

- **Per-client columns** (via the `config` blob, [table above](#config-mapping)): `token_endpoint_auth_method`, `grant_types`, `response_types`, `allowed_scopes`, `require_consent`, `require_pkce`, `access_token_ttl`, `refresh_token_ttl`, `required_acr`, `session_idle_timeout`, `session_absolute_timeout`, `jwks`/`jwks_uri`, `mtls_bound_cert_thumbprint`, `scope_claim_mappings`, `claim_mappers`. Nullable override columns mean "inherit the tenant `security_settings`/`token_config` default".
- **Per-client toggles**: `allow_registration` (default true), `allow_magic_link` (default **false**, opt-in passwordless email sign-in), `dpop_required`, back/front-channel logout URIs, `branding_id`, `service_id`.
- **`config` blob cap**: 16 KB (`maxClientConfigBytes`, `validation_client.go:14`).
- **Deployment-level**: `AppPublicHostname` (first-party same-site boundary, below) and `CORS_ALLOWED_ORIGINS` (operator-owned origins, separate from per-client `cors_origin_uri` rows).

## Security considerations

- **OAuth matrix validation** (`ValidateClientOAuthMatrix`, `client_matrix.go:59`), run on create & update after per-field allowlists:
  - `token_endpoint_auth_method=none` is valid **only** for public clients — accepting it on a confidential/m2m client would make the token endpoint unauthenticated (the `client_id` is public).
  - A public client may not use a secret-based method (`client_secret_basic/_post/_jwt`) — the secret ships in readable code.
  - Secret-based methods require a stored secret; `private_key_jwt` requires `jwks`/`jwks_uri`.
  - `client_credentials` is invalid for public clients and requires a non-empty `allowed_scopes` (an empty list = every scope, unbounded for a machine credential); `authorization_code` is invalid for `m2m`.
  - `tls_client_auth`/`self_signed_tls_client_auth` are refused at write time — unimplemented (`clientAuthMethodIsUnimplemented`).
- **Secret storage**: bcrypt `secret_hash` for verification; reversible AES `secret_encrypted` written **only** for `client_secret_jwt` (which needs the plaintext to HMAC assertions). Plaintext returned exactly once. Rotation keeps a grace-windowed previous hash, capped at 7 days.
- **Redirect-URI matching** (`redirect_match.go`): exact match per OAuth 2.0 Security BCP §4.1.3 — no wildcards/prefix/subdomain — with the RFC 8252 §7.3 loopback-port exception for native apps. Registration-time validation forbids fragments, embedded credentials, and plain `http` to non-loopback hosts; custom reverse-domain schemes are `mobile`-only.
- **Per-tenant CORS** (`cors_origins.go`): `cors_origin_uri` rows grant credentialed cross-origin access **scoped to the registering tenant only** — a flat allow-list would let one tenant read another's cookie-authenticated responses. Fails closed when the request's tenant can't be resolved.
- **First-party boundary** (`IsFirstPartyClient`, `service_client.go:308`): the account self-service surface is gated on whether the client's registered `domain` shares a registrable domain (eTLD+1) with `AppPublicHostname` — decided by domain, **not** by `is_system`, so a third-party token can't reach the user's account-management APIs. Native apps are third-party by this rule and go through consented OAuth.
- **Service binding** (`service_binding.go`): only an `active`, same-tenant `m2m` client may bind to a service; the binding puts the `svc` claim in its tokens, which the policy bundle and gRPC authorizer resolve as the principal — a privilege grant, not a label.
- **Protected clients**: `is_system` and `is_default` clients cannot be updated, status-toggled, deleted, or (system) secret-rotated; both flags are platform-owned and can't be set via the create/update API.
- **Optimistic concurrency**: `ClientUpdateRequestDTO.ExpectedUpdatedAt` guards against two operators clobbering each other on a full-object update.
- **Step-up (MFA)**: every mutating client route requires `RequireStepUp` in addition to its permission.
- **Claim-mapper guard**: client-defined `claim_mappers` cannot target reserved JWT claims (`jwt.IsReservedClaim`) — the issuer strips them, and the validator rejects them so the operator learns the mapper is ignored.

## Related

- [./federation.md](./federation.md) — providers a client connects to (`client_identity_providers`)
- [./oauth2-oidc.md](./oauth2-oidc.md) — authorize/token endpoints that consume client auth methods, redirect URIs, PKCE, and the secret grace window
- [./cryptography-and-keys.md](./cryptography-and-keys.md) — access/refresh/ID token issuance, TTL overrides, scope→claim mapping
- [./iam-authorization.md](./iam-authorization.md) — IAM services an `m2m` client binds to via `service_id`
- [./multi-tenancy.md](./multi-tenancy.md) — tenant-level `security_settings`/`token_config` defaults these columns override
