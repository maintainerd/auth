# Events & Integration

**Status:** Available — v1.0.0
**Audience:** Engineers integrating an external or peer service with Lula auth so it stays in sync when auth data changes.

When data changes in Lula auth — a user is created, a role's permissions change, a
client is deleted — services that **reference but do not own** that data need to know,
so they can refresh their own mirror, invalidate a cache, or provision/deprovision.
This is the **integration event system**.

It delivers over two channels, and you choose per tenant which events go where:

| Channel | Best for | Guide |
|---|---|---|
| **Webhooks** | External apps and third-party integrations (HTTP, no infra to run) | [webhooks.md](./webhooks.md) |
| **RabbitMQ** | First-party / internal services on your own message bus | [rabbitmq.md](./rabbitmq.md) |

---

## Two planes — don't confuse them

Lula emits two *different* kinds of records. Know which one you want:

| | **Audit plane** (`auth_events`) | **Integration plane** (this system) |
|---|---|---|
| Purpose | Security/compliance log of *what happened* | Signals to *sync your data* |
| Examples | login, logout, MFA, token issued, OAuth consent | `user.updated`, `role.permissions_changed`, `tenant.deleted` |
| Consumer | The tenant's SIEM / security team | Other services that mirror auth data |
| Delivery | Stored, queried via the auth-events API | Webhooks + RabbitMQ (this system) |

Authentication activity (logins, MFA, etc.) is **audit-plane** and is **not** sent to
integration webhooks. If you need security monitoring, read the audit events instead.

---

## Core principles (what you can rely on)

- **Thin events.** Payloads carry **identifiers** (UUIDs) and the **names** of changed
  fields — never the values. On receiving an event you **re-fetch** current state from
  the API. This keeps PII off the wire and makes ordering largely irrelevant.
- **At-least-once, unordered.** You may receive duplicates and out-of-order events.
  **Deduplicate on `event_id`** and treat events as triggers, not as the source of truth.
- **Per-tenant configuration.** Every tenant independently chooses which events it emits
  and to which channel. Enabling an event for one tenant never affects another.

## How enablement works (three levels)

1. **Global catalog** — `event_types.is_active`. A platform-level kill switch. If an
   event type is retired here, no tenant emits it. (You can't override this per tenant.)
2. **Per-tenant master switch** — a tenant can hard-disable an event type so it is never
   produced for that tenant, regardless of channel config.
3. **Per-channel subscription** — a webhook endpoint subscribes to specific event types
   (or all), and/or a RabbitMQ route is enabled for specific types.

An event is produced for a tenant only when it passes the global switch **and** the
tenant's master switch **and** at least one channel is listening. It is then delivered
only to the channels/endpoints that subscribed to that exact type.

See the per-channel guides for the configuration APIs and the full event catalog.
