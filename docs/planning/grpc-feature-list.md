# gRPC Feature List — Service-to-Service Control Plane Transport

**Status:** Phase 0 foundation complete; Phase 1 management surface mostly complete. Remaining Phase 1 backlog is explicitly listed below — proposed for **v1.1.0** (post REST/S2S-authz baseline).
**Owner:** rseguma@lula.life
**Created:** 2026-06-04
**Related:** [service-to-service-authorization.md](../documentations/service-to-service-authorization/service-to-service-authorization.md) · [architecture.md](../documentations/architecture/architecture.md) · [code-structure.md](../contributing/code-structure.md) · [testing.md](../contributing/testing.md)

This document is the implementation backlog for exposing Lula's auth server over
**gRPC** as an additional **service-to-service (S2S) control-plane transport**.
S2S is **not gRPC-only** in this project: the internal/private REST port already
supports S2S use cases and keeps doing so. The goal of this backlog is to add a
typed, high-performance gRPC channel that lets other services — the **core /
control plane** especially — **configure and manage** this auth service
(create/update/delete resources, settings, policies, users, clients, providers,
webhooks, templates, etc.) and perform necessary **verification / decision**
reads (for example token introspection and authorization checks).

It does **not** backlog every internal-port application API as gRPC. Browser,
frontend, and end-user-interactive flows stay REST-first/REST-only unless a clear
service-to-service management or verification need appears. This document assigns
each eligible management/config/verification RPC a status and pins down the
**contract layout, naming, and S2S authentication standard** so the work follows
gRPC best practice from the start.
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

> Everything is 🔴 **todo** unless noted. Phase 0 items that landed are marked ✅
> below; partially implemented foundation items stay 🟡 until their remaining
> coverage or integration/CI work lands.

**ID convention:** `GRPC-0xx` = Phase 0 foundation **plus cross-cutting
provisioning/setup items** (GRPC-015…023, gathered in §7.7). `GRPC-1xx` =
control-plane management, configuration, and verification services. `GRPC-2xx`
is intentionally reserved for a future decision and is not used for end-user
auth ceremonies in this backlog. Per-RPC status lives in each service's table.

---

## 1. Goal & scope

|                     |                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Independence**    | maintainerd-auth is **fully standalone** — it boots, migrates, seeds, and serves with **no dependency on the core / control plane**. The core is an *optional* manager, never a runtime requirement. The gRPC surface assumes nothing about the core's existence. See §7.                                                                                                                                                               |
| **Why gRPC**        | Typed contracts, codegen for every consumer language, streaming-capable, low overhead — a strong fit for **service-to-service** traffic (core/control plane ↔ auth, and peer services reading auth data). It is an additional S2S transport, not the only S2S path.                                                                                                                                               |
| **What it mirrors** | The **management/configuration subset** of the internal/private REST port (`:8080`, VPN/private-network only — see [router.go](../../internal/server/router.go) `buildInternalRouter`). gRPC becomes a private-network-only control surface beside REST, **never** exposed to the public internet (the public port `:8081` stays REST-only). Operational probes/spec endpoints are covered by gRPC health, reflection, and generated proto contracts. |
| **Two use-cases**   | **(a) Control/configuration** — external services (e.g. the core/control plane) mutate auth resources (tenants, clients, policies, users, settings…). **(b) Verification/decision reads** — peer services confirm auth facts needed to serve their own users (introspect tokens, ask `Authorize`, fetch policy bundles, read identities/roles where needed).                                                                                         |
| **Auth model**      | Transport-agnostic S2S. REST and gRPC both use **service-account access tokens** and PDP policy checks. For gRPC, every RPC carries the token in metadata and is policy-gated. **A policy must always exist** for a caller to reach a protected REST endpoint or gRPC RPC — default-deny. See §6.                                                                                                                                         |
| **Out of scope**    | Browser/public/end-user-interactive traffic stays on REST: self-registration, login, logout, password reset, magic-link/SMS login, user self-profile/account settings, and MFA ceremonies. gRPC-Web/public gateway and breaking v2 changes are also out of scope — see §10.                                                                                                                                                             |

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
| Domain RPC registrations | [grpc.go](../../internal/server/grpc.go) | ✅ all current Phase 1 services registered |
| Proto source | `proto/maintainerd/auth/v1/` (`v1` is now a directory) | ✅ restructured |
| Generated Go | `internal/platform/gen/go/maintainerd/auth/` | ✅ aligned |
| Codegen | `make proto` via buf (`buf generate`) | ✅ migrated |
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

**What was fixed (it's the _layout_, not the name):**

1. `v1` is a **directory**, not a filename (buf style requires
   `…/<version>/…proto`), split by service/domain instead of one giant file:

   ```
   proto/
     buf.yaml                      # module + lint + breaking config
     buf.gen.yaml                  # codegen plugins (protoc-gen-go, -go-grpc)
     maintainerd/
       auth/
         v1/
           common.proto            # shared: pagination, status enums, error msgs
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

2. The `go_package` / output-path mismatch is fixed. Generated code lives under
   **`internal/platform/gen/go`** (platform infrastructure per
   [code-structure.md](../contributing/code-structure.md) §`internal/platform/*`),
   and proto options, buf output, and imports all align.

3. **buf** is adopted for lint + breaking-change detection + codegen, replacing
   raw `protoc` usage.

These three are tracked as completed Phase-0 items GRPC-001/002/003 below.

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

### 7.2 The prepared control policy (GRPC-015)

A **default "control" policy template** is seeded as a **system policy**
(`Policy.IsSystem=true`, [model_policy.go](../../internal/iam/model_policy.go))
whose statements `allow` the management actions a manager needs over this app. It is
seeded by [012_control_policy.go](../../internal/setup/seeder/012_control_policy.go).
The actions **already exist as permissions** (every API is already permissioned —
§9 reuses `tenant:*`, `service:*`, `client:*`, `user:*`, `security-setting:*`,
`webhook-endpoint:*`, …), so this is a curated statement set, not a new auth model.

Key properties:

- **Seeded but attached to no one.** On a fresh standalone install the control
  policy *exists* but is **unattached** → by default-deny, **nobody controls this
  app**. Independence preserved.
- **Non-deletable template** (it is `IsSystem`), but **control is granted/revoked by
  attaching/detaching it** — the `service_policies` join row is the deletable,
  detachable part. This is exactly *"allowing the core to control this is attachable
  and detachable; other (attachment) records are deletable, the system records are
  not."*
- **Complete management coverage.** The template must stay in sync with the gRPC
  permission registry in [grpc_permissions.go](../../internal/server/grpc_permissions.go)
  and the internal REST management permissions. If a new management/config RPC is
  added, its permission namespace must be seeded and covered by this control policy
  in the same change.
- **Invite stays included.** `InviteService.SendInvite` is intentionally a gRPC
  management RPC because other services may trigger user invitations. It is
  policy-gated by `user:invite`, not treated as public registration or invite
  acceptance.

### 7.3 Registering a controller — core-driven **or** manual

Two paths, identical end-state: a **service principal** (§6) that holds the control
policy.

1. **Via setup (core-driven, gRPC or REST) — GRPC-191.** During setup (alongside
   `CreateTenant` / `CreateAdmin`) the caller passes the controller's **service name
   / identifier** (e.g. `core`). Auth then, in one transaction:
   - creates or reuses a `Service` row for it,
   - links that service to the system tenant,
   - **attaches the default control policy** to it.

   This is the **trust-on-first-use (TOFU)** moment — setup is the *one* place a
   control grant can be minted without already holding one.

   > **Setup is a one-time initialization window.**
   > [service_setup.go](../../internal/setup/service_setup.go) uses the persisted
   > setup-complete lock for this window. Standalone setup runs
   > `create_tenant` (which runs all default tenant seeders) → `create_admin` →
   > `create_profile`, then *optionally* `RegisterControlService`, then
   > **`CompleteSetup`** (`POST /setup/complete`,
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

### 7.6 Persisted setup lock state (GRPC-021/023)

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

Implemented shape: `internal/setup` owns a small `setup_state` table with a stable
bootstrap row and fields such as `is_complete`, `completed_at`, and metadata.
`GetSetupStatus` still reports the derived milestones (`IsTenantSetup`,
`IsAdminSetup`, `IsProfileSetup`), but `IsSetupComplete` reads the persisted lock.

`CompleteSetup` only flips this persisted flag. It does not create tenants, admins,
profiles, services, OAuth clients, or policies. A core-provisioned install may call it
immediately after `RegisterControlService`; a standalone operator calls it when they are
done and want to close the bootstrap registration window.

Decision notes from the review:

| ID | Decision | Status |
|----|----------|--------|
| D1 | Control-policy shape: one shared seeded system policy template attached to each controller vs. per-controller policy created at registration. | **Resolved** — one shared seeded system template (`auth-control`) is attached/detached per controller. |
| D2 | `IsSetupComplete` source of truth. | **Resolved** — use persisted `setup_state`, not derived tenant/admin/profile existence and not a `Tenant`/`Service` field. |
| D3 | `RegisterControlService` shape. | **Resolved** — dedicated setup endpoint/RPC, not an optional field on `CreateTenant` or `CreateAdmin`. |
| D4 | Standalone seeder RPC. | **Resolved** — removed. Seeders run only from the tenant-creation setup path; there is no standalone seeder gRPC contract. |

### 7.7 Provisioning & independence backlog (trackable)

Each item below is independently trackable. `GRPC-015` and `GRPC-191` also appear in
their phase tables (§8/§9); they are repeated here so the whole provisioning model
lives in one place.

| ID       | Status  | Item                                                                                                                                                                                                                                                                                                                                                                                                                                                    | Maps to your point                                                                                                           |
| -------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| GRPC-016 | ✅ done | **Independence guarantee:** a standalone setup seeds the app's own `auth` system service and the unattached control-policy template; no controller is attached unless `RegisterControlService` is explicitly called before `CompleteSetup`. Default-deny remains enforced by the shared PDP/permission registry.                                                                                                                                                                              | "standalone setup creates only one service (its own)"                                                                        |
| GRPC-017 | ✅ done  | **Self-service is non-deletable:** the seeded `auth` service is `IsSystem=true` and update/status/**delete** are already blocked in [service_service.go](../../internal/iam/service_service.go). Tracked as a guard — **do not regress**; add a test asserting it.                                                                                                                                                                                      | "the service representing this app is not deletable"                                                                         |
| GRPC-015 | ✅ done | **Seed the default control policy** (`Policy.IsSystem=true`, *unattached*) carrying all management/config permissions a controller needs, including invite trigger (`user:invite`). Keep it synchronized with the gRPC permission registry and internal REST management permission namespaces.                                                                                                                                                          | "a prepared/default policy for the control service"                                                                          |
| GRPC-191 | ✅ done | **Register a controller at init** (`RegisterControlService`, gRPC + REST): setup creates/fetches the controller service and **attaches the control policy**. TOFU-gated; **runs only during the setup window before the persisted `CompleteSetup` lock is set**.                                                                                                                     | "register the core/control plane during setup → another service + attach policy"                                             |
| GRPC-018 | ✅ done | **Runtime registration path (no core, or after setup is closed):** create a controller `Service` + attach `auth-control` through `AssignServicePolicy` (GRPC-110), authenticated + PDP-gated, **not** via the setup endpoint. This reaches the same end-state as GRPC-191 without reopening bootstrap setup.                                                                                                      | "manually register without the core by applying a policy defining the core service"                                          |
| GRPC-019 | ✅ done | **Un-provision / revoke control:** detach the controller's control-policy attachment through `RemoveServicePolicy`, and/or remove the controller service when allowed. PDP default-deny, webhook push, and short token TTL revoke control without touching the system `auth` service.                                                                                                                                               | "core can provision and unprovision"                                                                                         |
| GRPC-020 | ✅ done | **Multiple instances:** each maintainerd-auth instance seeds its own system service + control-policy template; a controller registers with **each instance independently** (TOFU per instance). Isolation is by per-instance persisted setup state, tenant data, service rows, and policy attachments.                                                                                                                                | "core can provision multiple instances"                                                                                      |
| GRPC-021 | ✅ done | **One-time setup gate parity (REST ↔ gRPC):** REST and gRPC setup reuse the **same persisted setup-complete flag** (set by `CompleteSetup`, GRPC-023); mutating setup operations are unavailable once the flag is set; only `GetSetupStatus` stays available. `IsSetupComplete` is no longer derived from tenant+admin+profile; it is read from setup state. | "all setup endpoints are available only once; same for gRPC setup"                                                           |
| GRPC-022 | ✅ done | **REST setup parity for control registration** — REST and gRPC both expose the same setup-time `RegisterControlService` behavior through the shared setup service method.                                                                                                                                                                                                                                                                | "add a REST equivalent of RegisterControlService (note: for REST)"                                                           |
| GRPC-023 | ✅ done | **`setup/complete` lock** — REST `POST /setup/complete` and gRPC `CompleteSetup`: sets the persisted setup-complete flag that **locks controller registration and all mutating setup ops**. It exists only to close the setup window; it provisions nothing. Anti-infiltration: prevents any other service from registering itself as controller after a standalone setup. Optional `RegisterControlService` lives in the window *before* this call. | "endpoint for setup/complete to lock control-plane registration; register is optional so lock it by flagging setup complete" |
| GRPC-024 | ✅ done | **Control-policy coverage guard:** the seeded control policy grants every current management/config permission namespace used by gRPC and keeps invite gated by `user:invite`; tests assert the policy grants newer namespaces and does not grant public registration/login/reset actions.                                                                                                                                            | "default seeder creates all permissions needed for the control service"                                                       |

---

## 8. Phase 0 — Foundation backlog (must land first)

| ID       | Status | Item                                                                                                                                                                                                                                             | Location                                                             |
| -------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------- |
| GRPC-001 | ✅ done | Adopt **buf**: add `buf.yaml` + `buf.gen.yaml`; replace `make proto` raw-`protoc` with `buf generate`; add `buf lint` + `buf breaking` targets and CI enforcement.                                                                               | `proto/`, `Makefile`, CI                                             |
| GRPC-002 | ✅ done | Restructure proto layout: `v1` becomes a directory, split per-domain files (see §4). Removed the obsolete standalone seeder contract; seeders run from tenant creation only.                                                                 | `proto/maintainerd/auth/v1/`                                         |
| GRPC-003 | ✅ done | Fix `go_package` ↔ output-path ↔ import mismatch; standardize generated code under `internal/platform/gen/go`.                                                                                                                                   | `*.proto`, `Makefile`, [grpc.go](../../internal/server/grpc.go)      |
| GRPC-004 | ✅ done | **Error mapping**: `apperror` → `google.rpc.Code` + `ErrorInfo`/`BadRequest` details helper, used by every handler.                                                                                                                              | `internal/platform/apperror`, new grpc error adapter                 |
| GRPC-005 | ✅ done | Recovery + structured logging interceptors (request_id correlation).                                                                                                                                                                             | [internal/server](../../internal/server)                             |
| GRPC-006 | ✅ done | **Auth interceptor**: extract `authorization` metadata, verify service-account token, denylist check, populate JWT claims context.                                                                                                               | `internal/server`, [jwt](../../internal/platform/jwt)                |
| GRPC-008 | ✅ done | Per-identity rate-limit + request-size + timeout interceptors.                                                                                                                                                                                   | `internal/server`, [middleware](../../internal/platform/middleware)  |
| GRPC-009 | ✅ done | Register `grpc.health.v1.Health` (wire to readiness probes in [health.go](../../internal/server/health.go)).                                                                                                                                     | [internal/server](../../internal/server)                             |
| GRPC-010 | ✅ done | Enable **server reflection** (gated to non-prod or behind authz) for `grpcurl`/control-plane discovery.                                                                                                                                          | [internal/server](../../internal/server)                             |
| GRPC-011 | ✅ done | TLS/mTLS transport config (cert loading, mesh cert verification, fail-closed).                                                                                                                                                                   | `internal/server`, [config](../../internal/platform/config)          |
| GRPC-012 | ✅ done | `common.proto`: shared pagination (`PageRequest`/`PageResponse`), status enums, audit/timestamp fields.                                                                                                                                          | `proto/maintainerd/auth/v1/common.proto`                             |
| GRPC-013 | ✅ done | Base server wiring pattern is established: gRPC server options, interceptor registration, health/reflection registration, and a no-domain-service baseline. Future per-domain service registrations are tracked with each Phase 1 service table. | [internal/server](../../internal/server)                             |
| GRPC-014 | ✅ done | gRPC test harness and conventions are established: `internal/server/grpctest` provides a reusable bufconn harness, and [testing.md](../contributing/testing.md) documents the RPC checklist.                                                     | `internal/server/grpctest`, [testing.md](../contributing/testing.md) |
| GRPC-015 | ✅ done | **Seed the default control policy** (`Policy.IsSystem=true`, *unattached*) — the "manager" template granting management/config actions over this app, including invite trigger. Granted/revoked by attaching/detaching it. See §7.2.             | `internal/setup/seeder/`, [iam](../../internal/iam)                  |
| GRPC-024 | ✅ done | Control-policy coverage guard: seeded `auth-control` action namespaces stay synchronized with gRPC management/config permission strings.                                                                                                           | `internal/setup/seeder/`, [internal/server](../../internal/server)   |

---

## 9. Phase 1 — Control-plane (management) services

These mirror the **management** internal-port routes — the primary S2S targets for
"control everything." Each RPC reuses the existing service layer and the REST
permission string shown. Status is per-RPC.

> **Convention:** REST `GET /xs/` → `ListXs`, `GET /xs/{uuid}` → `GetX`,
> `POST /xs/` → `CreateX`, `PUT /xs/{uuid}` → `UpdateX`, `PUT/PATCH …/status` →
> `SetXStatus`, `DELETE /xs/{uuid}` → `DeleteX`. `RequireStepUp` is noted where the
> REST route requires it (gRPC enforces it as an additional claim/authz check).

### GRPC-101 · TenantService — `tenant.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `GetDefaultTenant` | `GET /tenant/` | service-account (read) | ✅ done |
| `GetTenantByIdentifier` | `GET /tenant/{identifier}` | service-account (read) | ✅ done |
| `ListTenants` | `GET /tenants/` | `tenant:read` | ✅ done |
| `GetTenant` | `GET /tenants/{uuid}` | `tenant:read` | ✅ done |
| `CreateTenant` | `POST /tenants/` | `tenant:create` | ✅ done |
| `UpdateTenant` | `PUT /tenants/{uuid}` | `tenant:update` | ✅ done |
| `SetTenantStatus` | `PUT /tenants/{uuid}/status` | `tenant:update` + step-up | ✅ done |
| `SetTenantPublic` | `PUT /tenants/{uuid}/public` | `tenant:update` + step-up | ✅ done |
| `DeleteTenant` | `DELETE /tenants/{uuid}` | `tenant:delete` + step-up | ✅ done |
| `ListTenantMembers` | `GET /tenants/{uuid}/members/` | `tenant:read` | ✅ done |
| `AddTenantMember` | `POST /tenants/{uuid}/members/` | `tenant:update` | ✅ done |
| `UpdateTenantMemberRole` | `PATCH /tenants/{uuid}/members/{m}/role` | `tenant:update` | ✅ done |
| `RemoveTenantMember` | `DELETE /tenants/{uuid}/members/{m}` | `tenant:update` | ✅ done |

### GRPC-102 · TenantSettingService — `tenant.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `GetRateLimitConfig` / `UpdateRateLimitConfig` | `GET/PUT /tenant-settings/rate-limit` | `tenant-setting:read` / `:update` | ✅ done |
| `GetAuditConfig` / `UpdateAuditConfig` | `GET/PUT /tenant-settings/audit` | `tenant-setting:read` / `:update` | ✅ done |
| `GetMaintenanceConfig` / `UpdateMaintenanceConfig` | `GET/PUT /tenant-settings/maintenance` | `tenant-setting:read` / `:update` | ✅ done |
| `GetFeatureFlags` / `UpdateFeatureFlags` | `GET/PUT /tenant-settings/feature-flags` | `tenant-setting:read` / `:update` | ✅ done |

### GRPC-110 · ServiceService — `iam.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `GetMyPolicyBundle` | `GET /services/me/policy-bundle` | service-account (self) | 🔴 todo |
| `ListServices` | `GET /services/` | `service:read` | ✅ done |
| `GetService` | `GET /services/{uuid}` | `service:read` | ✅ done |
| `CreateService` | `POST /services/` | `service:create` | ✅ done |
| `UpdateService` | `PUT /services/{uuid}` | `service:update` | ✅ done |
| `SetServiceStatus` | `PUT /services/{uuid}/status` | `service:update` | ✅ done |
| `DeleteService` | `DELETE /services/{uuid}` | `service:delete` | ✅ done |
| `AssignServicePolicy` | `POST /services/{uuid}/policies/{p}` | `service:policy:assign` | ✅ done |
| `RemoveServicePolicy` | `DELETE /services/{uuid}/policies/{p}` | `service:policy:remove` | ✅ done |

### GRPC-111 · APIService — `iam.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListAPIs` / `GetAPI` | `GET /apis/`, `GET /apis/{uuid}` | `api:read` | ✅ done |
| `CreateAPI` | `POST /apis/` | `api:create` | ✅ done |
| `UpdateAPI` | `PUT /apis/{uuid}` | `api:update` | ✅ done |
| `SetAPIStatus` | `PUT /apis/{uuid}/status` | `api:update` | ✅ done |
| `DeleteAPI` | `DELETE /apis/{uuid}` | `api:delete` | ✅ done |

### GRPC-112 · PermissionService — `iam.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListPermissions` / `GetPermission` | `GET /permissions/`, `…/{uuid}` | `permission:read` | ✅ done |
| `CreatePermission` | `POST /permissions/` | `permission:create` | ✅ done |
| `UpdatePermission` | `PUT /permissions/{uuid}` | `permission:update` | ✅ done |
| `SetPermissionStatus` | `PUT /permissions/{uuid}/status` | `permission:update` | ✅ done |
| `DeletePermission` | `DELETE /permissions/{uuid}` | `permission:delete` | ✅ done |

### GRPC-113 · PolicyService — `iam.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListPolicies` / `GetPolicy` | `GET /policies/`, `…/{uuid}` | `policy:read` | ✅ done |
| `ListPolicyServices` | `GET /policies/{uuid}/services` | `policy:read` | ✅ done |
| `CreatePolicy` | `POST /policies/` | `policy:create` | ✅ done |
| `UpdatePolicy` | `PUT /policies/{uuid}` | `policy:update` | ✅ done |
| `SetPolicyStatus` | `PUT /policies/{uuid}/status` | `policy:update` | ✅ done |
| `DeletePolicy` | `DELETE /policies/{uuid}` | `policy:delete` | ✅ done |

### GRPC-114 · RoleService — `iam.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListRoles` / `GetRole` | `GET /roles/`, `…/{uuid}` | `role:read` | ✅ done |
| `CreateRole` | `POST /roles/` | `role:create` | ✅ done |
| `UpdateRole` | `PUT /roles/{uuid}` | `role:update` | ✅ done |
| `SetRoleStatus` | `PUT /roles/{uuid}/status` | `role:update` | ✅ done |
| `DeleteRole` | `DELETE /roles/{uuid}` | `role:delete` | ✅ done |
| `ListRolePermissions` | `GET /roles/{uuid}/permissions` | `role:read` | ✅ done |
| `AddRolePermissions` | `POST /roles/{uuid}/permissions` | `role:permission:create` | ✅ done |
| `RemoveRolePermission` | `DELETE /roles/{uuid}/permissions/{p}` | `role:permission:delete` | ✅ done |

### GRPC-115 · AuthorizationService — `iam.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `Authorize` | `POST /authorize/` | service-account | ✅ done |

> This is the **S2S decision RPC** (PDP). High value: peers can ask "can principal X
> do action Y on resource Z?" over gRPC instead of embedding the SDK.

### GRPC-120 · IdentityProviderService — `identity_provider.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListIdentityProviders` / `GetIdentityProvider` | `GET /identity_providers/`, `…/{uuid}` | `idp:read` | ✅ done |
| `CreateIdentityProvider` | `POST /identity_providers/` | `idp:create` | ✅ done |
| `UpdateIdentityProvider` | `PUT /identity_providers/{uuid}` | `idp:update` | ✅ done |
| `SetIdentityProviderStatus` | `PUT /identity_providers/{uuid}/status` | `idp:update` | ✅ done |
| `DeleteIdentityProvider` | `DELETE /identity_providers/{uuid}` | `idp:delete` | ✅ done |

### GRPC-121 · SignupFlowService — `identity_provider.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListSignupFlows` / `GetSignupFlow` | `GET /signup_flows/`, `…/{uuid}` | `signup-flow:read` | ✅ done |
| `CreateSignupFlow` | `POST /signup_flows/` | `signup-flow:create` | ✅ done |
| `UpdateSignupFlow` | `PUT /signup_flows/{uuid}` | `signup-flow:update` | ✅ done |
| `SetSignupFlowStatus` | `PATCH /signup_flows/{uuid}/status` | `signup-flow:update` | ✅ done |
| `DeleteSignupFlow` | `DELETE /signup_flows/{uuid}` | `signup-flow:delete` | ✅ done |
| `AssignSignupFlowRoles` | `POST /signup_flows/{uuid}/roles/` | `signup-flow:update` | ✅ done |
| `ListSignupFlowRoles` | `GET /signup_flows/{uuid}/roles/` | `signup-flow:read` | ✅ done |
| `RemoveSignupFlowRole` | `DELETE /signup_flows/{uuid}/roles/{r}` | `signup-flow:update` | ✅ done |

### GRPC-130 · ClientService — `client.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListClients` / `GetClient` | `GET /clients/`, `…/{uuid}` | `client:read` | ✅ done |
| `GetClientSecret` | `GET /clients/{uuid}/secret` | `client:secret:read` + step-up | ✅ done |
| `RotateClientSecret` | `POST /clients/{uuid}/rotate-secret` | `client:secret:rotate` + step-up | ✅ done |
| `GetClientConfig` | `GET /clients/{uuid}/config` | `client:config:read` | ✅ done |
| `CreateClient` | `POST /clients/` | `client:create` | ✅ done |
| `UpdateClient` | `PUT /clients/{uuid}` | `client:update` | ✅ done |
| `SetClientStatus` | `PUT /clients/{uuid}/status` | `client:update` | ✅ done |
| `DeleteClient` | `DELETE /clients/{uuid}` | `client:delete` | ✅ done |
| `ListClientURIs` | `GET /clients/{uuid}/uris` | `client:uri:read` | ✅ done |
| `CreateClientURI` | `POST /clients/{uuid}/uris` | `client:uri:create` | ✅ done |
| `UpdateClientURI` | `PUT /clients/{uuid}/uris/{u}` | `client:uri:update` | ✅ done |
| `DeleteClientURI` | `DELETE /clients/{uuid}/uris/{u}` | `client:uri:delete` | ✅ done |
| `ListClientAPIs` | `GET /clients/{uuid}/apis` | `client:api:read` | ✅ done |
| `AddClientAPIs` | `POST /clients/{uuid}/apis` | `client:api:create` | ✅ done |
| `RemoveClientAPI` | `DELETE /clients/{uuid}/apis/{a}` | `client:api:delete` | ✅ done |
| `ListClientAPIPermissions` | `GET /clients/{uuid}/apis/{a}/permissions` | `client:api:permission:read` | ✅ done |
| `AddClientAPIPermissions` | `POST /clients/{uuid}/apis/{a}/permissions` | `client:api:permission:create` | ✅ done |
| `RemoveClientAPIPermission` | `DELETE /clients/{uuid}/apis/{a}/permissions/{p}` | `client:api:permission:delete` | ✅ done |

### GRPC-131 · APIKeyService — `client.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListAPIKeys` / `GetAPIKey` | `GET /api_keys/`, `…/{uuid}` | `api_key:read` | ✅ done |
| `GetAPIKeyConfig` | `GET /api_keys/{uuid}/config` | `api_key:read` | ✅ done |
| `CreateAPIKey` | `POST /api_keys/` | `api_key:create` + step-up | ✅ done |
| `UpdateAPIKey` | `PUT /api_keys/{uuid}` | `api_key:update` + step-up | ✅ done |
| `SetAPIKeyStatus` | `PUT /api_keys/{uuid}/status` | `api_key:update` + step-up | ✅ done |
| `DeleteAPIKey` | `DELETE /api_keys/{uuid}` | `api_key:delete` + step-up | ✅ done |
| `ListAPIKeyAPIs` | `GET /api_keys/{uuid}/apis/` | `api_key:read` | ✅ done |
| `AddAPIKeyAPIs` | `POST /api_keys/{uuid}/apis/` | `api_key:update` + step-up | ✅ done |
| `RemoveAPIKeyAPI` | `DELETE /api_keys/{uuid}/apis/{a}` | `api_key:update` + step-up | ✅ done |
| `ListAPIKeyAPIPermissions` | `GET /api_keys/{uuid}/apis/{a}/permissions/` | `api_key:read` | ✅ done |
| `AddAPIKeyAPIPermissions` | `POST /api_keys/{uuid}/apis/{a}/permissions/` | `api_key:update` + step-up | ✅ done |
| `RemoveAPIKeyAPIPermission` | `DELETE /api_keys/{uuid}/apis/{a}/permissions/{p}` | `api_key:update` + step-up | ✅ done |

### GRPC-140 · UserService (admin) — `user.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListUsers` / `GetUser` | `GET /users/`, `…/{uuid}` | `user:read` | ✅ done |
| `CreateUser` | `POST /users/` | `user:create` | ✅ done |
| `UpdateUser` | `PUT /users/{uuid}` | `user:update` | ✅ done |
| `SetUserStatus` | `PATCH /users/{uuid}/status` | `user:update` + step-up | ✅ done |
| `VerifyUserEmail` | `PATCH /users/{uuid}/verify-email` | `user:update` | ✅ done |
| `VerifyUserPhone` | `PATCH /users/{uuid}/verify-phone` | `user:update` | ✅ done |
| `CompleteUserAccount` | `PATCH /users/{uuid}/complete-account` | `user:update` | ✅ done |
| `DeleteUser` | `DELETE /users/{uuid}` | `user:delete` + step-up | ✅ done |
| `ForceUserPasswordChange` | `PUT /users/{uuid}/force-password-change` | `user:update` + step-up | ✅ done |
| `ListUserRoles` | `GET /users/{uuid}/roles` | `user:read` | ✅ done |
| `ListUserIdentities` | `GET /users/{uuid}/identities` | `user:read` | ✅ done |
| `AssignUserRoles` | `POST /users/{uuid}/roles` | `user:create` + step-up | ✅ done |
| `RemoveUserRole` | `DELETE /users/{uuid}/roles/{r}` | `user:create` + step-up | ✅ done |

### GRPC-141 · InviteService — `user.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `SendInvite` | `POST /invite/` | `user:invite` | ✅ done |

### GRPC-150 · SecuritySettingService — `security.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `GetMFAConfig` / `UpdateMFAConfig` | `GET/PUT /security-settings/mfa` | `security-setting:read` / `:update` (+step-up) | ✅ done |
| `GetPasswordConfig` / `UpdatePasswordConfig` | `…/password` | `security-setting:read` / `:update` (+step-up) | ✅ done |
| `GetSessionConfig` / `UpdateSessionConfig` | `…/session` | `security-setting:read` / `:update` (+step-up) | ✅ done |
| `GetThreatConfig` / `UpdateThreatConfig` | `…/threat` | `security-setting:read` / `:update` (+step-up) | ✅ done |
| `GetLockoutConfig` / `UpdateLockoutConfig` | `…/lockout` | `security-setting:read` / `:update` (+step-up) | ✅ done |
| `GetRegistrationConfig` / `UpdateRegistrationConfig` | `…/registration` | `security-setting:read` / `:update` (+step-up) | ✅ done |
| `GetTokenConfig` / `UpdateTokenConfig` | `…/token` | `security-setting:read` / `:update` (+step-up) | ✅ done |

### GRPC-151 · IPRestrictionRuleService — `security.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListIPRestrictionRules` / `GetIPRestrictionRule` | `GET /ip-restriction-rules/`, `…/{uuid}` | `ip-restriction-rule:read` | ✅ done |
| `CreateIPRestrictionRule` | `POST /ip-restriction-rules/` | `ip-restriction-rule:create` | ✅ done |
| `UpdateIPRestrictionRule` | `PUT /ip-restriction-rules/{uuid}` | `ip-restriction-rule:update` | ✅ done |
| `DeleteIPRestrictionRule` | `DELETE /ip-restriction-rules/{uuid}` | `ip-restriction-rule:delete` | ✅ done |
| `SetIPRestrictionRuleStatus` | `PATCH /ip-restriction-rules/{uuid}/status` | `ip-restriction-rule:update` | ✅ done |

### GRPC-160 · BrandingService — `branding.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `GetBranding` / `UpdateBranding` | `GET/PUT /branding/` | `branding:read` / `:update` | ✅ done |

### GRPC-161 · EmailTemplateService — `branding.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListEmailTemplates` / `GetEmailTemplate` | `GET /email_templates/`, `…/{uuid}` | `email-template:read` | ✅ done |
| `CreateEmailTemplate` | `POST /email_templates/` | `email-template:create` | ✅ done |
| `UpdateEmailTemplate` | `PUT /email_templates/{uuid}` | `email-template:update` | ✅ done |
| `DeleteEmailTemplate` | `DELETE /email_templates/{uuid}` | `email-template:delete` | ✅ done |
| `SetEmailTemplateStatus` | `PATCH /email_templates/{uuid}/status` | `email-template:update` | ✅ done |

### GRPC-162 · SMSTemplateService — `branding.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListSMSTemplates` / `GetSMSTemplate` | `GET /sms_templates/`, `…/{uuid}` | `sms-template:read` | ✅ done |
| `CreateSMSTemplate` | `POST /sms_templates/` | `sms-template:create` | ✅ done |
| `UpdateSMSTemplate` | `PUT /sms_templates/{uuid}` | `sms-template:update` | ✅ done |
| `DeleteSMSTemplate` | `DELETE /sms_templates/{uuid}` | `sms-template:delete` | ✅ done |
| `SetSMSTemplateStatus` | `PATCH /sms_templates/{uuid}/status` | `sms-template:update` | ✅ done |

### GRPC-163 · LoginTemplateService — `branding.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListLoginTemplates` / `GetLoginTemplate` | `GET /login_templates/`, `…/{uuid}` | `login-template:read` | ✅ done |
| `CreateLoginTemplate` | `POST /login_templates/` | `login-template:create` | ✅ done |
| `UpdateLoginTemplate` | `PUT /login_templates/{uuid}` | `login-template:update` | ✅ done |
| `DeleteLoginTemplate` | `DELETE /login_templates/{uuid}` | `login-template:delete` | ✅ done |
| `SetLoginTemplateStatus` | `PATCH /login_templates/{uuid}/status` | `login-template:update` | ✅ done |

### GRPC-170 · EmailConfigService / SMSConfigService — `notifier.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `GetEmailConfig` / `UpdateEmailConfig` | `GET/PUT /email-config/` | `email-config:read` / `:update` | ✅ done |
| `GetSMSConfig` / `UpdateSMSConfig` | `GET/PUT /sms-config/` | `sms-config:read` / `:update` | ✅ done |

### GRPC-171 · WebhookEndpointService — `webhook.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListWebhookEndpoints` / `GetWebhookEndpoint` | `GET /webhook-endpoints/`, `…/{uuid}` | `webhook-endpoint:read` | ✅ done |
| `CreateWebhookEndpoint` | `POST /webhook-endpoints/` | `webhook-endpoint:create` | ✅ done |
| `UpdateWebhookEndpoint` | `PUT /webhook-endpoints/{uuid}` | `webhook-endpoint:update` | ✅ done |
| `DeleteWebhookEndpoint` | `DELETE /webhook-endpoints/{uuid}` | `webhook-endpoint:delete` | ✅ done |
| `SetWebhookEndpointStatus` | `PATCH /webhook-endpoints/{uuid}/status` | `webhook-endpoint:update` | ✅ done |

### GRPC-172 · AuthEventService — `authevent.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `ListAuthEvents` | `GET /auth-events/` | `auth_event:read` | ✅ done |
| `CountAuthEventsByType` | `GET /auth-events/count` | `auth_event:read` | ✅ done |
| `GetAuthEvent` | `GET /auth-events/{uuid}` | `auth_event:read` | ✅ done |

### GRPC-180 · OAuthIntrospectionService — `oauth.proto`
| RPC | REST origin | Permission | Status |
|-----|-------------|-----------|--------|
| `Introspect` | `POST /oauth/introspect` | service-account | ✅ done |

### GRPC-190 · SetupService — `setup.proto`
| RPC                                 | REST origin                                                        | Permission                      | Status            |
| ----------------------------------- | ------------------------------------------------------------------ | ------------------------------- | ----------------- |
| `GetSetupStatus`                    | `GET /setup/status`                                                | (bootstrap)                     | ✅ done           |
| `CreateTenant`                      | `POST /setup/create_tenant`                                        | (bootstrap)                     | ✅ done           |
| `CreateAdmin`                       | `POST /setup/create_admin`                                         | (bootstrap)                     | ✅ done           |
| `CreateProfile`                     | `POST /setup/create_profile`                                       | (bootstrap)                     | ✅ done           |
| `RegisterControlService` (GRPC-191) | `POST /setup/register-control-service`                             | (bootstrap, TOFU, **optional**) | ✅ done           |
| `CompleteSetup` (GRPC-023)          | `POST /setup/complete`                                             | (bootstrap, final)              | ✅ done           |

> **SetupService** is the control plane's natural provisioning entry point
> (a fresh tenant + admin + profile, with tenant seeders run during
> `CreateTenant`) — high S2S value despite living under the "bootstrap" auth
> flows. Bootstrap auth differs (no policy yet exists); guard with the existing
> setup gate, not the PDP.
>
> **One-time, then disabled — REST and gRPC share one gate (GRPC-021).** Both
> transports use one persisted setup-complete flag from
> [service_setup.go](../../internal/setup/service_setup.go). Every mutating setup
> operation (`CreateTenant`, `CreateAdmin`, `CreateProfile`,
> `RegisterControlService`) becomes unavailable once the persisted lock is set.
> `GetSetupStatus` is the **only** setup operation that stays available afterward
> (read-only probe).
>
> **`RegisterControlService` (GRPC-191)** implements §7.3 path 1: the caller passes a
> controller **service name/identifier**; auth creates or reuses the `Service` and
> **attaches the seeded control policy** (GRPC-015) — the trust-on-first-use grant
> that lets the core control this auth instance. It is **optional**. The REST
> endpoint (`POST /setup/register-control-service` — GRPC-022) and the gRPC RPC are
> implemented through the same setup service method.
>
> **Setup ordering & the explicit lock (`CompleteSetup` / GRPC-023).** A standalone
> install runs `create_tenant` (tenant row + all default tenant seeders) →
> `create_admin` → `create_profile` and then, optionally,
> `RegisterControlService`. Because the controller step is optional, completion
> **must be flagged explicitly** by **`CompleteSetup`** (`POST /setup/complete` or
> the gRPC RPC) —
> not merely inferred from "profile exists." `CompleteSetup` sets the persisted
> setup-complete flag, which **locks all mutating setup operations, including
> `RegisterControlService`**. It provisions nothing; it only closes the bootstrap
> window. This is the **anti-infiltration control**: once a standalone operator
> completes setup, no other service can sneak in and register itself as the
> controller through the setup path. After the lock, controller changes happen only
> via the runtime path (`AssignServicePolicy`, GRPC-018 — authenticated +
> PDP-gated). `GetSetupStatus` remains available; everything else is closed.

---

## 9.1 Backlog completeness audit

The current gRPC backlog intentionally includes every private management/config
surface that another service should use to control this app:

- Setup/provisioning: `SetupService`, including optional
  `RegisterControlService` before `CompleteSetup`.
- Tenant and IAM management: tenants, tenant settings, services, APIs,
  permissions, policies, roles, and `Authorize`.
- External-app configuration: identity providers, signup flows, clients, API keys,
  security settings, IP restrictions, branding, email/SMS/login templates,
  email/SMS delivery config, webhook endpoints, auth event reads, and OAuth token
  introspection.
- User administration: admin user CRUD/status/verification/role operations and
  `InviteService.SendInvite`.

The current remaining backlog is intentionally narrow:

| ID | Status | Item | Why it remains |
|----|--------|------|----------------|
| GRPC-110 | 🔴 todo | `ServiceService.GetMyPolicyBundle` | REST already serves `GET /services/me/policy-bundle`; the gRPC equivalent is still needed for peers that want bundle distribution over gRPC instead of REST. |

Everything else private and management-oriented is either implemented in Phase 1
or explicitly excluded in §10. End-user flows are not missing backlog items; they
are REST-only by design.

---

## 10. REST-only / not a gRPC backlog

The gRPC surface is **not** a second copy of every private REST route. It is for
service-to-service setup, configuration, management, and verification/decision
reads. End-user ceremonies remain REST because they are driven by browsers,
mobile apps, redirects, cookies, recovery tokens, and UI state.

Do **not** add gRPC checklist items or proto services for these REST flows:

| REST area | REST origin examples | Why it stays REST |
|-----------|----------------------|-------------------|
| Registration and invite acceptance | `POST /register`, `POST /register/invite` | End-user onboarding and frontend form flow. Core can manage users through `UserService` and can trigger invites through `InviteService.SendInvite`; it should not run self-registration or invite-acceptance ceremonies. |
| Login/logout/session ceremonies | `POST /login`, `POST /logout`, `/account/sessions` | Credential, cookie, token, and device/session UX flow. Peer services should use token introspection / authorization checks instead. |
| Password and account recovery | `POST /forgot-password`, `POST /reset-password`, `/recovery/backup-code` | Token/email/SMS-driven recovery flow intended for users. |
| Email verification and magic links | `/email-verification/*`, `/magic-link/*` | Public/user token ceremony bound to email links and frontend redirects. |
| SMS login OTP | `/sms-login/*` | Public/user OTP ceremony. |
| Self profile/account/settings | `/profile/*`, `/profiles/*`, `/account/*`, `/user-settings/*` | Authenticated user self-service. Core/admin management belongs in Phase 1 `UserService`. |
| User MFA enrollment/authentication | `/mfa/totp/*`, `/mfa/webauthn/*`, `/mfa/step-up/*` | User challenge ceremony. Only admin/configuration MFA surfaces should be considered for gRPC. |
| Federation callback/exchange | `/federation/token`, `/federation/oauth2/callback`, `/federation/hrd`, `/account/identities/*` | External IdP redirects/tokens and authenticated user account-linking ceremonies. IdP configuration belongs in Phase 1. |

Allowed exceptions must fit one of these buckets:

- **Management/configuration:** for example admin reset of another user's MFA,
  IdP configuration, signup-flow configuration, client/API-key management, and
  security/branding/notifier settings. Invite sending is included here because it
  is a service-triggered management action.
- **Verification/decision reads:** for example token introspection, `Authorize`,
  policy-bundle distribution, or narrowly scoped identity/permission reads needed
  by a peer service to serve its own request.

Any exception should be promoted into Phase 1 with a concrete S2S consumer and
permission string before implementation.

---

## 11. Deferred to a later version (v2 / future)

- **gRPC-Web / public gateway** — if any browser or third-party needs gRPC; would
  require its own public-port surface and rate-limit posture. End-user auth flows
  remain REST even if this exists later.
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
- Keep modified packages at 95-100% coverage for touched files, and do not finish
  below 95% package coverage for gRPC implementation work unless the gap is
  explicitly documented and approved.

---

## 13. Milestone summary

| Milestone | Items | Outcome |
|-----------|-------|---------|
| **M0 — Foundation** | GRPC-001…017, GRPC-024 | buf layout, codegen, auth/authz/observability interceptors, health, reflection, TLS, error mapping, **default control policy seed** + **independence guarantees** (GRPC-015/016/017), coverage guard, test harness. **Blocks everything.** |
| **M1 — IAM + Tenant core + controller lifecycle** | GRPC-101/102/110–115, GRPC-190/191, GRPC-018…023 | The control plane can register itself at init (TOFU, lockable via explicit `CompleteSetup` — GRPC-021/022/023), be un-registered, and run multiple instances; then manage tenants, services, policies, roles, and ask `Authorize`. Highest S2S value. |
| **M2 — Clients, Users, IdP** | GRPC-120/121/130/131/140/141 | Management/provisioning surface only: clients, API keys, admin user management, identity-provider configuration, signup-flow configuration, and invites. |
| **M3 — Settings, Branding, Notifier, Webhooks, Events, OAuth, Setup** | GRPC-150…190 | Remaining management surface + introspection + provisioning. |
