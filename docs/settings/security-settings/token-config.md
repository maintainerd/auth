# Token Configuration

> **Scope**: User Pool · **API Prefix**: `/security-settings/token` · **Storage**: `security_settings.token_config` (JSONB)

## Overview

Token configuration controls how JWT tokens are structured and validated — specifically the clock skew tolerance and which additional claims are included in access tokens and ID tokens. While [session configuration](session-config.md) controls token *lifetimes*, token configuration controls token *content* and *validation behavior*.

Tokens are the currency of authentication. Every API request carries an access token, and every SSO/OIDC flow produces an ID token. The claims in these tokens determine what the consuming application knows about the user and what decisions it can make without calling back to the auth server.

---

## Industry Standards & Background

### JWT Architecture

JSON Web Tokens (RFC 7519) are the standard token format for modern authentication:

```
┌──────────────────────────────────────────────────────────────┐
│  HEADER  .  PAYLOAD  .  SIGNATURE                            │
├──────────┬───────────┬───────────────────────────────────────┤
│ {"alg":  │ {"sub":   │ HMAC-SHA256(                        │
│  "RS256",│  "user1", │   base64url(header) + "." +          │
│  "typ":  │  "iss":   │   base64url(payload),                │
│  "JWT"}  │  "auth..", │   secret                            │
│          │  "exp":   │ )                                     │
│          │  1234..}  │                                       │
└──────────┴───────────┴───────────────────────────────────────┘
   Base64url   Base64url   Base64url
```

### Standard JWT Claims (RFC 7519)

| Claim | Name | Description | Required |
|-------|------|-------------|:--------:|
| `iss` | Issuer | Who created the token (e.g., `https://auth.example.com`) | ✅ |
| `sub` | Subject | Who the token is about (user ID) | ✅ |
| `aud` | Audience | Who the token is for (client ID or API) | ✅ |
| `exp` | Expiration | Unix timestamp when the token expires | ✅ |
| `iat` | Issued At | Unix timestamp when the token was created | ✅ |
| `nbf` | Not Before | Unix timestamp before which the token is invalid | Optional |
| `jti` | JWT ID | Unique identifier for the token (for revocation/replay prevention) | Recommended |

### OIDC Standard Claims (OpenID Connect Core 1.0)

ID tokens (used in OIDC flows) carry identity information:

| Claim | Description | Scope Required |
|-------|-------------|:--------------:|
| `name` | Full name | `profile` |
| `given_name` | First name | `profile` |
| `family_name` | Last name | `profile` |
| `email` | Email address | `email` |
| `email_verified` | Whether email is verified | `email` |
| `phone_number` | Phone number (E.164) | `phone` |
| `phone_number_verified` | Whether phone is verified | `phone` |
| `picture` | Profile picture URL | `profile` |
| `locale` | User's locale (e.g., `en-US`) | `profile` |
| `zoneinfo` | User's timezone (e.g., `America/New_York`) | `profile` |
| `updated_at` | Last profile update timestamp | `profile` |
| `address` | User's address (JSON object) | `address` |

### Relevant Standards

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **RFC 7519** | JSON Web Token (JWT) | Token format, standard claims, processing rules |
| **RFC 7515** | JSON Web Signature (JWS) | How tokens are signed (RS256, ES256, PS256) |
| **RFC 7516** | JSON Web Encryption (JWE) | How tokens can be encrypted (for sensitive claims) |
| **RFC 7517** | JSON Web Key (JWK) | Public key format for token verification (JWKS endpoint) |
| **RFC 7518** | JSON Web Algorithms (JWA) | Approved algorithms for signing and encryption |
| **RFC 9068** | JWT Profile for OAuth 2.0 Access Tokens | Standardized access token format |
| **OpenID Connect Core 1.0** | OIDC Spec | ID token requirements, standard claims, scope-to-claims mapping |
| **OWASP JWT Cheat Sheet** | OWASP Foundation | Security best practices for JWT handling |
| **OWASP ASVS v4** | V3.5 — Token-Based Session Management | Token entropy, validation, and handling |

### Signing Algorithms

| Algorithm | Type | Key Size | Performance | Recommendation |
|-----------|------|----------|-------------|----------------|
| **RS256** | RSA + SHA-256 | 2048+ bits | Slower signing, fast verification | ✅ Most common, widely supported |
| **RS384** | RSA + SHA-384 | 2048+ bits | Slower | Rarely needed |
| **RS512** | RSA + SHA-512 | 2048+ bits | Slowest RSA | Rarely needed |
| **ES256** | ECDSA + SHA-256 | P-256 curve | Fast signing, fast verification | ✅ Recommended for new deployments |
| **ES384** | ECDSA + SHA-384 | P-384 curve | Fast | Good alternative |
| **PS256** | RSA-PSS + SHA-256 | 2048+ bits | Similar to RS256 | More secure than RS256 (probabilistic) |
| **HS256** | HMAC + SHA-256 | 256+ bits | Fastest | ❌ Symmetric — DO NOT use for distributed verification |
| **none** | No signature | N/A | N/A | ❌ NEVER — `alg: none` attacks are common |

**OWASP recommendation:** Use asymmetric algorithms (RS256 or ES256). Never accept `alg: none`. Validate algorithm in token matches expected algorithm (prevent algorithm confusion attacks).

### Clock Skew — Why It Exists

In distributed systems, clocks are never perfectly synchronized:

```
Auth Server Time:  14:00:00.000 UTC ← issues token with exp = 14:15:00
API Server Time:   14:00:02.500 UTC ← 2.5 seconds ahead

Token arrives at API Server at 14:15:01.000 (auth time)
API Server thinks it's:        14:15:03.500
Token exp says:                14:15:00.000

Without skew tolerance → Token rejected (expired 3.5 seconds ago)
With 5-second skew    → Token accepted (within tolerance)
```

**NTP (Network Time Protocol)** typically keeps servers within 1-10 milliseconds, but:
- Cloud VMs can have higher clock drift (especially after VM migration)
- Containerized environments may inherit host clock drift
- Clock steps (NTP corrections) can cause sudden jumps
- Geographic distribution increases variance

**OWASP guidance:** Allow a small clock skew (30 seconds to 5 minutes). The default of 30 seconds is appropriate for most deployments. Larger values (5 minutes) are common in on-premises deployments with less reliable NTP.

### Access Token vs. ID Token — Claims Strategy

| Aspect | Access Token | ID Token |
|--------|:------------:|:--------:|
| **Purpose** | Authorize API requests | Prove user identity to the client |
| **Audience** | API/Resource Server | Client Application |
| **Contains** | Permissions, scopes, roles | User profile information |
| **Validated by** | API server on every request | Client after authentication |
| **Standard** | RFC 9068 (JWT Profile) | OIDC Core 1.0 |
| **Sensitive data** | Minimal — only authorization info | Can contain PII (name, email) |
| **Lifetime** | Short (5-60 minutes) | Short (matches access token or session) |

**Claim inclusion strategy:**
- **Minimize access token claims**: Only what the API needs for authorization (roles, permissions, tenant). Every byte increases request size.
- **ID token claims are scope-dependent**: Only include claims the client's scopes authorize.
- **Custom claims use namespaced keys**: `https://myapp.com/roles` to avoid collision with standard claims.
- **Sensitive data in claims**: Be cautious — JWTs are signed but not encrypted by default. Anyone with the token can read the claims.

### Token Size Considerations

JWTs are sent with every request (in `Authorization` header). Size matters:

| Component | Typical Size |
|-----------|-------------|
| Header | 36 bytes (base64) |
| Standard claims (6 claims) | ~150 bytes (base64) |
| Each additional claim | 20-100 bytes |
| RS256 signature | 342 bytes (base64) |
| **Minimal token** | **~530 bytes** |
| Token with 10 custom claims | ~1 KB |
| Token with roles + permissions | 1-3 KB |

**HTTP header size limits:**
- Most web servers: 8 KB default header size limit
- AWS ALB: 16 KB total headers
- Cloudflare: 16 KB total headers
- Some embedded systems: 4 KB

**Rule of thumb:** Keep tokens under 4 KB. If you need more claims, use a userinfo endpoint instead.

---

## Our Implementation

### Data Model

Stored in `security_settings.token_config` (JSONB column). The config is schema-free (`map[string]any`).

**Expected JSONB fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `clock_skew_leeway_seconds` | int | 30 | Allowed clock difference for token validation (exp, nbf, iat) |
| `additional_id_token_claims` | []string | [] | Extra claims to include in ID tokens (beyond OIDC standard) |
| `additional_access_token_claims` | []string | [] | Extra claims to include in access tokens (beyond sub, iss, aud, exp) |

### API Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/security-settings/token` | `GetTokenConfig` | Returns the current token configuration |
| `PUT` | `/security-settings/token` | `UpdateTokenConfig` | Replaces the token configuration |

**Example request body (PUT):**
```json
{
  "clock_skew_leeway_seconds": 30,
  "additional_id_token_claims": ["roles", "permissions", "tenant_id"],
  "additional_access_token_claims": ["roles", "tenant_id"]
}
```

### Service Layer

- **`GetTokenConfig(ctx, userPoolID)`** — Lazy-creates the security setting row, then returns the `token_config` JSONB.
- **`UpdateTokenConfig(ctx, userPoolID, config, updatedBy, ipAddress, userAgent)`** — Calls `updateConfig` with `"token"` config type.

The update runs in a transaction:
1. Find or create `security_settings` row for the user pool
2. Replace the `token_config` JSONB column
3. Increment `version`
4. Create a `security_settings_audit` row with `change_type: "update_token_config"`

### Audit Trail

Every update creates a `security_settings_audit` row:
- `change_type`: `"update_token_config"`
- `old_config`: Previous token JSONB
- `new_config`: New token JSONB
- `ip_address`, `user_agent`, `created_by`: Admin context

### Validation

Currently minimal — only validates the config map is non-empty. No schema validation.

**Source files:**
- Model: `internal/model/security_setting.go`
- Service: `internal/service/security_setting.go`
- Handler: `internal/rest/security_setting_handler.go`
- Routes: `internal/rest/security_setting_routes.go`
- DTO: `internal/dto/security_setting.go`

---

## Requirements Checklist

### Configuration Management (Admin API)
- [x] Get token config via `GET /security-settings/token`
- [x] Update token config via `PUT /security-settings/token`
- [x] Stored as JSONB for flexible schema evolution
- [x] Version tracking (auto-incremented on each update)
- [x] Audit trail (old/new config, who changed, IP, user agent)
- [x] Lazy-create on first access
- [x] OpenTelemetry span tracing
- [x] Unit tests for service layer
- [x] Unit tests for handler layer
- [ ] Validation: `clock_skew_leeway_seconds` must be 0–300 (5 minutes max)
- [ ] Validation: `additional_*_claims` must contain only known claim names
- [ ] Sane defaults on creation

### Token Generation
- [ ] Access token includes standard JWT claims (`iss`, `sub`, `aud`, `exp`, `iat`, `jti`)
- [ ] Access token includes `additional_access_token_claims` from config
- [ ] ID token includes required OIDC claims (`iss`, `sub`, `aud`, `exp`, `iat`, `auth_time`, `nonce`)
- [ ] ID token includes scope-dependent OIDC claims
- [ ] ID token includes `additional_id_token_claims` from config
- [ ] Custom claims use namespaced keys to avoid collision
- [ ] Token size stays under 4 KB with all configured claims

### Token Signing
- [ ] Use asymmetric algorithm (RS256 or ES256)
- [ ] Never accept `alg: none` during validation
- [ ] Validate `alg` header matches expected algorithm (prevent algorithm confusion)
- [ ] Key rotation via JWKS endpoint
- [ ] JWKS endpoint publicly accessible for verifiers
- [ ] Key ID (`kid`) included in token header
- [ ] Support for ES256 (recommended for new deployments)

### Token Validation
- [ ] Validate signature against known public keys (JWKS)
- [ ] Validate `exp` claim with `clock_skew_leeway_seconds` tolerance
- [ ] Validate `nbf` claim with `clock_skew_leeway_seconds` tolerance
- [ ] Validate `iss` claim matches expected issuer
- [ ] Validate `aud` claim contains expected audience
- [ ] Reject tokens with `alg: none`
- [ ] Reject tokens with unexpected algorithms

### Clock Skew Management
- [ ] Apply `clock_skew_leeway_seconds` to `exp` validation
- [ ] Apply `clock_skew_leeway_seconds` to `nbf` validation
- [ ] Apply `clock_skew_leeway_seconds` to `iat` validation (optional)
- [ ] Document operational guidance: ensure NTP is running on all servers
- [ ] Alert when clock drift exceeds skew tolerance

### Key Management
- [ ] RSA key pair generation (2048+ bits, 4096 recommended)
- [ ] EC key pair generation (P-256 for ES256)
- [ ] Key rotation without downtime (both old and new keys valid during transition)
- [ ] JWKS endpoint returns all active public keys
- [ ] Retired keys remain in JWKS for grace period (for in-flight tokens)
- [ ] Key storage: HSM or KMS for production (not filesystem)
- [ ] Key backup and recovery procedure

### OIDC Compliance
- [ ] ID tokens conform to OIDC Core 1.0 §2 (required claims)
- [ ] Claims controlled by scope (`openid`, `profile`, `email`, `phone`, `address`)
- [ ] `at_hash` claim in ID token when access token is issued alongside
- [ ] `c_hash` claim in ID token when authorization code is issued alongside
- [ ] `auth_time` claim when max_age was requested or `auth_time` is always included
- [ ] `nonce` claim echoed from authorization request

### Security Hardening
- [ ] Token encryption (JWE) for sensitive claims (optional)
- [ ] Token binding (DPoP — RFC 9449) to prevent token theft (future)
- [ ] Proof-of-possession tokens (mTLS-bound — RFC 8705) (future)
- [ ] Short-lived access tokens (enforced via session config TTL)
- [ ] No sensitive PII in access tokens (use ID token or userinfo endpoint)
- [ ] Token introspection endpoint (RFC 7662) for opaque token validation

### Integration & Testing
- [ ] Unit tests for token config service methods
- [ ] Integration test: clock skew tolerance in token validation
- [ ] Integration test: additional claims appear in generated tokens
- [ ] Integration test: JWKS endpoint returns correct keys
- [ ] Integration test: key rotation with in-flight token validation
- [ ] Integration test: audit trail with token config changes
- [ ] Performance test: token generation and validation latency

---

## References

- [RFC 7519 — JSON Web Token (JWT)](https://datatracker.ietf.org/doc/html/rfc7519)
- [RFC 7515 — JSON Web Signature (JWS)](https://datatracker.ietf.org/doc/html/rfc7515)
- [RFC 7516 — JSON Web Encryption (JWE)](https://datatracker.ietf.org/doc/html/rfc7516)
- [RFC 7517 — JSON Web Key (JWK)](https://datatracker.ietf.org/doc/html/rfc7517)
- [RFC 7518 — JSON Web Algorithms (JWA)](https://datatracker.ietf.org/doc/html/rfc7518)
- [RFC 9068 — JWT Profile for OAuth 2.0 Access Tokens](https://datatracker.ietf.org/doc/html/rfc9068)
- [OpenID Connect Core 1.0 — ID Token](https://openid.net/specs/openid-connect-core-1_0.html#IDToken)
- [OWASP JWT Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html)
- [OWASP ASVS v4.0 — V3.5 Token-Based Session Management](https://owasp.org/www-project-application-security-verification-standard/)
- [RFC 9449 — DPoP: OAuth 2.0 Demonstrating Proof of Possession](https://datatracker.ietf.org/doc/html/rfc9449)
