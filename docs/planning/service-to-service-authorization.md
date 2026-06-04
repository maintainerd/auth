# Service-to-Service Authorization (IAM Enforcement)

**Status:** Planned — required for **v1.0.0**.
**Owner:** rseguma@lula.life
**Created:** 2026-06-04

This document specifies how Lula's auth server graduates from an IAM *control plane*
(it can already store services, APIs, permissions, policies, roles) into a real IAM
*enforcement engine* — the piece that actually answers **"is service A allowed to call
service B?"** by evaluating attached policies.

It is the design for combining the two distribution patterns the team selected:

- **Pattern 1 — Policy distribution (pull + ETag poll).** Each service pulls its policy
  bundle from the auth server at startup, evaluates decisions **locally**, and re-polls
  with an `ETag` so unchanged bundles cost a cheap `304`.
- **Pattern 3 — Push invalidation on change.** When a policy/assignment changes, the auth
  server fires a webhook so caching services refresh **immediately** instead of waiting for
  the next poll. Closes the revocation-staleness gap.

> The goal of #1 is near-zero per-request latency and availability if the auth server is
> briefly down. The goal of #3 is fast revocation. Together: **author centrally, decide
> locally, invalidate on change** — the same shape as OPA + a control plane, or AWS IAM's
> internal caching.

---

## 1. Where we are today

| Layer | Term | Status | Location |
|-------|------|--------|----------|
| Author / store policies | **PAP** (Administration) | ✅ exists | [`internal/iam/`](../../internal/iam/) — full CRUD for service/api/permission/policy/role |
| Policy document model | — | ✅ AWS-shaped | `PolicyDocument` in [`internal/iam/types.go`](../../internal/iam/types.go) (`version` + `statement[]` of `effect`/`action`/`resource`) |
| Enforce at the edge | **PEP** (Enforcement) | ✅ RBAC only | [`internal/platform/middleware/permission_middleware.go`](../../internal/platform/middleware/permission_middleware.go) — `hasAnyPermission(user, required)` |
| **Decide from policies** | **PDP** (Decision) | ❌ **missing** | — policy documents are validated and stored but **never evaluated** |
| Service as a principal | — | ❌ **missing** | `Service` is a catalog row, not an authenticating actor |

**Concrete gap.** A trace of every consumer of `Policy.Document` and `service_policies`
shows they appear only in CRUD repositories/services and the migration — never in a
decision path. So today, *attaching a policy to a service is inert*: it stores a row and
changes no runtime behavior. The doc comment in `types.go` already admits this:

> *"Action and resource values are not validated against existing permissions/services.
> Invalid values will simply result in no access."*

---

## 2. Target flow

Both ends ask — defense-in-depth / zero-trust. A's outbound check fails fast on the client
side; B's inbound check is authoritative and cannot be bypassed by a spoofed client.

```
                       ┌──────────────────────────────────────┐
                       │   maintainerd-auth                    │
                       │   • PAP: author policies (existing)   │
                       │   • PDP: evaluate (NEW)               │
                       │   • bundle endpoint + ETag (NEW)      │
                       │   • policy-change webhook (NEW)       │
                       └──────────────────────────────────────┘
        startup pull / ETag poll ▲          │ webhook push on change
        + decision (local)       │          ▼
                          ┌──────────────┐   ┌──────────────┐
   (1) "can I call B?"───▶│  Service A   │   │  Service B   │◀── (3) "may A call me?"
        local allow       │ (embeds PDP, │   │ (embeds PDP, │     local allow
                          │  caches      │   │  caches      │
                          │  bundle)     │   │  bundle)     │
                          └──────┬───────┘   └──────▲───────┘
                                 │  (2) gRPC + A's token│
                                 └──────────────────────┘
```

1. **A (outbound):** before calling B, A evaluates *its own* cached bundle: does A hold an
   `allow` for `action=serviceB:invoke`, `resource=serviceB:*`? If deny → don't even dial.
2. **gRPC call:** A attaches its service-account access token (sub = `serviceA`) on the
   request metadata.
3. **B (inbound):** B extracts A's identity from the token, then evaluates **A's** bundle
   (which B also caches) for the same question. This is the authoritative check.

> **v1 simplification — identity policies only.** We do *not* add resource policies
> (a `principal` field on statements) in v1. Both checks ask the same question against
> **A's identity policy**; B just re-verifies A authoritatively. This needs no schema
> change to `PolicyDocument`. Resource policies (B independently controls its callers) are
> deferred — see §9.

---

## 3. Component 1 — Service principals (identity)

For "A calls the auth server to confirm," A must authenticate **as itself**. Reuse the
existing OAuth `client_credentials` grant ([`internal/oauth/`](../../internal/oauth/),
already ✅ in v1.0.0).

- Each `Service` that participates is linked to an OAuth **client** (`client_credentials`).
- The issued access token's `sub` (or a dedicated `svc` claim) carries the **service
  identifier** (e.g. `serviceA`, matching `Service.Name`).
- A scope or claim marks it a service token so middleware can branch from user-context.

**Schema touch:** add a nullable `service_id` FK on the OAuth client model (or a join), so a
client can resolve to its owning `Service`. Follow the create-only migration rule — edit the
client's original create migration in place (per
[`docs/contributing/database-migrations.md`](../contributing/database-migrations.md)).

---

## 4. Component 2 — The PDP (evaluation engine)

New file: `internal/iam/policy_evaluator.go` (+ `policy_evaluator_test.go`). Pure, no I/O —
takes already-loaded policy documents and answers a decision. This is the same logic that
gets compiled into the client SDK (§6), so it must be standalone.

```go
// Decision is the PDP verdict.
type Decision struct {
    Allowed bool
    Reason  string // matched statement / "explicit deny" / "no matching allow"
}

type AuthzRequest struct {
    Principal string   // "serviceA"  (informational; bundle is already principal-scoped)
    Action    string   // "serviceB:invoke"
    Resource  string   // "serviceB:grpc"  or "serviceB:*"
}

// Evaluate applies AWS-style semantics over all statements in the bundle.
func Evaluate(docs []PolicyDocument, req AuthzRequest) Decision
```

**Semantics (must match AWS mental model, and be documented + tested):**

1. **Default deny.** No matching `allow` → denied.
2. **Explicit deny wins.** Any matching `deny` overrides all `allow`s.
3. **Action match.** Exact (`user:create`) or wildcard. Support at least `service:*` and
   `*`; decide whether to support mid-segment globs (`user:*` already implied by doc).
4. **Resource match.** `service:api` exact, `service:*` (all APIs under a service), `*`.
5. **Version gate.** Only evaluate `version: "v1"` documents; unknown versions → ignored
   (logged), never silently allowed.

Validation of the *structure* already exists in
[`internal/iam/validation_policy.go`](../../internal/iam/validation_policy.go) — reuse it.

**Optional convenience endpoint** (for services that can't embed the SDK, e.g. another
language): `POST /api/v1/authorize { principal, action, resource } → { decision }`. Same
`Evaluate()` under the hood. Lower priority than the bundle path since the whole point is to
*avoid* per-request calls.

---

## 5. Component 3 — Bundle distribution (Pattern 1)

New endpoint on the management port:

```
GET /api/v1/services/me/policy-bundle
Authorization: Bearer <service-account token>     # sub = serviceA
If-None-Match: "v7"                                # optional, from prior fetch

200 OK
ETag: "v8"
Cache-Control: max-age=30
{
  "service": "serviceA",
  "version": "v8",
  "policies": [ { "version": "v1", "statement": [ ... ] }, ... ],   # resolved, attached docs
  "generated_at": "2026-06-04T..."
}

304 Not Modified          # when If-None-Match matches current ETag  (cheap path)
```

- **`me` resolves from the token**, never a path param — a service can only fetch *its own*
  bundle. (Reject if the token isn't a service token, or doesn't map to a `Service`.)
- **ETag derivation:** hash of the service's attached policy set, or `MAX(policies.updated_at,
  service_policies.created_at)` for that service. Cheap to compute, changes iff content
  changes.
- **Resolution:** join `service_policies → policies` for the principal, return the active
  `document`s. Skip `inactive` policies/services.
- Bundles are **per-service**, so A and B each fetch their own; B also fetches A's bundle
  on first inbound call from A (cached thereafter) — or, simpler, B uses the `/authorize`
  endpoint for callers it hasn't cached. Pick one; recommend: B caches caller bundles with
  the same ETag mechanism.

**Client behavior (SDK, §6):** fetch at startup → evaluate locally → re-poll every
`max-age` (e.g. 30s) with `If-None-Match` → swap cache only on `200`. On auth-server
unavailability, keep serving from the last good bundle (fail-static).

---

## 6. Component 4 — Push invalidation (Pattern 3)

Reuse the existing webhook machinery
([`internal/webhook/dispatcher.go`](../../internal/webhook/dispatcher.go) — HMAC-SHA256
signed, replay-protected, event-type subscription, per-tenant). Add new event types:

| Event type | Fired when |
|------------|------------|
| `iam.policy.updated` | a `Policy.Document` is updated (`service_policy.Update`) |
| `iam.service.policy.assigned` | `POST /services/{uuid}/policies/{uuid}` |
| `iam.service.policy.removed` | `DELETE /services/{uuid}/policies/{uuid}` |

On receipt, a subscribed service **immediately re-pulls** its bundle (with `If-None-Match`,
so if it already has the change it's a cheap `304`). This means revocation propagates in
~webhook-delivery time instead of waiting up to one poll interval.

**Belt-and-suspenders for revocation:**
- Keep **service-account tokens short-lived** (5–15 min) so even a missed webhook + missed
  poll bounds staleness to token TTL.
- The existing [`internal/iam/token_invalidator.go`](../../internal/iam/token_invalidator.go)
  + access-token denylist (Redis, ✅) remain the hard "revoke now" escape hatch.

---

## 7. Component 5 — Client SDK (`maintainerd-authz` Go module)

A small library each service imports so they don't reimplement caching/eval:

```go
authz, _ := authzclient.New(authzclient.Config{
    AuthServerURL: "https://auth.lula...",
    ClientID:      "serviceA", ClientSecret: "...",   // client_credentials
    PollInterval:  30 * time.Second,
    WebhookListen: ":9099",                            // optional, for push refresh
})
defer authz.Close()

// outbound (service A)
if !authz.Can("serviceB:invoke", "serviceB:grpc") { return ErrForbidden }

// inbound (service B) — caller identity from gRPC metadata token
if !authz.CanPrincipal(callerSvc, "serviceB:invoke", "serviceB:grpc") { ... }
```

The SDK = token acquisition + bundle cache + `Evaluate()` (the **same** code from §4) +
poll loop + optional webhook receiver. Ships in this repo or a sibling module; the
evaluator is shared so server `/authorize` and SDK never diverge.

---

## 8. Build checklist (v1.0.0)

Stable IDs `S2S-*` for tracking in [bugs-and-enhancements.md](./bugs-and-enhancements.md).

- [ ] **S2S-01** — Service principal: link `Service` ↔ OAuth `client_credentials` client; emit service identity in token (`sub`/`svc` claim). _(oauth, iam)_
- [ ] **S2S-02** — PDP engine `internal/iam/policy_evaluator.go`: default-deny, explicit-deny-wins, action/resource wildcard matching, v1-only. Exhaustive table tests. _(iam)_
- [ ] **S2S-03** — Bundle endpoint `GET /api/v1/services/me/policy-bundle` with `ETag`/`If-None-Match`/`304`, principal-from-token, active-only resolution. _(iam)_
- [ ] **S2S-04** — Webhook events `iam.policy.updated`, `iam.service.policy.assigned/removed` wired into existing dispatcher. _(webhook, iam)_
- [ ] **S2S-05** — Client SDK module: token acquisition + bundle cache + shared `Evaluate()` + poll loop + optional webhook receiver. _(new module)_
- [ ] **S2S-06** — Short service-token TTL default + denylist integration verified for instant revocation. _(oauth)_
- [ ] **S2S-07** — (Optional) `POST /api/v1/authorize` convenience endpoint reusing `Evaluate()` for non-Go callers. _(iam)_
- [ ] **S2S-08** — Docs: `docs/apis/iam/authorization.md` (bundle + authorize), integration guide for service A/B. _(docs)_

**Test obligations** (per [`docs/contributing/testing.md`](../contributing/testing.md)):
service-layer success/error branches for the bundle resolver; validation tests for the
evaluator semantics (one sub-test per rule: default-deny, explicit-deny, action wildcard,
resource wildcard, version gate); handler 9-step checklist for the bundle + authorize
endpoints. Keep `iam` package coverage ≥ 80%.

---

## 9. Deferred to v2.0.0

- **Resource policies** — a `principal` field on `PolicyStatement` so B independently
  controls *who* may call it, evaluated against B's own policies. v1 verifies A's identity
  policy at both ends instead.
- **Conditions / context** — `condition` block (IP, time, mTLS-bound, request attrs), like
  AWS condition keys.
- **mTLS service identity** as an alternative to client-credentials tokens (pairs with the
  pending `tls_client_auth` row in v1.0.0 §2).
- **Decision caching** (cache the verdict, not just the bundle) if eval cost ever matters.

---

## 10. References

- Current policy model: [`internal/iam/model_policy.go`](../../internal/iam/model_policy.go), [`internal/iam/types.go`](../../internal/iam/types.go)
- Service/policy assignment: [`internal/iam/routes.go`](../../internal/iam/routes.go), [`internal/iam/service_service.go`](../../internal/iam/service_service.go)
- Webhook dispatch: [`internal/webhook/dispatcher.go`](../../internal/webhook/dispatcher.go)
- OAuth client-credentials: [`internal/oauth/`](../../internal/oauth/)
- RBAC enforcement (existing PEP): [`internal/platform/middleware/permission_middleware.go`](../../internal/platform/middleware/permission_middleware.go)
- Release scope: [`docs/releases/v1.0.0.md`](../releases/v1.0.0.md) §5
