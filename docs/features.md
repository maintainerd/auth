# maintainerd-auth - Feature Audit

Last audited: 2026-06-04, after the v1.0.0 service-to-service authorization pass.

Legend:

- [x] Implemented in the current codebase and exposed or wired where applicable.
- [ ] Not implemented, not currently wired, or only present as a partial/internal helper.

This file is intentionally an implementation-facing checklist. Marketing copy and roadmap claims should live elsewhere unless there is source code, routing, migrations, and/or tests behind them.

---

## Core Authentication

- [x] Email and password login with bcrypt hashing
- [x] Brute-force protection, rate limiting, and account lockout helpers
- [x] Magic link login with signed, single-use, time-limited links
- [ ] Passwordless email OTP login as a standalone sign-in flow
- [x] Email verification OTP for signup and email verification
- [x] Forgot password and reset password token flow
- [x] User registration with registration-flow support and role assignment
- [x] Invite-based registration with pre-assigned roles
- [x] Internal login on the management surface without mandatory `client_id`
- [x] Public login on the identity surface with `client_id` and `provider_id`
- [ ] Logout that revokes the active refresh token. Current logout clears cookies only; OAuth revocation is available separately.

---

## OAuth 2.0 and OpenID Connect

### Endpoints

- [x] `POST /oauth/token`
- [x] `GET /oauth/authorize`
- [x] `POST /oauth/revoke`
- [x] `POST /oauth/introspect` on the management surface
- [x] `GET /oauth/userinfo`
- [x] `GET /.well-known/openid-configuration`
- [x] `GET /.well-known/oauth-authorization-server`
- [x] `GET /.well-known/jwks.json`
- [x] `POST /oauth/par`
- [x] `POST /oauth/device_authorization`
- [x] Device user approval and denial endpoints
- [x] `POST /oauth/register`
- [x] `GET/POST /oauth/end_session`
- [x] `POST /oauth/logout/backchannel`
- [x] `POST /oauth/ciba`
- [x] CIBA approve and deny endpoints
- [x] Consent challenge retrieval and consent decision submission
- [x] Consent grant listing and per-grant revocation

### Grant Types

- [x] `authorization_code` with PKCE S256
- [x] `refresh_token` with rotation, token-family tracking, and reuse detection
- [x] `client_credentials`
- [x] `device_code`
- [x] `urn:ietf:params:oauth:grant-type:token-exchange`
- [x] `urn:openid:params:grant-type:ciba`

### Token Security

- [x] RS256 JWT access tokens with 15-minute TTL
- [x] Refresh tokens stored hashed with SHA-256
- [x] Authorization codes stored hashed and consumed once
- [x] Refresh token family reuse detection with family-wide revocation
- [x] `sub`, `aud`, `iss`, `iat`, `exp`, `jti`, and `nbf` token claims
- [x] `kid` JWT header and multi-key JWKS support
- [x] RS256-only validation for normal JWTs
- [x] Crypto-secure JTI/token/random generation
- [ ] Fully wired DPoP-bound OAuth token flow. DPoP primitives exist, but middleware/token request integration is not broadly mounted.
- [ ] mTLS client authentication

### Client Model

- [x] Confidential and public clients
- [x] Client types such as traditional, SPA, mobile, and M2M
- [x] `token_endpoint_auth_method`: `client_secret_basic`, `client_secret_post`, and `none`
- [x] Per-client `grant_types` and `response_types`
- [x] Per-client access token and refresh token TTL fields
- [x] `require_consent` flag
- [x] Client URIs including redirect and logout URIs
- [x] Client logo, policy, and terms fields

---

## Multi-Factor Authentication

- [ ] Email OTP as an MFA second factor
- [x] TOTP enrollment, verification, disable, and backup-code generation
- [x] WebAuthn/passkey registration and authentication ceremonies
- [x] SMS OTP login support
- [x] Step-up challenge and verification endpoints
- [x] Per-tenant MFA policy storage through security settings
- [x] MFA reset flow through an admin reset endpoint
- [ ] Complete token `acr_values` and `amr` claim enforcement across auth flows

---

## Federation and External Identity Providers

- [x] OIDC upstream federation/token exchange
- [x] Generic identity provider configuration CRUD
- [x] Per-tenant provider configuration
- [x] Identity linking and unlinking
- [x] Just-in-time user provisioning for federated login
- [x] Attribute/metadata extraction from upstream claims
- [x] Home-realm discovery by email domain
- [ ] SAML 2.0 service provider
- [ ] SAML 2.0 IdP-initiated SSO

---

## Provisioning - SCIM 2.0

- [ ] SCIM 2.0 `/Users` resource
- [ ] SCIM 2.0 `/Groups` resource
- [ ] SCIM service metadata endpoints
- [ ] SCIM PATCH operations
- [ ] SCIM bulk endpoint
- [ ] SCIM bearer-token client authentication

---

## Multi-Tenancy, Organizations, and RBAC

- [x] Tenant lifecycle management with status, public flag, and settings
- [x] Tenant members with role assignment
- [x] Roles
- [x] Permissions
- [x] Policies
- [x] Services and APIs as IAM resources
- [x] Service principals linked to OAuth `client_credentials` clients
- [x] Service identity claims in access tokens (`sub_type=service`, `svc`)
- [x] IAM policy evaluator with default deny, explicit-deny-wins, and wildcard matching
- [x] Service policy bundle endpoint with `ETag` and `304 Not Modified`
- [x] Service-to-service authorization endpoint for external callers
- [x] Backend service-to-service APIs remain in this repo; external SDKs are separate consumers of these APIs
- [x] IAM policy update and service-policy assignment/removal webhook invalidation events
- [x] API keys scoped to APIs and permissions
- [x] Registration flows with automatic role assignment
- [x] Invite system with pre-assigned roles
- [x] Permission middleware on management routes
- [x] Per-tenant settings for rate limits, audit config, and maintenance mode
- [x] Per-tenant branding
- [x] Per-tenant security settings for MFA, password, session, lockout, threat, and token config

---

## Session Management

- [x] Cookie-based token delivery for browser clients
- [x] Refresh token family tracking
- [x] Refresh token reuse detection and family-wide revocation
- [x] Active session listing per user
- [x] Single session revocation by ID
- [x] Revoke-all-sessions endpoint
- [x] Session revocation on password reset/change paths
- [x] Session revocation on permission and role changes
- [x] Concurrent session limit enforcement by evicting the oldest session
- [x] Idle session timeout with sliding `last_used_at`
- [x] Absolute session lifetime cap

---

## Secret Management

- [x] Pluggable secret manager abstraction
- [x] Environment variable provider
- [x] Local file/Docker secrets provider
- [x] AWS SSM Parameter Store provider
- [x] AWS Secrets Manager provider
- [x] HashiCorp Vault KV provider
- [x] Azure Key Vault provider
- [x] GCP Secret Manager provider
- [x] Provider selection through `SECRET_PROVIDER`
- [x] Secret prefixing through `SECRET_PREFIX` where supported

---

## Password and Credential Policy

- [x] Minimum length default of 8 characters
- [x] Maximum length default of 128 characters
- [x] Uppercase, lowercase, digit, and special-character requirements by default
- [x] Common weak-password pattern blocklist
- [x] Configurable per-tenant password policy
- [x] Password history storage and reuse prevention
- [x] Password expiration policy fields and login-time expiry check
- [ ] HIBP k-anonymity breach check
- [ ] Disposable email blocklist on signup
- [ ] Argon2id password hashing support

---

## Security Hardening

- [x] Brute-force protection on login
- [x] Account lockout after repeated failures
- [x] Timing-safe login failure path with dummy bcrypt comparison
- [x] Timing-safe comparisons for secrets/tokens where implemented
- [x] CSRF protection on cookie-authenticated public state-changing routes
- [x] IP-based rate limiting on public and credential endpoints
- [x] Redis-backed global public rate limiting
- [x] CORS allow-list middleware that rejects unsafe wildcard credential use
- [x] HSTS header support
- [x] Secure, HTTP-only auth cookie helpers
- [x] Security headers including content type, frame, referrer, CSP, and permissions policy headers
- [x] DTO validation at HTTP boundaries
- [x] Redirect URI dangerous-scheme rejection
- [x] Request size limits on all routers and stricter auth endpoint limits
- [ ] DPoP enforcement on protected resource routes
- [ ] mTLS client authentication

---

## Cryptography and Key Management

- [x] RSA-2048 minimum key validation at startup
- [x] RS256 signing and validation
- [x] `kid` in JWTs
- [x] Multi-key JWKS with active and retiring keys
- [x] Automatic key rotation runner
- [x] PKCE S256 validation
- [x] SHA-256 hashing for refresh tokens and authorization codes
- [x] Crypto-secure random generation for tokens, OTPs, JTIs, and IDs
- [ ] KMS-backed signing
- [ ] ECDSA/ES256 signing support
- [ ] EdDSA signing support
- [ ] Argon2id KDF support

---

## Audit Logging

- [x] Structured auth event model and REST API
- [x] Login success and login failure events
- [x] Token issuance, revocation, and introspection events
- [x] Consent grant and revocation events
- [x] Selected privileged admin actions emit auth events
- [x] Retention runner for old auth events
- [x] Append-only auth event migration
- [x] Trace ID captured on auth events when present
- [ ] Comprehensive privileged admin action coverage for every CRUD/settings endpoint
- [ ] Configurable PII redaction policy for audit-event storage
- [ ] SIEM export or forwarding integration

---

## Webhooks

- [x] Configurable webhook endpoints per tenant
- [x] Event-type subscription model on webhook endpoints
- [x] HMAC-SHA256 signature helper for delivery payloads
- [ ] Webhook dispatcher wired to auth-event creation
- [ ] Replay protection verifier for inbound consumers
- [ ] Automatic retries with exponential backoff
- [ ] Dead-letter queue
- [ ] Webhook delivery history and status tracking

---

## Email and SMS

- [x] Email provider abstraction
- [x] SMTP provider
- [x] AWS SES provider
- [x] SendGrid provider
- [x] Postmark provider
- [x] Mailgun provider
- [x] Resend provider
- [x] SMS provider abstraction
- [x] Twilio provider
- [x] AWS SNS provider
- [x] Vonage provider
- [x] Per-tenant email and SMS provider configuration
- [x] Customizable email, SMS, and login templates per tenant
- [ ] Async delivery queue
- [ ] Retry with backoff on delivery failure
- [ ] Localized/i18n transactional templates
- [ ] Email sandbox mode

---

## REST API

- [x] Management REST surface on port 8080
- [x] Public identity REST surface on port 8081
- [x] Versioned `/api/v1` REST routes
- [x] Consistent JSON success envelope
- [x] RFC-style OAuth error responses
- [x] OpenAPI spec served at `/openapi.json` on the management surface
- [x] Pagination helpers and pagination on list endpoints
- [ ] Swagger UI or Redoc served by the app
- [ ] Problem Details (RFC 7807) for non-OAuth errors

---

## gRPC

- [x] gRPC server on `:50051`
- [x] OpenTelemetry gRPC stats handler
- [x] Setup and tenant management gRPC services
- [ ] Proto definitions for all core identity services
- [ ] Dedicated management-only gRPC port separate from the basic server bind
- [x] gRPC reflection
- [x] Auth, logging, and recovery interceptors
- [x] `grpc.health.v1` health check service
- [ ] gRPC-Gateway transcoding

---

## Architecture

- [x] `cmd/server` thin executable entrypoint
- [x] `internal/app` composition root
- [x] Domain-grouped packages under `internal/`
- [x] Cross-cutting infrastructure under `internal/platform/`
- [x] Domain-owned models and repositories
- [x] Dual-port REST server
- [x] PostgreSQL with GORM and OTEL instrumentation
- [x] Redis-backed cache/rate-limit/JTI primitives
- [x] OpenTelemetry tracing and metrics initialization
- [x] Background runners for migrations, seeding, secret refresh, and key rotation
- [ ] The old layered layout described in earlier docs: `internal/rest`, `internal/service`, `internal/repository`, `internal/model`, `internal/dto`, `internal/middleware`, `internal/config`, `internal/cache`, `internal/crypto`, `internal/security`, `internal/apperror`, `internal/jwt`
- [ ] Dual frontend repos in this backend repository

---

## Deployment

- [x] Multi-stage Dockerfile
- [x] Non-root runtime user
- [x] Container health check
- [x] `/health` liveness endpoint
- [x] `/ready` readiness endpoint checking DB and Redis
- [x] Graceful shutdown on `SIGTERM`
- [x] Prometheus `/metrics` endpoint on the management surface
- [ ] `/healthz`, `/readyz`, and `/livez` aliases
- [ ] Kubernetes Helm chart
- [ ] HPA, PodDisruptionBudget, or NetworkPolicy manifests
- [ ] Build-info Prometheus gauge
- [ ] Image signing with cosign/sigstore
- [ ] Vulnerability scanning CI config

---

## Release Roadmap Claims

- [x] v1.0.0 release documentation is aligned with implemented OAuth/OIDC, MFA, federation, IAM, S2S authorization, CI, and test coverage status.
- [ ] v2.0.0 enterprise roadmap items such as LDAP/AD, ABAC, adaptive MFA, FAPI 2.0, SDKs, multi-region HA, and compliance are not implemented in this repository.
