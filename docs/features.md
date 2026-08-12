# Features

The feature map for **maintainerd-auth**. Each row links to that feature's implementation
doc — what it is, how it flows end-to-end, the exact packages/files/tables/endpoints behind
it, its configuration, and its security properties.

New here? Read the [System Overview](overview.md) first for the conceptual model (tenants,
clients, identities, the two-plane architecture), then come back to drill into a feature.

**Status legend** — **Implemented**: shipped and wired. **Partial**: works, with the gaps
named explicitly in the row and the doc.

---

## Authentication & Sessions

| Feature | What it does | Status |
|---------|--------------|--------|
| [Authentication](features/authentication.md) | First-factor sign-in — email/password, magic-link, email/SMS OTP, and forgot/reset-password recovery — on the public identity plane | Implemented |
| [Sessions](features/sessions.md) | Server-side session records (idle/absolute timeout, concurrent-session cap, revocation) plus idempotent, family-aware refresh-token rotation | Implemented |
| [Multi-Factor Authentication](features/multi-factor-auth.md) | TOTP, WebAuthn/passkeys, SMS OTP, email OTP, and single-use backup codes with step-up elevation, gated by a per-tenant MFA policy | Implemented |
| [Registration & Invites](features/registration-and-invites.md) | Public self-service registration and admin email-invite provisioning, with registration-flow required fields and pre-assigned role grants | Implemented |
| [Setup & Bootstrap](features/setup-and-bootstrap.md) | First-run provisioning of the system tenant, super-admin, and control-plane principals via the REST setup wizard or a token-gated gRPC SetupService | Implemented |

## OAuth 2.0 / OIDC & Federation

| Feature | What it does | Status |
|---------|--------------|--------|
| [OAuth 2.0 & OpenID Connect](features/oauth2-oidc.md) | From-scratch multi-tenant OAuth 2.1 / OIDC provider — authorization-code+PKCE, refresh, client-credentials, device, token-exchange, CIBA, plus discovery, JWKS, revocation, introspection, UserInfo, PAR, DCR, and RP/back-channel logout | Implemented — introspection & DCR are control-plane-only; mTLS client-auth is registry-accepted but unimplemented |
| [Clients](features/clients.md) | OAuth2/OIDC client registrations (traditional/spa/mobile/m2m) — types, auth methods, grants, secrets, redirect/CORS rules, `config`→column mirroring | Implemented |
| [Federation & External IdPs](features/federation.md) | Brokering upstream OIDC, generic-OAuth2, and SAML 2.0 IdPs — per-tenant config, external-token exchange, JIT provisioning, identity linking, attribute extraction, home-realm discovery | Implemented |

## Authorization (IAM)

| Feature | What it does | Status |
|---------|--------------|--------|
| [IAM & Authorization](features/iam-authorization.md) | Tenant-scoped IAM catalog (services, APIs, permissions, roles, policies) with RBAC route guards plus an AWS-style PDP for service-to-service authorization — policy evaluator, ETag/304 bundle endpoint, `POST /authorize`, and `iam.*` webhook invalidation events | Implemented |

## Multi-Tenancy & Configuration

| Feature | What it does | Status |
|---------|--------------|--------|
| [Multi-Tenancy](features/multi-tenancy.md) | Tenant ownership tree with a singleton system tenant, subdomain-slug addressing, host-authoritative request binding, per-tenant settings/IP rules, and strict cross-tenant isolation | Implemented |
| [Security Settings](features/security-settings.md) | Per-tenant security policy stored as seven named JSONB configs in one audited, step-up-gated row, enforced at runtime by the auth flows | Implemented |
| [Branding & Templates](features/branding-and-templates.md) | Per-tenant white-label branding themes (console + hosted login) plus DB-backed email and SMS message templates | Partial — email/SMS template Create+Delete unrouted; `login_templates` DTOs-only |

## Messaging & Events

| Feature | What it does | Status |
|---------|--------------|--------|
| [Email & SMS Delivery](features/email-and-sms.md) | Tenant-scoped transactional messaging — SMTP-only email plus Twilio/SNS/Vonage/log SMS adapters, driven by encrypted per-tenant DB config and DB-backed templates | Partial — SMS: MessageBird unimplemented; only Twilio fields wired; `test_mode`/`sender_id` not enforced |
| [Webhooks](features/webhooks.md) | Per-tenant outbound HTTP delivery of thin integration events off the transactional outbox — HMAC-SHA256 signing, durable retries, dead-lettering, auto-quarantine, SSRF-hardened delivery | Implemented |
| [Event Bus](features/events.md) | Transactional-outbox event plane emitting thin, at-least-once integration events to HTTP webhooks and a RabbitMQ topic exchange | Implemented — RabbitMQ arm inert until `RABBITMQ_URL` is set |

## Platform & Security

| Feature | What it does | Status |
|---------|--------------|--------|
| [Audit Logging](features/audit-logging.md) | Two append-only tenant-scoped trails — a structured auth-event security log (OWASP vocabulary, partitioned, retention/PII config) and a management audit log of control-plane changes | Implemented |
| [Cryptography & Key Management](features/cryptography-and-keys.md) | RS256 JWT signing with kid-tagged multi-key JWKS and rotation, PKCE/SHA-256/CSPRNG, AES-256-GCM encrypt-at-rest, and DPoP proof-of-possession token binding | Implemented |
| [Secret Management](features/secret-management.md) | Pluggable secret loading (env / file / AWS Secrets Manager / AWS SSM / Vault / GCP / Azure) selected by `SECRET_PROVIDER`, with normalized values, env fallback, and JWT-key background refresh | Implemented |
| [gRPC Control Plane](features/grpc.md) | Opt-in, mTLS-guarded gRPC surface on `:50051` for orchestrator provisioning and runtime S2S decisions (PDP, introspection, user reads), gated by `GRPC_ENABLED` / `CONTROL_PLANE_ENABLED` / `INSTANCE_ROLE` | Implemented |
| [Observability](features/observability.md) | OpenTelemetry traces + OTLP logs (opt-in via `OTEL_ENABLED`) and always-on Prometheus `/metrics` on the management port, with `trace_id` log correlation | Implemented |

---

## Related documentation

- **[System Overview](overview.md)** — conceptual model and data hierarchy.
- **[Contributing](contributing/getting-started.md)** — local development, code structure, architecture, migrations, testing, environment variables.
- **[Operations](operations/operator-runbook.md)** — install, hardening, release preflight, compliance.
- **[API Reference](openapi.yaml)** — the OpenAPI 3.1 spec (also served at `/openapi.json`).
