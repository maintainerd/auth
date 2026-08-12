# Setup & Bootstrap

> First-run provisioning of a maintainerd-auth instance — the system tenant, super-admin, and (for orchestrated installs) the control-plane principals — via either an unauthenticated REST setup wizard or a token-gated gRPC `SetupService`.

| | |
|---|---|
| **Status** | Implemented |
| **Code** | `internal/setup` (service, REST + gRPC handlers, validation, bootstrap gating), `internal/setup/seeder` (baseline seed), `internal/server` (gRPC bootstrap auth, route mount) |
| **Endpoints** | REST (internal port `:8080`): `GET/POST /api/v1/setup/*`. gRPC (`:50051`): `maintainerd.auth.SetupService` (10 RPCs) |
| **Storage** | No dedicated setup table — completion is derived from `tenants` (system tenant `status`), `users` (super-admin), `profiles`, plus the IAM/client rows written by the seeder and the `Ensure*` RPCs. All migrations are create-only (`internal/platform/runner/migration.go`). |
| **Config** | `SETUP_BOOTSTRAP_TOKEN`, `SETUP_WINDOW_TTL` (default `30m`), `CONTROL_PLANE_ENABLED` |

## Overview

A fresh instance has no tenant, no users, and no service principals, so the first-run provisioning surface cannot authenticate with the normal JWT/PDP path. Setup exists to close that gap once, then lock itself.

There are **two mutually-exclusive bootstrap paths**, selected by whether the instance is orchestrated:

| Path | Who runs it | Transport | Gate |
|------|-------------|-----------|------|
| **REST setup wizard** | A human operator on a standalone / self-hosted install | Internal REST API on `:8080`, `/api/v1/setup/*`, **unauthenticated** | Closes the moment the system tenant is `active` |
| **gRPC `SetupService`** | An orchestrator (core, via maintainerd-docker) | gRPC on `:50051` | Pre-shared `SETUP_BOOTSTRAP_TOKEN` + mTLS, plus a time-bounded window |

The distinguishing fact is `CONTROL_PLANE_ENABLED` (`bootstrapControlPlaneEnabled`, `internal/setup/bootstrap_config.go:19`): control-plane ON ⇒ orchestrated (gRPC listener up, mTLS forced). Both paths converge on the same `setup.SetupService` layer, so the two transports cannot disagree about what provisioning means.

## How it works

### Common gate — `ensureSetupOpen`

Every mutating setup call passes through `ensureSetupOpen` (`internal/setup/service_setup.go:742`). It closes on two independent, one-way conditions:

1. **Setup finished.** An **active** system tenant is the durable, replica-shared record that this instance has been bootstrapped (`service_setup.go:747`). There is no separate "setup closed" flag — the single-system-tenant constraint settles the race when two replicas hit a fresh instance together, and a second boolean could only drift.
2. **Window expired** (orchestrated only). See `ensureSetupWindowOpen` (`service_setup.go:765`): if `CONTROL_PLANE_ENABLED` and `time.Since(setupProcessStart) > SETUP_WINDOW_TTL`, the call fails closed with 403. `setupProcessStart` is anchored at **process start** (`service_setup.go:759`), not first request, so an attacker who reaches the instance first cannot restart the clock. The **standalone REST wizard is exempt** from the TTL (an interrupted operator must not find a silently self-locked instance).

### Path A — REST setup wizard (standalone)

Routes are mounted on the **internal** router (`internal/server/router.go:63`, `buildInternalRouter` → `:8080`) under `/api/v1/setup`, with stricter middleware than the global API: 1 MB request cap and a 30 s timeout (`internal/setup/routes.go:14`). No authentication is required.

1. `GET /api/v1/setup/status` — always available; returns `is_tenant_setup`, `is_admin_setup`, `is_profile_setup`, `is_setup_complete` (`GetSetupStatus`, `service_setup.go:114`). `is_setup_complete` is computed as `system tenant.Status == "active"` (`service_setup.go:137`).
2. `POST /api/v1/setup/create_tenant` — one-time. Creates the system tenant (`IsSystem=true`, `Status="pending"`), then in the same transaction runs the full baseline seeder for it (`setupRunSeeders(tx, "v0.1.0")`, `service_setup.go:213`). Refuses if any tenant already exists (`service_setup.go:176`). The tenant name must be a DNS-safe slug (`validation_setup.go:71`).
3. `POST /api/v1/setup/create_admin` — one-time, requires the tenant. Creates the super-admin user + identity, assigns the seeded `registered` and `super-admin` roles, adds a `tenant_members` `owner` row, and **flips the system tenant `Status` → `active`** (`service_setup.go:439`). The password is validated against the shipped default policy (`service_setup.go:331`). Because this call activates the tenant, setup is effectively complete once the admin exists.
4. `POST /api/v1/setup/create_profile` — **optional**, idempotent (`CreateProfile`, `service_setup.go:599`). The admin's profile is normally collected on first sign-in through the identity app; this endpoint only exists for an unattended bootstrap that wants to seed it early.
5. `POST /api/v1/setup/complete` — explicit lock (`CompleteSetup`, `service_setup.go:677`). Idempotent; requires `IsTenantSetup && IsAdminSetup` (**not** profile — see the drift note below) and ensures the system tenant is `active`.

**Orchestrated instances refuse the REST wizard.** Every wizard endpoint except `status` calls `refuseWhenOrchestrated` (`internal/setup/handler_setup.go:31`): if `CONTROL_PLANE_ENABLED` **and** a bootstrap credential is configured, it returns 403 and directs the caller to the gRPC surface. Both conditions are required — with the control plane off there is no gRPC listener, so closing REST too would leave an instance unable to bootstrap at all.

### Path B — gRPC `SetupService` (orchestrated)

The 10 bootstrap RPCs are listed in `grpcBootstrapMethods` (`internal/server/grpc_permissions.go:180`): `GetSetupStatus`, `CreateTenant`, `CreateAdmin`, `CreateProfile`, `RegisterControlService`, `EnsureControlClient`, `EnsureResourceAPI`, `EnsureRole`, `EnsureConsoleClient`, `CompleteSetup`. They bypass the JWT/PDP path (no principals exist yet) and are gated by `authorizeSetupBootstrap` (`internal/server/grpc_interceptors.go:440`):

1. If `SETUP_BOOTSTRAP_TOKEN` is empty → `PermissionDenied` "gRPC setup is disabled".
2. Read the token from the `x-setup-token` metadata header (`grpc_interceptors.go:42`).
3. Rate-limit (`setup-bootstrap` bucket) before the credential is examined, so guessing is throttled.
4. **Constant-time compare** against `config.SetupBootstrapToken` (`grpc_interceptors.go:456`). There is **no separate "already spent" ledger** — single-use is enforced entirely by `ensureSetupOpen` refusing every mutating call once the system tenant is active, plus the window TTL. When the control plane is on, R2 makes mTLS mandatory, so the caller has already presented a client cert signed by this deployment's CA before reaching here.

A typical orchestrated sequence (each `Ensure*` is get-or-create, so the whole run is replay-safe):

1. `CreateTenant` → system tenant + baseline seed.
2. `CreateAdmin` → super-admin (activates the tenant).
3. `RegisterControlService` → registers the orchestrator's control service and **builds** its control policy from the request (`service_setup.go:522`; see [control policy](#control-policy)).
4. `EnsureResourceAPI` → the orchestrator's own API + permissions (must exist before tokens/roles reference them).
5. `EnsureControlClient` → the orchestrator's M2M client (`private_key_jwt`; the named service must already exist).
6. `EnsureRole` → a role carrying named (pre-existing) permissions, optionally assigned to a user.
7. `EnsureConsoleClient` → the orchestrator operators' browser SPA (public, PKCE).
8. `CompleteSetup` → formal lock.

### Seeding

`create_tenant` runs `runner.RunSeeders` → `seeder.RunAll` (`internal/setup/seeder/run.go:17`), which seeds the global integration event-type catalog once, then calls `SeedTenant` for the system tenant. `SeedTenant` (`internal/setup/seeder/seed_tenant.go:23`) is the single source of truth for a tenant's baseline and is reused verbatim by admin-side tenant creation: it seeds the tenant-scoped `auth` service, API + permissions, identity provider, clients (`auth-console`, `auth-identity`) + URIs, `registered` + `super-admin` roles, role-permission grants, registration flows, email/SMS templates, security settings, and branding. The orchestrator control policy is **not** seeded (see below).

## Implementation

| Concern | Location |
|---------|----------|
| Service interface + impl | `internal/setup/service_setup.go:29` (`SetupService`), `service_setup.go:42` (`setupService`) |
| Orchestrator `Ensure*` RPCs | `internal/setup/service_setup_provision.go` |
| REST handlers + `refuseWhenOrchestrated` | `internal/setup/handler_setup.go` |
| gRPC handlers | `internal/setup/handler_setup_grpc.go` |
| Route mount (internal `:8080`) | `internal/server/router.go:63`; `internal/setup/routes.go:10` |
| gRPC registration | `internal/server/grpc.go:184` (`RegisterSetupServiceServer`) |
| gRPC bootstrap auth | `internal/server/grpc_interceptors.go:440`; method set `grpc_permissions.go:180` |
| DTO validation | `internal/setup/validation_setup.go` |
| Bootstrap mode flags | `internal/setup/bootstrap_config.go` |
| Baseline seed | `internal/setup/seeder/seed_tenant.go`, `seeder/run.go` |
| Control-plane deps wiring | `internal/app/services.go:328` (`ControlRegistrationDeps`) |

**Dependency split.** The REST wizard needs only the core repos; the `Ensure*`/`RegisterControlService` RPCs need extra IAM/client repos, grouped in `ControlRegistrationDeps` (`service_setup.go:65`) and passed as a variadic option (`service_setup.go:85`). `requireProvisioningDeps` (`service_setup_provision.go:665`) fails closed if a provisioning RPC is called without them wired.

<a name="control-policy"></a>**Control policy (built, not seeded).** `RegisterControlService` constructs the orchestrator's control policy in the same request that registers the principal receiving it (`ensureControlPolicy`, `service_setup_provision.go:690`). An existing policy is returned **unchanged**, never widened — so re-running registration cannot escalate an already-registered principal. When no actions are requested, it falls back to `seeder.DefaultControlActions` (`internal/setup/seeder/013_control_actions.go:34`): `tenant:*`, `tenant-setting:*`, `service:*`, `api:*`, `permission:*`, `policy:*`, `role:*`, `client:*`, `workload-identity-federation:*`. A bare `*` / `*:*` is stripped (`normalizeControlActions`, `service_setup_provision.go:736`). Deliberately excluded: `user:*`, `account:*:self`, `security-setting:*`, `ip-restriction-rule:*`, `audit:read`, `auth_event:*`.

### Drift note — `CompleteSetup` and profile

`internal/setup/routes.go:34` comments that `create_profile` is "required before `/setup/complete`, which gates on `IsProfileSetup`." **The service does not gate on profile.** `CompleteSetup` requires only `IsTenantSetup && IsAdminSetup` (`service_setup.go:698`) and the comment there explains why: gating the lock on the profile once left setup unable to finish, because the tenant stayed `pending` and the tenant-status middleware then refused the very login that would have created the profile. Treat the profile as optional; the route comment is stale.

## Configuration

| Env var | Config field | Effect |
|---------|--------------|--------|
| `CONTROL_PLANE_ENABLED` | `config.ControlPlaneEnabled` (`config.go:170`) | `true` ⇒ orchestrated: enables the gRPC listener, forces gRPC mTLS, applies the setup-window TTL, and closes the REST wizard once a credential is set. |
| `SETUP_BOOTSTRAP_TOKEN` | `config.SetupBootstrapToken` (`config.go:43`, loaded `config.go:205`) | Per-instance pre-shared credential gating the gRPC `SetupService`. **Empty disables gRPC setup entirely** (standalone installs use the REST wizard). Never logged. |
| `SETUP_WINDOW_TTL` | `config.SetupWindowTTL` (`config.go:53`, parsed `config.go:175`) | Bounds how long the orchestrated setup surface stays reachable after process start. Default `30m`; must be `> 0`. Ignored for the standalone REST wizard. |

**Per-tenant / request settings.** The system tenant's identity is supplied at `create_tenant`: `name` (DNS slug, 3–63 chars), `display_name`, optional `description` (≤200 chars), and optional `metadata` (logo/favicon/language/timezone/date+time format/privacy+ToS URLs) — see `internal/setup/types.go:4` and `validation_setup.go:73`. The admin (`create_admin`) takes `username`, `email`, `password` (8–100 chars, plus the default policy check), optional `fullname`.

## Security considerations

- **First-admin password is policy-checked.** The bootstrap super-admin — the highest-privilege account, created on an unauthenticated route — is validated against `secpolicy.DefaultPasswordPolicy()` (`service_setup.go:331`). This closed a prior gap where the one creation path that skipped the blocklist/breach/strength checks accepted weak passwords like `password`.
- **DNS-slug tenant name.** `create_tenant` enforces the same slug pattern the tenant package uses elsewhere (`validation_setup.go:71`), preventing a bootstrap-time name (e.g. "My Company") that the console could then never re-save.
- **REST race closed on orchestrated instances.** `refuseWhenOrchestrated` (`handler_setup.go:31`) shuts the unauthenticated REST wizard whenever the instance is orchestrator-owned and a credential is issued, so the "whoever arrives first creates the admin" race only exists behind the token + mTLS gate.
- **Single-use without a ledger.** Bootstrap idempotence/single-use derives from the active system tenant + the window TTL, not a spent-token table — avoiding a second copy of that fact that could drift from it.
- **Orchestrator credentials avoid stored secrets.** `EnsureControlClient` uses `private_key_jwt` (RFC 7523) and stores no secret, so a DB dump yields nothing that can impersonate the orchestrator (`service_setup_provision.go:57`). `EnsureConsoleClient` is a public SPA client using `authorization_code` + PKCE S256 with no secret (`service_setup_provision.go:515`). Redirect / post-logout URIs must be absolute `https` (loopback `http` excepted) and may not contain wildcards (`validateHTTPSURIs`, `service_setup_provision.go:634`); a `jwks_uri` must be `https` (`service_setup_provision.go:99`).
- **Least-privilege control policy.** The default control grant covers only the provisioning surfaces and withholds end-user mutation, the defences, and the audit trail (`013_control_actions.go`).
- **gRPC mTLS.** When the control plane is on, mTLS is mandatory (`GRPCRequireMTLS` forced true, `config.go:27`), so a bootstrap caller has proven a CA-signed client certificate before the token is even compared.

## Related

- `./grpc.md` — gRPC surface overview (the `SetupService` bootstrap-gated row and control-plane split).
- `./iam-authorization.md` — how the control policy / principals provisioned here are enforced at runtime.
- `./multi-tenancy.md` — the system tenant and per-tenant seeding baseline reused by `SeedTenant`.
- `./iam-authorization.md` — services, APIs, permissions, roles, and policies created during setup.
