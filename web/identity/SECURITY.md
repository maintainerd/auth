# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | ✅ Active support   |
| < 0.1   | ❌ Development only |

## Reporting a Vulnerability

**Do not open a public issue.** Report security vulnerabilities privately to:

**Email:** security@maintainerd.dev

Include:
- A detailed description of the vulnerability
- Steps to reproduce
- Affected versions
- Any potential mitigations you've identified

We aim to acknowledge reports within 48 hours and provide a fix timeline within
5 business days. Critical vulnerabilities are typically patched within 72 hours.

## Scope

This repository is the **public, browser-delivered** hosted identity UI. It is a
static SPA served by nginx and talks only to the Maintainerd Auth public API
(port 8081) over HTTPS. It holds no server-side secrets; authentication state is
delivered as httpOnly cookies by the backend.

## Threat Model

| Boundary | Description |
|----------|-------------|
| Browser → SPA (static assets) | Served over HTTPS with a strict Content-Security-Policy |
| SPA → Public auth API (8081) | Untrusted client; all inputs validated server-side |
| Session tokens | httpOnly cookies issued by the backend; not readable by JS |

### Attack surface & mitigations

| Vector | Mitigation |
|--------|------------|
| XSS | React escaping, strict CSP (`default-src 'self'`, `frame-ancestors 'none'`) |
| Clickjacking | `X-Frame-Options: DENY` + CSP `frame-ancestors 'none'` |
| MIME sniffing | `X-Content-Type-Options: nosniff` |
| Referrer leakage | `Referrer-Policy: strict-origin-when-cross-origin` |
| CSRF | Double-submit `__Host-csrf` cookie echoed in `X-CSRF-Token` on unsafe requests |
| Token exposure | Session tokens delivered as httpOnly cookies; no source maps shipped |
| Open redirect | Server-validated OAuth `redirect_uri`; user-controlled targets validated client-side |
| Sensitive data at rest | `register_email` kept in `sessionStorage` and cleared on completion |

## Dependencies

Dependabot monitors npm dependency updates. CI runs `npm audit` and additional
security scans (Snyk, Semgrep, Scorecard) on every push and pull request.

## Acknowledgments

We appreciate the security research community. Hall of Fame contributors will be
listed here with permission.
