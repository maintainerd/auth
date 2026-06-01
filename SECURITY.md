# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | ✅ Active support   |
| < 1.0    | ❌ Development only |

## Reporting a Vulnerability

**Do not open a public issue.** Report security vulnerabilities privately to:

**Email:** security@maintainerd.dev

Include:
- A detailed description of the vulnerability
- Steps to reproduce
- Affected versions
- Any potential mitigations you've identified

We aim to acknowledge reports within 48 hours and provide a fix timeline within 5 business days. Critical vulnerabilities are typically patched within 72 hours.

## Threat Model

### Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Public internet → Public port (8081) | Untrusted — all inputs validated |
| Internal network → Management port (8080) | Semi-trusted — requires authentication |
| Application → Database | Encrypted via TLS in production |
| Application → Redis | Encrypted via TLS when REDIS_TLS is enabled |
| Application → External IdP | OIDC/OAuth2 with token validation |
| Webhook (outbound) → External services | HMAC-SHA256 signed, HTTPS only |

### Assets

| Asset | Sensitivity | Protection |
|-------|-------------|------------|
| JWT signing keys | Critical | Secret-manager-backed, never in plaintext env |
| User passwords | High | Bcrypt with cost ≥ 12, constant-time comparison |
| Client secrets | High | Bcrypt-hashed at rest, shown once at creation |
| Refresh tokens | High | SHA-256 hashed at rest, rotated on use |
| Access tokens | Medium | Short-lived (15 min), denylist via Redis |
| Encryption key (AES-256) | Critical | Secret-manager-backed, exactly 32 bytes |
| Audit logs | High | Append-only, immutable, PII-redacted |

### Attack Surface

| Vector | Mitigation |
|--------|------------|
| Brute-force login | Rate limiting (Redis-backed), account lockout, dummy bcrypt |
| Token theft (bearer) | DPoP binding, short-lived access tokens, refresh token rotation |
| Algorithm confusion (JWT) | RS256 enforced, `kid` header validated |
| CSRF | Double-submit cookie, `__Host-` prefix, SameSite=Strict |
| SSRF (webhooks) | HTTPS-only, private/loopback IP block, DNS resolution check |
| SQL injection | GORM parameterized queries, statement timeout |
| Secret leak | Gitleaks pre-commit + CI, secret-manager integration |
| Directory traversal | Path validation on file-based secret provider |
| Redirect injection | Exact-match redirect URI validation, `javascript:`/`data:` blocked |

## Security Features

| Feature | Implementation |
|---------|---------------|
| Password hashing | Bcrypt cost 12+, constant-time `crypto/subtle` comparison |
| Client secret hashing | Bcrypt cost 12+, rotation with grace period |
| Token hashing | SHA-256 for refresh tokens and authorization codes |
| Encryption at rest | AES-256 via `crypto.EncryptAtRest` for sensitive fields |
| MFA | TOTP (RFC 6238), WebAuthn/FIDO2, backup codes |
| OAuth 2.0 hardening | PKCE S256 required, redirect URI exact match, state/nonce enforcement |
| DPoP | RFC 9449 — access tokens bound to client key pair |
| JWT | RS256, key rotation, `kid` header, algorithm enforcement |
| Audit logging | Append-only, PII-redacted, per-tenant isolation |
| Secret management | Pluggable (env, file, AWS, GCP, Azure, Vault) |
| TLS | Enforced in production (DB SSLMode, Redis TLS, HTTPS redirect) |

## Dependencies

We use Dependabot to monitor dependency updates. CI runs weekly vulnerability scans on go.sum. Direct dependencies are pinned; indirect dependencies are updated via `go mod tidy`.

## Acknowledgments

We appreciate the security research community. Hall of Fame contributors will be listed here with permission.
