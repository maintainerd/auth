# gRPC Feature List — Service-to-Service Control Plane Transport

**Status:** Planning — proposed for **v1.1.0** (post REST/S2S-authz baseline).
**Owner:** rseguma@lula.life
**Created:** 2026-06-04
**Related:** [service-to-service-authorization.md](../documentations/service-to-service-authorization/service-to-service-authorization.md) · [architecture.md](../documentations/architecture/architecture.md) · [code-structure.md](../contributing/code-structure.md) · [testing.md](../contributing/testing.md)

This document is the implementation backlog for exposing Lula's auth server over
**gRPC** as an additional **service-to-service (S2S) control-plane transport**.
S2S is **not gRPC-only** in this project: the internal/private REST port already
supports S2S use cases and keeps doing so. The goal of this backlog is to add a
typed, high-performance gRPC channel that lets other services — the **core /
control plane** especially — **control** this auth service (create/update/delete
resources) and **read data** from it, mirroring the existing **internal (private)
REST port** (`:8080`).

It backlogs every internal-port **application API** endpoint as a gRPC RPC,
assigns each a status, and pins down the **contract layout, naming, and S2S
authentication standard** so the work follows gRPC best practice from the start.
Operational endpoints (`/health`, `/ready`, `/livez`, `/openapi.json`) map to
gRPC health/reflection/proto discovery instead of one-off product RPCs.

---

## 0. How to use this doc

Work top-to-bottom: **Phase 0 (foundation)** must land before any service RPCs,
because it establishes the proto layout, codegen, auth interceptors, and error
mapping every RPC depends on. Each item has a stable ID (`GRPC-NNN`), a status,
and a location.

**Status legend**

| Badge | Status | Meaning |
|-------|--------|---------|
| 🔴 | **todo** | Not started. |
| 🟡 | **in-development** | Actively being implemented. |
| 🔵 | **testing** | Code complete; under test / review per [testing.md](../contributing/testing.md). |
| ✅ | **done** | Merged, tested, and verified. |

> Everything is 🔴 **todo** unless noted. The ✅ **done** items today are the
> bootstrap `SeederService` (see §3) and the non-deletable system service guard
> (GRPC-017, see §7.7) — both already in the codebase.

**ID convention:** `GRPC-0xx` = Phase 0 foundation **plus cross-cutting
provisioning/setup items** (GRPC-015…023, gathered in §7.7). `GRPC-1xx` =
control-plane (management) services. `GRPC-2xx` = identity / end-user-flow services
(deferred — see §10). Per-RPC status lives in each service's table.

---

## 1. Goal & scope

|                     |                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Independence**    | maintainerd-auth is **fully standalone** — it boots, migrates, seeds, and serves with **no dependency on the core / control plane**. The core is an *optional* manager, never a runtime requirement. The gRPC surface assumes nothing about the core's existence. See §7.                                                                                                                                                               |
| **Why gRPC**        | Typed contracts, codegen for every consumer language, streaming-capable, low overhead — a strong fit for **service-to-service** traffic (core/control plane ↔ auth, and peer services reading auth data). It is an additional S2S transport, not the only S2S path.                                                                                                                                               |
| **What it mirrors** | The **internal (private) REST port** (`:8080`, VPN/private-network only — see [router.go](../../internal/server/router.go) `buildInternalRouter`). gRPC becomes a private-network-only control surface beside REST, **never** exposed to the public internet (the public port `:8081` stays REST-only). Product APIs are mirrored as RPCs; operational probes/spec endpoints are covered by gRPC health, reflection, and generated proto contracts. |
| **Two use-cases**   | **(a) Control** — external services (e.g. the core/control plane) mutate auth resources (tenants, clients, policies, users, settings…). **(b) Data** — peer services read auth data (introspect tokens, fetch policies, list roles…).                                                                                                                                                                                                   |
| **Auth model**      | Transport-agnostic S2S. REST and gRPC both use **service-account access tokens** and PDP policy checks. For gRPC, every RPC carries the token in metadata and is policy-gated. **A policy must always exist** for a caller to reach a protected REST endpoint or gRPC RPC — default-deny. See §6.                                                                                                                                         |
| **Out of scope**    | Browser/public/end-user-interactive traffic stays on REST. gRPC-Web/public gateway, and breaking v2 changes — see §11.                                                                                                                                                                                                                                                                                                                  |

---

## 2. Design principles (gRPC best practice adopted)

These are the standards every RPC must follow. They are non-negotiable defaults so
the contract stays idiomatic and forward-compatible.

1. **Resource-oriented design** (Google AIP-121/122/131-135). Standard methods map
   cleanly from REST: `List`, `Get`, `Create`, `Update`, `Delete`. Non-CRUD verbs
   become **custom methods** (AIP-136): `SetServiceStatus`, `AssignServicePolicy`,
   `RotateClientSecret`, `Authorize`.
2. **One request/response message per RPC**, named `<Verb><Noun>Request` /
   `<Verb><Noun>Response` (AIP-131+). Never reuse a message across RPCs and never
   take a bare scalar — wrap it so fields can be added without breaking the wire.
3. **Field conventions:** `google.protobuf.Timestamp` for times,
   `google.protobuf.FieldMask` (`update_mask`) for partial updates (AIP-134),
   `string` UUIDs to match REST path params, enums for status with a
   `*_UNSPECIFIED = 0` zero value (AIP-126).
4. **Pagination** (AIP-158): `page_size` + `page_token` → `next_page_token` +
   `total_size`. (The REST layer uses `page`/`limit`; the gRPC layer adopts the
   token form and the service layer adapts — see [pagination](../../internal/platform/pagination).)
5. **Canonical status codes + rich errors** (AIP-193): map `apperror` →
   `google.rpc.Code` and attach `google.rpc.ErrorInfo`/`BadRequest` details. No
   leaking raw Go errors. See GRPC-004.
6. **Versioned package, additive evolution.** Package `maintainerd.auth.v1`; only
   add fields/RPCs within v1. Breaking changes ⇒ a new `v2` package, enforced by
   `buf breaking`. See §4.
7. **Health + reflection are first-class.** Ship `grpc.health.v1.Health` and
   server reflection so the control plane and `grpcurl` can probe the server.
8. **Interceptors over per-handler glue.** Auth, authz, logging, recovery,
   rate-limit, and telemetry are interceptor-chain concerns, mirroring the REST
   `mountCommonMiddleware` stack. See §5.
9. **Transport security.** TLS always; **mTLS** for service identity where the
   mesh provides it (defense-in-depth alongside the bearer token). See §6.

---

## 3. Current state

| Thing | Where | Status |
|-------|-------|--------|
| gRPC server lifecycle (listen, graceful stop, otelgrpc stats handler) | [grpc.go](../../internal/server/grpc.go) `StartGRPCServer` | ✅ exists |
| Bound address constant | `shared.DefaultGRPCAddr` | ✅ exists |
| `SeederService` (1 RPC: `TriggerSeeder`) | [v1.proto](../../proto/maintainerd/auth/v1.proto), handler in [setup](../../internal/setup) | ✅ done |
| Proto source | `proto/maintainerd/auth/v1.proto` (flat — `v1` is the **filename**) | ⚠️ needs restructure (§4) |
| Generated Go | `internal/platform/gen/go/maintainerd/auth/` | ⚠️ path mismatch (§4) |
| Codegen | `make proto` (raw `protoc`) | ⚠️ migrate to `buf` (GRPC-001) |
| S2S authz primitives (OAuth `client_credentials`, PDP evaluator, `/authorize/`, policy bundle, default-deny) | [iam](../../internal/iam), [oauth](../../internal/oauth) | ✅ done — **reuse for gRPC** (§6) |

**The auth model is already built.** The S2S authorization design
([service-to-service-authorization.md](../documentations/service-to-service-authorization/service-to-service-authorization.md),
items S2S-01…S2S-08) is implemented for REST: service principals via OAuth
`client_credentials`, the PDP `Evaluate()` engine, and default-deny policy
resolution. **gRPC does not invent a new auth scheme — it reuses this one** through
interceptors (§6). That is the answer to "what is the standard for S2S auth here."

---

## 4. Contract structure — answering the `maintainerd/maintainerd-auth` concern

**Your concern:** the repo is `maintainerd/maintainerd-auth`; is `proto/maintainerd/auth`
the right namespace, or should it be `maintainerd/maintainerd-auth`?

**Verdict: `maintainerd.auth.v1` is correct — keep it. Do _not_ encode the repo
name (`maintainerd-auth`) into the proto package.**

Why this is the best practice:

- The buf / Google convention is **`<org>.<service-domain>.<version>`**, not
  `<org>.<repo-name>.<version>`. The repo is named `maintainerd-auth` only because
  a flat GitHub org needs the `maintainerd-` prefix to disambiguate sibling repos
  (`maintainerd-core`, `maintainerd-auth`, …). The **domain** inside the
  `maintainerd` org is simply **`auth`**.
- So the namespace is `maintainerd` (org) → `auth` (this service's domain) →
  `v1` (version). `maintainerd.auth.v1` reads naturally and stays consistent with
  sibling services that would be `maintainerd.core.v1`, `maintainerd.billing.v1`, etc.
- `maintainerd.maintainerd_auth.v1` would stutter and leak a packaging detail
  (the repo name) into a wire contract that should be stable forever.

**What to fix (it's the _layout_, not the name):**

1. Make `v1` a **directory**, not a filename (buf style requires
   `…/<version>/…proto`), and **split by service/domain** instead of one giant file:

   ```
   proto/
     buf.yaml                      # module + lint + breaking config
     buf.gen.yaml                  # codegen plugins (protoc-gen-go, -go-grpc)
     maintainerd/
       auth/
         v1/
           common.proto            # shared: pagination, status enums, error msgs
           seeder.proto            # existing SeederService (move here)
           tenant.proto            # TenantService, TenantSettingService
           iam.proto               # Service/API/Permission/Policy/Role/Authorization
           identity_provider.proto # IdentityProviderService, SignupFlowService
           client.proto            # ClientService, APIKeyService
           user.proto              # UserService (admin)
           security.proto          # SecuritySettingService, IPRestrictionRuleService
           branding.proto          # Branding + Email/SMS/Login template services
           notifier.proto          # EmailConfigService, SMSConfigService
           webhook.proto           # WebhookEndpointService
           authevent.proto         # AuthEventService
           oauth.proto             # OAuthIntrospectionService
           setup.proto             # SetupService (provisioning)
   ```

   All files keep `package maintainerd.auth.v1;` — splitting files does **not**
   split the package (same rule as Go).

2. **Fix the `go_package` / output-path mismatch.** Today the proto declares
   `option go_package = "…/internal/gen/go/maintainerd/auth;authv1"` and the
   `Makefile` sets `PROTO_OUT := internal/gen/go`, but the code actually imports
   `…/internal/platform/gen/go/maintainerd/auth`. Pick one — recommend
   **`internal/platform/gen/go`** (it is platform infrastructure per
   [code-structure.md](../contributing/code-structure.md) §`internal/platform/*`) —
   and align `go_package`, the Makefile/buf output, and the import in
   [grpc.go](../../internal/server/grpc.go).

3. **Adopt `buf`** for lint + breaking-change detection + codegen (replaces raw
   `protoc`). This is the modern standard and gives you `buf lint` and
   `buf breaking` in CI.

These three are Phase-0 items GRPC-001/002/003 below.

---

## 5. Interceptor chain (mirrors REST `mountCommonMiddleware`)

The gRPC server gets a unary + stream interceptor chain, ordered like the REST
middleware stack in [router.go](../../internal/server/router.go):

| Order | Interceptor | REST analogue | Phase-0 item |
|-------|-------------|---------------|--------------|
| 1 | **Recovery** (panic → `codes.Internal`, structured log) | `RecoveryMiddleware` | GRPC-005 |
| 2 | **Telemetry** (otelgrpc — already wired) | otelhttp | ✅ exists |
| 3 | **Logging** (request_id, method, peer, latency, code) | `LoggingMiddleware` | GRPC-005 |
| 4 | **Auth** (extract + verify service-account token from metadata) | `JWTAuthMiddleware` | GRPC-006 |
| 5 | **Authz** (PDP / permission check per-RPC) | `RequirePermission` | GRPC-007 |
| 6 | **Rate-limit** (per-caller-identity) | `IPRateLimitMiddleware` | GRPC-008 |
| 7 | **Request-size / timeout / panic budget** | `RequestSizeLimit` / `Timeout` | GRPC-008 |

Per-RPC permission/scope requirements are declared in a **registry** (a map of
`/maintainerd.auth.v1.XxxService/Method` → required permission string), reusing the
**exact same permission identifiers** the REST routes use (e.g. `tenant:read`,
`service:create`). This keeps one authorization vocabulary across both transports.

---

## 6. S2S authentication & authorization standard (transport-agnostic)

This is the standard for service-to-service calls **regardless of transport**.
REST already uses this model, and gRPC must reuse it. gRPC does **not** invent a
new auth scheme; it adds interceptors/adapters around the same service-account
tokens, policy bundles, and PDP.

1. **Caller identity = service principal.** The calling service authenticates as
   itself via OAuth **`client_credentials`** (already in v1.0.0,
   [oauth](../../internal/oauth)). Its `Service` row is linked to an OAuth client;
   the issued access token carries the service identifier (`sub`/`svc` claim).
2. **Token on the wire.** REST callers send `Authorization: Bearer <token>`;
   gRPC callers send the same bearer token as metadata:
   `authorization: Bearer <service-account token>`. The **Auth interceptor**
   (GRPC-006) extracts and verifies it for gRPC (signature, expiry, denylist —
   reusing [jwt](../../internal/platform/jwt) + the Redis denylist /
   [token_invalidator](../../internal/iam)), then puts the principal in context
   (same shape REST uses via `authctx`).
3. **Per-call authorization (default-deny).** REST middleware and the gRPC **Authz
   interceptor** (GRPC-007) look up the endpoint/RPC's required permission/action
   and ask the **PDP** (`iam.Evaluate()` — already built) against the caller's
   resolved policy bundle. **No matching `allow` → denied** (`403` for REST,
   `codes.PermissionDenied` for gRPC). Explicit deny wins. **A policy must always
   be attached to the calling service** or every protected REST endpoint / gRPC RPC
   is denied — exactly the property you asked for.
4. **Token validation for peers.** REST already exposes `POST /oauth/introspect`;
   the `OAuthIntrospectionService.Introspect` RPC (GRPC-180) adds the gRPC equivalent
   so peers can validate tokens over either private transport.
5. **Transport security.** Bind the gRPC port to the **private network only** (like
   the internal REST port). Require **TLS**; prefer **mTLS** where the service mesh
   issues client certs — the cert proves transport-level service identity and the
   bearer token proves application-level identity (defense-in-depth, zero-trust).
6. **Short-lived tokens + push invalidation.** Keep service-account tokens
   short-lived (5–15 min) and rely on the existing webhook policy-invalidation
   (`iam.policy.updated`, `iam.service.policy.assigned/removed`) so revocation
   propagates fast across REST and gRPC callers.

> **Net:** a service can only reach a protected REST endpoint or gRPC RPC if (a)
> it presents a valid service-account token, and (b) it holds an attached policy
> whose statements `allow` that action. Same author-centrally /
> decide-from-policy model on both private transports.

---

## 7. Independence & provisioning model (core-optional)

**Principle: maintainerd-auth is fully standalone.** It boots, migrates, seeds, and
serves with **no dependency on the core**. The core is an *optional* manager, not a
runtime requirement. Nothing in the gRPC (or REST) control surface assumes the core
exists — an install with zero controllers attached is a valid, fully-functional
deployment.

### 7.1 What setup already seeds about *this* app (no new work — don't regress)

- Seeder [001_service.go](../../internal/setup/seeder/001_service.go) creates a
  **default system service** `name="auth"`, `IsSystem=true`. This row **is** this
  app as an IAM principal.
- System rows are **immutable + non-deletable**:
  [service_service.go](../../internal/iam/service_service.go) rejects update
  (`system service cannot be updated`), status change
  (`system service status cannot be updated`), and delete
  (`system service cannot be deleted`). So *"the service record that represents this
  app is not deletable"* is **already enforced** today.
- ⇒ The app's own identity needs no core and cannot be removed by an API caller.

### 7.2 The prepared control policy (NEW — GRPC-015)

A **default "control" policy template** is seeded as a **system policy**
(`Policy.IsSystem=true`, [model_policy.go](../../internal/iam/model_policy.go))
whose statements `allow` the management actions a manager needs over this app. Those
actions **already exist as permissions** (every API is already permissioned — §9
reuses `tenant:*`, `service:*`, `client:*`, `user:*`, …), so this is just a curated
statement set, not new permissions.

Key properties:

- **Seeded but attached to no one.** On a fresh standalone install the control
  policy *exists* but is **unattached** → by default-deny, **nobody controls this
  app**. Independence preserved.
- **Non-deletable template** (it is `IsSystem`), but **control is granted/revoked by
  attaching/detaching it** — the `service_policies` join row is the deletable,
  detachable part. This is exactly *"allowing the core to control this is attachable
  and detachable; other (attachment) records are deletable, the system records are
  not."*

### 7.3 Registering a controller — core-driven **or** manual

Two paths, identical end-state: a **service principal** (§6) that holds the control
policy.

1. **Via setup (core-driven, gRPC or REST) — NEW GRPC-191.** During setup (alongside
   `CreateTenant` / `CreateAdmin`) the caller passes the controller's **service name
   / identifier** (e.g. `core`). Auth then, in one transaction:
   - creates a `Service` row for it,
   - provisions an OAuth **`client_credentials`** client for it (its S2S identity, §6),
   - **attaches the default control policy** to it.

   This is the **trust-on-first-use (TOFU)** moment — setup is the *one* place a
   control grant can be minted without already holding one.

   > ⚠️ **Setup is a one-time initialization window.** Today
   > [service_setup.go](../../internal/setup/service_setup.go) derives completion from
   > `tenant + admin + profile`, but gRPC setup must change that to a persisted,
   > explicit lock. Standalone setup runs
   > `create_tenant → create_admin → create_profile`, then *optionally*
   > `RegisterControlService`, then **`CompleteSetup`** (`POST /setup/complete`,
   > GRPC-023). `CompleteSetup` is not another provisioning step; it only closes the
   > bootstrap window for cases where the instance was **not** provisioned by a core,
   > or where the operator is done with setup. After the lock, the setup path is gone
   > and controller changes use path 2 (runtime). **Anti-infiltration:** the explicit
   > lock stops any other service from registering itself as the controller once a
   > standalone operator has finished.

2. **Manual / runtime (no core, or after setup is closed).** An operator (or an
   already-authorized controller) does the same at runtime over REST/gRPC: create a
   `Service` for the controller, then **attach the control policy**
   (`AssignServicePolicy`, GRPC-110) — authenticated normally and PDP-gated, **not**
   through the closed setup endpoint. Or attach nothing → the app runs fully
   independent forever. **This is the only way to add/change a controller after
   initialization.** Defining a controller is *opt-in by attachment*.

### 7.4 Provision / unprovision semantics

- The core can **provision and unprovision** this app, and provision **multiple
  instances** — but **only because it holds the attached control policy**. The
  runtime PDP (§6) enforces this on every call: no control policy → `PermissionDenied`.
  *"It can only provision/unprovision maintainerd-auth if it has policy/access to it."*
- **Un-registering the core = detaching (or deleting) its control-policy
  attachment.** Control is revoked promptly (webhook push + short token TTL, §6).
  The system `auth` service and its own self-policy are untouched — the app keeps
  running standalone.
- **Multiple instances:** each instance independently seeds its own system `auth`
  service + control-policy template, and the core registers with **each instance
  separately** (TOFU per instance). One removed/compromised instance never implicates
  another — independence holds **per instance**.

### 7.5 Why this needs no new auth mechanism

The controller is just a service principal whose attached policy happens to be the
**control policy**. *"Can the core provision?"* is answered by the **same PDP**
evaluating the **same `service_policies`** used for any other S2S call (§6). Setup is
the only special case (bootstrap, no policy can exist yet) — and it is guarded by the
setup gate, not the PDP.

### 7.6 Persisted setup lock state (NEW — GRPC-021/023)

`IsSetupComplete` should be persisted as **app/bootstrap state**, not stored on
`Tenant` or `Service`.

Why:

- It is not a tenant property. A tenant can exist before setup is deliberately locked,
  and future tenant changes should not reopen bootstrap setup.
- It is not a service property. The seeded `auth` service represents this app as an
  IAM principal; overloading it with bootstrap lifecycle state would mix identity with
  setup control.
- It is instance-wide state: "are mutating setup endpoints still open for this
  deployment?"

Recommended shape: add a small `setup_state` table owned by `internal/setup`, with a
single row keyed by a stable name such as `bootstrap`, plus fields like
`is_complete`, `completed_at`, and optional `completed_by` / `metadata`. In this
pre-release codebase, that means a new canonical create migration
(`NNN_create_setup_state_table.go`) and a matching setup model/repository, following
[database-migrations.md](../contributing/database-migrations.md). `GetSetupStatus`
should still report the derived milestones (`IsTenantSetup`, `IsAdminSetup`,
`IsProfileSetup`), but `IsSetupComplete` must read this persisted lock.

`CompleteSetup` only flips this persisted flag. It does not create tenants, admins,
profiles, services, OAuth clients, or policies. A core-provisioned install may call it
immediately after `RegisterControlService`; a standalone operator calls it when they are
done and want to close the bootstrap registration window.

Decision notes from the review:

| ID | Decision | Status |
|----|----------|--------|
| D1 | Control-policy shape: one shared seeded system policy template attached to each controller vs. per-controller policy created at registration. | **Open** — defaulting to one shared system template until confirmed. |
| D2 | `IsSetupComplete` source of truth. | **Resolved** — use persisted `setup_state`, not derived tenant/admin/profile existence and not a `Tenant`/`Service` field. |
| D3 | `RegisterControlService` shape. | **Resolved** — dedicated setup endpoint/RPC, not an optional field on `CreateTenant` or `CreateAdmin`. |

### 7.7 Provisioning & independence backlog (trackable)

Each item below is independently trackable. `GRPC-015` and `GRPC-191` also appear in
their phase tables (§8/§9); they are repeated here so the whole provisioning model
lives in one place.

| ID | Status | Item | Maps to your point |
|----|--------|------|--------------------|
| GRPC-016 | 🔴 todo | **Independence guarantee:** a standalone setup (tenant + admin) seeds **exactly one principal — its own `auth` system service** — with **no controller** attached; verify default-deny holds and the app is fully functional with zero controllers. Add a regression test. | "standalone setup creates only one service (its own)" |
| GRPC-017 | ✅ done | **Self-service is non-deletable:** the seeded `auth` service is `IsSystem=true` and update/status/**delete** are already blocked in [service_service.go](../../internal/iam/service_service.go). Tracked as a guard — **do not regress**; add a test asserting it. | "the service representing this app is not deletable" |
| GRPC-015 | 🔴 todo | **Seed the default control policy** (`Policy.IsSystem=true`, *unattached*) carrying all actions/permissions a controller needs. **D1 is still open** — defaulting to one shared template until confirmed. | "a prepared/default policy for the control service" |
| GRPC-191 | 🔴 todo | **Register a controller at init** (`RegisterControlService`, gRPC + REST): create a **second service record** for the controller, provision its OAuth `client_credentials` client, and **attach the control policy**. TOFU-gated; **runs only during the setup window before the persisted `CompleteSetup` lock is set**. | "register the core/control plane during setup → another service + attach policy" |
| GRPC-018 | 🔴 todo | **Runtime registration path (no core, or after setup is closed):** register a controller at runtime — create the service + `AssignServicePolicy` (GRPC-110), authenticated + PDP-gated, **not** via the setup endpoint. This is the **only** way to add/change a controller after init; verify it reaches the same end-state as GRPC-191. | "manually register without the core by applying a policy defining the core service" |
| GRPC-019 | 🔴 todo | **Un-provision / revoke control:** detaching (or deleting) the controller's control-policy attachment, and/or removing the controller service, revokes control immediately (PDP + webhook push + short token TTL). The `auth` system service is untouched. | "core can provision and unprovision" |
| GRPC-020 | 🔴 todo | **Multiple instances:** each maintainerd-auth instance seeds its own system service + control-policy template; a controller registers with **each instance independently** (TOFU per instance). Verify isolation between instances. | "core can provision multiple instances" |
| GRPC-021 | 🔴 todo | **One-time setup gate parity (REST ↔ gRPC):** both transports reuse the **same persisted setup-complete flag** (set by `CompleteSetup`, GRPC-023) — every mutating setup operation (incl. `RegisterControlService`) becomes unavailable once the flag is set; only `GetSetupStatus` stays available. `IsSetupComplete` is no longer derived from tenant+admin+profile; it is read from setup state. A completed setup disables setup on **both** ports. | "all setup endpoints are available only once; same for gRPC setup" |
| GRPC-022 | 🔴 todo | **REST equivalent of `RegisterControlService`** — `POST /setup/register-control-service` _(REST)_: same behavior as GRPC-191, exposed on the REST setup surface so a non-gRPC operator/core can register a controller during init. | "add a REST equivalent of RegisterControlService (note: for REST)" |
| GRPC-023 | 🔴 todo | **`setup/complete` lock** — `POST /setup/complete` _(REST + gRPC)_: sets the persisted setup-complete flag that **locks controller registration and all mutating setup ops**. It exists only to close the setup window; it provisions nothing. Anti-infiltration: prevents any other service from registering itself as controller after a standalone setup. Optional `RegisterControlService` lives in the window *before* this call. | "endpoint for setup/complete to lock control-plane registration; register is optional so lock it by flagging setup complete" |

---

## 8. Phase 0 — Foundation backlog (must land first)

| ID | Status | Item | Location |
|----|--------|------|----------|
| GRPC-001 | 🔴 todo | Adopt **buf**: add `buf.yaml` + `buf.gen.yaml`; replace `make proto` raw-`protoc` with `buf generate`; add `buf lint` + `buf breaking` to CI. | `proto/`, `Makefile`, CI |
| GRPC-002 | 🔴 todo | Restructure proto layout: `v1` becomes a directory, split per-domain files (see §4), move `SeederService` into `seeder.proto`. | `proto/maintainerd/auth/v1/` |
| GRPC-003 | 🔴 todo | Fix `go_package` ↔ output-path ↔ import mismatch; standardize generated code under `internal/platform/gen/go`. | `*.proto`, `Makefile`, [grpc.go](../../internal/server/grpc.go) |
| GRPC-004 | 🔴 todo | **Error mapping**: `apperror` → `google.rpc.Code` + `ErrorInfo`/`BadRequest` details helper, used by every handler. | `internal/platform/apperror`, new grpc error adapter |
| GRPC-005 | 🔴 todo | Recovery + structured logging interceptors (request_id correlation). | [internal/server](../../internal/server) |
| GRPC-006 | 🔴 todo | **Auth interceptor**: extract `authorization` metadata, verify service-account token, denylist check, populate `authctx`. | `internal/server`, [jwt](../../internal/platform/jwt) |
| GRPC-007 | 🔴 todo | **Authz interceptor + per-RPC permission registry** reusing REST permission strings + PDP `iam.Evaluate()`. | `internal/server`, [iam](../../internal/iam) |
| GRPC-008 | 🔴 todo | Per-identity rate-limit + request-size + timeout interceptors. | `internal/server`, [middleware](../../internal/platform/middleware) |
| GRPC-009 | 🔴 todo | Register `grpc.health.v1.Health` (wire to readiness probes in [health.go](../../internal/server/health.go)). | [internal/server](../../internal/server) |
| GRPC-010 | 🔴 todo | Enable **server reflection** (gated to non-prod or behind authz) for `grpcurl`/control-plane discovery. | [internal/server](../../internal/server) |
| GRPC-011 | 🔴 todo | TLS/mTLS transport config (cert loading, mesh cert verification, fail-closed). | `internal/server`, [config](../../internal/platform/config) |
| GRPC-012 | 🔴 todo | `common.proto`: shared pagination (`PageRequest`/`PageResponse`), status enums, audit/timestamp fields. | `proto/maintainerd/auth/v1/common.proto` |
| GRPC-013 | 🔴 todo | Handler wiring pattern + `internal/server` service-registration map; one adapter per domain service to its existing service layer (no business logic in transport). | [internal/server](../../internal/server) |
| GRPC-014 | 🔴 todo | gRPC integration test harness (bufconn) + handler test conventions extending [testing.md](../contributing/testing.md). | `tests/integration/` |
| GRPC-015 | 🔴 todo | **Seed the default control policy** (`Policy.IsSystem=true`, *unattached*) — the "manager" template granting management actions over this app; granted/revoked by attaching/detaching it. See §7.2. | `internal/setup/seeder/`, [iam](../../internal/iam) |

---

## 9. Phase 1 — Control-plane (management) services

These mirror the **management** internal-port routes — the primary S2S targets for
"control everything." Each RPC reuses the existing service layer and the REST
permission string shown. Status is per-RPC.

> **Convention:** REST `GET /xs/` → `ListXs`, `GET /xs/{uuid}` → `GetX`,
> `POST /xs/` → `CreateX`, `PUT /xs/{uuid}` → `UpdateX`, `PUT/PATCH …/status` →
> `SetXStatus`, `DELETE /xs/{uuid}` → `DeleteX`. `RequireStepUp` is noted where the
> REST route requires it (gRPC enforces it as an additional claim/authz check).

### GRPC-100 · SeederService — `seeder.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `TriggerSeeder` | (existing gRPC) | service-account | ✅ done |

### GRPC-101 · TenantService — `tenant.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `GetDefaultTenant` | `GET /tenant/` | (read) | 🔴 todo |
| `GetTenantByIdentifier` | `GET /tenant/{identifier}` | (read) | 🔴 todo |
| `ListTenants` | `GET /tenants/` | `tenant:read` | 🔴 todo |
| `GetTenant` | `GET /tenants/{uuid}` | `tenant:read` | 🔴 todo |
| `CreateTenant` | `POST /tenants/` | `tenant:create` | 🔴 todo |
| `UpdateTenant` | `PUT /tenants/{uuid}` | `tenant:update` | 🔴 todo |
| `SetTenantStatus` | `PUT /tenants/{uuid}/status` | `tenant:update` + step-up | 🔴 todo |
| `SetTenantPublic` | `PUT /tenants/{uuid}/public` | `tenant:update` + step-up | 🔴 todo |
| `DeleteTenant` | `DELETE /tenants/{uuid}` | `tenant:delete` + step-up | 🔴 todo |
| `ListTenantMembers` | `GET /tenants/{uuid}/members/` | `tenant:read` | 🔴 todo |
| `AddTenantMember` | `POST /tenants/{uuid}/members/` | `tenant:update` | 🔴 todo |
| `UpdateTenantMemberRole` | `PATCH /tenants/{uuid}/members/{m}/role` | `tenant:update` | 🔴 todo |
| `RemoveTenantMember` | `DELETE /tenants/{uuid}/members/{m}` | `tenant:update` | 🔴 todo |

### GRPC-102 · TenantSettingService — `tenant.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `GetRateLimitConfig` / `UpdateRateLimitConfig` | `GET/PUT /tenant-settings/rate-limit` | `tenant-setting:read` / `:update` | 🔴 todo |
| `GetAuditConfig` / `UpdateAuditConfig` | `GET/PUT /tenant-settings/audit` | `tenant-setting:read` / `:update` | 🔴 todo |
| `GetMaintenanceConfig` / `UpdateMaintenanceConfig` | `GET/PUT /tenant-settings/maintenance` | `tenant-setting:read` / `:update` | 🔴 todo |
| `GetFeatureFlags` / `UpdateFeatureFlags` | `GET/PUT /tenant-settings/feature-flags` | `tenant-setting:read` / `:update` | 🔴 todo |

### GRPC-110 · ServiceService — `iam.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `GetMyPolicyBundle` | `GET /services/me/policy-bundle` | service-account (self) | 🔴 todo |
| `ListServices` | `GET /services/` | `service:read` | 🔴 todo |
| `GetService` | `GET /services/{uuid}` | `service:read` | 🔴 todo |
| `CreateService` | `POST /services/` | `service:create` | 🔴 todo |
| `UpdateService` | `PUT /services/{uuid}` | `service:update` | 🔴 todo |
| `SetServiceStatus` | `PUT /services/{uuid}/status` | `service:update` | 🔴 todo |
| `DeleteService` | `DELETE /services/{uuid}` | `service:delete` | 🔴 todo |
| `AssignServicePolicy` | `POST /services/{uuid}/policies/{p}` | `service:policy:assign` | 🔴 todo |
| `RemoveServicePolicy` | `DELETE /services/{uuid}/policies/{p}` | `service:policy:remove` | 🔴 todo |

### GRPC-111 · APIService — `iam.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListAPIs` / `GetAPI` | `GET /apis/`, `GET /apis/{uuid}` | `api:read` | 🔴 todo |
| `CreateAPI` | `POST /apis/` | `api:create` | 🔴 todo |
| `UpdateAPI` | `PUT /apis/{uuid}` | `api:update` | 🔴 todo |
| `SetAPIStatus` | `PUT /apis/{uuid}/status` | `api:update` | 🔴 todo |
| `DeleteAPI` | `DELETE /apis/{uuid}` | `api:delete` | 🔴 todo |

### GRPC-112 · PermissionService — `iam.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListPermissions` / `GetPermission` | `GET /permissions/`, `…/{uuid}` | `permission:read` | 🔴 todo |
| `CreatePermission` | `POST /permissions/` | `permission:create` | 🔴 todo |
| `UpdatePermission` | `PUT /permissions/{uuid}` | `permission:update` | 🔴 todo |
| `SetPermissionStatus` | `PUT /permissions/{uuid}/status` | `permission:update` | 🔴 todo |
| `DeletePermission` | `DELETE /permissions/{uuid}` | `permission:delete` | 🔴 todo |

### GRPC-113 · PolicyService — `iam.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListPolicies` / `GetPolicy` | `GET /policies/`, `…/{uuid}` | `policy:read` | 🔴 todo |
| `ListPolicyServices` | `GET /policies/{uuid}/services` | `policy:read` | 🔴 todo |
| `CreatePolicy` | `POST /policies/` | `policy:create` | 🔴 todo |
| `UpdatePolicy` | `PUT /policies/{uuid}` | `policy:update` | 🔴 todo |
| `SetPolicyStatus` | `PUT /policies/{uuid}/status` | `policy:update` | 🔴 todo |
| `DeletePolicy` | `DELETE /policies/{uuid}` | `policy:delete` | 🔴 todo |

### GRPC-114 · RoleService — `iam.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListRoles` / `GetRole` | `GET /roles/`, `…/{uuid}` | `role:read` | 🔴 todo |
| `CreateRole` | `POST /roles/` | `role:create` | 🔴 todo |
| `UpdateRole` | `PUT /roles/{uuid}` | `role:update` | 🔴 todo |
| `SetRoleStatus` | `PUT /roles/{uuid}/status` | `role:update` | 🔴 todo |
| `DeleteRole` | `DELETE /roles/{uuid}` | `role:delete` | 🔴 todo |
| `ListRolePermissions` | `GET /roles/{uuid}/permissions` | `role:read` | 🔴 todo |
| `AddRolePermissions` | `POST /roles/{uuid}/permissions` | `role:permission:create` | 🔴 todo |
| `RemoveRolePermission` | `DELETE /roles/{uuid}/permissions/{p}` | `role:permission:delete` | 🔴 todo |

### GRPC-115 · AuthorizationService — `iam.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `Authorize` | `POST /authorize/` | service-account | 🔴 todo |

> This is the **S2S decision RPC** (PDP). High value: peers can ask "can principal X
> do action Y on resource Z?" over gRPC instead of embedding the SDK.

### GRPC-120 · IdentityProviderService — `identity_provider.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListIdentityProviders` / `GetIdentityProvider` | `GET /identity_providers/`, `…/{uuid}` | `idp:read` | 🔴 todo |
| `CreateIdentityProvider` | `POST /identity_providers/` | `idp:create` | 🔴 todo |
| `UpdateIdentityProvider` | `PUT /identity_providers/{uuid}` | `idp:update` | 🔴 todo |
| `SetIdentityProviderStatus` | `PUT /identity_providers/{uuid}/status` | `idp:update` | 🔴 todo |
| `DeleteIdentityProvider` | `DELETE /identity_providers/{uuid}` | `idp:delete` | 🔴 todo |

### GRPC-121 · SignupFlowService — `identity_provider.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListSignupFlows` / `GetSignupFlow` | `GET /signup_flows/`, `…/{uuid}` | `signup-flow:read` | 🔴 todo |
| `CreateSignupFlow` | `POST /signup_flows/` | `signup-flow:create` | 🔴 todo |
| `UpdateSignupFlow` | `PUT /signup_flows/{uuid}` | `signup-flow:update` | 🔴 todo |
| `SetSignupFlowStatus` | `PATCH /signup_flows/{uuid}/status` | `signup-flow:update` | 🔴 todo |
| `DeleteSignupFlow` | `DELETE /signup_flows/{uuid}` | `signup-flow:delete` | 🔴 todo |
| `AssignSignupFlowRoles` | `POST /signup_flows/{uuid}/roles/` | `signup-flow:update` | 🔴 todo |
| `ListSignupFlowRoles` | `GET /signup_flows/{uuid}/roles/` | `signup-flow:read` | 🔴 todo |
| `RemoveSignupFlowRole` | `DELETE /signup_flows/{uuid}/roles/{r}` | `signup-flow:update` | 🔴 todo |

### GRPC-130 · ClientService — `client.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListClients` / `GetClient` | `GET /clients/`, `…/{uuid}` | `client:read` | 🔴 todo |
| `GetClientSecret` | `GET /clients/{uuid}/secret` | `client:secret:read` + step-up | 🔴 todo |
| `RotateClientSecret` | `POST /clients/{uuid}/rotate-secret` | `client:secret:rotate` + step-up | 🔴 todo |
| `GetClientConfig` | `GET /clients/{uuid}/config` | `client:config:read` | 🔴 todo |
| `CreateClient` | `POST /clients/` | `client:create` | 🔴 todo |
| `UpdateClient` | `PUT /clients/{uuid}` | `client:update` | 🔴 todo |
| `SetClientStatus` | `PUT /clients/{uuid}/status` | `client:update` | 🔴 todo |
| `DeleteClient` | `DELETE /clients/{uuid}` | `client:delete` | 🔴 todo |
| `ListClientURIs` | `GET /clients/{uuid}/uris` | `client:uri:read` | 🔴 todo |
| `CreateClientURI` | `POST /clients/{uuid}/uris` | `client:uri:create` | 🔴 todo |
| `UpdateClientURI` | `PUT /clients/{uuid}/uris/{u}` | `client:uri:update` | 🔴 todo |
| `DeleteClientURI` | `DELETE /clients/{uuid}/uris/{u}` | `client:uri:delete` | 🔴 todo |
| `ListClientAPIs` | `GET /clients/{uuid}/apis` | `client:api:read` | 🔴 todo |
| `AddClientAPIs` | `POST /clients/{uuid}/apis` | `client:api:create` | 🔴 todo |
| `RemoveClientAPI` | `DELETE /clients/{uuid}/apis/{a}` | `client:api:delete` | 🔴 todo |
| `ListClientAPIPermissions` | `GET /clients/{uuid}/apis/{a}/permissions` | `client:api:permission:read` | 🔴 todo |
| `AddClientAPIPermissions` | `POST /clients/{uuid}/apis/{a}/permissions` | `client:api:permission:create` | 🔴 todo |
| `RemoveClientAPIPermission` | `DELETE /clients/{uuid}/apis/{a}/permissions/{p}` | `client:api:permission:delete` | 🔴 todo |

### GRPC-131 · APIKeyService — `client.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListAPIKeys` / `GetAPIKey` | `GET /api_keys/`, `…/{uuid}` | `api_key:read` | 🔴 todo |
| `GetAPIKeyConfig` | `GET /api_keys/{uuid}/config` | `api_key:read` | 🔴 todo |
| `CreateAPIKey` | `POST /api_keys/` | `api_key:create` + step-up | 🔴 todo |
| `UpdateAPIKey` | `PUT /api_keys/{uuid}` | `api_key:update` + step-up | 🔴 todo |
| `SetAPIKeyStatus` | `PUT /api_keys/{uuid}/status` | `api_key:update` + step-up | 🔴 todo |
| `DeleteAPIKey` | `DELETE /api_keys/{uuid}` | `api_key:delete` + step-up | 🔴 todo |
| `ListAPIKeyAPIs` | `GET /api_keys/{uuid}/apis/` | `api_key:read` | 🔴 todo |
| `AddAPIKeyAPIs` | `POST /api_keys/{uuid}/apis/` | `api_key:update` + step-up | 🔴 todo |
| `RemoveAPIKeyAPI` | `DELETE /api_keys/{uuid}/apis/{a}` | `api_key:update` + step-up | 🔴 todo |
| `ListAPIKeyAPIPermissions` | `GET /api_keys/{uuid}/apis/{a}/permissions/` | `api_key:read` | 🔴 todo |
| `AddAPIKeyAPIPermissions` | `POST /api_keys/{uuid}/apis/{a}/permissions/` | `api_key:update` + step-up | 🔴 todo |
| `RemoveAPIKeyAPIPermission` | `DELETE /api_keys/{uuid}/apis/{a}/permissions/{p}` | `api_key:update` + step-up | 🔴 todo |

### GRPC-140 · UserService (admin) — `user.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListUsers` / `GetUser` | `GET /users/`, `…/{uuid}` | `user:read` | 🔴 todo |
| `CreateUser` | `POST /users/` | `user:create` | 🔴 todo |
| `UpdateUser` | `PUT /users/{uuid}` | `user:update` | 🔴 todo |
| `SetUserStatus` | `PATCH /users/{uuid}/status` | `user:update` + step-up | 🔴 todo |
| `VerifyUserEmail` | `PATCH /users/{uuid}/verify-email` | `user:update` | 🔴 todo |
| `VerifyUserPhone` | `PATCH /users/{uuid}/verify-phone` | `user:update` | 🔴 todo |
| `CompleteUserAccount` | `PATCH /users/{uuid}/complete-account` | `user:update` | 🔴 todo |
| `DeleteUser` | `DELETE /users/{uuid}` | `user:delete` + step-up | 🔴 todo |
| `ForceUserPasswordChange` | `PUT /users/{uuid}/force-password-change` | `user:update` + step-up | 🔴 todo |
| `ListUserRoles` | `GET /users/{uuid}/roles` | `user:read` | 🔴 todo |
| `ListUserIdentities` | `GET /users/{uuid}/identities` | `user:read` | 🔴 todo |
| `AssignUserRoles` | `POST /users/{uuid}/roles` | `user:create` + step-up | 🔴 todo |
| `RemoveUserRole` | `DELETE /users/{uuid}/roles/{r}` | `user:create` + step-up | 🔴 todo |
| `ListUserProfiles` | `GET /users/{uuid}/profiles` | `user:read` | 🔴 todo |
| `CreateUserProfile` | `POST /users/{uuid}/profiles` | `user:update` | 🔴 todo |
| `GetUserProfile` | `GET /users/{uuid}/profiles/{p}` | `user:read` | 🔴 todo |
| `UpdateUserProfile` | `PUT /users/{uuid}/profiles/{p}` | `user:update` | 🔴 todo |
| `SetDefaultUserProfile` | `PUT /users/{uuid}/profiles/{p}/set-default` | `user:update` | 🔴 todo |
| `DeleteUserProfile` | `DELETE /users/{uuid}/profiles/{p}` | `user:delete` | 🔴 todo |

### GRPC-141 · InviteService — `user.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `SendInvite` | `POST /invite/` | (authenticated admin) | 🔴 todo |

### GRPC-150 · SecuritySettingService — `security.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `GetMFAConfig` / `UpdateMFAConfig` | `GET/PUT /security-settings/mfa` | `security-setting:read` / `:update` (+step-up) | 🔴 todo |
| `GetPasswordConfig` / `UpdatePasswordConfig` | `…/password` | `security-setting:read` / `:update` (+step-up) | 🔴 todo |
| `GetSessionConfig` / `UpdateSessionConfig` | `…/session` | `security-setting:read` / `:update` (+step-up) | 🔴 todo |
| `GetThreatConfig` / `UpdateThreatConfig` | `…/threat` | `security-setting:read` / `:update` (+step-up) | 🔴 todo |
| `GetLockoutConfig` / `UpdateLockoutConfig` | `…/lockout` | `security-setting:read` / `:update` (+step-up) | 🔴 todo |
| `GetRegistrationConfig` / `UpdateRegistrationConfig` | `…/registration` | `security-setting:read` / `:update` (+step-up) | 🔴 todo |
| `GetTokenConfig` / `UpdateTokenConfig` | `…/token` | `security-setting:read` / `:update` (+step-up) | 🔴 todo |

### GRPC-151 · IPRestrictionRuleService — `security.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListIPRestrictionRules` / `GetIPRestrictionRule` | `GET /ip-restriction-rules/`, `…/{uuid}` | `ip-restriction-rule:read` | 🔴 todo |
| `CreateIPRestrictionRule` | `POST /ip-restriction-rules/` | `ip-restriction-rule:create` | 🔴 todo |
| `UpdateIPRestrictionRule` | `PUT /ip-restriction-rules/{uuid}` | `ip-restriction-rule:update` | 🔴 todo |
| `DeleteIPRestrictionRule` | `DELETE /ip-restriction-rules/{uuid}` | `ip-restriction-rule:delete` | 🔴 todo |
| `SetIPRestrictionRuleStatus` | `PATCH /ip-restriction-rules/{uuid}/status` | `ip-restriction-rule:update` | 🔴 todo |

### GRPC-160 · BrandingService — `branding.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `GetBranding` / `UpdateBranding` | `GET/PUT /branding/` | `branding:read` / `:update` | 🔴 todo |

### GRPC-161 · EmailTemplateService — `branding.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListEmailTemplates` / `GetEmailTemplate` | `GET /email_templates/`, `…/{uuid}` | `email-template:read` | 🔴 todo |
| `CreateEmailTemplate` | `POST /email_templates/` | `email-template:create` | 🔴 todo |
| `UpdateEmailTemplate` | `PUT /email_templates/{uuid}` | `email-template:update` | 🔴 todo |
| `DeleteEmailTemplate` | `DELETE /email_templates/{uuid}` | `email-template:delete` | 🔴 todo |
| `SetEmailTemplateStatus` | `PATCH /email_templates/{uuid}/status` | `email-template:update` | 🔴 todo |

### GRPC-162 · SMSTemplateService — `branding.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListSMSTemplates` / `GetSMSTemplate` | `GET /sms_templates/`, `…/{uuid}` | `sms-template:read` | 🔴 todo |
| `CreateSMSTemplate` | `POST /sms_templates/` | `sms-template:create` | 🔴 todo |
| `UpdateSMSTemplate` | `PUT /sms_templates/{uuid}` | `sms-template:update` | 🔴 todo |
| `DeleteSMSTemplate` | `DELETE /sms_templates/{uuid}` | `sms-template:delete` | 🔴 todo |
| `SetSMSTemplateStatus` | `PATCH /sms_templates/{uuid}/status` | `sms-template:update` | 🔴 todo |

### GRPC-163 · LoginTemplateService — `branding.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListLoginTemplates` / `GetLoginTemplate` | `GET /login_templates/`, `…/{uuid}` | `login-template:read` | 🔴 todo |
| `CreateLoginTemplate` | `POST /login_templates/` | `login-template:create` | 🔴 todo |
| `UpdateLoginTemplate` | `PUT /login_templates/{uuid}` | `login-template:update` | 🔴 todo |
| `DeleteLoginTemplate` | `DELETE /login_templates/{uuid}` | `login-template:delete` | 🔴 todo |
| `SetLoginTemplateStatus` | `PATCH /login_templates/{uuid}/status` | `login-template:update` | 🔴 todo |

### GRPC-170 · EmailConfigService / SMSConfigService — `notifier.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `GetEmailConfig` / `UpdateEmailConfig` | `GET/PUT /email-config/` | `email-config:read` / `:update` | 🔴 todo |
| `GetSMSConfig` / `UpdateSMSConfig` | `GET/PUT /sms-config/` | `sms-config:read` / `:update` | 🔴 todo |

### GRPC-171 · WebhookEndpointService — `webhook.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListWebhookEndpoints` / `GetWebhookEndpoint` | `GET /webhook-endpoints/`, `…/{uuid}` | `webhook-endpoint:read` | 🔴 todo |
| `CreateWebhookEndpoint` | `POST /webhook-endpoints/` | `webhook-endpoint:create` | 🔴 todo |
| `UpdateWebhookEndpoint` | `PUT /webhook-endpoints/{uuid}` | `webhook-endpoint:update` | 🔴 todo |
| `DeleteWebhookEndpoint` | `DELETE /webhook-endpoints/{uuid}` | `webhook-endpoint:delete` | 🔴 todo |
| `SetWebhookEndpointStatus` | `PATCH /webhook-endpoints/{uuid}/status` | `webhook-endpoint:update` | 🔴 todo |

### GRPC-172 · AuthEventService — `authevent.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListAuthEvents` | `GET /auth-events/` | `auth_event:read` | 🔴 todo |
| `CountAuthEventsByType` | `GET /auth-events/count` | `auth_event:read` | 🔴 todo |
| `GetAuthEvent` | `GET /auth-events/{uuid}` | `auth_event:read` | 🔴 todo |

### GRPC-180 · OAuthIntrospectionService — `oauth.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `Introspect` | `POST /oauth/introspect` | service-account | 🔴 todo |

### GRPC-190 · SetupService — `setup.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `GetSetupStatus` | `GET /setup/status` | (bootstrap) | 🔴 todo |
| `CreateTenant` | `POST /setup/create_tenant` | (bootstrap) | 🔴 todo |
| `CreateAdmin` | `POST /setup/create_admin` | (bootstrap) | 🔴 todo |
| `CreateProfile` | `POST /setup/create_profile` | (bootstrap) | 🔴 todo |
| `RegisterControlService` (GRPC-191) | NEW (REST `POST /setup/register-control-service`, GRPC-022 + gRPC) | (bootstrap, TOFU, **optional**) | 🔴 todo |
| `CompleteSetup` (GRPC-023) | NEW (REST `POST /setup/complete` + gRPC) | (bootstrap, final) | 🔴 todo |

> **SetupService** is the control plane's natural provisioning entry point
> (a fresh tenant + admin) — high S2S value despite living under the "bootstrap"
> auth flows. Bootstrap auth differs (no policy yet exists); guard with the
> existing setup gate, not the PDP.
>
> **One-time, then disabled — REST and gRPC share one gate (GRPC-021).** Today REST
> derives completion from tenant/admin/profile existence in
> [service_setup.go](../../internal/setup/service_setup.go). GRPC-021 changes that
> to one persisted setup-complete flag used by both transports. Every mutating setup
> operation (`CreateTenant`, `CreateAdmin`, `CreateProfile`,
> `RegisterControlService`) becomes unavailable once the persisted lock is set.
> `GetSetupStatus` is the **only** setup operation that stays available afterward
> (read-only probe).
>
> **`RegisterControlService` (GRPC-191)** implements §7.3 path 1: the caller passes a
> controller **service name/identifier**; auth creates the `Service`, provisions its
> OAuth `client_credentials` client, and **attaches the seeded control policy**
> (GRPC-015) — the trust-on-first-use grant that lets the core provision/unprovision.
> It is **optional** and exposed on **both** transports (REST `POST
> /setup/register-control-service` — GRPC-022 — and the gRPC RPC).
>
> **Setup ordering & the explicit lock (`CompleteSetup` / GRPC-023).** A standalone
> install runs `create_tenant → create_admin → create_profile` and then, optionally,
> `RegisterControlService`. Because the controller step is optional, completion
> **must be flagged explicitly** by **`CompleteSetup`** (`POST /setup/complete`) —
> not merely inferred from "profile exists." `CompleteSetup` sets the persisted
> setup-complete flag, which **locks all mutating setup operations, including
> `RegisterControlService`**. It provisions nothing; it only closes the bootstrap
> window. This is the **anti-infiltration control**: once a standalone operator
> completes setup, no other service can sneak in and register itself as the
> controller through the setup path. After the lock, controller changes happen only
> via the runtime path (`AssignServicePolicy`, GRPC-018 — authenticated +
> PDP-gated). `GetSetupStatus` remains available; everything else is closed.

---

## 10. Phase 2 — Identity & end-user-flow services (deferred)

These internal-port routes are **interactive end-user auth flows** (browser/app
driven). They are listed for completeness because the gRPC backlog mirrors all
private application APIs, but they are **lower priority for S2S** — a control plane
rarely drives a user's login/MFA ceremony. Default status 🔴 **todo**, deferred
unless a concrete S2S consumer needs them.

### GRPC-200 · AuthnService — `authn.proto`
| RPC | REST origin | Auth shape | Status |
|-----|-------------|------------|--------|
| `Register` | `POST /register` | bootstrap/public-style authn | 🔴 todo |
| `RegisterInvite` | `POST /register/invite` | invite token | 🔴 todo |
| `Login` | `POST /login` | credential flow | 🔴 todo |
| `Logout` | `POST /logout` | authenticated session/token | 🔴 todo |
| `ForgotPassword` | `POST /forgot-password` | unauthenticated recovery | 🔴 todo |
| `ResetPassword` | `POST /reset-password` | reset token | 🔴 todo |
| `SendVerificationEmail` | `POST /email-verification/send` | authenticated/user context | 🔴 todo |
| `VerifyEmail` | `POST /email-verification/verify` | verification token | 🔴 todo |
| `SendMagicLink` | `POST /magic-link/send` | unauthenticated/email flow | 🔴 todo |
| `VerifyMagicLink` | `POST /magic-link/verify` | magic-link token | 🔴 todo |
| `SendSMSLoginOTP` | `POST /sms-login/send` | unauthenticated phone flow | 🔴 todo |
| `VerifySMSLoginOTP` | `POST /sms-login/verify` | OTP verification | 🔴 todo |

### GRPC-201 · ProfileService (self) — `user.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `GetDefaultProfile` | `GET /profile/` | `account:profile:read:self` | 🔴 todo |
| `CreateOrUpdateDefaultProfile` | `POST /profile/` | `account:profile:update:self` | 🔴 todo |
| `UpdateDefaultProfile` | `PUT /profile/` | `account:profile:update:self` | 🔴 todo |
| `DeleteDefaultProfile` | `DELETE /profile/` | `account:profile:delete:self` | 🔴 todo |
| `ListProfiles` | `GET /profiles/` | `account:profile:read:self` | 🔴 todo |
| `CreateProfileSelf` | `POST /profiles/` | `account:profile:update:self` | 🔴 todo |
| `GetProfileSelf` | `GET /profiles/{uuid}` | `account:profile:read:self` | 🔴 todo |
| `UpdateProfileSelf` | `PUT /profiles/{uuid}` | `account:profile:update:self` | 🔴 todo |
| `SetDefaultProfileSelf` | `PATCH /profiles/{uuid}/set-default` | `account:profile:update:self` | 🔴 todo |
| `DeleteProfileSelf` | `DELETE /profiles/{uuid}` | `account:profile:delete:self` | 🔴 todo |

### GRPC-202 · UserSettingService (self) — `user.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `CreateOrUpdateUserSettings` | `POST /user-settings/` | `settings:update:self` | 🔴 todo |
| `GetUserSettings` | `GET /user-settings/` | `settings:read:self` | 🔴 todo |
| `DeleteUserSettings` | `DELETE /user-settings/` | `settings:update:self` | 🔴 todo |

### GRPC-203 · AccountService (self) — `user.proto`
| RPC | REST origin | Auth shape | Status |
|-----|-------------|------------|--------|
| `InitiateEmailChange` | `POST /account/email/change` | authenticated user | 🔴 todo |
| `VerifyEmailChange` | `POST /account/email/verify` | authenticated user + token | 🔴 todo |
| `ChangeUsername` | `PUT /account/username` | authenticated user | 🔴 todo |
| `DeleteAccount` | `DELETE /account/` | authenticated user + step-up | 🔴 todo |
| `ExportAccountData` | `GET /account/export` | authenticated user | 🔴 todo |
| `GenerateBackupCodes` | `POST /account/backup-codes` | authenticated user + step-up | 🔴 todo |
| `ListSessions` | `GET /account/sessions` | authenticated user | 🔴 todo |
| `RevokeAllSessions` | `DELETE /account/sessions` | authenticated user + step-up | 🔴 todo |
| `RevokeSession` | `DELETE /account/sessions/{uuid}` | authenticated user | 🔴 todo |

### GRPC-204 · RecoveryService — `user.proto`
| RPC | REST origin | Auth shape | Status |
|-----|-------------|------------|--------|
| `VerifyBackupCode` | `POST /recovery/backup-code` | unauthenticated recovery | 🔴 todo |

### GRPC-205 · MFAService — `mfa.proto`
| RPC | REST origin | Auth shape | Status |
|-----|-------------|------------|--------|
| `GetMFAStatus` | `GET /mfa/status` | authenticated user | 🔴 todo |
| `BeginTOTPEnrollment` | `POST /mfa/totp/enroll` | authenticated user | 🔴 todo |
| `FinishTOTPEnrollment` | `POST /mfa/totp/verify` | authenticated user | 🔴 todo |
| `DisableTOTP` | `DELETE /mfa/totp` | authenticated user + step-up | 🔴 todo |
| `GetBackupCodesCount` | `GET /mfa/backup-codes/count` | authenticated user | 🔴 todo |
| `RegenerateBackupCodes` | `POST /mfa/backup-codes/regenerate` | authenticated user + step-up | 🔴 todo |
| `BeginWebAuthnRegistration` | `POST /mfa/webauthn/register/begin` | authenticated user | 🔴 todo |
| `FinishWebAuthnRegistration` | `POST /mfa/webauthn/register/finish` | authenticated user | 🔴 todo |
| `BeginWebAuthnAuthentication` | `POST /mfa/webauthn/auth/begin` | authenticated user | 🔴 todo |
| `FinishWebAuthnAuthentication` | `POST /mfa/webauthn/auth/finish` | authenticated user | 🔴 todo |
| `DeleteWebAuthnCredential` | `DELETE /mfa/webauthn/{uuid}` | authenticated user + step-up | 🔴 todo |
| `IssueStepUpChallenge` | `POST /mfa/step-up/challenge` | authenticated user | 🔴 todo |
| `VerifyStepUp` | `POST /mfa/step-up/verify` | authenticated user | 🔴 todo |
| `AdminResetMFA` | `POST /mfa/admin/users/{uuid}/reset` | authenticated admin + step-up | 🔴 todo |

### GRPC-206 · FederationService — `identity_provider.proto`
| RPC | REST origin | Auth shape | Status |
|-----|-------------|------------|--------|
| `ExchangeExternalToken` | `POST /federation/token` | upstream identity token | 🔴 todo |
| `ExchangeOAuth2Code` | `POST /federation/oauth2/callback` | OAuth2 callback/code | 🔴 todo |
| `HomeRealmDiscovery` | `GET /federation/hrd` | unauthenticated discovery | 🔴 todo |
| `ListLinkedIdentities` | `GET /account/identities/` | authenticated user | 🔴 todo |
| `LinkIdentity` | `POST /account/identities/link` | authenticated user | 🔴 todo |
| `UnlinkIdentity` | `DELETE /account/identities/{uuid}` | authenticated user | 🔴 todo |

> **Promotion candidates** from Phase 2 to Phase 1 if a control-plane need appears:
> `MFAService.AdminResetMFA` (admin op), `AuthnService` token-issuance for
> machine-driven onboarding.

---

## 11. Deferred to a later version (v2 / future)

- **gRPC-Web / public gateway** — if any browser or third-party needs gRPC; would
  require its own public-port surface and rate-limit posture (out of scope here).
- **Streaming RPCs** — e.g. server-stream `WatchPolicyChanges` as a gRPC-native
  alternative to the webhook push (Pattern 3). Nice-to-have once bundle
  distribution is gRPC-served.
- **Resource policies / conditions** — tracks the same v2 deferral as
  [service-to-service-authorization.md](../documentations/service-to-service-authorization/service-to-service-authorization.md) §9.
- **Bulk / batch RPCs** (AIP-231/233) — batched reads/writes for the control plane
  if N+1 call patterns emerge.

---

## 12. Testing obligations

Per [testing.md](../contributing/testing.md), gRPC work carries the same bar as REST:

- **Handler (RPC) tests** — adapt the 9-step checklist to interceptors: auth
  (missing/invalid token) → authz (no policy / explicit deny) → request validation
  → deps → business rules → service call → success. Use **bufconn** for in-process
  server tests (GRPC-014).
- **Interceptor tests** — the auth and authz interceptors get dedicated tests:
  default-deny when no policy, allow on matching statement, deny on explicit deny,
  token expiry/denylist rejection.
- **Error-mapping tests** — one sub-test per `apperror` → `codes.*` mapping
  (GRPC-004).
- **Reuse the service layer's existing unit tests** — gRPC handlers are thin
  adapters; don't duplicate business-rule coverage, just the transport mapping.
- Keep affected package coverage ≥ 80% (iam, server) per the project baseline.

---

## 13. Milestone summary

| Milestone | Items | Outcome |
|-----------|-------|---------|
| **M0 — Foundation** | GRPC-001…017 | buf layout, codegen, auth/authz/observability interceptors, health, reflection, TLS, error mapping, **default control policy seed** + **independence guarantees** (GRPC-015/016/017), test harness. **Blocks everything.** |
| **M1 — IAM + Tenant core + controller lifecycle** | GRPC-101/102/110–115, GRPC-190/191, GRPC-018…023 | The control plane can register itself at init (TOFU, lockable via explicit `CompleteSetup` — GRPC-021/022/023), be un-registered, and run multiple instances; then manage tenants, services, policies, roles, and ask `Authorize`. Highest S2S value. |
| **M2 — Clients, Users, IdP** | GRPC-120/121/130/131/140/141 | Full provisioning surface (clients, api-keys, users, identity providers, signup flows, invites). |
| **M3 — Settings, Branding, Notifier, Webhooks, Events, OAuth, Setup** | GRPC-150…190 | Remaining management surface + introspection + provisioning. |
| **M4 — (optional) Identity flows** | GRPC-2xx | Only if a concrete S2S consumer needs end-user flows. |
