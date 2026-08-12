# Multi-Tenancy

> Every account, credential, role, client, and policy in maintainerd-auth is owned by exactly one tenant; a singleton **system tenant** is the root that provisions and manages the rest, and tenants are addressed by a DNS-safe subdomain slug.

| | |
|---|---|
| **Status** | Implemented |
| **Code** | `internal/tenant` (lifecycle, members, settings), `internal/shared` (host ↔ tenant resolution, id↔uuid), `internal/secpolicy` (IP restriction rules), `internal/platform/middleware` (request-tenant, IP restriction, maintenance, status gates) |
| **Endpoints** | Public reads: `GET /api/v1/tenant`, `GET /api/v1/tenant?domain=<host>` (bootstrap), `GET /api/v1/tenant/{name}`. Management (`/api/v1/tenants…`, `/api/v1/tenant-settings…`, `/api/v1/ip-restriction-rules…`) + gRPC `TenantService` |
| **Storage** | `tenants`, `tenant_members`, `tenant_settings`, `ip_restriction_rules` (migrations `001`, `050`, `003`, `055`) |
| **Config** | `APP_FRONTEND_IDENTITY_HOSTNAME`, `APP_FRONTEND_CONSOLE_HOSTNAME` (system-tenant base hosts); per-tenant JSONB config in `tenant_settings`; per-tenant `ip_restriction_rules` |

## Overview

maintainerd-auth is multi-tenant by construction: the `tenants` table is the top of the ownership tree, and virtually every other domain table carries a `tenant_id`. A tenant is identified two ways:

- **Internally** by `tenant_id` (a bigint PK) used for every scoping join.
- **Externally** by `tenant_uuid` (opaque) and by `name` — a **DNS-safe subdomain slug** that is also the tenant's address on the frontend (`{name}.{base-host}`).

Exactly one tenant has `is_system = true` — the **system tenant**. It is the root: the no-`client_id` login/registration surface on the internal port, the source pool for all user accounts, and the only tenant whose members can create, delete, or cross-manage other tenants (`internal/tenant/service_tenant.go:136`, `internal/tenant/service_member.go:592`).

Tenant isolation is **strict**: a member of tenant A can never read tenant B's users, roles, clients, or IdPs, and even a system-tenant identity gets no cross-tenant read override — the system tenant's only cross-tenant power is tenant *management* (`ValidateTenantAccess`, `internal/tenant/service_tenant.go:471`).

## How it works

### Addressing and request-tenant resolution

1. Multi-tenancy is **subdomain-based**. The system tenant is served from the bare configured host (e.g. `auth.maintainerd.local`); every other tenant is served from `{name}.{host}` per surface — identity (hosted login) and console (admin) (`shared.FrontendURL`, `internal/shared/frontend.go:28`).
2. The inverse, `shared.ResolveTenantHost` (`internal/shared/frontend.go:84`), takes an incoming host and returns `(surface, slug, isSystem, ok)`. It strips the configured base host: an exact match → system tenant; a single-DNS-label prefix + `.` + base → a regular tenant whose slug is that prefix. Bases are checked longest-first so a shorter base can't shadow a longer one.
3. `RequestTenantMiddleware` (`internal/platform/middleware/request_tenant.go:118`) derives the request's tenant from the request itself — trying `Origin`, then `X-Forwarded-Host`, then `Host` — and stores it in context. This makes the **request host authoritative** for tenant binding, closing the hole where an external app drives a `client_id` from tenant A while the browser is on tenant B's subdomain.
4. For security decisions (pre-auth IP restriction, maintenance, status gates), `ResolveRequestTenantTrusted` (`request_tenant.go:71`) trusts forwarded host headers **only** from a trusted proxy; direct-to-origin, only the real `Host` is honored, so a forged `X-Forwarded-Host` can't select a different (unrestricted) tenant.

### Frontend bootstrap (the composite init call)

The hosted login/console apps boot before any authentication via `GET /tenant?domain=<host>` (`TenantHandler.GetDefault` → `getBootstrap`, `internal/tenant/handler_tenant.go:210`):

1. `ResolveTenantHost(domain)` maps the host to `(surface, slug, isSystem)`.
2. The tenant is loaded (`GetSystem` if system, else `GetByName(slug)`).
3. The response (`TenantBootstrapResponseDTO`, `internal/tenant/types.go:45`) carries the public tenant projection, the resolved `surface`, canonical `identity_url` / `console_url`, public password + registration policy, active branding, the seeded system client for that surface, its federated `connections`, and `magic_link_enabled`. Client/connection lookups are **best-effort** — a miss degrades to password-only login rather than failing the bootstrap.

Without `?domain`, `GET /tenant` returns the **system tenant** public projection; `GET /tenant/{name}` returns any tenant's public projection (`toPublicResponse`, `handler_tenant.go:377`). All three are unauthenticated and expose only public fields.

> **Drift note:** an earlier internal checklist described a planned `GET /tenant/{identifier}/config` composite endpoint. That path does **not** exist in code — the composite bootstrap is `GET /tenant?domain=<host>`.

### Tenant lifecycle

- **Create** (`tenantService.Create`, `service_tenant.go:173`) — only system-tenant members may call it (enforced in the handler, `handler_tenant.go:447`, and on the gRPC surface). The `name` is validated as a DNS slug and rejected if reserved (below). Inside one transaction it creates the row, runs the per-tenant baseline **seed** (roles, permissions, client, IdP, branding, …) when a seeder is wired, and emits `tenant.created`. Seeding requires a transactional unit of work — the whole tenant is provisioned or not created at all.
- **Ownership → active** — a new tenant starts `pending` and becomes complete only when its first **owner** is assigned; assigning an owner flips a `pending` tenant to `active` (`service_member.go:143`). Assigning the owner role is reserved to system-tenant administrators and implicitly grants the tenant-wide `super-admin` role (`service_member.go:138`).
- **Update / SetStatus** (`service_tenant.go:256`, `:342`) — authorized via `CanManageTenant` (member of the target tenant **or** the system tenant). Both rewrite privileged fields (the subdomain `name`, the lifecycle `status`), so both routes require **step-up** (`internal/tenant/routes.go:62`).
- **Delete** (`service_tenant.go:389`) — only system-tenant members; the **system tenant can never be deleted**; the actor must hold system-tenant membership. Deletion soft-deletes the row and runs `DeleteTenantCascade` for tenant-scoped children.
- **Retention purge** (`internal/tenant/retention.go`) — a background runner hard-deletes soft-deleted **non-system** tenants after a retention period (default 30 days, 24 h interval). The purge runs in one transaction that sets two transaction-local GUCs so the `ON DELETE CASCADE` can pass the append-only audit tables' immutability triggers.

### Members and roles

`tenant_members` binds a `user_id` to a `tenant_id` with a `role` of `owner`, `admin`, or `member` (CHECK constraint, migration `050`). A live tenant has **at most one owner** (partial unique index `uq_tenant_members_one_owner`).

- Members/owners are always sourced from the **system tenant's shared user pool**; `CreateByUserUUID` refuses a user that lives in another tenant, blocking cross-tenant credential/PII copy (`service_member.go:205`).
- **Ownership transfer** (`UpdateRole`, `service_member.go:348`) is the only way to change an owner (direct demote/removal is blocked). It atomically demotes the old owner to `member`, revokes their `super-admin`, promotes the target, grants `super-admin`, and emits `tenant.ownership_transferred`. Only the current owner or a system-tenant admin may transfer.
- Every member **mutation** route is step-up gated (`routes.go:83`): adding/promoting an owner grants tenant-wide super-admin, and removal strips access — as privileged as a tenant rename.

### Tenant-scoped runtime settings

`tenant_settings` holds one JSONB row per tenant with three config sections — `rate_limit_config`, `audit_config`, `maintenance_config` — auto-created with defaults on first read (`getOrCreate`, `service_setting.go:205`). Updates **merge** incoming keys over the stored JSONB rather than replacing the column, so saving one console form never drops another form's keys (`service_setting.go:165`). These feed public-port middleware:

- **Maintenance** (`middleware/maintenance.go`) — when a tenant's window is active, requests get `503 maintenance_mode` (with `Retry-After` when a scheduled end is known). Health probes are excluded. A pre-auth variant gates the **identity** surface only, so operators can always reach the console to lift it.
- **Rate limit** — `TenantRequestRateLimitMiddleware` consumes `rate_limit_config` (Redis-backed).
- **IP restriction** — see below.

### IP restriction

`ip_restriction_rules` are tenant-scoped `allow`/`deny` rules over an `INET` address (IPv4/IPv6/CIDR), managed via CRUD under `/ip-restriction-rules` (`internal/secpolicy/routes.go:11`). Two middlewares enforce them (`middleware/ip_restriction.go`):

- **Post-auth** (`TenantIPRestrictionMiddleware`) uses the session's `auth.Tenant` and **fails open** on a cold load error — an authenticated user mid-session isn't ejected by a store blip.
- **Pre-auth** (`AuthEndpointIPRestrictionMiddleware`) covers the credential surface (login, register, reset, SMS, magic link). The enforcement tenant is derived **strictly from the request's subdomain** (never the body/`client_id`), and it **fails closed** (`503`) when a resolved tenant's rules can't be loaded — a tenant that may restrict logins must not be reachable unverified.

Evaluation (`ipAllowed`, `ip_restriction.go:136`): a matching `deny` blocks; if any `allow` rules exist the IP must match one; a deny-only policy allows non-matching IPs; no rules ⇒ open.

### Lifecycle status gate

`AuthEndpointTenantStatusMiddleware` (`middleware/tenant_status.go`) refuses end-user authentication for a tenant whose status isn't `active`, returning `403 tenant_unavailable`. It gates the identity surface only, resolves status by subdomain slug, passes through unknown hosts, and **fails closed** on a resolver error. It deliberately doesn't distinguish suspended/inactive/unknown, to avoid disclosing tenant lifecycle state to anonymous callers.

### Token binding (id ↔ uuid)

JWTs carry the opaque `tenant_uuid` in the `tenant_id` claim (never the internal PK — RFC 9068 least-disclosure). `shared.TenantRefResolver` (`internal/shared/tenant_ref.go`) maps uuid↔id: mint stamps the uuid, JWT-parse resolves it back to the internal id every scoping check expects. The mapping is immutable and cached.

## Implementation

### Ports

- **Internal API — port 8080** (`buildInternalRouter`, `internal/server/router.go:34`, VPN-only): tenant management (`/tenants…`), `/tenant-settings…`, `/ip-restriction-rules…`, plus the public tenant reads for the console.
- **Public API — port 8081** (`buildPublicRouter`, `router.go:162`, public internet): `TenantPublicRoute` (`GET /tenant`, `GET /tenant/{name}`) and the credential surface wrapped in the pre-auth IP/maintenance/status gates.

### Data model

| Table | Migration | Key columns / constraints |
|---|---|---|
| `tenants` | `001_create_tenants_table.go` | `tenant_id` PK; `tenant_uuid` UNIQUE; `name VARCHAR(63)` UNIQUE per-live (`uq_tenants_name … WHERE deleted_at IS NULL`); `status` default `active`; `is_system` with singleton index `uq_tenants_single_system … WHERE is_system = TRUE`; `metadata` JSONB; soft delete (`deleted_at`) |
| `tenant_members` | `050_create_tenant_members_table.go` | `role` CHECK `('owner','admin','member')`; FK → `tenants`/`users` `ON DELETE CASCADE`; `uq_tenant_members_tenant_user` (one membership per user/tenant); `uq_tenant_members_one_owner` (one live owner) |
| `tenant_settings` | `003_create_tenant_settings_table.go` | 1:1 with tenant (`idx_tenant_settings_tenant_id` UNIQUE); `rate_limit_config` / `audit_config` / `maintenance_config` JSONB with defaults; FK `ON DELETE CASCADE`; no soft delete |
| `ip_restriction_rules` | `055_create_ip_restriction_rules_table.go` | `type` CHECK `('allow','deny')`; `ip_address INET`; `status` CHECK `('active','inactive')`; FK → `tenants` `ON DELETE CASCADE`; soft delete |

Models: `internal/tenant/model_tenant.go`, `model_member.go`, `model_setting.go`, `internal/secpolicy/model_ip_restriction_rule.go`.

### Endpoints

Management (port 8080, JWT + permission middleware, `internal/tenant/routes.go`):

| Method / path | Permission | Notes |
|---|---|---|
| `GET /tenants` | `tenant:read` | Listing scoped: system-tenant members see all; others see only their own (`handler_tenant.go:135`) |
| `GET /tenants/{tenant_uuid}` | `tenant:read` | Non-system caller may only read own tenant |
| `POST /tenants` | `tenant:create` | System-tenant members only |
| `PUT /tenants/{tenant_uuid}` | `tenant:update` + **step-up** | |
| `PUT /tenants/{tenant_uuid}/status` | `tenant:update` + **step-up** | |
| `DELETE /tenants/{tenant_uuid}` | `tenant:delete` + **step-up** | System-tenant members only; system tenant undeletable |
| `GET /tenants/{tenant_uuid}/members` | `tenant:read` | |
| `POST /tenants/{tenant_uuid}/members` | `tenant:update` + **step-up** | Owner role reserved to system-tenant admins |
| `PATCH /tenants/{tenant_uuid}/members/{tenant_member_uuid}/role` | `tenant:update` + **step-up** | Includes ownership transfer |
| `DELETE /tenants/{tenant_uuid}/members/{tenant_member_uuid}` | `tenant:update` + **step-up** | Owner cannot be removed directly |
| `GET/PUT /tenant-settings/{rate-limit\|audit\|maintenance}` | `tenant-setting:read` / `:update` | JSONB section per config type |
| `GET/POST /ip-restriction-rules`, `GET/PUT/DELETE /ip-restriction-rules/{uuid}`, `PATCH …/status` | `ip-restriction-rule:{read,create,update,delete}` | `internal/secpolicy/routes.go` |

Public (ports 8080 + 8081, no auth): `GET /tenant` (system tenant, or bootstrap with `?domain=`), `GET /tenant/{name}`.

gRPC `TenantService` (`internal/tenant/handler_tenant_grpc.go`): `GetDefaultTenant`, `ListTenants`, `GetTenant`, `CreateTenant`, `UpdateTenant`, `SetTenantStatus`, `DeleteTenant`, `ListTenantMembers`, `AddTenantMember`, `UpdateTenantMemberRole`, `RemoveTenantMember`. The acting user is taken from the **verified token**, never the request body (`grpcActorUserID`, `:130`), and mutating RPCs fail closed for service principals that carry no user identity.

### Reserved slugs & validation

`validateTenantSlug` (`internal/tenant/validation_tenant.go:38`, enforced at the service layer so REST and gRPC are both covered): 3–63 chars, pattern `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, and rejects reserved names that would shadow a platform host: `system, console, api, control-api, control, auth, www, admin, root, rabbitmq, prometheus, grafana, signoz`.

## Configuration

| Env var | Purpose |
|---|---|
| `APP_FRONTEND_IDENTITY_HOSTNAME` | System-tenant base host for the identity (login) surface; regular tenants resolve as `{name}.{host}` |
| `APP_FRONTEND_CONSOLE_HOSTNAME` | System-tenant base host for the console (admin) surface |

Per-tenant settings live in `tenant_settings` (managed via `/tenant-settings/*`), with these default shapes (`internal/tenant/defaults_setting.go`):

- `rate_limit_config`: `{enabled:false, requests_per_window:100, window_duration_seconds:60, per_ip:true, exempt_ips:[], endpoint_overrides:{}}`
- `audit_config`: `{enabled:true, retention_days:90, pii_masking:true, log_level:"info", event_types:[]}`
- `maintenance_config`: `{enabled:false, message:"…maintenance…", scheduled_start:null, scheduled_end:null}`

Per-tenant `ip_restriction_rules` are managed via `/ip-restriction-rules`. Tenant `status` (`active`/`inactive`/`pending`/`suspended`, `internal/shared/constants.go:8`) and member `role` (`owner`/`admin`/`member`) are per-tenant record state, not env config.

## Security considerations

- **Strict record isolation.** `ValidateTenantAccess` (`service_tenant.go:471`) grants access only to the actor's own tenant(s). A system-tenant identity is deliberately **not** a cross-tenant read override — that power is confined to tenant *management* via `CanManageTenant`, so a compromised system-admin token cannot read every tenant's users/roles/clients/IdPs.
- **Host is authoritative for tenant binding.** Request tenant is derived server-side from the request host, never from a caller-supplied `tenant_id`/`client_id`; security-sensitive resolution trusts forwarded host headers only from trusted proxies (`ResolveRequestTenantTrusted`).
- **Reserved-slug enforcement is security-critical.** A tenant named `console`/`auth`/… would otherwise resolve to a platform host; slugs are validated at the service layer for both REST and gRPC.
- **Fail-closed pre-auth gates.** Pre-auth IP restriction and tenant-status gates fail **closed** (`503`/`403`) when a resolved tenant's rules/status can't be verified; the post-auth IP gate fails **open** to avoid ejecting live sessions on transient blips.
- **Step-up on privileged tenant writes.** Tenant update/status/delete and all member mutations require `RequireStepUp`, so a replayed `acr=1` session can't rename/suspend a tenant or hand itself tenant-wide super-admin.
- **Cross-tenant member sourcing is blocked.** Members are provisioned only from the system-tenant user pool, preventing a cross-tenant credential/PII copy or account-existence oracle.
- **Least-disclosure token claims.** The `tenant_id` claim carries the opaque `tenant_uuid`, not the internal bigint PK (RFC 9068).
- **Deletion & retention respect audit immutability.** The retention purge sets transaction-local GUCs so the tenant-delete cascade can pass the append-only audit tables' immutability triggers, rather than weakening those triggers globally.

## Related

- `./authentication.md` — the credential surface these pre-auth tenant gates wrap
- `./security-settings.md` — password/registration/lockout/session config and IP-restriction rules
- `./setup-and-bootstrap.md` — first-run creation of the system tenant and per-tenant baseline seeding
- `./clients.md` — clients and federated connections advertised per surface in the bootstrap response
- `./registration-and-invites.md` — how users enter a tenant's shared user pool
