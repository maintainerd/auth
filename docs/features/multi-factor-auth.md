# Multi-Factor Authentication

> Second-factor enrollment, verification, and step-up elevation across TOTP, WebAuthn/passkeys, SMS OTP, email OTP, and single-use backup codes, gated by a per-tenant MFA policy.

| | |
|---|---|
| **Status** | Implemented (grace-period fields are loaded into policy but enforced by the login flow, not this package — see [Configuration](#configuration)) |
| **Code** | `internal/mfa` (service, WebAuthn, handlers, repos, models); `internal/secpolicy/mfa_policy.go` (effective policy); `internal/notifier` (OTP store); `internal/platform/sms`, `internal/platform/email` (delivery) |
| **Endpoints** | `/mfa/*` mounted on both the internal console (`:8080`) and public identity (`:8081`) surfaces — see [Implementation](#implementation) |
| **Storage** | `user_mfa_totp_secrets`, `user_mfa_webauthn_credentials`, `webauthn_challenges`, `user_mfa_phones`, `user_mfa_emails`, `user_mfa_backup_codes`, `user_otps`, `user_trusted_devices`; boolean flags on `users` |
| **Config** | `WEBAUTHN_RP_ID` (override), `AppPublicHostname` (RP-ID/issuer source); per-tenant `security_settings.mfa_config` (JSONB) |

## Overview

MFA lets a user prove possession of a second factor after a password. The `internal/mfa` package owns five factor types and two ceremonies:

| Factor | Enroll | Login 2nd step | Step-up | amr emitted |
|--------|:------:|:--------------:|:-------:|-------------|
| **TOTP** (RFC 6238, SHA-1) | yes | yes | yes | `pwd`, `otp` |
| **WebAuthn / passkey** (FIDO2) | yes | yes | yes | `pwd`, `user`, `swk`\|`hwk` |
| **SMS OTP** | yes | yes | yes | `pwd`, `sms` |
| **Email OTP** | yes | yes | yes | `pwd`, `otp` |
| **Backup code** (single-use) | auto-issued on first factor | — | yes | `pwd`, `mfa` |

Two ceremonies consume those factors:

- **Login second step** — after a correct password, `internal/authn` challenges for a factor and, on success, elevates the freshly issued session to `acr=2`. The mfa package exposes `VerifyFactor` / `SendSMSChallenge` / `SendEmailOTPChallenge` / `BeginWebAuthnLogin` / `EnrolledMFAMethods` to authn via the `MFAFactorAuthenticator` interface (`internal/authn/service_login.go:58`) so authn never imports mfa.
- **Step-up** — an already-authenticated session proves a factor *right now* to reach `acr=2` for a sensitive action (admin operation, email/password change). `IssueStepUpChallenge` → complete a factor → `VerifyStepUp` mints a new `acr=2` access token.

Every factor and both ceremonies are filtered through the tenant's effective MFA policy (`secpolicy.LoadMFAPolicy`); a factor the tenant disabled is refused end to end.

## How it works

### TOTP enrollment (`POST /mfa/totp/enroll` → `POST /mfa/totp/verify`)

1. `BeginTOTPEnrollment` checks the tenant permits `totp`, generates a secret via `pquerna/otp` with the tenant's issuer/digits/period, encrypts it with `crypto.EncryptAtRest`, and upserts a **pending** (`is_enabled=false`) row. It returns the base32 secret, the `otpauth://` URI (for QR rendering), and the digits/period so the client sizes its input (`service_mfa.go:248`).
2. `FinishTOTPEnrollment` validates the first code, sets `is_enabled=true`, stamps `users.is_totp_enabled` + `first_mfa_enrolled_at`, and issues a fresh set of backup codes returned once in plaintext (`service_mfa.go:332`).
3. Verification (`validateTOTPAndStep`, `service_mfa.go:464`) accepts the current step and ±1 step (±30 s skew), then `MarkStepUsed(userID, step)` records `last_used_step` so a code cannot be replayed within its own window.

### WebAuthn (`/mfa/webauthn/register/{begin,finish}`, `/mfa/webauthn/auth/{begin,finish}`)

1. Begin stores the ceremony `SessionData` in the cache (`webauthn:session:` prefix, 5-min TTL) **and** a `webauthn_challenges` row (single-use, server-issued).
2. Finish re-loads the session, `Consume`s the challenge (invalid/expired/reused → error), then validates against a **per-request** verifier built from the ceremony's actual origin (`waForOrigin`, `service_webauthn.go:542`). The origin is accepted only when its host equals the RP ID or is a subdomain of it — so any tenant subdomain under the RP ID passes, with no static origin list. On registration the credential (public key, sign count, transports, backup-eligible/active flags) is persisted and `users.is_webauthn_enabled` is set. On authentication the sign count is checked for regression (clone detection) and updated.

### SMS / Email OTP enrollment (`/mfa/sms/*`, `/mfa/email-otp/*`)

1. Enroll checks policy, applies a per-recipient send throttle (`checkAndRecordOTPSend`), generates a 6-digit code via `crypto.GenerateOTP`, stores its hash (`crypto.HashAuthorizationCode`) in `user_otps` with a 10-min TTL, associates the pending (unverified) phone/email, then dispatches: SMS via `sms.NewProviderFromDB`, email via `email.SendEmail` (SMTP).
2. Verify constant-time compares the hash (`subtle.ConstantTimeCompare`), marks the OTP used, sets `is_verified=true`, and stamps `first_mfa_enrolled_at` if still null (`ensureMFAFlag`).

### Login second step & step-up

`verifyFactor` (`service_mfa.go:1602`) is the single factor-verification switch shared by both ceremonies. For step-up, `VerifyStepUp` first validates the challenge token, confirms its `sub` matches the authenticated user, checks the method is in the token's `allowed_methods` (fails **closed** on a missing/malformed claim, `stepUpMethodAllowed`), verifies the factor, then re-mints the current session's access token with `acr=2` and the factor's `amr` — preserving `sub`, `client_id`, `provider_id`, `scope`, `session_id` (`service_mfa.go:1499`).

### Trusted devices

`IssueTrustedDevice` records the current device as trusted for N days via an atomic upsert on a `(user_id, device_fingerprint)` partial unique index, returning a one-time opaque secret; only its hash (`device_token_hash`) is stored. `TrustedDeviceValid` matches on `user_id + tenant_id + hash + not-expired` — the **tenant predicate is load-bearing** so trust granted in a lax tenant cannot skip MFA in a strict tenant on the same account (`service_mfa.go:1769`).

## Implementation

### Routes

Mounted by `MFAInternalRoute` (`:8080`) and `MFAPublicRoute` (`:8081`) in `internal/mfa/routes.go`. All routes require `JWTAuthMiddleware` + `UserContextMiddleware` and a per-route `account:mfa:*:self` (or `user:mfa:reset`) permission. Self-service **enrollment/management** is public-surface only; the console gets step-up + admin reset.

| Method + Path | Handler | Guard beyond permission |
|---|---|---|
| `GET /mfa/status` | `GetStatus` | — (both surfaces) |
| `POST /mfa/totp/enroll` · `/totp/verify` | `BeginTOTPEnrollment` · `FinishTOTPEnrollment` | `RequireStepUpForNewFactor` |
| `DELETE /mfa/totp` | `DisableTOTP` | `RequireFreshStepUp` |
| `GET /mfa/backup-codes/count` | `GetBackupCodesCount` | — |
| `POST /mfa/backup-codes/regenerate` | `RegenerateBackupCodes` | `RequireFreshStepUp` |
| `POST /mfa/webauthn/register/{begin,finish}` | `WebAuthnBegin/FinishRegistration` | `RequireStepUpForNewFactor` |
| `POST /mfa/webauthn/auth/{begin,finish}` | `WebAuthnBegin/FinishAuthentication` | — (step-up ceremony) |
| `DELETE /mfa/webauthn/{credential_uuid}` | `WebAuthnDeleteCredential` | `RequireFreshStepUp` |
| `GET /mfa/webauthn/{credential_uuid}/download` | `WebAuthnDownloadCredential` | `RequireFreshStepUp` |
| `POST /mfa/sms/enroll` · `/sms/verify` | `EnrollSMS` · `VerifySMS` | `RequireStepUpForNewFactor` |
| `DELETE /mfa/sms` | `DisableSMS` | `RequireFreshStepUp` |
| `POST /mfa/email-otp/enroll` · `/email-otp/verify` | `EnrollEmailOTP` · `VerifyEmailOTP` | `RequireStepUpForNewFactor` |
| `DELETE /mfa/email-otp` | `DisableEmailOTP` | `RequireFreshStepUp` |
| `POST /mfa/step-up/challenge` | `IssueStepUpChallenge` | — |
| `POST /mfa/step-up/send-sms` · `/send-email-otp` | `SendStepUpSMS` · `SendStepUpEmailOTP` | — |
| `POST /mfa/step-up/verify` | `VerifyStepUp` | — |
| `POST /mfa/reset` | `SelfResetMFA` | `RequireFreshStepUp` |
| `POST /mfa/admin/users/{user_uuid}/reset` | `AdminResetMFA` | `middleware.RequireStepUp` + `user:mfa:reset` |
| `POST /mfa/admin/users/{user_uuid}/reset/{method}` | `AdminResetMFAMethod` | `middleware.RequireStepUp` + `user:mfa:reset` |

`{method}` for the single-factor admin reset is `totp` \| `webauthn` \| `sms` \| `email_otp` \| `backup_code` (`service_mfa.go:918`).

### Step-up guards (`handler_mfa.go`)

- `RequireFreshStepUp` — the **caller** must hold `acr=2` issued within the tenant's step-up TTL. Destructive self-service (disable a factor, regenerate codes, self-reset, download/delete a passkey) requires it. Replaced a guard that passed if the *account* merely had any factor, which let a stolen `acr=1` cookie wipe MFA.
- `RequireStepUpForNewFactor` — same demand, but skipped while the account holds **no** factor (so the first factor can be bootstrapped from `acr=1`). Fails **closed** (503) if MFA state is unreadable. Closes the additive hole where an attacker enrolled their own authenticator on a hijacked `acr=1` session.
- `RequirePolicyStepUp` — gates sensitive non-MFA actions (e.g. email change) only when `require_mfa_for_sensitive_actions` is on **and** the user has an enrolled factor; fails closed (503) on a policy-read error.

### Key functions & files

| Area | Location |
|---|---|
| Factor verification switch (shared by login + step-up) | `verifyFactor`, `service_mfa.go:1602` |
| TOTP validate + step (±1 window) | `validateTOTPAndStep`, `service_mfa.go:464` |
| Backup-code count (bcrypt-only, filters stale SHA-256 rows) | `GetBackupCodesCount`, `service_mfa.go:545` |
| Effective per-tenant policy | `secpolicy.LoadMFAPolicy`, `mfa_policy.go:39` |
| Policy method gate | `methodAllowed`, `service_mfa.go:1994` |
| Per-request WebAuthn origin verifier | `waForOrigin`, `service_webauthn.go:542` |
| RP-ID derivation | `rpIDFromHostname`, `service_webauthn.go:522` |
| Trusted-device issue/validate | `IssueTrustedDevice` / `TrustedDeviceValid`, `service_mfa.go:1803` / `:1769` |
| OTP send throttle | `checkAndRecordOTPSend`, `service_mfa.go:80` |

### Storage / migrations

Tables created under `internal/platform/database/migration/`:

| Table | Migration | Notes |
|---|---|---|
| `user_mfa_totp_secrets` | `038_*.go` | Secret encrypted at rest; unique per user; `trg_sync_totp_flag` keeps `users.is_totp_enabled` in sync |
| `user_mfa_webauthn_credentials` | `039_*.go` | Public key, sign count, transports, backup flags; `sync_webauthn_flag` trigger |
| `webauthn_challenges` | `040_*.go` | Single-use, server-issued, RP-ID-scoped, 5-min TTL |
| `user_mfa_phones` | `041_*.go` | SMS MFA phone (separate from `users.phone`), `is_verified` |
| `user_mfa_emails` | `042_*.go` | Email OTP MFA address, `is_verified` |
| `user_mfa_backup_codes` | `037_*.go` | bcrypt-hashed, single-use (`used`/`used_at`) |
| `user_otps` | `033_*.go` | Transient OTP store (channel `sms`\|`email`), hash + TTL + failed-attempt counter |
| `user_trusted_devices` | `046_*.go` | Owned by `internal/user`; mfa reads/writes via a local struct; soft-delete on revoke |

## Configuration

### Environment

| Var | Effect |
|---|---|
| `WEBAUTHN_RP_ID` | Overrides the WebAuthn RP ID. When unset, the RP ID is derived from `AppPublicHostname` (`rpIDFromHostname` strips scheme/port). Origin validation is **per-request**: any origin whose host is the RP ID or a subdomain of it is accepted (covers all tenant subdomains). `WEBAUTHN_EXTRA_ORIGINS` no longer exists — origin allow-listing is dynamic, not config. |
| `AppPublicHostname` | Source of both the WebAuthn RP ID (fallback) and the step-up access-token **issuer** (`config.AppPublicHostname`, `service_mfa.go:1559`). Step-up tokens are RS256, signed by the `jwt` package. |

### Per-tenant policy — `security_settings.mfa_config` (JSONB)

Read by `secpolicy.LoadMFAPolicy` and validated on write in `internal/secpolicy/validation_setting.go`. Defaults come from the seeded `mfa` config (`DefaultSecuritySettingConfig("mfa")`); a lookup failure logs and **downgrades to `mode=optional`** so a settings outage never locks users out.

| Key | Type | Effect | Enforced by mfa pkg |
|---|---|---|:---:|
| `mode` | `disabled`\|`optional`\|`enforced` | `disabled` → no method allowed; `enforced` → MFA required at login | yes / login |
| `allowed_methods` | `[]string` | Whitelist (`totp`, `webauthn`, `sms`, `email_otp`, `backup_code`). Empty = all allowed | yes |
| `totp_issuer` | string | Issuer shown in authenticator apps (default `maintainerd-auth`) | yes |
| `totp_digits` | `6`\|`8` | Code length; surfaced on enroll + status so clients size input | yes |
| `totp_period_seconds` | `30`–`90` | TOTP rotation window | yes |
| `allow_sms` | bool | Extra gate on the `sms` method | yes |
| `allow_email_otp` | bool | Extra gate on the `email_otp` method | yes |
| `preferred_method` | string | Ordered first in step-up `allowed_methods` | yes |
| `recovery_codes_count` | `8`–`16` | Backup codes generated (default 10) | yes |
| `require_mfa_for_sensitive_actions` | bool | Drives `RequirePolicyStepUp` on sensitive flows | yes |
| `step_up_ttl_minutes` | int | Step-up freshness window (default 5 min) | yes |
| `trusted_device_period_days` | int | Days a "trust this device" grant lasts | via `IssueTrustedDevice` caller |
| `grace_period_days` / `admin_grace_period_days` | int | Enrollment grace window under `enforced` | login flow (not this pkg) |

## Security considerations

- **Fail-closed gates.** `stepUpMethodAllowed`, `RequireStepUpForNewFactor`, and `RequirePolicyStepUp` all fail closed (deny / 503) on missing, malformed, or unreadable state — a challenge with no `allowed_methods` authorizes nothing; an unreadable enrollment state does not count as "unprotected." `UserHasMFA` counts every factor (TOTP, WebAuthn, verified SMS, verified email OTP) and returns an error rather than `false` on a DB miss.
- **Step-up ≠ account ownership.** Destructive and additive factor changes require the **caller** to have freshly stepped up (`acr=2` within the TTL), not merely that the account owns some factor — closing single-cookie MFA-defeat and attacker-enrollment paths.
- **Replay & clone resistance.** TOTP codes are single-use within their window (`last_used_step`); WebAuthn rejects sign-count regression; OTP hashes are compared in constant time and marked used on success; backup codes are bcrypt-hashed and single-use.
- **Secrets at rest.** TOTP seeds are envelope-encrypted (`crypto.EncryptAtRest`, same KEK path as signing keys); SMS/email OTPs, backup codes, and trusted-device tokens are stored only as hashes; the trusted-device secret is returned once.
- **Tenant isolation.** WebAuthn origins are constrained to the RP ID and its subdomains; trusted-device validation is scoped to the issuing tenant; admin reset refuses a target outside the actor's tenant.
- **Abuse throttling.** Outbound OTP sends are throttled per `(channel, user, recipient)` and **recorded before delivery** so a provider error still costs a slot; verification attempts are rate-limited per factor (default 5 attempts / 15-min window via `security.CheckRateLimit`, fail-open only if Redis is unconfigured). Enrollment SMS is throttled too, since the destination is attacker-chosen.
- **Delivery is SMTP-only for email** (`email.SendEmail`); SMS goes through the tenant's `sms_config` provider (`twilio`, `sns`, `vonage`, or a `log` no-op), falling back to the system tenant's config, and is a no-op (logged, code redacted) when none is configured.
- **Backup-code hygiene.** `GetBackupCodesCount` counts only redeemable (bcrypt) rows, so a stale SHA-256 row from a retired code path is never advertised as usable. `SyncMFAState` purges leftover backup codes once no primary factor remains; `first_mfa_enrolled_at` is set-once and never cleared.

## Related

- [Authentication & login](./authentication.md) — password login and the MFA login second step (`acr` elevation).
- [Sessions & tokens](./sessions.md) — `acr`/`amr` claims, RS256 access tokens, refresh rotation.
- [Security settings](./security-settings.md) — the `security_settings` store and per-tenant policy surface.
- [Notifications](./email-and-sms.md) — SMS/email OTP delivery (`sms_config`, SMTP email).
