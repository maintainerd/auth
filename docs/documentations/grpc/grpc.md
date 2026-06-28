# gRPC API — Service-to-Service Control Plane

**Status:** Available — v1.0.0
**Audience:** Engineers integrating another service with the Lula auth server over gRPC.
**Related:** [Service-to-Service Authorization](../service-to-service-authorization/service-to-service-authorization.md) · [Architecture](../architecture/architecture.md) · [Environment Variables](../environment-variables/environment-variables.md)

This guide explains how to call the Lula auth server over **gRPC**. It is written
for developers building a *service* (the control plane, a peer microservice, a job
runner) that needs to **manage** auth resources or **verify** auth facts
programmatically.

> **gRPC here is a service-to-service (S2S) transport only.** It exposes the
> *management* and *verification* surface of the private/internal API. Browser and
> end-user flows (login, registration, password reset, MFA enrollment, consent UI)
> stay on REST and are intentionally **not** available over gRPC. If you are building
> a user-facing app, use the REST API instead.

---

## 1. At a glance

| | |
|---|---|
| **Proto package** | `maintainerd.auth.v1` |
| **Proto sources** | `proto/maintainerd/auth/v1/*.proto` |
| **Generated Go stubs** | `internal/platform/gen/go/maintainerd/auth/` |
| **Codegen** | [buf](https://buf.build) — `make proto` (`buf generate`) |
| **Network exposure** | **Private network only** (same posture as the internal REST port). Never exposed to the public internet. |
| **Transport security** | **TLS always**; **mTLS** where the mesh issues client certs |
| **Authentication** | OAuth 2.0 `client_credentials` service-account access token in `authorization` metadata |
| **Authorization** | Per-RPC permission, evaluated by the PDP against the caller's attached policy — **default-deny** |
| **Health** | `grpc.health.v1.Health` (dependency-aware: DB, Redis, JWKS) |
| **Reflection** | Enabled in non-production only |

Everything gRPC exposes maps 1:1 to an existing internal REST endpoint and reuses
the **same permission strings** (e.g. `tenant:read`, `client:create`). There is one
authorization vocabulary across both transports.

---

## 2. Getting the contract

Consumers generate their own client stubs from the `.proto` files. The package is
versioned (`maintainerd.auth.v1`); within `v1` the contract only ever grows
additively — breaking changes would ship as a new `v2` package.

```bash
# Example: vendor the protos and generate a client with buf
buf generate            # or point protoc at proto/maintainerd/auth/v1
```

All messages follow [Google AIP](https://google.aip.dev) conventions: standard
methods (`List`, `Get`, `Create`, `Update`, `Delete`), custom methods for non-CRUD
verbs (`SetServiceStatus`, `RotateClientSecret`, `Authorize`), `google.protobuf.Timestamp`
for times, and `string` UUIDs that match the REST path parameters.

---

## 3. Authenticating

A calling service authenticates **as itself** (a *service principal*), not as a user.

### Step 1 — obtain a service-account token

Your service is registered as a `Service` linked to an OAuth client. Exchange its
client credentials for an access token via the standard token endpoint:

```bash
curl -X POST https://auth.internal:8080/oauth/token \
  -d grant_type=client_credentials \
  -d client_id=$CLIENT_ID \
  -d client_secret=$CLIENT_SECRET
```

The returned JWT carries the service identity (`sub`/`svc` claim). Keep these tokens
**short-lived** (5–15 min) and refresh them; revocation propagates quickly via the
denylist and policy-change webhooks.

### Step 2 — send the token as metadata

Send the same bearer token REST uses, as the `authorization` gRPC metadata key:

```
authorization: Bearer <service-account access token>
```

The **auth interceptor** verifies the token (signature, expiry, denylist) and puts
the principal in context. A missing/invalid token → `codes.Unauthenticated`.

### Step 3 — hold the right policy (authorization)

Every protected RPC has a required permission. The **authz interceptor** asks the PDP
whether the caller's attached policy `allow`s that action. **No matching `allow` →
`codes.PermissionDenied`.** A service with *no* policy attached is denied everything —
attaching a policy is how you grant access. See
[Service-to-Service Authorization](../service-to-service-authorization/service-to-service-authorization.md).

> **Net:** a call succeeds only if (a) it presents a valid service-account token, and
> (b) the service holds a policy whose statements allow that RPC's action.

---

## 4. Request correlation (`x-request-id`)

Send `x-request-id` in metadata to stitch a request across service logs. The server
accepts it (and common aliases `request-id`, `x-correlation-id`, `correlation-id`),
synthesizes a UUID if absent, logs it, and returns it in **trailing** metadata. Always
propagate the inbound request ID from your edge, and read the returned trailing value
as the audit correlation key.

---

## 5. Error model

Errors use canonical gRPC status codes plus rich details
([AIP-193](https://google.aip.dev/193)): `google.rpc.ErrorInfo` and `BadRequest` are
attached where useful. Raw internal errors are never leaked.

| Situation | gRPC code |
|---|---|
| Missing/invalid/expired/denylisted token | `Unauthenticated` |
| Valid token, no policy allows the action | `PermissionDenied` |
| Validation failure (bad field) | `InvalidArgument` (+ `BadRequest` details) |
| Resource not found / not in caller's tenant | `NotFound` |
| Uniqueness / state conflict | `AlreadyExists` / `FailedPrecondition` |
| Per-identity rate limit exceeded | `ResourceExhausted` |
| Deadline hit | `DeadlineExceeded` |
| Unhandled server error | `Internal` |

Treat `Unauthenticated` and `PermissionDenied` as **terminal** security outcomes — do
not retry them.

---

## 6. Pagination

`List*` RPCs use the REST-shaped `Pagination` / `PageMetadata` messages
(`page`, `limit`, `sort_by`, `sort_order`) and return `PageMetadata` with totals. Token
pagination (AIP-158) is not used in `v1`.

---

## 7. Service catalog

All services live in package `maintainerd.auth.v1`. Permissions match the REST
internal API. `+step-up` marks RPCs that additionally require a recent
step-up/re-auth claim.

### Verification / decision (the high-value S2S reads)

| Service · RPC | Purpose | Permission |
|---|---|---|
| `AuthorizationService.Authorize` | Ask the PDP "can principal X do action Y on resource Z?" | service-account |
| `OAuthIntrospectionService.Introspect` | Validate/inspect a token (RFC 7662) | service-account |
| `ServiceService.GetMyPolicyBundle` | Pull this service's policy bundle (ETag/304) for local PDP caching | service-account (self) |

### Management / configuration

| Service | Resource | Key permissions |
|---|---|---|
| `TenantService`, `TenantSettingService` | Tenants, members, settings | `tenant:*`, `tenant-setting:*` |
| `ServiceService` | Service principals + policy assignment | `service:*`, `service:policy:assign|remove` |
| `APIService`, `PermissionService`, `PolicyService`, `RoleService` | IAM model | `api:*`, `permission:*`, `policy:*`, `role:*` |
| `ClientService`, `APIKeyService` | OAuth clients, URIs, API keys, grants | `client:*`, `api_key:*` (+step-up on secrets/keys) |
| `UserService` | Admin user CRUD, status, verify, roles | `user:*` (+step-up on status/delete/roles) |
| `UserProfileService` | Admin management of a user's profiles | `user:read|update|delete` |
| `InviteService.SendInvite` | Trigger a user invitation | `user:invite` |
| `IdentityProviderService`, `SignupFlowService` | Federation + signup config | `idp:*`, `signup-flow:*` |
| `SecuritySettingService`, `IPRestrictionRuleService` | Security policy + IP rules | `security-setting:*`, `ip-restriction-rule:*` |
| `BrandingService`, `EmailTemplateService`, `SMSTemplateService`, `LoginTemplateService` | Branding + templates | `branding:*`, `*-template:*` |
| `EmailConfigService`, `SMSConfigService` | Notifier delivery config | `email-config:*`, `sms-config:*` |
| `WebhookEndpointService` | Webhook endpoints | `webhook-endpoint:*` |
| `AuthEventService` | Read the audit-event log | `auth_event:read` |
| `SetupService` | One-time bootstrap / controller registration | bootstrap-gated (see below) |

> **SetupService** is special: it runs *before* any policy exists, so it is guarded by
> a one-time setup lock instead of the PDP. Once `CompleteSetup` is called, all mutating
> setup RPCs are closed and only `GetSetupStatus` remains. Controller registration after
> that uses normal PDP-gated `AssignServicePolicy`.

The former per-RPC planning backlog was retired when the planning folder was consolidated.
Use the service tables in this document for the gRPC reference and track current pre-release
work in
[docs/planning/develop-before-v0.1.0.md](../../planning/develop-before-v0.1.0.md).

---

## 8. Health & service discovery

- **Health:** `grpc.health.v1.Health/Check` and `Watch`. The overall status reflects
  **readiness** — DB reachable, optional Redis reachable, JWKS loaded — not just "process
  up." Use it for load-balancer health and startup gating.
- **Reflection:** enabled in non-production so you can explore with `grpcurl`:

```bash
grpcurl -H "authorization: Bearer $TOKEN" auth.internal:9090 list
grpcurl -H "authorization: Bearer $TOKEN" auth.internal:9090 \
  maintainerd.auth.v1.AuthorizationService/Authorize
```

Reflection is **off in production** by policy.

---

## 9. Consumer reliability rules (required)

gRPC reliability is half server, half client. Every service calling auth **must**:

- **Set a per-call deadline.** Default 5s for reads/verification, 10s for management
  writes, unless a workflow has a stronger SLA.
- **Propagate `x-request-id`** from the inbound request (synthesize at the edge if
  missing); log the returned trailing value.
- **Retry only safe calls.** Reads (`Get*`, `List*`, `Authorize`, `Introspect`,
  `GetMyPolicyBundle`) may retry on `Unavailable`, `DeadlineExceeded`, and transient
  `Internal`. **Mutating RPCs must not auto-retry** unless you supply your own
  idempotency strategy.
- **Exponential backoff with jitter.** Never tight-loop on `ResourceExhausted` — back
  off and surface throttling to a circuit breaker.
- **Reuse one `grpc.ClientConn` per target** (connection pool), not one per call.
- **Circuit-break by method + target.** When open, fail closed for authorization
  decisions and fail explicitly for management writes.
- **Treat `Unauthenticated` / `PermissionDenied` as terminal**, never as transport
  errors to retry.

---

## 10. Example — local authorization with a cached bundle

The recommended pattern for hot-path authorization is **pull + cache + invalidate**:

1. At startup, call `ServiceService.GetMyPolicyBundle` and cache the bundle locally
   (store the `ETag`).
2. Decide locally — no per-request round-trip to auth.
3. Re-poll with the `ETag`; an unchanged bundle returns `304`/`not_modified`.
4. Subscribe to the `iam.policy.updated` / `iam.service.policy.assigned|removed`
   webhooks (see [Webhooks](../events/webhooks.md)) to refresh **immediately** on change.

For one-off checks (or services that can't cache), call
`AuthorizationService.Authorize` directly.

```go
conn, _ := grpc.NewClient("auth.internal:9090", grpc.WithTransportCredentials(creds))
client := authv1.NewAuthorizationServiceClient(conn)

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
ctx = metadata.AppendToOutgoingContext(ctx,
    "authorization", "Bearer "+token,
    "x-request-id", inboundRequestID,
)

resp, err := client.Authorize(ctx, &authv1.AuthorizeRequest{
    Action:   "orders:read",
    Resource: "order/123",
})
// resp.Allowed → enforce
```

---

## 11. Not available over gRPC

These stay on REST because they are user-driven, not service-to-service:

- Registration, login, logout, password reset, magic-link / SMS login.
- MFA enrollment / authentication, **and admin MFA reset** (user-account remediation).
- Self-service profile/account/settings.
- Federation callbacks / redirects and user identity linking ceremonies.

If a management or verification need appears for one of these, it is added to the gRPC
backlog with a concrete S2S consumer first.
