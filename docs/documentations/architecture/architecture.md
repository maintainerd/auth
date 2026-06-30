# Target Architecture — Idiomatic Go, Domain-Grouped

This document specifies the **target folder structure and file naming** for maintainerd auth using Go's official module-layout guidance as the baseline, then adding project-specific conventions for a large authentication server. It is informed by Ory Hydra/Kratos, Zitadel, Authelia, Dex, and broader Go ecosystem practice (Kubernetes, Docker, Prometheus, Grafana, Caddy, Gitea, Etcd, Thanos, Cortex). Every existing file is accounted for and mapped to its new home.

Go intentionally does **not** prescribe one universal enterprise folder structure. The official guidance is smaller and more durable:

- Server code that is not meant to be imported by other modules should usually live under `internal/`.
- Executable entrypoints commonly live under `cmd/`.
- A module can contain many packages; each package is a directory.
- Package names and import paths matter more than architectural-layer folders.
- Avoid grab-bag packages such as `util`, `common`, and over-broad buckets.

Everything beyond that is an architectural choice for this project. The choices below are designed to make maintainerd auth readable to Go contributors, easy to refactor before public release, and honest about its domain boundaries.

---

## 1. Guiding Principles

1. **Organize by domain (bounded context), not by architectural layer.** A package owns _everything_ for its domain: service logic, HTTP handlers, storage interface, GORM implementation, types, routes, and tests.
2. **Small, focused packages.** No flat `service/` or `repository/` buckets holding 50+ files.
3. **`internal/` for everything private to this module.** `pkg/` is unused because this server currently exposes no public Go library API. If a package later becomes useful to other projects, split it into a separate module or deliberately promote it outside `internal/`.
4. **`cmd/server` stays thin** — `main.go` only calls `run(context.Context)`; focused bootstrap files handle config/logging/DB/Redis, then delegate to `internal/app`.
5. **Cross-cutting infrastructure is grouped under `internal/platform/`.** Domains sit at the top of `internal/`; infrastructure sits one level deeper. This keeps the top-level reading like a list of *what this service does*, not a list of *how it is built*.
6. **Two distinct interface patterns** — both correct, used in different places:
    - **Owner interface (repository contract):** A domain defines its own persistence interface in `repository.go`. The domain owns the contract because the domain owns the data.
    - **Consumer interface (cross-domain reader):** When `authn` needs to read users, it defines a small `UserReader` interface inside `authn/`, satisfied by an adapter on `user.Service` or `user.Repository`. The *consumer* specifies the shape it needs; this prevents import cycles and accidental coupling.
7. **Each domain owns its GORM models.** No shared `internal/model/` or `internal/platform/model/` package. The `User` struct lives in `internal/user/`, the `OAuthRefreshToken` struct lives in `internal/oauth/`, and so on. This matches Ory Hydra, Kratos, and Dex; a central models package is the [anemic domain model](https://martinfowler.com/bliki/AnemicDomainModel.html) anti-pattern and produces a god package every domain imports.
8. **Cross-aggregate references are IDs, not embedded structs.** When `user.User` needs to reference a tenant, the field is `TenantID uuid.UUID`, not `Tenant *tenant.Tenant`. Navigation across aggregates goes through service calls (`tenant.Service.GetByID(ctx, id)`), not through GORM `Preload`. This is the discipline that makes domain-owned models actually work — it eliminates ~90% of GORM-induced import cycles and is exactly what Kratos and Hydra do.
9. **Last-resort shared types in `internal/shared/`.** If a tiny value type (e.g. an enum, a strongly-typed ID alias) is genuinely needed by 3+ unrelated domains and creates an unavoidable cycle, put it in `internal/shared/`. This is a thin leaf package — no behavior, no imports from other internal packages, and **not a dumping ground**. If `internal/shared/` ever has more than ~5 files, it has become a god package and needs to be broken up.
10. **Composition root in `internal/app/`.** A single `wire.go` assembles all domain registries; split into `wire_<domain>.go` only when a single domain's wiring grows beyond ~30 lines.
11. **Keep domain packages flat until there is real pressure to split.** Flat files inside one package are idiomatic Go and keep cross-file collaboration simple. Allow `internal/<domain>/<subpkg>/` when a domain exceeds ~30 files or ~2500 LoC, or when a sub-area has a genuinely separate API and dependency set (e.g. `internal/oauth/par/`).

---

## 2. File Naming Conventions

These conventions are recommended defaults for consistency. Justified exceptions are allowed and should be noted in the package `doc.go`.

The important Go rule is that **packages are the unit of API design**, not files. File names are for humans and maintainers; package names and exported identifiers are what other code actually sees. This document standardizes file names so a new contributor can predict where things live without turning the convention into a false Go requirement.

### 2.1 General rules

| Rule | Example |
|---|---|
| Lowercase only, `_` separates words | `email_verification.go` |
| `_test.go` suffix marks test files (Go-enforced) | `login_test.go` |
| Avoid stuttering the package name | `user.Service`, not `user.UserService` |
| One dominant concept per file, file named after that concept | `authorize.go` defines `AuthorizeService` |
| Reserve `_<os>.go` / `_<arch>.go` for build constraints only | don't name files `login_darwin.go` for non-build reasons |

### 2.2 Per-domain file vocabulary

Every domain package uses these file names by default.

| File | Contents | Required? | Rationale |
|---|---|---|---|
| **`registry.go`** | `type Deps`, `type Registry`, `func NewRegistry(Deps) *Registry` — the package's wiring container | ✓ always | A small, per-package DI container. Named after Hydra/Kratos `Registry`, but scoped per-package rather than app-wide. |
| **`routes.go`** | `func (r *Registry) RegisterRoutes(rt chi.Router, mw middleware.Set)` | ✓ when domain exposes HTTP | Self-documenting; the transport layer calls it. |
| **`types.go`** | Request/response shapes and other types used across the package | ✓ when types are shared | Used by Authelia, etcd, Prometheus. Go projects do **not** use `dto.go` — DTO is Java terminology. |
| **`<entity>.go`** | GORM struct + entity methods for one aggregate the domain owns: `user.go`, `refresh_token.go`, `auth_code.go` | one per aggregate | Domain-owned models (principle 7). When the entity and the feature share a name (e.g. `invite.go` has both the Invite struct and InviteService), put them in the same file — Kratos style. |
| **`repository.go`** | Storage interface(s), one per aggregate the domain owns | only if the domain has persistence | Owner-interface pattern: the domain defines its persistence contract. |
| **`repository_gorm.go`** | GORM implementations of `repository.go` interfaces | only if the domain has persistence | The `_gorm` suffix flags the framework binding clearly. |
| **`<feature>.go`** | Service logic for one sub-feature: `authorize.go`, `token.go`, `login.go` | ✓ at least one | Standard Go: name files after the dominant concept they contain. May be the same file as `<entity>.go` when feature and entity coincide. |
| **`handler_<feature>.go`** | HTTP handler methods for that sub-feature | one per `<feature>.go` that exposes HTTP | Prefix style sorts handlers together and reads cleanly when a feature is handler-only (e.g. `handler_discovery.go`). |
| **`<feature>_test.go`** | Tests for `<feature>.go` | strongly recommended | Go-enforced naming. |
| **`handler_<feature>_test.go`** | Tests for `handler_<feature>.go` | strongly recommended | Mirrors the source naming. |
| **`errors.go`** | Package-level sentinel errors and custom error types | when there are 3+ errors | Convention in stdlib, Hydra, Kratos, Authelia, Caddy. Define errors so `errors.Is`/`errors.As` work. |
| **`doc.go`** | Package documentation only (`package foo` + doc comment) | optional | Standard Go convention; used in etcd, Cockroach. |
| **`mock_<dep>_test.go`** | Hand-written mocks compiled only in tests | as needed | Hand-written is fine; `moq`/`mockery` are acceptable alternatives. |

**Deliberately omitted from the recommended set:**

- `helpers.go` — same smell as `utils.go`. If you have helpers, name the file after what they do (`scope_match.go`, `token_parse.go`).
- `const.go` — inline constants near use are almost always better. Only create when there are dozens of unrelated constants.
- `interfaces.go` — don't centralize interfaces; define them where they're consumed (consumer pattern) or where they're owned (repository pattern).

> **Terminology note:** `Registry`, `Deps`, and `RegisterRoutes` are maintainerd conventions, not official Go terms. They are chosen because they keep wiring explicit and boring.

### 2.3 Names to avoid

| ❌ Don't use                                     | Why                                                                                                                                                                           | ✓ Use instead                                                      |
| ----------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| `module.go`                                     | NestJS/Angular term, not Go                                                                                                                                                   | `registry.go`                                                      |
| `dto.go`                                        | Java/.NET term, not Go                                                                                                                                                        | `types.go` (or inline in handler file)                             |
| `<feature>_handler.go`                          | Either prefix or suffix works; we pick **prefix** for consistency and because handler-only files (`handler_userinfo.go`) read cleanly without implying a missing service file | `handler_<feature>.go`                                             |
| `interfaces.go`                                 | Don't centralize interfaces; define them at the consumer or with the owner                                                                                                    | n/a                                                                |
| `utils.go` / `common.go` / `helpers.go`         | Almost always indicates code that belongs in a more specific package                                                                                                          | move to a domain-named package or name the file after what it does |
| `models.go` (singular bucket file)              | Don't lump all models in a single file. Use one `<entity>.go` per aggregate.                                                                                                  | per-entity `<entity>.go`                                           |
| `internal/model/` or `internal/platform/model/` | Central models package is an anti-pattern (anemic domain model + god package). 4 of 5 production Go auth projects don't do this.                                              | per-domain ownership (principle 7)                                 |
| `services.go` (as a flat bucket)                | Anti-pattern; the source of the migration                                                                                                                                     | per-domain `<feature>.go` files                                    |

### 2.4 Worked example — `oauth/` with corrected names

```
internal/oauth/
├── registry.go               # type Deps, type Registry, NewRegistry — package wiring
├── routes.go                 # RegisterRoutes(chi.Router, middleware.Set)
├── types.go                  # request/response types used across handlers
├── errors.go                 # ErrInvalidGrant, ErrUnsupportedResponseType, ...
├── repository.go             # AuthCodeRepository, RefreshTokenRepository, ...
├── repository_gorm.go        # GORM implementations
├── authorize.go              # AuthorizeService — /authorize service logic
├── handler_authorize.go      # HTTP handler for /authorize
├── authorize_test.go         # tests for authorize.go
├── handler_authorize_test.go # tests for handler_authorize.go
├── token.go                  # TokenService — /token service logic
├── handler_token.go          # HTTP handler for /token
├── token_test.go             # tests for token.go
├── consent.go                # ConsentService — consent service logic
├── handler_consent.go        # HTTP handler for consent
├── par.go                    # PARService — pushed authorization requests
├── handler_par.go            # HTTP handler for PAR
├── device.go                 # DeviceService — device authorization flow
├── handler_device.go         # HTTP handler for device flow
├── ciba.go                   # CIBAService — client-initiated backchannel auth
├── handler_ciba.go           # HTTP handler for CIBA
├── token_exchange.go         # TokenExchangeService — token exchange flow
├── handler_token_exchange.go # HTTP handler for token exchange
├── session.go                # OAuth session service logic
├── handler_session.go        # HTTP handler for OAuth sessions
├── register.go               # Dynamic Client Registration (RFC 7591)
├── handler_register.go       # HTTP handler for dynamic client registration
├── scope_match.go            # named helper, not "helpers.go"
├── handler_userinfo.go       # handler-only (no separate service file)
└── handler_discovery.go      # handler-only — /.well-known/openid-configuration, JWKS
```

Note one nice property of the prefix style: handler-only files like `userinfo` and `discovery` get a single clean name (`handler_userinfo.go`) without suggesting a missing service file.

---

## 3. Top-Level Layout

```
maintainerd-auth/
├── cmd/                            # executable entrypoints
│   └── server/                     # server binary entrypoint package
│       └── main.go                 # thin entrypoint
├── internal/                       # private application code
│   ├── app/                        # composition root (per-domain wiring)
│   ├── server/                     # transport hosting (REST + gRPC) — calls each domain's RegisterRoutes
│   ├── oauth/                      # OAuth 2.1 / OIDC flows
│   ├── user/                       # user, profile, settings, account
│   ├── authn/                      # login, register, password, magic link, sms login, session
│   ├── mfa/                        # TOTP, WebAuthn, backup codes
│   ├── tenant/                     # tenants, members, settings
│   ├── iam/                        # services (resources), APIs, permissions, roles, policies
│   ├── client/                     # OAuth clients, API keys
│   ├── idp/                        # identity providers, federation, registration flows
│   ├── invite/                     # invitations
│   ├── secpolicy/                  # security settings, IP restrictions
│   ├── branding/                   # branding + email/sms/login templates
│   ├── notifier/                   # outbound email/SMS provider configs (per-tenant)
│   ├── authevent/                  # auth event log + retention
│   ├── webhook/                    # outbound webhook endpoints + delivery
│   ├── setup/                      # bootstrap / initial setup
│   ├── shared/                     # last-resort shared value types (see principle 9) — empty by default
│   └── platform/                   # cross-cutting infrastructure
│       ├── apperror/               # app/domain errors + transport mapping
│       ├── cache/                  # Redis-backed cache abstraction
│       ├── config/                 # env loading and runtime config
│       ├── cookie/                 # secure cookie helpers
│       ├── crypto/                 # hashing, encryption, OTP, PKCE helpers
│       ├── database/               # DB setup, migrations, base repository helpers
│       ├── dpop/                   # DPoP proof validation
│       ├── email/                  # email provider adapters (Mailgun, SES, SMTP, ...)
│       ├── gen/                    # generated code (proto, etc.)
│       ├── jwt/                    # JWT/JWK signing and verification
│       ├── logging/                # slog setup and PII redaction
│       ├── middleware/             # shared HTTP middleware
│       ├── pagination/             # shared pagination helpers (REST + gRPC)
│       ├── ptr/                    # pointer helpers
│       ├── response/               # shared HTTP response helpers (JSON, error mapping)
│       ├── runner/                 # background workers (retention, etc.)
│       ├── security/               # rate-limiting and security primitives
│       ├── signedurl/              # signed URL generation/verification
│       ├── sms/                    # SMS provider adapters (Twilio, SNS, Vonage)
│       ├── telemetry/              # OpenTelemetry and Prometheus wiring
│       ├── templates/              # template rendering
│       └── valid/                  # common validators
├── proto/                          # protobuf definitions
├── migrations/                     # SQL schema migrations (see §11)
├── docs/                           # project documentation
├── tests/                          # black-box and multi-package tests
│   ├── e2e/                        # end-to-end tests
│   └── integration/                # integration tests
├── scripts/                        # developer and CI helper scripts
├── nginx/                          # local reverse-proxy config
├── go.mod                          # Go module definition
├── go.sum                          # Go module checksums
├── Dockerfile                      # production container build
├── docker-compose.yml              # local development stack
├── Makefile                        # common development commands
└── README.md                       # public project overview
```

> **Why `internal/platform/` (and not flat under `internal/`)?** With ~22 cross-cutting packages, putting them at the same level as 15 domains makes the top of `internal/` unreadable. Nesting them under `platform/` signals "this is infrastructure, not a business domain" and lets the top level read as the system's surface area.

---

## 4. Domain Packages — Full Listing

Every target tree below includes `#` comments so it can be used during the refactor without cross-referencing the mapping tables constantly:

- `new:` means the file is introduced by this architecture.
- `was:` means the file or behavior moves from the current location.
- `existing:` means the file already lives in that package and stays there.
- Model + service comments such as `model/user.go + service/user.go` mean the new file intentionally co-locates the GORM aggregate and the main domain behavior.

### 4.1 `internal/oauth/` — OAuth 2.1 / OIDC

Handles every OAuth flow plus discovery and JWKS.

```
oauth/
├── registry.go                     # new: type Deps, Registry, NewRegistry for OAuth wiring
├── routes.go                       # was: internal/rest/route/oauth.go
├── types.go                        # was: internal/dto/oauth.go
├── errors.go                       # new or moved: OAuth sentinel errors and error mapping helpers
├── scope_match.go                  # was: rest/handler/oauth_helpers.go (renamed by purpose)
├── repository.go                   # interfaces for auth_code, refresh_token, par, device, ciba, consent_grant, consent_challenge
├── repository_gorm.go              # was: internal/repository/oauth_*.go implementations
├── auth_code.go                    # OAuthAuthorizationCode model — was: model/oauth_authorization_code.go
├── refresh_token.go                # OAuthRefreshToken model — was: model/oauth_refresh_token.go
├── consent_challenge.go            # OAuthConsentChallenge model — was: model/oauth_consent_challenge.go
├── consent_grant.go                # OAuthConsentGrant model — was: model/oauth_consent_grant.go
├── authorize.go                    # AuthorizeService — was: service/oauth_authorize.go
├── handler_authorize.go            # was: rest/handler/oauth_authorize.go
├── authorize_test.go               # was: service/oauth_authorize_test.go, if present
├── token.go                        # TokenService — was: service/oauth_token.go
├── handler_token.go                # was: rest/handler/oauth_token.go
├── token_test.go                   # was: service/oauth_token_test.go, if present
├── consent.go                      # ConsentService — was: service/oauth_consent.go
├── handler_consent.go              # was: rest/handler/oauth_consent.go
├── consent_test.go                 # was: service/oauth_consent_test.go, if present
├── par.go                          # OAuthPARRequest model + PARService — was: service/oauth_par.go
├── handler_par.go                  # was: rest/handler/oauth_par.go
├── device.go                       # OAuthDeviceCode model + DeviceService — was: service/oauth_device.go
├── handler_device.go               # was: rest/handler/oauth_device.go
├── ciba.go                         # OAuthCIBARequest model + CIBAService — was: service/oauth_ciba.go
├── handler_ciba.go                 # was: rest/handler/oauth_ciba.go
├── token_exchange.go               # was: service/oauth_token_exchange.go
├── handler_token_exchange.go       # was: rest/handler/oauth_token_exchange.go
├── session.go                      # was: service/oauth_session.go
├── handler_session.go              # was: rest/handler/oauth_session.go
├── register.go                     # was: service/oauth_register.go     (DCR)
├── handler_register.go             # was: rest/handler/oauth_register.go
├── handler_userinfo.go             # was: rest/handler/oauth_userinfo.go
└── handler_discovery.go            # was: rest/handler/oauth_discovery.go
```

**Owns aggregates (GORM models):** `OAuthAuthorizationCode`, `OAuthCIBARequest`, `OAuthConsentChallenge`, `OAuthConsentGrant`, `OAuthDeviceCode`, `OAuthPARRequest`, `OAuthRefreshToken`. Cross-aggregate references (`ClientID`, `UserID`) are `uuid.UUID` — never embedded structs.

---

### 4.2 `internal/user/` — User accounts, profile, settings

```
user/
├── registry.go                     # new: type Deps, Registry, NewRegistry for user wiring
├── routes.go                       # was: rest/route/user.go, profile.go, user_setting.go, account.go
├── types.go                        # was: dto/user.go, dto/profile.go, dto/user_setting.go, dto/account.go
├── repository.go                   # interfaces for user, user_identity, user_role, user_password_history, user_token, profile, user_setting
├── repository_gorm.go              # was: repository/user*.go, profile.go, user_setting.go
├── user.go                         # User model + UserService — was: model/user.go + service/user.go
├── handler_user.go                 # was: rest/handler/user.go
├── user_test.go                    # was: service/user_test.go, if present
├── user_identity.go                # UserIdentity model — was: model/user_identity.go
├── user_role.go                    # UserRole model — was: model/user_role.go
├── user_password_history.go        # UserPasswordHistory model — was: model/user_password_history.go
├── user_token.go                   # UserToken model — was: model/user_token.go
├── profile.go                      # Profile model + ProfileService — was: model/profile.go + service/profile.go
├── handler_profile.go              # was: rest/handler/profile.go
├── setting.go                      # UserSetting model + SettingService — was: model/user_setting.go + service/user_setting.go
├── handler_setting.go              # was: rest/handler/user_setting.go
├── account.go                      # AccountService — was: service/account.go (self-service: delete, export, recovery codes)
└── handler_account.go              # was: rest/handler/account.go
```

**Owns aggregates (GORM models):** `User`, `UserIdentity`, `UserRole`, `UserPasswordHistory`, `UserToken`, `Profile`, `UserSetting`. Cross-aggregate references (`TenantID`, `IdentityProviderID`) are `uuid.UUID`.

---

### 4.3 `internal/authn/` — Authentication flows

Everything a user does to prove identity in a session.

```
authn/
├── registry.go                     # new: type Deps, Registry, NewRegistry for authn wiring
├── routes.go                       # was: rest/route/login.go, register.go, password/reset/magic/session routes
├── types.go                        # was: dto/login.go, register.go, forgot_password.go, reset_password.go, magic_link.go, email_verification.go, sms_login.go, session.go
├── deps.go                         # consumer-side interfaces: UserReader, ClientReader, ...
├── login.go                        # was: service/login.go
├── handler_login.go                # was: rest/handler/login.go
├── login_test.go                   # was: service/login_test.go, if present
├── register.go                     # was: service/register.go
├── handler_register.go             # was: rest/handler/register.go
├── register_test.go                # was: service/register_test.go, if present
├── forgot_password.go              # was: service/forgot_password.go
├── handler_forgot_password.go      # was: rest/handler/forgot_password.go
├── reset_password.go               # was: service/reset_password.go
├── handler_reset_password.go       # was: rest/handler/reset_password.go
├── password_policy.go              # was: service/password_policy.go
├── email_verification.go           # was: service/email_verification.go
├── handler_email_verification.go   # was: rest/handler/email_verification.go
├── magic_link.go                   # was: service/magic_link.go
├── handler_magic_link.go           # was: rest/handler/magic_link.go
├── sms_login.go                    # was: service/sms_login.go
├── handler_sms_login.go            # was: rest/handler/sms_login.go
├── session.go                      # was: service/session.go
└── handler_session.go              # (currently embedded in other handlers; consolidate)
```

**Note:** `authn` depends on `user` for user reads/writes and on `client` for client validation. Those dependencies are **consumer-defined interfaces** in `deps.go` (e.g. `UserReader`, `ClientReader`), implemented in the upstream domain package via small adapters.

**No aggregates of its own** — operates on `UserToken` (owned by `user`), `SMSOTP` (owned by `notifier`), etc., via consumer-defined interfaces.

---

### 4.4 `internal/mfa/` — Multi-factor authentication

```
mfa/
├── registry.go                     # new: type Deps, Registry, NewRegistry for MFA wiring
├── routes.go                       # was: rest/route/mfa.go
├── types.go                        # was: dto/mfa.go
├── repository.go                   # interfaces for totp_secret, webauthn_credential, backup_code
├── repository_gorm.go              # was: repository/user_totp_secret.go, user_webauthn_credential.go, user_backup_code.go
├── mfa.go                          # MFAService — was: service/mfa.go (TOTP + backup codes orchestration)
├── handler_mfa.go                  # was: rest/handler/mfa.go
├── mfa_test.go                     # was: service/mfa_test.go, if present
├── totp.go                         # UserTOTPSecret model + TOTP logic — was: model/user_totp_secret.go
├── webauthn.go                     # UserWebAuthnCredential model + WebAuthn logic — was: model/user_webauthn_credential.go
└── backup_codes.go                 # UserBackupCode model + backup-code logic — was: model/user_backup_code.go
```

**Owns aggregates (GORM models):** `UserTOTPSecret`, `UserWebAuthnCredential`, `UserBackupCode`. `UserID` is `uuid.UUID`.

---

### 4.5 `internal/tenant/` — Multi-tenancy

```
tenant/
├── registry.go                     # new: type Deps, Registry, NewRegistry for tenant wiring
├── routes.go                       # was: rest/route/tenant.go, tenant_setting.go
├── types.go                        # was: dto/tenant.go, tenant_member.go, tenant_setting.go
├── repository.go                   # interfaces for tenant, tenant_member, tenant_service, tenant_setting
├── repository_gorm.go              # was: repository/tenant*.go
├── tenant.go                       # Tenant model + TenantService — was: model/tenant.go + service/tenant.go
├── handler_tenant.go               # was: rest/handler/tenant.go
├── tenant_test.go                  # was: service/tenant_test.go, if present
├── member.go                       # TenantMember model + MemberService — was: model/tenant_member.go + service/tenant_member.go
├── handler_member.go               # (currently in tenant.go handler — split out)
├── tenant_service.go               # TenantServiceLink model — was: model/tenant_service.go
├── setting.go                      # TenantSetting model + SettingService — was: model/tenant_setting.go + service/tenant_setting.go
├── handler_setting.go              # was: rest/handler/tenant_setting.go
├── access.go                       # was: service/tenant_access.go
└── isolation_test.go               # was: service/tenant_isolation_test.go
```

**Owns aggregates (GORM models):** `Tenant`, `TenantMember`, `TenantService`, `TenantSetting`. Note: rename the `TenantService` model to `TenantServiceLink` (or similar) to avoid collision with the `TenantService` Go service interface — pivot/association tables benefit from a `Link`/`Association` suffix.

---

### 4.6 `internal/iam/` — Identity & Access Management resources

The "what can be accessed" half of authorization: services (resources), APIs, permissions, roles, policies.

```
iam/
├── registry.go                     # new: type Deps, Registry, NewRegistry for IAM wiring
├── routes.go                       # was: rest/route/service.go, api.go, permission.go, role.go, policy.go
├── types.go                        # was: dto/service.go, api.go, permission.go, role.go, policy.go
├── repository.go                   # interfaces for service, api, permission, role, role_permission, policy, service_policy
├── repository_gorm.go              # was: repository/service.go, api.go, permission.go, role.go, policy.go, pivot repositories
├── service.go                      # Service model + ServiceLogic — was: model/service.go + service/service.go
├── handler_service.go              # was: rest/handler/service.go
├── api.go                          # API model + APIService — was: model/api.go + service/api.go
├── handler_api.go                  # was: rest/handler/api.go
├── permission.go                   # Permission model + PermissionService — was: model/permission.go + service/permission.go
├── api_permission.go               # APIPermission pivot model — was: model/api_permission.go
├── handler_permission.go           # was: rest/handler/permission.go
├── role.go                         # Role model + RoleService — was: model/role.go + service/role.go
├── role_permission.go              # RolePermission pivot model — was: model/role_permission.go
├── handler_role.go                 # was: rest/handler/role.go
├── policy.go                       # Policy model + PolicyService — was: model/policy.go + service/policy.go
├── service_policy.go               # ServicePolicy pivot model — was: model/service_policy.go
└── handler_policy.go               # was: rest/handler/policy.go
```

**Owns aggregates (GORM models):** `Service`, `API`, `Permission`, `APIPermission`, `Role`, `RolePermission`, `Policy`, `ServicePolicy`. Cross-aggregate references are `uuid.UUID`.

---

### 4.7 `internal/client/` — OAuth clients and API keys

```
client/
├── registry.go                     # new: type Deps, Registry, NewRegistry for client wiring
├── routes.go                       # was: rest/route/client.go, api_key.go
├── types.go                        # was: dto/client.go, api_key.go
├── repository.go                   # interfaces for client, client_uri, client_api, client_permission, api_key, api_key_api, api_key_permission
├── repository_gorm.go              # was: repository/client*.go, api_key*.go
├── client.go                       # Client model + ClientService — was: model/client.go + service/client.go
├── client_uri.go                   # ClientURI model — was: model/client_uri.go
├── client_api.go                   # ClientAPI pivot model — was: model/client_api.go
├── client_permission.go            # ClientPermission pivot model — was: model/client_permission.go
├── handler_client.go               # was: rest/handler/client.go
├── client_test.go                  # was: service/client_test.go, if present
├── api_key.go                      # APIKey model + APIKeyService — was: model/api_key.go + service/api_key.go
├── api_key_api.go                  # APIKeyAPI pivot model — was: model/api_key_api.go
├── api_key_permission.go           # APIKeyPermission pivot model — was: model/api_key_permission.go
├── handler_api_key.go              # was: rest/handler/api_key.go
└── api_key_test.go                 # was: service/api_key_test.go, if present
```

**Owns aggregates (GORM models):** `Client`, `ClientURI`, `ClientAPI`, `ClientPermission`, `APIKey`, `APIKeyAPI`, `APIKeyPermission`. Cross-aggregate references are `uuid.UUID`.

---

### 4.8 `internal/idp/` — Identity providers, federation, registration flows

```
idp/
├── registry.go                     # new: type Deps, Registry, NewRegistry for IDP wiring
├── routes.go                       # was: rest/route/identity_provider.go, federation.go, signup_flow.go
├── types.go                        # was: dto/idp.go, federation.go, signup_flow.go, signup_flow_role.go
├── repository.go                   # interfaces for identity_provider, registration_flow, registration_flow_role
├── repository_gorm.go              # was: repository/identity_provider.go, signup_flow.go, signup_flow_role.go
├── provider.go                     # IdentityProvider model + ProviderService — was: model/identity_provider.go + service/identity_provider.go
├── handler_provider.go             # was: rest/handler/identity_provider.go
├── federation.go                   # Federation model/service — was: model/federation.go + service/federation.go
├── handler_federation.go           # was: rest/handler/federation.go
├── registration_flow.go            # RegistrationFlow model + RegistrationFlowService — was: model/signup_flow.go + service/signup_flow.go
├── registration_flow_role.go       # RegistrationFlowRole pivot model — was: model/signup_flow_role.go
└── handler_registration_flow.go    # was: rest/handler/signup_flow.go
```

**Owns aggregates (GORM models):** `IdentityProvider`, `Federation`, `RegistrationFlow`, `RegistrationFlowRole`. Cross-aggregate references are `uuid.UUID`.

---

### 4.9 `internal/invite/` — Invitations

```
invite/
├── registry.go                     # new: type Deps, Registry, NewRegistry for invite wiring
├── routes.go                       # was: rest/route/invite.go
├── types.go                        # was: dto/invite.go
├── repository.go                   # interfaces for invite, invite_role
├── repository_gorm.go              # was: repository/invite.go, invite_role.go
├── invite.go                       # Invite model + InviteService — was: model/invite.go + service/invite.go
├── invite_role.go                  # InviteRole pivot model — was: model/invite_role.go
├── handler_invite.go               # was: rest/handler/invite.go
└── invite_test.go                  # was: service/invite_test.go, if present
```

**Owns aggregates (GORM models):** `Invite`, `InviteRole`. `RoleID` and `TenantID` are `uuid.UUID`.

---

### 4.10 `internal/secpolicy/` — Security settings + IP restrictions

```
secpolicy/
├── registry.go                     # new: type Deps, Registry, NewRegistry for security-policy wiring
├── routes.go                       # was: rest/route/security_setting.go, ip_restriction_rule.go
├── types.go                        # was: dto/security_setting.go, ip_restriction_rule.go
├── repository.go                   # interfaces for security_setting, security_settings_audit, ip_restriction_rule
├── repository_gorm.go              # was: repository/security_setting.go, security_settings_audit.go, ip_restriction_rule.go
├── setting.go                      # SecuritySetting model + SettingService — was: model/security_setting.go + service/security_setting.go
├── settings_audit.go               # SecuritySettingsAudit model — was: model/security_settings_audit.go
├── handler_setting.go              # was: rest/handler/security_setting.go
├── ip_restriction.go               # IPRestrictionRule model + service — was: model/ip_restriction_rule.go + service/ip_restriction_rule.go
└── handler_ip_restriction.go       # was: rest/handler/ip_restriction_rule.go
```

**Owns aggregates (GORM models):** `SecuritySetting`, `SecuritySettingsAudit`, `IPRestrictionRule`. `TenantID` is `uuid.UUID`.

> **Naming note:** Don't call this package `security/` — that collides with the platform `security/` package (rate-limiting primitives). `secpolicy/` (security policy) is unambiguous.

---

### 4.11 `internal/branding/` — Branding + templates

```
branding/
├── registry.go                     # new: type Deps, Registry, NewRegistry for branding wiring
├── routes.go                       # was: rest/route/branding.go, email_template.go, sms_template.go, login_template.go
├── types.go                        # was: dto/branding.go, email_template.go, sms_template.go, login_template.go
├── repository.go                   # interfaces for branding, email_template, sms_template, login_template
├── repository_gorm.go              # was: repository/branding.go, email_template.go, sms_template.go, login_template.go
├── branding.go                     # Branding model + BrandingService — was: model/branding.go + service/branding.go
├── handler_branding.go             # was: rest/handler/branding.go
├── email_template.go               # EmailTemplate model + service — was: model/email_template.go + service/email_template.go
├── handler_email_template.go       # was: rest/handler/email_template.go
├── sms_template.go                 # SMSTemplate model + service — was: model/sms_template.go + service/sms_template.go
├── handler_sms_template.go         # was: rest/handler/sms_template.go
├── login_template.go               # LoginTemplate model + service — was: model/login_template.go + service/login_template.go
└── handler_login_template.go       # was: rest/handler/login_template.go
```

**Owns aggregates (GORM models):** `Branding`, `EmailTemplate`, `SMSTemplate`, `LoginTemplate`. `TenantID` is `uuid.UUID`.

---

### 4.12 `internal/notifier/` — Outbound delivery configuration

Configuration for _how_ the system sends email and SMS (per-tenant provider, credentials, sender info). The provider _adapters_ themselves live in `internal/platform/email/` and `internal/platform/sms/`.

```
notifier/
├── registry.go                     # new: type Deps, Registry, NewRegistry for notifier wiring
├── routes.go                       # was: rest/route/email_config.go, sms_config.go
├── types.go                        # was: dto/email_config.go, sms_config.go
├── repository.go                   # interfaces for email_config, sms_config, sms_otp
├── repository_gorm.go              # was: repository/email_config.go, sms_config.go, sms_otp.go
├── email_config.go                 # EmailConfig model + service — was: model/email_config.go + service/email_config.go
├── handler_email_config.go         # was: rest/handler/email_config.go
├── sms_config.go                   # SMSConfig model + service — was: model/sms_config.go + service/sms_config.go
├── handler_sms_config.go           # was: rest/handler/sms_config.go
└── sms_otp.go                      # SMSOTP model — was: model/sms_otp.go
```

**Owns aggregates (GORM models):** `EmailConfig`, `SMSConfig`, `SMSOTP`. `TenantID` is `uuid.UUID`.

> **Why `notifier` and not `channel`?** "channel" overloads with Go's `chan` keyword and is too generic. "notifier" reads as "the thing that sends notifications," which is what this package configures.

---

### 4.13 `internal/authevent/` — Auth event log

```
authevent/
├── registry.go                     # new: type Deps, Registry, NewRegistry for auth-event wiring
├── routes.go                       # was: rest/route/auth_event.go
├── types.go                        # was: dto/auth_event.go
├── repository.go                   # AuthEventRepository interface
├── repository_gorm.go              # was: repository/auth_event.go
├── event.go                        # AuthEvent model + EventService — was: model/auth_event.go + service/auth_event.go
├── handler_event.go                # was: rest/handler/auth_event.go
└── retention.go                    # was: runner/StartRetentionRunner (auth-event specific bits)
```

**Owns aggregates (GORM models):** `AuthEvent`. `UserID`, `TenantID`, `ClientID` are `uuid.UUID`.

> **Why `authevent` and not `event`?** "event" is too generic — when you eventually add a domain event bus (publish login events to webhook + analytics + audit), it deserves the name `eventbus/` or `event/`. Reserve the obvious name now and call this package what it actually is: the auth event log.

---

### 4.14 `internal/webhook/` — Outbound webhooks

Today this package only has the delivery mechanism. We promote it to a full domain by adding endpoint CRUD here.

```
webhook/
├── registry.go                     # new: type Deps, Registry, NewRegistry for webhook wiring
├── routes.go                       # was: rest/route/webhook_endpoint.go
├── types.go                        # was: dto/webhook_endpoint.go
├── repository.go                   # WebhookEndpointRepository interface
├── repository_gorm.go              # was: repository/webhook_endpoint.go
├── endpoint.go                     # WebhookEndpoint model + EndpointService — was: model/webhook_endpoint.go + service/webhook_endpoint.go
├── handler_endpoint.go             # was: rest/handler/webhook_endpoint.go
├── dispatcher.go                   # existing: internal/webhook/dispatcher.go
├── deliver.go                      # existing: internal/webhook/deliver.go
├── payload.go                      # existing: internal/webhook/payload.go
└── signer.go                       # existing: internal/webhook/signer.go
```

**Owns aggregates (GORM models):** `WebhookEndpoint`. `TenantID` is `uuid.UUID`.

---

### 4.15 `internal/setup/` — Bootstrap

Kept as a domain because there is a real long-lived `/setup` HTTP surface (first-run admin creation, system IDP/client provisioning). If that surface is later removed in favor of CLI-only bootstrap, demote this package to `internal/app/bootstrap.go`.

```
setup/
├── registry.go                     # new: type Deps, Registry, NewRegistry for setup wiring
├── routes.go                       # was: rest/route/setup.go
├── types.go                        # was: dto/setup.go
├── setup.go                        # was: service/setup.go
├── handler_setup.go                # was: rest/handler/setup.go
└── grpc.go                         # was: grpc/handler/seeder.go
```

**No models of its own.**

---

## 5. Platform Packages (Cross-Cutting Infrastructure)

All cross-cutting infrastructure lives under `internal/platform/`. Domains import from `platform/`; `platform/` never imports from a domain.

| Package | Purpose | Source today |
|---|---|---|
| `internal/platform/apperror/` | Domain error types + HTTP mapping | `internal/apperror/` |
| `internal/platform/cache/` | Redis-backed cache abstraction + JTI denylist | `internal/cache/` |
| `internal/platform/config/` | Env loading, DB init, Redis client | `internal/config/` |
| `internal/platform/cookie/` | Secure cookie helpers | `internal/cookie/` |
| `internal/platform/crypto/` | Hashing, encryption helpers | `internal/crypto/` |
| `internal/platform/database/` | DB hardening, pool tuning, base repository helpers | `internal/database/` + `internal/repository/base*.go` |
| `internal/platform/dpop/` | OAuth DPoP proof validation (RFC 9449) | `internal/dpop/` |
| `internal/platform/email/` | Email provider adapters (Mailgun, Postmark, Resend, SendGrid, SES, SMTP) | `internal/email/` |
| `internal/platform/gen/` | Generated code (proto) | `internal/gen/` |
| `internal/platform/jwt/` | JWT signing/verification, JWK set | `internal/jwt/` |
| `internal/platform/logging/` | slog setup + PII redaction handler | `internal/logging/` |
| `internal/platform/middleware/` | HTTP middleware (auth, CORS, CSRF, rate limit, logging, permissions, sessions, user, security headers, content-type) | `internal/middleware/` |
| `internal/platform/pagination/` | Shared pagination helpers (REST + gRPC) | `internal/dto/pagination.go` + `internal/rest/handler/query.go` |
| `internal/platform/ptr/` | Pointer helpers | `internal/ptr/` |
| `internal/platform/response/` | Shared HTTP response helpers (JSON encoding, error mapping) | `internal/rest/response/` |
| `internal/platform/runner/` | Background workers (retention, migrations) | `internal/runner/` |
| `internal/platform/security/` | Rate limiter init, security primitives | `internal/security/` |
| `internal/platform/signedurl/` | Signed URL generation/verification | `internal/signedurl/` |
| `internal/platform/sms/` | SMS provider adapters (SNS, Twilio, Vonage) | `internal/sms/` |
| `internal/platform/telemetry/` | OTEL traces + Prometheus metrics | `internal/telemetry/` |
| `internal/platform/templates/` | Template rendering | `internal/templates/` |
| `internal/platform/valid/` | Common validators | `internal/valid/` |

---

## 6. Transport Hosting

```
internal/server/
├── rest.go                          # was: internal/rest/server/server.go
├── grpc.go                          # was: internal/grpc/server/server.go
└── openapi.go                       # was: internal/rest/handler/openapi.go
```

`server/` only contains the *transport host* — the chi router and gRPC server that aggregate each domain's `RegisterRoutes`. Shared helpers (`response/`, `pagination/`) live under `platform/`, not here, so that domain packages can import them without depending on `server`.

`rest.go` builds the chi router by calling each domain's `RegisterRoutes(...)`:

```go
func StartRESTServer(app *app.App) {
    r := chi.NewRouter()
    r.Use(middleware.Logging, middleware.RequestID, ...)

    app.OAuth.RegisterRoutes(r, app.Middleware)
    app.User.RegisterRoutes(r, app.Middleware)
    app.Authn.RegisterRoutes(r, app.Middleware)
    app.MFA.RegisterRoutes(r, app.Middleware)
    // ... one line per domain registry
}
```

`grpc.go` does the same for the gRPC server.

> **Where is `Middleware` defined?** The `middleware.Set` type (the bundle of middleware passed to `RegisterRoutes`) lives in `internal/platform/middleware/`. Every domain imports it; the package is the natural shared dependency.

---

## 7. Composition Root

```
internal/app/
├── app.go                          # type App struct { ... per-registry fields }
└── wire.go                         # all domain wiring in one file
```

`wire.go` contains one constructor per domain (`newOAuthRegistry`, `newUserRegistry`, ...). If a single constructor grows beyond ~30 lines, **only then** split that domain into its own `wire_<domain>.go`. Premature splitting (16 tiny files for 16 tiny constructors) hurts more than it helps — putting the assembly graph on one screen is more valuable than per-domain file locality for wiring code that changes rarely.

`app.App` holds one field per registry, _not_ per service:

```go
type App struct {
    DB    *gorm.DB
    Cache *cache.Cache

    Middleware *middleware.Set

    OAuth     *oauth.Registry
    User      *user.Registry
    Authn     *authn.Registry
    MFA       *mfa.Registry
    Tenant    *tenant.Registry
    IAM       *iam.Registry
    Client    *client.Registry
    IDP       *idp.Registry
    Invite    *invite.Registry
    SecPolicy *secpolicy.Registry
    Branding  *branding.Registry
    Notifier  *notifier.Registry
    AuthEvent *authevent.Registry
    Webhook   *webhook.Registry
    Setup     *setup.Registry
}
```

Each constructor in `wire.go` is short and explicit:

```go
// wire.go
func newOAuthRegistry(db *gorm.DB, c *cache.Cache, ev event.Publisher, usr user.Reader, cli client.Reader) *oauth.Registry {
    return oauth.NewRegistry(oauth.Deps{
        DB:      db,
        Cache:   c,
        Events:  ev,
        Users:   usr,
        Clients: cli,
    })
}
```

> **Why hand-rolled wiring (not `google/wire`)?** With ~16 domains and explicit `Deps` structs, hand-written constructors fit on one screen and are obvious to a new reader. `google/wire` becomes valuable around 50+ providers or when graph mistakes start being subtle. Revisit if the dependency graph grows.

This replaces today's [internal/app/services.go](../internal/app/services.go) (125 lines, 45 constructor calls) with one focused file containing per-domain constructors.

---

## 8. Complete Migration Mapping

Every existing file → new location.

### 8.1 Services → domains

| Current file | New location |
|---|---|
| `internal/service/account.go` | `internal/user/account.go` |
| `internal/service/api.go` | `internal/iam/api.go` |
| `internal/service/api_key.go` | `internal/client/api_key.go` |
| `internal/service/auth_event.go` | `internal/authevent/event.go` |
| `internal/service/branding.go` | `internal/branding/branding.go` |
| `internal/service/client.go` | `internal/client/client.go` |
| `internal/service/email_config.go` | `internal/notifier/email_config.go` |
| `internal/service/email_template.go` | `internal/branding/email_template.go` |
| `internal/service/email_verification.go` | `internal/authn/email_verification.go` |
| `internal/service/federation.go` | `internal/idp/federation.go` |
| `internal/service/forgot_password.go` | `internal/authn/forgot_password.go` |
| `internal/service/identity_provider.go` | `internal/idp/provider.go` |
| `internal/service/invite.go` | `internal/invite/invite.go` |
| `internal/service/ip_restriction_rule.go` | `internal/secpolicy/ip_restriction.go` |
| `internal/service/login.go` | `internal/authn/login.go` |
| `internal/service/login_template.go` | `internal/branding/login_template.go` |
| `internal/service/magic_link.go` | `internal/authn/magic_link.go` |
| `internal/service/mfa.go` | `internal/mfa/mfa.go` |
| `internal/service/oauth_authorize.go` | `internal/oauth/authorize.go` |
| `internal/service/oauth_ciba.go` | `internal/oauth/ciba.go` |
| `internal/service/oauth_consent.go` | `internal/oauth/consent.go` |
| `internal/service/oauth_device.go` | `internal/oauth/device.go` |
| `internal/service/oauth_par.go` | `internal/oauth/par.go` |
| `internal/service/oauth_register.go` | `internal/oauth/register.go` |
| `internal/service/oauth_session.go` | `internal/oauth/session.go` |
| `internal/service/oauth_token.go` | `internal/oauth/token.go` |
| `internal/service/oauth_token_exchange.go` | `internal/oauth/token_exchange.go` |
| `internal/service/password_policy.go` | `internal/authn/password_policy.go` |
| `internal/service/permission.go` | `internal/iam/permission.go` |
| `internal/service/policy.go` | `internal/iam/policy.go` |
| `internal/service/profile.go` | `internal/user/profile.go` |
| `internal/service/register.go` | `internal/authn/register.go` |
| `internal/service/reset_password.go` | `internal/authn/reset_password.go` |
| `internal/service/role.go` | `internal/iam/role.go` |
| `internal/service/security_setting.go` | `internal/secpolicy/setting.go` |
| `internal/service/service.go` | `internal/iam/service.go` |
| `internal/service/session.go` | `internal/authn/session.go` |
| `internal/service/setup.go` | `internal/setup/setup.go` |
| `internal/service/signup_flow.go` | `internal/idp/registration_flow.go` |
| `internal/service/sms_config.go` | `internal/notifier/sms_config.go` |
| `internal/service/sms_login.go` | `internal/authn/sms_login.go` |
| `internal/service/sms_template.go` | `internal/branding/sms_template.go` |
| `internal/service/tenant.go` | `internal/tenant/tenant.go` |
| `internal/service/tenant_access.go` | `internal/tenant/access.go` |
| `internal/service/tenant_member.go` | `internal/tenant/member.go` |
| `internal/service/tenant_setting.go` | `internal/tenant/setting.go` |
| `internal/service/user.go` | `internal/user/user.go` |
| `internal/service/user_compat.go` | `internal/user/profile_compat.go` (temporary compatibility helpers; delete after API callers use canonical fields) |
| `internal/service/user_setting.go`* | `internal/user/setting.go` |
| `internal/service/webauthn.go` | `internal/mfa/webauthn.go` |
| `internal/service/webhook_endpoint.go` | `internal/webhook/endpoint.go` |

\* If a standalone file exists; otherwise lives inside `profile.go` today.

### 8.2 Handlers → domains

| Current file | New location |
|---|---|
| `internal/rest/handler/account.go` | `internal/user/handler_account.go` |
| `internal/rest/handler/api.go` | `internal/iam/handler_api.go` |
| `internal/rest/handler/api_key.go` | `internal/client/handler_api_key.go` |
| `internal/rest/handler/auth_event.go` | `internal/authevent/handler_event.go` |
| `internal/rest/handler/branding.go` | `internal/branding/handler_branding.go` |
| `internal/rest/handler/client.go` | `internal/client/handler_client.go` |
| `internal/rest/handler/email_config.go` | `internal/notifier/handler_email_config.go` |
| `internal/rest/handler/email_template.go` | `internal/branding/handler_email_template.go` |
| `internal/rest/handler/email_verification.go` | `internal/authn/handler_email_verification.go` |
| `internal/rest/handler/federation.go` | `internal/idp/handler_federation.go` |
| `internal/rest/handler/forgot_password.go` | `internal/authn/handler_forgot_password.go` |
| `internal/rest/handler/identity_provider.go` | `internal/idp/handler_provider.go` |
| `internal/rest/handler/invite.go` | `internal/invite/handler_invite.go` |
| `internal/rest/handler/ip_restriction_rule.go` | `internal/secpolicy/handler_ip_restriction.go` |
| `internal/rest/handler/login.go` | `internal/authn/handler_login.go` |
| `internal/rest/handler/login_template.go` | `internal/branding/handler_login_template.go` |
| `internal/rest/handler/magic_link.go` | `internal/authn/handler_magic_link.go` |
| `internal/rest/handler/mfa.go` | `internal/mfa/handler_mfa.go` |
| `internal/rest/handler/oauth_authorize.go` | `internal/oauth/handler_authorize.go` |
| `internal/rest/handler/oauth_ciba.go` | `internal/oauth/handler_ciba.go` |
| `internal/rest/handler/oauth_consent.go` | `internal/oauth/handler_consent.go` |
| `internal/rest/handler/oauth_device.go` | `internal/oauth/handler_device.go` |
| `internal/rest/handler/oauth_discovery.go` | `internal/oauth/handler_discovery.go` |
| `internal/rest/handler/oauth_helpers.go` | `internal/oauth/scope_match.go` (or split by purpose) |
| `internal/rest/handler/oauth_par.go` | `internal/oauth/handler_par.go` |
| `internal/rest/handler/oauth_register.go` | `internal/oauth/handler_register.go` |
| `internal/rest/handler/oauth_session.go` | `internal/oauth/handler_session.go` |
| `internal/rest/handler/oauth_token.go` | `internal/oauth/handler_token.go` |
| `internal/rest/handler/oauth_token_exchange.go` | `internal/oauth/handler_token_exchange.go` |
| `internal/rest/handler/oauth_userinfo.go` | `internal/oauth/handler_userinfo.go` |
| `internal/rest/handler/openapi.go` | `internal/server/openapi.go` |
| `internal/rest/handler/permission.go` | `internal/iam/handler_permission.go` |
| `internal/rest/handler/policy.go` | `internal/iam/handler_policy.go` |
| `internal/rest/handler/profile.go` | `internal/user/handler_profile.go` |
| `internal/rest/handler/query.go` | `internal/platform/pagination/query.go` |
| `internal/rest/handler/register.go` | `internal/authn/handler_register.go` |
| `internal/rest/handler/reset_password.go` | `internal/authn/handler_reset_password.go` |
| `internal/rest/handler/role.go` | `internal/iam/handler_role.go` |
| `internal/rest/handler/security_setting.go` | `internal/secpolicy/handler_setting.go` |
| `internal/rest/handler/service.go` | `internal/iam/handler_service.go` |
| `internal/rest/handler/setup.go` | `internal/setup/handler_setup.go` |
| `internal/rest/handler/signup_flow.go` | `internal/idp/handler_registration_flow.go` |
| `internal/rest/handler/sms_config.go` | `internal/notifier/handler_sms_config.go` |
| `internal/rest/handler/sms_login.go` | `internal/authn/handler_sms_login.go` |
| `internal/rest/handler/sms_template.go` | `internal/branding/handler_sms_template.go` |
| `internal/rest/handler/tenant.go` | `internal/tenant/handler_tenant.go` |
| `internal/rest/handler/tenant_setting.go` | `internal/tenant/handler_setting.go` |
| `internal/rest/handler/user.go` | `internal/user/handler_user.go` |
| `internal/rest/handler/user_setting.go` | `internal/user/handler_setting.go` |
| `internal/rest/handler/webhook_endpoint.go` | `internal/webhook/handler_endpoint.go` |

### 8.3 Routes → domains (all merged into each domain's `routes.go`)

Each file in `internal/rest/route/*.go` becomes part of the matching domain's `routes.go`:

| Current file | New location |
|---|---|
| `internal/rest/route/account.go` | merged into `internal/user/routes.go` |
| `internal/rest/route/api.go` | merged into `internal/iam/routes.go` |
| `internal/rest/route/api_key.go` | merged into `internal/client/routes.go` |
| `internal/rest/route/auth_event.go` | `internal/authevent/routes.go` |
| `internal/rest/route/branding.go` | merged into `internal/branding/routes.go` |
| `internal/rest/route/client.go` | merged into `internal/client/routes.go` |
| `internal/rest/route/email_config.go` | merged into `internal/notifier/routes.go` |
| `internal/rest/route/email_template.go` | merged into `internal/branding/routes.go` |
| `internal/rest/route/email_verification.go` | merged into `internal/authn/routes.go` |
| `internal/rest/route/federation.go` | merged into `internal/idp/routes.go` |
| `internal/rest/route/forgot_password.go` | merged into `internal/authn/routes.go` |
| `internal/rest/route/identity_provider.go` | merged into `internal/idp/routes.go` |
| `internal/rest/route/invite.go` | `internal/invite/routes.go` |
| `internal/rest/route/ip_restriction_rule.go` | merged into `internal/secpolicy/routes.go` |
| `internal/rest/route/login.go` | merged into `internal/authn/routes.go` |
| `internal/rest/route/login_template.go` | merged into `internal/branding/routes.go` |
| `internal/rest/route/magic_link.go` | merged into `internal/authn/routes.go` |
| `internal/rest/route/mfa.go` | `internal/mfa/routes.go` |
| `internal/rest/route/oauth.go` | `internal/oauth/routes.go` |
| `internal/rest/route/permission.go` | merged into `internal/iam/routes.go` |
| `internal/rest/route/policy.go` | merged into `internal/iam/routes.go` |
| `internal/rest/route/profile.go` | merged into `internal/user/routes.go` |
| `internal/rest/route/register.go` | merged into `internal/authn/routes.go` |
| `internal/rest/route/reset_password.go` | merged into `internal/authn/routes.go` |
| `internal/rest/route/role.go` | merged into `internal/iam/routes.go` |
| `internal/rest/route/security_setting.go` | merged into `internal/secpolicy/routes.go` |
| `internal/rest/route/service.go` | merged into `internal/iam/routes.go` |
| `internal/rest/route/setup.go` | `internal/setup/routes.go` |
| `internal/rest/route/signup_flow.go` | merged into `internal/idp/routes.go` |
| `internal/rest/route/sms_config.go` | merged into `internal/notifier/routes.go` |
| `internal/rest/route/sms_login.go` | merged into `internal/authn/routes.go` |
| `internal/rest/route/sms_template.go` | merged into `internal/branding/routes.go` |
| `internal/rest/route/tenant.go` | merged into `internal/tenant/routes.go` |
| `internal/rest/route/tenant_setting.go` | merged into `internal/tenant/routes.go` |
| `internal/rest/route/user.go` | merged into `internal/user/routes.go` |
| `internal/rest/route/user_setting.go` | merged into `internal/user/routes.go` |
| `internal/rest/route/webhook_endpoint.go` | merged into `internal/webhook/routes.go` |

### 8.4 Repositories → domains (interface + GORM impl in each domain)

| Current file | New location |
|---|---|
| `internal/repository/api.go` | `internal/iam/repository_gorm.go` |
| `internal/repository/api_key.go` | `internal/client/repository_gorm.go` |
| `internal/repository/api_key_api.go` | `internal/client/repository_gorm.go` |
| `internal/repository/api_key_permission.go` | `internal/client/repository_gorm.go` |
| `internal/repository/auth_event.go` | `internal/authevent/repository_gorm.go` |
| `internal/repository/base.go` | `internal/platform/database/base_repository.go` |
| `internal/repository/base_interface.go` | `internal/platform/database/base_repository.go` |
| `internal/repository/branding.go` | `internal/branding/repository_gorm.go` |
| `internal/repository/client.go` | `internal/client/repository_gorm.go` |
| `internal/repository/client_api.go` | `internal/client/repository_gorm.go` |
| `internal/repository/client_permission.go` | `internal/client/repository_gorm.go` |
| `internal/repository/client_uri.go` | `internal/client/repository_gorm.go` |
| `internal/repository/email_config.go` | `internal/notifier/repository_gorm.go` |
| `internal/repository/email_template.go` | `internal/branding/repository_gorm.go` |
| `internal/repository/identity_provider.go` | `internal/idp/repository_gorm.go` |
| `internal/repository/invite.go` | `internal/invite/repository_gorm.go` |
| `internal/repository/invite_role.go` | `internal/invite/repository_gorm.go` |
| `internal/repository/ip_restriction_rule.go` | `internal/secpolicy/repository_gorm.go` |
| `internal/repository/login_template.go` | `internal/branding/repository_gorm.go` |
| `internal/repository/oauth_authorization_code.go` | `internal/oauth/repository_gorm.go` |
| `internal/repository/oauth_ciba_request.go` | `internal/oauth/repository_gorm.go` |
| `internal/repository/oauth_consent_challenge.go` | `internal/oauth/repository_gorm.go` |
| `internal/repository/oauth_consent_grant.go` | `internal/oauth/repository_gorm.go` |
| `internal/repository/oauth_device_code.go` | `internal/oauth/repository_gorm.go` |
| `internal/repository/oauth_par_request.go` | `internal/oauth/repository_gorm.go` |
| `internal/repository/oauth_refresh_token.go` | `internal/oauth/repository_gorm.go` |
| `internal/repository/permission.go` | `internal/iam/repository_gorm.go` |
| `internal/repository/policy.go` | `internal/iam/repository_gorm.go` |
| `internal/repository/profile.go` | `internal/user/repository_gorm.go` |
| `internal/repository/role.go` | `internal/iam/repository_gorm.go` |
| `internal/repository/role_permission.go` | `internal/iam/repository_gorm.go` |
| `internal/repository/security_setting.go` | `internal/secpolicy/repository_gorm.go` |
| `internal/repository/security_settings_audit.go` | `internal/secpolicy/repository_gorm.go` |
| `internal/repository/service.go` | `internal/iam/repository_gorm.go` |
| `internal/repository/service_policy.go` | `internal/iam/repository_gorm.go` |
| `internal/repository/signup_flow.go` | `internal/idp/repository_gorm.go` |
| `internal/repository/signup_flow_role.go` | `internal/idp/repository_gorm.go` |
| `internal/repository/sms_config.go` | `internal/notifier/repository_gorm.go` |
| `internal/repository/sms_otp.go` | `internal/notifier/repository_gorm.go` |
| `internal/repository/sms_template.go` | `internal/branding/repository_gorm.go` |
| `internal/repository/tenant.go` | `internal/tenant/repository_gorm.go` |
| `internal/repository/tenant_member.go` | `internal/tenant/repository_gorm.go` |
| `internal/repository/tenant_service.go` | `internal/tenant/repository_gorm.go` |
| `internal/repository/tenant_setting.go` | `internal/tenant/repository_gorm.go` |
| `internal/repository/user.go` | `internal/user/repository_gorm.go` |
| `internal/repository/user_backup_code.go` | `internal/mfa/repository_gorm.go` |
| `internal/repository/user_identity.go` | `internal/user/repository_gorm.go` |
| `internal/repository/user_password_history.go` | `internal/user/repository_gorm.go` |
| `internal/repository/user_role.go` | `internal/user/repository_gorm.go` |
| `internal/repository/user_setting.go` | `internal/user/repository_gorm.go` |
| `internal/repository/user_token.go` | `internal/user/repository_gorm.go` |
| `internal/repository/user_totp_secret.go` | `internal/mfa/repository_gorm.go` |
| `internal/repository/user_webauthn_credential.go` | `internal/mfa/repository_gorm.go` |
| `internal/repository/webhook_endpoint.go` | `internal/webhook/repository_gorm.go` |

> **Note on splitting `repository_gorm.go`:** If a domain owns more than ~5 tables (e.g. `oauth`, `iam`, `user`, `client`), split into `repository_<table>.go` files instead of one big file. Use judgment — readability first.

### 8.5 Types (was DTOs) → domains

| Current file | New location |
|---|---|
| `internal/dto/account.go` | merged into `internal/user/types.go` |
| `internal/dto/api.go` | merged into `internal/iam/types.go` |
| `internal/dto/api_key.go` | merged into `internal/client/types.go` |
| `internal/dto/auth_event.go` | merged into `internal/authevent/types.go` |
| `internal/dto/branding.go` | merged into `internal/branding/types.go` |
| `internal/dto/client.go` | merged into `internal/client/types.go` |
| `internal/dto/date.go` | `internal/platform/valid/date.go` |
| `internal/dto/email_config.go` | merged into `internal/notifier/types.go` |
| `internal/dto/email_template.go` | merged into `internal/branding/types.go` |
| `internal/dto/email_verification.go` | merged into `internal/authn/types.go` |
| `internal/dto/federation.go` | merged into `internal/idp/types.go` |
| `internal/dto/forgot_password.go` | merged into `internal/authn/types.go` |
| `internal/dto/idp.go` | merged into `internal/idp/types.go` |
| `internal/dto/invite.go` | merged into `internal/invite/types.go` |
| `internal/dto/ip_restriction_rule.go` | merged into `internal/secpolicy/types.go` |
| `internal/dto/login.go` | merged into `internal/authn/types.go` |
| `internal/dto/login_template.go` | merged into `internal/branding/types.go` |
| `internal/dto/magic_link.go` | merged into `internal/authn/types.go` |
| `internal/dto/mfa.go` | merged into `internal/mfa/types.go` |
| `internal/dto/oauth.go` | merged into `internal/oauth/types.go` |
| `internal/dto/pagination.go` | `internal/platform/pagination/pagination.go` |
| `internal/dto/permission.go` | merged into `internal/iam/types.go` |
| `internal/dto/policy.go` | merged into `internal/iam/types.go` |
| `internal/dto/profile.go` | merged into `internal/user/types.go` |
| `internal/dto/register.go` | merged into `internal/authn/types.go` |
| `internal/dto/reset_password.go` | merged into `internal/authn/types.go` |
| `internal/dto/role.go` | merged into `internal/iam/types.go` |
| `internal/dto/security_setting.go` | merged into `internal/secpolicy/types.go` |
| `internal/dto/service.go` | merged into `internal/iam/types.go` |
| `internal/dto/session.go` | merged into `internal/authn/types.go` |
| `internal/dto/setup.go` | merged into `internal/setup/types.go` |
| `internal/dto/signup_flow.go` | merged into `internal/idp/types.go` |
| `internal/dto/signup_flow_role.go` | merged into `internal/idp/types.go` |
| `internal/dto/sms_config.go` | merged into `internal/notifier/types.go` |
| `internal/dto/sms_login.go` | merged into `internal/authn/types.go` |
| `internal/dto/sms_template.go` | merged into `internal/branding/types.go` |
| `internal/dto/tenant.go` | merged into `internal/tenant/types.go` |
| `internal/dto/tenant_member.go` | merged into `internal/tenant/types.go` |
| `internal/dto/tenant_setting.go` | merged into `internal/tenant/types.go` |
| `internal/dto/user.go` | merged into `internal/user/types.go` |
| `internal/dto/user_setting.go` | merged into `internal/user/types.go` |
| `internal/dto/webhook_endpoint.go` | merged into `internal/webhook/types.go` |

### 8.6 Models — move per-domain (one file per aggregate)

Each model file in `internal/model/` moves into the domain that owns the aggregate. The GORM struct definition lives alongside the service that operates on it. **Cross-aggregate foreign keys become `uuid.UUID` fields, not embedded structs** — this is the work that breaks the cycles a central package was hiding.

| Current file | New location |
|---|---|
| `internal/model/user.go` | `internal/user/user.go` |
| `internal/model/user_identity.go` | `internal/user/user_identity.go` |
| `internal/model/user_role.go` | `internal/user/user_role.go` |
| `internal/model/user_password_history.go` | `internal/user/user_password_history.go` |
| `internal/model/user_token.go` | `internal/user/user_token.go` |
| `internal/model/profile.go` | `internal/user/profile.go` |
| `internal/model/user_setting.go` | `internal/user/setting.go` |
| `internal/model/user_totp_secret.go` | `internal/mfa/totp.go` |
| `internal/model/user_webauthn_credential.go` | `internal/mfa/webauthn.go` |
| `internal/model/user_backup_code.go` | `internal/mfa/backup_codes.go` |
| `internal/model/tenant.go` | `internal/tenant/tenant.go` |
| `internal/model/tenant_member.go` | `internal/tenant/member.go` |
| `internal/model/tenant_service.go` | `internal/tenant/tenant_service.go` (rename type to `TenantServiceLink`) |
| `internal/model/tenant_setting.go` | `internal/tenant/setting.go` |
| `internal/model/service.go` | `internal/iam/service.go` |
| `internal/model/api.go` | `internal/iam/api.go` |
| `internal/model/permission.go` | `internal/iam/permission.go` |
| `internal/model/api_permission.go` | `internal/iam/api_permission.go` |
| `internal/model/role.go` | `internal/iam/role.go` |
| `internal/model/role_permission.go` | `internal/iam/role_permission.go` |
| `internal/model/policy.go` | `internal/iam/policy.go` |
| `internal/model/service_policy.go` | `internal/iam/service_policy.go` |
| `internal/model/client.go` | `internal/client/client.go` |
| `internal/model/client_uri.go` | `internal/client/client_uri.go` |
| `internal/model/client_api.go` | `internal/client/client_api.go` |
| `internal/model/client_permission.go` | `internal/client/client_permission.go` |
| `internal/model/api_key.go` | `internal/client/api_key.go` |
| `internal/model/api_key_api.go` | `internal/client/api_key_api.go` |
| `internal/model/api_key_permission.go` | `internal/client/api_key_permission.go` |
| `internal/model/identity_provider.go` | `internal/idp/provider.go` |
| `internal/model/federation.go` | `internal/idp/federation.go` |
| `internal/model/signup_flow.go` | `internal/idp/registration_flow.go` |
| `internal/model/signup_flow_role.go` | `internal/idp/registration_flow_role.go` |
| `internal/model/invite.go` | `internal/invite/invite.go` |
| `internal/model/invite_role.go` | `internal/invite/invite_role.go` |
| `internal/model/security_setting.go` | `internal/secpolicy/setting.go` |
| `internal/model/security_settings_audit.go` | `internal/secpolicy/settings_audit.go` |
| `internal/model/ip_restriction_rule.go` | `internal/secpolicy/ip_restriction.go` |
| `internal/model/branding.go` | `internal/branding/branding.go` |
| `internal/model/email_template.go` | `internal/branding/email_template.go` |
| `internal/model/sms_template.go` | `internal/branding/sms_template.go` |
| `internal/model/login_template.go` | `internal/branding/login_template.go` |
| `internal/model/email_config.go` | `internal/notifier/email_config.go` |
| `internal/model/sms_config.go` | `internal/notifier/sms_config.go` |
| `internal/model/sms_otp.go` | `internal/notifier/sms_otp.go` |
| `internal/model/auth_event.go` | `internal/authevent/event.go` |
| `internal/model/webhook_endpoint.go` | `internal/webhook/endpoint.go` |
| `internal/model/oauth_authorization_code.go` | `internal/oauth/auth_code.go` |
| `internal/model/oauth_refresh_token.go` | `internal/oauth/refresh_token.go` |
| `internal/model/oauth_par_request.go` | `internal/oauth/par.go` |
| `internal/model/oauth_device_code.go` | `internal/oauth/device.go` |
| `internal/model/oauth_ciba_request.go` | `internal/oauth/ciba.go` |
| `internal/model/oauth_consent_challenge.go` | `internal/oauth/consent_challenge.go` |
| `internal/model/oauth_consent_grant.go` | `internal/oauth/consent_grant.go` |
| `internal/model/constants.go` | split into owning domains (`internal/user/constants.go`, `internal/client/constants.go`, `internal/iam/constants.go`, `internal/idp/constants.go`, `internal/secpolicy/constants.go`, `internal/branding/constants.go`); keep only truly cross-domain values in `internal/shared/` |

> **The work this surfaces.** Existing models that use `Preload` or embedded sibling structs (e.g. `User.Tenant *Tenant`) will need to be rewritten as ID-only fields (`User.TenantID uuid.UUID`) with cross-aggregate reads going through services. This is real work and is the part that pays for itself long-term — central models hide this coupling rather than eliminate it.

### 8.7 App wiring

| Current file | New location |
|---|---|
| `internal/app/app.go` | `internal/app/app.go` (reduced — registries instead of services) |
| `internal/app/services.go` | **deleted** — replaced by `wire.go` |
| `internal/app/repositories.go` | **deleted** — each domain owns its repositories |

### 8.8 Transport hosting

| Current file | New location |
|---|---|
| `internal/rest/server/server.go` | `internal/server/rest.go` |
| `internal/rest/response/*` | `internal/platform/response/` |
| `internal/grpc/server/server.go` | `internal/server/grpc.go` |
| `internal/grpc/handler/seeder.go` | `internal/setup/grpc.go` |

### 8.9 Database, seeders, and runners

The current database code is Go-based. Keep it under platform during this refactor so schema and seed behavior remain covered. If the project later switches to SQL files at repo-root `migrations/`, do that as a separate migration-system change, not as part of the package-layout refactor.

| Current file/location | New location |
|---|---|
| `internal/database/migration/*.go` | `internal/platform/database/migration/` |
| `internal/database/seeder/*.go` | `internal/platform/database/seeder/` |
| `internal/runner/migration.go` | `internal/platform/runner/migration.go` |
| `internal/runner/seeder.go` | `internal/platform/runner/seeder.go` |
| `internal/runner/key_rotation.go` | `internal/platform/runner/key_rotation.go` |
| `internal/runner/secret_refresh.go` | `internal/platform/runner/secret_refresh.go` |
| `internal/runner/retention.go` | `internal/authevent/retention.go` |
| `internal/runner/retention_test.go` | `internal/authevent/retention_test.go` |

### 8.10 Tests and test helpers

Tests move with the code they verify. This is how we make sure no feature disappears during the refactor.

| Current file pattern | New location |
|---|---|
| `internal/service/*_test.go` | matching domain package next to the moved service file |
| `internal/service/mock_repos_test.go` | split into `mock_<dep>_test.go` files in the consuming domain packages |
| `internal/service/mock_auth_event_test.go` | move to domains that need auth-event test doubles, or replace with `authevent` test adapter |
| `internal/rest/handler/*_test.go` | matching domain package next to `handler_<feature>.go` |
| `internal/rest/handler/mock_services_test.go` | split into per-domain `mock_<dep>_test.go` files |
| `internal/rest/handler/testhelpers_test.go` | move reusable HTTP test helpers to `internal/platform/response` or duplicate locally if only one domain needs them |
| `internal/dto/*_test.go` | matching domain package next to `types.go`; `pagination_test.go` moves to `internal/platform/pagination/`; `date_test.go` moves to `internal/platform/valid/` |
| `internal/rest/response/response_test.go` | `internal/platform/response/response_test.go` |
| `internal/grpc/handler/seeder_test.go` | `internal/setup/grpc_test.go` |

### 8.11 Platform package moves (cross-cutting packages)

| Current location | New location |
|---|---|
| `internal/apperror/` | `internal/platform/apperror/` |
| `internal/cache/` | `internal/platform/cache/` |
| `internal/config/` | `internal/platform/config/` |
| `internal/cookie/` | `internal/platform/cookie/` |
| `internal/crypto/` | `internal/platform/crypto/` |
| `internal/database/` | `internal/platform/database/` |
| `internal/dpop/` | `internal/platform/dpop/` |
| `internal/email/` | `internal/platform/email/` |
| `internal/gen/` | `internal/platform/gen/` |
| `internal/jwt/` | `internal/platform/jwt/` |
| `internal/logging/` | `internal/platform/logging/` |
| `internal/middleware/` | `internal/platform/middleware/` |
| `internal/ptr/` | `internal/platform/ptr/` |
| `internal/runner/` | `internal/platform/runner/` |
| `internal/security/` | `internal/platform/security/` |
| `internal/signedurl/` | `internal/platform/signedurl/` |
| `internal/sms/` | `internal/platform/sms/` |
| `internal/telemetry/` | `internal/platform/telemetry/` |
| `internal/templates/` | `internal/platform/templates/` |
| `internal/valid/` | `internal/platform/valid/` |

### 8.12 Things that stay exactly where they are

```
cmd/server/
proto/
docs/
tests/
scripts/
nginx/
```

### 8.13 Coverage audit

Current scan baseline:

- `go list ./...` discovers **35 packages**.
- The repo currently has **578 Go files** across `cmd/`, `internal/`, `docs/`, and `tests/`.
- High-risk legacy buckets covered by this plan:
    - `internal/service/`: 89 files
    - `internal/model/`: 56 files
    - `internal/repository/`: 55 files
    - `internal/dto/`: 75 files
    - `internal/rest/handler/`: 87 files
    - `internal/rest/route/`: 37 files
    - `internal/runner/`: 6 files
    - `internal/database/migration/`: 56 files
    - `internal/database/seeder/`: 11 files

When this document was last audited, every current Go file was covered either by an exact mapping row, a package move row, or a test pattern row. Re-run this check before starting the migration and after each large edit.

---

## 9. Worked Example — A Tiny Domain in Full

To make this concrete, here is what the smallest domain (`invite/`) looks like end-to-end:

```go
// internal/invite/registry.go
package invite

import (
    "github.com/maintainerd/auth/internal/branding"
    "github.com/maintainerd/auth/internal/client"
    "github.com/maintainerd/auth/internal/iam"
    "github.com/maintainerd/auth/internal/platform/middleware"
    "gorm.io/gorm"
)

type Deps struct {
    DB        *gorm.DB
    Templates branding.EmailTemplateReader
    Clients   client.Reader
    Roles     iam.RoleReader
}

type Registry struct {
    Service Service
    Handler *Handler
}

func NewRegistry(d Deps) *Registry {
    repo := newGormRepository(d.DB)
    svc := newService(d.DB, repo, d.Clients, d.Roles, d.Templates)
    return &Registry{
        Service: svc,
        Handler: newHandler(svc),
    }
}
```

```go
// internal/invite/repository.go
package invite

type Repository interface {
    Create(ctx context.Context, inv *Invite) error
    GetByToken(ctx context.Context, token string) (*Invite, error)
    // ...
}
```

```go
// internal/invite/routes.go
package invite

import (
    "github.com/go-chi/chi/v5"
    "github.com/maintainerd/auth/internal/platform/middleware"
)

func (r *Registry) RegisterRoutes(rt chi.Router, mw *middleware.Set) {
    rt.Route("/invites", func(rt chi.Router) {
        rt.With(mw.RequireAuth).Post("/", r.Handler.Create)
        rt.With(mw.RequireAuth).Get("/", r.Handler.List)
        rt.Get("/{token}", r.Handler.GetByToken)
        rt.Post("/{token}/accept", r.Handler.Accept)
    })
}
```

```go
// internal/invite/types.go
package invite

type CreateInviteRequest struct {
    Email     string   `json:"email"`
    RoleUUIDs []string `json:"role_uuids"`
    ExpiresIn int      `json:"expires_in_seconds"`
}

type InviteResponse struct {
    UUID      string    `json:"uuid"`
    Email     string    `json:"email"`
    Token     string    `json:"token"`
    ExpiresAt time.Time `json:"expires_at"`
}
```

```go
// internal/invite/handler_invite.go
package invite

import (
    "net/http"

    "github.com/maintainerd/auth/internal/platform/response"
)

type Handler struct {
    svc Service
}

func newHandler(svc Service) *Handler {
    return &Handler{svc: svc}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateInviteRequest
    if err := response.DecodeJSON(r, &req); err != nil {
        response.Error(w, err)
        return
    }
    inv, err := h.svc.Create(r.Context(), req)
    if err != nil {
        response.Error(w, err)
        return
    }
    response.JSON(w, http.StatusCreated, inv)
}
```

Every domain follows this exact shape.

---

## 10. What Changes Behaviorally

**Nothing user-visible.** This is a pure structural refactor: every HTTP route returns the same response, every service has the same behavior, every test still passes against the same external contract. The user-visible API surface is identical.

What changes is:
- Where code lives.
- How packages depend on each other (explicit, via consumer-defined interfaces).
- How easy it is to find related code (everything for "invites" is in one folder).
- How extractable a domain becomes if it ever needs to split into a separate service.
- **Test surfaces shift.** Mocks that lived in `internal/repository/` now live alongside the consuming code. Tests that imported `github.com/.../internal/repository` need their imports rewritten. The behavior is unchanged; the import paths are not.

---

## 11. Operational Policies & Migration Plan

### 11.1 Policies

- **Naming is recommended, not dogma.** Follow these conventions by default. If a package needs a different file for clarity, document the exception in the package `doc.go` and call it out in the PR.
- **Models ownership:** Each domain owns its GORM models. There is no central `internal/model/` package. Cross-aggregate references are `uuid.UUID` IDs, never embedded structs — navigation across aggregates goes through service calls. This eliminates the cycles a central package was hiding.
- **`internal/shared/` is a sealed package.** Adding a type here requires a justification in the PR description: which 3+ unrelated domains need it and why it cannot live in any of them. Reject anything behavioral (helpers, services, methods beyond stringer/marshalling).
- **Scoped subpackages allowed** when a domain exceeds ~30 files or ~2500 LoC (e.g. `internal/oauth/par/`).
- **Interfaces and cycles** — define consumer-facing interfaces in the consuming package; define owner repository interfaces in the owning package. Example:

```go
// in internal/authn/deps.go
//
// Consumer-defined interface. Returns a small projection type that authn
// declares — NOT user.User — so the dependency is one-way (user does not
// know authn exists; authn does not import user).
type UserReader interface {
        GetByID(ctx context.Context, id uuid.UUID) (UserSnapshot, error)
}

type UserSnapshot struct {
        ID       uuid.UUID
        Email    string
        TenantID uuid.UUID
}

// in internal/user/adapters.go
//
// The user package provides a small adapter that maps user.User → authn.UserSnapshot.
// Wire injects this adapter into authn.NewRegistry as the UserReader dependency.
type AuthnUserReader struct{ Repo Repository }

func (a AuthnUserReader) GetByID(ctx context.Context, id uuid.UUID) (authn.UserSnapshot, error) {
        u, err := a.Repo.GetByID(ctx, id)
        if err != nil {
                return authn.UserSnapshot{}, err
        }
        return authn.UserSnapshot{ID: u.ID, Email: u.Email, TenantID: u.TenantID}, nil
}
```

- **Generated code:** Keep generated artifacts under `internal/platform/gen/` or `proto/` and mark them with `// Code generated` headers. Do not hand-edit generated files.
- **Schema migrations** live in `migrations/` at the repo root as numbered SQL files (e.g. `0001_init.sql`, `0002_add_webhooks.sql`). `gorm.AutoMigrate` is acceptable for local development only; production migrations always run through the SQL files via the runner.
- **CI / PR gates:**
    - `go build ./...`
    - `gofmt` / `goimports` applied
    - `golangci-lint run ./...` (zero issues)
    - Unit tests for touched packages: `go test ./... -count=1 -coverprofile=coverage.out`
    - Integration / e2e tests as applicable

### 11.2 Recommended migration order

Move **one domain at a time, end-to-end**. Within a single PR, move every layer of one domain (handler, route, service, repository, types, **GORM model**) and update its imports. This avoids the broken-intermediate-state problem of moving "all handlers, then all services" — at every commit the build is green and tests pass.

Before the first migration PR, take a baseline:

- `go test ./... -count=1`
- `go test ./... -count=1 -coverprofile=coverage.out` if coverage is already expected to pass locally
- `go list ./...` to confirm package discovery
- A route inventory from the current REST and gRPC servers, so route behavior can be compared after each domain move

Because the app is not deployed yet, this migration can be aggressive about package moves and type renames. Still keep every commit buildable where possible; Go's compiler is the migration assistant here.

Per-domain PR checklist:
- Move handler, route, service, repository, types, GORM struct, tests into `internal/<domain>/`.
- Rewrite cross-aggregate references in the GORM struct from embedded structs (`User.Tenant *Tenant`) to ID-only (`User.TenantID uuid.UUID`).
- Replace any `Preload("Tenant")` chains in this domain's services with explicit service calls (`tenant.Service.GetByID(ctx, ...)`).
- Define consumer interfaces in `deps.go` for upstream dependencies; provide adapters in the upstream domain.
- Build + test; no behavior change visible at the HTTP / gRPC surface.

1. **Bootstrap (one small PR).** Create `internal/platform/` and move the cross-cutting packages with no behavior changes (`git mv`). Update import paths. Build + test.
2. **`internal/server/`, `internal/app/wire.go`, `app.App` skeleton (one small PR).** Set up the transport host and the empty composition root. The old `services.go` / `repositories.go` and `internal/model/` still work in parallel until each domain is migrated.
3. **One domain per PR.** Recommended order (leaf domains first, so consumers don't have to update twice):
    - `invite/`, `secpolicy/`, `notifier/`, `branding/`, `webhook/`, `authevent/`, `mfa/`
    - `tenant/`, `idp/`, `client/`, `iam/`, `user/`
    - `authn/` (depends on `user/`, `client/`)
    - `oauth/` (depends on `user/`, `client/`, `authn/`)
    - `setup/` last
4. **Cleanup PR.** Delete `internal/service/`, `internal/repository/`, `internal/dto/`, `internal/model/`, `internal/rest/`, `internal/grpc/`. Run `gofmt`, `goimports`, full test suite.

Use `git mv` everywhere to preserve history.

> **Heads up on the model rewrite.** This is the largest single piece of work in the migration. Any `Preload` chain that crosses an aggregate boundary (e.g. `db.Preload("Tenant.Members").Find(&users)`) becomes two queries with a service call in between. This is real work, not a renaming. Budget for it — and let the type checker drive it: once you remove the embedded struct field, the compiler will surface every call site that needs to change.

---

## 12. References

Naming conventions in this document were informed by the following Go projects:

**Auth-domain projects:**
- [Ory Hydra](https://github.com/ory/hydra) — `registry.go`, `handler.go`, `handler_<feature>.go`, `manager.go`. Note that Hydra's `Registry` is a single app-wide interface, not a per-package container — we adapted the *name* but scoped it per-package.
- [Ory Kratos](https://github.com/ory/kratos) — similar shape to Hydra.
- [Zitadel](https://github.com/zitadel/zitadel) — nested `model/` and `repository/` sub-packages (alternate style; we don't follow this).
- [Authelia](https://github.com/authelia/authelia) — `types.go`, `errors.go`, `handler_<feature>.go`, `store.go`.
- [Dex](https://github.com/dexidp/dex) — feature-grouped flat packages.

**General Go ecosystem:**
- Kubernetes — `scheduler.go`, `eventhandlers.go`, `<concept>.go` style
- Docker (Moby) — `container_routes.go`, concept-named files
- Prometheus — `interface.go`, `errors.go`, concept-named files
- Caddy — `errors.go`, `matchers.go`, concept-named files
- Etcd — `interface.go`, `doc.go`, `server.go`
- Gitea — concept-named files (`oauth2.go`, `webauthn.go`, `session.go`)
- Cockroach — `doc.go`, descriptive concept files

**Official guidance:**
- [Organizing a Go module](https://go.dev/doc/modules/layout) — official guidance for small, command, package, and server-project layouts. This is the main official source for `cmd/`, `internal/`, and avoiding unnecessary public packages.
- [Effective Go](https://go.dev/doc/effective_go) — official guidance on package names, exported identifiers, comments, errors, interfaces, and general Go style.
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) — practical Go reviewer expectations, especially around package names, interfaces, error strings, and initialisms.
- [Go blog: Package names](https://go.dev/blog/package-names) — why package names should be short, clear, and non-stuttering.
- [Go blog: Organizing Go code](https://go.dev/blog/organizing-go-code) — historical Go team guidance on packages as the primary organization unit.

**Anti-pattern reference (intentionally not followed):**
- [golang-standards/project-layout](https://github.com/golang-standards/project-layout) — popular community layout that the Go team has publicly distanced itself from. We adopt some of its conventions (`cmd/`, `internal/`) but reject its layered structure suggestions.

**Conventions we picked based on evidence from production Go auth projects:**
- **Domain-owned GORM models (no central `model/` package).** Verified against Ory Hydra ([client/client.go](https://github.com/ory/hydra/blob/master/client/client.go)), Ory Kratos ([identity/identity.go](https://github.com/ory/kratos/blob/master/identity/identity.go)), and Dex ([storage/storage.go](https://github.com/dexidp/dex/blob/master/storage/storage.go)) — none of these use a central models package. Authelia ([internal/model/](https://github.com/authelia/authelia/tree/master/internal/model)) is the lone counter-example among production-grade Go auth projects, and its model set is much smaller than ours. A central package produces the anemic domain model and a god import.
- **Cross-aggregate references as `uuid.UUID`, not embedded structs.** This is the discipline that lets domain-owned models avoid import cycles. Both Hydra and Kratos do this. GORM `Preload` chains across aggregates get rewritten as service calls.

**Conventions we picked between equal-quality alternatives (no "majority" claim):**
- **`handler_<feature>.go` (prefix) over `<feature>_handler.go` (suffix).** Both styles are common across major Go projects. Prefix sorts handlers together and reads cleanly when a feature is handler-only.
- **`internal/platform/` over flat `internal/`.** With many cross-cutting packages, nesting them under `platform/` clarifies the boundary between domains and infrastructure.
- **One `wire.go` over per-domain `wire_<domain>.go`.** Per-domain wire files are valuable for very large dependency graphs (Hydra-scale); for ~16 domains, a single file shows the assembly graph at a glance.
