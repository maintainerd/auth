# IAM & Authorization

> The tenant-scoped IAM control plane — services, APIs, permissions, roles, and AWS-style policy documents — plus the enforcement engine that decides "may this service principal perform this action on this resource?" from the policies attached to it.

| | |
|---|---|
| **Status** | Implemented. Two enforcement models coexist: **RBAC** (permission-named route guards, in force everywhere) and **policy-based service-to-service authorization** (the PDP/bundle/authorize path). Not implemented: resource policies (a `principal` field on statements), condition blocks, and mTLS-bound service identity. |
| **Code** | `internal/iam` (services, APIs, permissions, roles, policies, policy evaluator, bundle/authorize handlers, token invalidator). RBAC enforcement in `internal/platform/middleware/permission_middleware.go`; service-principal token minting in `internal/oauth/service_token.go`; client↔service binding in `internal/client/service_binding.go`. Security-setting policies are a **separate** subsystem — see [security-settings.md](./security-settings.md). |
| **Endpoints** | REST `/api/v1/{services,apis,permissions,policies,roles}/*`, `/api/v1/services/me/policy-bundle`, `POST /api/v1/authorize/` (internal/management plane :8080); parallel gRPC `ServiceService`/`APIService`/`PermissionService`/`PolicyService`/`RoleService`/`AuthorizationService` (:50051) |
| **Storage** | `services`, `apis`, `permissions`, `api_permissions`, `roles`, `role_permissions`, `policies`, `service_policies`, `policy_version_history` (migrations `006`–`015`); join tables `client_roles`/`client_permissions`/`client_apis`/`user_roles` (`021`–`031`) |
| **Config** | No dedicated env vars. Everything is per-tenant data. Service-token TTL follows the client's effective session policy; sensitive IAM mutations require step-up (`acr=2`). |

## Overview

This feature is two layers that share the same catalog:

1. **RBAC (role-based access control)** — the enforcement in force on every management route. A route declares the permission name(s) it requires (`PermissionMiddleware([]string{"policy:update"})`); the caller's resolved permission set (user→roles→permissions, or client→roles/permissions) must contain at least one of them (`hasAnyPermission`, `internal/platform/middleware/permission_middleware.go:62`). Permission strings are `resource:action` (`api:read`, `service:policy:assign`).

2. **Policy-based service-to-service (S2S) authorization** — an AWS-IAM-shaped **PDP** (Policy Decision Point). Policy *documents* are attached to `services`; a service authenticates as a principal (via a `client_credentials` token carrying `sub_type=service`), and the PDP answers allow/deny by evaluating that principal's attached documents. Distribution is **author centrally, decide locally, invalidate on change**: a service pulls its policy *bundle* (ETag-cached), evaluates decisions in-process, and re-pulls when a webhook signals a change.

The catalog rows themselves:

| Row | Table | What it is |
|-----|-------|------------|
| **Service** | `services` | A named principal / logical service (e.g. `auth`). Unique **per tenant**. Bundles are served keyed off it. |
| **API** | `apis` | A protected resource surface owned by a service; its `identifier` is the token audience/resource permission scoping resolves against. |
| **Permission** | `permissions` | A `resource:action` grant string; the unit RBAC route guards match on. Belongs to an API. |
| **Role** | `roles` | A named bundle of permissions (`role_permissions`), assignable to users (`user_roles`) or clients (`client_roles`). |
| **Policy** | `policies` | A versioned AWS-style `PolicyDocument` (JSONB `document`). Inert until attached to a service. |
| **Service↔Policy** | `service_policies` | The assignment edge. This is where an inert document starts deciding real requests. |

All rows carry `tenant_id` and an `is_system` flag; system rows are protected from update/delete/status change (`service_policy.go:394`, `:532`, `:611`).

## How it works

### Service principal identity

A `Service` becomes an authenticating actor only through an OAuth **client**:

1. An `m2m` client is bound to a service at create/update time via `serviceUUID` → `resolveServiceBinding` (`internal/client/service_binding.go:40`). Rules enforced there: same tenant, service must be `active`, and **only `m2m`** clients may bind (a public SPA/mobile client cannot keep a credential). The binding sets `clients.service_id` (`internal/oauth/deps.go:91`, FK `Service`).
2. On the `client_credentials` grant (`exchangeClientCredentials`, `internal/oauth/service_token.go:536`), if the authenticated client has a linked service, the issued access token is stamped with `sub = Service.Name`, `svc = Service.Name`, and `sub_type = "service"` (`service_token.go:578-587`). The `svc`/`sub_type` claims are written by the JWT layer (`internal/platform/jwt/jwt.go:518-522`).
3. RS256 only; `iss` derives from the client's public hostname via `jwt.TokenIssuerPtr` (`service_token.go:562`) — not a static `ISSUER_URL`.
4. `sub_type=service` tokens are structurally sessionless — session-validation middleware exempts them (`internal/platform/middleware/session_middleware.go`), because an m2m token has no browser session behind it.

The principal name is derived on the enforcement side by `authorizationPrincipal` (`handler_authorization.go:82`): the `svc` claim, or `sub` when `sub_type=service`; otherwise empty (not a service token).

### The PDP — `Evaluate` (`policy_evaluator.go:32`)

Pure, no I/O. Given the principal-scoped `[]PolicyDocument` and an `AuthzRequest{Action, Resource}`, it applies AWS-style **identity-policy** semantics:

| Rule | Behavior | Evidence |
|------|----------|----------|
| **Missing action/resource** | Deny (`missing action or resource`) | `policy_evaluator.go:33` |
| **Default deny** | No matching `allow` → deny (`no matching allow`) | `:63-66` |
| **Explicit deny wins** | Any matching `deny` returns immediately, overriding all allows | `:51-52` |
| **Unsupported version** | A document whose `version != "v1"` **refuses the whole request** (deny) — it is *not* skipped | `:43-45` |
| **Unknown effect** | A statement effect that is neither `allow` nor `deny` **refuses the whole request** (deny) | `:55-58` |
| **Match** | `allow` requires **both** an action pattern and a resource pattern to match | `statementMatches`, `:69` |

The refuse-don't-skip stance on unknown versions/effects is deliberate: dropping an `allow` is safe, but silently dropping a `deny` would remove a guardrail while sibling allows still apply. Wildcards (`wildcardMatch`, `:82`) support `*` (match-all), exact match, and embedded `*` segments (prefix/suffix/contains), e.g. `serviceB:*`, `*:invoke`, `auth:login`.

### Bundle distribution (pull + ETag) — `GET /services/me/policy-bundle`

`AuthorizationHandler.PolicyBundle` (`handler_authorization.go:20`) → `serviceAuthorizationService.PolicyBundle` (`service_policy_bundle.go:52`):

1. Resolve the principal from the token; **`me` is always the token's own service** (no path param), so a service can only ever fetch its own bundle.
2. Tenant is required. `PolicyBundle` refuses `TenantID <= 0` (`:58`) and resolves the service by **name + tenant** (`FindByNameAndTenantID`) — because a service name is unique per tenant, not globally (the seeder creates an `auth` service in every tenant). A tenant-less lookup would collapse every principal onto the oldest tenant's service.
3. Service must exist and be `active`, else not-found (`:65`).
4. Load attached policies (`FindPoliciesByServiceID`), keep only `active` ones, unmarshal each `document`. An **unparseable document fails the whole bundle** (`:81-87`) — same reasoning as the evaluator: a dropped document might have carried a deny.
5. Compute a content ETag: SHA-256 over each `policyUUID:updatedAt:document`, sorted, truncated to 12 hex chars, prefixed `v` (`policyBundleVersion`, `:109`). It changes iff the resolved content changes.
6. Handler compares `If-None-Match`; on a match it returns **`304 Not Modified`** with no body (`handler_authorization.go:32`), otherwise `200` with `ETag` + `Cache-Control: max-age=30`.

Client pattern (per the design): fetch at startup, evaluate locally, re-poll every `max-age` with `If-None-Match`, swap cache only on `200`, and fail-static (serve the last good bundle) if the auth server is briefly unavailable.

### Direct decision — `POST /authorize/`

`AuthorizationHandler.Authorize` (`handler_authorization.go:42`) for callers that want a decision from the server instead of evaluating a bundle locally. The caller supplies **only the question** (`action`, `resource`) in the body; `Principal` and `TenantID` are overwritten from the signed token and never trusted from the body — a prior version left both mass-assignable, letting any token probe allow/deny against any principal in any tenant. It rejects a token with no principal (403) or no tenant (403), then calls `Authorize` (`service_policy_bundle.go:101`), which resolves the principal's bundle and runs `Evaluate`. A gRPC twin exists: `AuthorizationServiceServer.Authorize` (`handler_authorization_grpc.go:21`), with the identical principal/tenant-from-token discipline.

### Push invalidation (webhook events)

IAM mutations emit transactional-outbox integration events (`internal/event`), which the outbox `Relay` delivers to subscribed webhook endpoints and the RabbitMQ broker (`internal/event/relay.go`). A consumer that receives one re-pulls its bundle (with `If-None-Match`, so an already-applied change is a cheap `304`), closing the revocation-staleness gap between polls.

| Event type | Fired by | Constant |
|------------|----------|----------|
| `iam.policy.updated` | policy `Update` (changed fields) **and** `SetStatusByUUID` (status change — the revocation path) | `service_policy.go:475`, `:554` |
| `policy.created` | policy `Create` | `service_policy.go:346` |
| `policy.deleted` | policy `DeleteByUUID` | `service_policy.go:624` |
| `iam.service.policy.assigned` | `POST /services/{uuid}/policies/{uuid}` | `service_service.go:563` |
| `iam.service.policy.removed` | `DELETE /services/{uuid}/policies/{uuid}` | `service_service.go:638` |

All are emitted **inside** the mutating DB transaction so the change and its notification commit atomically. Definitions: `internal/event/constants.go:34-38`.

### RBAC token invalidation

Separately from S2S, IAM changes that affect *users* revoke their sessions. `AuthorizationTokenInvalidator` (`token_invalidator.go`) walks `user_roles`/`role_permissions` to find affected users, flips `user_tokens.is_revoked = true`, and clears cached authorization (`InvalidateUserAll`) — the hard "revoke now" path complementing the bundle-poll softness.

## Implementation

Key files in `internal/iam` (each concept is `model_*` / `repository_*` / `service_*` / `handler_*` / `validation_*`, with `_grpc` handler twins):

- **Evaluator:** `policy_evaluator.go` — `Evaluate`, `statementMatches`, `wildcardMatch`.
- **Bundle + authorize service:** `service_policy_bundle.go` — `PolicyBundle`, `Authorize`, `ServiceIdentity`, `policyBundleVersion`.
- **Authorize handlers:** `handler_authorization.go` (REST), `handler_authorization_grpc.go` (gRPC).
- **Policy CRUD + history:** `service_policy.go` (Create/Update/SetStatus/Delete, version-history snapshotting, event emission), `handler_policy_history.go`, `repository_policy_version_history.go`.
- **Service↔policy assignment:** `service_service.go` — `AssignPolicy` (`:504`), `RemovePolicy` (`:587`); both idempotent, tenant-isolated, event-emitting.
- **Document types & validation:** `types.go` (`PolicyDocument`, `PolicyStatement`), `validation_policy.go` (structure only — actions/resources are **not** validated against existing permissions/services; invalid values simply match nothing).
- **Routes:** `routes.go`. Registered on the internal plane in `internal/server/router.go:77-112`; gRPC in `internal/server/grpc.go:193-212`.

DB tables (migrations under `internal/platform/database/migration/`): `services` (`006`), `policies` (`008`), `service_policies` (`009`), `policy_version_history` (`010`), `apis` (`011`), `permissions` (`012`), `api_permissions` (`013`), `roles` (`014`), `role_permissions` (`015`).

**Policy version history:** every non-system policy `Update` snapshots the *before* state into `policy_version_history` in the same transaction, recording actor (`ChangedByUserID`/`ChangedByClientID`) and an optional `change_reason` (`service_policy.go:434-458`). Read via `GET /policies/{uuid}/history` and `.../history/{version}`. History is optional infrastructure — injected by `SetPolicyVersionHistory`; when absent the two read endpoints return not-found and updates simply skip the snapshot.

## Configuration

No dedicated environment variables — the IAM catalog is entirely per-tenant data authored through the API. Governing settings that apply:

- **Service-token TTL** is the linked client's effective session-policy `AccessTokenTTLSeconds` (`service_token.go:629`). Short TTLs bound revocation staleness even if a webhook and a poll are both missed.
- **Bundle cache hint:** `Cache-Control: max-age=30` (`handler_authorization.go:38`).
- **Step-up (`acr=2`) gate:** every *editing* IAM route — API/permission/policy/role/service update, status, delete, role permission add/remove, and service↔policy assign/remove — is wrapped in `middleware.RequireStepUp` (`routes.go`). `create` routes are intentionally **not** gated: a new row grants nothing until wired to a role or service, and those edges are themselves gated.

## Security considerations

- **Default-deny, deny-wins, fail-closed.** The evaluator denies on missing input, no-match, unknown version, and unknown effect; the bundle resolver fails the whole bundle on an unparseable or (implicitly) tenant-less request rather than serving a partial allow-set.
- **Principal & tenant come only from the signed token.** Both REST and gRPC `Authorize` overwrite any body-supplied `principal`/`tenant_id`. `me` on the bundle endpoint is token-derived, so a service cannot fetch another's bundle.
- **Tenant isolation.** Service resolution is by name **+ tenant**; assign/remove verify the service and policy both belong to the caller's tenant (`service_service.go:525`, `:608`); a tenant-less token is rejected (`:69`).
- **Step-up on the escalation surface.** Editing an existing API identifier, permission name, policy document, role grant, or service↔policy edge silently re-points authorization already in force — hence `acr=2` is required so a stolen `acr=1` session cannot rewrite live authorization.
- **Management-plane placement.** The REST authorize/bundle/CRUD routes are mounted on the internal API (:8080), which additionally applies `RequireManagementClient` — a JWT not minted for the management client is rejected (`internal/platform/middleware/management_client_middleware.go`). The gRPC `AuthorizationService` (:50051) is the service-facing decision surface.
- **RS256 signing, hostname-derived issuer.** Service tokens are RS256; `iss` matches the discovery document's issuer (derived from public hostname), so a compliant relying party can validate it.
- **Not yet implemented (documented gaps):** resource policies (a `principal` field letting callee B independently control who may call it — v1 verifies A's identity policy at both ends), `condition` blocks, mTLS-bound service identity (the `tls_client_auth` client-auth methods are accepted by the column CHECK but have no certificate-binding behind them — see [clients.md](./clients.md)), and decision (verdict) caching.

## Related

- [clients.md](./clients.md) — OAuth clients; the `m2m` client↔service binding that gives a service its identity.
- [authentication.md](./authentication.md) — token issuance, RS256 signing, the `client_credentials` grant.
- [multi-tenancy.md](./multi-tenancy.md) — tenant scoping that every IAM row and lookup enforces.
- [events.md](./events.md) / [webhooks.md](./webhooks.md) — the outbox → relay → webhook path that carries the `iam.*` invalidation events.
- [security-settings.md](./security-settings.md) — the separate `internal/secpolicy` subsystem (password/lockout/MFA/threat/registration/IP-restriction policies); distinct from IAM authorization policies despite the shared "policy" word.
</content>
</invoke>
