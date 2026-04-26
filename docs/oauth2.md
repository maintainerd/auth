# OAuth 2.0 Authorization Server — Research & Implementation Plan

This document covers the standards research, provider study, and implementation approach for building a custom OAuth 2.0 authorization server within maintainerd-auth. No pre-built OAuth packages are used.

> **Architecture Note:** maintainerd-auth is a **backend API only**. There is no frontend in this
> project. The login page, consent screen, and all other user-facing UI will be provided by a
> **separate frontend application** that does not yet exist. All OAuth endpoints in this service
> expose JSON APIs or issue HTTP redirects — they never render HTML. The authorization flow
> relies on redirecting the user-agent to configurable frontend URLs (derived from `LoginURI`
> on the client) for authentication and consent, and the frontend calls back into our API to
> finalize those steps.

---

## Table of Contents

1. [Standards Overview](#1-standards-overview)
2. [Core RFCs](#2-core-rfcs)
3. [Grant Types](#3-grant-types)
4. [Endpoints](#4-endpoints)
5. [Token Design](#5-token-design)
6. [PKCE (Proof Key for Code Exchange)](#6-pkce-proof-key-for-code-exchange)
7. [Client Authentication](#7-client-authentication)
8. [Consent Model](#8-consent-model)
9. [Security Considerations](#9-security-considerations)
10. [Provider Study](#10-provider-study)
11. [Existing Infrastructure](#11-existing-infrastructure)
    - [Identity Provider, Client & OAuth 2.0 Association](#identity-provider-client--oauth-20-association)
12. [Implementation Approach](#12-implementation-approach)
13. [Database Schema](#13-database-schema)
14. [Endpoint Specifications](#14-endpoint-specifications)
15. [Implementation Phases](#15-implementation-phases)

---

## 1. Standards Overview

We target **OAuth 2.1** (draft-ietf-oauth-v2-1-11+) as our specification baseline, which consolidates:

| Specification | RFC / Draft | Role |
|---|---|---|
| OAuth 2.0 Authorization Framework | RFC 6749 | Base protocol |
| Bearer Token Usage | RFC 6750 | Token transmission |
| PKCE | RFC 7636 | Authorization code protection |
| Token Revocation | RFC 7009 | Token invalidation endpoint |
| Token Introspection | RFC 7662 | Token metadata endpoint |
| Authorization Server Metadata | RFC 8414 | Discovery endpoint |
| OAuth Security BCP | draft-ietf-oauth-security-topics | Current security best practices |
| OAuth 2.1 | draft-ietf-oauth-v2-1 | Consolidated modern standard |
| JWT | RFC 7519 | Token format |
| JWT Profile for Access Tokens | RFC 9068 | Structured access tokens |

### What OAuth 2.1 Changes From 2.0

- **Removes** the Implicit grant (`response_type=token`) — insecure, tokens exposed in URL fragments
- **Removes** the Resource Owner Password Credentials (ROPC) grant — credentials shared with clients
- **Requires** PKCE for all authorization code flows (both public and confidential clients)
- **Requires** exact string matching for redirect URIs (no wildcard or partial matching)
- **Requires** refresh token rotation or sender-constraining for public clients
- **Removes** bearer tokens in URI query strings

### Grant Types We Support

| Grant Type | `grant_type` Value | Use Case |
|---|---|---|
| Authorization Code + PKCE | `authorization_code` | Web apps, SPAs, mobile, native |
| Client Credentials | `client_credentials` | Machine-to-machine (M2M) |
| Refresh Token | `refresh_token` | Obtaining new access tokens |
| Device Authorization | `urn:ietf:params:oauth:grant-type:device_code` | TVs, CLI tools, IoT (future) |

We intentionally **omit** Implicit and ROPC per OAuth 2.1.

---

## 2. Core RFCs

### RFC 6749 — OAuth 2.0 Authorization Framework

Defines the four roles:

| Role | In Our System |
|---|---|
| **Resource Owner** | The end user (model `User`) |
| **Client** | The application requesting access (model `Client`) |
| **Authorization Server** | maintainerd-auth itself |
| **Resource Server** | APIs protected by our tokens (model `API`) |

Key protocol flows: authorization endpoint (user-facing), token endpoint (back-channel), redirect-based communication. All parameters use `application/x-www-form-urlencoded`.

### RFC 6750 — Bearer Token Usage

Three methods to transmit bearer tokens (in order of preference):

1. **Authorization header** (REQUIRED to support): `Authorization: Bearer <token>`
2. **Form-encoded body** (MAY support): `access_token=<token>` in POST body
3. **URI query parameter** (MUST NOT use per OAuth 2.1): `?access_token=<token>`

Error codes: `invalid_request`, `invalid_token`, `insufficient_scope`. Must include `WWW-Authenticate` header with `Bearer` scheme on 401 responses.

### RFC 7636 — PKCE

Protects the authorization code flow from interception attacks:

1. Client generates `code_verifier`: 43–128 chars, high-entropy random string (`[A-Z]/[a-z]/[0-9]/-/./_/~`)
2. Client derives `code_challenge`: `BASE64URL(SHA256(code_verifier))` (S256 method, mandatory to implement)
3. Client sends `code_challenge` + `code_challenge_method=S256` in authorization request
4. Client sends `code_verifier` in token request
5. Server verifies: `BASE64URL(SHA256(code_verifier)) == stored code_challenge`

**We require PKCE for ALL clients** (both public and confidential) per OAuth 2.1 §7.5.1.

### RFC 7009 — Token Revocation

- `POST /revoke` with `token` (required) and `token_type_hint` (optional: `access_token` or `refresh_token`)
- Client MUST authenticate (confidential) or send `client_id` (public)
- Server responds 200 OK on success (even for invalid/already-revoked tokens)
- Revoking a refresh token SHOULD also invalidate associated access tokens
- Revoking an access token MAY also revoke the refresh token

### RFC 7662 — Token Introspection

- `POST /introspect` with `token` parameter
- Endpoint MUST be authenticated (client credentials or separate bearer token)
- Response: JSON with `active` (required boolean), plus optional: `scope`, `client_id`, `username`, `token_type`, `exp`, `iat`, `nbf`, `sub`, `aud`, `iss`, `jti`
- Inactive tokens return `{"active": false}` — must not reveal why inactive
- Response MAY be cached (trade-off: freshness vs. performance)

### RFC 8414 — Authorization Server Metadata

Published at `/.well-known/oauth-authorization-server` (or `/.well-known/openid-configuration` for OIDC compatibility).

Required/recommended fields:

```json
{
  "issuer": "https://auth.example.com",
  "authorization_endpoint": "https://auth.example.com/oauth/authorize",
  "token_endpoint": "https://auth.example.com/oauth/token",
  "jwks_uri": "https://auth.example.com/.well-known/jwks.json",
  "revocation_endpoint": "https://auth.example.com/oauth/revoke",
  "introspection_endpoint": "https://auth.example.com/oauth/introspect",
  "scopes_supported": ["openid", "profile", "email"],
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "client_credentials", "refresh_token"],
  "token_endpoint_auth_methods_supported": ["client_secret_basic", "client_secret_post", "none"],
  "code_challenge_methods_supported": ["S256"],
  "response_modes_supported": ["query", "fragment", "form_post"]
}
```

Consumers MUST validate that the returned `issuer` matches the URL used to fetch the document.

---

## 3. Grant Types

### 3.1 Authorization Code Grant + PKCE

The primary grant type for all interactive user flows.

```
┌──────────┐                                    ┌────────────────────┐
│  Client   │──(1) Authorization Request ──────>│  Authorization     │
│           │     response_type=code             │  Endpoint          │
│           │     client_id, redirect_uri        │                    │
│           │     scope, state                   │  ┌──────────────┐  │
│           │     code_challenge, method=S256    │  │ User Login   │  │
│           │                                    │  │ + Consent    │  │
│           │<─(2) Redirect to redirect_uri ────│  └──────────────┘  │
│           │     code, state                    └────────────────────┘
│           │
│           │──(3) Token Request ──────────────> ┌────────────────────┐
│           │     grant_type=authorization_code  │  Token Endpoint    │
│           │     code, redirect_uri             │                    │
│           │     code_verifier                  │  Validates:        │
│           │     client authentication          │  - code            │
│           │                                    │  - PKCE verifier   │
│           │<─(4) Token Response ──────────────│  - client auth     │
│           │     access_token, token_type       │  - redirect_uri    │
│           │     expires_in, refresh_token      └────────────────────┘
└──────────┘
```

**Authorization Request Parameters** (`GET /oauth/authorize`):

| Parameter | Required | Description |
|---|---|---|
| `response_type` | REQUIRED | Must be `code` |
| `client_id` | REQUIRED | Client identifier |
| `redirect_uri` | CONDITIONAL | Required if multiple redirect URIs registered |
| `scope` | OPTIONAL | Space-delimited scope list |
| `state` | RECOMMENDED | Opaque CSRF protection value |
| `code_challenge` | REQUIRED | PKCE challenge derived from verifier |
| `code_challenge_method` | OPTIONAL | `S256` (default if omitted), `plain` NOT supported |

**Authorization Response** (redirect to `redirect_uri`):

| Parameter | Description |
|---|---|
| `code` | Authorization code (max 10 min lifetime, single use) |
| `state` | Echoed back from request |
| `iss` | Authorization server issuer identifier (mix-up prevention) |

**Token Request Parameters** (`POST /oauth/token`):

| Parameter | Required | Description |
|---|---|---|
| `grant_type` | REQUIRED | `authorization_code` |
| `code` | REQUIRED | The authorization code |
| `redirect_uri` | NOT SENT | Per OAuth 2.1, no longer required in token request |
| `code_verifier` | REQUIRED | PKCE verifier string |
| `client_id` | REQUIRED | If client is not authenticating via other means |

### 3.2 Client Credentials Grant

For confidential machine-to-machine clients. No user involvement.

```
┌──────────┐                                    ┌────────────────────┐
│  Client   │──(1) Token Request ──────────────>│  Token Endpoint    │
│  (M2M)    │     grant_type=client_credentials │                    │
│           │     scope (optional)               │  Authenticates     │
│           │     client_secret_basic or _post   │  client            │
│           │                                    │                    │
│           │<─(2) Token Response ──────────────│  Returns:          │
│           │     access_token, token_type       │  access_token      │
│           │     expires_in                     │  (NO refresh_token)│
└──────────┘                                    └────────────────────┘
```

- MUST only be used by confidential clients (`traditional`, `m2m` types)
- MUST NOT issue refresh tokens (client can always re-authenticate)
- Scopes granted must be pre-approved for this client

### 3.3 Refresh Token Grant

Obtain new access tokens without user interaction.

**Token Request Parameters:**

| Parameter | Required | Description |
|---|---|---|
| `grant_type` | REQUIRED | `refresh_token` |
| `refresh_token` | REQUIRED | The refresh token |
| `scope` | OPTIONAL | Must not exceed original grant scope |

**Refresh Token Security:**

- Confidential clients MUST authenticate when refreshing
- Public clients: we implement **refresh token rotation** — each refresh issues a new refresh token and invalidates the old one
- **Reuse detection**: if a rotated-out refresh token is used, ALL tokens in that family are revoked (detects token theft)
- Refresh tokens SHOULD expire on inactivity (configurable, default: 30 days idle, 90 days absolute)
- Automatic revocation on: password change, admin-initiated logout, security events

---

## 4. Endpoints

### Endpoint Registry

| Endpoint         | Method | Port            | Path                                | RFC                  |
| ---------------- | ------ | --------------- | ----------------------------------- | -------------------- |
| Authorization    | GET    | Public (8081)   | `/oauth/authorize`                  | RFC 6749 §3.1        |
| Token            | POST   | Public (8081)   | `/oauth/token`                      | RFC 6749 §3.2        |
| Consent (read)   | GET    | Public (8081)   | `/oauth/consent`                    | — (API for frontend) |
| Consent (submit) | POST   | Public (8081)   | `/oauth/consent`                    | — (API for frontend) |
| Revocation       | POST   | Public (8081)   | `/oauth/revoke`                     | RFC 7009             |
| Introspection    | POST   | Internal (8080) | `/oauth/introspect`                 | RFC 7662             |
| UserInfo         | GET    | Public (8081)   | `/oauth/userinfo`                   | OIDC Core §5.3       |
| JWKS             | GET    | Public (8081)   | `/.well-known/jwks.json`            | RFC 7517             |
| Discovery        | GET    | Public (8081)   | `/.well-known/openid-configuration` | RFC 8414             |

**Design Decisions:**
- Authorization and Token endpoints on the **public port** (8081) — they are client-facing
- Consent endpoints on the **public port** (8081) — the separate frontend application calls these to retrieve consent details and submit the user's decision
- Introspection on the **internal port** (8080) — only resource servers (internal) call this
- JWKS and Discovery are public and unauthenticated (read-only metadata)
- UserInfo is protected by a valid access token
- All endpoints return JSON or issue HTTP redirects — no HTML is rendered (this is an API-only service)
- CORS MUST NOT be enabled on the Authorization endpoint (redirect-based, not AJAX)
- CORS SHOULD be enabled on the Token, Consent, and UserInfo endpoints (for SPA/frontend clients)

---

## 5. Token Design

### Access Tokens

**Format**: JWT (RFC 9068 profile) — self-contained for resource server validation without introspection.

**Claims:**

```json
{
  "iss": "https://auth.example.com",
  "sub": "<user-identity-sub>",
  "aud": ["https://api.example.com"],
  "exp": 1700000000,
  "iat": 1699999100,
  "nbf": 1699999100,
  "jti": "<unique-token-id>",
  "client_id": "<client-identifier>",
  "scope": "openid profile email",
  "tenant_id": "<tenant-uuid>"
}
```

**Lifetime**: 15 minutes (configurable per tenant via `security_settings`). Short-lived to limit damage from token theft.

**Signing**: RS256 with key rotation via `kid` header. Existing JWT infrastructure supports this.

### Refresh Tokens

**Format**: Opaque (random, high-entropy) — stored server-side, never parsed by clients.

**Generation**: 256-bit cryptographically random bytes, base64url-encoded.

**Storage**: Hashed in the database (SHA-256). The raw token is sent to the client once and never stored in plaintext.

**Lifetime**: 7 days (configurable). Absolute lifetime: 90 days. Inactivity timeout: 30 days.

**Rotation**: Every use generates a new refresh token. Previous token is invalidated. Reuse of an invalidated token revokes the entire token family.

### Authorization Codes

**Format**: Opaque, high-entropy random string.

**Lifetime**: Maximum 10 minutes (RECOMMENDED by OAuth 2.1). Single use.

**Storage**: Server-side with associated metadata (client_id, redirect_uri, scope, code_challenge, code_challenge_method, user, expiry).

### ID Tokens (OIDC)

**Format**: JWT signed with RS256.

**Claims**: Standard OIDC claims (`sub`, `iss`, `aud`, `exp`, `iat`, `nonce`, `auth_time`, plus profile claims when `openid` scope is requested).

**Lifetime**: 1 hour (configurable).

### Token Response Format

```json
{
  "access_token": "<jwt>",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "<opaque>",
  "scope": "openid profile email",
  "id_token": "<jwt>"
}
```

Headers MUST include: `Cache-Control: no-store`, `Pragma: no-cache`.

---

## 6. PKCE (Proof Key for Code Exchange)

### Why Required for All Clients

OAuth 2.1 mandates PKCE even for confidential clients. PKCE provides:
- Protection against authorization code interception (network-level and on-device)
- CSRF protection (replaces `state` for this purpose, though `state` is still RECOMMENDED)
- Defense-in-depth for confidential clients

### Implementation

```
Client:
  code_verifier = base64url(random(32 bytes))  // 43 chars
  code_challenge = base64url(sha256(code_verifier))

Authorization Request:
  &code_challenge=<challenge>&code_challenge_method=S256

Token Request:
  &code_verifier=<verifier>

Server Validation:
  base64url(sha256(code_verifier)) == stored_code_challenge
```

**We only support S256** — the `plain` method is insecure against network attackers and is NOT supported. If `code_challenge_method` is omitted, we reject the request (we don't default to `plain`).

---

## 7. Client Authentication

### Methods

| Method | `token_endpoint_auth_method` | Client Types | Description |
|---|---|---|---|
| HTTP Basic | `client_secret_basic` | `traditional`, `m2m` | `Authorization: Basic base64(client_id:client_secret)` |
| POST Body | `client_secret_post` | `traditional`, `m2m` | `client_id` + `client_secret` in form body |
| None | `none` | `spa`, `mobile` | Public client, `client_id` only |

### Client Secret Handling

- Secrets are generated with 256-bit entropy (32 random bytes, base64url-encoded)
- Stored as bcrypt hashes (never plaintext) in the `clients` table `secret` column
- The raw secret is shown once at client creation and cannot be retrieved again
- Secret rotation: generate new secret, grace period for old secret, revoke old

### How We Map Existing Client Types

| Our Client Type | OAuth Client Type | Auth Method | Supports Refresh Tokens |
|---|---|---|---|
| `traditional` | Confidential | `client_secret_basic` / `client_secret_post` | Yes |
| `m2m` | Confidential | `client_secret_basic` / `client_secret_post` | No (client_credentials) |
| `spa` | Public | `none` | Yes (with rotation) |
| `mobile` | Public | `none` | Yes (with rotation) |

---

## 8. Consent Model

### When Consent Is Required

Following Google and Auth0 patterns:

| Scenario | Consent Required |
|---|---|
| First-party client (`is_system = true`) | No — skip consent |
| Third-party client, first authorization | Yes |
| Third-party client, previously consented (same scopes) | No — reuse existing consent |
| Third-party client, requesting additional scopes | Yes — incremental consent |
| `prompt=consent` parameter | Yes — always show consent |

### Consent Flow

Since maintainerd-auth is an API-only backend, it never renders a consent screen. Instead, the flow delegates to the frontend:

1. After user authenticates, check if consent record exists for this `(user, client, scopes)` tuple
2. If complete consent exists and `prompt != consent`, proceed directly — issue authorization code and redirect to `redirect_uri`
3. If partial or no consent, redirect the user-agent to the frontend consent URL with an opaque `consent_challenge` identifier
4. The frontend calls `GET /oauth/consent?consent_challenge=<id>` to retrieve consent details (client name, logo, requested scopes with descriptions)
5. The frontend displays the consent UI and collects the user's decision
6. The frontend calls `POST /oauth/consent` with the `consent_challenge` and the user's decision (`accept` or `reject`)
7. On approval, the API stores the consent record, issues an authorization code, and returns the redirect URL to the frontend
8. On denial, the API returns the redirect URL with `error=access_denied` for the frontend to execute

### Consent Storage

Consent grants are stored per user-client pair with the granted scopes list and timestamp. Consents can be revoked by the user, which also revokes all associated tokens.

---

## 9. Security Considerations

### From OAuth 2.1 Security BCP and RFC 6749 §10

| Threat | Mitigation |
|---|---|
| **Authorization code interception** | PKCE required (S256 only) |
| **Authorization code injection** | PKCE verifier binds code to original requester |
| **CSRF on redirect** | PKCE provides CSRF protection; `state` recommended as defense-in-depth |
| **Token leakage via referrer** | Authorization codes in query (not fragment); short lifetime |
| **Open redirector** | Exact redirect URI matching (no wildcards, no partial match) |
| **Clickjacking** | Authorization endpoint is API-only (JSON/redirects); frontend must set `X-Frame-Options: DENY` and CSP `frame-ancestors 'none'` on its consent/login pages |
| **Mix-up attacks** | `iss` parameter in authorization response (RFC 9207) |
| **Token theft** | Short-lived access tokens (15 min); refresh token rotation with reuse detection |
| **Client impersonation** | Client secret for confidential clients; PKCE for all |
| **Refresh token theft** | Rotation + family-based revocation; bound to client_id |
| **Credential guessing** | Rate limiting on token endpoint (existing rate limit infra) |
| **Token scanning** | Introspection requires authentication; returns `active: false` for unknown tokens |
| **Phishing** | Frontend login/consent on a known, configurable hostname; API enforces exact redirect URI matching |
| **HTTP 307 redirect** | Use 302 redirects only (307 would forward POST body with credentials) |

### Additional Hardening

- **TLS required** on all OAuth endpoints (enforced at proxy/infra level)
- **Token binding**: `jti` claim for access tokens to enable revocation checking
- **Scope restriction**: access tokens carry only the scopes granted, not all scopes the client could request
- **Audience restriction**: `aud` claim limits which resource servers accept the token
- **No bearer tokens in URLs**: query parameter method is not supported
- **Constant-time comparison** for code verifier, client secrets, and tokens
- **Auth event logging**: all OAuth events logged via the auth events system (OWASP compliant)

### OAuth Events to Log (via Auth Events System)

| Event Type | When |
|---|---|
| `authn_login_success` | User authenticates at authorization endpoint |
| `authn_login_fail` | Authentication failure at authorization endpoint |
| `authz_consent_granted` | User grants consent to a client |
| `authz_consent_denied` | User denies consent |
| `authz_token_created` | Access/refresh token issued via token endpoint |
| `authz_token_revoked` | Token revoked via revocation endpoint |
| `authz_token_refresh` | Refresh token used to obtain new access token |
| `authz_code_created` | Authorization code issued |
| `authz_code_redeemed` | Authorization code exchanged for tokens |
| `authz_code_replay` | Attempt to reuse an authorization code (attack indicator) |

---

## 10. Provider Study

### How Major Providers Compare

| Feature | Google | GitHub | Auth0 | Okta | Our Approach |
|---|---|---|---|---|---|
| **Grant types** | Auth code, refresh, device, JWT bearer | Auth code, device | Auth code+PKCE, client creds, device, refresh | Auth code+PKCE, client creds, device, refresh | Auth code+PKCE, client creds, refresh |
| **Access token format** | Opaque | Opaque (prefixed) | JWT (RS256) | JWT (RS256, custom AS) | **JWT (RS256)** |
| **Refresh token format** | Opaque | None (long-lived access) | Opaque | Opaque | **Opaque (hashed storage)** |
| **PKCE** | Supported (S256, plain) | S256 only, recommended | Required for SPAs | Required for public clients | **Required for all (S256 only)** |
| **Refresh rotation** | No (count-limited: 100) | N/A | Yes (with reuse detection) | Configurable | **Yes (with reuse detection)** |
| **Consent** | Google-hosted, incremental | GitHub-hosted, accumulative | Customizable, skippable for 1P | Customizable | **Configurable, skippable for 1P** |
| **Discovery** | Full OIDC discovery | None | Full OIDC discovery | Dual (OAuth + OIDC) | **Full OIDC-compatible discovery** |
| **Introspection** | tokeninfo (non-standard) | None | Standard | Standard | **RFC 7662 standard** |
| **Revocation** | Standard | None | Standard | Standard | **RFC 7009 standard** |
| **Multi-tenant** | Per-project | Per-org | Per-tenant subdomain | Per-AS with custom issuer | **Per-tenant (existing infra)** |
| **Client auth** | secret_basic, secret_post | POST body | secret_basic, secret_post, private_key_jwt | secret_basic, secret_post, private_key_jwt | **secret_basic, secret_post, none** |

### Key Patterns Adopted From Providers

**From Google:**
- Token count limits per user/client (configurable, default 25)
- Refresh token expiry on password change and security events
- Incremental scope authorization

**From Auth0:**
- Refresh token rotation with reuse detection (family tracking)
- Multi-tenant with per-tenant signing keys (future consideration)
- First-party client skips consent

**From Okta:**
- Per-authorization-server configuration model (maps to our per-tenant model)
- JWT access tokens for resource server validation
- Access policies controlling which clients get which scopes/lifetimes

**From ory/fosite (Architecture):**
- Handler chain pattern for extensibility (authorize handlers, token handlers, revocation handlers)
- Per-feature storage interfaces (AuthorizeCodeStorage, AccessTokenStorage, RefreshTokenStorage)
- Strategy pattern for token generation (we already use something similar via our JWT package)
- Compose/factory pattern for assembling grant type support

---

## 11. Existing Infrastructure

What we already have that maps to OAuth needs:

| Need | Existing Asset | Status |
|---|---|---|
| JWT signing/validation | `internal/jwt/` (RS256, key rotation, 3 token types) | Ready — extend for OAuth-specific claims |
| Client model | `internal/model/client.go` (types: traditional, spa, mobile, m2m; includes OAuth 2.0 fields) | Ready |
| Redirect URIs | `internal/model/client_uri.go` (type: `redirect-uri`) | Ready |
| User model + identity | `internal/model/user.go`, `user_identity.go` | Ready |
| Password verification | `internal/service/login.go` (bcrypt comparison) | Reuse for authorization endpoint |
| Token storage | `internal/model/user_token.go` | Extend for OAuth token types |
| Rate limiting | Existing Redis-based rate limiter | Reuse for token endpoint |
| Crypto utilities | `internal/crypto/rand.go`, `otp.go` | Extend for code_verifier validation |
| Middleware | JWT validation, user context, permission checking | Reuse for protected endpoints |
| Auth events | 27 OWASP-compliant event types | Wire OAuth events |
| Security headers | `internal/middleware/security_middleware.go` | Apply to authorization endpoint |
| Multi-tenancy | Per-tenant model with identity providers per tenant | Maps to per-tenant OAuth config |
| Scopes/Permissions | `internal/model/permission.go`, `client_permission.go` | Map permissions to OAuth scopes |
| API resources | `internal/model/api.go`, `client_api.go` | Map APIs to OAuth `resource` / `audience` |

### What We Need to Build

| Component | Layer | Priority |
|---|---|---|
| Authorization code model + migration | Model / DB | P0 |
| OAuth consent grant model + migration | Model / DB | P0 |
| Refresh token family tracking model + migration | Model / DB | P0 |
| Authorization code repository | Repository | P0 |
| OAuth consent repository | Repository | P0 |
| Refresh token repository (OAuth-specific) | Repository | P0 |
| Authorization service (authorize flow) | Service | P0 |
| Token service (token endpoint logic) | Service | P0 |
| OAuth consent service | Service | P0 |
| PKCE validation utilities | Service/Util | P0 |
| Authorization handler | Handler | P0 |
| Token handler | Handler | P0 |
| Revocation handler | Handler | P0 |
| Introspection handler | Handler | P1 |
| UserInfo handler | Handler | P1 |
| Discovery handler (metadata) | Handler | P1 |
| JWKS handler | Handler | P1 |
| OAuth routes (public + internal) | Routes | P0 |
| Scope-to-permission mapping | Service | P1 |
| Consent challenge API (GET + POST /oauth/consent) | Handler | P0 |

### Identity Provider, Client & OAuth 2.0 Association

OAuth 2.0 in maintainerd-auth is **not a standalone system** — it is the authorization layer for the **builtin identity provider**. Every OAuth 2.0 resource (authorization code, refresh token, consent grant, consent challenge) is linked back to the existing identity provider and client hierarchy through foreign key relationships.

#### Entity Hierarchy

```
Tenant
 └── Identity Provider  (builtin: provider = "internal", is_system = true)
       └── Client  (traditional, spa, mobile, m2m — each with OAuth 2.0 fields)
             ├── OAuth Authorization Codes   (issued during authorize flow)
             ├── OAuth Refresh Tokens        (issued during token exchange)
             ├── OAuth Consent Grants        (stored per user-client pair)
             ├── OAuth Consent Challenges    (pending consent decisions)
             └── Client URIs                 (registered redirect URIs)
```

Every OAuth table (`oauth_authorization_codes`, `oauth_refresh_tokens`, `oauth_consent_grants`, `oauth_consent_challenges`) contains a `client_id` foreign key that references `clients.client_id`. The `clients` table in turn references `identity_providers.identity_provider_id`, forming a chain from any OAuth artifact all the way back to its tenant and identity provider.

#### Builtin Identity Provider

During tenant setup (seeder `005_identity_provider`), a **builtin identity provider** is created with:
- `provider = "internal"` — the system's own username/password authentication
- `provider_type = "identity"` — a primary authentication source
- `is_system = true`, `is_default = true` — cannot be deleted, used as the default

This builtin IDP owns the default clients that are created by seeder `006_client`:

| Default Client | Client Type | Auth Method | Grant Types | Consent |
|---|---|---|---|---|
| `traditional-default` | `traditional` | `client_secret_basic` | `authorization_code`, `refresh_token` | No (first-party) |
| `spa-default` | `spa` | `none` | `authorization_code`, `refresh_token` | No (first-party) |
| `mobile-default` | `mobile` | `none` | `authorization_code`, `refresh_token` | No (first-party) |
| `m2m-default` | `m2m` | `client_secret_basic` | `client_credentials` | No (first-party) |

All default clients are `is_system = true` and skip consent because they are first-party applications.

#### User Identity Binding

Identity records are created **during user registration and login** — never by the OAuth 2.0 layer. When a user registers or logs in via an identity provider, a `user_identities` record is created linking:
- `user_id` → the authenticated user
- `client_id` → the client that initiated the session
- `provider` → `"default"` for builtin authentication (or the external provider name)
- `sub` → a stable, unique identifier used as the `sub` claim in OAuth tokens

A single user can have **multiple identities** from different providers (e.g., builtin, Google, Facebook, Auth0, Cognito). Each identity has its own `sub`, and the same provider can appear more than once when the user authenticates through distinct instances of that provider (e.g., two Cognito pools).

The `users.user_uuid` field is for **internal management only** and is never used as an authentication identifier. Users are always identified by `user_identities.sub` in authentication contexts.

**OAuth 2.0 is read-only with respect to identities.** The token service resolves `sub` via `user_identities.FindByUserIDAndClientID()` and returns an error if no identity exists — it never creates or modifies identity records. If a user has not registered or logged in through a given client, the authorization code exchange will fail. This enforces the separation of concerns: registration/login provisions identities, OAuth 2.0 authorizes access using them.

#### External Identity Providers (Future)

External IDPs (Cognito, Google, Auth0) have their own identity provider records and clients. When federated login is implemented, those flows will also produce `user_identities` records (with `provider` set to the external provider name). The OAuth 2.0 authorization server will handle the code/token exchange for all clients regardless of their IDP, but the authentication step will differ — builtin IDP uses username/password, external IDPs use federated redirect.

---

## 12. Implementation Approach

### Architecture

Following the project's established layering: **Handler → Service → Repository → DB**.

Since maintainerd-auth is API-only, the authorization endpoint does not render any HTML pages. Instead, it issues HTTP redirects to the frontend for login and consent, and exposes JSON endpoints that the frontend calls to retrieve consent details and submit decisions.

```
                            ┌─────────────────────────────┐
                            │     Public Port (8081)       │
                            │                             │
                            │  GET  /oauth/authorize      │
                            │  POST /oauth/token          │
                            │  POST /oauth/revoke         │
                            │  GET  /oauth/consent        │
                            │  POST /oauth/consent        │
                            │  GET  /oauth/userinfo       │
                            │  GET  /.well-known/*        │
                            └──────────┬──────────────────┘
                                       │
                            ┌──────────▼──────────────────┐
                            │    Internal Port (8080)      │
                            │                             │
                            │  POST /oauth/introspect     │
                            └──────────┬──────────────────┘
                                       │
              ┌────────────────────────┼────────────────────────┐
              ▼                        ▼                        ▼
   ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
   │ AuthorizeHandler │    │  TokenHandler   │    │ RevocationHandler│
   └────────┬────────┘    └────────┬────────┘    └────────┬────────┘
            │                      │                      │
   ┌────────▼────────┐    ┌────────▼────────┐    ┌────────▼────────┐
   │AuthorizeService  │    │  TokenService   │    │  TokenService   │
   │ (consent + code) │    │ (grant routing) │    │  (revocation)   │
   └────────┬────────┘    └────────┬────────┘    └────────┬────────┘
            │                      │                      │
   ┌────────▼──────────────────────▼──────────────────────▼────────┐
   │                     Repositories                              │
   │  AuthorizationCodeRepo │ ConsentRepo │ OAuthTokenRepo │ ...   │
   └────────────────────────┬──────────────────────────────────────┘
                            │
                       ┌────▼────┐
                       │   DB    │
                       └─────────┘
```

### Service Interfaces

```go
// AuthorizeService handles the authorization endpoint logic.
// This service never renders HTML — it returns redirect URLs or JSON responses
// that the frontend (a separate application) consumes.
type AuthorizeService interface {
    // Authorize processes an authorization request, validates parameters,
    // and either returns a redirect URI (code issued) or a redirect to the
    // frontend consent/login page.
    Authorize(ctx context.Context, req AuthorizeRequest) (AuthorizeResponse, error)
    
    // GetConsentChallenge returns the details needed by the frontend to render
    // a consent screen (client info, requested scopes, etc.)
    GetConsentChallenge(ctx context.Context, challengeID string) (ConsentChallenge, error)
    
    // HandleConsent processes the user's consent decision submitted by the frontend
    HandleConsent(ctx context.Context, req ConsentDecision) (AuthorizeResponse, error)
}

// TokenService handles token issuance, refresh, and lifecycle
type TokenService interface {
    // Exchange processes a token request for any supported grant type
    Exchange(ctx context.Context, req TokenRequest) (TokenResponse, error)
    
    // Revoke invalidates a token
    Revoke(ctx context.Context, req RevocationRequest) error
    
    // Introspect returns metadata about a token
    Introspect(ctx context.Context, req IntrospectionRequest) (IntrospectionResponse, error)
}

// ConsentService manages user consent grants
type ConsentService interface {
    // HasConsent checks if user has previously consented to this client+scopes
    HasConsent(ctx context.Context, userID, clientID string, scopes []string) (bool, error)
    
    // GrantConsent records a consent decision
    GrantConsent(ctx context.Context, userID, clientID string, scopes []string) error
    
    // RevokeConsent revokes consent for a client, also revoking associated tokens
    RevokeConsent(ctx context.Context, userID, clientID string) error
}
```

### Token Exchange Flow (TokenService.Exchange)

```go
func (s *tokenService) Exchange(ctx, req) (TokenResponse, error) {
    switch req.GrantType {
    case "authorization_code":
        return s.exchangeAuthorizationCode(ctx, req)
    case "client_credentials":
        return s.exchangeClientCredentials(ctx, req)
    case "refresh_token":
        return s.exchangeRefreshToken(ctx, req)
    default:
        return TokenResponse{}, NewOAuthError("unsupported_grant_type", ...)
    }
}
```

### OAuth Error Response Format

Per RFC 6749 §5.2, all OAuth errors use a standard JSON format:

```json
{
  "error": "invalid_request",
  "error_description": "The code_verifier parameter is required",
  "error_uri": "https://docs.example.com/oauth/errors#invalid_request"
}
```

We create an `OAuthError` type separate from `apperror` since OAuth errors have specific format requirements (no HTTP 500 details leaked, specific error codes per endpoint).

---

## 13. Database Schema

### New Tables

#### `oauth_authorization_codes`

Stores pending authorization codes (short-lived, deleted after use or expiry).

```sql
CREATE TABLE oauth_authorization_codes (
    id                    BIGSERIAL PRIMARY KEY,
    code                  TEXT NOT NULL UNIQUE,       -- hashed authorization code
    client_id             UUID NOT NULL REFERENCES clients(client_id),
    user_id               UUID NOT NULL REFERENCES users(user_id),
    tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id),
    redirect_uri          TEXT NOT NULL,
    scope                 TEXT NOT NULL DEFAULT '',
    state                 TEXT,
    code_challenge        TEXT NOT NULL,              -- PKCE challenge
    code_challenge_method TEXT NOT NULL DEFAULT 'S256',
    nonce                 TEXT,                        -- OIDC nonce passthrough
    expires_at            TIMESTAMPTZ NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_code_challenge_method CHECK (code_challenge_method IN ('S256'))
);

CREATE INDEX idx_oauth_auth_codes_code ON oauth_authorization_codes(code);
CREATE INDEX idx_oauth_auth_codes_expires ON oauth_authorization_codes(expires_at);
```

#### `oauth_refresh_tokens`

Stores refresh tokens with family tracking for rotation/reuse detection.

```sql
CREATE TABLE oauth_refresh_tokens (
    id                BIGSERIAL PRIMARY KEY,
    token_id          UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    token_hash        TEXT NOT NULL UNIQUE,           -- SHA-256 hash of token
    family_id         UUID NOT NULL,                  -- links rotated tokens
    client_id         UUID NOT NULL REFERENCES clients(client_id),
    user_id           UUID NOT NULL REFERENCES users(user_id),
    tenant_id         UUID NOT NULL REFERENCES tenants(tenant_id),
    scope             TEXT NOT NULL DEFAULT '',
    is_revoked        BOOLEAN NOT NULL DEFAULT FALSE,
    revoked_at        TIMESTAMPTZ,
    expires_at        TIMESTAMPTZ NOT NULL,
    last_used_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_revoked_at CHECK (
        (is_revoked = FALSE AND revoked_at IS NULL) OR
        (is_revoked = TRUE AND revoked_at IS NOT NULL)
    )
);

CREATE INDEX idx_oauth_refresh_token_hash ON oauth_refresh_tokens(token_hash);
CREATE INDEX idx_oauth_refresh_family ON oauth_refresh_tokens(family_id);
CREATE INDEX idx_oauth_refresh_user_client ON oauth_refresh_tokens(user_id, client_id);
CREATE INDEX idx_oauth_refresh_expires ON oauth_refresh_tokens(expires_at);
```

#### `oauth_consent_grants`

Stores user consent decisions per client.

```sql
CREATE TABLE oauth_consent_grants (
    id          BIGSERIAL PRIMARY KEY,
    grant_id    UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    user_id     UUID NOT NULL REFERENCES users(user_id),
    client_id   UUID NOT NULL REFERENCES clients(client_id),
    tenant_id   UUID NOT NULL REFERENCES tenants(tenant_id),
    scopes      TEXT NOT NULL DEFAULT '',              -- space-delimited granted scopes
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT uq_consent_user_client UNIQUE (user_id, client_id)
);

CREATE INDEX idx_oauth_consent_user ON oauth_consent_grants(user_id);
```

#### `oauth_consent_challenges`

Stores pending consent challenges. Created when the authorization endpoint determines that consent is needed, and consumed when the frontend submits the user's decision. Short-lived (10 minutes).

```sql
CREATE TABLE oauth_consent_challenges (
    id                    BIGSERIAL PRIMARY KEY,
    challenge_id          UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    client_id             UUID NOT NULL REFERENCES clients(client_id),
    user_id               UUID NOT NULL REFERENCES users(user_id),
    tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id),
    redirect_uri          TEXT NOT NULL,
    scope                 TEXT NOT NULL DEFAULT '',
    state                 TEXT,
    code_challenge        TEXT NOT NULL,
    code_challenge_method TEXT NOT NULL DEFAULT 'S256',
    nonce                 TEXT,
    expires_at            TIMESTAMPTZ NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_challenge_method CHECK (code_challenge_method IN ('S256'))
);

CREATE INDEX idx_oauth_consent_challenges_id ON oauth_consent_challenges(challenge_id);
CREATE INDEX idx_oauth_consent_challenges_expires ON oauth_consent_challenges(expires_at);
```

#### `oauth_access_tokens` (Optional — for introspection/revocation of JWT tokens)

If we need server-side access token tracking (for revocation of JWTs before expiry):

```sql
CREATE TABLE oauth_access_tokens (
    id          BIGSERIAL PRIMARY KEY,
    jti         TEXT NOT NULL UNIQUE,                  -- JWT ID claim
    client_id   UUID NOT NULL REFERENCES clients(client_id),
    user_id     UUID REFERENCES users(user_id),        -- NULL for client_credentials
    tenant_id   UUID NOT NULL REFERENCES tenants(tenant_id),
    scope       TEXT NOT NULL DEFAULT '',
    is_revoked  BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth_access_jti ON oauth_access_tokens(jti);
CREATE INDEX idx_oauth_access_expires ON oauth_access_tokens(expires_at);
```

**Alternative**: Use a Redis-based revocation list (JTI blacklist) instead of a DB table, since access tokens are short-lived. Store revoked JTIs in Redis with TTL matching the token's remaining lifetime.

### OAuth 2.0 Fields on the `clients` Table

The OAuth 2.0 fields are **part of the original `clients` table definition** (migration `017_create_clients_table`), not a later alteration. This reflects the design principle that OAuth 2.0 capability is intrinsic to every client registered under the builtin identity provider.

| Column | Type | Default | Description |
|---|---|---|---|
| `token_endpoint_auth_method` | `VARCHAR(30)` | `client_secret_basic` | How this client authenticates at the token endpoint |
| `grant_types` | `TEXT[]` | `{authorization_code}` | Allowed OAuth grant types |
| `response_types` | `TEXT[]` | `{code}` | Allowed OAuth response types |
| `access_token_ttl` | `INTEGER` | `NULL` | Override default access token lifetime (seconds) |
| `refresh_token_ttl` | `INTEGER` | `NULL` | Override default refresh token lifetime (seconds) |
| `require_consent` | `BOOLEAN` | `TRUE` | Whether to prompt user for consent |

A CHECK constraint (`chk_clients_token_auth_method`) ensures `token_endpoint_auth_method` is one of `client_secret_basic`, `client_secret_post`, or `none`. A GIN index on `grant_types` supports efficient array lookups.

---

## 14. Endpoint Specifications

### GET /oauth/authorize

**Request**: Query parameters per §3.1.

**Flow**:
1. Validate all required parameters (`response_type`, `client_id`, `code_challenge`)
2. Look up client; verify it supports `authorization_code` grant type
3. Validate `redirect_uri` against registered URIs (exact string match)
4. Validate `scope` against client's allowed scopes
5. Validate `code_challenge_method` is `S256`
6. If user is not authenticated → redirect user-agent to the client's `LoginURI` (frontend) with a `login_challenge` parameter so the frontend can authenticate the user, then retry
7. If user is authenticated → check consent
8. If consent exists → issue authorization code and redirect to `redirect_uri`
9. If consent needed → create a consent challenge record and redirect user-agent to the frontend consent URL with a `consent_challenge` parameter
10. (Consent resolution happens via `GET/POST /oauth/consent` below)

**Error handling**: If `redirect_uri` is invalid or missing, return a JSON error response (do NOT redirect). For all other errors, redirect to `redirect_uri` with error parameters.

### GET /oauth/consent

**Purpose**: Called by the frontend to retrieve consent challenge details for rendering the consent screen.

**Request**: Query parameter `consent_challenge`.

**Flow**:
1. Validate the `consent_challenge` parameter
2. Look up the pending consent challenge — verify not expired
3. Return JSON with: client name, client logo, requested scopes (with human-readable descriptions), the user context
4. If challenge is invalid or expired → return error JSON

**Response**: JSON body (never HTML).

### POST /oauth/consent

**Purpose**: Called by the frontend to submit the user's consent decision.

**Request**: JSON body with `consent_challenge`, `decision` (`accept` | `reject`), and optionally `granted_scopes` (subset of requested scopes).

**Flow**:
1. Validate the consent challenge — verify not expired, belongs to the authenticated user
2. If `decision == accept`:
   - Store consent grant in DB
   - Issue authorization code
   - Return JSON with `redirect_to` URL (the client's `redirect_uri` with `code` and `state`)
3. If `decision == reject`:
   - Return JSON with `redirect_to` URL (the client's `redirect_uri` with `error=access_denied`)
4. Log consent auth event
5. Delete the consent challenge record

### POST /oauth/token

**Request**: Form-encoded body per §3.2.

**Flow** (per grant type):

**`authorization_code`**:
1. Authenticate client (if confidential)
2. Look up authorization code → validate not expired, not already used
3. Verify `client_id` matches
4. Verify `code_verifier` against stored `code_challenge` (PKCE)
5. Mark code as used (delete from DB)
6. Issue access token + refresh token + id_token (if `openid` scope)
7. Log auth event

**`client_credentials`**:
1. Authenticate client (REQUIRED — must be confidential)
2. Validate requested scopes against client's pre-approved scopes
3. Issue access token only (no refresh token, no id_token)
4. Log auth event

**`refresh_token`**:
1. Authenticate client (if confidential) or validate `client_id` (if public)
2. Look up refresh token by hash → validate not expired, not revoked
3. Verify `client_id` matches
4. If already used (rotation detection) → revoke entire family, return error
5. Issue new access token + new refresh token (rotation)
6. Invalidate old refresh token
7. Log auth event

**Response**: JSON with `Cache-Control: no-store`.

### POST /oauth/revoke

**Request**: Form-encoded with `token` and optional `token_type_hint`.

**Flow**:
1. Authenticate client
2. Determine token type (try hint first, then search all types)
3. If refresh token → revoke it and all associated access tokens
4. If access token → add JTI to revocation list (Redis) or mark in DB
5. Return 200 OK (even if token was invalid)

### POST /oauth/introspect

**Request**: Form-encoded with `token` and optional `token_type_hint`.

**Flow**:
1. Authenticate caller (client credentials or internal service token)
2. If JWT access token → validate signature, check revocation list
3. If refresh token → look up in DB
4. Return JSON response with `active` boolean and metadata

### GET /oauth/userinfo

**Request**: Bearer token in Authorization header.

**Flow**:
1. Validate access token (JWT signature + expiry + revocation check)
2. Check `openid` scope is present
3. Return user profile claims based on granted scopes (`profile`, `email`, `phone`, `address`)

### GET /.well-known/openid-configuration

**Response**: JSON metadata document per RFC 8414. Dynamically generated based on tenant configuration.

### GET /.well-known/jwks.json

**Response**: JWK Set containing the public signing key(s). Key rotation support via `kid`.

---

## 15. Implementation Phases

### Phase 1 — Core Authorization Code Flow (P0)

1. Database migrations (authorization codes, refresh tokens, consent grants, consent challenges, client OAuth fields)
2. Models and repositories for new tables
3. PKCE validation utilities
4. OAuth error response type
5. Authorization service (authorize flow + login/consent challenge creation)
6. Consent challenge service (get challenge details, accept/reject decision)
7. Token service (authorization_code grant + refresh_token grant)
8. Consent service
9. Authorization handler + Consent handler + Token handler + Revocation handler
10. OAuth routes on public port
11. Wire into app layer

### Phase 2 — Discovery & Metadata (P1)

1. JWKS handler (expose public key)
2. Discovery handler (metadata document)
3. UserInfo handler
4. Client credentials grant type in token service
5. Introspection handler + route on internal port

### Phase 3 — Hardening & Polish (P1)

1. Token revocation via Redis JTI blacklist
2. Refresh token family-based reuse detection
3. Rate limiting on OAuth endpoints
4. CORS configuration for token/userinfo endpoints
5. Wire all OAuth events to auth events system
6. Scope-to-permission mapping service
7. Consent management API (list/revoke consents)

### Phase 4 — Advanced Features (P2, Future)

1. Device Authorization Grant (RFC 8628)
2. Pushed Authorization Requests (RFC 9126)
3. DPoP sender-constrained tokens (RFC 9449)
4. Per-tenant signing key isolation
5. Dynamic Client Registration (RFC 7591)

---

## References

- [RFC 6749 — The OAuth 2.0 Authorization Framework](https://datatracker.ietf.org/doc/html/rfc6749)
- [RFC 6750 — Bearer Token Usage](https://datatracker.ietf.org/doc/html/rfc6750)
- [RFC 7009 — OAuth 2.0 Token Revocation](https://datatracker.ietf.org/doc/html/rfc7009)
- [RFC 7519 — JSON Web Token (JWT)](https://datatracker.ietf.org/doc/html/rfc7519)
- [RFC 7636 — Proof Key for Code Exchange (PKCE)](https://datatracker.ietf.org/doc/html/rfc7636)
- [RFC 7662 — OAuth 2.0 Token Introspection](https://datatracker.ietf.org/doc/html/rfc7662)
- [RFC 8414 — Authorization Server Metadata](https://datatracker.ietf.org/doc/html/rfc8414)
- [RFC 9068 — JWT Profile for OAuth 2.0 Access Tokens](https://datatracker.ietf.org/doc/html/rfc9068)
- [OAuth 2.1 Draft](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1)
- [OAuth Security BCP](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-security-topics)
- [ory/fosite — Extensible OAuth 2.0 SDK for Go](https://github.com/ory/fosite)
