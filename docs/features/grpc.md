# gRPC Control Plane

> An opt-in, mTLS-guarded gRPC surface on `:50051` that an orchestrator (maintainerd core) and peer services drive to provision IAM resources and run runtime decisions (authorization, introspection, user reads) against this auth instance.

| | |
|---|---|
| **Status** | Implemented. Listener is OPT-IN and OFF by default; the provisioning half is a further explicit opt-in. |
| **Code** | `internal/server` (server, interceptors, permission map, audit, health, TLS); `internal/setup` (bootstrap SetupService handler); `internal/platform/config` (gating); `internal/platform/apperror` (gRPC error mapping) |
| **Endpoints** | gRPC on `:50051` (`internal/shared/constants.go:113`). Package `maintainerd.auth.v1`. 14 app services + `grpc.health.v1.Health` + server reflection (non-prod). Not an HTTP route surface. |
| **Storage** | No gRPC-specific tables. Reuses the same repositories/tables as REST; mutating calls write `management_audit_log` via the audit interceptor. Bootstrap single-use state is inferred from the system-tenant row (no ledger). |
| **Config** | `GRPC_ENABLED`, `CONTROL_PLANE_ENABLED`, `INSTANCE_ROLE`, `GRPC_TLS_CERT_FILE`, `GRPC_TLS_KEY_FILE`, `GRPC_CLIENT_CA_FILE`, `GRPC_REQUIRE_MTLS`, `SETUP_BOOTSTRAP_TOKEN` |

## Overview

gRPC here is **not** a second product API. It is the machine control plane and runtime data plane for service-to-service traffic. Browser and end-user flows (login, registration, password reset, MFA enrollment, consent) stay on REST and are intentionally absent from gRPC.

The surface splits into two roles gated by two independent switches (`internal/platform/config/control_plane.go:24-70`):

| Role | Switch | What it serves |
|---|---|---|
| **Runtime / data plane** | `GRPC_ENABLED=true` | Authorization PDP, token introspection, and the user/profile **reads** a peer service makes to run itself. Any multi-service deployment needs this; it does not expose provisioning. |
| **Control plane / provisioning** | `CONTROL_PLANE_ENABLED=true` | Everything above **plus** tenant/service/api/permission/policy/role/client and workload-identity provisioning, plus one-time bootstrap. Implies `GRPC_ENABLED` and **forces mTLS**. |

The two were split deliberately: gating a PDP behind "control plane" would force an organization to expose tenant/client/policy provisioning just to obtain an authorization check — "taking the dangerous surface to get the safe one" (`control_plane.go:29-37`).

Default deployment is **standalone**: both switches off → **no listener is bound at all** (`internal/server/grpc.go:56-64`). Not bound-and-firewalled, not bound-and-authenticated — not bound.

## How it works

### Startup and gating

1. `StartGRPCServer` (`grpc.go:47`) runs as a best-effort goroutine alongside REST (`cmd/server/workers.go:80-84`); REST owns process lifetime.
2. If `config.GRPCEnabled` is false, it logs once and returns without binding (`grpc.go:56-64`). `GRPCEnabled = ControlPlaneEnabled || GRPC_ENABLED` (`config.go:174`).
3. Otherwise it listens on `:50051`, builds server options (TLS + interceptor chains + 10 MiB max recv/send), registers services, sets health, and `Serve`s (`grpc.go:65-110`).
4. On `ctx` cancellation it flips overall health to `NOT_SERVING` and `GracefulStop`s to drain in-flight RPCs (`grpc.go:94-99`).

### Which services get registered

`grpcServices` (`grpc.go:140-159`) reads one authoritative list (`allGRPCServices`, `grpc.go:181-227`):

- **Control plane ON** → all 14 services registered.
- **Control plane OFF** → administrative services are **not registered at all** (`grpcAdministrativeServices`, `grpc.go:168-179`), so they never appear in reflection or the health surface. Only runtime services register: `AuthorizationService`, `OAuthIntrospectionService`, `UserService`, `UserProfileService`.

`UserService`/`UserProfileService` are **mixed**: their reads are runtime, their writes are administrative. Since gRPC registers whole services, their write methods are refused per-method instead of by withholding the service (`grpcAdministrativeMethods`, `grpc_permissions.go:205-220`; refusal at `grpc_interceptors.go:216-220`).

### Per-request auth pipeline

Unary interceptor chain (`grpc_interceptors.go:96-107`): **recovery → logging → timeout(60s) → auth → audit**. Stream chain is the same minus audit (`:109-117`). `authenticateAndAuthorizeGRPC` (`:203-312`) runs, in order:

1. **Instance-role check** (`authorizeInstanceRole`, `:409-424`) — core-provisioning RPCs are refused unless `CONTROL_PLANE_ENABLED` **and** `INSTANCE_ROLE=system`. Fails closed → `FailedPrecondition`. This is checked before any signature verification.
2. **Administrative-method-on-mixed-service** — a write on `UserService`/`UserProfileService` with the control plane off → `FailedPrecondition` (`:216-220`).
3. **Bootstrap methods** — `SetupService` RPCs authenticate with the pre-shared `x-setup-token` metadata (`authorizeSetupBootstrap`, `:440-475`), not a JWT, because at first boot no principals exist.
4. **Permission lookup** — `grpcServicePermissions` (`grpc_permissions.go:27-134`) is the entire authenticated surface. Any `maintainerd.auth.v1.*` method absent from it is **default-denied** (`PermissionDenied`); infrastructure services (health, reflection) fall outside the app prefix and pass through (`:231-240`).
5. **Verified caller** (`grpcVerifiedCaller`, `:505-564`) — requires `authorization: Bearer <jwt>`. Validated as an **access token only** (`jwt.ValidateAccessTokenWithContext`); an ID token is rejected. **DPoP sender-constrained tokens are refused** — a DPoP proof covers an HTTP method+URL and cannot be proven over gRPC (`:524-531`).
6. **Certificate binding** (RFC 8705, `enforceGRPCCertBinding`, `:656-681`) — if the client is registered with a bound cert thumbprint, the token must be presented over the connection holding that exact client certificate. This is gRPC's sender-constraint substitute for DPoP.
7. **Rate limit** — in-memory sliding window, **600 requests / minute per principal** (`defaultGRPCRateLimit`, `:33-34`), keyed `svc:`/`client:`/`sub:` (`grpcPrincipalKey`, `:599-607`). Over → `ResourceExhausted`.
8. **Service-account check** — `svc` claim or `sub_type=service` required, else `PermissionDenied` (`:270-272`).
9. **Step-up** — methods in `grpcStepUpMethods` require `acr` = level 2 (`jwt.ACRLevel2`), else `PermissionDenied` (`:273-275`).
10. **PDP decision** — for PDP-gated methods (non-empty permission), calls `AuthorizationService.Authorize` with the caller's `svc` principal, the method's action, and the token's `TenantID`. A missing PDP fails closed (`Internal`); a token with no tenant is refused so the lookup cannot collapse onto the system tenant's service (`:276-299`).
11. **Auth context + actor** — builds `AuthContext` (`grpcAuthContext`, `:333-392`). For state-changing on-behalf-of RPCs (`grpcActorRequiredMethods`), the acting user comes **only** from the signed `on_behalf_of` claim, must live in the caller's own tenant (`user.ValidateTenantAccess`, no system override), and must be active — otherwise `PermissionDenied`.

### Health, reflection, correlation, telemetry

- **Health** — `grpc.health.v1.Health/Check` and `Watch`. Each served service is advertised `SERVING` at register; overall (`""`) reflects **readiness** — DB reachable, optional Redis reachable, JWKS public key loaded — refreshed every 5 s (`grpc.go:229-267`).
- **Reflection** — registered only when `AppEnv != "production"` (`grpc.go:89-91`).
- **Request correlation** — accepts `x-request-id` (aliases `request-id`, `x-correlation-id`, `correlation-id`), synthesizes a UUID if absent, logs it, and returns it in **trailing** metadata (`:622-643`, `:149`).
- **Telemetry** — OpenTelemetry via `otelgrpc.NewServerHandler()` stats handler (`grpc.go:271`).
- **Audit** — every **mutating** RPC writes a `management_audit_log` row from the interceptor (who/what/outcome), independent of whether the handler logs its own row; reads are not audited (`grpc_audit_interceptor.go`).

## Implementation

### Served services (`maintainerd.auth.v1`)

`allGRPCServices` (`internal/server/grpc.go:181-227`). "System-only" = registered only with the control plane on **and** `INSTANCE_ROLE=system`.

| Service | Kind | Handler package |
|---|---|---|
| `AuthorizationService` (`Authorize`) | Runtime PDP | `iam` |
| `OAuthIntrospectionService` (`Introspect`) | Runtime | `oauth` |
| `UserService` | Mixed (reads runtime; writes control-plane) | `user` |
| `UserProfileService` | Mixed | `user` |
| `SetupService` | Bootstrap (system-only) | `setup` |
| `TenantService` | Control plane (system-only) | `tenant` |
| `TenantSettingService` | Control plane (system-only) | `tenant` |
| `ServiceService` | Control plane (system-only) | `iam` |
| `APIService` | Control plane (system-only) | `iam` |
| `PermissionService` | Control plane (system-only) | `iam` |
| `PolicyService` | Control plane (system-only) | `iam` |
| `RoleService` | Control plane (system-only) | `iam` |
| `ClientService` | Control plane (system-only) | `client` |
| `WorkloadIdentityFederationService` | Control plane (system-only) | `federation` |

`ServiceService.GetMyPolicyBundle` and `TenantService.GetDefaultTenant` are **peer-service** methods (`grpcPeerServiceMethods`, `grpc_permissions.go:274-277`): read-only, service-account-authenticated (permission `""`), and exempt from the system-instance restriction so ordinary instances can still call them.

> **Drift corrected:** earlier documentation listed ~12 additional gRPC services (auth events, branding, email/SMS config + templates, identity providers, invites, IP restriction rules, registration flows, security settings, webhook endpoints, login templates). Those service blocks were **removed from the protos** — the `.proto` files retain only messages, `grep '^service' proto/maintainerd/auth/v1/*.proto` shows zero service declarations for them — so they are **REST/console-only** and no longer part of the gRPC surface (`grpc.go:120-139`; `grpc_permissions.go:19-26`). Port is **`:50051`**, not `9090`.

### Key files

| Concern | File |
|---|---|
| Listener, opt-in gate, registration, TLS, health | `internal/server/grpc.go` |
| Interceptor chain, auth/authz, cert binding, on-behalf-of, rate limiter, request-id | `internal/server/grpc_interceptors.go` |
| Permission map, step-up set, bootstrap set, administrative/system-only/peer sets | `internal/server/grpc_permissions.go` |
| Mutating-call audit interceptor | `internal/server/grpc_audit_interceptor.go` |
| Bootstrap SetupService gRPC handler | `internal/setup/handler_setup_grpc.go` |
| Two-switch + instance-role config, control-plane TLS validation | `internal/platform/config/control_plane.go`, `internal/platform/config/config.go` |
| `error → gRPC status` mapping (+ AIP-193 details) | `internal/platform/apperror/grpc.go` |
| Generated stubs | `internal/platform/gen/go/maintainerd/auth/` (package `authv1`) |
| Proto sources | `proto/maintainerd/auth/v1/*.proto` |

### Permission gating

`grpcServicePermissions` (`grpc_permissions.go:27-134`) maps every served method → the permission the caller's service account must hold; `""` = authenticated-but-not-PDP-gated (verification-style reads like `Authorize`, `Introspect`, `GetMyPolicyBundle`, `GetDefaultTenant`). Workload-identity federation permissions are merged in via `init()` (`:337-341`). Permission strings are identical to the REST internal API — one authorization vocabulary across both transports.

Step-up-required methods (`grpcStepUpMethods`, `:136-146`): `SetTenantStatus`, `DeleteTenant`, `GetClientSecret`, `RotateClientSecret`, `SetUserStatus`, `DeleteUser`, `ForceUserPasswordChange`, `AssignUserRoles`, `RemoveUserRole`.

### Bootstrap (SetupService)

Bootstrap RPCs (`grpcBootstrapMethods`, `:180-191`): `GetSetupStatus`, `CreateTenant`, `CreateAdmin`, `CreateProfile`, `RegisterControlService`, `EnsureControlClient`, `EnsureResourceAPI`, `EnsureRole`, `EnsureConsoleClient`, `CompleteSetup`. They are gated by `SETUP_BOOTSTRAP_TOKEN` (constant-time compare, `authorizeSetupBootstrap`), and locked by the setup service (`ensureSetupOpen`) once the system tenant exists — single-use is a property of that row, not a separate ledger (`grpc_interceptors.go:460-471`). `RegisterControlService`, `EnsureControlClient`, `EnsureResourceAPI`, `EnsureRole`, `EnsureConsoleClient` are additionally **system-instance-only** (`grpcSystemInstanceOnlyMethods`, `:289-300`). See [./setup-and-bootstrap.md](./setup-and-bootstrap.md).

### Error model

`apperror.ToGRPCError` / `classifyGRPCError` (`internal/platform/apperror/grpc.go`) map domain errors to canonical codes, attaching AIP-193 `ErrorInfo` and, where relevant, `BadRequest` (validation) or `RetryInfo` (throttling) details. Raw internal errors are never leaked.

| Situation | gRPC code |
|---|---|
| Missing/invalid/expired token; DPoP-bound token; wrong bound cert; bad setup token | `Unauthenticated` |
| Valid token, no policy/permission; not a service account; missing step-up; unmapped method; on-behalf-of guard | `PermissionDenied` |
| Control plane off / wrong instance role for the method | `FailedPrecondition` |
| Validation failure | `InvalidArgument` (+ `BadRequest`) |
| Not found / not in caller's tenant | `NotFound` |
| Uniqueness / state conflict | `AlreadyExists` |
| Rate limit exceeded | `ResourceExhausted` (+ `RetryInfo`) |
| Dependency down | `Unavailable` |
| Deadline hit | `DeadlineExceeded` |
| Panic / unhandled | `Internal` |

## Configuration

All resolved in `internal/platform/config` at startup.

| Env var | Default | Effect |
|---|---|---|
| `GRPC_ENABLED` | `false` | Binds `:50051` and serves runtime services. Forced `true` when `CONTROL_PLANE_ENABLED=true` (`config.go:174`). |
| `CONTROL_PLANE_ENABLED` | `false` | Adds the provisioning surface; **implies `GRPC_ENABLED`** and **forces mTLS** (`config.go:170-190`). |
| `INSTANCE_ROLE` | `system` | `system` or `regular`. Only a `system` instance serves core-provisioning RPCs; unrecognized values fail startup (`control_plane.go:87-97`). Inert unless the control plane is on. |
| `GRPC_TLS_CERT_FILE` / `GRPC_TLS_KEY_FILE` | unset | Server TLS keypair. **Required** when the control plane is on or in production. |
| `GRPC_CLIENT_CA_FILE` | unset | CA that must have issued the client cert. **Required** whenever mTLS is required. An empty/unparsable pool is fatal (would reject every peer). |
| `GRPC_REQUIRE_MTLS` | `false` | Requires+verifies client certs on a **non-control-plane** listener. The control plane derives mTLS from `CONTROL_PLANE_ENABLED` regardless of this flag (`grpc.go:313`). |
| `SETUP_BOOTSTRAP_TOKEN` | unset | Pre-shared credential for gRPC `SetupService`. Empty → gRPC setup disabled (standalone instances bootstrap via the REST wizard). |

TLS resolution (`loadGRPCTLSConfig`, `grpc.go:301-361`): control plane never runs in the clear; production requires TLS; a non-production, non-control-plane listener without a keypair logs a warning and runs plaintext. `MinVersion` is TLS 1.2. The control-plane guard is re-asserted at server construction (`grpc.go:287-294`) even though `loadGRPCTLSConfig` already enforces it — the cost of a mistake is a control plane serving an unverified peer.

Fixed (not env-tunable): rate limit 600/min, request timeout 60 s, max message size 10 MiB, health refresh 5 s.

## Security considerations

- **Off by default, opt-in in two stages.** No switch → no socket. The dangerous provisioning surface is a separate, explicit opt-in from the safe runtime surface.
- **mTLS proves the peer on the control plane.** A bearer token asserts "it is core"; a verified client certificate demonstrates it. mTLS is mandatory and non-disableable when the control plane is on; an unreadable/empty client CA is fatal, never a silent downgrade (`control_plane.go:99-130`).
- **Instance role is unforgeable config.** `INSTANCE_ROLE` is fixed at provision time — not in the DB, not carried on any request — so no RPC can promote an ordinary instance into the ecosystem's system IAM. Ordinary instances answer no orchestrator-provisioning RPCs.
- **Access tokens only; theft-resistant tokens are honored, not bypassed.** ID tokens are rejected. DPoP-bound tokens are refused over gRPC (their proof can't apply); certificate-bound tokens (RFC 8705) must be presented over the exact bound client cert, verified from `VerifiedChains` (not client-supplied `PeerCertificates`).
- **Default-deny surface.** Any `maintainerd.auth.v1` method not explicitly classified is denied. A missing PDP fails closed. A tenant-less service token cannot reach a PDP-gated method.
- **On-behalf-of is signed, tenant-bounded, and revocation-aware.** The actor comes only from the signed `on_behalf_of` claim, must belong to the caller's tenant (no system-tenant override), and must be active — closing the cross-tenant takeover the old request-body actor field allowed.
- **Rate limiting precedes expensive work** and is checked before the cert-binding budget can be consumed by another principal.
- **Full audit of mutations.** The interceptor logs who/what/outcome for every mutating RPC, so the control plane — the actor most worth auditing — is never invisible.

## Related

- [./setup-and-bootstrap.md](./setup-and-bootstrap.md) — the SetupService bootstrap sequence gRPC exposes.
- [./federation.md](./federation.md) — workload identity federation served over the control plane.
- [./multi-tenancy.md](./multi-tenancy.md) — tenant boundary the token/actor checks enforce.
- [./clients.md](./clients.md) — OAuth clients, secrets, and certificate binding administered here.
- [./authentication.md](./authentication.md) — access-token issuance and the claims the interceptor verifies.
