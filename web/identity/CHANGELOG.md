# Changelog

All notable changes to maintainerd-auth-identity will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-07-03

### Added
- Initial public release of the Maintainerd Auth hosted identity UI
- Password login, registration, invite-based registration, and profile completion
- Email verification with resend (success/failure surfaced to the user)
- Magic-link (passwordless) and SMS one-time-code login
- Login MFA step: TOTP, SMS, email OTP, WebAuthn/passkeys, and backup codes
- Self-service MFA enrollment, linked identities, and backup-code recovery
- OAuth2/OIDC flows: authorize, consent, device, CIBA, grants, and end-session
- Account lockout and rate-limit screens driven by backend 423/429 responses
- Top-level error boundary with reload fallback and a not-found catch-all route
- Route-level code splitting via `React.lazy` + `Suspense`
- Runtime config injection (`window.__ENV__`) so one image targets multiple API origins
- Production nginx config with SPA fallback, gzip, immutable asset caching, and security headers

### Security
- Content-Security-Policy (`default-src 'self'`, `frame-ancestors 'none'`), `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`
- Production source maps disabled
- `register_email` moved from `localStorage` to `sessionStorage` and cleared on completion
- Distinct, non-leaking user-facing messages for API error statuses (never raw `HTTP <status>`)
- CSRF double-submit cookie wired for cookie-authenticated requests
- Development-only debug utilities excluded from the production bundle

### Changed
- Invite registration now enforces the shared tenant password policy
- CI and security workflows run on Node 22
