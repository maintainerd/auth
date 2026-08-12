# Authentication

> First-factor, interactive user sign-in for maintainerd-auth: email/password, magic-link, SMS/email-verification OTP, and the forgot/reset-password recovery flow — all served on the public identity plane (`:8081`).

| | |
|---|---|
| **Status** | Implemented |
| **Code** | `internal/authn` (services, handlers, routes); `internal/platform/security` (hashing, rate-limit, lockout); `internal/platform/jwt` (token minting); `internal/secpolicy` (per-tenant policy) |
| **Endpoints** | `POST /api/v1/login`, `/refresh-token`, `/logout`, `/register`, `/register/invite`, `/forgot-password`, `/reset-password`, `/email-verification/send`, `/email-verification/verify`, `/magic-link/send`, `/magic-link/verify`, `/sms-login/send`, `/sms-login/verify`, `GET /api/v1/registration_context` — all on the public plane (`:8081`) |
| **Storage** | `users`, `user_tokens`, `user_identities`, `user_password_history`, `user_sessions` |
| **Config** | `APP_PUBLIC_HOSTNAME`, `CAPTCHA_SECRET` (optional); per-tenant `security_settings` (`password_config`, `lockout_config`, `registration_config`, `threat_config`, `mfa_config`, `session_config`, `token_config`) |

## Overview

`internal/authn` owns every way a human establishes a **first factor** with the system and every credential-recovery path. Second-factor verification (TOTP, WebAuthn, SMS/email OTP as step-up), enrollment, and trusted devices live in the MFA subsystem and are only *invoked* from here — see `./multi-factor-auth.md`.

Interactive authentication is **public-only**. All the routes below are mounted exclusively on the public identity router (port `:8081`, `internal/server/router.go:234-311`). The internal/management router (port `:8080`, VPN-only) mounts **no** interactive login: the admin console authenticates its operators by starting an OAuth flow against the hosted identity app rather than posting credentials to the internal API (`internal/server/router.go:65-73`). The near-identical internal-plane handler twins that used to exist (`LoginRoute`, `EmailVerificationRoute`, `ForgotPasswordRoute`, `ResetPasswordRoute`, `SMSLoginInternalRoute`) were removed because nothing mounted them (`internal/authn/routes.go:40-49`, `service_login.go:30-34`, `service_register.go:42-47`).

Users are **tenant-isolated**: the same username/email/phone can exist in many tenants, so every lookup is tenant-scoped and the caller must supply a tenant context. On the public plane that context is a **`client_id`** (query param); `tenant_id` is rejected on the public login/register/magic-link surfaces (`handler_login.go:327-333`). The service derives the tenant from the resolved client.

Email is delivered **over SMTP only**. Any relay (Amazon SES, Mailgun, SendGrid, Postmark, …) is reached through its SMTP endpoint — there are no dedicated email-provider API integrations (`internal/platform/email/provider.go:21-22`, `factory.go:77-84`). See `./email-and-sms.md`.

## How it works

### Password login (`LoginService.LoginPublic`, `service_login.go:205`)

1. **Resolve client** for rate-limit scope and tenant (`resolvePublicClient`); load the tenant's lockout policy.
2. **Rate limit** by `tenantID:usernameOrEmail` against the tenant lockout policy (`security.CheckRateLimitWithConfig`). Router additionally caps credential endpoints at **10 req/min per IP** (`router.go:232`).
3. **Threat detection** — pre-auth velocity / brute-force / impossible-travel risk (`security.AssessLoginThreat`); a `Blocked` decision rejects, a step-up decision may force MFA later.
4. **Lockout check** before any password comparison (`s.checkLockout`).
5. In a read transaction, re-resolve the client (must be active, have a domain, resolve a tenant) and **find the user** within that tenant — username first, then email if the input contains `@` (`findLoginUser`, `service_login.go:142`).
6. **Verify password** with `security.ComparePassword`. On a user miss the code still runs a **dummy bcrypt compare** (`GetDummyBcryptHash`) so response time does not reveal whether the account exists (`verifyLoginPassword:953`).
7. On failure: record the failed attempt, feed the threat + lockout counters, return a generic `invalid credentials`.
8. On success: enforce **account status** (`active`; a `pending` unverified account returns `email is not verified`), then the tenant's **email-verification** and **phone-verification** gates (`enforceLoginEmailVerification`, `enforceLoginPhoneVerification`), ensure a `user_identities` row for the client, reset lockout/rate counters.
9. Post-auth: opportunistic **HIBP compromised-password** check, **password-expiry** and **temporary-password-expiry** evaluation (may set `ForcePasswordChange` or reject an expired temp password).
10. If `ForcePasswordChange` is set, return `{RequirePasswordChange: true}` with **no tokens**.
11. Apply the tenant **login-MFA policy** (`loginMFAChallengeResponse`). When a second factor is required it returns an MFA challenge (`MFARequired`, challenge token, allowed methods) instead of tokens; otherwise the login proceeds at `acr=1`.
12. **Issue the token set** (`generateTokenResponseWithAuth`): create a `user_sessions` row (enforcing the concurrent-session limit), then mint access + ID + refresh JWTs bound to that session. `amr=[pwd]`, `acr=1`.

### Registration (`RegisterService`, `service_register.go`)

- **`RegisterPublic`** (`:387`) — self-service signup. Gates: `self_registration_enabled` (tenant) and `AllowRegistration` (client), the per-IdP registration gate, optional named **registration flow** (grants extra roles, may force email verification), required-field enforcement, abuse controls (signup CAPTCHA *if* `CAPTCHA_SECRET` is configured, per-IP registration rate limit), email-domain allow/block lists, password-policy validation + history, and generic account-enumeration-safe conflict messages for email/phone (username conflicts stay explicit). Creates the `users` row (`is_email_verified` always starts **false** — verification reflects proven control, not policy), a `user_identities` row, the default `registered` role plus any flow roles, records ToS consent, and **signs the user in** (creates a session + token set). If the tenant requires email verification, a verification OTP is sent asynchronously after commit.
- **`RegisterInvitePublic`** (`:624`) — redeem an invite token. The invited email is trusted (`is_email_verified: true`), email-OTP MFA is auto-enrolled, and the user is created `active`. Hard email-domain **blocks** still apply even though the invite overrides the self-signup allowlist.

### Magic-link login (`MagicLinkService`, `service_magic_link.go`)

- **Send** (`:83`): opt-in per client (`AllowMagicLink`, enforced server-side). Looks up the user by email in the client's tenant, revokes outstanding magic-link tokens, mints a **32-byte opaque token** (64 hex), stores only its hash (`hashUserBearerToken`) with a **15-minute** TTL (`MagicLinkTokenTTL`), and emails a **signed URL** to `/api/v1/magic-link/verify`.
- **Verify** (`LoginWithMagicLink:189`): matches the hashed token, checks expiry, resolves the tenant-matched active user, **single-use revokes** it (and any siblings), marks the email verified (possession proves email ownership), then hands off to the login coordinator for the phone-verification gate → login-MFA policy → session issuance. With no coordinator it **fails closed** (a token with no `sid` can't be revoked and is rejected by session middleware).

### Email-verification OTP (`EmailVerificationService`, `service_email_verification.go`)

- **Send** (`:80`): tenant-scoped user lookup; no-op (masked success) if the address is unknown, already verified, or the account is suspended. Revokes prior tokens, generates a **6-digit OTP** (`crypto.GenerateOTP`), stores its hash (`crypto.HashAuthorizationCode`) with a **1-hour** TTL (overridable by `registration.verification_token_ttl_hours`), and emails it.
- **Verify** (`:208`): resolves the user **from the token itself** (not a global email lookup) to avoid cross-tenant collisions on a 6-digit code, requires the supplied email to match, sets `is_email_verified=true` + `status=active` + `email_verified_at`, single-use revokes all verification tokens, and **invalidates the cached user context** so `/account` reflects the change immediately.

### Forgot / reset password (`service_forgot_password.go`, `service_reset_password.go`)

- **Forgot** (`SendPasswordResetEmail:72`): mints a **32-byte** reset token (hash stored, **1-hour** TTL, revokes prior), emails a signed URL to `/api/v1/reset-password`. Two **account-enumeration** defenses: the SMTP send is dispatched off the request path (goroutine on a detached context), and *every* response — hit or miss — is padded to a **250 ms floor** (`padToMinDuration`). The response body is always the generic "if an account exists…".
- **Reset** (`ResetPassword:64`): validates the token (active, unexpired, tenant-matched, active user), enforces password policy + reuse history, hashes and writes the new password, clears `force_password_change` / `temporary_password_expires_at`, then — unless the tenant sets `revoke_sessions_on_password_change=false` — **revokes every `user_sessions` row AND every OAuth refresh token** for the user (a reset is the compromised-account recovery path, so it must terminate all live access), and single-use revokes the reset token + siblings.

### Logout (`LoginService.Logout`, `service_login.go:440`)

Parses the access token, **denylists its `jti`** in Redis for the remaining TTL, records a DB revocation row (RFC 7009, best-effort, tenant-scoped), then revokes **only the session bound to the token's `sid`** — never all sessions. Console + identity share one browser session so this signs the user out of both; a second device keeps its own session. "Sign out everywhere" is the separate `DELETE /account/sessions` control. `?forget_device=true` additionally revokes the trusted-device row and clears its cookies.

### Token issuance (`token_helper.go`)

The access, ID, and refresh tokens are minted together (`generateTokenSetWithAuthContext:87`). The **`sub`** is the `user_identities.sub` for the client (an independently minted UUID for built-in identities), **not** `users.user_uuid`. The **`iss`** is derived from `APP_PUBLIC_HOSTNAME` — the authorization server's public hostname advertised by discovery — via `jwt.TokenIssuer` (`internal/platform/jwt/issuer.go:24`); the client domain is only a fallback when that env var is unset. The **realm** (`provider_id`) is anchored to the **tenant name**, so password-login tokens match the OAuth and federation paths. Tokens are signed **RS256 by default**; a tenant's `signing_algorithm` may select **PS256** (ES256 is explicitly rejected — `jwt.go:948-958`), and all three tokens in the set honor that choice consistently.

## Implementation

| Concern | Location |
|---|---|
| Route mounting (public plane) | `internal/server/router.go:251-308`; `internal/authn/routes.go` |
| Password login | `service_login.go:205` (`LoginPublic`), handler `handler_login.go:54` |
| Refresh / logout handlers | `handler_login.go:335` (`Logout`), `:395` (`RefreshToken`) |
| Registration | `service_register.go:387` / `:624`; handler `handler_register.go` |
| Magic-link | `service_magic_link.go`; handler `handler_magic_link.go` |
| Email verification | `service_email_verification.go`; handler `handler_email_verification.go` |
| Forgot / reset password | `service_forgot_password.go`, `service_reset_password.go`; handlers `handler_forgot_password.go`, `handler_reset_password.go` |
| SMS OTP login | `service_sms_login.go` (`smsOTPLength=6`, `smsOTPTTL=10m`, max 3 failures); handler `handler_sms_login.go` |
| Account-link confirm | `service_account_link_request.go`, `handler_account_link.go`, route `routes.go:22` (JWT-authenticated, **not** on the unauthenticated public surface) |
| Password hashing / verify | `internal/platform/security/hash.go` |
| Token minting / issuer / realm | `token_helper.go`; `internal/platform/jwt/issuer.go`, `jwt.go` |

**Password hashing** (`internal/platform/security/hash.go`): the plaintext is SHA-256-prehashed before hashing (avoids bcrypt's 72-byte truncation). `HashPasswordWithPolicy` selects the storage KDF from the tenant password policy — `bcrypt` (cost **12**, `BcryptCost`) is the legacy/default-compatible format, while `argon2id` (the seeded policy default, `password_policy.go:52`), `scrypt`, and `pbkdf2` are stored in a self-describing `$maintainerd$<alg>$<params>$<salt>$<key>` envelope. `ComparePassword` detects the stored format from a cheap prefix and runs exactly one algorithm family, so verification never leaks the stored algorithm via timing and existing bcrypt users are not locked out during rollout.

**Key tables** (`internal/shared/constants.go` token types): `users` (nullable `password`, `is_email_verified`, `is_phone_verified`, `status`, `password_changed_at`, `force_password_change`, `temporary_password_expires_at`, `last_login_at`, `login_count`); `user_tokens` (`token_type` ∈ `user:magic_link` / `user:email:verification` / `user:password:reset`, hashed `token`, `expires_at`, `is_revoked`); `user_identities` (`sub` per client); `user_password_history`; `user_sessions` (canonical session store validated by `UserContextMiddleware`).

## Configuration

| Setting | Where | Governs |
|---|---|---|
| `APP_PUBLIC_HOSTNAME` | env (`config.go:156`) | Token `iss`, discovery issuer, and the base of signed magic-link / reset-password URLs |
| `CAPTCHA_SECRET` | env | Enables the signup CAPTCHA check; when unset, a tenant's `captcha_on_signup` flag is logged as unenforceable (one line per tenant) rather than blocking all signups (`service_register.go:183-212`) |
| `BcryptCost` = 12 | constant (`hash.go:23`) | bcrypt work factor (not env-tunable) |
| `password_config` | per-tenant `security_settings` | Length/complexity, `hash_algorithm`, common-password + HIBP rejection, `history_count`, expiry days, temporary-password window |
| `lockout_config` | per-tenant | Max failed attempts, duration, progressive/auto-unlock, reset-on-success |
| `registration_config` | per-tenant | `self_registration_enabled`, email/phone-verification requirements, allow/block email domains, `verification_token_ttl_hours`, `captcha_on_signup`, per-IP registration rate limit |
| `threat_config` | per-tenant | Brute-force / velocity / impossible-travel detection, risk-based step-up |
| `session_config` / `token_config` | per-tenant | Session lifetimes + cookie policy, access/refresh TTLs, `signing_algorithm` (RS256/PS256), `revoke_sessions_on_password_change` |
| `mfa_config` | per-tenant | Login-MFA mode (`disabled`/`optional`/`enforced`), allowed methods, grace periods — consumed here, owned by `./multi-factor-auth.md` |
| Client flags | `clients` row | `AllowRegistration`, `AllowMagicLink`, active status + domain (all required to authenticate) |

Fixed token lifetimes in code: magic-link **15 min**, email-verification OTP **1 h** (policy-overridable), password-reset **1 h**, SMS OTP **10 min**.

## Security considerations

- **Account-enumeration resistance**: dummy bcrypt compare on user miss (login); generic messages on login, forgot-password, magic-link send, and email-verification send; forgot-password additionally moves SMTP off the request path and pads every response to a 250 ms floor; public registration returns a generic conflict for existing email/phone (username stays explicit by design).
- **Tenant isolation**: every credential lookup is tenant-scoped; email-verification and password-reset resolve the user from the token to avoid cross-tenant OTP/token collisions and verify the token's tenant matches the resolved client.
- **Single-use, hashed tokens**: magic-link, email-verification, and reset tokens are stored only as hashes and revoked on use (plus siblings); reset tokens are never written to logs (only the user UUID is).
- **Credential-change blast radius**: a password reset revokes all sessions *and* all OAuth refresh tokens (opt-out via tenant policy); plain logout revokes only the one bound session; email verification busts the cached user context.
- **Rate limiting & lockout**: per-IP (10 req/min) at the router, per-identifier rate limit + tenant lockout policy + threat detection before password verification.
- **Password storage**: SHA-256 prehash + bcrypt cost 12 (or argon2id/scrypt/pbkdf2 per tenant policy); constant-time comparison; format-detection avoids algorithm-disclosure timing.
- **Session-bound tokens**: issued tokens carry a `sid`; a token without one is rejected by session middleware, so no unrevocable credential is ever handed out (magic-link fails closed without a session coordinator; registration and SMS login create sessions like login).
- **First-party guard**: self-service account mutations (including account-link confirm) sit behind `RequireFirstPartyClient` + CSRF, so a third-party OAuth token for `openid profile` cannot drive them.

## Related

- `./multi-factor-auth.md` — second-factor verification, enrollment, trusted devices, step-up (invoked from login here)
- `./sessions.md` — `user_sessions` store, concurrent-session limits, idle/absolute timeouts, cookie policy
- `./oauth2-oidc.md` — OAuth/OIDC grant flows and the hosted authorize surface the console logs in through
- `./cryptography-and-keys.md` — access/ID/refresh token structure, RS256/PS256 signing, issuer, refresh rotation + reuse detection
- `./authentication.md` — passwordless SMS one-time-code sign-in
- `./security-settings.md` — full password-policy, history, HIBP, and hashing-algorithm details
- `./email-and-sms.md` — SMTP-only transactional email and per-tenant templates
</content>
