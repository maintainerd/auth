# maintainerd-auth — Implementation Checklist

A complete checklist for an open-source identity & access management server
comparable to Keycloak / Zitadel / Auth0. Items already implemented in this
repository are pre-checked. Unchecked items are gaps or recommended additions.

Legend:
- `[x]` Implemented
- `[ ]` Missing / not yet implemented
- `[~]` Partially implemented (notes inline)
- `🔴` Critical — fix before public launch
- `🟡` High priority
- `🟢` Medium / nice-to-have
- `⚪` Low / optional

> Scope: REST API only. gRPC is intentionally a placeholder and is tracked at
> the bottom under "gRPC" but not gated as required.

---

## Table of Contents

1. [Architecture & Project Layout](#1-architecture--project-layout)
2. [Core Authentication Features](#2-core-authentication-features)
3. [OAuth 2.0 / OIDC Compliance](#3-oauth-20--oidc-compliance)
4. [Multi-Factor Authentication (MFA)](#4-multi-factor-authentication-mfa)
5. [Federation & External Identity Providers](#5-federation--external-identity-providers)
6. [Provisioning (SCIM)](#6-provisioning-scim)
7. [Tenancy, Organizations & RBAC](#7-tenancy-organizations--rbac)
8. [Session Management](#8-session-management)
9. [Cryptography & Key Management](#9-cryptography--key-management)
10. [Secret Management](#10-secret-management)
11. [Password & Credential Policy](#11-password--credential-policy)
12. [JWT & Token Security](#12-jwt--token-security)
13. [Cookies & Browser Security](#13-cookies--browser-security)
14. [CSRF / CORS / Security Headers](#14-csrf--cors--security-headers)
15. [Rate Limiting & Abuse Protection](#15-rate-limiting--abuse-protection)
16. [Input Validation & Sanitization](#16-input-validation--sanitization)
17. [Audit Logging & Auth Events](#17-audit-logging--auth-events)
18. [Observability — Logs, Metrics, Traces](#18-observability--logs-metrics-traces)
19. [Database & Migrations](#19-database--migrations)
20. [Cache (Redis)](#20-cache-redis)
21. [Email / SMS / Notifications](#21-email--sms--notifications)
22. [Webhooks](#22-webhooks)
23. [REST API Quality](#23-rest-api-quality)
24. [Configuration & Environment](#24-configuration--environment)
25. [Health, Readiness, Liveness](#25-health-readiness-liveness)
26. [Deployment & Operations](#26-deployment--operations)
27. [Testing](#27-testing)
28. [Go Best Practices & Code Quality](#28-go-best-practices--code-quality)
29. [Advanced Security Hardening](#29-advanced-security-hardening)
30. [Compliance (SOC2 / ISO27001 / GDPR / PCI / HIPAA)](#30-compliance)
31. [Documentation](#31-documentation)
32. [Admin & End-User UX](#32-admin--end-user-ux)
33. [gRPC (placeholder)](#33-grpc-placeholder)

---

## 1. Architecture & Project Layout

- [x] Standard Go layout (`cmd/`, `internal/`)
- [x] Clear separation: handler → service → repository → model
- [x] DTOs decoupled from DB models (`internal/dto/`)
- [x] Centralized dependency injection (`internal/app/`)
- [x] Generic base repository (`internal/repository/base.go`)
- [x] Typed application errors (`internal/apperror/`)
- [x] OAuth-specific RFC 6749 errors (`internal/apperror/oauth_error.go`)
- [x] Dual-port server (8080 management, 8081 public)
- [x] Background runners for migration / seeding / retention (`internal/runner/`)
- [x] Structured DTO validation package (`internal/valid/`)
- [x] Helper packages: `ptr`, `cookie`, `signedurl`, `crypto`
- [ ] 🟢 Single canonical `Config` struct injected into `app.New(cfg)` (currently package-level globals in `internal/config`)
- [ ] 🟢 Public router invariant test that fails the build if any management-only route is mounted on port 8081
- [ ] 🟢 `CONTRIBUTING.md`, `ARCHITECTURE.md`, `CODE_OF_CONDUCT.md`
- [ ] ⚪ ADRs (Architecture Decision Records) under `docs/adr/`
- [ ] ⚪ Domain-driven module split if the project grows beyond ~150 service files

---

## 2. Core Authentication Features

- [x] Username/email + password login (`internal/service/login.go`)
- [x] Internal login (no client_id required) — port 8080
- [x] Public login (client_id + provider_id required) — port 8081
- [x] Logout endpoint (clears cookies)
- [x] User registration (`internal/service/register.go`)
- [x] Configurable signup flows with role assignment (`signup_flow*`)
- [x] Forgot password (token issuance + email)
- [x] Reset password (token consumption)
- [x] Bcrypt password hashing
- [x] Initial bootstrap / setup flow (`internal/service/setup.go`)
- [x] Invite flow with role assignment
- [ ] 🟡 Email verification on signup (verification token + status flag)
- [ ] 🟡 Account recovery via secondary channel (SMS / backup codes)
- [ ] 🟡 Magic link / passwordless email login
- [ ] 🟡 SMS one-time code login
- [ ] 🟢 Username/email change with re-verification
- [ ] 🟢 Account deletion / GDPR right-to-erasure flow
- [ ] 🟢 Account export (GDPR data portability)
- [ ] 🟢 Force-password-change on next login flag
- [ ] 🟢 Password expiry / rotation policy
- [ ] ⚪ Anonymous / guest user upgrade flow

---

## 3. OAuth 2.0 / OIDC Compliance

### 3.1 Endpoints
- [x] POST /oauth/token (RFC 6749)
- [x] GET  /oauth/authorize (RFC 6749)
- [x] POST /oauth/revoke (RFC 7009)
- [x] POST /oauth/introspect (RFC 7662) — management port only
- [x] GET  /oauth/userinfo (OIDC Core 5.3)
- [x] GET  /.well-known/openid-configuration (RFC 8414 / OIDC Discovery)
- [x] GET  /.well-known/jwks.json (RFC 7517)
- [x] Consent challenge + decision endpoints
- [x] List & revoke consent grants per user
- [ ] 🟢 GET /.well-known/oauth-authorization-server (RFC 8414, separate from OIDC)
- [ ] 🟢 POST /oauth/par — Pushed Authorization Requests (RFC 9126)
- [ ] 🟢 POST /oauth/device_authorization — Device Authorization Grant (RFC 8628)
- [ ] 🟢 POST /oauth/register — Dynamic Client Registration (RFC 7591)
- [ ] 🟢 Token Exchange grant (RFC 8693)
- [ ] 🟢 Backchannel logout endpoint (OIDC Back-Channel Logout 1.0)
- [ ] 🟢 RP-Initiated Logout (/oauth/end_session) (OIDC Session Mgmt 1.0)
- [ ] 🟢 CIBA — Client-Initiated Backchannel Authentication
- [ ] ⚪ check_session_iframe (front-channel session monitoring)

### 3.2 Grants
- [x] authorization_code with PKCE S256 (RFC 7636)
- [x] refresh_token with rotation, family revocation, reuse detection (RFC 6749 §6)
- [x] client_credentials (RFC 6749 §4.4)
- [ ] 🟢 device_code grant (RFC 8628)
- [ ] 🟢 token-exchange grant (RFC 8693)
- [ ] ⚪ password grant (legacy; generally avoid)

### 3.3 Client model
- [x] Confidential / public client types
- [x] token_endpoint_auth_method: client_secret_basic, client_secret_post, none
- [x] grant_types and response_types arrays
- [x] Per-client access/refresh token TTL override
- [x] require_consent flag
- [x] Client URIs (redirect URIs, logo, policy, tos)
- [ ] 🔴 **Hash client secrets at rest** — currently plaintext `*string` compared with `!=`
- [ ] 🔴 **Constant-time comparison** for client secret check (`crypto/subtle`)
- [ ] 🟡 Show client_secret only once at creation, return masked thereafter
- [ ] 🟡 Client secret rotation API (issue new + grace window for old)
- [ ] 🟡 Per-client allowed scopes list
- [ ] 🟢 Per-client allowed grant types enforcement at token endpoint
- [ ] 🟢 private_key_jwt and client_secret_jwt auth methods (RFC 7523)
- [ ] 🟢 tls_client_auth / self_signed_tls_client_auth (RFC 8705 mTLS)
- [ ] 🟢 Software statement / SBOM attestation for clients
- [ ] ⚪ Sector identifier URI for pairwise subjects

### 3.4 Tokens
- [x] Access token (JWT, RS256, 15 min)
- [x] ID token with profile claims and nonce
- [x] Refresh token (random, hashed at rest, 7 days, rotated)
- [x] Authorization code (random, hashed at rest, single-use)
- [x] jti claim with high entropy
- [x] kid header for key rotation
- [x] Algorithm-confusion attack protection (RS256 enforced)
- [ ] 🟡 Access token denylist (Redis, TTL = access_token_ttl) keyed by jti
- [ ] 🟡 HMAC-SHA256 with server-side pepper for refresh/auth-code hashing
- [ ] 🟢 Pairwise subject identifiers
- [ ] 🟢 Encrypted ID tokens (JWE)
- [ ] 🟢 Reference (opaque) access tokens as alternative to JWT
- [ ] 🟢 Audience-restricted access tokens (aud per resource server)

### 3.5 Scopes & Claims
- [x] Standard OIDC scopes: openid, profile, email, offline_access
- [x] Standard OIDC claims (sub, email, name, picture, address, phone, etc.)
- [ ] 🟡 Scope-to-claim mapping per client
- [ ] 🟡 Custom claim mappers (per tenant/client)
- [ ] 🟢 Resource indicators (resource parameter, RFC 8707)
- [ ] 🟢 Rich Authorization Requests (RAR, RFC 9396)
- [ ] ⚪ Claims request parameter (OIDC Core 5.5)

### 3.6 Authorization endpoint hardening
- [x] PKCE required for authorization_code
- [x] Redirect URI exact-match validation
- [x] Authorization code single-use enforcement
- [ ] 🟡 state parameter requirement enforcement
- [ ] 🟡 nonce requirement when response_type includes id_token
- [ ] 🟢 Reject loopback redirects in production for confidential clients
- [ ] 🟢 Deny prompt=none for unauthenticated users
- [ ] 🟢 PAR-only mode for high-security clients
- [ ] 🟢 JAR (signed Request Objects, RFC 9101)
- [ ] 🟢 JARM (signed Response Mode, FAPI)

---

## 4. Multi-Factor Authentication (MFA)

- [x] Email OTP utility (`internal/crypto/otp.go`)
- [ ] 🟡 TOTP (RFC 6238) enrollment + verification
- [ ] 🟡 TOTP recovery / backup codes (one-time use)
- [ ] 🟡 WebAuthn / FIDO2 (passkeys) registration
- [ ] 🟡 WebAuthn login / 2FA assertion
- [ ] 🟡 Step-up authentication (re-auth required for sensitive ops)
- [ ] 🟢 acr_values and amr claim support in tokens
- [ ] 🟢 SMS OTP (with rate-limit + cost guard)
- [ ] 🟢 Email magic-link as 2nd factor
- [ ] 🟢 Push notification 2FA
- [ ] 🟢 Hardware token (U2F) — covered by WebAuthn but track separately
- [ ] 🟢 Per-tenant MFA policy (required / optional / risk-based)
- [ ] 🟢 Risk-based / adaptive MFA (new device, new IP, geo-velocity)
- [ ] 🟢 MFA reset flow (admin-approved or recovery code)
- [ ] ⚪ Biometric / platform authenticator preference flag

---

## 5. Federation & External Identity Providers

- [x] IdentityProvider model + repository + service
- [x] Per-tenant provider configuration
- [x] User identity linking (`user_identity` model)
- [ ] 🟡 OIDC upstream provider (Google, Microsoft, Apple, GitHub, GitLab)
- [ ] 🟡 Generic OAuth2 upstream connector
- [ ] 🟡 Identity linking flow (UI + API) for existing users
- [ ] 🟡 Identity unlinking
- [ ] 🟢 SAML 2.0 SP (Service Provider)
- [ ] 🟢 SAML 2.0 IdP-initiated SSO
- [ ] 🟢 LDAP / Active Directory bind
- [ ] 🟢 Kerberos / SPNEGO
- [ ] 🟢 Just-in-time user provisioning from upstream IdP
- [ ] 🟢 Attribute mapping (upstream → local user fields)
- [ ] 🟢 Home-realm discovery (HRD) by email domain
- [ ] ⚪ OAuth2 token exchange against upstream IdP

---

## 6. Provisioning (SCIM)

- [ ] 🟢 SCIM 2.0 /Users resource (RFC 7644)
- [ ] 🟢 SCIM 2.0 /Groups resource
- [ ] 🟢 SCIM 2.0 /ServiceProviderConfig, /ResourceTypes, /Schemas
- [ ] 🟢 SCIM 2.0 PATCH operations
- [ ] 🟢 SCIM 2.0 bulk endpoint
- [ ] 🟢 SCIM Bearer-token client authentication
- [ ] 🟢 Outbound SCIM push to downstream apps
- [ ] ⚪ SCIM filter expression parser (eq, sw, co, etc.)

---

## 7. Tenancy, Organizations & RBAC

- [x] Multi-tenant `Tenant` model
- [x] `TenantMember` for user-to-tenant membership
- [x] `TenantService` and `TenantSetting` per tenant
- [x] `Role` and `UserRole` (many-to-many)
- [x] `Permission` model with role permission mapping
- [x] `Policy` and `ServicePolicy`
- [x] Permission middleware (`internal/middleware/permission_middleware.go`)
- [x] User-context middleware joins JWT to DB user
- [x] API key model with API/permission scoping
- [x] Invite system with role pre-assignment
- [x] Setup / bootstrap flow for first-run
- [ ] 🟡 Hierarchical orgs / sub-organizations / projects
- [ ] 🟡 Group model (separate from role) for human grouping
- [ ] 🟡 ABAC (attribute-based) policy evaluation alongside RBAC
- [ ] 🟢 Tenant isolation invariant tests (cross-tenant access denied)
- [ ] 🟢 Per-tenant feature flags
- [ ] 🟢 Tenant-scoped API rate limits
- [ ] 🟢 Tenant deletion with cascade or soft-delete with retention
- [ ] 🟢 Tenant export / clone / migrate
- [ ] ⚪ Delegated admin (admin-of-organization)

---

## 8. Session Management

- [x] User-token model (`user_token`) for refresh tracking
- [x] Refresh token rotation with family
- [x] Refresh token reuse detection → family revocation
- [x] Cookie-based session (HTTP-only, access_token cookie)
- [x] User-context cache invalidation
- [x] Concurrent session limit constant (5) in `security` package
- [ ] 🟡 List active sessions per user (API + UI)
- [ ] 🟡 Revoke single session by ID
- [ ] 🟡 Revoke-all-sessions endpoint
- [ ] 🟡 Session-revoked-on-password-change
- [ ] 🟡 Session-revoked-on-permission-change
- [ ] 🟢 Idle session timeout (sliding)
- [ ] 🟢 Absolute session lifetime cap
- [ ] 🟢 Device fingerprinting / device registration
- [ ] 🟢 Trusted-device management (skip MFA on remembered devices)
- [ ] 🟢 Geo-/IP-anomaly detection on session creation
- [ ] 🟢 Enforce MaxConcurrentSessions in login flow (currently constant exists but not enforced)
- [ ] ⚪ Session impersonation / "view as user" for admins (audit-logged)

---

## 9. Cryptography & Key Management

- [x] RSA-2048 minimum key strength enforced (`MinKeySize` in `jwt`)
- [x] Key-pair consistency check at startup
- [x] RS256 signing only (algorithm-confusion-safe)
- [x] `kid` in JWT header
- [x] Crypto-secure random for OTP, JTI, IDs (`crypto/rand`)
- [x] PKCE S256 implementation (`internal/crypto/pkce.go`)
- [x] SHA-256 hashing for refresh tokens and authorization codes
- [x] Bcrypt for password hashing
- [x] Pre-computed dummy bcrypt hash for timing-safe operations
- [ ] 🔴 Propagate request `ctx` into `HashPassword` span (currently uses `context.Background()`)
- [ ] 🟡 Argon2id support as KDF (configurable algo)
- [ ] 🟡 Bcrypt cost ≥ 12 (currently `DefaultCost` = 10)
- [ ] 🟡 Multi-key JWKS (active + retiring keys, both served via JWKS)
- [ ] 🟡 Automatic key rotation runner (configurable period, e.g. 90 days)
- [ ] 🟡 KMS-backed signing (AWS KMS / GCP KMS / Azure Key Vault) — sign without exporting private key
- [ ] 🟢 ECDSA (ES256) and EdDSA support in addition to RS256
- [ ] 🟢 HMAC pepper for token-hash storage
- [ ] 🟢 Envelope encryption for stored secrets (client_secret, MFA seed, SMTP pass)
- [ ] 🟢 Field-level encryption for PII columns
- [ ] ⚪ HSM (PKCS#11) signing backend

---

## 10. Secret Management

- [x] Pluggable secret-provider abstraction (`internal/config/secret_manager.go`)
- [x] Environment-variable provider
- [x] AWS SSM Parameter Store provider
- [x] AWS Secrets Manager provider
- [x] HashiCorp Vault provider
- [x] Azure Key Vault provider
- [x] GCP Secret Manager provider
- [x] Provider selection via `SECRET_PROVIDER` env var
- [x] Configurable secret name prefix (`SECRET_PREFIX`)
- [x] Tests for every provider
- [ ] 🟡 Secret refresh / hot-reload without restart
- [ ] 🟢 Secret-version pinning + rollback
- [ ] 🟢 Local file provider (Docker/K8s mounted secret)
- [ ] 🟢 Doppler / 1Password Connect / Infisical providers
- [ ] ⚪ Audit log entry on every secret read

---

## 11. Password & Credential Policy

- [x] Minimum length 8, maximum 128
- [x] Requires upper, lower, digit, special character
- [x] Common-password substring blocklist
- [x] Bcrypt hashing
- [ ] 🟡 Configurable password policy per tenant (length, classes, blocklist)
- [ ] 🟡 Password breach check via HIBP k-anonymity API
- [ ] 🟡 Password history (prevent last N reuse)
- [ ] 🟡 Password expiration / forced rotation policy
- [ ] 🟢 zxcvbn / passphrase-strength scoring
- [ ] 🟢 Compromised-credentials check on every login
- [ ] 🟢 Password-strength meter feedback in API
- [ ] 🟢 Disposable-email blocklist on signup
- [ ] ⚪ Username blocklist (admin, root, etc.)

---

## 12. JWT & Token Security

- [x] RS256 signing only
- [x] Required claims: sub, aud, iss, iat, exp, jti
- [x] `nbf` not-before claim
- [x] Per-token-type `token_type` claim
- [x] Custom `client_id` and `provider_id` claims for multi-tenant routing
- [x] Validation rejects unknown algorithms
- [x] Validation rejects unknown `kid`
- [x] Validation enforces all required claims
- [x] OTEL spans on token generation and validation
- [ ] 🟡 Multi-`kid` lookup (current implementation only matches one kid env var)
- [ ] 🟡 Clock-skew tolerance configurable (currently library default)
- [ ] 🟡 Audience whitelist enforcement (validate `aud` against expected resource)
- [ ] 🟢 Issuer whitelist enforcement (`iss` exact match check)
- [ ] 🟢 Token replay detection via `jti` cache
- [ ] 🟢 Per-claim PII redaction in logs
- [ ] ⚪ x5c / x5t certificate-bound JWTs

---

## 13. Cookies & Browser Security

- [x] HTTP-only cookies for `access_token`
- [x] Cookie-based fallback when no Authorization header
- [x] Cookie utility package (`internal/cookie`)
- [ ] 🟡 `Secure: true` enforced in production
- [ ] 🟡 `SameSite` configurable (Lax for redirect flows, Strict for management)
- [ ] 🟡 `__Host-` cookie prefix for session cookie
- [ ] 🟡 Configurable cookie `Domain` per environment
- [ ] 🟢 Separate refresh-token cookie (HTTP-only, `__Host-`, narrow path)
- [ ] 🟢 CSRF double-submit cookie pattern for cookie-auth flows
- [ ] 🟢 Hardened cookie API replacing reflection-based helpers
- [ ] ⚪ Cookie partitioning (CHIPS) for embedded contexts

---

## 14. CSRF / CORS / Security Headers

- [x] CORS middleware (`internal/middleware/cors.go`)
- [x] Security-headers middleware (`internal/middleware/security_headers.go`)
- [ ] 🔴 CSRF protection on cookie-authenticated state-changing endpoints (login form, consent decision)
- [ ] 🟡 Per-environment CORS allow-list (no wildcard `*` with credentials)
- [ ] 🟡 Strict `Content-Security-Policy` for any HTML rendered (login pages)
- [ ] 🟡 `Strict-Transport-Security` (HSTS) with preload
- [ ] 🟡 `X-Content-Type-Options: nosniff`
- [ ] 🟡 `X-Frame-Options: DENY` (or CSP `frame-ancestors`)
- [ ] 🟡 `Referrer-Policy: strict-origin-when-cross-origin`
- [ ] 🟢 `Permissions-Policy` minimal allow-list
- [ ] 🟢 `Cross-Origin-Opener-Policy` and `Cross-Origin-Resource-Policy`
- [ ] 🟢 Trusted-Types CSP directive for any rendered UI
- [ ] ⚪ Subresource Integrity for any external JS/CSS

---

## 15. Rate Limiting & Abuse Protection

- [x] Brute-force protection on login (`internal/security/bruteforce.go`)
- [x] Account lockout after N failed attempts (`internal/security/lockout.go`)
- [x] Pre-computed dummy bcrypt to mask user-existence timing
- [ ] 🔴 IP-based rate limiting on `/login`, `/oauth/token`, `/forgot-password`, `/register`
- [ ] 🔴 Global request rate limiter (per IP) on public port 8081
- [ ] 🟡 Distributed rate limiter (Redis-backed token bucket / sliding window)
- [ ] 🟡 Per-client rate limits on `/oauth/token`
- [ ] 🟡 CAPTCHA / Turnstile / hCaptcha integration after N failures
- [ ] 🟡 Slow-loris / request-body size limits at HTTP server level
- [ ] 🟢 Connection rate limit per IP (SYN flood mitigation, often handled at LB)
- [ ] 🟢 Anomaly detection (impossible travel, new-device alerts)
- [ ] 🟢 Honeypot fields on signup/login forms
- [ ] 🟢 Adaptive auth (step-up MFA on risk score)
- [ ] ⚪ Web Application Firewall (WAF) integration / rules

---

## 16. Input Validation & Sanitization

- [x] Centralized validation package (`internal/valid/`)
- [x] DTO-level validation with struct tags (`go-playground/validator`)
- [x] Email and password format validation
- [x] OAuth parameter validation (`oauth_authorize.go`, `oauth_token.go`)
- [ ] 🟡 Maximum body size middleware (e.g. 1 MB) on all endpoints
- [ ] 🟡 Strict Content-Type enforcement (`application/json` or `application/x-www-form-urlencoded`)
- [ ] 🟡 Reject unknown JSON fields (`DisallowUnknownFields`) where appropriate
- [ ] 🟢 URI/redirect validation rejecting `javascript:`, `data:`, `vbscript:` schemes
- [ ] 🟢 Unicode normalization (NFKC) for usernames/emails
- [ ] 🟢 Homograph / confusable detection on usernames
- [ ] 🟢 Output encoding helpers for any HTML rendering
- [ ] ⚪ Schema-driven validation via OpenAPI

---

## 17. Audit Logging & Auth Events

- [x] Audit-log model and repository
- [x] Auth-event model (login success/failure, token issued, etc.)
- [x] Retention runner (`internal/runner/audit_retention.go`)
- [x] Recording on login success, login failure, lockout
- [ ] 🟡 Audit every privileged admin action (user CRUD, role changes, client CRUD)
- [ ] 🟡 Audit consent grant / revoke / token revoke
- [ ] 🟡 Tamper-evident chain (HMAC chained over previous record's hash)
- [ ] 🟡 Append-only storage with no UPDATE/DELETE permission
- [ ] 🟢 Streaming export to SIEM (S3 / Kinesis / Kafka / GCS)
- [ ] 🟢 Per-tenant audit isolation
- [ ] 🟢 PII redaction policy in audit payloads
- [ ] 🟢 Standardized event taxonomy (CADF / OpenTelemetry events)
- [ ] ⚪ Cryptographic timestamping (RFC 3161) for compliance archives

---

## 18. Observability — Logs, Metrics, Traces

### 18.1 Logging
- [x] Structured logging (`internal/logger`)
- [x] Request ID middleware (`internal/middleware/request_id.go`)
- [x] Recovery middleware with stack capture
- [x] Trace-correlated log fields
- [ ] 🟡 PII redaction layer (emails, tokens, IPs) before log output
- [ ] 🟡 Log sampling for high-volume routes
- [ ] 🟡 Per-environment log level via config
- [ ] 🟢 OpenTelemetry log signal (OTLP) export

### 18.2 Metrics
- [x] OpenTelemetry meter provider
- [ ] 🔴 HTTP server metrics (request count, duration, in-flight) via `otelhttp`
- [ ] 🟡 Auth-specific counters: logins ok/fail, tokens issued, MFA challenges
- [ ] 🟡 Database query duration histogram
- [ ] 🟡 Cache hit/miss counters
- [ ] 🟢 Go runtime metrics (goroutines, GC, memory)
- [ ] 🟢 Build-info gauge (version, commit, date)
- [ ] 🟢 Prometheus `/metrics` endpoint on management port

### 18.3 Tracing
- [x] OpenTelemetry tracer provider with OTLP exporter
- [x] Spans on service-layer operations
- [x] Spans on JWT generation/validation
- [x] Spans on bcrypt password operations
- [x] GORM tracing instrumentation (`go.nhat.io/otelsql`)
- [ ] 🔴 HTTP server middleware via `otelhttp.NewHandler` for full trace context propagation
- [ ] 🔴 Pass request `ctx` into `HashPassword` (currently uses `context.Background()` and detaches the trace)
- [ ] 🟡 Outbound HTTP client instrumentation (federation, webhooks)
- [ ] 🟡 Redis client instrumentation
- [ ] 🟡 Trace sampling configuration (head + tail)
- [ ] 🟢 Span attributes follow OpenTelemetry semantic conventions
- [ ] 🟢 Exemplars linking metrics ↔ traces

---

## 19. Database & Migrations

- [x] PostgreSQL via GORM
- [x] Auto-migrate runner (`internal/runner/migrate.go`)
- [x] Seed runner (`internal/runner/seed.go`)
- [x] Generic base repository
- [x] Soft-delete and audit timestamps via `model.Base`
- [x] OTEL-instrumented driver (`go.nhat.io/otelsql`)
- [ ] 🟡 Versioned migrations tool (golang-migrate / goose / atlas) instead of GORM auto-migrate in prod
- [ ] 🟡 Forward + rollback migration scripts
- [ ] 🟡 Connection-pool tuning explicit in config (max open, max idle, lifetime)
- [ ] 🟡 Read-replica routing for read-heavy endpoints (jwks, userinfo)
- [ ] 🟢 Statement timeout enforced (`SET statement_timeout`)
- [ ] 🟢 Database SSL/TLS required in production
- [ ] 🟢 Indexes audited (covering indexes on `email`, `username`, `client_id`, `provider_id`)
- [ ] 🟢 Partitioning strategy for `audit_log` and `auth_event` (monthly)
- [ ] 🟢 Backup + point-in-time recovery procedure documented
- [ ] ⚪ Logical replication / CDC for downstream analytics

---

## 20. Cache (Redis)

- [x] Redis client with TLS support (`internal/cache/cache.go`)
- [x] User-context cache invalidation
- [x] Cache abstraction interface
- [ ] 🟡 Redis OTEL instrumentation (spans + metrics)
- [ ] 🟡 Sentinel / Cluster support for HA
- [ ] 🟡 Per-key TTL audit (no unbounded keys)
- [ ] 🟢 Circuit breaker around cache reads (degrade gracefully)
- [ ] 🟢 Stampede protection (singleflight)
- [ ] 🟢 Encrypted values at rest in Redis (envelope encryption)
- [ ] ⚪ KeyDB / Dragonfly compatibility verified

---

## 21. Email / SMS / Notifications

- [x] SMTP email service (`internal/service/email.go`, `internal/notification/`)
- [x] Email templates (forgot password, invite)
- [ ] 🟡 Pluggable email provider (SES / SendGrid / Postmark / Mailgun / Resend)
- [ ] 🟡 Async email delivery via queue (avoid blocking auth flows)
- [ ] 🟡 Email delivery retry with backoff
- [ ] 🟡 SMS provider abstraction (Twilio / SNS / Vonage) for OTP
- [ ] 🟢 Push-notification provider (APNs / FCM) for MFA push
- [ ] 🟢 Localized email templates (i18n)
- [ ] 🟢 DMARC / SPF / DKIM documentation for sender domain
- [ ] 🟢 Email sandbox mode for development
- [ ] ⚪ Slack / Teams notifier for high-severity events

---

## 22. Webhooks

- [ ] 🟡 Outbound webhooks for auth events (login, signup, mfa-enrolled, etc.)
- [ ] 🟡 HMAC-SHA256 signature header for webhook authenticity
- [ ] 🟡 Replay protection (timestamp + tolerance window)
- [ ] 🟡 Retries with exponential backoff + dead-letter queue
- [ ] 🟢 Per-tenant webhook configuration
- [ ] 🟢 Webhook delivery dashboard (recent attempts, status)
- [ ] 🟢 Event-type subscription model
- [ ] ⚪ CloudEvents-compatible payload format

---

## 23. REST API Quality

- [x] Versioned route mounting (`internal/rest/route/`)
- [x] Consistent JSON error envelope via `apperror`
- [x] OAuth-spec-compliant error format (`oauth_error.go`)
- [x] Standard middleware: logger, recovery, request ID
- [ ] 🟡 OpenAPI 3.1 spec generated and served on `/openapi.json`
- [ ] 🟡 Swagger UI / Redoc on management port only
- [ ] 🟡 Pagination, sorting, filtering conventions documented and applied
- [ ] 🟡 ETag / If-None-Match for cacheable resources (jwks, discovery)
- [ ] 🟡 Idempotency-Key support on POSTs that create resources
- [ ] 🟢 Problem Details (RFC 7807) response shape for non-OAuth errors
- [ ] 🟢 API deprecation headers (`Sunset`, `Deprecation`)
- [ ] 🟢 Generated SDK clients (go, ts, python)
- [ ] ⚪ HATEOAS links where appropriate

---

## 24. Configuration & Environment

- [x] Env-driven config (`internal/config/`)
- [x] Secret-manager-driven config
- [x] Per-port config (management vs identity)
- [ ] 🟡 Single canonical `Config` struct passed via DI (no package-level globals)
- [ ] 🟡 Validation of required env at boot (fail-fast with clear error)
- [ ] 🟡 Defaults documented and tested
- [ ] 🟡 `.env.example` kept in sync with code
- [ ] 🟢 Hot-reload of non-secret config via SIGHUP
- [ ] 🟢 Feature flags (LaunchDarkly / OpenFeature / internal)
- [ ] 🟢 Config schema doc auto-generated from struct tags

---

## 25. Health, Readiness, Liveness

- [x] `/healthz` endpoint
- [ ] 🟡 `/readyz` (checks DB + Redis + secret-manager + JWKS loaded)
- [ ] 🟡 `/livez` (process-level only, never depends on downstream)
- [ ] 🟡 Startup probe distinct from readiness
- [ ] 🟢 Health check JSON includes version + dependency status
- [ ] 🟢 Component-level health propagated to metrics

---

## 26. Deployment & Operations

- [x] Dockerfile (multi-stage build expected)
- [ ] 🟡 Distroless / minimal base image
- [ ] 🟡 Non-root container user
- [ ] 🟡 Read-only root filesystem
- [ ] 🟡 Kubernetes manifests / Helm chart
- [ ] 🟡 Graceful shutdown on SIGTERM (drain in-flight requests)
- [ ] 🟡 Pod-disruption budget + HPA defaults
- [ ] 🟡 NetworkPolicy isolating management port from public ingress
- [ ] 🟢 Terraform / Pulumi modules for cloud deploy
- [ ] 🟢 Blue/green or canary deploy strategy documented
- [ ] 🟢 Image signing (cosign / sigstore)
- [ ] 🟢 SBOM generated at build (syft / cyclonedx)
- [ ] 🟢 Vulnerability scanning in CI (trivy / grype)
- [ ] ⚪ FIPS-validated build option

---

## 27. Testing

- [x] Unit tests for `apperror`
- [x] Unit tests for `secret_manager` providers
- [x] Unit tests for `valid` package
- [x] Unit tests for selected services
- [ ] 🟡 ≥ 70% coverage on `internal/service/`
- [ ] 🟡 ≥ 70% coverage on `internal/security/`
- [ ] 🟡 Integration tests with real Postgres + Redis (testcontainers-go)
- [ ] 🟡 End-to-end OAuth flow tests (authorize → token → introspect → revoke)
- [ ] 🟡 Conformance test against OpenID Foundation suite
- [ ] 🟢 Fuzz tests on token parsing, redirect-URI validation, PKCE verifier
- [ ] 🟢 Load tests (k6 / vegeta) on `/oauth/token`
- [ ] 🟢 Mutation testing (go-mutesting) for critical paths
- [ ] 🟢 Race-detector enabled in CI (`go test -race`)
- [ ] 🟢 Golden-file tests for JWT + JWKS shapes
- [ ] ⚪ Property-based tests (gopter)

---

## 28. Go Best Practices & Code Quality

- [x] Go 1.26.x toolchain
- [x] `internal/` for non-public packages
- [x] Package names lowercase, single-word, no underscores
- [x] Errors wrapped with context (`fmt.Errorf("...: %w", err)`)
- [x] Dependency injection via constructors
- [ ] 🟡 `golangci-lint` with strict config in CI (govet, staticcheck, errcheck, gosec, revive, gocritic)
- [ ] 🟡 `gosec` clean (no high/medium findings)
- [ ] 🟡 `go vet` and `staticcheck` clean
- [ ] 🟡 No `interface{}` / `any` in public APIs without justification
- [ ] 🟡 Context as first parameter on every I/O-bound function
- [ ] 🟡 No `context.Background()` inside request-scoped code paths (except documented exceptions)
- [ ] 🟢 Errors are values; no `panic` outside `init`
- [ ] 🟢 No global mutable state outside `internal/config` (which itself should be replaced by DI)
- [ ] 🟢 Public exported identifiers have doc comments
- [ ] 🟢 `go-fmt`, `goimports` enforced via pre-commit
- [ ] 🟢 Consistent naming: `New<Type>` constructors, `<Verb><Noun>` methods
- [ ] 🟢 No circular imports between `service` ↔ `repository`
- [ ] 🟢 Build tags for optional providers (e.g. `aws`, `gcp`, `azure`) to shrink binary
- [ ] ⚪ Race detector enabled in dev container

---

## 29. Advanced Security Hardening

- [ ] 🟡 mTLS between maintainerd-auth and downstream services
- [ ] 🟡 Mutual-TLS-bound access tokens (RFC 8705)
- [ ] 🟡 DPoP-bound access/refresh tokens (RFC 9449)
- [ ] 🟡 FAPI 2.0 baseline conformance
- [ ] 🟢 Token binding via client-cert thumbprint
- [ ] 🟢 SBOM published per release
- [ ] 🟢 Reproducible builds
- [ ] 🟢 Supply-chain hardening (SLSA level ≥ 3)
- [ ] 🟢 Dependency review automation (Dependabot / Renovate + auto-merge for patch)
- [ ] 🟢 Static analysis: CodeQL workflow
- [ ] 🟢 Secret scanning: gitleaks pre-commit + CI
- [ ] 🟢 Penetration test annually (and pre-launch)
- [ ] 🟢 Bug bounty program / security.txt
- [ ] 🟢 Threat model document (STRIDE)
- [ ] 🟢 Incident response runbook
- [ ] ⚪ Confidential-computing (TEE) signing path

---

## 30. Compliance

### 30.1 SOC 2 Type II
- [ ] 🟡 Access reviews quarterly (admin role list)
- [ ] 🟡 Change management evidence (PR reviews + CI gates)
- [ ] 🟡 Vendor / sub-processor list maintained
- [ ] 🟡 Encryption-in-transit policy documented (TLS 1.2+)
- [ ] 🟡 Encryption-at-rest policy documented (DB, Redis, backups)
- [ ] 🟡 Backup + restore drills logged

### 30.2 ISO 27001
- [ ] 🟡 Asset inventory + data classification
- [ ] 🟡 Risk register with treatment plans
- [ ] 🟡 Statement of Applicability (Annex A controls)
- [ ] 🟡 Internal audit cadence

### 30.3 GDPR
- [ ] 🟡 Right to access (data export endpoint)
- [ ] 🟡 Right to erasure (account deletion + cascade)
- [ ] 🟡 Right to rectification
- [ ] 🟡 Consent records auditable
- [ ] 🟡 Data Processing Agreement template
- [ ] 🟡 Configurable data residency (EU-only deployment option)
- [ ] 🟢 Privacy notice + cookie banner template

### 30.4 PCI-DSS (if storing cardholder data — generally avoid)
- [ ] ⚪ N/A by design — auth server should not touch PAN

### 30.5 HIPAA (if used in healthcare)
- [ ] 🟢 BAA-friendly deployment guide
- [ ] 🟢 PHI minimization in claims and logs
- [ ] 🟢 Audit log retention ≥ 6 years configurable

---

## 31. Documentation

- [x] README with quick-start
- [ ] 🟡 `ARCHITECTURE.md` (component diagram, request flow)
- [ ] 🟡 `SECURITY.md` (threat model, reporting, supported versions)
- [ ] 🟡 `CONTRIBUTING.md`
- [ ] 🟡 `CODE_OF_CONDUCT.md`
- [ ] 🟡 OpenAPI spec (machine-readable) for both ports
- [ ] 🟡 Operator runbook (deploy, rotate keys, scale, recover)
- [ ] 🟡 Migration guide from Keycloak / Auth0 / Zitadel
- [ ] 🟢 ADRs under `docs/adr/`
- [ ] 🟢 Per-feature how-to guides (configure social IdP, SCIM, MFA, etc.)
- [ ] 🟢 Public docs site (mkdocs / docusaurus) with versioned releases
- [ ] 🟢 Changelog following Keep-A-Changelog
- [ ] ⚪ Video tutorials

---

## 32. Admin & End-User UX

- [ ] 🟢 Hosted login UI (server-rendered, themeable)
- [ ] 🟢 Hosted consent UI
- [ ] 🟢 Hosted MFA enrollment / challenge UI
- [ ] 🟢 Account self-service portal (email change, MFA, devices, sessions)
- [ ] 🟢 Admin console (users, clients, tenants, audit log viewer)
- [ ] 🟢 Themable templates per tenant (logo, colors, copy)
- [ ] 🟢 i18n (at minimum: en, es, fr, de, ja)
- [ ] 🟢 Accessibility (WCAG 2.2 AA)
- [ ] ⚪ Dark-mode default

---

## 33. gRPC (placeholder)

> Out of scope for the current milestone. Tracked here only to keep the inventory complete.

- [ ] ⚪ gRPC service definitions (`api/proto/`)
- [ ] ⚪ Generated stubs build target
- [ ] ⚪ gRPC reflection on management port only
- [ ] ⚪ gRPC interceptors mirroring REST middleware (auth, logging, tracing, recovery)
- [ ] ⚪ gRPC health-check service
- [ ] ⚪ gRPC-Gateway transcoding to REST (if dual surface desired)

---

## Quick-Start Recommendation

If you address these eight items before public exposure of port 8081 you will close the most acute risks:

1. Hash client secrets at rest + constant-time compare (§3.3, §1 of original report)
2. Propagate `ctx` into `HashPassword` (§9, §18.3)
3. IP-based rate limiting on `/login` and `/oauth/token` (§15)
4. `otelhttp.NewHandler` server instrumentation (§18.2, §18.3)
5. CSRF protection on cookie-authenticated state-changing endpoints (§14)
6. CORS allow-list (no wildcard with credentials) and HSTS (§14)
7. Secure / SameSite / `__Host-` cookie flags in production (§13)
8. Email verification on signup (§2)

Everything else is incremental hardening on top of those foundations.
