# Webhooks

**Status:** Available — v1.0.0
**Audience:** Engineers receiving Lula auth events over HTTP.
**See also:** [Events overview](./README.md) · [RabbitMQ](./rabbitmq.md)

Webhooks let your service receive an HTTP `POST` whenever data changes in Lula auth, so
you can keep a mirror in sync, invalidate a cache, or react to lifecycle changes. This
follows the same model as Stripe, GitHub, and Svix: you register an HTTPS endpoint,
subscribe it to event types, verify a signature on each delivery, and respond `2xx`.

---

## 1. Quickstart

1. **Register an endpoint** and **save the signing secret** (returned once):

   ```bash
   curl -X POST https://auth.internal:8080/api/v1/webhook-endpoints/ \
     -H "Authorization: Bearer $ADMIN_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "url": "https://your-service.example.com/webhooks/lula",
       "subscribe_all": false,
       "max_retries": 3,
       "timeout_seconds": 30,
       "description": "User mirror sync"
     }'
   ```

   The response includes `signing_secret` **once** — store it in your secret manager.
   It is never returned again (rotate via the update API if lost).

2. **Subscribe to the events you care about:**

   ```bash
   curl -X POST https://auth.internal:8080/api/v1/webhook-endpoints/{webhook_endpoint_uuid}/subscriptions \
     -H "Authorization: Bearer $ADMIN_TOKEN" \
     -d '{ "event_types": ["user.created", "user.updated", "user.deleted"] }'
   ```

   Or set `subscribe_all: true` on the endpoint to receive every enabled type.

3. **Receive, verify, and ack** (see [§4 Verifying](#4-verifying-a-delivery)). Respond
   with any `2xx` quickly; do heavy work asynchronously.

---

## 2. Managing endpoints (admin API)

All routes require an admin-tier token (`webhook-endpoint:*` permission) and are scoped
to the caller's tenant.

| Method & path | Purpose | Permission |
|---|---|---|
| `GET /webhook-endpoints/` | List endpoints | `webhook-endpoint:read` |
| `GET /webhook-endpoints/{uuid}` | Get one | `webhook-endpoint:read` |
| `POST /webhook-endpoints/` | Create (returns secret once; rate-limited + capped per tenant) | `webhook-endpoint:create` |
| `PUT /webhook-endpoints/{uuid}` | Update (set `rotate_secret: true` to issue a new secret) | `webhook-endpoint:update` |
| `PATCH /webhook-endpoints/{uuid}/status` | Enable/disable (`active` / `inactive`) | `webhook-endpoint:update` |
| `DELETE /webhook-endpoints/{uuid}` | Delete | `webhook-endpoint:delete` |
| `POST /webhook-endpoints/{uuid}/subscriptions` | Subscribe to event types | `webhook-endpoint:update` |
| `DELETE /webhook-endpoints/{uuid}/subscriptions` | Unsubscribe | `webhook-endpoint:update` |

Discover available event types via `GET /event-types/`. A tenant can also hard-disable
an event type for itself via `PUT /tenant-event-types/` (the per-tenant master switch).

**Endpoint constraints (security):**
- URLs **must be HTTPS**. URLs that resolve to loopback, private, link-local, or cloud
  metadata addresses are rejected at registration and at delivery (SSRF protection).
- There is a per-tenant cap on the number of endpoints.
- The signing secret is generated server-side with a CSPRNG, stored encrypted, and shown
  only once.

---

## 3. The event payload

Every delivery is a `POST` with a JSON body in this envelope. **It is thin** — IDs and
changed-field *names*, never field values:

```json
{
  "event_id":       "9c8e0b3e-5a2b-4f1a-9b7c-1f2e3d4c5b6a",
  "event_type":     "user.updated",
  "event_version":  1,
  "tenant_id":      42,
  "actor_user_id":  17,
  "subject_uuid":   "1b9d6bcd-bbfd-4b2d-9b5d-ab8dfbbd4bed",
  "subject_type":   "user",
  "changed_fields": ["email", "status"],
  "occurred_at":    "2026-06-06T10:15:30Z",
  "trace_id":       "4bf92f3577b34da6a3ce929d0e0e4736",
  "request_id":     "c0ffee00-1234-..."
}
```

- `event_id` — **stable, unique per event.** Use it as your idempotency / dedup key. It
  is the same across retries.
- `subject_uuid` / `subject_type` — the resource that changed. **Re-fetch it** from the
  API to get current values.
- `changed_fields` — names only, to hint what to refresh. Never contains secrets/PII.

> **Why thin?** It keeps user data off third-party endpoints and makes out-of-order
> delivery harmless — you always read the latest state from the API.

---

## 4. Verifying a delivery

Each request carries these headers:

| Header | Meaning |
|---|---|
| `X-Maintainerd-Event` | event type (e.g. `user.updated`) |
| `X-Maintainerd-Event-Id` | stable event ID — **your dedup key** |
| `X-Maintainerd-Delivery` | per-attempt delivery ID (changes on retry) |
| `X-Maintainerd-Attempt` | attempt number |
| `X-Maintainerd-Timestamp` | unix seconds when signed — for replay protection |
| `X-Maintainerd-Signature-256` | `sha256=<hex>` HMAC of `"{timestamp}.{raw body}"` |

**Verify every request:**

1. Reject if `X-Maintainerd-Timestamp` is outside your tolerance window (e.g. ±5 min) —
   stops replay attacks.
2. Compute `HMAC_SHA256(secret, "{timestamp}.{raw_request_body}")` and compare to
   `X-Maintainerd-Signature-256` using a **constant-time** comparison.
3. Use the **raw** body bytes for the HMAC (do not re-serialize parsed JSON).

```javascript
// Node.js (Express, raw body)
const crypto = require("crypto");

function verify(req, secret) {
  const ts  = req.header("X-Maintainerd-Timestamp");
  const sig = req.header("X-Maintainerd-Signature-256"); // "sha256=..."
  if (Math.abs(Date.now() / 1000 - Number(ts)) > 300) return false; // 5-min window

  const expected = "sha256=" + crypto
    .createHmac("sha256", secret)
    .update(ts + "." + req.rawBody)   // raw bytes, not JSON.stringify
    .digest("hex");

  const a = Buffer.from(sig), b = Buffer.from(expected);
  return a.length === b.length && crypto.timingSafeEqual(a, b);
}
```

```go
// Go
func verify(secret, timestamp string, body []byte, signature string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(timestamp + "."))
    mac.Write(body)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

---

## 5. Responding & idempotency

- Respond with any **`2xx`** to acknowledge. Any non-`2xx` (or a timeout past the
  endpoint's `timeout_seconds`) is treated as a failure and scheduled for retry.
- **Deduplicate on `event_id`.** Because delivery is at-least-once, you *will*
  occasionally get the same event twice — process it once.
- **Do work asynchronously.** Ack fast, then re-fetch and process. Slow handlers cause
  timeouts and unnecessary retries.
- **Re-fetch on receive.** Treat the event as "something about `subject_uuid` changed —
  go read it." A `404` on re-fetch means the resource was deleted — a valid terminal
  outcome, not an error.

---

## 6. Delivery, retries & failure handling

- **At-least-once, best-effort order.** No strict ordering guarantee; rely on re-fetch +
  `event_id` dedup.
- **Durable retries.** Failed deliveries are retried with **exponential backoff +
  jitter**, driven by a persisted schedule that **survives restarts**. The number of
  attempts follows the endpoint's `max_retries`.
- **Dead-letter.** After retries are exhausted, the delivery is moved to a dead-letter
  state, retained for inspection and replay.
- **Auto-quarantine.** An endpoint that fails repeatedly (consecutive failures) is
  automatically set to `quarantined` and stops receiving deliveries until you fix and
  re-activate it. A successful delivery resets the failure counter.
- **Delivery history.** Every attempt is recorded (status, response summary, error,
  next retry time) for support and debugging.

### Replay

Re-send a past event to one endpoint or to all of a tenant's active endpoints:

```bash
curl -X POST https://auth.internal:8080/api/v1/webhook-replay/ \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{ "event_id": "9c8e0b3e-...", "webhook_endpoint_id": "<endpoint-uuid, optional>" }'
```

Omit `webhook_endpoint_id` to replay to all active endpoints. Replays are flagged as
replays in delivery history and carry the same `event_id`, so your dedup still applies.

---

## 7. Event catalog

Subscribe to any of these. Names are stable and follow `resource.action`. (Use
`GET /event-types/` for the live list.)

### User identity
`user.created` · `user.updated` · `user.status_changed` · `user.deleted` ·
`user.role_assigned` · `user.role_removed`

### Authorization model
`role.created` · `role.updated` · `role.deleted` · `role.permissions_changed` ·
`permission.created` · `permission.updated` · `permission.deleted` ·
`policy.created` · `policy.deleted` · `iam.policy.updated` ·
`iam.service.policy.assigned` · `iam.service.policy.removed`

### Tenant / organization
`tenant.created` · `tenant.updated` · `tenant.status_changed` · `tenant.deleted` ·
`tenant_member.added` · `tenant_member.removed`

### OAuth clients & credentials
`client.created` · `client.updated` · `client.status_changed` · `client.deleted` ·
`client.secret_rotated` · `api_key.created` · `api_key.status_changed` · `api_key.revoked`

### Sessions, identities & service principals
`session.revoked` · `token.revoked` · `identity.linked` · `identity.unlinked` ·
`api.created` · `api.updated` · `api.status_changed` · `api.deleted` ·
`service.created` · `service.updated` · `service.status_changed` · `service.deleted`

> Credential events (`*.secret_rotated`, `api_key.*`) signal *that* a secret changed —
> they never contain the secret value. The `iam.policy.*` / `iam.service.policy.*` events
> are the cache-invalidation signals used by the
> [S2S authorization](../service-to-service-authorization/service-to-service-authorization.md)
> pull-and-cache pattern.

---

## 8. Best practices checklist

- [ ] Verify the signature **and** the timestamp window on every request.
- [ ] Use the **raw** request body for HMAC.
- [ ] Deduplicate on `event_id`.
- [ ] Respond `2xx` fast; process asynchronously.
- [ ] Re-fetch the resource from the API; treat `404` as "deleted."
- [ ] Subscribe to only the event types you need (less noise, less load).
- [ ] Store the signing secret in a secret manager; rotate it via the update API.
- [ ] Monitor for `quarantined` endpoints and the dead-letter state.
