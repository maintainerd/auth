# Events and Webhooks Planning

This plan covers how maintainerd-auth notifies the outside world when its data
changes, so that other services that **reference but do not own** our data (for
example an external app that stores a user list only to render user names) can
stay in sync.

It is a **complete v1.0.0 feature** — nothing here is deferred. Both delivery
channels ship together:

- **Webhooks** — per-tenant, per-endpoint HTTP delivery to external services.
- **RabbitMQ broker** — per-tenant event publishing for internal/first-party
  consumers.

Kafka is the only thing explicitly excluded — RabbitMQ is the single broker.

The emission model is **persist-when-listened, filter-at-delivery** (the model
Stripe, WorkOS, and Svix use): if a tenant has any active listener, that tenant's
integration events are written to a durable outbox and the exact per-type match
is applied at delivery; a tenant with no listeners stores nothing. A per-tenant
**master switch** lets a tenant hard-disable any event type so it is never stored
— giving direct control over storage cost without losing replay for the types it
keeps.

## Design Principles

1. **Two planes, different consumers.** Security/audit events and data-change
   integration events are not the same product and do not share a delivery path.
   See [The Two Planes](#the-two-planes).
2. **Persist when any listener exists; filter at delivery.** If a tenant has at
   least one active subscription (a webhook endpoint or a broker route), that
   tenant's integration events are written to the outbox; the exact event-type
   match is applied per-subscription at delivery. A tenant with zero listeners
   stores nothing. This preserves replay/backfill and avoids the
   "subscribe-just-after-the-event → lost forever" race.
3. **Per-tenant cost control.** A tenant can hard-disable an event type via the
   master switch (`tenant_event_types`); a disabled type is never built, stored,
   or delivered for that tenant. This is how a tenant keeps the outbox lean
   without giving up replay for the types it cares about.
4. **Thin events — names, not values.** Integration events carry stable
   identifiers plus a list of changed **field names** (never their values).
   Consumers re-fetch through the authenticated API for fresh data. This keeps
   PII off the wire and removes most ordering hazards. Enforced — see
   [Emission Rules](#emission-rules).
5. **Curated catalog, not every action.** We expose an event only when a service
   that references our data would react to it (mirror sync, cache invalidation,
   provisioning). Internal configuration and authentication ceremonies are not
   exposed — see [Not Exposed](#not-exposed-as-integration-events-and-why).
6. **Audit stays audit.** Login/MFA/token/OAuth activity remains in `auth_events`
   for the tenant's own security monitoring; it is not routed to integration
   webhooks by default.

## The Two Planes

### Audit / security plane (already exists — unchanged)

- Backed by `auth_events` (`AuthEventService.Log`), modeled on the OWASP logging
  vocabulary.
- Consumer: the tenant's security team / SIEM. Records for humans and detection
  rules, not signals other applications react to.
- Login, logout, MFA, token issuance, OAuth consent, lockouts, impossible travel
  all live here.
- Not pushed to integration webhooks by default. A future "audit firehose / log
  streaming" mode could opt a tenant into receiving these, but that is a
  separate, explicitly-enabled feature.

This matches how Okta (System Log / Log Streaming vs Event Hooks), Auth0 (Log
Streams vs Actions), and GitHub (audit-log streaming vs webhooks) split the two
concerns.

### Integration / data-change plane (the work this plan scopes)

- Domain events emitted after a successful mutation: `user.updated`,
  `role.permissions_changed`, `tenant.deleted`, etc.
- Consumer: other services that mirror or reference our data and need to know
  when to refresh.
- Delivered through **both** webhooks and the RabbitMQ broker.
- This is the Stripe / WorkOS model: a focused catalog of resource lifecycle
  events, never "one event per API request."

## Status Values

- `todo` — not started.
- `in-development` — partially built; not complete across emission, gating,
  delivery, and tests.
- `done` — the action emits the expected event only after success, the event is
  gated correctly, it reaches the configured channel(s), and tests prove the
  behavior.

## Current State

All 48 integration events wired across 5 groups. The transaction-bound outbox spine is built end-to-end: envelope → write gate → outbox → relay → webhook dispatch with delivery history, durable retries, dead-letter, and auto-quarantine.

- `event_types` canonical catalog seeded at startup with 48 v1.0.0 event types.
- `webhook_endpoints` normalized: `events JSONB` replaced with `subscribe_all` + `webhook_endpoint_events` M:N junction.
- Write gate cached in memory with Redis pub/sub invalidation + 30s TTL fallback.
- Transactional outbox written inside mutation transactions; async relay dispatches to webhooks.
- Webhook delivery history tracks every attempt with `next_retry_time`, `final_status`, and dead-letter.
- `BackgroundRetrier` polls `next_retry_time` for pending retries, surviving process restarts.
- Endpoints auto-quarantined after 10 consecutive dead-letter deliveries.
- Webhook signing secret generated server-side (`crypto.GenerateRandomString`), returned once on create.
- `RateLimitAndCapMiddleware` limits webhook endpoint creation to 50 per tenant.
- `RetentionRunner` purges published outbox rows >7d and delivery history >90d.
- Config APIs: `GET /event-types`, `GET/PUT /tenant-event-types`, `GET/POST/PUT/DELETE /event-routes`.
- Replay API: `POST /webhook-replay` replays an event to one or all endpoints.
- `X-Maintainerd-Event-Id` header provides stable receiver idempotency key.
- RabbitMQ publisher adapter (function-injected) available for broker channel.
- gRPC parity: domain events emitted in service layer, shared by REST and gRPC handlers.
- All 42 test packages pass.

## Scope (everything below is v1.0.0)

In scope and built for v1.0.0:

- The integration event plane and its curated catalog.
- The normalized routing registry (catalog + subscriptions + broker routes +
  per-tenant master switch).
- The transactional outbox + async relay.
- Webhook delivery history, durable retries, dead-letter queue, replay tooling.
- The RabbitMQ publisher and per-tenant broker routing.
- Tenant-scoped config APIs for the frontend, with the config-API security
  hardening.

Deliberately **excluded** (not deferred — out of scope by design):

- **"Every API endpoint emits a request-level event"** and **read/`GET` firehose
  events** — an audit-log concern, huge volume, no integration consumer.
- **Routing login/MFA/OAuth ceremonies to integration webhooks** — stays in the
  audit plane.
- **Kafka** — RabbitMQ is the only broker.

## Target Architecture

```text
API handler/service
  -> domain event built after successful business mutation
  -> WRITE GATE (resolved from in-memory cache BEFORE opening the tx):
       type globally active?  AND  not disabled for this tenant?  AND
       tenant has any active listener (webhook endpoint OR broker route)?
       - no  -> skip (nothing stored)
       - yes -> write event to durable outbox in the SAME transaction as the mutation
  -> outbox relay (async)
       -> webhook dispatcher: per-endpoint exact-type match, then signed POST
       -> RabbitMQ publisher: per-tenant route match, then publish
  (audit plane is independent: auth_events is always written for compliance)
```

The write gate is a cheap in-memory check resolved **before** the transaction
opens, so a gate query can never deadlock or abort the mutation. The outbox write
shares the mutation's transaction so the data change and the event cannot
disagree. Exact per-type subscription matching happens at the relay's delivery
step, so adding a subscription later still replays earlier events.

## Event Routing Registry (normalized)

Enablement is stored relationally, not as a JSONB array. Today's
`webhook_endpoints.events JSONB` holds a repeating group of event-type strings in
one column — that breaks 1NF and has no referential integrity (a typo'd or
retired type is silently accepted). The normalized design follows the existing
M:N convention in the codebase (`client_permissions`, `api_permissions`, …):
`BIGSERIAL` PK, `UUID` unique key, FK columns with `ON DELETE CASCADE`, a
`UNIQUE` pair constraint, and supporting indexes.

Both channels are **tenant-bound** — every tenant manages its own webhook and
broker configuration, and one tenant's choices never affect another's. Per-type
selection is applied at delivery; the write gate only asks the cheap question
*"does this tenant have any active listener, and is this type enabled?"*

There are two master switches that stop an event from being produced **at all**:

1. **Global (platform):** `event_types.is_active`. Disabling a type retires it
   system-wide. Used to kill a noisy or deprecated event everywhere.
2. **Per tenant:** `tenant_event_types.enabled`. A tenant hard-disables a type
   for itself; it is never built, stored, or delivered for that tenant. This is
   the tenant's lever for keeping the outbox lean.

The model in one line:

```text
store outbox row     ⇔  is_active(global) AND tenant-enabled AND tenant-has-any-listener
deliver to webhook   ⇔  endpoint subscribes to this exact type (or subscribe_all)
publish to broker    ⇔  an enabled event_route matches this type for the tenant
```

Five tables, each with one responsibility.

### 1. `event_types` — canonical event catalog (new, seeded)

Single source of truth for which event types exist. Platform-defined and seeded
(not tenant-created); subscriptions reference it by FK so you cannot subscribe to
a non-existent type. Powers the admin "what can I subscribe to?" listing.

| Column | Purpose |
| --- | --- |
| `event_type_id` (BIGSERIAL PK) | Internal key referenced by FKs |
| `event_type_uuid` (UUID UNIQUE) | External ID |
| `key` (VARCHAR UNIQUE) | Canonical type, e.g. `user.updated` |
| `category` (VARCHAR) | e.g. `USER`, `TENANT`, `IAM`, `CLIENT` |
| `description` (TEXT) | Human-readable summary |
| `version` (INT) | Schema version of the event |
| `is_active` (BOOL) | Global kill switch |
| `created_at` / `updated_at` | Timestamps |

### 2. `webhook_endpoints` — endpoint config (edit migration 007 in place)

Per the create-only rule, edit `007_create_webhook_endpoints_table.go` in place:
**drop the `events JSONB` column** and add `subscribe_all BOOLEAN NOT NULL DEFAULT
false`. The table keeps URL, secret, retries, timeout, status, etc. Subscriptions
move to the junction below.

### 3. `webhook_endpoint_events` — endpoint ↔ event-type subscriptions (new, M:N)

One row per subscribed event type, following the `client_permissions` pattern.

| Column | Purpose |
| --- | --- |
| `webhook_endpoint_event_id` (BIGSERIAL PK) | — |
| `webhook_endpoint_id` (FK → `webhook_endpoints`, CASCADE) | Owning endpoint |
| `event_type_id` (FK → `event_types`, CASCADE) | Subscribed type |
| `created_at` | Timestamp |
| `UNIQUE (webhook_endpoint_id, event_type_id)` | No duplicate subscriptions |

"Subscribe to everything" is the explicit `webhook_endpoints.subscribe_all` flag
— it replaces the old implicit "empty array / `*` means all," which was a magic
value (a brand-new endpoint silently received everything). Explicit opt-in is
safer.

### 4. `event_routes` — broker (RabbitMQ) routing, per tenant (new)

Which event types each tenant publishes to the broker. Tenant-bound like
webhooks: every tenant manages its own broker config from the frontend. Consumers
read `tenant_id` from the event payload to route/filter downstream.

| Column | Purpose |
| --- | --- |
| `event_route_id` (BIGSERIAL PK) | — |
| `event_route_uuid` (UUID UNIQUE) | External ID |
| `tenant_id` (FK → `tenants`, CASCADE) | Owning tenant |
| `event_type_id` (FK → `event_types`, CASCADE) | Routed type |
| `channel` (VARCHAR, default `rabbitmq`) | Reserved for future channels; Kafka excluded |
| `destination` (VARCHAR) | RabbitMQ exchange/queue name |
| `enabled` (BOOL) | Whether this route is active |
| `created_at` / `updated_at` | Timestamps |
| `UNIQUE (tenant_id, event_type_id, channel)` | One route per type per channel per tenant |

### 5. `tenant_event_types` — per-tenant master switch (new)

A tenant's enable/disable toggle for an event type, applied at the write gate.
Absence of a row means **enabled** (default-on), so this table only stores
deliberate "off" overrides — a tenant that never disables anything has no rows
here. A disabled type is never built, stored, or delivered for that tenant.

| Column | Purpose |
| --- | --- |
| `tenant_event_type_id` (BIGSERIAL PK) | — |
| `tenant_event_type_uuid` (UUID UNIQUE) | External ID |
| `tenant_id` (FK → `tenants`, CASCADE) | Owning tenant |
| `event_type_id` (FK → `event_types`, CASCADE) | The type being toggled |
| `enabled` (BOOL, default `true`) | When `false`, suppressed for this tenant regardless of subscriptions |
| `created_at` / `updated_at` | Timestamps |
| `UNIQUE (tenant_id, event_type_id)` | One toggle per type per tenant |

New tables (`event_types`, `webhook_endpoint_events`, `event_routes`,
`tenant_event_types`) get create-only migrations appended to the registry in
`internal/platform/runner/migration.go`, plus the in-place edit to migration 007.
Order matters: `event_types` must be created (and seeded) before the tables that
FK to it.

### How the gate reads it

Two cheap checks, deliberately split.

**Write gate** (resolved from in-memory cache, before opening the tx): is this
type globally active, not disabled for this tenant, and does this tenant have any
active listener?

```sql
-- (a) global kill switch — cached set of active type keys
SELECT key FROM event_types WHERE is_active;

-- (b) is the type disabled for this tenant? (cached per-tenant "off" set)
SELECT 1 FROM tenant_event_types
  WHERE tenant_id = $1 AND event_type_id = $2 AND enabled = false;

-- (c) does the tenant have ANY active listener? (cached per tenant)
SELECT EXISTS (
  SELECT 1 FROM webhook_endpoints
   WHERE tenant_id = $1 AND status = 'active' AND deleted_at IS NULL
  UNION ALL
  SELECT 1 FROM event_routes WHERE tenant_id = $1 AND enabled
);
```

Write the outbox row when (a) contains the type AND (b) returns nothing AND (c)
is true. No per-endpoint lookup happens on the write path.

**Delivery filter** (in the relay): exact per-type matching lives here.

```sql
-- webhook endpoints in this tenant that should receive this event type
SELECT we.webhook_endpoint_id, we.url
  FROM webhook_endpoints we
  WHERE we.tenant_id = $1 AND we.status = 'active' AND we.deleted_at IS NULL
    AND (
      we.subscribe_all
      OR EXISTS (
        SELECT 1 FROM webhook_endpoint_events wee
        JOIN event_types et ON et.event_type_id = wee.event_type_id
        WHERE wee.webhook_endpoint_id = we.webhook_endpoint_id AND et.key = $2
      )
    );

-- broker routes in this tenant for this event type
SELECT er.destination
  FROM event_routes er
  JOIN event_types et ON et.event_type_id = er.event_type_id
  WHERE er.tenant_id = $1 AND er.enabled AND et.key = $2;
```

The write-gate state is cached in memory and invalidated when endpoints,
subscriptions, broker routes, the master switch, or `is_active` change. Because
the write gate is coarse, **adding a subscription later replays earlier events** —
they are already in the outbox / delivery history.

> **Outbox note:** the durable outbox / delivery-history rows store the
> denormalized `event_type` **string** (not just the FK), because an append-only
> event log must stay accurate for replay even if a catalog row is later renamed
> or deactivated. The FK-normalized form is for *subscription* tables; the *log*
> keeps the historical value.

### Multi-replica cache invalidation

The write gate reads an in-memory cache, so with multiple auth replicas a config
change on one replica must reach the others. **Mechanism: Redis pub/sub** (the
app already has an optional Redis client). On any routing-config write, publish
`(tenant_id, change)` on an invalidation channel; every replica subscribes and
drops its cached gate state for that tenant. When Redis is absent (single replica
or dev), fall back to a short cache TTL (e.g. 30s). Versioned cache keys are an
acceptable alternative.

### Tenant-managed configuration

Both the webhook event selection and the broker routing are **tenant-scoped
configuration**, managed from the separate frontend app. Each tenant configures
its own setup independently. The frontend drives this through tenant-scoped admin
APIs:

- **List available events:** read the `event_types` catalog (filtered to
  `is_active`) so the UI can render the full set of toggles/checkboxes.
- **Tenant event master switches:** a switch per event type that enables/disables
  it for the whole tenant; writes `tenant_event_types`. A disabled type is greyed
  out and suppressed everywhere downstream.
- **Webhook registration form:** create/edit a `webhook_endpoints` row and submit
  the selected event types; writes `webhook_endpoint_events` (or sets
  `subscribe_all`).
- **Broker config form:** toggle which event types publish to RabbitMQ; writes
  `event_routes` for that tenant.

Every write invalidates that tenant's cached gate state (via the Redis channel
above) so the change takes effect immediately.

#### Config-API security requirements

Registering a webhook or broker route configures **data egress**, so these
endpoints carry more risk than ordinary config:

- **Admin-scoped authorization.** Requires an admin-tier permission
  (`webhook-endpoint:*`), not any authenticated tenant user; PDP-gated. The role
  model must keep that permission admin-only.
- **SSRF validation at registration** — *already implemented* via `webhookURLRule`
  on the create/update DTOs (`internal/webhook/validation_endpoint.go`). Keep it;
  do not regress.
- **Server-side signing secret** — *to fix*: today `Create`/`Update` accept a
  caller-supplied `secret` (`internal/webhook/service_endpoint.go`). Generate the
  HMAC signing secret **server-side**, return it **once** on creation, store only
  the encrypted form, and stop accepting a caller secret.
- **Rate-limit + cap endpoint creation per tenant**, so a tenant cannot create
  thousands of endpoints and turn one event into a delivery-amplification fan-out.

## Emission Rules

- A domain event is emitted **only after** the mutation succeeds, **inside the
  mutation's transaction** (via the outbox), and **only if** the write gate is
  open (type globally active + not disabled for the tenant + tenant has a
  listener).
- **Thin = names, not values (enforced).** The changed-fields list contains field
  **names only** (`["email","status"]`), never their values. The payload carries
  identifiers (UUIDs), `tenant_id`, actor, and event metadata — nothing a consumer
  would need authorization to see. Consumers refetch values over the
  authenticated API. This is a hard rule: **a test must assert the outbox payload
  contains no resource value fields** (no email, name, phone, secret, token).
- Failed mutations emit **no** domain event. (Failure visibility for security
  belongs to the audit plane.)
- Existing audit-event constants and the `auth_events` path remain unchanged.
- Existing IAM cache-invalidation events (`iam.policy.updated`,
  `iam.service.policy.*`) are integration events already and stay stable.

### Delivery contract (what consumers can rely on)

- **At-least-once, unordered.** Delivery may duplicate and may arrive out of
  order. Consumers **dedup on `event_id`** and treat ordering as best-effort.
- **Refetch on receive.** An event is a *signal*, not a source of truth; the
  consumer fetches current state from the API. A `404` on refetch means the
  resource was deleted — a valid terminal outcome, not an error.
- **Best-effort per-tenant ordering.** The relay processes a tenant's outbox rows
  in insertion order on the happy path; thin+refetch makes strict ordering
  unnecessary, so no global ordering guarantee is offered.

### Retention & right-to-erasure

Because events are thin (UUIDs + field names, no PII values), the privacy surface
is small. Policy:

- **Outbox:** rows deleted after successful publish + a short grace window (e.g.
  7 days) for replay.
- **Delivery history:** retained ~90 days, then purged by a retention runner
  (same pattern as `auth_events` retention).
- **Dead-letter:** retained until resolved or expired (e.g. 30 days).
- **Right-to-erasure:** a hard-deleted user's events carry only their UUID, so
  erasing the user record removes the PII linkage; the retention runner also
  purges outbox/history/DLQ rows referencing the deleted subject UUID.

## Recommended Integration Event Catalog

The complete v1.0.0 catalog. Every event is thin, gated by the routing registry,
emitted only after success, inside the mutation transaction. All groups are in
v1.0.0; events fire only when a tenant has subscribed/enabled them, so defining
the full set carries no runtime cost for tenants that do not use them.

### Group 1 — User identity (primary mirror-sync)

The "render the user's name elsewhere" case. Externally-visible identity and
profile field changes are folded into `user.updated` (names only).

| Status | Event | Trigger | Why a consumer needs it |
| --- | --- | --- | --- |
| done | `user.created` | User created | Add to the mirrored user list |
| done | `user.updated` | Identity/profile fields change | Refresh displayed user data |
| done | `user.status_changed` | Activated / suspended / locked | Block or restore access downstream |
| done | `user.deleted` | User deleted | Remove from the mirror |
| done | `user.role_assigned` | Role assigned to user | Refresh the user's authorization |
| done | `user.role_removed` | Role removed from user | Refresh the user's authorization |

### Group 2 — Authorization model (services that enforce/cache authz)

| Status | Event | Trigger | Why a consumer needs it |
| --- | --- | --- | --- |
| done | `role.created` / `role.updated` / `role.deleted` | Role lifecycle | Keep the cached role catalog correct |
| done | `role.permissions_changed` | Permissions added/removed on a role | Re-evaluate what the role grants |
| done | `permission.created` / `permission.updated` / `permission.deleted` | Permission lifecycle | Refresh cached permission catalog |
| done | `iam.policy.updated` | Policy updated | Invalidate cached policy bundle (already emitted) |
| done | `policy.created` / `policy.deleted` | Policy lifecycle | Complete policy cache invalidation |
| done | `iam.service.policy.assigned` / `iam.service.policy.removed` | Service ↔ policy link | Service-principal cache invalidation (already emitted) |

### Group 3 — Tenant / organization (provisioning)

| Status | Event | Trigger | Why a consumer needs it |
| --- | --- | --- | --- |
| done | `tenant.created` | Tenant created | Provision the org downstream |
| done | `tenant.updated` | Tenant attributes change | Sync org metadata |
| done | `tenant.status_changed` | Tenant activated/suspended | Enable/disable org access |
| done | `tenant.deleted` | Tenant deleted | Deprovision the org |
| done | `tenant_member.added` / `tenant_member.removed` | Org membership change | Mirror who belongs to an org |

### Group 4 — OAuth clients & credentials (security / config sync)

Credential events signal *that* a secret changed; they never carry the value.

| Status | Event | Trigger | Why a consumer needs it |
| --- | --- | --- | --- |
| done | `client.created` / `client.updated` / `client.deleted` | Client lifecycle | Sync the set of valid client apps |
| done | `client.status_changed` | Client enabled/disabled | Stop honoring a disabled client |
| done | `client.secret_rotated` | Secret rotated | Expect new credentials (no secret in payload) |
| done | `api_key.created` | API key created | Track issued keys |
| done | `api_key.status_changed` | Key enabled/disabled | Honor/deny the key |
| done | `api_key.revoked` | Key revoked/deleted | Invalidate cached key immediately (security) |

### Group 5 — Sessions, identities & service principals

| Status | Event | Trigger | Why a consumer needs it |
| --- | --- | --- | --- |
| done | `session.revoked` / `token.revoked` | A session or token is revoked | Cache-eviction parity with `api_key.revoked`  |
| done | `identity.linked` / `identity.unlinked` | External identity linked/unlinked | Consumers that track federated identities |
| done | `api.created` / `api.updated` / `api.status_changed` / `api.deleted` | IAM resource-server (API) lifecycle | Gateways caching the API/audience catalog |
| done | `service.created` / `service.updated` / `service.status_changed` / `service.deleted` | Service-principal lifecycle | Service-to-service consumers; status/delete is security-relevant |

## Not Exposed As Integration Events (and why)

These actions happen in the app but are deliberately **not** integration events.
Exposing every action would flood consumers with data they never use.

| Area | Examples | Why not exposed |
| --- | --- | --- |
| Authentication activity | login, logout, token issue/refresh, MFA enroll/verify, OAuth authorize/consent | Audit-plane concern (`auth_events`), for the tenant's SIEM. (Revocation is the exception: `session.revoked`/`token.revoked` are exposed as cache-eviction events in Group 5; the audit record stays in `auth_events`.) |
| Account recovery & verification | password reset, email verification, magic link | Security/audit-plane; carries sensitive tokens; redacted audit only |
| Tenant & security settings | rate-limit, audit, maintenance, feature-flags, password/session/lockout/threat policy | Internal configuration; no external service mirrors it |
| Notifier config | email config, SMS config | Internal delivery config; secret-bearing |
| Branding & templates | branding, email/login/SMS templates | Presentation config; nothing external syncs it |
| Signup flows & IP rules | signup flow config, IP restriction rules | Internal policy/config |
| Identity provider config | IdP create/update/status/delete | Internal config (add to Group 5 only if a consumer needs it) |
| Setup / bootstrap | setup complete, control-service register, create admin/profile | One-time internal bootstrap |
| Self-service preferences | user settings, profile-only edits | Covered by `user.*` where externally relevant, otherwise internal/audit. (Session *revocation* is a Group 5 event.) |
| Reads & per-request firehose | all `GET`s, "one event per request" | Huge volume, no integration consumer; audit-plane only if ever needed |
| Authorization decisions | PDP allow/deny per request | Telemetry/audit, very high volume; not a data-change event |

## Implementation Checklist

All items are v1.0.0.

| Status | Work item                                           | Done criteria                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| -------------- | --------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| done           | Canonical event envelope                            | Shared fields: `event_id` (also the receiver dedup/idempotency key — no separate field), `event_type`, `event_version`, `tenant_id`, actor, subject/target, **changed-field names** (no values), `occurred_at`, trace/request ID. Thin by design; every event fits the envelope.                                                                                                                                                                                                                                 |
| done           | Event routing registry (normalized)                 | Add `event_types` (catalog, seeded), `webhook_endpoint_events` (M:N), `event_routes` (broker, per tenant), `tenant_event_types` (per-tenant master switch); edit migration 007 in place to drop `events JSONB` and add `subscribe_all`. Cache the write-gate state per tenant (global active types, per-tenant "off" set, "has any listener") with invalidation on change.                                                                                                                                       |
| done           | Multi-replica cache invalidation                    | Redis pub/sub channel broadcasting `(tenant_id, change)` on any routing-config write; each replica drops its cached gate state. Short-TTL fallback when Redis is absent.                                                                                                                                                                                                                                                                                                                                         |
| done           | Tenant-scoped config APIs (frontend)                | Admin-scoped, tenant-isolated endpoints: list `event_types`; set tenant master switches (`tenant_event_types`); set webhook endpoint subscriptions (`webhook_endpoint_events` / `subscribe_all`); set broker routes (`event_routes`). **Generate the signing secret server-side, return once, store encrypted.** **Rate-limit + cap endpoint creation.** Keep SSRF validation on create/update. Every write invalidates the tenant cache.                                                                        |
| done           | Emission gate (coarse write + delivery filter)      | Write gate resolved from cache **before** opening the tx (global active + not tenant-disabled + tenant has any listener) → never deadlocks/aborts the mutation. Exact per-type match applied at delivery. Tests: no-listener tenant writes zero rows; adding a listener replays earlier events; outbox payload contains **no resource value fields** (PII guard).                                                                                                                                                |
| done           | Internal event bus interface                        | Publisher/subscriber boundary + noop/in-memory test impl so feature packages emit without depending on webhook/broker internals.                                                                                                                                                                                                                                                                                                                                                                                 |
| done           | Transaction-bound outbox (replaces fire-and-forget) | Create-only outbox table written **inside** the mutation transaction. This is a NEW integration-emit call at the Group 1–5 mutations (not a conversion of the ~18 audit `Log` sites, which stay post-commit). The real per-site cost is computing the **changed-field-name diff** from old/new state inside the mutation. **Phase it: build the whole spine end-to-end for `user.updated` first** (envelope, gate, outbox, relay, history, retry, DLQ), prove atomicity + delivery, then fan out group by group. |
| done           | Outbox relay → webhook + broker                     | A relay reads unpublished outbox rows and dispatches to the webhook dispatcher and the RabbitMQ publisher, replacing the direct in-process dispatch from `Log`. Removes the drop-on-overflow loss path.                                                                                                                                                                                                                                                                                                          |
| done           | RabbitMQ publisher                                  | Adapter behind the event-bus interface. Publishes per-tenant `event_routes` matches to the configured exchange/queue; preserves `tenant_id`/`event_id`; disables cleanly when RabbitMQ config is absent. Kafka is out of scope.                                                                                                                                                                                                                                                                                  |
| done           | Webhook delivery history                            | Durable records: endpoint UUID, event ID, event type, attempt count, response status/summary, error reason, `next_retry_time`, final status, timestamps. For support, replay, and DLQ.                                                                                                                                                                                                                                                                                                                           |
| done           | Durable retry scheduler                             | Replace the in-memory backoff loop with retries driven by `next_retry_time`, so retries survive process restarts.                                                                                                                                                                                                                                                                                                                                                                                                |
| done           | Dead-letter + endpoint quarantine                   | Final failure state after retry exhaustion, with enough context to inspect/replay. Auto-disable (quarantine) an endpoint after sustained failures. **Poison-message handling:** an outbox row that cannot be serialized/delivered goes straight to the DLQ and must never wedge the relay head-of-line.                                                                                                                                                                                                          |
| done           | Receiver idempotency header                         | Add a stable `X-Maintainerd-Event-Id` header (= envelope `event_id`) for receiver dedup; document that `X-Maintainerd-Delivery` is per-attempt.                                                                                                                                                                                                                                                                                                                                                                  |
| done           | Replay tooling                                      | Replay one delivery, one event, or a tenant-scoped range. Replays preserve idempotency and are marked as replays.                                                                                                                                                                                                                                                                                                                                                                                                |
| done           | Retention & erasure                                 | Retention runner for outbox/delivery-history/DLQ per the policy above; right-to-erasure purges rows referencing a hard-deleted subject UUID.                                                                                                                                                                                                                                                                                                                                                                     |
| done           | gRPC parity                                         | Domain-event emission lives in the service layer so gRPC mutations emit the same events as REST.                                                                                                                                                                                                                                                                                                                                                                                                                 |
| done           | Event schema + versioning docs                      | Document envelope fields, naming convention, versioning, redaction rules, the delivery contract, and an example payload per event. Resolve the snake_case (`authn_login_success`) vs dotted (`user.updated`) naming for new vs existing constants.                                                                                                                                                                                                                                                               |

## Resolved Decisions

- **Emission model:** persist-when-listened + filter-at-delivery (coarse write
  gate), with the per-tenant master switch for cost control. Matches
  Stripe/WorkOS/Svix; preserves replay/backfill.
- **Payload shape:** thin, changed-field **names** only (no values); consumers
  refetch. Enforced by test.
- **Both channels in v1.0.0:** webhooks and the per-tenant RabbitMQ broker.
- **Per-tenant master switch:** kept (`tenant_event_types`), default-on.
- **Multi-replica cache invalidation:** Redis pub/sub, short-TTL fallback.
- **Retention/erasure:** retention runner + UUID-keyed purge (see above).
- **Kafka:** out of scope; RabbitMQ is the only broker.

## Open Decisions

- Should webhook/broker config changes emit their own integration events? Useful
  for audit, but guard against notification loops.
- Naming convention: keep new events dotted (`user.updated`) and leave existing
  audit constants (`authn_login_success`) as-is, or introduce versioned aliases?
