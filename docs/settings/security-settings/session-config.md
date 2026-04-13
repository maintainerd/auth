# Session Configuration

> **Scope**: User Pool · **API Prefix**: `/security-settings/session` · **Storage**: `security_settings.session_config` (JSONB)

## Overview

Session configuration controls the lifecycle of authenticated sessions — how long they last, how many can coexist, and how tokens are refreshed. Session management is one of the most critical security boundaries: a poorly configured session policy turns every authentication investment into a time-limited illusion.

A session begins when a user authenticates successfully and ends when the session is revoked, expires, or is replaced. Between those events, the session is represented by tokens (access token + refresh token) that the client presents with every request.

---

## Industry Standards & Background

### The Session Lifecycle

```
┌──────────────┐    ┌────────────────┐    ┌───────────────┐
│   Login       │───▶│  Session Active │───▶│  Session End   │
│  (AuthN)      │    │                │    │               │
└──────────────┘    │  Access Token   │    │  • Explicit    │
                     │  Refresh Token  │    │    logout      │
                     │  Idle Timer     │    │  • Idle        │
                     │  Absolute Timer │    │    timeout     │
                     └────────────────┘    │  • Absolute    │
                            ▲  │            │    timeout     │
                            │  ▼            │  • Revocation  │
                     ┌────────────────┐    │  • Max sessions│
                     │  Token Refresh  │    └───────────────┘
                     │  (silent)       │
                     └────────────────┘
```

### Types of Timeouts

| Timeout Type | What It Does | Why It Exists |
|-------------|-------------|---------------|
| **Access Token TTL** | Time before the short-lived access token expires | Limits the window of a stolen access token |
| **Refresh Token TTL** | Time before the refresh token expires (forces re-login) | Bounds the total session lifetime |
| **Idle Timeout** | Time of inactivity before session is invalidated | Protects when user walks away from an unlocked device |
| **Absolute Timeout** | Maximum session duration regardless of activity | Forces periodic reauthentication even for active users |

### Relevant Standards

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **OWASP Session Management Cheat Sheet** | OWASP Foundation | Comprehensive guidance on session lifecycle, token handling, timeouts |
| **NIST SP 800-63B §7** | Reauthentication | AAL1: 30-day max, 30-min idle. AAL2: 12-hour max, 30-min idle. AAL3: 12-hour max, 15-min idle |
| **RFC 6749** | OAuth 2.0 | Access token and refresh token model, token exchange |
| **RFC 6819** | OAuth 2.0 Threat Model | Session fixation, token theft, refresh token rotation |
| **RFC 7519** | JSON Web Tokens (JWT) | Token format using `exp`, `iat`, `nbf` claims |
| **PCI DSS v4.0 Req 8.2.8** | Session timeout | Idle session timeout of max 15 minutes for CDE access |
| **CIS Controls v8** | Control 3.3 | Configure Data Access Control Lists — includes session management |
| **OWASP ASVS v4** | V3 — Session Management | 30+ requirements for session security |
| **ISO 27001:2022** | A.8.5 | Secure session handling for authenticated users |

### NIST SP 800-63B Session Requirements

NIST provides specific timeouts per assurance level:

| Requirement | AAL1 | AAL2 | AAL3 |
|------------|------|------|------|
| **Maximum session duration** | 30 days | 12 hours | 12 hours |
| **Idle timeout** | 30 minutes | 30 minutes | 15 minutes |
| **Reauthentication trigger** | After max or idle | After max or idle | After max or idle |
| **Reauthentication method** | Any single factor | Both factors (or session binding) | Hardware-bound factor |

### OWASP ASVS v4 — Session Management (V3)

| ID | Requirement | Our Status |
|----|-------------|:----------:|
| **V3.1.1** | Verify the application never reveals session tokens in URL parameters | N/A (JWT in header) |
| **V3.2.1** | Verify new session tokens are generated on authentication | ✅ |
| **V3.2.2** | Verify session tokens possess at least 64 bits of entropy | ✅ (JWT) |
| **V3.3.1** | Verify logout invalidates the session | 🔲 |
| **V3.3.2** | Verify idle timeout invalidates the session | 🔲 |
| **V3.3.3** | Verify absolute timeout invalidates the session | 🔲 |
| **V3.3.4** | Verify admin can terminate any active session | 🔲 |
| **V3.4.1** | Verify cookie-based session tokens have Secure flag | ✅ (cookie pkg) |
| **V3.4.2** | Verify cookie-based session tokens have HttpOnly flag | ✅ (cookie pkg) |
| **V3.4.3** | Verify cookie-based session tokens use SameSite | ✅ (cookie pkg) |
| **V3.5.1** | Verify the application allows users to revoke OAuth tokens | 🔲 |
| **V3.5.2** | Verify the application uses short-lived access tokens | Configurable |
| **V3.7.1** | Verify the application limits active sessions per user | Configurable |

### Access Token vs. Refresh Token Model

```
Short-lived Access Token                Long-lived Refresh Token
┌─────────────────────┐                ┌─────────────────────────┐
│ Lifetime: 5-60 min  │                │ Lifetime: 1-90 days     │
│ Stateless (JWT)     │                │ Stateful (DB/Redis)     │
│ Contains claims     │                │ Contains no claims      │
│ Not revocable*      │                │ Revocable immediately   │
│ Sent with every req │                │ Sent only to /token     │
│ Verified locally    │                │ Verified at server      │
└─────────────────────┘                └─────────────────────────┘

* Access tokens are not individually revocable in a stateless JWT model.
  Revocation takes effect when the access token expires and the refresh
  token is checked during renewal.
```

**Trade-off**: Shorter access token TTL = faster revocation but more token refreshes. Longer TTL = fewer refreshes but longer window if compromised.

### Refresh Token Rotation

**What it is:** Issue a new refresh token with every token refresh, invalidating the old one.

**Why it matters:**
- If a refresh token is stolen, the attacker races the legitimate user
- When the legitimate user refreshes, the stolen token is invalidated
- If the attacker refreshes first, the legitimate user's token is invalidated — they must re-login, which is a detectable event

**Reuse detection:**
- If an already-consumed refresh token is presented, it means either:
  - The legitimate user's token was stolen (attacker used it first), or
  - The attacker stole a token and is trying to use it
- In both cases, **invalidate the entire refresh token family** — force re-login

**Reuse interval:**
- A small grace window (0–10 seconds) allows for network retries and race conditions
- Client sends refresh request, gets new token, but response is lost (network error)
- Client retries with the old token — reuse interval allows this to succeed
- Without this interval, legitimate users get logged out due to network flakiness

### Concurrent Session Control

| Strategy | Behavior | Use Case |
|----------|----------|----------|
| **Unlimited** | Any number of simultaneous sessions | Consumer apps, low-risk |
| **Limited (N)** | Max N sessions; oldest is revoked when N+1 is created | Business apps, most SaaS |
| **Single session** | Only one session at a time; new login kills old | High-security, shared devices |
| **Per-device-type** | Separate limits per device type (web, mobile, API) | Multi-platform apps |

---

## Our Implementation

### Data Model

Stored in `security_settings.session_config` (JSONB column). The config is schema-free (`map[string]any`).

**Expected JSONB fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `access_token_ttl_minutes` | int | 15 | Access token lifetime |
| `refresh_token_ttl_days` | int | 30 | Refresh token lifetime |
| `max_concurrent_sessions` | int | 5 | Maximum active sessions per user (0 = unlimited) |
| `idle_timeout_minutes` | int | 30 | Inactivity timeout before session is invalidated |
| `absolute_timeout_hours` | int | 24 | Maximum session lifetime regardless of activity |
| `rotate_refresh_tokens` | bool | true | Issue a new refresh token on each refresh |
| `refresh_token_reuse_interval_seconds` | int | 10 | Grace period for reuse detection (for network retries) |

### API Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/security-settings/session` | `GetSessionConfig` | Returns the current session configuration |
| `PUT` | `/security-settings/session` | `UpdateSessionConfig` | Replaces the session configuration |

**Example request body (PUT):**
```json
{
  "access_token_ttl_minutes": 15,
  "refresh_token_ttl_days": 30,
  "max_concurrent_sessions": 3,
  "idle_timeout_minutes": 30,
  "absolute_timeout_hours": 12,
  "rotate_refresh_tokens": true,
  "refresh_token_reuse_interval_seconds": 10
}
```

### Service Layer

- **`GetSessionConfig(ctx, userPoolID)`** — Lazy-creates the security setting row, then returns the `session_config` JSONB.
- **`UpdateSessionConfig(ctx, userPoolID, config, updatedBy, ipAddress, userAgent)`** — Calls `updateConfig` with `"session"` config type.

The update runs in a transaction:
1. Find or create `security_settings` row for the user pool
2. Replace the `session_config` JSONB column
3. Increment `version`
4. Create a `security_settings_audit` row with `change_type: "update_session_config"`

### Audit Trail

Every update creates a `security_settings_audit` row:
- `change_type`: `"update_session_config"`
- `old_config`: Previous session JSONB
- `new_config`: New session JSONB
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

## Recommended Defaults by Application Type

| Setting | Consumer App | Business SaaS | Healthcare/Finance | High Security |
|---------|:------------:|:-------------:|:-----------------:|:-------------:|
| `access_token_ttl_minutes` | 30 | 15 | 5 | 5 |
| `refresh_token_ttl_days` | 90 | 30 | 7 | 1 |
| `max_concurrent_sessions` | 0 (unlimited) | 5 | 3 | 1 |
| `idle_timeout_minutes` | 60 | 30 | 15 | 5 |
| `absolute_timeout_hours` | 720 (30 days) | 24 | 12 | 8 |
| `rotate_refresh_tokens` | true | true | true | true |
| `refresh_token_reuse_interval_seconds` | 10 | 10 | 5 | 0 |

---

## Requirements Checklist

### Token Lifecycle
- [x] Access token TTL is configurable
- [x] Refresh token TTL is configurable
- [x] Configuration is stored per user pool
- [x] Version tracking (auto-incremented on each update)
- [x] Audit trail (old/new config, who changed, IP, user agent)
- [ ] Validation: `access_token_ttl_minutes` must be 1–60
- [ ] Validation: `refresh_token_ttl_days` must be 1–365
- [ ] Sane defaults on first creation
- [ ] Access token contains standard claims (`sub`, `iss`, `aud`, `exp`, `iat`, `jti`)
- [ ] Access token expiry (`exp`) matches configured TTL
- [ ] Refresh token stored in database/Redis (stateful)
- [ ] Refresh token is cryptographically random with at least 128 bits of entropy
- [ ] Refresh token is bound to client ID and user ID

### Refresh Token Rotation
- [ ] New refresh token issued on each refresh when `rotate_refresh_tokens` is true
- [ ] Old refresh token invalidated on rotation
- [ ] Reuse detection: if old token is used again, revoke the entire token family
- [ ] Reuse interval: allow reuse within `refresh_token_reuse_interval_seconds`
- [ ] Token family tracking (link all tokens from original authentication)

### Idle Timeout
- [ ] Track last activity timestamp per session
- [ ] Update last activity on API calls (debounced — not every request)
- [ ] Session invalidated when idle exceeds `idle_timeout_minutes`
- [ ] Idle check occurs during token refresh (not during access token use)
- [ ] Client receives clear error indicating idle timeout vs. other expiry

### Absolute Timeout
- [ ] Track session creation timestamp
- [ ] Session invalidated when total duration exceeds `absolute_timeout_hours`
- [ ] Absolute timeout cannot be extended by activity
- [ ] Force reauthentication after absolute timeout

### Concurrent Session Management
- [ ] Track active sessions per user
- [ ] Enforce `max_concurrent_sessions` on new login
- [ ] Eviction strategy: revoke oldest session when limit exceeded
- [ ] User can view their active sessions (device, IP, last activity, created)
- [ ] User can terminate specific sessions
- [ ] Admin can terminate any session for any user
- [ ] Admin can terminate all sessions for a user (force logout)

### Session Revocation
- [ ] Logout endpoint revokes refresh token
- [ ] Logout endpoint clears session cookies
- [ ] Admin "revoke all sessions" invalidates all refresh tokens for a user
- [ ] Password change revokes all sessions for that user
- [ ] MFA reset revokes all sessions for that user
- [ ] Email change revokes all other sessions

### Session Security
- [ ] Session tokens are not exposed in URLs
- [ ] Refresh token is HttpOnly cookie (not accessible via JavaScript)
- [ ] New tokens issued on privilege escalation (role change, MFA enrollment)
- [ ] Session binding (tie session to user-agent and/or IP range — optional)
- [ ] Session fixation protection (new tokens on authentication)
- [ ] Sliding window refresh: access token refresh resets idle timer but not absolute timer

### Integration & Testing
- [ ] Unit tests for session config service methods
- [ ] Integration test: token refresh flow with rotation
- [ ] Integration test: idle timeout enforcement
- [ ] Integration test: absolute timeout enforcement
- [ ] Integration test: concurrent session eviction
- [ ] Integration test: session revocation on password change
- [ ] Integration test: audit trail with session config changes

---

## References

- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [NIST SP 800-63B §7 — Session Management](https://pages.nist.gov/800-63-3/sp800-63b.html#sec7)
- [RFC 6749 — OAuth 2.0 Authorization Framework](https://datatracker.ietf.org/doc/html/rfc6749)
- [RFC 6819 — OAuth 2.0 Threat Model and Security Considerations](https://datatracker.ietf.org/doc/html/rfc6819)
- [RFC 7519 — JSON Web Token (JWT)](https://datatracker.ietf.org/doc/html/rfc7519)
- [OWASP ASVS v4.0 — V3 Session Management](https://owasp.org/www-project-application-security-verification-standard/)
- [Auth0 Token Best Practices](https://auth0.com/docs/secure/tokens/token-best-practices)
- [PCI DSS v4.0 Requirement 8.2.8 — Session Timeouts](https://www.pcisecuritystandards.org/)
