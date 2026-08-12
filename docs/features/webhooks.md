# Webhooks
> Per-tenant outbound HTTP delivery of integration events, signed with HMAC-SHA256, driven off the transactional outbox with durable retries, dead-lettering, and auto-quarantine.

| | |
|---|---|
| **Status** | Implemented. (Broker/RabbitMQ arm is structurally wired but inert unless an AMQP publisher is injected — see [Related](#related). This doc covers the webhook arm.) |
| **Code** | `internal/webhook` (endpoints, subscriptions, delivery history, signing, SSRF guard); `internal/app/webhook_delivery.go` (dispatcher/attempt logic); `internal/event` (outbox relay, background retrier, retention) |
| **Endpoints** | `internal/webhook/routes.go` — `/api/v1/webhook-endpoints*` and `/api/v1/webhook-replay` on the internal/control API (port 8080) |
| **Storage** | `webhook_endpoints`, `webhook_endpoint_events`, `webhook_delivery_history`, `integration_event_outbox`, `event_types` (GORM `AutoMigrate` — no SQL migration files) |
| **Config** | Per-endpoint: `url`, `subscribe_all`, `max_retries`, `timeout_seconds`, `status`, `description`. No env vars govern webhook delivery. |

## Overview

A tenant registers one or more HTTPS endpoints, subscribes each to a set of event types (or `subscribe_all`), and receives a signed `POST` whenever a subscribed event occurs. Payloads are **thin** — identifiers and changed-field *names* only, never field values (`internal/event/envelope.go:10`) — so consumers re-fetch current state from the API and out-of-order delivery is harmless.

Delivery is decoupled from the request that produced the event via a **transactional outbox**: a business mutation writes an `integration_event_outbox` row in the same DB transaction (`event.IntegrationEvent` → `ToOutbox`, e.g. `internal/iam/service_role.go:304`). A background **relay** polls the outbox and fans out to subscribed endpoints; per-endpoint retries live in `webhook_delivery_history`, driven by a **background retrier**. This is the same register-subscribe-sign-ack model as Stripe/GitHub/Svix.

Webhook management is a control-plane surface: the routes are mounted on the internal API (port 8080, VPN access only — `internal/server/router.go:32`), gated by `webhook-endpoint:*` permissions, and scoped to the caller's tenant.

## How it works

End-to-end path for one event:

1. **Emit (transactional).** A service mutation calls `eventService.Emit(ctx, tx, IntegrationEvent)` inside its DB transaction, persisting an `integration_event_outbox` row (`internal/event/envelope.go:72`). The event either commits with the mutation or not at all.
2. **Claim.** The `Relay` polls every 5s, claiming up to 50 unpublished rows with `FOR UPDATE SKIP LOCKED` so multiple replicas never process the same row (`internal/event/relay.go:87`). Rows are processed with concurrency 4.
3. **Two independent arms.** Each outbox row carries per-arm state (`webhook_delivered_at`, `broker_published_at`). The webhook arm and broker arm each run at-most-once to completion; a row is marked fully published only when both are done, so a broker outage never re-fires the webhook fan-out on re-claim (`internal/event/relay.go:117`).
4. **Fan-out.** `deliverToWebhooks` loads the tenant's **active** endpoints, filters by subscription (`subscribe_all` or an exact `event_types.key` match), and creates one `pending` `webhook_delivery_history` row per matching endpoint (`internal/app/webhook_delivery.go:47`). First attempts run with bounded concurrency (8) so one slow endpoint can't serialize the fan-out.
5. **Attempt.** `attemptOnce` decrypts the endpoint secret, computes the signature, sets the delivery headers, validates the destination (SSRF), and POSTs with a per-endpoint `timeout_seconds` deadline (`internal/app/webhook_delivery.go:141`). A `2xx`/`3xx` **status < 300** is success.
6. **Record outcome** (the sole place delivery state transitions):
   - **Success** → history `success`; endpoint `consecutive_failures` reset to 0.
   - **Retryable failure**, attempts remaining → history stays `pending` with a jittered `next_retry_time`.
   - **Attempts exhausted** (`attempt == max_retries + 1`) → history `dead_letter`; endpoint `consecutive_failures++`.
7. **Retry.** The `BackgroundRetrier` polls every 30s, loads up to 100 due `pending` rows (`final_status='pending' AND next_retry_time <= now`), skips rows whose endpoint is no longer `active`, and re-runs `attemptOnce` (`internal/event/retrier.go:99`, `internal/app/webhook_delivery.go:283`). Backoff is exponential full-jitter (`internal/app/webhook_delivery.go:248`).
8. **Quarantine.** When `consecutive_failures` reaches **10** (`quarantineThreshold`), the endpoint is set `quarantined` (inactive), and its still-`pending` deliveries are dead-lettered immediately so they aren't orphaned (`internal/app/webhook_delivery.go:185`).
9. **Retention.** A `RetentionRunner` runs every 6h: purges published outbox rows older than 7 days and terminal (`success`/`dead_letter`) delivery-history rows older than 90 days (`internal/event/retention_runner.go`).

### Signature & headers

Each delivery carries (`internal/app/webhook_delivery.go:224`):

| Header | Value |
|---|---|
| `X-Maintainerd-Event` | event type (e.g. `user.updated`) |
| `X-Maintainerd-Event-Id` | stable event ID — the same across retries; use as dedup key |
| `X-Maintainerd-Delivery` | per-delivery-row UUID |
| `X-Maintainerd-Attempt` | attempt number |
| `X-Maintainerd-Timestamp` | unix seconds at signing time |
| `X-Maintainerd-Signature-256` | `sha256=<hex>` |

Signature = `HMAC_SHA256(secret, "{timestamp}.{raw_body}")`, hex-encoded, prefixed `sha256=` (`computeWebhookSignature`, `internal/app/webhook_delivery.go:266`). Verify against the **raw** body bytes and reject stale timestamps to defeat replay.

### Payload envelope

```json
{
  "event_id":       "9c8e0b3e-...",
  "event_type":     "user.updated",
  "event_version":  1,
  "tenant_id":      42,
  "actor_user_id":  17,
  "subject_uuid":   "1b9d6bcd-...",
  "subject_type":   "user",
  "changed_fields": ["email", "status"],
  "payload":        {},
  "occurred_at":    "2026-06-06T10:15:30Z",
  "trace_id":       "...",
  "request_id":     "..."
}
```

Shape is `OutboxPayload` (`internal/event/delivery_adapter.go:43`). `payload` is an optional JSON object that must never carry PII/secret values.

## Implementation

### HTTP surface (control plane, port 8080, `/api/v1`)

Routes: `internal/webhook/routes.go`. All require `JWTAuthMiddleware` + `UserContextMiddleware` and a `webhook-endpoint:*` permission; tenant-scoped.

| Method & path | Handler | Permission |
|---|---|---|
| `GET /webhook-endpoints` | `WebhookEndpointHandler.GetAll` (filters: `status` comma-list, `url` ILIKE) | `webhook-endpoint:read` |
| `GET /webhook-endpoints/{uuid}` | `.Get` | `webhook-endpoint:read` |
| `POST /webhook-endpoints` | `.Create` (rate-limited + per-tenant cap) | `webhook-endpoint:create` |
| `PUT /webhook-endpoints/{uuid}` | `.Update` (`rotate_secret: true` → new secret) | `webhook-endpoint:update` |
| `DELETE /webhook-endpoints/{uuid}` | `.Delete` (soft delete) | `webhook-endpoint:delete` |
| `PATCH /webhook-endpoints/{uuid}/status` | `.UpdateStatus` (`active`/`inactive`) | `webhook-endpoint:update` |
| `GET /webhook-endpoints/{uuid}/subscriptions` | `SubscriptionHandler.ListSubscriptions` | `webhook-endpoint:read` |
| `POST /webhook-endpoints/{uuid}/subscriptions` | `.AddSubscription` (`{ "event_type_uuid": "..." }`, one at a time) | `webhook-endpoint:update` |
| `DELETE /webhook-endpoints/{uuid}/subscriptions` | `.RemoveSubscription` (`{ "event_type_uuid": "..." }`) | `webhook-endpoint:update` |
| `GET /webhook-endpoints/{uuid}/deliveries` | `DeliveryHistoryHandler.GetDeliveries` | `webhook-endpoint:read` |
| `POST /webhook-replay` | `ReplayHandler.ReplayDelivery` | `webhook-endpoint:update` |

Notes:
- Handlers: `handler_endpoint.go`, `handler_subscription.go`, `handler_delivery_history.go`, `handler_replay.go`. Permissions seeded in `internal/setup/seeder/004_permission.go:355`.
- The subscription API takes **one `event_type_uuid` per call** (not an `event_types` array). Available types come from `GET /event-types`. Tenant isolation is enforced: the event type must belong to the caller's tenant (`handler_subscription.go:128`).
- **Replay** (`handler_replay.go:86`): body `{ "event_id": "...", "webhook_endpoint_id": "<optional endpoint uuid>" }`. With an endpoint UUID it replays to that one endpoint; omit it to replay to all the tenant's active endpoints. Replays create fresh `is_replay=true` history rows with the same `event_id`. `newReplayFn` enforces that the outbox event's `tenant_id` matches the endpoint's tenant, preventing cross-tenant replay (`internal/app/webhook_delivery.go:329`). (The route is `/api/v1/webhook-replay`; a stale doc comment in the handler says `/webhook-endpoints/replay` — the router is authoritative.)

### Dispatcher & background workers

- `internal/app/webhook_delivery.go` — `deliverToWebhooks` (fan-out), `attemptOnce` (single attempt + all state transitions), `doDeliveryRequest` (HTTP + SSRF), `computeWebhookSignature`, `jitteredBackoff`, `newRetryDeliveryFn`, `newReplayFn`, `newBrokerDeliverFn`, `deliveryRetrierAdapter`.
- `internal/event/relay.go` — `Relay` (5s poll, batch 50, concurrency 4; panic-restart loop).
- `internal/event/retrier.go` — `BackgroundRetrier` (30s poll, concurrency 4).
- `internal/event/retention_runner.go` — `RetentionRunner` (6h interval; outbox 7d, delivery history 90d).
- Wiring: `internal/app/services.go:163` (`deliverToWebhooks` closure), `:178` (relay), `:187` (retrier), `:193` (retention).

### Constants

| Constant | Value | File |
|---|---|---|
| `quarantineThreshold` | 10 consecutive dead-letters | `webhook_delivery.go:30` |
| `maxInlineDeliveryConcurrency` | 8 | `webhook_delivery.go:34` |
| `webhookSuccessMaxStatus` | 300 (success = status < 300) | `webhook_delivery.go:25` |
| `webhookMaxBackoff` | 60s (backoff cap) | `webhook_delivery.go:26` |
| `relayPollInterval` / `relayBatchSize` / `relayMaxConcurrency` | 5s / 50 / 4 | `relay.go:10` |
| `retryPollInterval` | 30s | `retrier.go:15` |
| `retentionInterval` / `outboxRetentionDays` / `deliveryHistoryRetentionDays` | 6h / 7 / 90 | `retention_runner.go:15` |
| `maxEndpointsPerTenant` | 50 | `handler_replay.go:15` |

### Data model

**`webhook_endpoints`** (`model_endpoint.go`): `webhook_endpoint_id`, `webhook_endpoint_uuid`, `tenant_id`, `url`, `secret_encrypted` (`json:"-"`), `subscribe_all`, `max_retries` (default 3), `timeout_seconds` (default 30), `description`, `metadata` (jsonb), `status` (default `active`), `consecutive_failures` (default 0), `last_triggered_at`, `created_by`/`updated_by`, timestamps, `deleted_at` (soft delete). Status values in use: `active`, `inactive`, `quarantined`.

**`webhook_endpoint_events`** (`model_endpoint_event.go`): M:N join of endpoint → `event_type_id`. Subscription matching resolves the event by canonical `event_types.key` via `ExistsByEndpointAndEventKey` (`repository_endpoint_event.go:44`).

**`webhook_delivery_history`** (`model_delivery_history.go`): `delivery_history_uuid`, `webhook_endpoint_id`, `event_id`, `event_type`, `tenant_id`, `attempt_count`, `response_status`, `response_summary`, `error_reason`, `next_retry_time`, `final_status` (`pending`/`success`/`dead_letter`), `is_replay`, timestamps. Retention deletes only terminal rows (`DeleteOlderThan`, `repository_delivery_history.go:117`).

**`integration_event_outbox`** (`internal/event/model_outbox.go`): the durable event log; per-arm columns `webhook_delivered_at` / `broker_published_at` / `claimed_at` decouple delivery arms.

## Configuration

No environment variables govern webhook delivery — everything is per-endpoint state stored in the DB and set through the management API. Validation (`validation_endpoint.go`):

| Field | Rule |
|---|---|
| `url` | Required; must be a valid **HTTPS** URL; literal-IP blocklist enforced; best-effort DNS pre-check (2s, non-blocking on DNS failure) |
| `max_retries` | Optional; `0`–`10` (default 3 → total attempts = `max_retries + 1`) |
| `timeout_seconds` | Optional; `1`–`120` (default 30) |
| `description` | Optional; ≤ 500 chars |
| `status` | Optional on create/update; `active` or `inactive` (`quarantined` is set only by the system) |
| `subscribe_all` | Bool; when true, receives every subscribed-eligible event without per-type rows |

Per-tenant cap: **50** endpoints, enforced by `RateLimitAndCapMiddleware` on create, which **fails closed** (503) if the count can't be verified and returns 429 when the cap is hit (`handler_replay.go:21`).

## Security considerations

- **SSRF hardening (defense in depth).** URLs must be HTTPS. At registration, literal-IP hosts are checked against a blocklist and hostnames get a best-effort DNS check (`validation_endpoint.go:12`). The authoritative guard is at **delivery time**: `SafeDeliveryClient` resolves the host **once inside `DialContext`**, rejects if *any* resolved address is loopback/private/link-local/unspecified/multicast or in the extra blocked ranges (RFC 6598 CGNAT, RFC 6890, RFC 2544, NAT64 prefixes), then dials the **pinned IP literal** — closing the DNS-rebinding/TOCTOU window (`deliver_client.go:44`, `security_url.go:105`). Redirects reuse the pinned transport and re-enforce HTTPS on every hop (`deliver_client.go:34`). TLS still verifies against the original hostname.
- **Signing secret.** 48-char CSPRNG value (`crypto.GenerateRandomString`), stored **encrypted at rest** (`crypto.EncryptAtRest`), returned **once** on create and on `rotate_secret` — never again (`service_endpoint.go:145`). Excluded from all reads via `json:"-"` and omitted from the response DTO on non-create paths.
- **Tenant isolation.** Every lookup is scoped by `FindByUUIDAndTenantID`; subscriptions and replays reject event types / outbox events belonging to another tenant (`handler_subscription.go:128`, `webhook_delivery.go:345`).
- **Thin payloads.** Only IDs and changed-field names cross the wire; consumers must re-fetch. Keeps user data off third-party endpoints.
- **At-least-once delivery.** Duplicates are possible (retries, replays, per-arm re-claim); consumers must dedup on `event_id`.
- **Auto-quarantine** protects both sides: a dead endpoint stops being retried for every event after 10 consecutive dead-letters. Reactivating an endpoint (`Update`/`UpdateStatus` back to `active`) resets `consecutive_failures` so it doesn't immediately re-quarantine (`service_endpoint.go:210`).
- **Transport auth of the receiver is the consumer's job:** verify the HMAC signature and the timestamp window on every request using the raw body.

## Related

- `./events.md` — the integration-event / transactional-outbox producer side (envelope, `Emit`, relay).
- `./events.md` — the sibling broker delivery arm off the same outbox (`newBrokerDeliverFn`).
- `./events.md` — the per-tenant event-type catalog that subscriptions resolve against.
- `./secret-management.md` — `crypto.EncryptAtRest` used for the signing secret at rest.
</content>
</invoke>
