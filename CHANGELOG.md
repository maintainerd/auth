# Changelog

All notable changes to maintainerd-auth will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- DPoP-bound access/refresh tokens (RFC 9449)
- private_key_jwt + client_secret_jwt client auth (RFC 7523)
- Generic OAuth2 upstream connector for federation
- Recovery middleware with structured logging + stack traces
- Canonical Config struct with GetConfig() accessor
- model.Base embeddable struct for soft-delete + audit timestamps
- Redis TLS support (REDIS_TLS env or rediss:// scheme)
- SMS OTP rate limiting (per-phone + brute-force protection)
- AMR/ACR claims in ID tokens
- Build-info gauge with commit SHA + build date
- Consent revoke audit event
- PII redaction for Description + ErrorReason in audit payloads
- `/healthz`, `/readyz`, `/livez` endpoints with dependency status
- `.env.example` with all documented environment variables
- Gitleaks pre-commit hook
- CI: gosec security audit step
- GitLab provider constant

### Fixed
- Logout now revokes server-side sessions (was cookie-only)
- Per-client access token TTL now reflected in JWT exp claim
- Grant type enforcement for authorization_code and refresh_token in token exchange
- `offline_access` scope now required for refresh token issuance
- ID tokens now include Name, FirstName, LastName, Picture from user profile
- Scope-to-claim mapping + custom claim mappers now wired
- CSP: removed `'unsafe-inline'` from style-src
- HMAC_SECRET_KEY validated at boot (fail-fast)

### Changed
- Dockerfile: distroless static base image (UID 65532 as m9d user)
- README: all references updated from maintainerd/auth to maintainerd/maintainerd-auth
- CI test timeout increased to 600s for race detector

## [1.0.0] — 2026-06

Initial production release.

### Core Authentication
- Username/email + password login (internal + public)
- User registration with configurable signup flows
- Forgot/reset password with email delivery
- Email verification (6-digit OTP, 1h TTL)
- Magic link / passwordless login (15min, single-use)
- Invite flow with role pre-assignment
- Bootstrap/setup flow
- Force password change, account deletion (GDPR), data export (GDPR)
- SMS one-time code login, account recovery via backup codes

### OAuth 2.0 / OIDC
- Full OAuth 2.0 endpoints: token, authorize, revoke, introspect, userinfo
- OIDC Discovery + RFC 8414 Authorization Server Metadata
- Grants: authorization_code (PKCE), refresh_token (rotation), client_credentials, device_code, token-exchange
- CIBA (poll mode), Pushed Authorization Requests (PAR), Dynamic Client Registration
- Back-Channel Logout, RP-Initiated Logout

### Multi-Factor Authentication
- TOTP (RFC 6238), WebAuthn/FIDO2 passkeys
- Backup codes, step-up authentication
- Per-tenant MFA policy, admin reset flow

### Federation
- OIDC upstream connectors (Google, Microsoft, Apple, GitHub, GitLab)
- JIT provisioning, attribute mapping, home-realm discovery
- Identity linking/unlinking

### Tenancy & RBAC
- Multi-tenant isolation, tenant members, services, settings
- Roles, permissions, policies with middleware enforcement
- API keys with scoping and invite system

### Security
- Bcrypt password hashing (cost ≥ 12), constant-time comparison
- Rate limiting on all auth endpoints, account lockout
- CORS, security headers, CSRF, HSTS, CSP
- Cookie security (Secure, SameSite, __Host- prefix)
- URI/redirect validation, body size limits, Content-Type enforcement
- Secret management: env, file, AWS, GCP, Azure, Vault

### Operations
- Structured JSON logging (slog) with PII redaction
- OpenTelemetry tracing + Prometheus metrics
- Audit logging (append-only, immutable, per-tenant)
- Pluggable email (SMTP, SES, SendGrid, Postmark, Mailgun, Resend)
- Pluggable SMS (Twilio, SNS, Vonage)
- Webhooks with HMAC-SHA256 + replay protection
- Graceful shutdown, health/readiness probes
