# maintainerd-auth — Features

maintainerd-auth is a self-hosted, open-source identity and access management server. It is designed to be the authentication and authorization backbone for any application — from a single-tenant SaaS to a large multi-tenant platform. It is comparable to Keycloak, Zitadel, and Auth0, but built with flexibility and developer ergonomics as first-class concerns.

---

## Core Authentication

Every standard authentication flow is built in. Users can sign in with email and password, receive a one-time magic link by email, or verify their identity with an email OTP code. Passwordless login is fully supported out of the box.

- **Email + password login** with bcrypt hashing (cost-tunable), brute-force protection, and account lockout
- **Magic link login** — time-limited (15 min), single-use signed URLs sent by email; auto-verifies the user's email address on first use
- **Email OTP** — 6-digit code with configurable TTL
- **Forgot password** flow with secure token issuance and consumption
- **Email verification** on signup — send + verify endpoints, triggers on registration
- **User registration** with configurable signup flows and automatic role assignment
- **Invite-based registration** — send invite links with pre-assigned roles
- **Internal login** (management port 8080, no `client_id` required)
- **Public login** (identity port 8081, `client_id` + `provider_id` required)
- **Logout** — clears session cookies and revokes the active refresh token

---

## OAuth 2.0 & OpenID Connect

maintainerd-auth implements the full OAuth 2.0 and OpenID Connect specification, including several advanced extensions that most identity servers only partially support or defer to later releases.

### Endpoints

All standard and extended OAuth 2.0 / OIDC endpoints are available:

- `POST /oauth/token` — token issuance for all grant types (RFC 6749)
- `GET /oauth/authorize` — authorization endpoint with PKCE enforcement (RFC 6749, RFC 7636)
- `POST /oauth/revoke` — token revocation (RFC 7009)
- `POST /oauth/introspect` — token introspection, management port only (RFC 7662)
- `GET /oauth/userinfo` — OpenID Connect UserInfo endpoint (OIDC Core §5.3)
- `GET /.well-known/openid-configuration` — OIDC Discovery (OIDC Discovery 1.0)
- `GET /.well-known/oauth-authorization-server` — Authorization server metadata (RFC 8414)
- `GET /.well-known/jwks.json` — JSON Web Key Set (RFC 7517)
- `POST /oauth/par` — Pushed Authorization Requests (RFC 9126)
- `POST /oauth/device_authorization` — Device Authorization Grant (RFC 8628)
- `POST /oauth/register` — Dynamic Client Registration (RFC 7591)
- `GET/POST /oauth/end_session` — RP-Initiated Logout (OIDC Session Management 1.0)
- `POST /oauth/logout/backchannel` — Back-Channel Logout (OIDC Back-Channel Logout 1.0)
- `POST /oauth/ciba` — Client-Initiated Backchannel Authentication, poll mode (CIBA Core)
- Consent challenge retrieval, consent decision submission
- Consent grant listing and per-grant revocation

### Grant Types

- `authorization_code` with mandatory PKCE S256 (RFC 7636)
- `refresh_token` with automatic rotation, token family tracking, and reuse detection → family-wide revocation
- `client_credentials` for machine-to-machine flows (RFC 6749 §4.4)
- `device_code` for input-constrained devices (RFC 8628)
- `urn:ietf:params:oauth:grant-type:token-exchange` — Token Exchange (RFC 8693)
- `urn:openid:params:grant-type:ciba` — Client-Initiated Backchannel Authentication

### Token Security

Access tokens are signed JWTs (RS256) with a 15-minute TTL. Refresh tokens are random, hashed with SHA-256 at rest, rotated on every use, and bound to a family — reusing a superseded refresh token immediately revokes the entire family. Authorization codes are single-use and hashed at rest.

- All tokens carry `sub`, `aud`, `iss`, `iat`, `exp`, `jti`, `nbf`
- `kid` header supports key rotation without service disruption
- Algorithm-confusion attacks prevented — RS256 is the only accepted algorithm
- High-entropy `jti` claim on every token

### Client Model

Clients are first-class resources with fine-grained configuration:

- Confidential and public client types (`traditional`, `spa`, `mobile`, `m2m`)
- `token_endpoint_auth_method`: `client_secret_basic`, `client_secret_post`, `none`
- Per-client `grant_types` and `response_types` arrays
- Per-client access token and refresh token TTL overrides
- `require_consent` flag — consent screen can be skipped per client
- Client URIs — redirect URIs, logout URIs, logo, policy, terms of service

---

## Multi-Factor Authentication (MFA)

A layered MFA system covering all major second-factor methods.

- **Email OTP** — built-in OTP utility for email-based second factors
- **TOTP** (RFC 6238) — authenticator app enrollment, verification, and recovery codes
- **WebAuthn / Passkeys** (FIDO2) — passkey registration and assertion for passwordless and 2FA flows
- **SMS OTP** — one-time codes via configurable SMS provider (Twilio, SNS, Vonage)
- **Step-up authentication** — trigger re-authentication for sensitive operations
- Per-tenant MFA policy — required, optional, or step-up only
- `acr_values` and `amr` claims in tokens for downstream policy enforcement
- MFA reset flow with admin approval or recovery code

---

## Federation & External Identity Providers

Users can authenticate through any external identity provider. maintainerd-auth acts as the OAuth 2.0 / OIDC relying party and maps external identities to local user accounts.

- **OIDC upstream connectors** — Google, Microsoft, Apple, GitHub, GitLab, and any standard OIDC provider
- **Generic OAuth2 upstream connector** — connect any OAuth2-compatible provider
- **SAML 2.0 Service Provider** — integrate with enterprise SAML identity providers
- **SAML 2.0 IdP-initiated SSO**
- Per-tenant provider configuration — each tenant can have its own set of identity providers
- Identity linking — connect an existing local account to an external provider
- Identity unlinking — remove an external provider link from an account
- Just-in-time (JIT) user provisioning — create local users automatically on first federated login
- Attribute mapping — map upstream IdP claims to local user fields
- Home-realm discovery — route users to the correct IdP by email domain

---

## Provisioning — SCIM 2.0

Enterprise directory synchronization via the System for Cross-domain Identity Management standard.

- **SCIM 2.0 `/Users` resource** (RFC 7644) — full CRUD with filter support
- **SCIM 2.0 `/Groups` resource** — group provisioning and membership management
- **Service metadata** — `/ServiceProviderConfig`, `/ResourceTypes`, `/Schemas`
- **PATCH operations** — incremental attribute updates
- **Bulk endpoint** — batch create/update/delete operations
- Bearer-token client authentication for provisioning clients

---

## Multi-Tenancy, Organizations & RBAC

maintainerd-auth is multi-tenant by design. Every resource — users, clients, API keys, roles, permissions — is scoped to a tenant. This makes it suitable as the identity layer for multi-tenant SaaS platforms.

- **Tenant model** — full lifecycle management including status, branding, and settings
- **Tenant members** — user-to-tenant membership with role assignment
- **Roles** — named collections of permissions, assigned to users
- **Permissions** — fine-grained action identifiers scoped per service and API
- **Policies** — reusable permission sets attached to services
- **API keys** — scoped to specific APIs and permissions, suitable for M2M and integrations
- **Signup flows** — configurable registration flows with automatic role assignment per flow
- **Invite system** — invite users to a tenant with pre-assigned roles
- **Permission middleware** — every management API endpoint enforces declared permissions
- **Per-tenant settings** — rate limits, audit config, maintenance mode, feature flags
- **Per-tenant branding** — logo, colors, custom copy
- **Per-tenant security settings** — MFA policy, password policy, session lifetime, lockout config

---

## Session Management

Session state is tracked with refresh tokens, not server-side sessions. This keeps the server stateless while still enabling full session control.

- Cookie-based sessions — HTTP-only `access_token` cookie for browser clients
- Refresh token family tracking — every refresh token belongs to a lineage
- Reuse detection — using an old refresh token immediately revokes the entire family
- Active session listing per user
- Single session revocation by ID
- Revoke-all-sessions endpoint
- Session revocation on password change and permission change
- Configurable concurrent session limit
- Idle session timeout (sliding window) and absolute session lifetime cap

---

## Secret Management

Secrets are never hardcoded. maintainerd-auth ships with a pluggable secret provider abstraction that supports every major cloud secret store and local environments.

| Provider | Notes |
|----------|-------|
| Environment variables | Default for local and CI |
| AWS SSM Parameter Store | Native AWS integration |
| AWS Secrets Manager | Full secret versioning support |
| HashiCorp Vault | KV v2 and dynamic secrets |
| Azure Key Vault | Native Azure integration |
| GCP Secret Manager | Native GCP integration |
| Local file | Docker / Kubernetes mounted secrets |

Provider selection is controlled by a single `SECRET_PROVIDER` environment variable. A configurable prefix (`SECRET_PREFIX`) namespaces all secret keys.

---

## Password & Credential Policy

- Minimum length 8, maximum 128 characters
- Requires uppercase, lowercase, digit, and special character by default
- Common-password blocklist (most frequent breach passwords blocked)
- Configurable policy per tenant — length, character classes, blocklist
- Password breach check via HIBP k-anonymity API (no plaintext sent)
- Password history — prevent reuse of the last N passwords
- Password expiration and forced rotation policy per tenant
- Disposable email blocklist on signup

---

## Security Hardening

Security is not an afterthought. The following protections are built into the request path.

- **Brute-force protection** on login with exponential backoff
- **Account lockout** after N failed attempts (configurable per tenant)
- **Timing-safe operations** — pre-computed dummy bcrypt hash prevents user-existence timing leaks; `crypto/subtle` for secret comparison
- **CSRF protection** on all cookie-authenticated state-changing endpoints
- **IP-based rate limiting** on `/login`, `/oauth/token`, `/register`, `/forgot-password`
- **Global per-IP rate limit** on public port 8081 (Redis-backed sliding window)
- **CORS allow-list** — wildcard `*` with credentials is rejected
- **HSTS** (`Strict-Transport-Security` with preload)
- **Secure cookie flags** — `Secure`, `HttpOnly`, `SameSite`, `__Host-` prefix
- **Security headers** — `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Content-Security-Policy`
- **Input validation** — all DTOs validated at the boundary; redirect URIs validated against allowlist; `javascript:` and `data:` schemes rejected
- **Request size limits** — maximum body size enforced on all endpoints
- **DPoP-bound tokens** (RFC 9449) — proof-of-possession binding for access and refresh tokens
- **mTLS client authentication** (RFC 8705) — mutual TLS for confidential clients

---

## Cryptography & Key Management

- RSA-2048 minimum enforced at startup — the server refuses to start with a weaker key
- RS256 signing only — no algorithm negotiation, no confusion attacks
- `kid` header in every JWT for seamless key rotation
- Multi-key JWKS — active and retiring keys both served, enabling zero-downtime rotation
- Automatic key rotation runner — configurable rotation period
- KMS-backed signing — AWS KMS, GCP KMS, Azure Key Vault (private key never leaves the KMS)
- ECDSA (ES256) and EdDSA support alongside RS256
- Argon2id support as an alternative KDF alongside bcrypt
- PKCE S256 — SHA-256 code challenge, the only accepted method
- All refresh tokens and authorization codes are SHA-256 hashed at rest
- Crypto-secure random for all tokens, OTPs, JTIs, and IDs (`crypto/rand`)

---

## Audit Logging

Every authentication and authorization event is recorded. Audit logs are append-only, tenant-isolated, and structured for SIEM ingestion.

- Login success, login failure, account lockout
- Token issuance, token revocation, token introspection
- Consent grant and consent revocation
- All privileged admin actions — user CRUD, role changes, client CRUD, settings changes
- Retention runner with configurable per-tenant retention period
- PII redaction policy — configurable fields redacted before storage
- Structured event format for SIEM compatibility

---

## Observability

maintainerd-auth is instrumented end-to-end with OpenTelemetry.

### Tracing
Every request produces a distributed trace from the HTTP handler through the service layer to the database. Spans cover JWT generation and validation, bcrypt operations, GORM queries, and Redis operations.

### Metrics
Prometheus-compatible metrics exposed on the management port:
- HTTP request count, duration, and in-flight requests
- Auth-specific counters — logins ok/fail, tokens issued, MFA challenges
- Database query duration histograms
- Cache hit/miss ratios
- Build-info gauge (version, commit hash, build date)

### Logging
Structured JSON logs with request ID correlation, trace ID injection, and per-environment log level configuration. PII is redacted before log output.

---

## Webhooks

Outbound webhooks notify downstream systems of auth events in real time.

- Configurable webhook endpoints per tenant
- HMAC-SHA256 signature header on every delivery for authenticity verification
- Replay protection — timestamp + tolerance window
- Automatic retries with exponential backoff and a dead-letter queue
- Per-tenant event-type subscription model
- Webhook status and delivery history

---

## Email & SMS

Transactional messaging is pluggable. The email and SMS layers use provider abstractions so operators can swap backends without code changes.

**Email providers:** SMTP, AWS SES, SendGrid, Postmark, Mailgun, Resend

**SMS providers:** Twilio, AWS SNS, Vonage

- Async delivery via queue — auth flows never block on message delivery
- Retry with backoff on delivery failure
- Localized templates — i18n support for all transactional emails
- Email sandbox mode for development (no real sends)
- Fully customizable templates per tenant (logo, colors, copy)

---

## REST API

Both ports serve a versioned, consistent REST API.

- **Port 8080** — management / internal (VPN-only): tenant admin, user admin, IAM, client management, settings, audit
- **Port 8081** — public / identity: OAuth2, authentication flows, user self-service

All endpoints follow a consistent JSON envelope:

```json
{
  "success": true,
  "message": "...",
  "data": { }
}
```

OAuth errors follow RFC 6749 / RFC 6750 format exactly. The full API reference is documented in [`docs/apis/`](apis/).

- OpenAPI 3.1 spec served at `/openapi.json`
- Swagger UI / Redoc on the management port
- Pagination, filtering, and sorting on all list endpoints
- Problem Details (RFC 7807) for non-OAuth errors

---

## gRPC

A gRPC surface mirrors the REST API for high-performance service-to-service communication inside a cluster.

- Proto definitions for all core identity services
- Dedicated management-only gRPC port
- gRPC reflection for service discovery
- Auth, logging, recovery, and OpenTelemetry tracing interceptors
- `grpc.health.v1` health check service
- Optional gRPC-Gateway transcoding for dual REST + gRPC surface

---

## Architecture

maintainerd-auth follows clean architecture principles throughout.

```
cmd/          → entry points
internal/
  app/        → dependency injection and wiring
  rest/       → handlers, routes, middleware
  service/    → business logic
  repository/ → data access (GORM + PostgreSQL)
  model/      → database models
  dto/        → request / response types
  middleware/ → JWT auth, permissions, CORS, rate limiting, tracing
  config/     → environment + secret-manager configuration
  cache/      → Redis abstraction
  crypto/     → PKCE, OTP, hashing utilities
  security/   → brute-force, lockout, input sanitization
  apperror/   → typed error handling (RFC 6749-compliant for OAuth)
  jwt/        → RS256 token generation + validation
```

- **Dual-port server** — management (8080) and public identity (8081) on separate chi routers
- **Dual frontend repos** — `maintainerd-auth-console` (admin) and `maintainerd-auth-identity` (end-user portal)
- **PostgreSQL** — primary data store, GORM ORM, OTEL-instrumented driver
- **Redis** — session cache, rate limiting, token denylist
- **OpenTelemetry** — traces, metrics, and logs exported via OTLP
- **Pluggable secret management** — six cloud providers + env vars + local file
- **Background runners** — migrations, seeding, audit retention, key rotation

---

## Deployment

maintainerd-auth is designed to run anywhere containers run.

- Multi-stage Dockerfile with distroless final image and non-root user
- Kubernetes-ready — Helm chart with HPA, PodDisruptionBudget, and NetworkPolicy defaults
- `/healthz` — liveness probe (process-level, no downstream checks)
- `/readyz` — readiness probe (checks DB, Redis, JWKS loaded)
- `/livez` — Kubernetes startup + liveness
- Graceful shutdown on `SIGTERM` — drains in-flight requests before exit
- Build-info Prometheus gauge for version tracking across deployments
- Image signing with cosign / sigstore
- Vulnerability scanning in CI (trivy / grype)

---

## Release Roadmap

| Release | Theme |
|---------|-------|
| [v1.0.0](releases/v1.0.0.md) | Complete identity server — OAuth2/OIDC, MFA, SAML, SCIM, federation, gRPC, full security hardening |
| [v2.0.0](releases/v2.0.0.md) | Enterprise depth — LDAP/AD, ABAC, adaptive MFA, FAPI 2.0, SDKs, multi-region HA, compliance |
