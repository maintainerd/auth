# Target Architecture — Go Team Style (Domain-Grouped)

This document specifies the **target folder structure and file naming** for Lula auth if we adopt the Go team's recommended layout style, modeled after Ory Hydra / Kratos with elements of Zitadel. Every existing file is accounted for and mapped to its new home.

The naming conventions in this document have been **verified against** Hydra, Kratos, Zitadel, Authelia, Dex, Kubernetes, Docker, Prometheus, Grafana, Caddy, Gitea, Etcd, Thanos, and Cortex.

---

## 1. Guiding Principles

1. **Organize by domain (bounded context), not by architectural layer.** A package owns _everything_ for its domain: service logic, HTTP handlers, storage interface, GORM implementation, types, routes, and tests.
2. **Small, focused packages.** No flat "service/" or "repository/" buckets holding 50+ files.
3. **`internal/` for everything private to this module.** `pkg/` is unused (per Go team guidance).
4. **`cmd/server/main.go` stays thin** — bootstraps config/logging/DB/Redis, then delegates to `internal/app`.
5. **Shared GORM models stay central** in `internal/model/` — splitting them per-domain causes import cycles and is rarely worth the cost.
6. **Cross-domain dependencies go through interfaces.** If `authn` needs to read users, it depends on a small `user.Reader` interface defined in `authn/`, satisfied by `user.Service`.
7. **Composition root is per-domain.** `internal/app/wire_<domain>.go` files replace the single 125-line `services.go`.
8. **No subfolders inside a domain package.** Flat is correct, verified across every major Go project surveyed.

---

## 2. File Naming Conventions

These conventions are mandatory for consistency. They are derived from observed practice in the Go projects listed above.

### 2.1 General rules

| Rule | Example |
|---|---|
| Lowercase only, `_` separates words | `email_verification.go` |
| `_test.go` suffix marks test files (Go-enforced) | `login_test.go` |
| Avoid stuttering the package name | `user.Service`, not `user.UserService` |
| One dominant concept per file, file named after that concept | `authorize.go` defines `AuthorizeService` |
| Reserve `_<os>.go` / `_<arch>.go` for build constraints only | don't name files `login_darwin.go` for non-build reasons |

### 2.2 Per-domain file vocabulary

Every domain package uses **exactly these file names**. Memorize this list.

| File | Contents | Required? | Rationale |
|---|---|---|---|
| **`registry.go`** | `type Deps`, `type Registry`, `func NewRegistry(Deps) *Registry` — package's wiring/DI container | ✓ always | Hydra and Kratos use this exact name. Universal in Go DI patterns. |
| **`routes.go`** | `func (r *Registry) RegisterRoutes(rt chi.Router, mw Middleware)` | ✓ always | Self-documenting; transport layer calls it. |
| **`types.go`** | Request/response shapes + any other types used across the package | ✓ when types are shared | Used by Authelia, etcd, Prometheus. Go projects do **not** use `dto.go` — DTO is Java terminology. |
| **`repository.go`** | Storage interface(s), one per aggregate the domain owns | only if domain has persistence | Common Go term; clearer than `manager.go` or `store.go` to mixed audiences. |
| **`repository_gorm.go`** | GORM implementations of `repository.go` interfaces | only if domain has persistence | The `_gorm` suffix flags the framework binding clearly. |
| **`<feature>.go`** | Service logic for one sub-feature: `authorize.go`, `token.go`, `login.go` | ✓ at least one | Standard Go: name files after the dominant concept they contain. |
| **`handler_<feature>.go`** | HTTP handler methods for that sub-feature | one per `<feature>.go` that exposes HTTP | **Prefix style** (Hydra, Kratos, Authelia). Sorts all handlers together. |
| **`<feature>_test.go`** | Tests for `<feature>.go` | strongly recommended | Go-enforced naming. |
| **`handler_<feature>_test.go`** | Tests for `handler_<feature>.go` | strongly recommended | Mirrors the source naming. |
| **`errors.go`** | Package-level sentinel errors and custom error types | when there are 3+ errors | Convention in stdlib, Hydra, Kratos, Authelia, Caddy. |
| **`helpers.go`** | Tiny package-private utilities — **use sparingly** | avoid unless small | Used in Hydra (`helpers.go`) and Kratos (`helper.go`). Avoid bloat. |
| **`doc.go`** | Package documentation only (`package foo` + doc comment) | optional | Standard Go convention; used in etcd, Cockroach. |
| **`const.go`** | Package-scoped constants when there are many | optional, when justified | Used by Authelia. Inline constants are preferred. |
| **`mock_<dep>_test.go`** | Hand-written mocks compiled only in tests | as needed | Convention from existing code; clear naming. |

### 2.3 Names to **avoid**

| ❌ Don't use | Why | ✓ Use instead |
|---|---|---|
| `module.go` | NestJS/Angular term, not Go | `registry.go` |
| `dto.go` | Java/.NET term, not Go | `types.go` (or inline in handler file) |
| `<feature>_handler.go` | Suffix style is minority practice | `handler_<feature>.go` |
| `interfaces.go` | Don't centralize interfaces; define them at the consumer | n/a |
| `utils.go` / `common.go` | Almost always indicates code that belongs in a more specific package | move to a domain-named package |
| `models.go` (in domain pkgs) | Models live centrally in `internal/model/` | n/a |
| `services.go` (as a flat bucket) | Anti-pattern; the source of the migration | per-domain `<feature>.go` files |

### 2.4 Worked example — `oauth/` with corrected names

```
internal/oauth/
├── registry.go               # type Deps, type Registry, NewRegistry — package wiring
├── routes.go                 # RegisterRoutes(chi.Router, Middleware)
├── types.go                  # request/response types used across handlers
├── helpers.go                # shared validation/parsing utilities
├── errors.go                 # ErrInvalidGrant, ErrUnsupportedResponseType, ...
├── repository.go             # AuthCodeRepository, RefreshTokenRepository, ...
├── repository_gorm.go        # GORM implementations
├── authorize.go              # AuthorizeService — /authorize service logic
├── handler_authorize.go      # HTTP handler for /authorize
├── authorize_test.go
├── handler_authorize_test.go
├── token.go                  # TokenService — /token service logic
├── handler_token.go
├── token_test.go
├── consent.go
├── handler_consent.go
├── par.go
├── handler_par.go
├── device.go
├── handler_device.go
├── ciba.go
├── handler_ciba.go
├── token_exchange.go
├── handler_token_exchange.go
├── session.go
├── handler_session.go
├── register.go               # Dynamic Client Registration (RFC 7591)
├── handler_register.go
├── handler_userinfo.go       # handler-only (no separate service file)
└── handler_discovery.go      # handler-only — /.well-known/openid-configuration, JWKS
```

Note one nice property of the prefix style: handler-only files like `userinfo` and `discovery` get a single clean name (`handler_userinfo.go`) without suggesting a missing service file.

---

## 3. Top-Level Layout

```
maintainerd-auth/
├── cmd/
│   └── server/
│       └── main.go                 # thin entrypoint
├── internal/
│   ├── app/                        # composition root (per-domain wiring)
│   ├── oauth/                      # OAuth 2.1 / OIDC flows
│   ├── user/                       # user, profile, settings, account
│   ├── authn/                      # login, register, password, magic link, sms login, session
│   ├── mfa/                        # TOTP, WebAuthn, backup codes
│   ├── tenant/                     # tenants, members, settings
│   ├── iam/                        # services (resources), APIs, permissions, roles, policies
│   ├── client/                     # OAuth clients, API keys
│   ├── idp/                        # identity providers, federation, signup flows
│   ├── invite/                     # invitations
│   ├── secpolicy/                  # security settings, IP restrictions
│   ├── branding/                   # branding + email/sms/login templates
│   ├── channel/                    # outbound email/SMS provider config
│   ├── event/                      # auth event log + retention
│   ├── webhook/                    # outbound webhook endpoints + delivery
│   ├── setup/                      # bootstrap / initial setup
│   ├── server/                     # transport (REST + gRPC) hosting
│   │   ├── rest.go
│   │   └── grpc.go
│   ├── platform/                   # cross-cutting infrastructure (optional grouping)
│   │   ├── apperror/
│   │   ├── cache/
│   │   ├── config/
│   │   ├── cookie/
│   │   ├── crypto/
│   │   ├── database/
│   │   ├── dpop/
│   │   ├── jwt/
│   │   ├── logging/
│   │   ├── middleware/
│   │   ├── ptr/
│   │   ├── security/
│   │   ├── signedurl/
│   │   ├── telemetry/
│   │   ├── templates/
│   │   └── valid/
│   ├── email/                      # email provider adapters (Mailgun, SES, SMTP, ...)
│   ├── sms/                        # SMS provider adapters (Twilio, SNS, Vonage)
│   ├── model/                      # shared GORM models (all aggregates)
│   ├── runner/                     # background workers (retention, etc.)
│   └── gen/                        # generated code (proto, etc.)
├── proto/                          # protobuf definitions
├── docs/
├── tests/
│   ├── e2e/
│   └── integration/
├── scripts/
├── nginx/
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

> **Note on `platform/`:** This is a grouping convenience. You can either keep cross-cutting packages at the top of `internal/` (current state) or nest them under `internal/platform/`. The latter visually separates "infrastructure" from "domains" and is cleaner once the domain list grows. Pick one and stick with it.

---

## 4. Domain Packages — Full Listing

### 4.1 `internal/oauth/` — OAuth 2.1 / OIDC

Handles every OAuth flow plus discovery and JWKS.

```
oauth/
├── registry.go
├── routes.go
├── types.go
├── helpers.go                        # was: rest/handler/oauth_helpers.go
├── errors.go
├── repository.go                     # interfaces for auth_code, refresh_token, par, device, ciba, consent_grant, consent_challenge
├── repository_gorm.go
├── authorize.go                      # was: service/oauth_authorize.go
├── handler_authorize.go              # was: rest/handler/oauth_authorize.go
├── authorize_test.go
├── token.go                          # was: service/oauth_token.go
├── handler_token.go                  # was: rest/handler/oauth_token.go
├── token_test.go
├── consent.go                        # was: service/oauth_consent.go
├── handler_consent.go                # was: rest/handler/oauth_consent.go
├── consent_test.go
├── par.go                            # was: service/oauth_par.go
├── handler_par.go                    # was: rest/handler/oauth_par.go
├── device.go                         # was: service/oauth_device.go
├── handler_device.go                 # was: rest/handler/oauth_device.go
├── ciba.go                           # was: service/oauth_ciba.go
├── handler_ciba.go                   # was: rest/handler/oauth_ciba.go
├── token_exchange.go                 # was: service/oauth_token_exchange.go
├── handler_token_exchange.go         # was: rest/handler/oauth_token_exchange.go
├── session.go                        # was: service/oauth_session.go
├── handler_session.go                # was: rest/handler/oauth_session.go
├── register.go                       # was: service/oauth_register.go     (DCR)
├── handler_register.go               # was: rest/handler/oauth_register.go
├── handler_userinfo.go               # was: rest/handler/oauth_userinfo.go
└── handler_discovery.go              # was: rest/handler/oauth_discovery.go
```

**Owns models:** `oauth_authorization_code`, `oauth_ciba_request`, `oauth_consent_challenge`, `oauth_consent_grant`, `oauth_device_code`, `oauth_par_request`, `oauth_refresh_token`.

---

### 4.2 `internal/user/` — User accounts, profile, settings

```
user/
├── registry.go
├── routes.go
├── types.go                          # was: dto/user.go, dto/profile.go, dto/user_setting.go, dto/account.go
├── repository.go                     # user, user_identity, user_role, user_pool, user_password_history, user_token, profile, user_setting
├── repository_gorm.go
├── user.go                           # user CRUD + identity
├── handler_user.go                   # was: rest/handler/user.go
├── user_test.go
├── profile.go                        # was: service/profile.go
├── handler_profile.go                # was: rest/handler/profile.go
├── setting.go                        # was: service/user_setting (if present)
├── handler_setting.go                # was: rest/handler/user_setting.go
├── account.go                        # was: service/account.go (self-service account ops: delete, export, recovery codes)
└── handler_account.go                # was: rest/handler/account.go
```

**Owns models:** `user`, `user_identity`, `user_role`, `user_pool`, `user_password_history`, `user_token`, `profile`, `user_setting`.

---

### 4.3 `internal/authn/` — Authentication flows

Everything a user does to prove identity in a session.

```
authn/
├── registry.go
├── routes.go
├── types.go                          # was: dto/login.go, register.go, forgot_password.go, reset_password.go, magic_link.go, email_verification.go, sms_login.go, session.go
├── login.go                          # was: service/login.go
├── handler_login.go                  # was: rest/handler/login.go
├── login_test.go
├── register.go                       # was: service/register.go
├── handler_register.go               # was: rest/handler/register.go
├── register_test.go
├── forgot_password.go                # was: service/forgot_password.go
├── handler_forgot_password.go        # was: rest/handler/forgot_password.go
├── reset_password.go                 # was: service/reset_password.go
├── handler_reset_password.go         # was: rest/handler/reset_password.go
├── password_policy.go                # was: service/password_policy.go
├── email_verification.go             # was: service/email_verification.go
├── handler_email_verification.go     # was: rest/handler/email_verification.go
├── magic_link.go                     # was: service/magic_link.go
├── handler_magic_link.go             # was: rest/handler/magic_link.go
├── sms_login.go                      # was: service/sms_login.go
├── handler_sms_login.go              # was: rest/handler/sms_login.go
├── session.go                        # was: service/session.go
└── handler_session.go                # (currently embedded in other handlers; consolidate)
```

**Note:** `authn` depends on `user` for user reads/writes and on `client` for client validation. These dependencies are explicit through interfaces declared inside `authn/`.

**No models of its own** — uses `user_token`, `sms_otp`, etc. (owned by other packages).

---

### 4.4 `internal/mfa/` — Multi-factor authentication

```
mfa/
├── registry.go
├── routes.go
├── types.go                          # was: dto/mfa.go
├── repository.go                     # totp_secret, webauthn_credential, backup_code
├── repository_gorm.go
├── mfa.go                            # was: service/mfa.go (TOTP + backup codes orchestration)
├── handler_mfa.go                    # was: rest/handler/mfa.go
├── mfa_test.go
├── totp.go                           # TOTP-specific logic
├── webauthn.go                       # WebAuthn-specific logic
└── backup_codes.go
```

**Owns models:** `user_totp_secret`, `user_webauthn_credential`, `user_backup_code`.

---

### 4.5 `internal/tenant/` — Multi-tenancy

```
tenant/
├── registry.go
├── routes.go
├── types.go                          # was: dto/tenant.go, tenant_member.go, tenant_setting.go
├── repository.go                     # tenant, tenant_member, tenant_service, tenant_setting
├── repository_gorm.go
├── tenant.go                         # was: service/tenant.go
├── handler_tenant.go                 # was: rest/handler/tenant.go
├── tenant_test.go
├── member.go                         # was: service/tenant_member.go
├── handler_member.go                 # (currently in tenant.go handler — split out)
├── setting.go                        # was: service/tenant_setting.go
├── handler_setting.go                # was: rest/handler/tenant_setting.go
├── access.go                         # was: service/tenant_access.go
└── isolation_test.go                 # was: service/tenant_isolation_test.go
```

**Owns models:** `tenant`, `tenant_member`, `tenant_service`, `tenant_setting`.

---

### 4.6 `internal/iam/` — Identity & Access Management resources

The "what can be accessed" half of authorization: services (resources), APIs, permissions, roles, policies.

```
iam/
├── registry.go
├── routes.go
├── types.go                          # was: dto/service.go, api.go, permission.go, role.go, policy.go
├── repository.go                     # service, api, permission, role, role_permission, policy, service_policy
├── repository_gorm.go
├── service.go                        # was: service/service.go (IAM "service" resource)
├── handler_service.go                # was: rest/handler/service.go
├── api.go                            # was: service/api.go
├── handler_api.go                    # was: rest/handler/api.go
├── permission.go                     # was: service/permission.go
├── handler_permission.go             # was: rest/handler/permission.go
├── role.go                           # was: service/role.go
├── handler_role.go                   # was: rest/handler/role.go
├── policy.go                         # was: service/policy.go
└── handler_policy.go                 # was: rest/handler/policy.go
```

**Owns models:** `service`, `api`, `permission`, `role`, `role_permission`, `policy`, `service_policy`, `api_permission`.

---

### 4.7 `internal/client/` — OAuth clients and API keys

```
client/
├── registry.go
├── routes.go
├── types.go                          # was: dto/client.go, api_key.go
├── repository.go                     # client, client_uri, client_api, client_permission, api_key, api_key_api, api_key_permission
├── repository_gorm.go
├── client.go                         # was: service/client.go
├── handler_client.go                 # was: rest/handler/client.go
├── client_test.go
├── api_key.go                        # was: service/api_key.go
├── handler_api_key.go                # was: rest/handler/api_key.go
└── api_key_test.go
```

**Owns models:** `client`, `client_uri`, `client_api`, `client_permission`, `api_key`, `api_key_api`, `api_key_permission`.

---

### 4.8 `internal/idp/` — Identity providers, federation, signup flows

```
idp/
├── registry.go
├── routes.go
├── types.go                          # was: dto/idp.go, federation.go, signup_flow.go, signup_flow_role.go
├── repository.go                     # identity_provider, signup_flow, signup_flow_role
├── repository_gorm.go
├── provider.go                       # was: service/identity_provider.go
├── handler_provider.go               # was: rest/handler/identity_provider.go
├── federation.go                     # was: service/federation.go
├── handler_federation.go             # was: rest/handler/federation.go
├── signup_flow.go                    # was: service/signup_flow.go
└── handler_signup_flow.go            # was: rest/handler/signup_flow.go
```

**Owns models:** `identity_provider`, `federation`, `signup_flow`, `signup_flow_role`.

---

### 4.9 `internal/invite/` — Invitations

```
invite/
├── registry.go
├── routes.go
├── types.go                          # was: dto/invite.go
├── repository.go                     # invite, invite_role
├── repository_gorm.go
├── invite.go                         # was: service/invite.go
├── handler_invite.go                 # was: rest/handler/invite.go
└── invite_test.go
```

**Owns models:** `invite`, `invite_role`.

---

### 4.10 `internal/secpolicy/` — Security settings + IP restrictions

```
secpolicy/
├── registry.go
├── routes.go
├── types.go                          # was: dto/security_setting.go, ip_restriction_rule.go
├── repository.go                     # security_setting, security_settings_audit, ip_restriction_rule
├── repository_gorm.go
├── setting.go                        # was: service/security_setting.go
├── handler_setting.go                # was: rest/handler/security_setting.go
├── ip_restriction.go                 # was: service/ip_restriction_rule.go
└── handler_ip_restriction.go         # was: rest/handler/ip_restriction_rule.go
```

**Owns models:** `security_setting`, `security_settings_audit`, `ip_restriction_rule`.

> **Naming note:** Don't call this package `security/` — that collides with the existing platform package. `secpolicy/` (security policy) is unambiguous.

---

### 4.11 `internal/branding/` — Branding + templates

```
branding/
├── registry.go
├── routes.go
├── types.go                          # was: dto/branding.go, email_template.go, sms_template.go, login_template.go
├── repository.go                     # branding, email_template, sms_template, login_template
├── repository_gorm.go
├── branding.go                       # was: service/branding.go
├── handler_branding.go               # was: rest/handler/branding.go
├── email_template.go                 # was: service/email_template.go
├── handler_email_template.go         # was: rest/handler/email_template.go
├── sms_template.go                   # was: service/sms_template.go
├── handler_sms_template.go           # was: rest/handler/sms_template.go
├── login_template.go                 # was: service/login_template.go
└── handler_login_template.go         # was: rest/handler/login_template.go
```

**Owns models:** `branding`, `email_template`, `sms_template`, `login_template`.

---

### 4.12 `internal/channel/` — Outbound delivery channel config

Configuration for _how_ the system sends email and SMS (provider, credentials, sender info). The provider _adapters_ themselves stay in `internal/email/` and `internal/sms/`.

```
channel/
├── registry.go
├── routes.go
├── types.go                          # was: dto/email_config.go, sms_config.go
├── repository.go                     # email_config, sms_config, sms_otp
├── repository_gorm.go
├── email_config.go                   # was: service/email_config.go
├── handler_email_config.go           # was: rest/handler/email_config.go
├── sms_config.go                     # was: service/sms_config.go
└── handler_sms_config.go             # was: rest/handler/sms_config.go
```

**Owns models:** `email_config`, `sms_config`, `sms_otp`.

---

### 4.13 `internal/event/` — Auth event log

```
event/
├── registry.go
├── routes.go
├── types.go                          # was: dto/auth_event.go
├── repository.go                     # auth_event
├── repository_gorm.go
├── event.go                          # was: service/auth_event.go
├── handler_event.go                  # was: rest/handler/auth_event.go
└── retention.go                      # was: runner/StartRetentionRunner (auth-event specific bits)
```

**Owns models:** `auth_event`.

---

### 4.14 `internal/webhook/` — Outbound webhooks

Today this package only has the delivery mechanism. We promote it to a full domain by adding the endpoint CRUD here.

```
webhook/
├── registry.go
├── routes.go
├── types.go                          # was: dto/webhook_endpoint.go
├── repository.go                     # webhook_endpoint
├── repository_gorm.go
├── endpoint.go                       # was: service/webhook_endpoint.go
├── handler_endpoint.go               # was: rest/handler/webhook_endpoint.go
├── dispatcher.go                     # existing
├── deliver.go                        # existing
├── payload.go                        # existing
└── signer.go                         # existing
```

**Owns models:** `webhook_endpoint`.

---

### 4.15 `internal/setup/` — Bootstrap

Orchestrates first-time setup (creates default tenant, admin user, system IDP/client, etc.). Cross-cutting orchestrator — depends on most other packages.

```
setup/
├── registry.go
├── routes.go
├── types.go                          # was: dto/setup.go
├── setup.go                          # was: service/setup.go
└── handler_setup.go                  # was: rest/handler/setup.go
```

**No models of its own.**

---

## 5. Cross-Cutting Packages (Platform / Infrastructure)

These stay roughly as-is. Optional: nest under `internal/platform/` for visual separation.

| Package | Purpose | Source today |
|---|---|---|
| `apperror/` | Domain error types + HTTP mapping | `internal/apperror/` |
| `cache/` | Redis-backed cache abstraction + JTI denylist | `internal/cache/` |
| `config/` | Env loading, DB init, Redis client | `internal/config/` |
| `cookie/` | Secure cookie helpers | `internal/cookie/` |
| `crypto/` | Hashing, encryption helpers | `internal/crypto/` |
| `database/` | DB hardening, pool tuning | `internal/database/` |
| `dpop/` | OAuth DPoP proof validation (RFC 9449) | `internal/dpop/` |
| `jwt/` | JWT signing/verification, JWK set | `internal/jwt/` |
| `logging/` | slog setup + PII redaction handler | `internal/logging/` |
| `middleware/` | HTTP middleware (auth, CORS, CSRF, rate limit, logging, permissions, sessions, user, security headers, content-type) | `internal/middleware/` |
| `ptr/` | Pointer helpers | `internal/ptr/` |
| `security/` | Rate limiter init, security primitives | `internal/security/` |
| `signedurl/` | Signed URL generation/verification | `internal/signedurl/` |
| `telemetry/` | OTEL traces + Prometheus metrics | `internal/telemetry/` |
| `templates/` | Template rendering | `internal/templates/` |
| `valid/` | Common validators | `internal/valid/` |
| `email/` | Email provider adapters (Mailgun, Postmark, Resend, SendGrid, SES, SMTP) | `internal/email/` |
| `sms/` | SMS provider adapters (SNS, Twilio, Vonage) | `internal/sms/` |
| `model/` | Shared GORM models (all aggregates, central) | `internal/model/` |
| `runner/` | Background workers (retention, migrations) | `internal/runner/` |
| `gen/` | Generated code (proto) | `internal/gen/` |

---

## 6. Transport Hosting

```
internal/server/
├── rest.go                          # was: internal/rest/server/server.go
└── grpc.go                          # was: internal/grpc/server/server.go
```

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

---

## 7. Composition Root

```
internal/app/
├── app.go                           # type App struct { ... per-registry fields }
├── wire_oauth.go                    # builds oauth.Registry
├── wire_user.go                     # builds user.Registry
├── wire_authn.go                    # builds authn.Registry
├── wire_mfa.go                      # builds mfa.Registry
├── wire_tenant.go
├── wire_iam.go
├── wire_client.go
├── wire_idp.go
├── wire_invite.go
├── wire_secpolicy.go
├── wire_branding.go
├── wire_channel.go
├── wire_event.go
├── wire_webhook.go
└── wire_setup.go
```

`app.App` holds one field per registry, _not_ per service:

```go
type App struct {
    DB    *gorm.DB
    Cache *cache.Cache

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
    Channel   *channel.Registry
    Event     *event.Registry
    Webhook   *webhook.Registry
    Setup     *setup.Registry
}
```

Each `wire_<domain>.go` is ~15 lines, takes the shared `*gorm.DB`, `*cache.Cache`, and any cross-domain dependencies, and returns the registry:

```go
// wire_oauth.go
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

This replaces today's [internal/app/services.go](../internal/app/services.go) (125 lines, 45 constructor calls) with one focused file per domain.

---

## 8. Complete Migration Mapping

Every existing file → new location.

### 8.1 Services → domains

| Current file | New location |
|---|---|
| `internal/service/account.go` | `internal/user/account.go` |
| `internal/service/api.go` | `internal/iam/api.go` |
| `internal/service/api_key.go` | `internal/client/api_key.go` |
| `internal/service/auth_event.go` | `internal/event/event.go` |
| `internal/service/branding.go` | `internal/branding/branding.go` |
| `internal/service/client.go` | `internal/client/client.go` |
| `internal/service/email_config.go` | `internal/channel/email_config.go` |
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
| `internal/service/signup_flow.go` | `internal/idp/signup_flow.go` |
| `internal/service/sms_config.go` | `internal/channel/sms_config.go` |
| `internal/service/sms_login.go` | `internal/authn/sms_login.go` |
| `internal/service/sms_template.go` | `internal/branding/sms_template.go` |
| `internal/service/tenant.go` | `internal/tenant/tenant.go` |
| `internal/service/tenant_access.go` | `internal/tenant/access.go` |
| `internal/service/tenant_member.go` | `internal/tenant/member.go` |
| `internal/service/tenant_setting.go` | `internal/tenant/setting.go` |
| `internal/service/user_setting.go`* | `internal/user/setting.go` |
| `internal/service/webhook_endpoint.go` | `internal/webhook/endpoint.go` |

\* If a standalone file exists; otherwise lives inside `profile.go` today.

### 8.2 Handlers → domains

| Current file | New location |
|---|---|
| `internal/rest/handler/account.go` | `internal/user/handler_account.go` |
| `internal/rest/handler/api.go` | `internal/iam/handler_api.go` |
| `internal/rest/handler/api_key.go` | `internal/client/handler_api_key.go` |
| `internal/rest/handler/auth_event.go` | `internal/event/handler_event.go` |
| `internal/rest/handler/branding.go` | `internal/branding/handler_branding.go` |
| `internal/rest/handler/client.go` | `internal/client/handler_client.go` |
| `internal/rest/handler/email_config.go` | `internal/channel/handler_email_config.go` |
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
| `internal/rest/handler/oauth_helpers.go` | `internal/oauth/helpers.go` |
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
| `internal/rest/handler/query.go` | `internal/server/query.go` (shared query parser) |
| `internal/rest/handler/register.go` | `internal/authn/handler_register.go` |
| `internal/rest/handler/reset_password.go` | `internal/authn/handler_reset_password.go` |
| `internal/rest/handler/role.go` | `internal/iam/handler_role.go` |
| `internal/rest/handler/security_setting.go` | `internal/secpolicy/handler_setting.go` |
| `internal/rest/handler/service.go` | `internal/iam/handler_service.go` |
| `internal/rest/handler/setup.go` | `internal/setup/handler_setup.go` |
| `internal/rest/handler/signup_flow.go` | `internal/idp/handler_signup_flow.go` |
| `internal/rest/handler/sms_config.go` | `internal/channel/handler_sms_config.go` |
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
| `internal/rest/route/auth_event.go` | `internal/event/routes.go` |
| `internal/rest/route/branding.go` | merged into `internal/branding/routes.go` |
| `internal/rest/route/client.go` | merged into `internal/client/routes.go` |
| `internal/rest/route/email_config.go` | merged into `internal/channel/routes.go` |
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
| `internal/rest/route/sms_config.go` | merged into `internal/channel/routes.go` |
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
| `internal/repository/auth_event.go` | `internal/event/repository_gorm.go` |
| `internal/repository/base.go` | `internal/platform/database/base_repository.go` |
| `internal/repository/base_interface.go` | `internal/platform/database/base_repository.go` |
| `internal/repository/branding.go` | `internal/branding/repository_gorm.go` |
| `internal/repository/client.go` | `internal/client/repository_gorm.go` |
| `internal/repository/client_api.go` | `internal/client/repository_gorm.go` |
| `internal/repository/client_permission.go` | `internal/client/repository_gorm.go` |
| `internal/repository/client_uri.go` | `internal/client/repository_gorm.go` |
| `internal/repository/email_config.go` | `internal/channel/repository_gorm.go` |
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
| `internal/repository/sms_config.go` | `internal/channel/repository_gorm.go` |
| `internal/repository/sms_otp.go` | `internal/channel/repository_gorm.go` |
| `internal/repository/sms_template.go` | `internal/branding/repository_gorm.go` |
| `internal/repository/tenant.go` | `internal/tenant/repository_gorm.go` |
| `internal/repository/tenant_member.go` | `internal/tenant/repository_gorm.go` |
| `internal/repository/tenant_service.go` | `internal/tenant/repository_gorm.go` |
| `internal/repository/tenant_setting.go` | `internal/tenant/repository_gorm.go` |
| `internal/repository/user.go` | `internal/user/repository_gorm.go` |
| `internal/repository/user_backup_code.go` | `internal/mfa/repository_gorm.go` |
| `internal/repository/user_identity.go` | `internal/user/repository_gorm.go` |
| `internal/repository/user_password_history.go` | `internal/user/repository_gorm.go` |
| `internal/repository/user_pool.go` | `internal/user/repository_gorm.go` |
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
| `internal/dto/auth_event.go` | merged into `internal/event/types.go` |
| `internal/dto/branding.go` | merged into `internal/branding/types.go` |
| `internal/dto/client.go` | merged into `internal/client/types.go` |
| `internal/dto/date.go` | `internal/platform/valid/date.go` |
| `internal/dto/email_config.go` | merged into `internal/channel/types.go` |
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
| `internal/dto/pagination.go` | `internal/server/pagination.go` (shared) |
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
| `internal/dto/sms_config.go` | merged into `internal/channel/types.go` |
| `internal/dto/sms_login.go` | merged into `internal/authn/types.go` |
| `internal/dto/sms_template.go` | merged into `internal/branding/types.go` |
| `internal/dto/tenant.go` | merged into `internal/tenant/types.go` |
| `internal/dto/tenant_member.go` | merged into `internal/tenant/types.go` |
| `internal/dto/tenant_setting.go` | merged into `internal/tenant/types.go` |
| `internal/dto/user.go` | merged into `internal/user/types.go` |
| `internal/dto/user_setting.go` | merged into `internal/user/types.go` |
| `internal/dto/webhook_endpoint.go` | merged into `internal/webhook/types.go` |

### 8.6 Models — stay in `internal/model/`

All files under `internal/model/` stay where they are. They are shared across packages and moving them per-domain creates import cycles.

### 8.7 App wiring

| Current file | New location |
|---|---|
| `internal/app/app.go` | `internal/app/app.go` (reduced — registries instead of services) |
| `internal/app/services.go` | **deleted** — replaced by `wire_<domain>.go` files |
| `internal/app/repositories.go` | **deleted** — each domain owns its repositories |

### 8.8 Transport hosting

| Current file | New location |
|---|---|
| `internal/rest/server/server.go` | `internal/server/rest.go` |
| `internal/rest/response/*` | `internal/server/response/` |
| `internal/grpc/server/server.go` | `internal/server/grpc.go` |
| `internal/grpc/handler/seeder.go` | `internal/setup/grpc.go` |

### 8.9 Things that stay exactly where they are

```
cmd/server/main.go
internal/apperror/
internal/cache/
internal/config/
internal/cookie/
internal/crypto/
internal/database/
internal/dpop/
internal/email/
internal/jwt/
internal/logging/
internal/middleware/
internal/model/
internal/ptr/
internal/runner/
internal/security/
internal/signedurl/
internal/sms/
internal/telemetry/
internal/templates/
internal/valid/
internal/webhook/                    (gets endpoint.go + handler_endpoint.go added)
internal/gen/
proto/
docs/
tests/
scripts/
nginx/
```

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

import "github.com/go-chi/chi/v5"

func (r *Registry) RegisterRoutes(rt chi.Router, mw Middleware) {
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

Every domain follows this exact shape.

---

## 10. What Changes Behaviorally

**Nothing.** This is a pure structural refactor. Every HTTP route returns the same response, every service has the same behavior, every test still passes. The user-visible API surface is identical.

What changes is:
- Where code lives.
- How packages depend on each other (explicit, via interfaces).
- How easy it is to find related code (everything for "invites" is in one folder).
- How extractable a domain becomes if you ever want to split into multiple services.

---

## 11. References & Evidence

Naming conventions in this document were verified against the following Go projects:

**Auth-domain projects:**
- [Ory Hydra](https://github.com/ory/hydra) — `registry.go`, `handler.go`, `handler_<feature>.go`, `manager.go`
- [Ory Kratos](https://github.com/ory/kratos) — `registry.go`, `handler.go`, `handler_<feature>.go`, `manager.go`
- [Zitadel](https://github.com/zitadel/zitadel) — nested `model/` and `repository/` sub-packages (alternate style)
- [Authelia](https://github.com/authelia/authelia) — `types.go`, `const.go`, `errors.go`, `handler_<feature>.go`, `store.go`
- [Dex](https://github.com/dexidp/dex) — feature-grouped flat packages

**General Go ecosystem:**
- Kubernetes — `scheduler.go`, `eventhandlers.go`, `<concept>.go` style
- Docker (Moby) — `container_routes.go`, concept-named files
- Prometheus — `interface.go`, `errors.go`, concept-named files
- Caddy — `errors.go`, `matchers.go`, concept-named files
- Etcd — `interface.go`, `doc.go`, `server.go`
- Gitea — concept-named files (`oauth2.go`, `webauthn.go`, `session.go`)
- Cockroach — `doc.go`, descriptive concept files

**Official guidance:**
- [go.dev/doc/modules/layout](https://go.dev/doc/modules/layout) — Go team's project layout guidance
- [Effective Go](https://go.dev/doc/effective_go) — package and file conventions
- [Russ Cox on project structure](https://research.swtch.com/) — informal Go team perspective

**Anti-pattern reference (intentionally not followed):**
- [golang-standards/project-layout](https://github.com/golang-standards/project-layout) — popular community layout that the Go team has publicly distanced itself from. We adopt some of its conventions (`cmd/`, `internal/`) but reject its layered structure suggestions.
