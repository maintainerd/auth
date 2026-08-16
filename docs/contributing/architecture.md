# Architecture

This document describes the internal architecture of **Maintainerd Auth** — a multi-tenant
authentication and authorization service written in Go. If you are new to the codebase, read
this first to understand how the pieces fit together before diving into any specific package.

The code is organized **by domain (bounded context)**, not by architectural layer. Each domain
package under `internal/` owns everything for its slice of the product — model, repository,
service, HTTP/gRPC handlers, routes, and types — and reusable infrastructure lives one level
deeper under `internal/platform/`.

For feature-level detail, see the [feature docs](../features/) (e.g.
[OAuth 2.0 / OIDC](../features/oauth2-oidc.md), [multi-tenancy](../features/multi-tenancy.md),
[IAM authorization](../features/iam-authorization.md), [gRPC control plane](../features/grpc.md)).
For a package-tree walkthrough see [`./code-structure.md`](./code-structure.md); to run the
service locally see [`./getting-started.md`](./getting-started.md).

---

## Table of Contents

- [High-Level Overview](#high-level-overview)
- [Architecture Diagram](#architecture-diagram)
- [End-to-End Flow](#end-to-end-flow)
- [Startup Flow](#startup-flow)
- [Request Lifecycle](#request-lifecycle)
- [Entry Point & Bootstrap](#entry-point--bootstrap)
- [Dependency Injection](#dependency-injection)
- [File & Naming Conventions](#file--naming-conventions)
- [Top-Level Layout](#top-level-layout)
- [Domain Package Catalog](#domain-package-catalog)
- [Platform Packages](#platform-packages)
- [Transport Hosting](#transport-hosting)
- [Composition Root](#composition-root)
- [Multi-Tenancy](#multi-tenancy)
- [Database & Migrations](#database--migrations)
- [Caching](#caching)
- [Platform Boundary](#platform-boundary)
- [Key Design Patterns](#key-design-patterns)
- [Technology Stack](#technology-stack)

---

## High-Level Overview

Within a domain, dependencies still flow one way:

```
Handlers (REST / gRPC)
       ↓
   Services  (business logic)
       ↓
  Repositories  (data access)
       ↓
   Database / Cache
```

Each layer depends only on the layer below it, and cross-layer communication happens through
**interfaces**, so every layer is independently testable. What is *not* layered is the folder
structure: instead of global `service/`, `repository/`, and `model/` buckets, each domain owns
its own `service_*.go`, `repository_*.go`, and `model_*.go` files. Cross-domain reads go through
small **consumer interfaces** (declared in the consuming domain's `deps.go` and satisfied by a
thin adapter in the composition root), which keeps import cycles out and coupling explicit.

---

## Architecture Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                       cmd/server/                            │
│  config → DB → Redis → migrations → seeders → DI → servers  │
└────────────┬─────────────────────────────────┬───────────────┘
             │                                 │
  ┌──────────▼───────────┐          ┌──────────▼──────────┐
  │  REST Server :8080   │          │  REST Server :8081   │
  │  (Internal / Admin)  │          │  (Public / Auth)     │
  │  VPN-only access     │          │  Public-facing       │
  └──────────┬───────────┘          └──────────┬───────────┘
             │                                 │
             └────────────────┬────────────────┘
                              │
  ┌───────────────────────────▼─────────────────────────────┐
  │                    Middleware Stack                      │
  │                                                         │
  │  Security Headers → Request Size Limit → Timeout        │
  │  → Security Context → JWT Auth → User Context           │
  │  → Permission Check                                     │
  └───────────────────────────┬─────────────────────────────┘
                              │
  ┌───────────────────────────▼─────────────────────────────┐
  │                  Domain REST Handlers                    │
  │                                                         │
  │  Parse request → Validate DTO → Call service            │
  │  → Map result to response DTO → Write JSON response     │
  └───────────────────────────┬─────────────────────────────┘
                              │
  ┌───────────────────────────▼─────────────────────────────┐
  │                    Domain Services                       │
  │                                                         │
  │  Business rules, transaction management,                │
  │  tenant isolation, audit + event emission               │
  └──────────┬────────────────────────────────┬─────────────┘
             │                                │
  ┌──────────▼───────────┐         ┌──────────▼──────────┐
  │  Domain Repositories │         │      Redis           │
  │                      │         │  (cache + rate       │
  │  Generic base +      │         │   limiting)          │
  │  domain queries +    │         └─────────────────────┘
  │  tenant scoping      │
  └──────────┬───────────┘
             │
  ┌──────────▼───────────┐         ┌──────────────────────────────┐
  │    PostgreSQL        │         │  gRPC Control Plane :50051   │
  │    (GORM + JSONB)    │         │  (opt-in, mTLS-gated)        │
  └──────────────────────┘         └──────────────────────────────┘
```

The gRPC listener is **opt-in** and off by default — it is the machine control plane an
orchestrator drives, not a second product API. See [Transport Hosting](#transport-hosting).

---

## End-to-End Flow

This diagram shows how the application connects from the executable entrypoint down into the
transport runtime, domain packages, and reusable platform infrastructure. It shows ownership
boundaries, not every function call.

```text
┌──────────────────────────────────────────────────────────────┐
│                       cmd/server/                            │
│                                                              │
│  main.go                                                     │
│    │                                                         │
│    ▼                                                         │
│  bootstrap.go: run(ctx)                                      │
│    ├─ logging.go            → bootstrap logger → configured  │
│    ├─ platform/config       → env + secrets                  │
│    ├─ telemetry.go          → traces + metrics               │
│    ├─ platform/jwt          → RSA keys + token config        │
│    ├─ platform/config       → PostgreSQL + Redis             │
│    ├─ platform/security     → Redis-backed rate limiter      │
│    ├─ platform/runner       → database migrations + seeders  │
│    ├─ internal/app          → dependency graph               │
│    ├─ workers.go            → retention + key-rotation +      │
│    │                          opt-in gRPC background workers  │
│    └─ internal/server       → foreground REST lifecycle       │
└───────────────────────────────┬──────────────────────────────┘
                                │
                                ▼
┌──────────────────────────────────────────────────────────────┐
│                       internal/app                           │
│                                                              │
│  repositories.go → construct domain repositories bound to DB │
│  services.go     → construct domain services; inject repos + │
│                     platform helpers                         │
│  adapters_*.go   → connect cross-domain consumer interfaces  │
│  app.go          → expose App: services + DB + Redis + cache │
│  application.go  → adapt App into server.Application         │
└───────────────────────────────┬──────────────────────────────┘
                                │
                                ▼
┌──────────────────────────────────────────────────────────────┐
│                      internal/server                         │
│                                                              │
│  handlers.go → construct REST handlers from services         │
│  router.go   → mount internal/public routes + middleware     │
│  rest.go     → serve :8080 internal and :8081 public         │
│  grpc.go     → serve opt-in control plane on :50051          │
│  health.go   → /health and /ready                            │
│  openapi.go  → /openapi.json                                 │
└───────────────────────────────┬──────────────────────────────┘
                                │
                                ▼
┌──────────────────────────────────────────────────────────────┐
│                 internal/<domain> packages                   │
│                                                              │
│  routes.go                                                   │
│    ↓                                                         │
│  handler_<name>.go                 REST + gRPC handlers      │
│    ├─ types.go                     request/response DTOs     │
│    ├─ validation_<name>.go         DTO validation            │
│    └─ service_<name>.go            business logic            │
│          ├─ deps.go                upstream contracts        │
│          └─ repository_<name>.go   persistence boundary      │
│                └─ model_<name>.go  GORM model + hooks        │
└───────────────────────────────┬──────────────────────────────┘
                                │
                                ▼
┌──────────────────────────────────────────────────────────────┐
│                      internal/platform                       │
│                                                              │
│  database / cache / jwt / dpop / signedurl / crypto          │
│  security / middleware / logging / telemetry / geoip         │
│  email / sms / templates                                     │
│  apperror / response / pagination / ptr / valid / jsonutil   │
│                                                              │
│  Platform is reusable infrastructure. It does not import      │
│  domain packages.                                            │
└──────────────────────────────────────────────────────────────┘
```

---

## Startup Flow

```text
cmd/server/main.go
  │
  ▼
run(context.Background())
  │
  ├─ initBootstrapLogger()
  │    Temporary JSON logger for early startup failures.
  │
  ├─ config.Init()
  │    Load environment variables and resolve secrets.
  │
  ├─ initConfiguredLogger()
  │    Apply LOG_LEVEL and PII redaction.
  │
  ├─ initTelemetry(ctx)
  │    Start tracing and metrics; return one deferred shutdown hook.
  │
  ├─ jwt.InitJWTKeys()
  │    Validate the RSA key pair before accepting traffic.
  │
  ├─ config.InitDB() + config.NewRedisClient()
  │    Create shared PostgreSQL and Redis clients.
  │
  ├─ security.InitRateLimiter(redis)
  │    Wire Redis-backed rate limiting.
  │
  ├─ runner.RunMigrations(db)
  │    Bring the schema current before repositories/services start.
  │
  ├─ app.NewApp(db, redis)
  │    Build repositories, services, cross-domain adapters, and cache.
  │
  ├─ application.ServerApplication()
  │    Convert the app bundle into transport dependencies.
  │
  ├─ startBackgroundWorkers(ctx, app, serverApp)
  │    Start retention, key-rotation, and the opt-in gRPC control plane.
  │
  └─ server.StartRESTServer(serverApp)
       Start internal/public REST servers and block until shutdown.
```

If any step through `app.NewApp` fails, the process exits immediately with a non-zero code.

---

## Request Lifecycle

A typical authenticated API request flows through these steps:

1. A reverse proxy (or a Kubernetes Service) forwards the request to the appropriate port.
2. **Security Headers Middleware** sets CSP, HSTS, `X-Frame-Options`, etc.
3. **Request Size Limit Middleware** rejects oversized payloads.
4. **Timeout Middleware** applies a deadline to the request context.
5. **Security Context Middleware** extracts client IP, user-agent, and generates a `X-Request-ID`.
6. **JWT Auth Middleware** extracts and validates the Bearer token (or `access_token` cookie),
   populating context with `sub`, `scope`, `aud`, `iss`, `jti`, `client_id`, `provider_id`.
7. **User Context Middleware** resolves the full user (roles, permissions, tenant, client) from
   the Redis cache or the database and populates context.
8. **Permission Middleware** checks that the user holds at least one required permission.
9. The **handler** parses and validates the request DTO, calls the service, maps the result, and
   writes the response.
10. The **service** runs business logic inside a database transaction.
11. The **repository** executes tenant-scoped queries via GORM.

```text
HTTP Client
  │
  ▼
REST Port
  ├─ :8080 internal API
  └─ :8081 public API
        │
        └─ public IP rate limit

Both ports
  │
  ▼
Common Middleware  (internal/platform/middleware)
  ├─ recoverer
  ├─ security headers
  ├─ security context + request ID
  ├─ structured request logging
  ├─ request size limit
  ├─ timeout
  ├─ CORS
  └─ JSON content-type enforcement
  │
  ▼
Route Middleware
  ├─ JWT or cookie validation
  ├─ user context lookup
  └─ permission checks
  │
  ▼
Domain Route (internal/<domain>/routes.go)
  │
  ▼
handler_<name>.go
  ├─ parse request
  ├─ validate types.go DTO with validation_<name>.go
  └─ call service
  │
  ▼
service_<name>.go
  ├─ business rules
  ├─ transaction boundaries
  ├─ cache / session / event integrations
  └─ repository calls
  │
  ▼
repository_<name>.go
  │
  ▼
model_<name>.go
  │
  ▼
PostgreSQL

Side paths:
  Route middleware ───────→ Redis auth/session/rate-limit cache
  Service layer ──────────→ Redis cache and token state
  Service layer ──────────→ authevent + event planes → webhook dispatcher
```

---

## Entry Point & Bootstrap

**Package:** `cmd/server`

`main.go` is thin — it calls `run(context.Context)`, and focused bootstrap files
(`bootstrap.go`, `logging.go`, `telemetry.go`, `workers.go`) handle each concern, then delegate
to `internal/app`. The boot sequence is strictly sequential — each step must succeed before the
next begins:

```
1. Configure structured JSON logging
2. Load environment variables and resolve secrets
3. Parse and validate the RSA JWT key pair
4. Connect to PostgreSQL
5. Connect to Redis and validate the connection
6. Run database migrations (with an advisory lock) + seeders
7. Wire the dependency graph  →  app.NewApp(db, redisClient)
8. Start background workers (retention, key rotation, opt-in gRPC)
9. Start two REST servers (internal + public)
10. Wait for an OS signal (SIGINT / SIGTERM)
11. Graceful shutdown: drain REST, cancel worker contexts, exit
```

If any step from 1–7 fails, the process exits immediately with a non-zero code.

---

## Dependency Injection

**Files:** `internal/app/app.go`, `internal/app/repositories.go`, `internal/app/services.go`,
`internal/app/adapters_*.go`, `internal/app/application.go`

DI is **manual and constructor-based** — no frameworks, no reflection, no globals. The wiring
happens in phases:

```
Phase 1: initRepos(db)            → construct domain repositories bound to the DB
Phase 2: initServices(db, repos)  → construct domain services, injecting repos + platform helpers
Phase 3: adapters_*.go            → satisfy each domain's consumer interfaces (deps.go) with
                                     thin cross-domain adapters
Phase 4: NewApp(db, redis)        → return an App struct exposing services + DB + Redis + cache
```

The `App` struct exposes **service interfaces** (never concrete types), which makes every
component swappable for testing. It is adapted into a `server.Application` (`application.go`) and
passed to both the REST and gRPC hosts, which build their handlers from those interfaces.

The `adapters_*.go` files are the glue that lets `authn` read a `user`, `oauth` read a `client`,
and so on, without any domain importing another domain's package — the consumer declares the
narrow interface it needs in its own `deps.go`, and the adapter binds it to the upstream service.

---

## File & Naming Conventions

Packages — not files — are the unit of API design in Go. File names exist for humans; these
conventions let a contributor predict where things live. Every domain package uses this
**role-first** file vocabulary (a package only includes files for the roles it actually owns):

```text
internal/<domain>/
  model_<name>.go          GORM model, table name, hooks
  repository_<name>.go     persistence interface + GORM implementation
  service_<name>.go        business behavior and transactions
  handler_<name>.go        HTTP handler logic
  handler_<name>_grpc.go   gRPC handler (only where a control-plane surface exists)
  validation_<name>.go     DTO validation
  types.go                 API request/response DTOs and shared package types
  deps.go                  consumer-side interfaces onto other domains
  foundation.go            local aliases over platform helpers (re-export glue only)
  routes.go                route mounting
  <name>_test.go           tests (Go-enforced suffix)
```

General rules:

| Rule | Example |
|---|---|
| Lowercase only, `_` separates words | `email_verification.go` |
| One dominant concept per file, named after it | `service_login.go` defines the login service |
| Avoid stuttering the package name | `user.Service`, not `user.UserService` |
| No grab-bag files | name helpers after what they do (`redirect_match.go`), never `utils.go` |
| Cross-aggregate references are IDs, not embedded structs | `TenantID uuid.UUID`, not `*tenant.Tenant` |

Each domain owns its own GORM models — there is **no** central `internal/model/` package.
Navigation across aggregates goes through service calls, not GORM `Preload`, which is what keeps
domain-owned models free of import cycles.

---

## Top-Level Layout

```
maintainerd-auth/
├── cmd/
│   └── server/                     # thin binary entrypoint + bootstrap files
├── internal/                       # private application code
│   ├── app/                        # composition root (wiring + cross-domain adapters)
│   ├── server/                     # transport hosting (REST + gRPC)
│   ├── oauth/                      # OAuth 2.0 / OIDC flows
│   ├── user/                       # users, profiles, settings, accounts
│   ├── authn/                      # login, register, password, magic link, sms login, sessions
│   ├── mfa/                        # TOTP, WebAuthn, backup codes
│   ├── tenant/                     # tenants, members, settings
│   ├── iam/                        # services, APIs, permissions, roles, policies, authorization
│   ├── client/                     # OAuth clients
│   ├── idp/                        # identity providers, registration flows
│   ├── federation/                 # workload identity federation
│   ├── invite/                     # invitations
│   ├── secpolicy/                  # security settings, IP restrictions
│   ├── branding/                   # branding + email/sms/login templates
│   ├── notifier/                   # per-tenant outbound email/SMS provider config
│   ├── authevent/                  # auth event log + retention
│   ├── auditlog/                   # management (control-plane) audit log
│   ├── event/                      # integration event plane (outbox → RabbitMQ)
│   ├── webhook/                    # outbound webhook endpoints + delivery
│   ├── setup/                      # first-run bootstrap / provisioning
│   ├── dashboard/                  # tenant summary metrics
│   ├── authctx/                    # shared auth-context types (thin, no behavior)
│   ├── webui/                      # embedded admin console + hosted identity SPAs
│   ├── shared/                     # last-resort shared value types (thin leaf)
│   └── platform/                   # cross-cutting infrastructure (see below)
├── proto/                          # protobuf definitions
├── docs/                           # project documentation
├── scripts/                        # developer and CI helper scripts
├── go.mod / go.sum                 # Go module definition
├── Dockerfile                      # production container build
└── README.md
```

Domains sit at the top of `internal/`; infrastructure sits one level deeper under
`internal/platform/`. This keeps the top level reading like a list of *what this service does*,
not *how it is built*. `internal/app/` and `internal/server/` are the composition root and
transport host — they are not business domains, and `internal/shared/` is a thin leaf for value
types shared by three or more unrelated domains (kept small on purpose).

---

## Domain Package Catalog

One concise entry per domain — responsibility plus representative files.

**`oauth/`** — Every OAuth 2.0 / OIDC flow: authorization, token, consent, PAR, device, CIBA,
token exchange, dynamic client registration, userinfo, and discovery/JWKS. Owns the OAuth
aggregates (`OAuthAuthorizationCode`, `OAuthRefreshToken`, `OAuthConsentGrant`,
`OAuthConsentChallenge`, and the PAR/device/CIBA records). Representative:
`service_oauth_authorize.go`, `service_oauth_token.go`, `handler_discovery.go`,
`handler_oauth_introspection_grpc.go`. See [OAuth 2.0 / OIDC](../features/oauth2-oidc.md).

**`user/`** — User accounts, identities, profiles, settings, and self-service account actions.
Owns `User`, `UserIdentity`, `UserRole`, `UserToken`, `Profile`, `UserSetting`. Representative:
`service_user.go`, `handler_user.go`, `handler_user_grpc.go`, `model_user.go`.

**`authn/`** — Everything a user does to prove identity: login, register, forgot/reset password,
password policy, email verification, magic link, SMS login, and sessions. Holds **no aggregates
of its own** — it reads `user`, `client`, etc. through consumer interfaces in `deps.go`.
Representative: `service_login.go`, `handler_login.go`, `service_reset_password.go`.
See [authentication](../features/authentication.md) and [sessions](../features/sessions.md).

**`mfa/`** — Multi-factor auth: TOTP, WebAuthn, and backup codes. Owns `UserTOTPSecret`,
`UserWebAuthnCredential`, `UserBackupCode`. Representative: `service_mfa.go`, `handler_mfa.go`.
See [multi-factor auth](../features/multi-factor-auth.md).

**`tenant/`** — Tenants, members, tenant-service links, and settings. Owns `Tenant`,
`TenantMember`, `TenantSetting`. Exposes both REST and gRPC control-plane handlers.
Representative: `service_tenant.go`, `handler_tenant.go`, `handler_tenant_grpc.go`.
See [multi-tenancy](../features/multi-tenancy.md).

**`iam/`** — The "what can be accessed" half of authorization: services (resources), APIs,
permissions, roles, policies, and the authorization decision engine. Owns `Service`, `API`,
`Permission`, `Role`, `Policy` and their pivots. Representative: `service_permission.go`,
`service_role.go`, `service_authorization.go`, `handler_role_grpc.go`.
See [IAM authorization](../features/iam-authorization.md).

**`client/`** — OAuth clients: redirect URIs, permissions, API bindings, CORS origins, and secret
encryption. Owns `Client` and its association models. Representative: `service_client.go`,
`redirect_match.go`, `handler_client_grpc.go`. See [clients](../features/clients.md).

**`idp/`** — Identity providers and registration flows. Owns `IdentityProvider`,
`RegistrationFlow`, `RegistrationFlowRole`. Representative: `service_identity_provider.go`,
`handler_identity_provider.go`.

**`federation/`** — Workload identity federation (exchange an external workload token for a
maintainerd token). Owns the federation config model and exposes both REST and gRPC surfaces.
Representative: `service_workload_identity_federation.go`, `service_workload_exchange.go`,
`handler_workload_identity_federation_grpc.go`. See [federation](../features/federation.md).

**`invite/`** — Invitations and invite-to-role assignment. Owns `Invite`, `InviteRole`.
Representative: `service_invite.go`, `handler_invite.go`.
See [registration and invites](../features/registration-and-invites.md).

**`secpolicy/`** — Security settings and IP-restriction rules (named `secpolicy`, not `security`,
to avoid colliding with the platform `security/` package). Owns `SecuritySetting`,
`SecuritySettingsAudit`, `IPRestrictionRule`. Representative: `service_security_setting.go`,
`service_ip_restriction_rule.go`. See [security settings](../features/security-settings.md).

**`branding/`** — Branding plus email, SMS, and login templates. Owns `Branding`, `EmailTemplate`,
`SMSTemplate`, `LoginTemplate`. Representative: `service_branding.go`, `service_email_template.go`.
See [branding and templates](../features/branding-and-templates.md).

**`notifier/`** — Per-tenant configuration for *how* email and SMS are sent (provider, sender,
credentials); the provider *adapters* themselves live in `platform/email` and `platform/sms`.
Owns `EmailConfig`, `SMSConfig`, `SMSOTP`. Representative: `service_email_config.go`,
`service_sms_config.go`. See [email and SMS](../features/email-and-sms.md).

**`authevent/`** — The auth event log (login success/failure, lockouts, etc.) with retention and
partitioning. Owns `AuthEvent`. Snake_case event names (`authn_login_success`). Representative:
`service_event.go`, `partition.go`, `handler_event.go`. See [events](../features/events.md).

**`auditlog/`** — The management (control-plane) audit log — a record of privileged provisioning
actions, distinct from the per-user `authevent` log. Representative:
`service_management_audit_log.go`, `handler_management_audit_log.go`.
See [audit logging](../features/audit-logging.md).

**`event/`** — The integration event plane: a transactional outbox that relays domain events
(dotted names like `user.created`) to RabbitMQ for downstream consumers. Representative:
`service_event.go`, `relay.go`, `rabbitmq_publisher.go`, `model_outbox.go`.

**`webhook/`** — Outbound webhook endpoints plus the delivery mechanism (dispatch, sign, deliver,
retry). Owns `WebhookEndpoint`. Representative: `service_webhook_endpoint.go`, `dispatcher.go`,
`deliver.go`, `signer.go`. See [webhooks](../features/webhooks.md).

**`setup/`** — First-run bootstrap: creates the system tenant, admin, and default IDP/client, and
runs seeders. Replay-guarded and idempotent. Exposes both a REST `/setup` surface and a gRPC
`SetupService`. Representative: `service_setup.go`, `service_setup_provision.go`,
`handler_setup_grpc.go`, `seeder/`. See [setup and bootstrap](../features/setup-and-bootstrap.md).

**`dashboard/`** — Read-only tenant summary metrics (user/role/client counts) for the admin
console. Representative: `service.go`, `handler.go`.

**`authctx/`** — A thin, behavior-free package holding the shared auth-context types
(`UserContext`, `AuthContext`, and the `AuthUser`/`AuthTenant`/`AuthProvider`/`AuthClient`
structs) that middleware populates and the cache serializes. Single file: `types.go`.

**`webui/`** — Serves the bundled admin console and hosted identity SPAs directly from the Go
process (embedded assets), so the all-in-one release image is a single binary with no nginx and
no process supervisor. Representative: `webui.go`, `assets_embed.go`.

---

## Platform Packages

All cross-cutting infrastructure lives under `internal/platform/`. Domains import from `platform/`;
`platform/` never imports a domain.

| Package | Purpose |
|---|---|
| `platform/apperror` | Domain error types + HTTP/gRPC status mapping |
| `platform/cache` | Redis-backed cache abstraction + JTI denylist |
| `platform/config` | Env loading, secret resolution, DB init, Redis client |
| `platform/cookie` | Secure cookie helpers |
| `platform/crypto` | Hashing, encryption, OTP, PKCE helpers |
| `platform/database` | DB setup, pool tuning, and the generic base repository |
| `platform/dpop` | OAuth DPoP proof validation (RFC 9449) |
| `platform/email` | SMTP email delivery (see note below) |
| `platform/gen` | Generated code (protobuf → `platform/gen/go/...`) |
| `platform/geoip` | IP geolocation lookups |
| `platform/jsonutil` | JSON helpers |
| `platform/jwt` | JWT signing/verification, JWK set, issuer allowlist |
| `platform/logging` | slog setup + PII redaction handler |
| `platform/middleware` | Shared HTTP middleware (auth, CORS, rate limit, permissions, headers, …) |
| `platform/model` | Shared low-level persistence primitives (not domain models) |
| `platform/pagination` | Shared pagination helpers (REST + gRPC) |
| `platform/ptr` | Pointer helpers |
| `platform/response` | Shared HTTP response helpers (JSON encoding, error mapping) |
| `platform/retry` | Retry/backoff helpers |
| `platform/runner` | Background workers: migrations, seeders, retention, key rotation, secret refresh |
| `platform/security` | Rate limiter, password hashing, security primitives |
| `platform/signedurl` | Signed URL generation/verification |
| `platform/sms` | SMS provider adapters (SNS, Twilio, Vonage) |
| `platform/telemetry` | OpenTelemetry traces + Prometheus metrics |
| `platform/templates` | Template rendering |
| `platform/valid` | Common validators |

> **Email is SMTP-only.** `platform/email` sends through a single SMTP adapter; there are no
> SES/SendGrid/Mailgun/Postmark providers. SMS, by contrast, still supports multiple adapters
> (SNS, Twilio, Vonage) selected per tenant via `notifier`.

---

## Transport Hosting

**Package:** `internal/server`

`server/` contains only the *transport host* — the chi routers and the gRPC server that aggregate
each domain's routes. Shared response/pagination helpers live under `platform/`, so domain
packages can import them without depending on `server`.

**REST (`rest.go`, `router.go`, `handlers.go`)** runs **two separate HTTP servers**:

| Server | Port | Purpose | Access |
|---|---|---|---|
| Internal | `:8080` | Admin and management endpoints, token introspection | VPN / private network only |
| Public | `:8081` | Login, register, password reset, OAuth 2.0 flows, OIDC discovery | Public internet |

This enforces a hard trust boundary at the network level. `health.go` serves `/health` and
`/ready`; `openapi.go` serves `/openapi.json`.

**gRPC (`grpc.go`, `grpc_interceptors.go`, `grpc_permissions.go`, `grpc_audit_interceptor.go`)**
is the **machine control plane** — the interface an orchestrator (maintainerd core) drives to
provision tenants, services, clients, and users. It is deliberately gated:

- **Opt-in.** The listener binds only when `CONTROL_PLANE_ENABLED=true` (or `GRPC_ENABLED=true`).
  The default deployment is standalone, and the socket is never opened — a caller against a
  disabled instance gets connection-refused, and the reason is logged once at startup.
- **mTLS-gated.** When enabled, the server requires client certificates; interceptors enforce
  per-RPC permissions and write an audit trail for every call.
- **Bind address `:50051`** (`shared.DefaultGRPCAddr`); it drains in-flight RPCs via `GracefulStop`
  on context cancellation.

Registered control-plane services include `SetupService`, `TenantService`/`TenantSettingService`,
the IAM services (`ServiceService`, `APIService`, `PermissionService`, `PolicyService`,
`RoleService`, `AuthorizationService`), `ClientService`, `UserService`/`UserProfileService`,
`WorkloadIdentityFederationService`, and `OAuthIntrospectionService`. Proto definitions live in
`proto/`; generated Go lands in `internal/platform/gen/go/` (never hand-edited). See the
[gRPC control plane](../features/grpc.md) doc for the full surface.

---

## Composition Root

**Package:** `internal/app`

`internal/app` is the single place where the whole graph is assembled:

```
internal/app/
  repositories.go     construct all domain repositories, bound to the DB
  services.go         construct all domain services, injecting repos + platform helpers
  adapters_*.go       satisfy each domain's deps.go consumer interfaces (cross-domain glue)
  issuer_allowlist.go seed the JWT issuer allowlist from config
  webhook_delivery.go wire the webhook dispatcher into the event path
  app.go              the App struct: service interfaces + DB + Redis + cache
  application.go      adapt App into the server.Application the transport host consumes
```

Wiring is hand-rolled and explicit — no `google/wire`, no reflection. The large number of
`adapters_*.go` files is deliberate: each one implements one consumer interface a domain declared
in its `deps.go`, which is exactly what lets domains stay independent while still collaborating.

---

## Multi-Tenancy

Multi-tenancy is a first-class concern built into every layer:

```
Tenant
  ├── Members (users, via tenant membership)
  ├── Roles (tenant-scoped)
  │     └── Permissions
  ├── Clients (OAuth apps)
  ├── Identity Providers
  ├── Services & APIs
  └── Security Settings & IP Rules
```

**How isolation is enforced:**

1. **Middleware** extracts the tenant from the authenticated user's context.
2. **Handlers** pass the tenant ID into every service call.
3. **Services** include `tenantID` in every repository query.
4. **Repositories** scope all data access with `WHERE tenant_id = ?` (joining tenant membership
   where needed).

A user can belong to **multiple tenants** via the `UserIdentity` model, each with separate roles
and permissions. See [multi-tenancy](../features/multi-tenancy.md).

---

## Database & Migrations

**Engine:** PostgreSQL with GORM. **Runner:** `internal/platform/runner/migration.go`.

- Migrations are versioned functions registered in an ordered slice; applied versions are tracked
  in a `schema_migrations` table.
- A PostgreSQL **session-level advisory lock** (key `7316949`) ensures only one pod runs
  migrations at a time across a multi-pod deployment.
- **Rule:** never reorder or delete an existing migration — only append new ones.
- **Seeders** (`internal/platform/runner/seeder.go`) seed the default data the service needs to
  function (system tenant, roles, permissions, system clients).

See [database migrations](./database-migrations.md).

---

## Caching

**Engine:** Redis (`internal/platform/cache`).

| Use Case | Key Pattern | TTL |
|---|---|---|
| User context (middleware) | `user:{sub}:{client_id}` | ~5 minutes |
| Rate limiting | Identifier-based counters | 15-minute window |
| JTI denylist / token state | per-token keys | token lifetime |

> **Note:** if a user's roles or permissions change, the cached context may be stale for up to
> the context TTL.

---

## Platform Boundary

`internal/platform` is for reusable infrastructure that does not import domain packages. Domain
packages may import platform helpers; platform must not import domains.

```text
Allowed dependencies:

  cmd/server
      ├──────────────→ internal/app
      └──────────────→ internal/server

  internal/app
      └──────────────→ internal/<domain>

  internal/server
      └──────────────→ internal/<domain>

  internal/<domain>
      ├──────────────→ internal/platform/*
      └──────────────→ (other domains ONLY via consumer interfaces + app adapters)

Forbidden dependency:

  internal/platform/* ─────X────→ internal/<domain>
```

If a platform helper ever needs a domain type, it belongs in that domain, not in
`internal/platform`. Cross-domain reads never `import` the other domain directly — the consumer
declares a narrow interface in its `deps.go`, and `internal/app` supplies the adapter.

---

## Key Design Patterns

| Pattern | Where | Why |
|---|---|---|
| **Domain-grouped packages** | `internal/<domain>/` | Each domain owns model→service→handler; the top level reads as the product's surface area. |
| **Constructor-based DI** | `internal/app/` | Explicit wiring, no magic, easy to trace. |
| **Owner + consumer interfaces** | `repository_*.go` / `deps.go` | The owner defines its persistence contract; the consumer defines the shape it needs — this kills import cycles. |
| **Cross-domain adapters** | `internal/app/adapters_*.go` | Bind consumer interfaces to upstream services without domain-to-domain imports. |
| **Generic repository** | `internal/platform/database` | Eliminates boilerplate CRUD; domain repos embed it and add queries. |
| **Transaction propagation** | Services → `WithTx()` | Ensures atomicity across multi-entity operations. |
| **DTO boundary** | `types.go` + `validation_*.go` | Decouples API shape from models; sanitizes and validates at the edge. |
| **Middleware composition** | `internal/platform/middleware` | Separates cross-cutting concerns from business logic. |
| **Dual-server split** | `internal/server/rest.go` | Hard network-level separation between admin (`:8080`) and public (`:8081`) APIs. |
| **Opt-in mTLS control plane** | `internal/server/grpc.go` | The machine API is off unless an orchestrator needs it; when on, it is cert-gated and audited. |
| **Advisory-locked migrations** | `internal/platform/runner` | Safe for multi-pod deployments — only one pod migrates at a time. |
| **Transactional outbox** | `internal/event` | Integration events are relayed to RabbitMQ atomically with the state change that produced them. |

---

## Technology Stack

| Category | Technology |
|---|---|
| Language | Go 1.26 |
| Database | PostgreSQL |
| Cache | Redis |
| Message broker | RabbitMQ (integration event plane) |
| ORM | GORM (`gorm.io/gorm`) |
| HTTP router | chi (`github.com/go-chi/chi/v5`) |
| gRPC | `google.golang.org/grpc` + protobuf (opt-in control plane) |
| JWT | `github.com/golang-jwt/jwt/v5` (RSA-256; issuer derived from `APP_PUBLIC_HOSTNAME`) |
| Validation | ozzo-validation (`github.com/go-ozzo/ozzo-validation/v4`) |
| Crypto | `golang.org/x/crypto` (bcrypt cost 12, argon2) |
| UUID | `github.com/google/uuid` |
| Email | SMTP (`gopkg.in/gomail.v2`) |
| SMS | SNS / Twilio / Vonage adapters |
| Observability | OpenTelemetry traces + Prometheus metrics |
| Testing | testify (`github.com/stretchr/testify`), go-sqlmock |
| Container | Docker (multi-stage, single all-in-one binary with embedded web UI) |

### Token & cookie reference

| Token | Default TTL | Notes |
|---|---|---|
| Access token | 15 minutes | API authentication |
| ID token | 1 hour | OIDC user identity claims |
| Refresh token | 7 days | Family-based rotation with reuse detection |
| OAuth refresh token | 7 days (configurable per client) | Opaque; rotated with reuse detection |
| Authorization code | ~10 minutes | Single-use, stored as a SHA-256 hash; PKCE (S256) required |

Session cookies are `HttpOnly` and `Secure`, with `SameSite=Lax` by default (configurable via
`COOKIE_SAMESITE`). `Lax` — not `Strict` — is required so that federated SSO redirects back from
an external IdP still carry the session cookie. JWTs are RSA-256 signed with a minimum 2048-bit
key validated at startup, and the accepted-issuer allowlist is seeded from `APP_PUBLIC_HOSTNAME`.
