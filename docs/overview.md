# Maintainerd Auth — System Overview

This document is the conceptual entry point for **maintainerd-auth**. Read this before diving into the architecture, API, or contributing guides.

---

## Table of Contents

- [What it is](#what-it-is)
- [Deployment Modes](#deployment-modes)
- [Two-Port Architecture](#two-port-architecture)
- [Frontends](#frontends)
- [Data Hierarchy](#data-hierarchy)
- [Tenant](#tenant)
- [User Pools](#user-pools)
- [Identity Providers](#identity-providers)
- [Clients](#clients)
- [Roles and Permissions](#roles-and-permissions)
- [Users and Identities](#users-and-identities)
- [Services, APIs, and Permissions](#services-apis-and-permissions)
- [Policies](#policies)
- [API Keys](#api-keys)
- [Signup Flows](#signup-flows)
- [Invites](#invites)
- [Branding and Templates](#branding-and-templates)
- [Settings](#settings)
- [Tokens and JWT](#tokens-and-jwt)
- [Admin Users vs External Users](#admin-users-vs-external-users)
- [Known Limitations and Planned Work](#known-limitations-and-planned-work)

---

## What it is

**Maintainerd Auth** is a self-contained authentication and authorization service. It is designed to do what Keycloak, Auth0, AWS Cognito, and Google Identity do — issue tokens, manage users and roles, enforce permissions, and serve as the identity backbone for one or more applications.

Key characteristics:

- **One instance = one tenant.** There is no multi-tenancy within a single running process. Each organization gets their own isolated deployment.
- **Standalone or ecosystem.** It can run as an independent auth microservice inside any stack, or be provisioned and managed by the **maintainerd-core** control plane (see [Deployment Modes](#deployment-modes)).
- **Pluggable identity providers.** Authentication can be handled by the built-in provider or delegated to any external provider (Google, GitHub, Cognito, Auth0, etc.).
- **Multiple user pools.** A single instance can serve multiple independent applications, each with its own users, roles, clients, and settings — analogous to AWS Cognito User Pools or Keycloak Realms.

---

## Deployment Modes

### Standalone

The service runs as a single process with its own PostgreSQL database and Redis instance. A developer drops it into their microservice stack the same way they would any other service. There is no dependency on any external control plane.

Appropriate for:
- Teams building a product that needs auth but do not want to build it themselves.
- Organizations that want to self-host and own their identity layer.

### Maintainerd Ecosystem

When deployed as part of the broader **maintainerd** platform, each instance of maintainerd-auth is provisioned, configured, and monitored by **maintainerd-core** — the control plane analogous to AWS itself. Core handles:

- Provisioning new auth instances per organization.
- Database and infrastructure lifecycle.
- Health monitoring, scaling, and inter-service connectivity.
- Connecting auth to other maintainerd services (storage, notifications, etc.).

In this mode, maintainerd-auth is one of many services an organization can provision — similar to how Firebase provides Auth, Storage, and Firestore as separate but connected services.

---

## Two-Port Architecture

The service exposes two completely isolated HTTP servers on different ports. They share the same database and Redis, but serve entirely different user populations, have separate route sets, and run separate middleware chains.

| Port | Name | Access | Consumer | Purpose |
|---|---|---|---|---|
| `:8080` | Internal / Admin | VPN-only | **maintainerd-auth-console** | Administrative management of the entire auth instance: tenant config, user pools, users, roles, permissions, clients, identity providers, templates, security settings. Has its own private login endpoint for admin users. |
| `:8081` | Public / Identity | Internet | **maintainerd-auth-identity** | End-user flows: login, registration, password reset, token refresh, user self-service. Used as the redirect target in OAuth flows (`redirect_uri`). |

These two ports have nothing to do with each other beyond accessing the same data store. A request that arrives on `:8080` is never routed through any `:8081` handler and vice versa.

---

## Frontends

### maintainerd-auth-console (port 8080)

The admin frontend. Used by the operator of the auth instance to manage everything: create and configure user pools, identity providers, clients, roles, permissions, users, templates, and security settings. Equivalent to the AWS Cognito or Auth0 management dashboard.

On boot, it calls `GET :8080/tenant/` to retrieve the system tenant and verify it is active before rendering the UI.

Access is VPN-only. Admin users log in through a private login endpoint on port 8080 itself.

### maintainerd-auth-identity (port 8081)

The public-facing identity frontend. This is the login and registration page that end-users interact with — the equivalent of Google's sign-in page. Applications redirect their users here to authenticate, then receive the user back at their `redirect_uri`.

On boot, it extracts the pool identifier from the subdomain and calls `GET :8081/tenant/{identifier}/config` to validate the pool is active and retrieve everything needed to render the login page: branding, enabled identity providers, security settings, and signup configuration.

This frontend is fully customizable per user pool through `login_templates`.

---

## Data Hierarchy

```
TENANT  (one record per running instance)
│
├── branding                — auth-console UI identity (logo, colors, URLs)
├── tenant_settings         — rate limits, audit/compliance, maintenance mode, feature flags
├── email_config            — SMTP / transactional email delivery
├── sms_config              — SMS delivery provider
├── webhook_endpoints       — outbound event notifications (many per tenant)
├── ip_restriction_rules    — instance-wide IP allow/block rules
├── TenantMembers           — admin users (port 8080 consumers)
│
├── Services → APIs → Permissions
│   └── ServicePolicies → Policies
│
├── Policies                — JSONB authorization policy documents
│
└── UserPool  (one or more; see User Pools section)
    │
    ├── IdentityProviders   — built-in or external (Google, GitHub, Cognito, Auth0…)
    │   └── Clients         — OAuth/OIDC app registrations (spa, mobile, m2m, traditional)
    │       ├── ClientURIs  — redirect, logout, CORS, origin URIs
    │       └── ClientAPIs → ClientPermissions
    │
    ├── APIKeys → APIKeyAPIs → APIKeyPermissions
    ├── Roles   → RolePermissions → Permissions
    ├── SignupFlows → SignupFlowRoles → Roles
    ├── Invites → InviteRoles → Roles
    ├── login_templates     — per-pool auth-identity UI branding
    ├── email_templates     — per-pool transactional email content
    ├── sms_templates       — per-pool SMS content
    ├── security_settings   — password policy, MFA, session, lockout, threat, registration
    │   └── security_settings_audit
    │
    └── Users
        ├── UserIdentities  — links user to a pool/client/provider (stores JWT sub)
        ├── UserTokens      — email verification and password reset tokens
        ├── UserSettings    — i18n, consent, contact preferences, privacy
        └── Profiles        — personal info, avatar, address, timezone
```

> **Phase 2 note:** Child tables under `UserPool` currently carry a `tenant_id` foreign key rather than a `user_pool_id`. Migrating the FK columns to reference `user_pool_id` directly is planned work. The `user_pools` table and seeding are already in place.
>
> The tables `branding`, `tenant_settings`, `email_config`, `sms_config`, and `webhook_endpoints` are now implemented with full migrations, models, repositories, services, handlers, and routes. The current `security_settings` table also needs its columns refactored (see [Settings](#settings)).

---

## Tenant

The **tenant** is the root entity — there is exactly one tenant record per running instance. It does not represent a user or an application. It represents the organization that owns the deployment.

Everything in the system lives under the tenant. The tenant record holds the instance identity (`identifier`, `name`, `display_name`, `status`) but delegates application-level configuration down to user pools.

The tenant-level tables cover concerns that are shared across the entire instance regardless of which user pool a request belongs to: IP restrictions, email and SMS delivery, branding for the admin console, rate limits, audit configuration, webhooks, and maintenance mode.

---

## User Pools

A **user pool** is the isolation boundary for a single application's user namespace. It is the direct equivalent of an AWS Cognito User Pool or a Keycloak Realm.

Each pool is independently configurable — a consumer gaming app and a fintech app can live on the same instance with completely different password policies, MFA requirements, branding, and identity providers.

Each pool has:
- Its own users, roles, and permissions.
- Its own identity providers and clients.
- Its own branding (login templates, email/SMS templates).
- Its own security settings (password policy, MFA, session, lockout).
- Its own signup flows and invite configuration.
- Its own API keys.

### The System Pool

Every instance has exactly one user pool with `is_system = true`. This pool:
- **Cannot be deleted.**
- Contains the admin users who operate the auth console (port 8080).
- Is linked to admin users via `tenant_members`, not `user_identities`.
- Has a default built-in identity provider pre-configured.
- Has at least one default client that identifies the `maintainerd-auth-console` frontend.

### Regular Pools

Developers create regular pools — one per application they are building. Each pool is fully independent and does not affect other pools on the same instance.

---

## Identity Providers

An **identity provider (IDP)** defines *where* authentication happens for users in a pool.

### Built-in Provider

The default provider. Authentication (credential verification, MFA, token issuance) is handled entirely within this service. This is the mode that works out of the box with no external dependencies.

### External Providers

Developers who prefer to delegate authentication to a third-party service — or who already have users in another system — can configure an external provider:

| Provider type | Examples |
|---|---|
| Social / OIDC | Google, GitHub, Apple, Microsoft |
| Managed auth services | AWS Cognito, Auth0, Okta, Firebase Auth |
| Enterprise SSO | SAML-based, LDAP-backed providers |

When an external provider is configured:
- Authentication (credential check, MFA) happens at the external provider.
- This service receives the identity assertion (OAuth code / OIDC token) from the external provider.
- It creates or updates a `UserIdentity` record mapping the external `sub` to a local user.
- It manages that user's **roles and permissions** going forward — authorization still lives here.

The IDP record stores the provider type, credentials (client ID, client secret), and any provider-specific configuration in a JSONB `config` field.

---

## Clients

A **client** is an application registration within an identity provider — the same concept as an OAuth 2.0 client, a Cognito App Client, or an Auth0 Application.

Each client belongs to an identity provider (and therefore a user pool) and has a type:

| Type | Use case |
|---|---|
| `spa` | Single-page application (public client, PKCE) |
| `mobile` | Native mobile app (public client, PKCE) |
| `traditional` | Server-rendered web app (confidential client) |
| `m2m` | Machine-to-machine (client credentials) |

A client holds:
- Redirect URIs (`ClientURIs`) — validated on OAuth flows.
- Allowed APIs and permissions (`ClientAPIs`, `ClientPermissions`).
- A `secret` (for confidential clients).
- A `config` JSONB for provider-specific settings.

The system pool's built-in IDP has at least one pre-seeded client that represents **maintainerd-auth-console** (the admin frontend on port 8080).

---

## Roles and Permissions

### Roles

A **role** is a named collection of permissions scoped to a user pool. Roles are assigned to users directly (`user_roles`) or automatically via signup flows.

Each role has:
- A name and description.
- A status (`active` / `inactive`).
- An `is_system` flag for roles that are pre-seeded and cannot be deleted (e.g., `super-admin`, `registered`).
- An `is_default` flag to automatically assign the role to all newly registered users in the pool.
- A set of permissions via `role_permissions`.

### Permissions

A **permission** is a named operation scoped to an API (e.g., `user:read`, `user:create`). Permissions come from the service/API/permission registry and are assigned to roles. A user's effective permissions are the union of permissions across all their assigned roles.

Permissions are also assigned directly to clients (`client_permissions`) to restrict which operations an OAuth client can request on behalf of its users.

---

## Users and Identities

### Users

A `User` record is a global entity — it is not scoped to a pool or tenant directly. It holds the canonical identity: `email`, `username`, `phone`, `password` (nullable for external-provider users), verification flags, and account status.

Associated with each user:
- **Profile** — extended personal information: name, bio, avatar, address, timezone, language, gender.
- **UserSettings** — per-user preferences: timezone, language, locale, social links, contact method preference, marketing consent, privacy settings, terms acceptance.
- **UserTokens** — short-lived tokens for email verification and password reset flows.

### User Identities

A `UserIdentity` record is the bridge that places a user inside a specific pool/client/provider context. One user can have multiple identities — for example, the same person authenticated via the built-in provider in pool A and via Google in pool B.

Each identity stores:
- `user_id` — which user.
- `tenant_id` / `client_id` — which app context.
- `provider` — which identity provider.
- `sub` — the stable subject identifier from that provider (used as the JWT `sub` claim).

---

## Services, APIs, and Permissions

These three tables form the **resource registry** — a catalog of what APIs and operations exist within this auth instance.

```
Service  (a top-level product or microservice)
  └── API  (a specific API surface: REST, gRPC, GraphQL, etc.)
        └── Permission  (a named operation: user:read, user:create, etc.)
```

Permissions from this registry are assigned to roles and to clients, forming the basis of all authorization checks.

---

## Policies

**Policies** are JSONB-document-based authorization rules attached to services. They allow expressing complex access control logic beyond simple role-permission checks — similar to AWS IAM policies. Each policy has a `document` (the rule set), a `version`, and a `status`.

---

## API Keys

**API keys** provide machine-to-machine access without going through an OAuth flow. They are scoped to a user pool and can be restricted to specific APIs and permissions.

Each key has:
- A hashed key value (`key_hash`) and a short display prefix (`key_prefix`) — the raw key is only shown once on creation.
- An optional expiry date.
- An optional rate limit (requests per time window).
- Explicit API and permission scopes via `api_key_apis` and `api_key_permissions`.

---

## Signup Flows

A **signup flow** defines the registration experience for a specific client within a pool. Different clients can have different signup configurations — a consumer app can allow open registration while a B2B client on the same pool requires an invite.

Each signup flow has:
- A name and identifier tied to a specific client.
- A `config` JSONB with the flow-specific rules (required fields, domain restrictions, etc.).
- An assigned set of roles automatically granted to users who register through this flow (`signup_flow_roles`).

---

## Invites

The **invite system** provides controlled user onboarding. An admin or authorized user sends an invite to an email address. The recipient registers using the invite token and is automatically placed into the pool with the pre-assigned roles.

Each invite stores:
- The invited email, the inviting user, and the client context.
- A unique, hashed invite token with an expiry.
- A status: `pending`, `accepted`, `revoked`, or `expired`.
- Roles to assign on acceptance (`invite_roles`).

---

## Branding and Templates

Branding operates at two levels to serve two different audiences.

### Tenant Branding (`branding`)

Consumed by **auth-console** (port 8080). Covers the operator's experience inside the admin dashboard: company name, logo, favicon, primary color, support URLs, privacy policy, and terms of service links. One record per tenant.

### User Pool Login Template (`login_templates`)

Consumed by **auth-identity** (port 8081). Defines what end-users see on the login and registration page. Each user pool can have its own login template, which means different apps on the same instance can have completely different visual identities.

Covers: layout choice, logo, primary and background colors, custom CSS, and other UI configuration. This is the equivalent of Cognito's hosted UI customization.

### Email Templates (`email_templates`)

Per-pool HTML and plain-text templates for all transactional emails sent by the auth system: email verification, password reset, invite, welcome, and lockout notifications.

### SMS Templates (`sms_templates`)

Per-pool SMS message templates for one-time passwords and phone verification messages.

---

## Settings

### Tenant-level settings

Settings at the tenant level cover concerns shared across the entire instance. They are split into focused tables by concern and lifecycle so that different teams can own different records without contention.

**`tenant_settings`** — Core operational flags: global rate limits, audit/compliance settings (log retention, GDPR mode, PII masking, data deletion strategy), maintenance mode (with per-IP bypass list), and feature toggles (API keys, invite system, webhooks).

**`email_config`** — Transactional email delivery: provider selection (SMTP, SES, SendGrid, Mailgun, Postmark, Resend), sender identity (from address, from name, reply-to), TLS mode, and test mode.

**`sms_config`** — SMS delivery: provider selection (Twilio, SNS, Vonage, MessageBird), sender number, sender ID, and test mode.

**`branding`** — Admin console visual identity. See [Branding and Templates](#branding-and-templates).

**`webhook_endpoints`** — Outbound event notifications. One row per endpoint. Each endpoint has its own URL, HMAC signing secret, event filter list, retry count, timeout, and enabled/disabled status.

**`ip_restriction_rules`** — IP allow/block rules applied instance-wide before any pool-level logic runs.

---

### User Pool Security Settings (`security_settings`)

Per-pool configuration. Each pool can have entirely different values. Stored as named JSONB columns — one per concern — on a single row per pool.

**Password policy** — minimum and maximum length, complexity requirements (uppercase, lowercase, numbers, symbols), common password rejection, compromised password check (HaveIBeenPwned), password history (prevent reuse of last N), maximum age and expiry, temporary password window.

**MFA config** — mode (disabled / optional / enforced), allowed methods (TOTP authenticator app, SMS, email OTP), authenticator issuer name, trusted device period, grace period for enforcement rollout on existing users.

**Session config** — access token TTL, refresh token TTL, maximum concurrent sessions, idle timeout, absolute session timeout, refresh token rotation, reuse interval for concurrent requests.

**Lockout config** — enable/disable, maximum failed login attempts, lockout duration, progressive lockout (duration doubles on repeat offences), auto-unlock vs. admin-required, reset failed count on success.

**Registration config** — open registration or invite-only, email verification required, phone verification required, allowed and blocked email domains, auto-confirm bypass.

**Threat config** — brute-force detection, impossible travel detection, new device login notification, velocity checks (too many new accounts from the same IP), risk-based step-up auth, compromised credential monitoring.

**Token config** — JWT clock-skew leeway, additional claims to include in the ID token, additional claims to include in the access token.

---

## Tokens and JWT

The service issues three types of JWT:

| Token | TTL | Purpose |
|---|---|---|
| Access token | 15 minutes | Authorizes API requests. Short-lived by design. |
| ID token | 1 hour | Carries user identity claims for the client application. |
| Refresh token | 7 days | Exchanges for a new access/id token pair without re-authentication. |

All tokens are signed with RSA-256 using a minimum 2048-bit key pair. The key pair is loaded from environment variables at startup and is the same key used for both ports.

Each access token carries: `sub` (the `UserIdentity.sub`), `scope`, `aud`, `iss`, `jti`, `client_id`, `provider_id`.

The `iss` (issuer) claim is set from the `ISSUER_URL` environment variable and must match the value in the OIDC discovery document.

Refresh tokens are hashed before storage in `user_tokens` and are bound to the client and user agent that requested them.

---

## Admin Users vs External Users

| | Admin users | External / end users |
|---|---|---|
| **Port** | `:8080` | `:8081` |
| **Frontend** | maintainerd-auth-console | maintainerd-auth-identity |
| **Pool** | System pool (`is_system = true`) | Regular user pool(s) |
| **Linked via** | `tenant_members` | `user_identities` |
| **Purpose** | Manage the auth instance | Authenticate into developer apps |
| **Deletable pool?** | No | Yes |

Admin users log in through a private login endpoint on port 8080 (VPN-only). External users log in through the public login endpoint on port 8081, which is also the landing page for all OAuth redirect flows initiated by third-party applications.

---

## Known Limitations and Planned Work

The following are known gaps between the intended design and the current implementation. They are not bugs — they are scoped to future milestones.

| Item | Status | Notes |
|---|---|---|
| `user_pool_id` FK on child tables | Phase 2 | Roles, clients, IDPs, signup flows, etc. currently reference `tenant_id` directly. Migrating to `user_pool_id` is planned. Security settings and audit already reference `user_pool_id`. |
| `security_settings` scoping | Done | Migrated from `tenant_id` to `user_pool_id`. Columns refactored: `general_config` and `ip_config` replaced with `mfa_config`, `lockout_config`, `registration_config`, `token_config`. |
| `tenant_settings` table | Done | Migration 004, model `TenantSetting`. |
| `email_config` table | Done | Migration 005, model `EmailConfig`. |
| `sms_config` table | Done | Migration 006, model `SMSConfig`. |
| `branding` table | Done | Migration 003, model `Branding`. Tenant-level admin console branding. |
| `webhook_endpoints` table | Done | Migration 007, model `WebhookEndpoint`. Needs dispatcher implementation. |
| `login_templates` `tenant_id` FK | Phase 2 | Should reference `user_pool_id` since branding is per-pool. |
| OIDC provider (JWKS + discovery) | In progress | `/.well-known/jwks.json` and `/.well-known/openid-configuration` on port 8081. See `docs/v1-features/oidc-provider.md`. |
| Frontend init endpoint | In progress | `GET /tenant/{identifier}/config` on port 8081. See `docs/v1-features/frontend-initialization.md`. |
| Health and readiness endpoints | Planned | `/healthz` and `/readyz` on both ports. |
| CORS on public port | Planned | Required for browser-based clients on port 8081. |
| gRPC layer | Future | Only a stub (`SeederService`) exists. Not in active development. |
