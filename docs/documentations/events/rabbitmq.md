# RabbitMQ Event Streaming

**Status:** Available — v1.0.0 (broker activates once an AMQP connection is configured)
**Audience:** Engineers consuming Lula auth events from RabbitMQ in a first-party / internal service.
**See also:** [Events overview](./README.md) · [Webhooks](./webhooks.md)

RabbitMQ is the **broker channel** for the same integration events described in the
[Events overview](./README.md). It is intended for **internal / first-party services**
that already run on a message bus and want to consume auth data-change events at scale,
rather than receiving HTTP webhooks.

The payload, event catalog, and delivery semantics are **identical to webhooks** — only
the transport differs. If you have not read the [Webhooks guide](./webhooks.md), read
its payload and event-catalog sections first; this guide covers only what's RabbitMQ-specific.

---

## 1. Topology

| | |
|---|---|
| **Exchange** | `maintainerd.events` (topic) |
| **Routing key** | `event.<event_type>` — e.g. `event.user.updated`, `event.tenant.deleted` |
| **Message body** | The same thin JSON envelope as webhooks (see [Webhooks §3](./webhooks.md#3-the-event-payload)) |
| **Message ID** | `event_id` (AMQP `message-id` property) — your dedup key |
| **Delivery** | At-least-once; deduplicate on `event_id` |

Because all tenants publish to the same exchange, the **`tenant_id` is in every message
body** — consumers filter/route by it downstream.

---

## 2. Enabling the broker

There are two independent switches; both must be on for events to flow to RabbitMQ.

### 2a. Connect the broker (operator / deployment)

The publisher is wired into the delivery relay but **disabled until an AMQP connection
is configured**. When no connection is present it cleanly no-ops, so environments
without RabbitMQ are unaffected.

To activate it, provide the connection via environment configuration (same pattern as
`REDIS_*`):

```
RABBITMQ_URL=amqp://user:pass@rabbitmq:5672/     # unset/empty = broker disabled
RABBITMQ_EXCHANGE=maintainerd.events             # optional, defaults shown
RABBITMQ_TLS=false                               # optional
```

> **Note:** this repository ships the publisher and route-filtering logic; the AMQP
> client/connection is the deployment hook that injects the publish function. Until
> `RABBITMQ_URL` is set and the connection is wired, the broker channel is inert (the
> webhook channel is unaffected and works on its own).

### 2b. Enable events per tenant (route config)

Even with the broker connected, an event publishes to RabbitMQ **only when the tenant
has an enabled route for that event type**. Manage routes via the admin API
(`webhook-endpoint:*` permission, tenant-scoped):

| Method & path | Purpose |
|---|---|
| `GET /event-routes/` | List the tenant's broker routes |
| `POST /event-routes/` | Enable a broker route (`{ "event_type_id": ..., "destination": "..." }`) |
| `PUT /event-routes/{event_route_uuid}` | Update destination / toggle `enabled` |
| `DELETE /event-routes/{event_route_uuid}` | Remove a route |

```bash
# Enable broker delivery of an event type for this tenant
curl -X POST https://auth.internal:8080/api/v1/event-routes/ \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{ "event_type_id": 12, "destination": "maintainerd.events" }'
```

Get `event_type_id` values from `GET /event-types/`. The per-tenant master switch
(`PUT /tenant-event-types/`) still applies — a type disabled there is never produced for
the tenant, on any channel.

---

## 3. Consuming events

Bind a queue to the `maintainerd.events` topic exchange with the routing-key patterns you
care about (topic wildcards: `*` = one segment, `#` = zero or more):

```text
event.user.*            → all user events
event.user.updated      → only user.updated
event.tenant.#          → all tenant + tenant_member events
#                       → everything
```

Example (Go, `amqp091-go`):

```go
ch.QueueBind(
    "my-service.user-sync",   // your queue
    "event.user.*",           // routing key pattern
    "maintainerd.events",     // exchange
    false, nil,
)

for d := range deliveries {
    if seen(d.MessageId) {    // dedup on event_id
        d.Ack(false)
        continue
    }
    var ev IntegrationEvent
    json.Unmarshal(d.Body, &ev)

    // Re-fetch current state from the API — the event is a signal, not the truth.
    // A 404 means the resource was deleted.
    syncFromAPI(ev.TenantID, ev.SubjectType, ev.SubjectUUID)

    markSeen(d.MessageId)
    d.Ack(false)
}
```

**Consumer rules (same contract as webhooks):**
- **At-least-once** — deduplicate on `event_id` (the AMQP `message-id`).
- **Unordered** — do not assume per-tenant or global ordering; re-fetch resolves it.
- **Thin payload** — the body has IDs and changed-field *names*, not values. Re-fetch
  from the API for current data; handle `404` as "deleted."
- **Filter by `tenant_id`** in the body if your consumer is tenant-aware.
- Ack only after you have durably handled (or enqueued) the event.

---

## 4. Webhooks vs RabbitMQ — which to use

| | Webhooks | RabbitMQ |
|---|---|---|
| Best for | External apps, third parties, no infra | Internal/first-party services on a bus |
| Transport | HTTPS `POST` to your endpoint | AMQP topic exchange you consume |
| Setup | Register endpoint + subscribe | Connect broker + enable routes |
| Fan-out / throughput | Per-endpoint HTTP | Native pub/sub, high throughput, multiple consumer groups |
| Verification | HMAC signature header | Trusted internal bus (+ `tenant_id` in body) |

Both consume the **same events** and obey the **same delivery contract**. A tenant can
enable either, both, or neither per event type — see the
[Events overview](./README.md#how-enablement-works-three-levels).

---

## 5. Event catalog

Identical to the webhook catalog — see [Webhooks §7](./webhooks.md#7-event-catalog) for
the full list, or query `GET /event-types/` for the live set. Routing keys are
`event.<event_type>` (e.g. `event.role.permissions_changed`).
