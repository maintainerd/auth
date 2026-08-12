# Sessions

> Server-side session records that anchor a login's lifecycle (idle + absolute timeout, concurrent-session cap, revocation) and the idempotent, family-aware refresh-token rotation that re-mints tokens against that session.

| | |
|---|---|
| **Status** | Implemented |
| **Code** | `internal/authn` (session service, refresh rotation, reuse detection), `internal/platform/cache` (`RefreshReplayStore` + JTI denylist), `internal/secpolicy` (effective policy resolution), `internal/user` (self-service session endpoints) |
| **Endpoints** | `GET/DELETE /account/sessions`, `DELETE /account/sessions/others`, `DELETE /account/sessions/{session_uuid}` (control plane :8080); `POST /refresh-token`, `POST /logout` (public plane :8081) |
| **Storage** | `user_sessions` table (migration `047_create_user_sessions_table.go`); Redis JTI-denylist keyspace (`jti:deny:*`, incl. logical `rtused:` / `rtgrace:` / `rtfam:` / `rtreplay:` sub-keys) |
| **Config** | Per-tenant `security_settings.session_config` (JSONB) + per-client column overrides. No dedicated env vars — session behavior is tenant-config driven. |

## Overview

A **session** is a durable, server-side record of one authentication event. It is created when a user authenticates (password login, SMS OTP login, or registration auto-login) and lives in the `user_sessions` table. Every access/ID/refresh token minted from that login carries the session's UUID in its `sid` claim, so the session — not the token — is the unit that timeouts and revocation act on.

Two subsystems make up this feature:

1. **Session lifecycle** (`internal/authn/service_session.go`) — create, list, validate-and-touch (idle/absolute enforcement), concurrent-session eviction, and revocation (single / all / all-except-caller).
2. **Refresh-token rotation** (`internal/authn/service_refresh.go` + `internal/platform/cache/cache.go`) — single-use rotation with an idempotent in-window replay window and whole-family revocation on out-of-window reuse (RFC 9700 §4.14.2 / RFC 6819 §5.2.1.1 / OAuth 2.1 §6.1).

A session ends on explicit logout, idle timeout, absolute-lifetime expiry, concurrent-limit eviction, an explicit "sign out everywhere / other devices", or a credential change (password change/reset, MFA change).

## How it works

### Session creation (at login)

1. The login/SMS-login/register service resolves the effective session policy for the client's tenant (`resolveEffectiveSessionPolicy`, `service_security_policy.go:12`).
2. It enforces the concurrent-session cap **before** creating the new row (`enforceConcurrentLimitWithPolicy`), then creates the session (`createSessionWithPolicy`) — e.g. `service_login.go:1081`/`:1098`.
3. `CreateSessionWithPolicy` (`service_session.go:180`) stamps `auth_time`, `ip_address`, `user_agent`, and the authentication facts of the event via `SessionAttributes{AMR, ACR, ClientID, IdentityProviderID, IDPSessionID}`. `acr` defaults to `"1"` when unset; `idle_timeout_seconds` and `expires_at = now + AbsoluteTimeoutSeconds` come from the policy (falling back to 1800s idle / 24h absolute — `defaultIdleTimeoutSeconds`, `defaultAbsoluteLifetimeHours`, `service_session.go:18-21`).
4. The new session UUID becomes the `sid` claim of the issued token set.

### Concurrent-session cap (eviction)

`EnforceConcurrentLimitWithPolicy` (`service_session.go:236`):
- `MaxConcurrentSessions <= 0` disables the cap.
- Counts active sessions; if `count < max`, returns.
- Otherwise sorts active sessions **oldest-first** by `created_at` and revokes `count - max + 1` of them with reason `concurrent_limit`, so once the caller creates the incoming session the user lands at exactly `max`. (Both the "oldest" sort direction and the `+1` trim are deliberate fixes documented inline at `service_session.go:263-281` — the prior code evicted the newest login and never converged.)

### Idle + absolute timeout (validate-and-touch)

`ValidateAndTouch` (`service_session.go:293`) runs on every refresh that reuses a session:
1. `FindActiveByUUID` — a revoked/absent row is treated as unauthorized ("session not found or has been revoked").
2. **Absolute**: `now > expires_at` → revoke with reason `session_expired`, return `401 "session has expired"`.
3. **Idle**: `now - last_active_at > idle_timeout_seconds` → revoke `session_expired`, return `401 "session has expired due to inactivity"`.
4. Otherwise `Touch` bumps `last_active_at = now` (touch failure is logged, not fatal).

### Refresh-token rotation (`POST /refresh-token`)

`RefreshToken` (`service_refresh.go:35`):

1. **Reuse detection runs FIRST** (`rejectRefreshReuse`) — before the shared signature validator, because writing the consumed JTI to the generic access-token denylist would otherwise make a replay fail as merely "invalid" and skip family revocation. Claims are read **unverified** here only to look up `jti`/`rfid`; the signature is verified immediately after.
2. `ValidateTokenWithContext` verifies signature (RS256), expiry, and denylist. The token must have `token_type == "refresh_token"`; `sub` + `client_id` must be present.
3. The user is resolved via `FindBySubAndClientID` and **must be active** — a refresh is the one place a deactivated account would otherwise be handed fresh tokens. The client is resolved by its globally-unique identifier alone (`FindByClientIDAndIdentityProvider(id, "")`).
4. **Session binding**: `sid` is taken from the **signed** refresh token, falling back to the transport value (`X-Session-ID` header or the `sid` read from an unverified access-token cookie) only for legacy tokens. `resolveRefreshSession` (`service_refresh.go:236`) `ValidateAndTouch`es that session. **A refresh never establishes a new session** — a token presenting no `sid` gets `401 "refresh token is not bound to a session"`.
5. A new access/ID/refresh set is minted, carrying the same `sid` and the family id `rfid`.
6. **If `RotateRefreshTokens` is enabled**: the consumed token is denylisted (`denylistConsumedRefreshToken`) and its exact minted response is cached for the overlap window (`cacheRefreshReplay`). When rotation is disabled the token stays reusable by design.

### Reuse guard outcomes (`rejectRefreshReuse`, `service_refresh.go:307`)

| Situation | Detection | Result |
|---|---|---|
| Family already revoked | `rtfam:<rfid>` present | `401 "refresh token family has been revoked"` |
| Token not yet consumed | `rtused:<jti>` (and bare `jti`) absent | Proceed with rotation |
| Consumed, **in-window** duplicate | `rtused:<jti>` present **and** `rtreplay:<jti>` cached | Return the **cached** token set verbatim — idempotent (racing tabs / retries stay signed in) |
| Consumed, **out-of-window** reuse | `rtused:<jti>` present, no cached replay | Revoke the **whole family** (`rtfam:<rfid>`), log HIGH-severity `refresh_token_reuse` security event, `401 "refresh token reuse detected"` |

The idempotent in-window branch means a stolen token replayed inside the window cannot fork a second undetectable session; reuse after the window trips family revocation.

### Revocation paths

| Trigger | Method | Scope | Refresh tokens |
|---|---|---|---|
| Logout (`POST /logout`) | `loginService.Logout` → `RevokeSession` | **Only** the token's `sid` session (no `sid` → revokes nothing, access JTI still denylisted) | `RevokeBySession(sid)` (best-effort) |
| Revoke one session | `RevokeSession` | Single UUID, reason `logout` | `RevokeBySession` |
| Sign out other devices (`DELETE /account/sessions/others`) | `accountService.RevokeOtherSessions` → `RevokeAllExceptUUID` | All except caller's signed `sid`, reason `user_revoke` | n/a (session rows) |
| Sign out everywhere (`DELETE /account/sessions`) | `RevokeAllSessions` | All sessions for user, reason `user_revoke` | `RevokeByUserID` (best-effort) |
| Password change / reset, MFA change | `RevokeAllExceptUUID` / `RevokeAllSessions` (via `internal/user`) | All (or all-except-caller for self-service change) | family revoked |

`RevokeSession` and `RevokeAllSessions` are the two places OAuth refresh tokens are also revoked, through the optional `RefreshTokenRevoker` interface (nil = revocation skipped); `internal/app` binds the real OAuth repository as the adapter. Global refresh revocation is deliberately confined to the "everywhere"/credential-change path — an ordinary per-session logout must never reach it.

### Expiry cleanup

`userSessionRepository.DeleteExpired()` (`repository_user_session.go:113`) is swept by the OAuth cleanup runner's background ticker (`internal/oauth/cleanup_runner.go:94`). It hard-deletes rows where `expires_at < now` OR (revoked and `revoked_at < now - 30 days`), keeping recently-revoked rows for auditability.

## Implementation

**Session service** — `internal/authn/service_session.go`
- `SessionService` interface: `ListSessions`, `RevokeSession`, `RevokeAllSessions`, `CreateSession`, `EnforceConcurrentLimit`, `ValidateAndTouch` (`:32`). Policy-aware twins `CreateSessionWithPolicy` / `EnforceConcurrentLimitWithPolicy` are reached by runtime type-assertion (`service_security_policy.go:136-166`).
- `RefreshTokenRevoker` (`:47`): `RevokeByUserID`, `RevokeBySession` — the OAuth-token slice this package needs without an import cycle.
- `SessionAttributes` (`:167`): AMR / ACR / ClientID / IdentityProviderID / IDPSessionID — properties of the auth event, not the token.

**Model + repository** — `model_user_session.go`, `repository_user_session.go`
- `UserSession` (`model_user_session.go:11`) maps table `user_sessions`; `IsRevoked()`/`IsExpired()` helpers; UUID auto-generated in `BeforeCreate`.
- Repo methods: `FindActiveByUserID` (newest-first, active only), `FindActiveByUUID`, `CountActive`, `Create`, `Touch`, `RevokeByUUID`, `RevokeAllByUserID`, `RevokeAllExceptUUID`, `DeleteExpired`, `WithTx`.

**Refresh rotation** — `internal/authn/service_refresh.go`
- `RefreshToken` (`:35`), `rejectRefreshReuse` (`:307`), `resolveRefreshSession` (`:236`), `denylistConsumedRefreshToken` (`:267`), `cacheRefreshReplay`/`fetchRefreshReplay` (`:193`/`:216`).
- Key helpers: `refreshUsedKey`→`rtused:`, `refreshGraceKey`→`rtgrace:`, `refreshFamilyKey`→`rtfam:` (`:384-388`). `logSafe` strips CR/LF from unverified-token values (CWE-117).

**Replay + denylist cache** — `internal/platform/cache/cache.go`
- `JTIDenylister` (`DenyJTI`/`IsJTIDenied`, keyspace `jti:deny:`) — `IsJTIDenied` returns `false` on Redis error so an outage does not break validation.
- `RefreshReplayStore` (`StoreRefreshReplay`/`GetRefreshReplay`, keyspace `rtreplay:`, gob-encoded `LoginResponseDTO`) — backends that don't implement it (e.g. `NopJTIDenylister`) simply skip idempotent replay and fall back to strict revoke-on-reuse.

**Self-service endpoints** — `internal/user/handler_account.go`, `internal/user/routes.go`
- `ListSessions` (`:528`), `RevokeSession` (`:547`), `RevokeAllSessions` (`:580`), `RevokeOtherSessions` (`:615`, caller session read from the **signed** `sid` claim). All revoke actions write an audit event.

**Policy resolution** — `internal/secpolicy/validation_setting.go:440` (`ResolveEffectiveSessionPolicy`), struct `EffectiveSessionPolicy` (`internal/secpolicy/types.go:173`), defaults `internal/secpolicy/defaults_setting.go:41`.

**Table** — `user_sessions` (migration 047): `user_session_id` PK, `user_session_uuid` UNIQUE, `user_id`, `tenant_id`, `client_id`, `identity_provider_id`, `auth_time`, `ip_address` (INET), `user_agent`, `amr` (text[]), `acr` (varchar(10) default `'1'`), `idp_session_id`, `idle_timeout_seconds` (default 1800), `last_active_at`, `expires_at`, `revoked_at`, `revoked_reason` (CHECK-constrained), `created_at`. Partial indexes on `(user_id, created_at)`, `(tenant_id, user_id)`, `expires_at`, `(user_id, last_active_at)`, `client_id`, `idp_session_id` — all `WHERE revoked_at IS NULL` (except idp_session_id). FKs cascade on user/tenant delete; client/IdP set-null.

`revoked_reason` values (`internal/shared/constants.go:125-134`): `logout`, `admin_revoke`, `password_change`, `password_reset`, `mfa_change`, `session_expired`, `concurrent_limit`, `suspicious_activity`, `user_revoke`, `role_change`.

## Configuration

Session behavior is **per-tenant config**, not environment variables. It lives in `security_settings.session_config` (JSONB) and is resolved into `EffectiveSessionPolicy` at runtime.

| `session_config` key | Default | Effective field | Meaning |
|---|---|---|---|
| `access_token_ttl_minutes` | 15 | `AccessTokenTTLSeconds` | Access-token lifetime |
| `refresh_token_ttl_days` | 30 | `RefreshTokenTTLSeconds` | Refresh-token lifetime (session ceiling) |
| `max_concurrent_sessions` | 5 | `MaxConcurrentSessions` | Concurrent cap (`0` = unlimited) |
| `idle_timeout_minutes` | 30 | `IdleTimeoutSeconds` | Inactivity window |
| `absolute_timeout_hours` | 24 | `AbsoluteTimeoutSeconds` | Hard session lifetime |
| `rotate_refresh_tokens` | `true` | `RotateRefreshTokens` | Enable single-use rotation + reuse detection |
| `refresh_token_reuse_interval_seconds` | 10 | `RefreshTokenReuseIntervalSeconds` | Idempotent overlap window for in-flight retries |
| `cookie_secure` | `true` | `CookieSecure` | `Secure` on token cookies |
| `cookie_http_only` | `true` | `CookieHTTPOnly` | `HttpOnly` on token cookies |
| `cookie_same_site` | `"Lax"` | `CookieSameSite` | `SameSite` on token cookies |
| `revoke_sessions_on_password_change` | `true` | `RevokeSessionsOnPasswordChange` | Kill sessions when password changes |

**MFA coupling**: when the tenant's `mfa_config.mode == "enforced"`, `RequiredACR` is raised to `"2"` (`validation_setting.go:464`).

**Per-client overrides** (columns on `clients`, via `SecuritySettingClientOverrides`, `service_security_policy.go:90`): `AccessTokenTTL`, `RefreshTokenTTL`, `SessionIdleTimeout`, `SessionAbsoluteTimeout`, `RequiredACR`, `RequirePKCE`. Timeout/TTL overrides only ever **tighten** — they apply solely when set, `> 0`, and **shorter** than the tenant value; `RequiredACR` can only escalate (to `"2"`), never relax.

## Security considerations

- **Session is the revocation unit, not the token.** Every token carries `sid`; logout and "sign out" act on the session row, and `ValidateAndTouch` re-checks the row on every refresh — so a revoked session immediately stops re-minting tokens.
- **Refresh rotation is single-use with idempotent grace.** In-window duplicates return the cached set (no session fork); out-of-window reuse is treated as compromise and revokes the entire token family, emitting a HIGH-severity `refresh_token_reuse` security event.
- **Reuse detection precedes signature validation** by design; consumed JTIs are written to a refresh-scoped denylist key (`rtused:`) rather than the generic access-token denylist, so replay reaches the family-revocation branch instead of failing as a generic "invalid token".
- **A refresh cannot mint a session.** A token with no `sid` is rejected, closing the hole where a stolen refresh token could survive "sign out everywhere" by omitting the session cookie.
- **Deactivated accounts are stopped at refresh** — the user must be `active` or the exchange returns 401.
- **Least-scope revocation** — per-session logout never triggers global refresh revocation; only explicit "everywhere" and credential-change flows do.
- **Signed `sid` for caller identity** — "sign out other devices" reads the session to spare from the signed claim, so a caller cannot nominate another user's session.
- **Cache-outage resilience** — `IsJTIDenied` fails open on validation (does not block auth) but replay/denylist writes fail loudly; missing replay storage degrades to strict revoke-on-reuse, never to a weaker check.
- **Log-injection hardening** — unverified token values are CR/LF-stripped (`logSafe`) before logging.
- Access tokens are **RS256-signed**; the issuer derives from the server's public-hostname config, not a legacy `ISSUER_URL`.

## Related

- [./cryptography-and-keys.md](./cryptography-and-keys.md) — JWT issuance, claims (`sid`, `rfid`, `acr`, `amr`), RS256 signing, JTI denylist.
- [./authentication.md](./authentication.md) — the authentication flows that create sessions (password, SMS OTP, registration auto-login).
- [./oauth2-oidc.md](./oauth2-oidc.md) — OAuth refresh tokens revoked via the `RefreshTokenRevoker` adapter, and `sid`-stamped tokens minted at `/authorize`.
- [./multi-factor-auth.md](./multi-factor-auth.md) — MFA-enforced mode escalates a session's `acr`/`RequiredACR`.
- [./security-settings.md](./security-settings.md) — full per-tenant `session_config` / client-override reference.
