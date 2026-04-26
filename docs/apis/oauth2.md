# OAuth 2.0 API Reference

Complete API documentation for all OAuth 2.0 and OpenID Connect endpoints in maintainerd-auth.

---

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Response Formats](#response-formats)
4. [Error Handling](#error-handling)
5. [Endpoints](#endpoints)
   - [Authorization Endpoint](#1-authorization-endpoint)
   - [Get Consent Challenge](#2-get-consent-challenge)
   - [Submit Consent Decision](#3-submit-consent-decision)
   - [Token Endpoint](#4-token-endpoint)
   - [Token Revocation](#5-token-revocation)
   - [Token Introspection](#6-token-introspection)
   - [UserInfo Endpoint](#7-userinfo-endpoint)
   - [List Consent Grants](#8-list-consent-grants)
   - [Revoke Consent Grant](#9-revoke-consent-grant)
   - [OpenID Discovery](#10-openid-discovery)
   - [JWKS Endpoint](#11-jwks-endpoint)
6. [Flows](#flows)
   - [Authorization Code Flow (with PKCE)](#authorization-code-flow-with-pkce)
   - [Authorization Code Flow (with Consent)](#authorization-code-flow-with-consent)
   - [Client Credentials Flow](#client-credentials-flow)
   - [Refresh Token Flow](#refresh-token-flow)
   - [Token Revocation Flow](#token-revocation-flow)
   - [Consent Management Flow](#consent-management-flow)

---

## Overview

| Property | Value |
|---|---|
| Base URL (public) | `https://{host}/api/v1` (port 8081) |
| Base URL (internal) | `https://{host}/api/v1` (port 8080, VPN-only) |
| Protocol | OAuth 2.1 (RFC 6749 + PKCE mandatory) |
| Token format | JWT (RS256) for access/ID tokens; opaque for refresh tokens |
| PKCE | **Required** for all authorization code flows (`S256` only) |
| Content types | `application/x-www-form-urlencoded` (token/revoke/introspect), `application/json` (consent/authorize responses) |

### Ports

| Port | Purpose | Access |
|---|---|---|
| **8081** | Identity / public endpoints | Public (via load balancer) |
| **8080** | Management / internal endpoints | VPN-only |

---

## Authentication

### JWT Bearer Token

Endpoints marked **JWT Required** expect an `Authorization` header:

```
Authorization: Bearer <access_token>
```

The JWT must be issued by this authorization server, signed with RS256, and not expired.

### Client Authentication

The token, revocation, and introspection endpoints authenticate the client using one of these methods:

#### HTTP Basic Authentication (`client_secret_basic`)

```
Authorization: Basic base64(client_id:client_secret)
```

#### POST Body (`client_secret_post`)

```
client_id=my-client&client_secret=my-secret
```

#### None (`none`)

Public clients (SPAs, mobile apps) provide only `client_id` with no secret:

```
client_id=my-spa-client
```

> HTTP Basic authentication takes precedence over POST body credentials when both are present.

---

## Response Formats

### Standard API Response

Endpoints using the standard application response wrapper return:

```json
{
  "success": true,
  "data": { ... },
  "message": "Description of the result"
}
```

Error variant:

```json
{
  "success": false,
  "error": "Error description"
}
```

### OAuth Token Response

The `/oauth/token` endpoint returns the raw OAuth 2.0 JSON format (not wrapped):

```json
{
  "access_token": "eyJhbGciOi...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "dGhpcyBpcyBh...",
  "id_token": "eyJhbGciOi...",
  "scope": "openid profile email"
}
```

### OAuth Error Response

OAuth-specific errors follow RFC 6749 §5.2:

```json
{
  "error": "invalid_request",
  "error_description": "Human-readable explanation"
}
```

All OAuth error responses include these headers:

```
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache
```

---

## Error Handling

### OAuth Error Codes

| Error Code | HTTP Status | Description |
|---|---|---|
| `invalid_request` | 400 | Malformed request, missing required parameters |
| `unauthorized_client` | 401 | Client not authorized for requested grant type |
| `access_denied` | 403 | Resource owner or server denied the request |
| `unsupported_response_type` | 400 | Response type not supported by client |
| `invalid_scope` | 400 | Requested scope is invalid or unknown |
| `server_error` | 500 | Unexpected internal error |
| `invalid_grant` | 400 | Authorization code, refresh token, or credential is invalid/expired/revoked |
| `unsupported_grant_type` | 400 | Grant type not supported |
| `invalid_client` | 401 | Client authentication failed |
| `login_required` | 401 | User is not authenticated |
| `consent_required` | 403 | User consent needed but not given |

### Standard API Error Codes

Non-OAuth endpoints return standard HTTP errors:

| HTTP Status | Meaning |
|---|---|
| 400 | Bad Request — invalid input or validation failure |
| 401 | Unauthorized — missing or invalid JWT |
| 404 | Not Found — resource does not exist |
| 500 | Internal Server Error |

---

## Endpoints

---

### 1. Authorization Endpoint

Initiates the OAuth 2.0 Authorization Code flow. The user must be already authenticated.

| Property | Value |
|---|---|
| **URL** | `GET /api/v1/oauth/authorize` |
| **Port** | 8081 (public) |
| **Auth** | JWT Required |
| **Content-Type** | N/A (query parameters) |

#### Query Parameters

| Parameter | Required | Description |
|---|---|---|
| `response_type` | Yes | Must be `code` |
| `client_id` | Yes | The client identifier (max 255 chars) |
| `redirect_uri` | Yes | Registered redirect URI (max 2048 chars) |
| `code_challenge` | Yes | PKCE code challenge (43–128 chars, base64url-encoded SHA-256) |
| `code_challenge_method` | Yes | Must be `S256` |
| `scope` | No | Space-delimited scopes (max 1024 chars). E.g., `openid profile email` |
| `state` | No | Opaque value for CSRF protection (max 512 chars, recommended) |
| `nonce` | No | String to associate with the ID token (max 512 chars) |

#### Response — Authorization Code Issued (no consent needed)

```http
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "success": true,
  "data": {
    "redirect_uri": "https://app.example.com/callback?code=SplxlOBeZQQYbYS6WxSbIA&state=xyz"
  },
  "message": "Authorization successful"
}
```

The frontend should redirect the user-agent to the `redirect_uri` value.

#### Response — Consent Required

```http
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "success": true,
  "data": {
    "consent_challenge": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
  },
  "message": "Consent required"
}
```

The frontend should redirect the user to the consent screen using the `consent_challenge` identifier.

#### Error Responses

**Missing required parameter:**

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json
```

```json
{
  "success": false,
  "error": "Validation failed",
  "details": {
    "code_challenge": "code_challenge is required",
    "code_challenge_method": "code_challenge_method is required"
  }
}
```

**Invalid client:**

```json
{
  "error": "invalid_request",
  "error_description": "unknown or inactive client_id"
}
```

**Client not authorized for grant type:**

```json
{
  "error": "unauthorized_client",
  "error_description": "client is not authorized for authorization_code grant"
}
```

**Invalid redirect URI:**

```json
{
  "error": "invalid_request",
  "error_description": "redirect_uri does not match any registered URI"
}
```

**Unauthenticated:**

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json
```

```json
{
  "success": false,
  "error": "Authentication required"
}
```

#### cURL Example

```bash
curl -X GET \
  'https://auth.example.com/api/v1/oauth/authorize?response_type=code&client_id=spa-default&redirect_uri=https://app.example.com/callback&scope=openid%20profile%20email&state=abc123&code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&code_challenge_method=S256' \
  -H 'Authorization: Bearer eyJhbGciOi...'
```

---

### 2. Get Consent Challenge

Retrieves the details of a pending consent challenge so the frontend can render the consent screen.

| Property | Value |
|---|---|
| **URL** | `GET /api/v1/oauth/consent/{challenge_id}` |
| **Port** | 8081 (public) |
| **Auth** | JWT Required |
| **Content-Type** | N/A |

#### Path Parameters

| Parameter | Required | Description |
|---|---|---|
| `challenge_id` | Yes | The consent challenge UUID returned by the authorize endpoint |

#### Response — Success

```http
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "success": true,
  "data": {
    "challenge_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "client_name": "My SPA Application",
    "client_uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "scopes": ["openid", "profile", "email"],
    "redirect_uri": "https://app.example.com/callback",
    "expires_at": 1700001000
  },
  "message": "Consent challenge retrieved"
}
```

#### Error Responses

**Invalid challenge ID format:**

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json
```

```json
{
  "success": false,
  "error": "Invalid challenge ID"
}
```

**Challenge not found or expired:**

```http
HTTP/1.1 404 Not Found
Content-Type: application/json
```

```json
{
  "success": false,
  "error": "Failed to retrieve consent challenge"
}
```

**Challenge belongs to another user:**

```http
HTTP/1.1 403 Forbidden
Content-Type: application/json
```

```json
{
  "success": false,
  "error": "Failed to retrieve consent challenge"
}
```

#### cURL Example

```bash
curl -X GET \
  'https://auth.example.com/api/v1/oauth/consent/f47ac10b-58cc-4372-a567-0e02b2c3d479' \
  -H 'Authorization: Bearer eyJhbGciOi...'
```

---

### 3. Submit Consent Decision

Processes the user's consent decision (approve or deny) and returns a redirect URI.

| Property | Value |
|---|---|
| **URL** | `POST /api/v1/oauth/consent` |
| **Port** | 8081 (public) |
| **Auth** | JWT Required |
| **Content-Type** | `application/json` |

#### Request Body

| Field | Type | Required | Description |
|---|---|---|---|
| `challenge_id` | string (UUID) | Yes | The consent challenge UUID |
| `approved` | boolean | Yes | `true` to approve, `false` to deny |

#### Request — Approve Consent

```json
{
  "challenge_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "approved": true
}
```

#### Response — Approved

```http
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "success": true,
  "data": {
    "redirect_uri": "https://app.example.com/callback?code=SplxlOBeZQQYbYS6WxSbIA&state=xyz"
  },
  "message": "Consent processed"
}
```

#### Request — Deny Consent

```json
{
  "challenge_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "approved": false
}
```

#### Response — Denied

```http
HTTP/1.1 403 Forbidden
Content-Type: application/json
```

```json
{
  "error": "access_denied",
  "error_description": "the user denied the consent request"
}
```

#### Error Responses

**Invalid request body:**

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json
```

```json
{
  "success": false,
  "error": "Invalid request body"
}
```

**Missing or invalid challenge_id:**

```json
{
  "success": false,
  "error": "Validation failed",
  "details": {
    "challenge_id": "challenge_id must be a valid UUID"
  }
}
```

**Challenge expired or not found:**

```json
{
  "error": "invalid_request",
  "error_description": "consent challenge has expired"
}
```

#### cURL Example

```bash
curl -X POST \
  'https://auth.example.com/api/v1/oauth/consent' \
  -H 'Authorization: Bearer eyJhbGciOi...' \
  -H 'Content-Type: application/json' \
  -d '{
    "challenge_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "approved": true
  }'
```

---

### 4. Token Endpoint

Exchanges an authorization code, refresh token, or client credentials for tokens.

| Property | Value |
|---|---|
| **URL** | `POST /api/v1/oauth/token` |
| **Port** | 8081 (public) |
| **Auth** | Client authentication (Basic, POST body, or none) |
| **Content-Type** | `application/x-www-form-urlencoded` |

---

#### 4a. Grant Type: `authorization_code`

Exchanges an authorization code (obtained from the authorize endpoint) for tokens.

##### Request Parameters

| Parameter | Required | Description |
|---|---|---|
| `grant_type` | Yes | `authorization_code` |
| `code` | Yes | The authorization code |
| `redirect_uri` | Yes | Must match the redirect_uri used in the authorize request |
| `code_verifier` | Yes | PKCE code verifier (43–128 chars, base64url `[A-Za-z0-9\-._~]`) |
| `client_id` | Yes* | Client identifier (*not needed if using HTTP Basic auth) |
| `client_secret` | Conditional | Required for confidential clients (`client_secret_basic` or `client_secret_post`) |

##### Response — Success

```http
HTTP/1.1 200 OK
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache
```

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6Im1haW50YWluZXJkLWF1dGgta2V5LTEifQ.eyJpc3MiOiJodHRwczovL2F1dGguZXhhbXBsZS5jb20iLCJzdWIiOiJ1c2VyLXN1Yi0xMjMiLCJhdWQiOlsiaHR0cHM6Ly9hcGkuZXhhbXBsZS5jb20iXSwiZXhwIjoxNzAwMDAwOTAwLCJpYXQiOjE2OTk5OTkxMDAsIm5iZiI6MTY5OTk5OTEwMCwianRpIjoiYWJjZDEyMzQiLCJjbGllbnRfaWQiOiJzcGEtZGVmYXVsdCIsInNjb3BlIjoib3BlbmlkIHByb2ZpbGUgZW1haWwiLCJ0ZW5hbnRfaWQiOiJ0ZW5hbnQtdXVpZCJ9.signature",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "dGhpcyBpcyBhIGhpZ2gtZW50cm9weSByYW5kb20gcmVmcmVzaCB0b2tlbg",
  "id_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL2F1dGguZXhhbXBsZS5jb20iLCJzdWIiOiJ1c2VyLXN1Yi0xMjMiLCJhdWQiOiJzcGEtZGVmYXVsdCIsImV4cCI6MTcwMDAwMDkwMCwiaWF0IjoxNjk5OTk5MTAwLCJub25jZSI6ImFiYzEyMyJ9.signature",
  "scope": "openid profile email"
}
```

##### Access Token JWT Claims

```json
{
  "iss": "https://auth.example.com",
  "sub": "user-identity-sub-123",
  "aud": ["https://api.example.com"],
  "exp": 1700000900,
  "iat": 1699999100,
  "nbf": 1699999100,
  "jti": "unique-token-id",
  "client_id": "spa-default",
  "scope": "openid profile email",
  "tenant_id": "tenant-uuid"
}
```

> The `sub` claim is resolved from `user_identities.sub` — it is the stable identity identifier tied to the user and client. It is **not** the `users.user_uuid`.

##### Error Responses

**Invalid authorization code:**

```json
{
  "error": "invalid_grant",
  "error_description": "authorization code is invalid or has expired"
}
```

**Authorization code already used (replay attack):**

```json
{
  "error": "invalid_grant",
  "error_description": "authorization code has already been used"
}
```

**PKCE verification failed:**

```json
{
  "error": "invalid_grant",
  "error_description": "PKCE verification failed"
}
```

**Client mismatch:**

```json
{
  "error": "invalid_grant",
  "error_description": "client_id does not match the authorization code"
}
```

**Redirect URI mismatch:**

```json
{
  "error": "invalid_grant",
  "error_description": "redirect_uri does not match the authorization request"
}
```

**Client authentication failed:**

```json
{
  "error": "invalid_client",
  "error_description": "client authentication failed"
}
```

**No user identity found:**

```json
{
  "error": "server_error",
  "error_description": "an unexpected error occurred"
}
```

##### cURL Example — Public Client (SPA)

```bash
curl -X POST \
  'https://auth.example.com/api/v1/oauth/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=authorization_code&code=SplxlOBeZQQYbYS6WxSbIA&redirect_uri=https://app.example.com/callback&code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk&client_id=spa-default'
```

##### cURL Example — Confidential Client (HTTP Basic)

```bash
curl -X POST \
  'https://auth.example.com/api/v1/oauth/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -H 'Authorization: Basic dHJhZGl0aW9uYWwtZGVmYXVsdDpteS1zZWNyZXQ=' \
  -d 'grant_type=authorization_code&code=SplxlOBeZQQYbYS6WxSbIA&redirect_uri=https://app.example.com/callback&code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk'
```

##### cURL Example — Confidential Client (POST Body)

```bash
curl -X POST \
  'https://auth.example.com/api/v1/oauth/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=authorization_code&code=SplxlOBeZQQYbYS6WxSbIA&redirect_uri=https://app.example.com/callback&code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk&client_id=traditional-default&client_secret=my-secret'
```

---

#### 4b. Grant Type: `refresh_token`

Exchanges a refresh token for a new token set. The previous refresh token is invalidated (rotation).

##### Request Parameters

| Parameter | Required | Description |
|---|---|---|
| `grant_type` | Yes | `refresh_token` |
| `refresh_token` | Yes | The refresh token |
| `scope` | No | Space-delimited scopes (must be subset of originally granted scopes) |
| `client_id` | Yes* | Client identifier (*not needed if using HTTP Basic auth) |
| `client_secret` | Conditional | Required for confidential clients |

##### Response — Success

```http
HTTP/1.1 200 OK
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache
```

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "bmV3LXJlZnJlc2gtdG9rZW4tYWZ0ZXItcm90YXRpb24",
  "id_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "scope": "openid profile email"
}
```

> **Refresh token rotation**: Each use generates a new refresh token. The previous token is invalidated immediately. Store the new `refresh_token` from each response.

##### Error Responses

**Refresh token not found or invalid:**

```json
{
  "error": "invalid_grant",
  "error_description": "refresh token is invalid"
}
```

**Refresh token reuse detected (family revoked):**

```json
{
  "error": "invalid_grant",
  "error_description": "token reuse detected; token family revoked"
}
```

When reuse of an already-rotated refresh token is detected, the entire token family is revoked as a security measure. The user must re-authenticate.

**Refresh token expired:**

```json
{
  "error": "invalid_grant",
  "error_description": "refresh token has expired"
}
```

**Client mismatch:**

```json
{
  "error": "invalid_grant",
  "error_description": "client_id does not match the refresh token"
}
```

**Scope exceeds original grant:**

```json
{
  "error": "invalid_scope",
  "error_description": "requested scope exceeds the original grant"
}
```

##### cURL Example

```bash
curl -X POST \
  'https://auth.example.com/api/v1/oauth/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=refresh_token&refresh_token=dGhpcyBpcyBhIGhpZ2gtZW50cm9weSByYW5kb20gcmVmcmVzaCB0b2tlbg&client_id=spa-default'
```

##### cURL Example — With Scope Narrowing

```bash
curl -X POST \
  'https://auth.example.com/api/v1/oauth/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=refresh_token&refresh_token=dGhpcyBpcyBhIGhpZ2gtZW50cm9weSByYW5kb20gcmVmcmVzaCB0b2tlbg&client_id=spa-default&scope=openid%20email'
```

---

#### 4c. Grant Type: `client_credentials`

Obtains an access token for machine-to-machine (M2M) communication. No user context — no refresh token or ID token is issued.

##### Request Parameters

| Parameter | Required | Description |
|---|---|---|
| `grant_type` | Yes | `client_credentials` |
| `scope` | No | Space-delimited scopes |
| `client_id` | Yes* | Client identifier (*not needed if using HTTP Basic auth) |
| `client_secret` | Yes | Client secret (always required for `client_credentials`) |

##### Response — Success

```http
HTTP/1.1 200 OK
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache
```

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "read:data write:data"
}
```

> Note: No `refresh_token` or `id_token` is returned for client credentials grants.

##### Access Token JWT Claims (Client Credentials)

```json
{
  "iss": "https://auth.example.com",
  "sub": "m2m-default",
  "aud": ["https://api.example.com"],
  "exp": 1700000900,
  "iat": 1699999100,
  "nbf": 1699999100,
  "jti": "unique-token-id",
  "client_id": "m2m-default",
  "scope": "read:data write:data",
  "tenant_id": "tenant-uuid"
}
```

> For client credentials, the `sub` claim is the `client_id` (there is no user).

##### Error Responses

**Grant type not allowed for client:**

```json
{
  "error": "unauthorized_client",
  "error_description": "client is not authorized for client_credentials grant"
}
```

**Client authentication failed:**

```json
{
  "error": "invalid_client",
  "error_description": "client authentication failed"
}
```

##### cURL Example — HTTP Basic Auth

```bash
curl -X POST \
  'https://auth.example.com/api/v1/oauth/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -H 'Authorization: Basic bTJtLWRlZmF1bHQ6bXktbTJtLXNlY3JldA==' \
  -d 'grant_type=client_credentials&scope=read:data%20write:data'
```

##### cURL Example — POST Body Auth

```bash
curl -X POST \
  'https://auth.example.com/api/v1/oauth/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=client_credentials&client_id=m2m-default&client_secret=my-m2m-secret&scope=read:data%20write:data'
```

---

#### Token Endpoint — Common Errors

**Missing grant_type:**

```json
{
  "error": "invalid_request",
  "error_description": "grant_type: grant_type is required."
}
```

**Unsupported grant_type:**

```json
{
  "error": "invalid_request",
  "error_description": "grant_type: grant_type must be one of: authorization_code, refresh_token, client_credentials."
}
```

**Malformed request body:**

```json
{
  "error": "invalid_request",
  "error_description": "malformed request body"
}
```

---

### 5. Token Revocation

Revokes an access token or refresh token. Follows RFC 7009: always returns 200 OK regardless of whether the token was found or already revoked.

| Property | Value |
|---|---|
| **URL** | `POST /api/v1/oauth/revoke` |
| **Port** | 8081 (public) |
| **Auth** | Client authentication |
| **Content-Type** | `application/x-www-form-urlencoded` |

#### Request Parameters

| Parameter | Required | Description |
|---|---|---|
| `token` | Yes | The token to revoke |
| `token_type_hint` | No | `access_token` or `refresh_token` (helps the server locate the token faster) |
| `client_id` | Yes* | Client identifier (*not needed if using HTTP Basic auth) |
| `client_secret` | Conditional | Required for confidential clients |

#### Response — Success

Per RFC 7009, the server always responds with 200 OK, regardless of outcome:

```http
HTTP/1.1 200 OK
```

Empty body. A successful revocation, already-revoked token, unknown token, or client mismatch all return 200 OK.

#### Error Responses

**Client authentication failed:**

```json
{
  "error": "invalid_client",
  "error_description": "client authentication failed"
}
```

**Missing token:**

```json
{
  "error": "invalid_request",
  "error_description": "token: token is required."
}
```

**Invalid token_type_hint:**

```json
{
  "error": "invalid_request",
  "error_description": "token_type_hint: token_type_hint must be 'access_token' or 'refresh_token'."
}
```

#### cURL Example

```bash
curl -X POST \
  'https://auth.example.com/api/v1/oauth/revoke' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'token=dGhpcyBpcyBhIGhpZ2gtZW50cm9weSByYW5kb20gcmVmcmVzaCB0b2tlbg&token_type_hint=refresh_token&client_id=spa-default'
```

---

### 6. Token Introspection

Validates a token and returns its metadata. Available only on the internal management port. Follows RFC 7662.

| Property | Value |
|---|---|
| **URL** | `POST /api/v1/oauth/introspect` |
| **Port** | 8080 (internal / VPN-only) |
| **Auth** | JWT Required |
| **Content-Type** | `application/x-www-form-urlencoded` |

#### Request Parameters

| Parameter | Required | Description |
|---|---|---|
| `token` | Yes | The token to introspect |
| `token_type_hint` | No | `access_token` or `refresh_token` |

#### Response — Active Token (JWT Access Token)

```http
HTTP/1.1 200 OK
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache
```

```json
{
  "active": true,
  "scope": "openid profile email",
  "client_id": "spa-default",
  "token_type": "Bearer",
  "exp": 1700000900,
  "iat": 1699999100,
  "nbf": 1699999100,
  "sub": "user-identity-sub-123",
  "aud": "https://api.example.com",
  "iss": "https://auth.example.com",
  "jti": "unique-token-id"
}
```

#### Response — Active Token (Refresh Token)

```json
{
  "active": true,
  "scope": "openid profile email",
  "client_id": "spa-default",
  "token_type": "refresh_token",
  "exp": 1700604900,
  "iat": 1699999100,
  "sub": "user-identity-sub-123"
}
```

#### Response — Inactive Token

For invalid, expired, revoked, or unknown tokens:

```json
{
  "active": false
}
```

Per RFC 7662, no additional fields are returned for inactive tokens.

#### Error Responses

**Missing token:**

```json
{
  "error": "invalid_request",
  "error_description": "token: token is required."
}
```

#### cURL Example

```bash
curl -X POST \
  'https://internal-auth.example.com/api/v1/oauth/introspect' \
  -H 'Authorization: Bearer eyJhbGciOi...' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'token=eyJhbGciOiJSUzI1NiJ9...&token_type_hint=access_token'
```

---

### 7. UserInfo Endpoint

Returns claims about the authenticated user. Follows OpenID Connect Core §5.3.

| Property | Value |
|---|---|
| **URL** | `GET /api/v1/oauth/userinfo` |
| **Port** | 8081 (public) |
| **Auth** | JWT Required |
| **Content-Type** | N/A |

#### Response — Success

```http
HTTP/1.1 200 OK
Content-Type: application/json
Cache-Control: no-store
```

```json
{
  "sub": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "email": "user@example.com",
  "email_verified": true,
  "phone_number": "+1234567890",
  "phone_number_verified": false,
  "name": "Jane Doe",
  "picture": "https://cdn.example.com/photos/jane.jpg",
  "updated_at": 1699999100
}
```

#### Field Descriptions

| Field | Type | Condition | Description |
|---|---|---|---|
| `sub` | string | Always | User's unique identifier (UUID) |
| `email` | string | If available | User's email address |
| `email_verified` | boolean | If email present | Whether the email has been verified |
| `phone_number` | string | If available | User's phone number |
| `phone_number_verified` | boolean | If phone present | Whether the phone has been verified |
| `name` | string | If available | User's full name |
| `picture` | string | If profile has URL | URL of the user's profile picture |
| `updated_at` | integer | Always | Unix timestamp of last profile update |

#### Response — Unauthenticated

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json
```

```json
{
  "error": "invalid_token",
  "error_description": "the access token is invalid or has expired"
}
```

#### cURL Example

```bash
curl -X GET \
  'https://auth.example.com/api/v1/oauth/userinfo' \
  -H 'Authorization: Bearer eyJhbGciOi...'
```

---

### 8. List Consent Grants

Returns all consent grants for the authenticated user.

| Property | Value |
|---|---|
| **URL** | `GET /api/v1/oauth/consent/grants` |
| **Port** | 8081 (public) |
| **Auth** | JWT Required |
| **Content-Type** | N/A |

#### Response — Success

```http
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "success": true,
  "data": [
    {
      "consent_grant_id": "c1d2e3f4-a5b6-7890-cdef-123456789012",
      "client_name": "Third-Party Analytics",
      "client_uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "scopes": ["openid", "profile"],
      "granted_at": "2024-11-14T10:30:00Z",
      "updated_at": "2024-11-14T10:30:00Z"
    },
    {
      "consent_grant_id": "d2e3f4a5-b6c7-8901-defa-234567890123",
      "client_name": "External Dashboard",
      "client_uuid": "b2c3d4e5-f6a7-8901-bcde-f23456789012",
      "scopes": ["openid", "email", "profile"],
      "granted_at": "2024-11-10T08:00:00Z",
      "updated_at": "2024-11-12T15:45:00Z"
    }
  ],
  "message": "Consent grants retrieved"
}
```

#### Response — No Grants

```json
{
  "success": true,
  "data": [],
  "message": "Consent grants retrieved"
}
```

#### cURL Example

```bash
curl -X GET \
  'https://auth.example.com/api/v1/oauth/consent/grants' \
  -H 'Authorization: Bearer eyJhbGciOi...'
```

---

### 9. Revoke Consent Grant

Revokes an existing consent grant. The user will be prompted for consent again on the next authorization request to this client.

| Property | Value |
|---|---|
| **URL** | `DELETE /api/v1/oauth/consent/grants/{grant_uuid}` |
| **Port** | 8081 (public) |
| **Auth** | JWT Required |
| **Content-Type** | N/A |

#### Path Parameters

| Parameter | Required | Description |
|---|---|---|
| `grant_uuid` | Yes | UUID of the consent grant to revoke |

#### Response — Success

```http
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "success": true,
  "message": "Consent grant revoked"
}
```

#### Error Responses

**Invalid grant UUID format:**

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json
```

```json
{
  "success": false,
  "error": "Invalid grant UUID"
}
```

**Grant not found or does not belong to user:**

```http
HTTP/1.1 404 Not Found
Content-Type: application/json
```

```json
{
  "success": false,
  "error": "Failed to revoke consent grant"
}
```

#### cURL Example

```bash
curl -X DELETE \
  'https://auth.example.com/api/v1/oauth/consent/grants/c1d2e3f4-a5b6-7890-cdef-123456789012' \
  -H 'Authorization: Bearer eyJhbGciOi...'
```

---

### 10. OpenID Discovery

Returns the OpenID Provider Metadata document. Fully public, no authentication required.

| Property | Value |
|---|---|
| **URL** | `GET /.well-known/openid-configuration` |
| **Port** | 8081 (public) |
| **Auth** | None |
| **Content-Type** | N/A |

#### Response

```http
HTTP/1.1 200 OK
Content-Type: application/json
Cache-Control: public, max-age=3600
```

```json
{
  "issuer": "https://auth.example.com",
  "authorization_endpoint": "https://auth.example.com/api/v1/oauth/authorize",
  "token_endpoint": "https://auth.example.com/api/v1/oauth/token",
  "userinfo_endpoint": "https://auth.example.com/api/v1/oauth/userinfo",
  "jwks_uri": "https://auth.example.com/.well-known/jwks.json",
  "revocation_endpoint": "https://auth.example.com/api/v1/oauth/revoke",
  "introspection_endpoint": "https://auth.example.com/api/v1/oauth/introspect",
  "scopes_supported": [
    "openid",
    "profile",
    "email",
    "offline_access"
  ],
  "response_types_supported": [
    "code"
  ],
  "grant_types_supported": [
    "authorization_code",
    "refresh_token",
    "client_credentials"
  ],
  "subject_types_supported": [
    "public"
  ],
  "id_token_signing_alg_values_supported": [
    "RS256"
  ],
  "token_endpoint_auth_methods_supported": [
    "client_secret_basic",
    "client_secret_post",
    "none"
  ],
  "code_challenge_methods_supported": [
    "S256"
  ]
}
```

#### cURL Example

```bash
curl -X GET 'https://auth.example.com/.well-known/openid-configuration'
```

---

### 11. JWKS Endpoint

Returns the JSON Web Key Set containing the public RSA key used to verify JWTs. Fully public, no authentication required.

| Property | Value |
|---|---|
| **URL** | `GET /.well-known/jwks.json` |
| **Port** | 8081 (public) |
| **Auth** | None |
| **Content-Type** | N/A |

#### Response

```http
HTTP/1.1 200 OK
Content-Type: application/json
Cache-Control: public, max-age=3600
```

```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "kid": "maintainerd-auth-key-1",
      "alg": "RS256",
      "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
      "e": "AQAB"
    }
  ]
}
```

#### Field Descriptions

| Field | Type | Description |
|---|---|---|
| `kty` | string | Key type — always `RSA` |
| `use` | string | Key usage — `sig` (signature) |
| `kid` | string | Key ID — matches the `kid` header in JWTs |
| `alg` | string | Algorithm — `RS256` |
| `n` | string | RSA modulus (base64url-encoded) |
| `e` | string | RSA exponent (base64url-encoded) |

#### Error Response

If JWT keys are not yet initialized:

```http
HTTP/1.1 500 Internal Server Error
Content-Type: application/json
```

```json
{
  "error": "keys not initialised"
}
```

#### cURL Example

```bash
curl -X GET 'https://auth.example.com/.well-known/jwks.json'
```

---

## Flows

### Authorization Code Flow (with PKCE)

The standard flow for web apps, SPAs, and mobile apps when the user has already consented (or consent is not required for the client).

```
┌──────────┐     ┌──────────────┐     ┌──────────────────┐
│  Frontend │     │  Auth Server  │     │  Resource Server  │
└─────┬─────┘     └──────┬───────┘     └────────┬─────────┘
      │                   │                      │
      │  1. Generate PKCE code_verifier + code_challenge
      │                   │                      │
      │  2. GET /oauth/authorize                 │
      │    ?response_type=code                   │
      │    &client_id=spa-default                │
      │    &redirect_uri=https://app/callback    │
      │    &scope=openid profile email           │
      │    &state=random-state                   │
      │    &code_challenge=E9Melh...             │
      │    &code_challenge_method=S256           │
      │ ─────────────────►│                      │
      │                   │                      │
      │  3. Response:     │                      │
      │    { "redirect_uri": ".../callback       │
      │       ?code=abc&state=random-state" }    │
      │ ◄─────────────────│                      │
      │                   │                      │
      │  4. Redirect user │                      │
      │    to redirect_uri│                      │
      │                   │                      │
      │  5. POST /oauth/token                    │
      │    grant_type=authorization_code          │
      │    &code=abc                             │
      │    &redirect_uri=https://app/callback    │
      │    &code_verifier=original-verifier      │
      │    &client_id=spa-default                │
      │ ─────────────────►│                      │
      │                   │                      │
      │  6. Response:     │                      │
      │    { access_token, refresh_token,        │
      │      id_token, ... }                     │
      │ ◄─────────────────│                      │
      │                   │                      │
      │  7. GET /api/resource                    │
      │    Authorization: Bearer <access_token>  │
      │ ──────────────────────────────────────►  │
      │                   │                      │
      │  8. Resource response                    │
      │ ◄──────────────────────────────────────  │
```

#### Step-by-Step Implementation

**Step 1: Generate PKCE Values**

```javascript
// Generate a random code_verifier (43-128 characters)
const codeVerifier = base64URLEncode(crypto.getRandomValues(new Uint8Array(32)));

// Hash it to create the code_challenge
const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(codeVerifier));
const codeChallenge = base64URLEncode(new Uint8Array(digest));
```

**Step 2: Authorization Request**

```bash
curl -X GET \
  'https://auth.example.com/api/v1/oauth/authorize?response_type=code&client_id=spa-default&redirect_uri=https%3A%2F%2Fapp.example.com%2Fcallback&scope=openid%20profile%20email&state=xyzABC123&code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&code_challenge_method=S256' \
  -H 'Authorization: Bearer eyJhbGciOi...'
```

Response (no consent needed):

```json
{
  "success": true,
  "data": {
    "redirect_uri": "https://app.example.com/callback?code=SplxlOBeZQQY&state=xyzABC123"
  },
  "message": "Authorization successful"
}
```

**Step 3: Exchange Code for Tokens**

```bash
curl -X POST \
  'https://auth.example.com/api/v1/oauth/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=authorization_code&code=SplxlOBeZQQY&redirect_uri=https://app.example.com/callback&code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk&client_id=spa-default'
```

Response:

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiJ9...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "dGhpcyBpcyBh...",
  "id_token": "eyJhbGciOiJSUzI1NiJ9...",
  "scope": "openid profile email"
}
```

---

### Authorization Code Flow (with Consent)

When the client requires user consent (third-party applications), the flow includes additional consent steps.

```
┌──────────┐     ┌──────────────┐
│  Frontend │     │  Auth Server  │
└─────┬─────┘     └──────┬───────┘
      │                   │
      │  1. GET /oauth/authorize (same as above)
      │ ─────────────────►│
      │                   │
      │  2. Response: consent required
      │    { "consent_challenge": "uuid-..." }
      │ ◄─────────────────│
      │                   │
      │  3. GET /oauth/consent/{challenge_id}
      │    (retrieve challenge details for UI)
      │ ─────────────────►│
      │                   │
      │  4. Response: challenge details
      │    { client_name, scopes, ... }
      │ ◄─────────────────│
      │                   │
      │  5. Display consent screen to user
      │    (frontend renders scopes + client info)
      │                   │
      │  6. POST /oauth/consent
      │    { challenge_id: "uuid-...",
      │      approved: true }
      │ ─────────────────►│
      │                   │
      │  7. Response: redirect with code
      │    { "redirect_uri": ".../callback?code=abc" }
      │ ◄─────────────────│
      │                   │
      │  8. POST /oauth/token (same as standard flow)
      │ ─────────────────►│
      │                   │
      │  9. Token response
      │ ◄─────────────────│
```

#### Step-by-Step Implementation

**Step 1: Authorization Request (same as standard)**

```bash
curl -X GET \
  'https://auth.example.com/api/v1/oauth/authorize?response_type=code&client_id=third-party-app&redirect_uri=https%3A%2F%2Fthirdparty.example.com%2Fcallback&scope=openid%20profile%20email&state=xyzABC123&code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&code_challenge_method=S256' \
  -H 'Authorization: Bearer eyJhbGciOi...'
```

Response (consent required):

```json
{
  "success": true,
  "data": {
    "consent_challenge": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
  },
  "message": "Consent required"
}
```

**Step 2: Retrieve Consent Challenge**

```bash
curl -X GET \
  'https://auth.example.com/api/v1/oauth/consent/f47ac10b-58cc-4372-a567-0e02b2c3d479' \
  -H 'Authorization: Bearer eyJhbGciOi...'
```

Response:

```json
{
  "success": true,
  "data": {
    "challenge_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "client_name": "Third-Party Analytics",
    "client_uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "scopes": ["openid", "profile", "email"],
    "redirect_uri": "https://thirdparty.example.com/callback",
    "expires_at": 1700001000
  },
  "message": "Consent challenge retrieved"
}
```

**Step 3: Submit Consent Decision (Approve)**

```bash
curl -X POST \
  'https://auth.example.com/api/v1/oauth/consent' \
  -H 'Authorization: Bearer eyJhbGciOi...' \
  -H 'Content-Type: application/json' \
  -d '{
    "challenge_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "approved": true
  }'
```

Response:

```json
{
  "success": true,
  "data": {
    "redirect_uri": "https://thirdparty.example.com/callback?code=SplxlOBeZQQY&state=xyzABC123"
  },
  "message": "Consent processed"
}
```

**Step 3 (alternative): Deny Consent**

```bash
curl -X POST \
  'https://auth.example.com/api/v1/oauth/consent' \
  -H 'Authorization: Bearer eyJhbGciOi...' \
  -H 'Content-Type: application/json' \
  -d '{
    "challenge_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "approved": false
  }'
```

Response:

```json
{
  "error": "access_denied",
  "error_description": "the user denied the consent request"
}
```

**Step 4: Exchange Code for Tokens (same as standard flow)**

---

### Client Credentials Flow

For machine-to-machine communication where no user is involved.

```
┌──────────┐     ┌──────────────┐
│  M2M App  │     │  Auth Server  │
└─────┬─────┘     └──────┬───────┘
      │                   │
      │  1. POST /oauth/token
      │    grant_type=client_credentials
      │    &client_id=m2m-default
      │    &client_secret=secret
      │    &scope=read:data
      │ ─────────────────►│
      │                   │
      │  2. Response:      │
      │    { access_token, │
      │      token_type,   │
      │      expires_in }  │
      │ ◄─────────────────│
      │                   │
      │  3. Use token      │
      │ ──────────────────────►  Resource Server
```

#### Implementation

```bash
# Using HTTP Basic auth
curl -X POST \
  'https://auth.example.com/api/v1/oauth/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -H 'Authorization: Basic bTJtLWRlZmF1bHQ6bXktbTJtLXNlY3JldA==' \
  -d 'grant_type=client_credentials&scope=read:data%20write:data'
```

Response:

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiJ9...",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "read:data write:data"
}
```

---

### Refresh Token Flow

Obtain new tokens using a refresh token. The old refresh token is invalidated (rotation).

```
┌──────────┐     ┌──────────────┐
│  Frontend │     │  Auth Server  │
└─────┬─────┘     └──────┬───────┘
      │                   │
      │  Access token expired
      │                   │
      │  1. POST /oauth/token
      │    grant_type=refresh_token
      │    &refresh_token=old-token
      │    &client_id=spa-default
      │ ─────────────────►│
      │                   │
      │  2. Response:      │
      │    { access_token  │  (new)
      │      refresh_token │  (new — MUST store this)
      │      id_token }    │  (new)
      │ ◄─────────────────│
      │                   │
      │  Continue using new access_token
```

#### Implementation

```bash
curl -X POST \
  'https://auth.example.com/api/v1/oauth/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=refresh_token&refresh_token=dGhpcyBpcyBh...&client_id=spa-default'
```

Response:

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiJ9...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "bmV3LXJlZnJlc2gtdG9rZW4...",
  "id_token": "eyJhbGciOiJSUzI1NiJ9...",
  "scope": "openid profile email"
}
```

> **Important**: Always store the new `refresh_token` from the response. The old one is immediately invalidated. Attempting to reuse the old token triggers family revocation — all tokens in the family are revoked and the user must re-authenticate.

#### Reuse Detection Example

If an attacker captures an old refresh token and tries to use it:

```bash
# Attacker uses the old (already rotated) refresh token
curl -X POST \
  'https://auth.example.com/api/v1/oauth/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=refresh_token&refresh_token=old-already-rotated-token&client_id=spa-default'
```

Response:

```json
{
  "error": "invalid_grant",
  "error_description": "token reuse detected; token family revoked"
}
```

All tokens in the family (including the legitimate user's current tokens) are revoked. Both parties must re-authenticate.

---

### Token Revocation Flow

Explicitly revoke a refresh token (e.g., during logout).

```
┌──────────┐     ┌──────────────┐
│  Frontend │     │  Auth Server  │
└─────┬─────┘     └──────┬───────┘
      │                   │
      │  User clicks "Log out"
      │                   │
      │  1. POST /oauth/revoke
      │    token=refresh-token
      │    &token_type_hint=refresh_token
      │    &client_id=spa-default
      │ ─────────────────►│
      │                   │
      │  2. 200 OK (empty body)
      │ ◄─────────────────│
      │                   │
      │  3. Clear local tokens
      │  4. Redirect to login
```

#### Implementation

```bash
curl -X POST \
  'https://auth.example.com/api/v1/oauth/revoke' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'token=dGhpcyBpcyBh...&token_type_hint=refresh_token&client_id=spa-default'
```

Response: `200 OK` (empty body).

---

### Consent Management Flow

Users can view and revoke their existing consent grants.

#### List All Grants

```bash
curl -X GET \
  'https://auth.example.com/api/v1/oauth/consent/grants' \
  -H 'Authorization: Bearer eyJhbGciOi...'
```

Response:

```json
{
  "success": true,
  "data": [
    {
      "consent_grant_id": "c1d2e3f4-a5b6-7890-cdef-123456789012",
      "client_name": "Third-Party Analytics",
      "client_uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "scopes": ["openid", "profile"],
      "granted_at": "2024-11-14T10:30:00Z",
      "updated_at": "2024-11-14T10:30:00Z"
    }
  ],
  "message": "Consent grants retrieved"
}
```

#### Revoke a Grant

```bash
curl -X DELETE \
  'https://auth.example.com/api/v1/oauth/consent/grants/c1d2e3f4-a5b6-7890-cdef-123456789012' \
  -H 'Authorization: Bearer eyJhbGciOi...'
```

Response:

```json
{
  "success": true,
  "message": "Consent grant revoked"
}
```

After revoking a consent grant, the next authorization request from that client will trigger the consent screen again.
