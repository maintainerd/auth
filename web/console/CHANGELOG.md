# Changelog

All notable changes to maintainerd-auth-console will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-07-03

### Added
- Initial release of the Maintainerd Auth admin console
- Tenant, user, role, permission, policy, service, API, client, and API-key management
- Registration-flow, invitation, identity-provider, and webhook/event management
- Security configuration surfaces: MFA, password policies, sessions, tokens, account lockout, registration, threat detection, IP restrictions, and rate limits
- Branding, email/SMS template, and messaging (email/SMS delivery) management
- Self-service account MFA enrollment (TOTP, passkeys, SMS, email OTP) and profile/settings
- Log monitoring with auth-event detail views
- Top-level error boundary with reload fallback and a not-found catch-all route
- Route-level code splitting via `React.lazy` + `Suspense`
- Runtime config injection (`window.__ENV__`) so one image targets multiple API origins
- Production nginx config with SPA fallback, gzip, immutable asset caching, and security headers

### Security
- Content-Security-Policy (`default-src 'self'`, `frame-ancestors 'none'`), `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`
- Production source maps disabled
- Session tokens delivered as httpOnly cookies (`X-Token-Delivery: cookie`)
- Distinct, non-leaking user-facing messages for API error statuses (never raw `HTTP <status>`)
- Config-edit screens block on load failure instead of rendering default-valued forms

### Changed
- CI and security workflows run on Node 22
