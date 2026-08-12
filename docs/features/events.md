# Event Bus (Integration Events)

> A transactional-outbox event plane that emits thin, at-least-once integration events on IAM/tenant/client data changes and delivers them to two independent arms — HTTP webhooks and a RabbitMQ topic exchange.

| | |
|---|---|
| **Status** | Implemented. RabbitMQ broker arm is **inert until `RABBITMQ_URL` is set** (cleanly no-ops); webhook arm works independently. |
| **Code** | `internal/event` (outbox, relay, write gate, RabbitMQ publisher, AMQP, retention, config/route APIs); wired in `internal/app/services.go`; broker filter in `internal/app/webhook_delivery.go` |
| **Endpoints** | Control port `:8080`, `/api/v1`: `GET /event-types`, `GET|PUT /tenant-event-types`, `GET|POST|PUT|DELETE /event-routes` (all require `webhook-endpoint:*` permissions) |
| **Storage** | `integration_event_outbox`, `event_types`, `tenant_event_types`, `event_routes` (migrations 074, 078, 079, 080) |
| **Config** | `RABBITMQ_URL` (env; unset = broker disabled). Per-tenant: event-type master switch + per-type broker routes. No `RABBITMQ_EXCHANGE` / `RABBITMQ_TLS` env vars exist. |

## Overview

When IAM/tenant/client data changes in maintainerd-auth (a user is created, a role's permissions change, a client is deleted), downstream services that **mirror but do not own** that data need a signal to re-sync. That is the job of the **integration event plane** implemented in `internal/event`.

This is distinct from the **audit plane** (`auth_events`): audit records are a durable security log (logins, MFA, token issuance) and are **never** fanned out to webhooks or the broker (`internal/app/services.go:206-209`). Integration events use dotted names (`user.updated`); audit events use snake_case (`authn_login_success`) — see `internal/event/doc.go:4-7`.

The plane is a classic **transactional outbox**: the emitting service writes an event row inside the same DB transaction as the business mutation, and a background **relay** later reads unpublished rows and dispatches them to two independent delivery arms:

- **Webhook arm** — HTTP `POST` to registered endpoints (owned by the `webhook` package).
- **Broker arm** — publish to a RabbitMQ topic exchange (this doc's focus).

Both arms carry the **same thin envelope** and the **same at-least-once, unordered** contract; only the transport differs.

## How it works

End-to-end path for one event:

1. **Emit (in-transaction).** A domain service builds an `IntegrationEvent` (`internal/event/envelope.go:28-63`) and calls `EventService.Emit(ctx, tx, event)` with the caller's mutation transaction (`internal/event/service_event.go:46`). Callers include IAM, tenant, client, user, and IDP services (e.g. `internal/iam/service_role.go`, `internal/client/service_client.go`).
2. **Write gate.** `Emit` first calls `WriteGate.ShouldEmit` (`internal/event/write_gate.go:130`). If the gate is closed, nothing is written and `Emit` returns `(nil, nil)` — no outbox row, no delivery. The gate is a cached three-level check (see below).
3. **Outbox insert.** If the gate passes, the event is converted to an `Outbox` row (`envelope.go:72` → `ToOutbox`) and inserted via the tenant-scoped `OutboxRepository.Create` using the caller's `tx` (`service_event.go:67-79`). Because the insert shares the business transaction, the event is persisted **atomically** with the change it describes — no lost or phantom events.
4. **Relay poll + claim.** The `Relay` polls every 5s (`relay.go:11,73-85`). Each tick calls `OutboxRepository.ClaimUnpublished(50)` (`relay.go:90`), which runs a single `UPDATE ... RETURNING` with `FOR UPDATE SKIP LOCKED` (`repository_outbox.go:64-79`). This atomically claims a batch and sets `claimed_at = now()`, so concurrent relay replicas never process the same row. A claimed row is hidden for **5 minutes**; if delivery never completes, the claim expires and the row is re-claimable.
5. **Per-arm delivery.** For each claimed row, `deliverOne` runs the webhook and broker arms **independently** (`relay.go:117-158`). Each arm is skipped if already done — the row carries per-arm timestamps `webhook_delivered_at` / `broker_published_at`. On success each arm calls its `Mark…` method (`repository_outbox.go:100-108`). This decoupling means a broker outage never re-drives the webhook arm on re-claim (which would fan out duplicate HTTP deliveries).
6. **Broker route filter.** The broker arm (`newBrokerDeliverFn`, `internal/app/webhook_delivery.go:374-399`) publishes **only** when the publisher is enabled **and** the tenant has an `event_routes` row for that event type with `enabled = true`. Otherwise it no-ops (returns nil → arm marked done).
7. **AMQP publish.** `RabbitMQPublisher.Publish` (`rabbitmq_publisher.go:39-57`) marshals the envelope to JSON and publishes to exchange `maintainerd-auth.events` with **routing key = the bare event type** (e.g. `user.updated`) and AMQP `message-id = event_id`. The publish uses **publisher confirms** and `mandatory=true`; it blocks until the broker acks (`amqp.go:117-145`). A nack or unroutable/return leaves the row unpublished for re-claim.
8. **Mark published.** Only when **both** arms are done does `MarkPublished` set `is_published = true, published_at = now()` (`relay.go:153-157`). A partially-delivered row stays unpublished and only the incomplete arm re-runs on the next poll (at-least-once, per-arm).
9. **Consume.** Consumers bind a queue to `maintainerd-auth.events` with topic patterns (`user.*`, `tenant.#`, `#`), dedup on `message-id` (`event_id`), and **re-fetch current state from the API** — the payload is a signal, not the source of truth. A `404` on re-fetch means the resource was deleted (a valid terminal outcome).

### The write gate (three levels of enablement)

`WriteGate.ShouldEmit` (`write_gate.go:130-155`) returns true only when all three hold:

| Level | Source | Cached in gate |
|---|---|---|
| **Global type active** | `event_types.is_active` per key (`FindAllActive`) | `activeTypes` map, refreshed every 25s |
| **Tenant master switch** | `tenant_event_types` disabled keys (`FindDisabledKeysByTenantID`) | `tenantDisabledKeys[tenantID]` |
| **At least one listener** | any active webhook endpoint OR any enabled `event_routes` row (`listenerChecker.HasAnyActiveListener`, `internal/app/services.go:120-133`) | `tenantHasListener[tenantID]` |

Cache invalidation: when Redis is present the gate subscribes to the pub/sub channel `event:gate:invalidate` (`write_gate.go:15,66,109-126`) so a config change on one replica clears per-tenant state on all replicas; `InvalidateTenant` publishes the tenant ID (`write_gate.go:231-244`). When Redis is absent (`ttlOnly`), per-tenant state is fully flushed on the 25s refresh tick instead (`write_gate.go:99-104`). Errors fail **open** (assume a listener exists) so a lookup failure never silently drops events (`write_gate.go:171-177`).

### Background workers

Three long-lived goroutines are started at service init (`internal/app/services.go:178-196`):

| Worker | Interval | Job |
|---|---|---|
| `Relay` (`relay.go`) | 5s poll | Claim + deliver unpublished outbox rows; panic-recovering loop |
| `BackgroundRetrier` (`retrier.go`) | 30s poll | Re-attempt pending **webhook** `delivery_history` rows (broker uses re-claim, not this) |
| `RetentionRunner` (`retention_runner.go`) | 6h | Purge published outbox rows >7d and delivery history >90d |

## Implementation

Key files in `internal/event`:

| File | Role |
|---|---|
| `envelope.go` | `IntegrationEvent` canonical envelope + builders + `ToOutbox` |
| `service_event.go` | `EventService.Emit` — write-gate check then in-tx outbox insert |
| `write_gate.go` | `WriteGate` three-level cached enablement + Redis invalidation |
| `relay.go` | `Relay` — poll, claim, per-arm decoupled delivery, mark-published |
| `repository_outbox.go` | `ClaimUnpublished` (`FOR UPDATE SKIP LOCKED`), `Mark*`, retention deletes |
| `model_outbox.go` | `Outbox` GORM model → table `integration_event_outbox` |
| `amqp.go` | `ConnectAMQP` — dial (retry/backoff), declare topic exchange, publisher confirms, credential-redacted logging |
| `rabbitmq_publisher.go` | `RabbitMQPublisher.Publish` — `BrokerDeliveryFunc`; exchange `maintainerd-auth.events`, routing key = event type |
| `delivery_adapter.go` | `DeliveryAdapter` bridges relay arms to webhook/broker fns; `OutboxPayload` builds the wire map |
| `retrier.go` | `BackgroundRetrier` — durable webhook retry scheduler |
| `retention_runner.go` | `RetentionRunner` — periodic outbox + delivery-history purge |
| `constants.go` | Event-type catalog (`AllEventTypes`) + category constants |
| `handler_config.go` / `service_config.go` | `/event-types`, `/tenant-event-types` REST |
| `service_management.go` / `routes.go` | `/event-routes` REST + route registration |
| `model_event_route.go` | `EventRoute` model → table `event_routes` |

**Exchange topology:** exchange `maintainerd-auth.events`, type `topic`, durable, non-auto-delete, declared on connect (`amqp.go:78-90`). Messages are `DeliveryMode: Persistent`, `ContentType: application/json` (`amqp.go:117-130`).

### Event catalog

`AllEventTypes()` seeds **42** event types across 5 groups (`constants.go:72-126`). (The `doc.go:37` package comment says "48" — that count is stale; the authoritative list is `AllEventTypes`, which enumerates 42.) Groups: User identity (6), Authorization model / IAM (12), Tenant/organization (7), OAuth clients (5), Sessions/identities/service principals (12). The **live** set for a tenant is `GET /event-types`. `event_types` is **per-tenant** (`event_types.tenant_id`, unique `(tenant_id, key)` — migration 074), so each tenant is seeded its own catalog rows.

### Envelope / wire payload

`IntegrationEvent` (`envelope.go:12-25`) and the JSON published to the broker (`OutboxPayload`, `delivery_adapter.go:43-58`):

| Field | Notes |
|---|---|
| `event_id` (UUID) | Dedup key; also AMQP `message-id` |
| `event_type` | e.g. `user.updated`; also the routing key + AMQP `type` |
| `event_version` (int) | Schema version, starts at 1 |
| `tenant_id` (int64) | Present in every message; consumers filter by it |
| `actor_user_id` | Who triggered the change (nullable) |
| `subject_uuid` / `subject_type` | The changed resource |
| `changed_fields` | Field **names** only (e.g. `["email","status"]`) — never values |
| `payload` | Optional JSON metadata; must not contain PII values |
| `occurred_at`, `trace_id`, `request_id` | Timing + correlation |

### Endpoints

All under control port `:8080`, prefix `/api/v1` (`internal/server/router.go:105`), JWT + tenant context + permission-gated (`routes.go`):

| Method & path | Permission | Purpose |
|---|---|---|
| `GET /event-types` | `webhook-endpoint:read` | List active event types for the tenant |
| `GET /tenant-event-types` | `webhook-endpoint:read` | Get per-tenant master-switch config |
| `PUT /tenant-event-types` | `webhook-endpoint:update` | Enable/disable a type for the tenant. Body: `{ "event_type_uuid": "...", "enabled": bool }` |
| `GET /event-routes` | `webhook-endpoint:read` | List the tenant's broker routes |
| `GET /event-routes/{event_route_uuid}` | `webhook-endpoint:read` | Get one route |
| `POST /event-routes` | `webhook-endpoint:create` | Enable a broker route. Body: `{ "event_type_uuid": "..." }`; `channel` is always `rabbitmq` (`service_management.go:133`) |
| `PUT /event-routes/{event_route_uuid}` | `webhook-endpoint:update` | Toggle `enabled` |
| `DELETE /event-routes/{event_route_uuid}` | `webhook-endpoint:delete` | Remove a route |

### Storage

| Table (migration) | Key columns |
|---|---|
| `integration_event_outbox` (080) | `outbox_id` PK, `outbox_uuid`, `event_id`, `event_type`, `event_version`, `tenant_id`, `actor_user_id`, `subject_uuid`, `subject_type`, `changed_fields` jsonb, `payload` jsonb, `occurred_at`, `trace_id`, `request_id`, `is_published`, `published_at`, `webhook_delivered_at`, `broker_published_at`, `claimed_at`, `created_at`. Partial indexes on unpublished rows drive the claim query. |
| `event_types` (074) | `event_type_id` PK, `event_type_uuid`, `tenant_id` (FK `tenants`, cascade), `key`, `category`, `description`, `version`, `is_active`. Unique `(tenant_id, key)`. |
| `tenant_event_types` (079) | Per-tenant master switch: `tenant_id`, `event_type_id`, `enabled`. |
| `event_routes` (078) | Per-tenant broker route: `tenant_id`, `event_type_id`, `channel` (default `rabbitmq`), `enabled`. |

## Configuration

**Environment (operator):**

| Var | Effect |
|---|---|
| `RABBITMQ_URL` | AMQP DSN, e.g. `amqp://user:pass@rabbitmq:5672/`. **Unset/empty → broker disabled** (publisher is a no-op; `NewAMQPConfigFromEnv` returns nil, `amqp.go:44-50`). Set → relay dials with exponential backoff, declares the exchange, enables publisher confirms. |

There is **no** `RABBITMQ_EXCHANGE` or `RABBITMQ_TLS` variable — the exchange name `maintainerd-auth.events` is hardcoded (`amqp.go:79`, `rabbitmq_publisher.go:51`) and only `RABBITMQ_URL` is read (`amqp.go:45`).

**Per-tenant (admin API):** an event reaches the broker only when: (1) the type's `event_types.is_active` is true, **and** (2) the tenant hasn't disabled it via `PUT /tenant-event-types`, **and** (3) an `event_routes` row exists for that tenant + type with `enabled = true`. Independently, at least one listener (webhook or route) must exist for the write gate to even persist the outbox row.

**Retention constants** (compiled, not env — `retention_runner.go:15-19`): outbox 7 days, delivery history 90 days, sweep every 6 hours.

## Security considerations

- **Thin events / no PII on the wire.** Payloads carry identifiers and changed-field **names** only, never values (`doc.go:16-26`, enforced by tests). Consumers re-fetch from the authenticated API, so a leaked/misrouted event never exposes resource data. `payload` and `changed_fields` must not carry values.
- **Broker credential redaction.** `RABBITMQ_URL` contains the broker password inline; the startup log strips userinfo via `redactAMQPURL` (`amqp.go:29-40`) so a live credential is never written to logs (regression-tested in `amqp_redact_test.go`).
- **Publisher confirms + mandatory.** Publishes block on broker ack and flag unroutable messages (`amqp.go:97-113,137-143`); a nack/return leaves the outbox row unpublished for safe re-claim, so a broker crash before persistence never silently loses an event while the row is marked published.
- **Exactly-once insert, at-least-once delivery.** The outbox row is written in the business transaction (no dual-write gap). Delivery is at-least-once and unordered; consumers **must** dedup on `event_id`. Per-arm decoupling prevents duplicate webhook fan-out during a broker outage (`relay.go:117-124`).
- **Concurrency safety.** `FOR UPDATE SKIP LOCKED` + a 5-minute claim visibility timeout make the relay safe to run in multiple replicas without double-delivery (`repository_outbox.go:64-79`).
- **Authorization.** All config/route mutations require `webhook-endpoint:*` permissions and are tenant-scoped via JWT + tenant-context middleware (`routes.go`), so one tenant cannot read or route another tenant's events.
- **Tenant isolation on publish.** Every message body carries `tenant_id`; consumers on the shared exchange must filter by it.

## Related

- `./webhooks.md` — the sibling HTTP delivery arm (same envelope, same contract; HMAC-signed).
- `./audit-logging.md` — the separate audit plane (`auth_events`); not fanned out to this bus.
- `./iam-authorization.md` — the IAM services that emit most integration events.
