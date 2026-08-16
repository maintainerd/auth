# Changelog

All notable changes to maintainerd-auth will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] — pre-release (moving until public launch)

> A single moving pre-release during testing — the `v0.1.0` tag is re-pointed and
> Docker Hub's `0.1.0` + `latest` are rebuilt on each update, until it is locked as
> the public v0.1.0 at launch. Newest changes first.

### Supply chain / hardening (OpenSSF Scorecard)
- **Signed releases** — release artifacts are now signed with **cosign keyless**
  (Sigstore/Fulcio/Rekor): a registry signature over the image, plus a detached
  signature + certificate over the SBOM attached to the GitHub Release.
- **Fuzzing** — added Go native fuzz tests over the untrusted-input parsers (tenant-host
  resolver, email/phone validators, WebAuthn RP-ID derivation, log sanitizer).
- **Branch protection** on `main` (force-push + deletion blocked, linear history).

### Security — logging
- Sanitize the refresh-token `jti` (CR/LF stripped) before it is logged on rotation
  error paths, closing a log-injection sink (CWE-117 / CodeQL `go/log-injection`): the
  jti can originate from an unverified token in the reuse-detection path.

### Fixed — frequent logout from refresh-token rotation
- **A benign concurrent refresh no longer signs the user out.** Refresh-token reuse
  detection revoked the entire session family on *any* duplicate refresh, and the
  `refresh_token_reuse_interval_seconds` "grace window" only changed the error text —
  the family was revoked regardless. So two tabs, a retry, or parallel requests racing
  on the shared cookie tore down the whole session, causing repeated logouts during
  active use.
- Implemented **idempotent rotation** (OAuth 2.0 Security BCP, RFC 9700 §4.14.2): the
  first request to consume a refresh token caches the exact token set it minted for the
  reuse-interval window; an in-window duplicate of that token now gets the **same** set
  back (a normal 200) instead of a family-revoking 401. Because the same set is returned
  rather than an independent one, an in-window replay of a stolen token cannot fork a
  separate session, and reuse **outside** the window still revokes the family
  (RFC 6819 §5.2.1.1). The overlap window is driven entirely by the existing
  `refresh_token_reuse_interval_seconds` security setting (0 = strict single-use).
- **No change to token lifetimes or session limits.** Access-token TTL, refresh-token
  TTL, idle timeout, absolute timeout, and the rotation flag remain sourced from the
  security settings (global + per-tenant + per-client) and are untouched.

### Fixed — WebAuthn on tenant subdomains
- **Passkeys now work for every regular tenant.** WebAuthn origin validation used a
  static `RPOrigins` list (the system surfaces only), so registering/authenticating a
  passkey from a tenant subdomain (`{tenant}.console.auth…` / `{tenant}.identity.auth…`)
  failed with "Error validating origin". go-webauthn matches origins exactly (no
  wildcards), and tenant origins are open-ended, so a static list can't cover them.
- The RP ID stays constant (a registrable suffix such as `auth.maintainerd.dev` that
  covers every surface and tenant). At the Finish step the verifier is now built
  per-request from the ceremony's own (signed) origin, and that origin is accepted
  **only** when its host is the RP ID or a subdomain of it — so only pages served under
  our own domain are ever honored, and suffix-confusion lookalikes are refused. The
  origin is cryptographically bound to the ceremony and the challenge is single-use, so
  this is the standard spec-compliant pattern for a multi-tenant subdomain RP, not a
  relaxation of validation.

### Changed — email delivery
- **Email delivery is now SMTP-only.** Removed the API-based providers (Amazon SES,
  SendGrid, Mailgun, Postmark, Resend). Their credentials were never persisted — the
  `email_config` table has no `api_key`/`domain`/`region` columns and the config
  service never accepted them — so those providers could never actually send. Every
  provider is now reached through its SMTP relay instead, which is the standard
  self-hosted IAM approach (e.g. Keycloak is SMTP-only). The email config form,
  request validation, and the `email_config` provider CHECK constraint now accept
  `smtp` only. Existing non-smtp rows should be reconfigured as SMTP.

### Security — clients
- **Closed an unauthenticated token-minting path.** `token_endpoint_auth_method`
  `none` means "presents no credential" and is now refused for anything other than
  a public (spa/mobile) client, at both the token endpoint and the registry.
  Previously a confidential or m2m client could be configured with `none`, and
  since `client_id` is public, anyone could mint that client's tokens and receive
  its resolved permissions. The existing client_credentials tests encoded the
  vulnerable behaviour and were corrected to a confidential client with a secret.
- Added a cross-field validity matrix for clients (`ValidateClientOAuthMatrix`):
  `client_credentials` requires client authentication and is refused for public
  clients, `authorization_code` is refused for m2m, secret-based auth methods are
  refused for public clients and require a secret to exist, and the mTLS methods
  are refused until certificate binding is implemented.
- Public clients (spa/mobile) are no longer issued a client secret, and default to
  `none` rather than inheriting the column default `client_secret_basic`.
- **`is_system` clients are now immutable.** Update, delete and secret rotation
  previously guarded only `is_default`; the seeded `auth-identity` client is
  `is_system` without being `is_default`, so it could be renamed, deactivated or
  deleted — which breaks the tenant's hosted login UI, resolved by name.
- **The default identity-provider invariant is now enforced, not just seeded.**
  The built-in provider connection cannot be disabled or demoted, no connection
  change may leave a client with zero enabled providers, and the built-in check
  fails CLOSED when the provider cannot be resolved (it previously skipped the
  guard whenever the relation was not preloaded). Losing that connection silently
  disabled password sign-in and self-registration.
- Dynamic Client Registration (`POST /oauth/register`) is disabled and removed from
  the discovery document. It was unauthenticated, resolved every registration into
  the system tenant, and accepted arbitrary grant types with
  `token_endpoint_auth_method=none`. It never worked (it wrote a `client_type` that
  violates the CHECK constraint), so nothing regresses.
- Secret rotation now rejects a grace period over 168 hours; it was unbounded, so a
  compromised secret could be kept valid indefinitely.

### Fixed — clients
- `RequireConsent`, `AllowRegistration` and `ClientIdentityProvider.Enabled` are
  now pointers. As non-pointer bools with a `default:` tag, GORM omitted the zero
  value on INSERT and the DB default won: an explicit `false` did not persist. The
  seeded console and identity clients therefore got `require_consent = TRUE` on a
  fresh install and `false` on re-seed, and a client created with
  `allow_registration: false` had self-registration silently enabled.
- The client seeder's re-provision path wrote `require_pkce = NULL` into a NOT NULL
  column (it saved a literal struct, so `Save` wrote every field including nils),
  aborting re-seeding; it now updates the loaded row.
- 11 mutating client endpoints dereferenced a possibly-nil user before checking it,
  panicking when no user was in context. They now return 401.
- Deleting a client left its children live: `clients` is soft-deleted, so the FK
  cascade never fired. URIs and identity-provider connections stayed active, and
  APIs, permissions and roles — which have no `deleted_at` — became permanent
  orphans that still resolved permissions for the deleted client.
- `?sort_by=` on clients accepted any column from the global allowlist, so sorting
  by e.g. `email` produced an undefined-column 500. Clients now has its own
  allowlist (`SanitizeOrderInPrefixed`).
- The connection-update DTO now carries omitted-means-unchanged semantics. Its
  fields were non-pointers, so the console's partial toggles silently rewrote what
  they did not mention — changing `enabled` alone cleared `is_default`, demoting
  the client's default identity provider.
- `client_type` is now required on create and update; an empty string passed the
  enum check and failed at the DB CHECK as a 500.
- `domain` is validated as a hostname or https URL and allowed the column's full
  253 characters. It becomes the token `iss` and is compared in the
  `private_key_jwt` audience check, so free text there is load-bearing.
- The client `config` blob is capped at 16KB; it was unbounded.

### Changed — clients
- `ClientResponseDTO` now exposes `identifier` — the actual OAuth `client_id` an
  application presents. It was absent, so an operator could not read the value
  their app needs. Note `client_id` in the JSON remains the management UUID.
- `ClientResponseDTO` now exposes the OAuth metadata the runtime enforces
  (`token_endpoint_auth_method`, `grant_types`, `response_types`, `allowed_scopes`,
  `require_consent`, `access_token_ttl`, `refresh_token_ttl`) so consumers stop
  reading the free-form config blob, which can hold values the server rejected.

### Added
- `GET /registration_context` (public, port 8081) returns what a signup form must
  collect for a client and optional registration flow: the effective
  `required_fields` and `verification_required`. Without it a flow requiring full
  name or phone was an unresolvable 400 — the hosted form never asked. It exposes
  nothing else; in particular it never reveals which roles a flow grants, since
  the flow name is guessable by design. Every resolution failure returns one
  identical 404. Deliberately NOT added to `/oauth/connections`: flow-derived
  fields were removed from that response so a flow can never change the login page.
- The hosted signup form now collects full name and phone when the resolved flow
  requires them, and shows a dedicated screen when a sign-up link is no longer
  valid instead of failing at submit.
- `TRUSTED_PROXY_CIDRS` / `TRUST_ALL_PROXIES` configure which peers may speak for
  the client via forwarding headers. Defaults to loopback + private ranges.

### Security
- **A public registration flow can no longer grant administrative access.** A
  non-invite flow may only attach roles whose every permission is `public:*` or
  `account:*:self`; anything on the management plane is refused with the offending
  permission named. Enforced when an admin attaches a role AND again at redemption,
  so a role that later gains an administrative permission stops being granted.
  This is the control that makes a readable, guessable flow name safe.
- Client IP is no longer taken from caller-supplied `X-Forwarded-For` unless the
  peer is a trusted proxy, and the list is walked right-to-left. Previously any
  caller could rotate one header per request and reset every per-IP rate limit,
  registration abuse counter and IP restriction app-wide.
- Redeem-time role grants are filtered in SQL to same-tenant, active, non-system
  roles. Cross-tenant isolation on the unauthenticated grant path was previously
  assumed from write-time checks rather than enforced at read time.
- `/register` no longer distinguishes an inactive registration flow from an unknown
  one. Status is the operator's kill switch for a published link, so a
  distinguishable response let whoever held a leaked link poll until it was
  re-enabled — and confirmed which flow names exist.
- Inactive roles can no longer be attached to a registration flow.
- Registration flows can no longer grant privileges the acting admin does not
  hold. Attaching a role to a flow now requires that the actor already possesses
  that role, and system roles can never be attached — closing a path where
  `registration-flow:update` conferred more power than `user:invite`.
- System registration flows (e.g. the seeded owner-onboarding flow, which grants
  `super-admin`) are now invite-only: they can no longer be redeemed from a
  self-service registration link, on either `/register` or
  `/oauth/authorize?screen_hint=signup`.
- Public registration-flow resolution is now scoped by tenant as well as client.
  A client match alone proved existence, not ownership.
- System registration flows are immutable through the admin API. Previously only
  delete was protected, so a system flow could be renamed, re-activated, or
  re-pointed at other roles.
- All registration-flow mutations now validate the acting user's tenant access at
  the service layer, and reject requests with no authenticated user (401) instead
  of proceeding with an unattributed audit entry.
- `/oauth/authorize` no longer distinguishes "unknown" from "inactive"
  registration flows, removing an unauthenticated existence oracle.

### Fixed
- Public registration silently ignored the `registration_flow` query parameter,
  so flow status, `required_fields` and automatic role assignment never applied
  on the self-service path. The parameter is now honored.
- A partial `PUT` on a registration flow no longer silently re-activates a
  disabled flow, disables email verification, or wipes `required_fields`.
  Omitted fields are left unchanged.
- Registration-flow name uniqueness is checked against the tenant, matching the
  `uq_registration_flows_tenant_name` index — a collision returns 409 instead of
  a driver-level 500.
- Deleting a registration flow now clears its role membership. The flow is
  soft-deleted, which does not fire the foreign-key cascade, so those rows
  previously outlived the parent.
- Invalid role UUIDs in a registration-flow request are rejected instead of
  silently dropped, which had produced partial role assignments.
- A unique-index violation now returns 409 instead of 500 across every table:
  GORM error translation is enabled and `ErrDuplicatedKey` is mapped at the
  transport layer. A service pre-check narrows the race but cannot close it.
- `?sort_by=identifier` on registration flows returned 500 after the column was
  removed. Sort allowlists can now be scoped per resource
  (`database.SanitizeOrderIn`) rather than sharing one union across every table.
- The identity app discarded every registration error message and showed a generic
  string, because the rejection is a plain object rather than an `Error`. Backend
  messages and status codes now reach the form.
- The hosted signup page's "Sign in" link forwarded `screen_hint=signup`, which
  bounced the user straight back to signup — an inescapable loop.
- `/oauth/connections` documented and accepted a `registration_flow` parameter it
  deliberately ignores; removed from the contract and from both callers.

### Changed
- **BREAKING** — a registration flow no longer has a separate `identifier`. The
  flow `name` is now the public selector in a registration link
  (`?registration_flow=<name>`), so it is validated as a slug (lowercase
  alphanumerics with single hyphens/underscores, e.g. `partner-signup`) and is
  unique per tenant. The `identifier` column, request field and response field
  are all gone. Note the consequence: renaming a flow changes its link, so any
  link an external app has already published stops resolving.
- The seeded owner-onboarding flow is renamed `system-onboarding-owner` (was
  `system:onboarding:owner`) to satisfy the slug rule.
- Registration-flow responses split into a lean list shape and a detail shape;
  the detail shape resolves the bound client to a name instead of a bare UUID.
- Registration-flow listing gained free-text `search` (name + description) and an
  `is_system` filter, and `status` now accepts multiple comma-separated values.
- Added a unique index on `(tenant_id, name)` for registration flows.

### Baseline — 2026-07-03

### Added
- Initial public release of Maintainerd Auth
- Full OAuth2 / OIDC provider (authorize, token, PKCE, refresh, revocation, introspection, device, CIBA, PAR, token-exchange)
- Multi-tenant architecture with strict tenant isolation
- IAM: services, APIs, roles, permissions, policies, service-to-service authorization
- Multi-factor authentication: TOTP, SMS, email OTP, WebAuthn/passkeys, backup codes
- Self-service MFA enrollment UI in hosted identity app
- Identity federation with OIDC upstream providers + JIT provisioning
- SMS passwordless login
- Client ↔ Identity Provider connection management
- Registration flows with customizable branding and callback URLs
- Invite-based registration with signed tokens
- Webhook endpoints with delivery history and replay
- Audit event logging (OWASP-compliant) with CSV/JSON export
- Branding: themes, logo upload, login layout customization
- Admin console: users, clients, providers, roles, permissions, webhooks, auth events
- Hosted identity UI: login, registration, MFA, OAuth consent, device/CIBA flows
- Docker images: multi-arch (amd64/arm64), non-root, HEALTHCHECK
- OpenTelemetry tracing and Prometheus metrics
- gRPC API alongside REST

### Security
- argon2id password hashing with per-user salt
- JWT algorithm pinning + JWKS key rotation
- CSRF protection on cookie-authenticated endpoints
- Account enumeration resistance
- Brute-force lockout and rate limiting
- CORS + secure cookie flags + HSTS guidance
- Signed webhook payloads with HMAC
- Container vulnerability scanning + SBOM in release pipeline
- DPoP-bound access/refresh tokens (RFC 9449)
- private_key_jwt + client_secret_jwt client auth (RFC 7523)
- Generic OAuth2 upstream connector for federation
- Recovery middleware with structured logging + stack traces
- Canonical Config struct with GetConfig() accessor
- model.Base embeddable struct for soft-delete + audit timestamps
- Redis TLS support (REDIS_TLS env or rediss:// scheme)
- SMS OTP rate limiting (per-phone + brute-force protection)
- AMR/ACR claims in ID tokens
- Build-info gauge with commit SHA + build date
- Consent revoke audit event
- PII redaction for Description + ErrorReason in audit payloads
- `/healthz`, `/readyz`, `/livez` endpoints with dependency status
- `.env.example` with all documented environment variables
- Gitleaks pre-commit hook
- CI: gosec security audit step
- GitLab provider constant

### Fixed
- Logout now revokes server-side sessions (was cookie-only)
- Per-client access token TTL now reflected in JWT exp claim
- Grant type enforcement for authorization_code and refresh_token in token exchange
- `offline_access` scope now required for refresh token issuance
- ID tokens now include Name, FirstName, LastName, Picture from user profile
- Scope-to-claim mapping + custom claim mappers now wired
- CSP: removed `'unsafe-inline'` from style-src
- HMAC_SECRET_KEY validated at boot (fail-fast)

### Changed
- Dockerfile: distroless static base image (UID 65532 as m9d user)
- README: all references updated from maintainerd/auth to maintainerd/maintainerd-auth
- CI test timeout increased to 600s for race detector

## [1.0.0] — 2026-06

Initial production release.

### Core Authentication
- Username/email + password login (internal + public)
- User registration with configurable signup flows
- Forgot/reset password with email delivery
- Email verification (6-digit OTP, 1h TTL)
- Magic link / passwordless login (15min, single-use)
- Invite flow with role pre-assignment
- Bootstrap/setup flow
- Force password change, account deletion (GDPR), data export (GDPR)
- SMS one-time code login, account recovery via backup codes

### OAuth 2.0 / OIDC
- Full OAuth 2.0 endpoints: token, authorize, revoke, introspect, userinfo
- OIDC Discovery + RFC 8414 Authorization Server Metadata
- Grants: authorization_code (PKCE), refresh_token (rotation), client_credentials, device_code, token-exchange
- CIBA (poll mode), Pushed Authorization Requests (PAR), Dynamic Client Registration
- Back-Channel Logout, RP-Initiated Logout

### Multi-Factor Authentication
- TOTP (RFC 6238), WebAuthn/FIDO2 passkeys
- Backup codes, step-up authentication
- Per-tenant MFA policy, admin reset flow

### Federation
- OIDC upstream connectors (Google, Microsoft, Apple, GitHub, GitLab)
- JIT provisioning, attribute mapping, home-realm discovery
- Identity linking/unlinking

### Tenancy & RBAC
- Multi-tenant isolation, tenant members, services, settings
- Roles, permissions, policies with middleware enforcement

### Security
- Bcrypt password hashing (cost ≥ 12), constant-time comparison
- Rate limiting on all auth endpoints, account lockout
- CORS, security headers, CSRF, HSTS, CSP
- Cookie security (Secure, SameSite, __Host- prefix)
- URI/redirect validation, body size limits, Content-Type enforcement
- Secret management: env, file, AWS, GCP, Azure, Vault

### Operations
- Structured JSON logging (slog) with PII redaction
- OpenTelemetry tracing + Prometheus metrics
- Audit logging (append-only, immutable, per-tenant)
- Pluggable email (SMTP, SES, SendGrid, Postmark, Mailgun, Resend)
- Pluggable SMS (Twilio, SNS, Vonage)
- Webhooks with HMAC-SHA256 + replay protection
- Graceful shutdown, health/readiness probes
