# Registration & Invites

> How maintainerd-auth provisions new user accounts — public self-service registration and admin-issued email invites — and the registration-flow mechanism that pre-assigns roles on either path.

| | |
|---|---|
| **Status** | Implemented |
| **Code** | `internal/authn` (registration + role assignment), `internal/invite` (invite tokens, resend/revoke), `internal/secpolicy` (registration policy) |
| **Endpoints** | Public (:8081): `POST /register`, `POST /register/invite`, `GET /registration_context`, `GET /invite`. Admin (:8080, VPN): `GET/POST /invite`, `GET /invite/{uuid}`, `POST /invite/{uuid}/resend`, `DELETE /invite/{uuid}` |
| **Storage** | `invites`, `registration_flows`, `registration_flow_roles` tables; `users`, `user_identities`, `user_roles`, `user_password_history` written on redeem; `security_settings.registration_config` (JSONB) holds the tenant policy |
| **Config** | `INVITE_TTL_HOURS`, `CAPTCHA_SECRET`, `AppPrivateHostname`; per-tenant `registration_config` keys (see [Configuration](#configuration)) |

## Overview

Two ways a new account comes into being, both ending in the same "user + identity + roles + session + token set" outcome:

1. **Self-service registration** — an external app links a user to the hosted signup form; the user posts `username`/`password` (+ optional `email`/`phone`/`fullname`) to `POST /register`. Gated by the tenant's registration policy and, optionally, a named **registration flow** that adds required fields and role grants.
2. **Invite registration** — an admin (holding `user:invite` + a stepped-up session) issues an email invite bound to a specific address and, optionally, a registration flow. The recipient redeems the signed link via `POST /register/invite`. The invited email is treated as proven, so the account is created active and email-verified.

Both live on the **public data plane (:8081)**; admin invite management lives on the **internal control plane (:8080, VPN-only)** (`internal/server/router.go:89`, `:251`, `:270-271`). All public routes require `client_id` and reject `tenant_id` (`handler_login.go:327` `authenticationContextQuery`); the tenant is derived from the resolved client, never trusted from the request.

A **registration flow** (`registration_flows`) is a named, tenant+client-scoped policy object that can (a) demand extra fields, (b) force email verification, and (c) pre-assign roles to the new user. Flow CRUD is owned by `internal/idp`; registration only reads the selection, policy, and role fields (`internal/authn/deps.go:427-441`).

## How it works

### Self-service registration — `POST /register` (`handler_register.go:25`, `service_register.go:387`)

1. **Handler guards** (`handler_register.go:25-105`): require `client_id`, reject `tenant_id`; validate User-Agent (suspicious → 400 + `suspicious_user_agent` event); decode + validate/sanitize the body (`ValidateForRegistration`); the CAPTCHA token is stashed on the context.
2. **Client + tenant resolution** (`service_register.go:428-441`): `resolvePublicClient` resolves the public client from `client_id`; the client must be active with a domain, and its tenant must resolve (non-zero).
3. **Policy gate** (`:444-450`): `secpolicy.LoadRegistrationPolicy` loads the tenant's effective policy. `self_registration_enabled=false` → 403; the client's own `AllowRegistration=false` → 403. Per-IdP gate: an in-house identity provider that is inactive or has `AllowRegistration=false` → 403 (`enforceIdentityProviderRegistrationGate`, `:222`).
4. **Flow resolution** (`:454-461`): if `registration_flow=<name>` is present, `registrationFlowByName` resolves it scoped to **client AND tenant**. System flows, inactive flows, and wrong-tenant/unknown flows all collapse to a single not-found (no enumeration; `:244-279`). The flow's `required_fields` are then enforced (`enforceRequiredRegistrationFields`, `:312`), and `effectiveRegistrationPolicy` ORs the flow's `verification_required` onto the tenant policy (`:343`).
5. **Abuse controls** (`enforceRegistrationAbuseControls`, `:183`): per-username rate limit (`security.CheckRateLimit`, `:408`); CAPTCHA when `captcha_on_signup` is set **and** a provider is configured (deferred/fail-open with a per-tenant warning when no `CAPTCHA_SECRET`, `:187-212`); registration rate limit per IP per hour (`:213`).
6. **Domain + PII checks** (`:469-509`): email domain must pass the allow/block lists (`EmailDomainAllowed`); if `require_phone_verification`, phone is mandatory; username conflict → explicit 409, but an existing email/phone returns a **generic** conflict (enumeration hardening H8).
7. **Password + user creation** (`:511-546`): the tenant password policy is loaded and enforced (`secValidatePasswordPolicy`), the password is hashed with that policy, and the `users` row is created. Initial status = `pending` when verification is required, an email is present, and auto-confirm is off; otherwise `active` (`registrationInitialStatus`/`InitialUserStatus`, `:875` / `registration_policy.go:95`). `IsEmailVerified` is **always false** here — activation ≠ proof of the address (OIDC Core §5.1, `:528-533`).
8. **Identity, roles, consent** (`:546-599`): records first password into `user_password_history` (tx-scoped); creates a `user_identities` row (`provider=maintainerd`, random UUID `sub`); assigns the system **`registered`** role (falling back to any `is_default` role, `findDefaultRole` `:126`); `assignRegistrationFlowRoles` adds the flow's grantable roles; records `terms_of_service` consent (best-effort).
9. **Commit, verification email, sign-in** (`:604-617`): after commit, if verification is required and an email was supplied, a verification email is sent (best-effort, non-fatal). `generateTokenResponse` creates a real session and returns the access/id/refresh token set (registration signs the user in like login).

### Invite issuance — `POST /invite` (admin, `handler_invite.go:53`, `service_invite.go:83`)

1. Requires `user:invite` permission **and** a stepped-up session (`RequireStepUp`, `routes.go:34`). Tenant + actor come from the authenticated context.
2. Resolves the initiating tenant's system identity client; generates a **32-byte CSPRNG token** (`crypto.GenerateIdentifier(32)`). Only the token's **SHA-256 digest** is persisted in `invites.invite_token`; the raw token exists only in memory to build the emailed link (`service_invite.go:122-138`, `repository_invite.go:29`).
3. If a `registration_flow_uuid` is supplied: the flow is resolved/validated (active, same tenant), bound to the invite by internal id, and the invite's client/branding switch to the flow's client. **Grant-what-you-hold check**: the inviter must already possess every role the flow would grant, else 400 (`service_invite.go:140-197`).
4. Optional `callback_url` is validated against the client's registered redirect URIs (`:199-208`).
5. Builds a **signed URL** (`expires` + `sig`) to `AppPrivateHostname + /register/invite` carrying `invite_token`, `email`, `client_id`, and optional `callback_url`, then converts it to the tenant's identity-app frontend URL (per-tenant subdomain derived from the tenant name). Default TTL **72h**, overridable by `INVITE_TTL_HOURS` (`inviteTTL`, `:41`).
6. Sends the invite email over **SMTP** (`email.SendEmail`) using the `user:invite` template (tenant-scoped, falling back to the global template) (`sendInviteEmail`, `:361`). A send failure fails the call.

### Invite redemption — `POST /register/invite` (`handler_register.go:153`, `service_register.go:624`)

1. Handler requires `client_id` (rejects `tenant_id`), validates the **signed URL** (`signedurl.ValidateSignedURL`), validates query params, and decodes `username`/`password` from the body (`LoginRequestDTO`).
2. Service resolves the client/tenant, then loads the invite by token **for update** (row lock, `FindByTokenForUpdate`). The invite must be `pending`, unexpired, and belong to the resolved client's tenant (`:665-683`).
3. The invite's registration flow (if any) is re-validated by id (`validateInviteRegistrationFlow`, `:281`). **Blocklist still applies** — an invite overrides the self-signup allowlist and `self_registration_enabled`, but not a hard-blocked domain (`EmailDomainBlocked`, `:702`).
4. Username + invited-email conflicts are checked; the tenant password policy is enforced and the password hashed.
5. The `users` row is created with `Email = invite.InvitedEmail`, `Status = active`, and **`IsEmailVerified = true`** (the invite link proves email ownership). An **email-OTP MFA** enrollment is auto-created (best-effort raw SQL upsert into `user_mfa_emails`) (`:729-749`).
6. Identity + default `registered` role + flow roles + consent are written exactly as on the self-service path; then the invite is marked used (`MarkAsUsed` flips `pending → accepted`, sets `used_at`, atomically via a status predicate). Commit → session + token set.

### Registration context — `GET /registration_context` (`handler_registration_context.go:31`, `service_registration_context.go:73`)

Public, unauthenticated read that tells a hosted signup form what to collect: the **effective `required_fields`** (flow fields merged with policy-derived `email`/`phone`) and **`verification_required`**. It resolves the flow through the same guard set as `/register` and returns `Cache-Control: no-store`. It deliberately **withholds** the flow's roles, description, status, `is_system`, ids, and timestamps — the flow name is guessable, so publishing role grants would hand an attacker a ranked target list.

### Invite context — `GET /invite` (public, `handler_invite.go:207`)

Given a raw `invite_token`, returns the invited email, callback URL, expiry, and status for the identity app to pre-fill the redemption form. 410 Gone if the invite is not `pending` or has expired. It echoes back the **raw** token the caller supplied (the stored value is a digest and cannot be redeemed).

### Invite lifecycle

| From | Action | To | Notes |
|------|--------|-----|-------|
| `pending` | redeemed | `accepted` | `MarkAsUsed`, atomic status predicate; sets `used_at` |
| `pending` | admin revoke | `revoked` | `RevokeByUUID`, only from `pending` (`service_invite.go:430`) |
| `pending` / `expired` | admin resend | `pending` | new token + expiry, clears `used_at`; **only** resendable states (`resendableStatuses`, `repository_invite.go:53`) |
| `pending` (past `expires_at`) | — | rejected at redeem | expiry is checked at read time; there is no sweeper that flips `pending → expired` |
| any (30 days past `expires_at`, or `expired`) | cleanup runner | **deleted** | `DeleteExpired`, batched, called by `internal/oauth/cleanup_runner.go:76` |

Resend and revoke are **idempotent under races**: the status predicate lives in the SQL `WHERE`, so a concurrent acceptance/revoke loses rather than being silently overwritten (`ResetForResend`/`RevokeByUUID`).

## Implementation

### Packages & key files

| Concern | File:line |
|---------|-----------|
| Register handlers (public + invite) | `internal/authn/handler_register.go:25`, `:153` |
| Register service (`RegisterPublic`, `RegisterInvitePublic`) | `internal/authn/service_register.go:387`, `:624` |
| Flow resolution / role assignment | `service_register.go:244` (`registrationFlowByName`), `:351` (`assignRegistrationFlowRoles`), `:312` (`enforceRequiredRegistrationFields`) |
| Abuse controls (captcha, rate limit) | `service_register.go:183` (`enforceRegistrationAbuseControls`) |
| Registration-context service | `internal/authn/service_registration_context.go:73` |
| Public register routes | `internal/authn/routes.go:128` (`RegisterPublicRoute`), `:181` (`RegistrationContextPublicRoute`) |
| Register DTOs + validation | `internal/authn/types.go:124`, `internal/authn/validation_register.go` |
| Invite service (send/resend/revoke/list/get) | `internal/invite/service_invite.go:83` |
| Invite repository (token hashing, atomic writes) | `internal/invite/repository_invite.go:29` (`hashInviteToken`), `:173`–`:230` |
| Invite model | `internal/invite/model_invite.go:10` |
| Invite routes (admin + public) | `internal/invite/routes.go:12`, `:52` |
| Registration policy (allow/block, initial status) | `internal/secpolicy/registration_policy.go` |
| Flow role grant cap | `internal/authn/deps.go:408` (`RegistrationFlowRoleRepository`) |
| Expired-invite cleanup | `internal/oauth/cleanup_runner.go:76` |

### DB tables / migrations

- **`invites`** — migration `053_create_invites_table.go`. Columns: `invite_uuid`, `tenant_id`, `client_id`, `registration_flow_id` (FK→`registration_flows`, `ON DELETE SET NULL`), `invited_by_user_id`, `invited_email`, `invite_token` (`UNIQUE`; **holds the SHA-256 digest**, historical column name), `callback_url`, `status` (`CHECK IN ('pending','accepted','expired','revoked')`), `expires_at`, `used_at`, soft-delete `deleted_at`.
- **`registration_flows`** — migration `051`. Columns: `tenant_id`, `client_id`, `name` (slug-shaped, tenant-unique; the `?registration_flow=` selector), `required_fields` (JSONB array), `verification_required`, `is_system`, `status` (`active`/`inactive`).
- **`registration_flow_roles`** — migration `052`. Join table of the roles a flow grants.
- On redeem: `users`, `user_identities`, `user_roles`, `user_password_history`, and (invite path) `user_mfa_emails`.

### Role-grant authority

`FindGrantableRoleIDsByRegistrationFlowID` is the **authoritative redeem-time cap** (`deps.go:415-424`): it returns only roles that are same-tenant, active, non-system, not soft-deleted, and carry **no management-plane permission**. This is a time-of-**use** test — a role that later gains an admin permission stops being granted — layered over the time-of-check `assertRolesGrantable` an admin hits when attaching a role. `assignRegistrationFlowRoles` additionally skips the already-assigned default role and de-dupes (`service_register.go:351`).

## Configuration

### Environment variables

| Var | Default | Effect |
|-----|---------|--------|
| `INVITE_TTL_HOURS` | `72` | Invite link/token lifetime (`service_invite.go:41`) |
| `CAPTCHA_SECRET` | unset | Presence enables signup CAPTCHA enforcement; unset ⇒ `captcha_on_signup` is warned-but-not-enforced (`service_register.go:162`) |
| `AppPrivateHostname` | — | Base host for the signed `/register/invite` API URL (`service_invite.go:242`) |

### Per-tenant `registration_config` (JSONB in `security_settings`, via `secpolicy.LoadRegistrationPolicy`)

| Key | Default | Meaning |
|-----|---------|---------|
| `self_registration_enabled` | `true` | Allow `POST /register`; `false` = invite-only |
| `require_email_verification` | `true` | Account stays `pending` until email is verified (unless auto-confirm) |
| `require_phone_verification` | `false` | Phone is mandatory at registration |
| `allowed_email_domains` | `[]` | If non-empty, email domain must match (exact or `*.domain`); **self-signup only** |
| `blocked_email_domains` | `[]` | Domain hard-blocked on **every** provisioning path incl. invite; matches subdomains |
| `auto_confirm_enabled` | `false` | Activate immediately without waiting on verification (does **not** set `email_verified`) |
| `verification_token_ttl_hours` | `24` | Verification-token lifetime (floored to 24 if ≤0) |
| `captcha_on_signup` | `true` | Require CAPTCHA on signup (only enforced when a provider is configured) |
| `registration_rate_limit_per_ip_per_hour` | `10` | Per-IP signup ceiling (floored to 10 if ≤0) |

Per-flow settings (owned by `internal/idp`): `required_fields`, `verification_required`, `is_system`, `status`, plus the granted roles in `registration_flow_roles`.

## Security considerations

- **Public-surface contract**: every public route requires `client_id` and rejects `tenant_id`; the tenant is always derived from the resolved client, never from the request (`authenticationContextQuery`).
- **Invite tokens are digest-at-rest bearer credentials**: 32 bytes of CSPRNG, stored only as a SHA-256 digest (`base64url`), delivered only inside a **signed URL** (`expires`+`sig`). A DB read (backup, replica, SQLi) yields no redeemable token. `json:"-"` keeps the digest out of audit snapshots. Empty tokens fail closed in the repository.
- **Step-up + permission on issuance**: sending/resending an invite (which can grant roles) needs `user:invite` **and** an `acr=2` session.
- **Grant-what-you-hold / grant cap**: an inviter cannot bind a flow granting roles they lack; and the redeem-time cap strips cross-tenant, system, soft-deleted, and management-permission roles regardless of what the flow lists.
- **System flows are invite-only**: a self-service link can never redeem an `is_system` flow (e.g. owner onboarding / super-admin) — reported as not-found so existence isn't confirmed.
- **Enumeration hardening**: unknown/inactive/wrong-tenant/system flows all return one not-found; existing email/phone at signup returns a generic conflict; `registration_context` withholds roles and is `no-store`.
- **Email verification honesty**: `IsEmailVerified` reflects *proven* control only — false on self-service (even with auto-confirm), true only on the invite path where the link proves the address. Account activation is a separate axis (`InitialUserStatus`).
- **Blocklist supremacy**: `blocked_email_domains` is enforced on self-signup, invite, and (per `EmailDomainBlocked`'s contract) federated JIT provisioning; bare entries also block subdomains so the rule can't be dodged via `mail.blocked.com`.
- **Atomic state transitions**: redeem/resend/revoke embed the status predicate in SQL and use `SELECT … FOR UPDATE` on redeem, so concurrent acceptance/revoke/resend races resolve deterministically rather than corrupting `used_at`/status.
- **Password policy is tenant-authoritative**: the DTO layer deliberately does **not** apply a hardcoded strength policy; the service loads and enforces the tenant's real policy (the single authoritative check), and records password history in-transaction.
- **Registration signs the user in with a real session** — newly registered users are subject to idle timeout, absolute lifetime, concurrent-session limits, and logout revocation, exactly like login.

## Related

- [Email Verification](./authentication.md) — the post-registration proof-of-address flow that clears the `pending` status.
- [Authentication & Login](./authentication.md) — the session + token-set path registration reuses to sign the user in.
- [Sessions](./sessions.md) — session lifecycle, idle/absolute limits, and concurrent-session enforcement applied to registered users.
- [Password Policy](./security-settings.md) — the tenant-authoritative password rules enforced at registration.
- [Registration Flows](./registration-and-invites.md) — flow CRUD and role attachment (owned by `internal/idp`).
