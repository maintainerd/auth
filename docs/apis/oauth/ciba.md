# Client-Initiated Backchannel Authentication (CIBA)

CIBA is a decoupled authentication flow where the client triggers an authentication request for a user identified by a login hint, and the user approves or denies the request out-of-band (e.g., via a push notification, email, or authenticator app). The client polls for a token independently of the user's interaction.

This server implements the **poll mode** delivery model. Push and ping delivery modes are not supported.

**Spec reference:** [OpenID Connect Client-Initiated Backchannel Authentication Flow — Core 1.0](https://openid.net/specs/openid-client-initiated-backchannel-authentication-core-1_0.html)

---

## Endpoints

| Method | Path | Auth | Port |
|--------|------|------|------|
| POST | `/api/v1/oauth/ciba` | Client credentials | 8081 |
| POST | `/api/v1/oauth/ciba/approve` | Bearer JWT (end user) | 8081 |
| POST | `/api/v1/oauth/ciba/deny` | Bearer JWT (end user) | 8081 |
| POST | `/api/v1/oauth/token` | Client credentials | 8081 |

---

## Flow Overview

```
1. Client          → POST /oauth/ciba                  → receive auth_req_id
2. Server          → Send notification to the user (email)
3. User (app/UI)   → POST /oauth/ciba/approve          → approve (or /oauth/ciba/deny to deny)
4. Client          → POST /oauth/token (poll)           → receive access_token when approved
```

The client begins polling after step 1 using the `interval` from the initiation response (5 seconds). The `auth_req_id` expires after 5 minutes.

---

## POST /api/v1/oauth/ciba

Initiates a backchannel authentication request. The server identifies the user via `login_hint`, creates a pending authentication request, sends an out-of-band notification to the user, and returns an `auth_req_id` for polling.

### Authentication

Client credentials via HTTP Basic auth or POST body. The client must be authorized for the `urn:openid:params:grant-type:ciba` grant.

### Request

#### Headers

| Header | Value |
|--------|-------|
| `Content-Type` | `application/x-www-form-urlencoded` |
| `Authorization` | `Basic <base64(client_id:client_secret)>` (confidential clients) |

#### Body

`application/x-www-form-urlencoded`

| Field | Type | Required | Constraints | Description |
|-------|------|----------|-------------|-------------|
| `scope` | string | Yes | 1–1024 chars | Space-separated scopes to request. Must include `openid` for OIDC flows. |
| `login_hint` | string | Conditionally required | — | Email or identifier of the user to authenticate. Required when `login_hint_token` and `id_token_hint` are both absent. |
| `client_id` | string | Yes | — | The client identifier. Required when not using Basic auth. |
| `binding_message` | string | No | Max 128 chars | Short message displayed to the user to associate the request with the client action. |
| `login_hint_token` | string | No | — | Token containing the identity hint. Alternative to `login_hint`. |
| `id_token_hint` | string | No | — | Previously issued ID token identifying the user. Alternative to `login_hint`. |
| `client_notification_token` | string | No | — | Token for push/ping delivery modes. Not used in poll mode. |
| `acr_values` | string | No | — | Requested Authentication Context Class Reference values. |
| `user_code` | string | No | — | User-provided code for step-up authentication. |
| `requested_expiry` | integer | No | — | Requested expiry in seconds for the `auth_req_id`. |
| `client_secret` | string | No | — | Client secret for `client_secret_post` auth. |

At least one of `login_hint`, `login_hint_token`, or `id_token_hint` must be provided.

### Response

#### Success — 200 OK

```json
{
  "auth_req_id": "1c266114-a1be-4252-8ad1-04986c5b9ac1",
  "expires_in": 300,
  "interval": 5
}
```

| Field | Type | Description |
|-------|------|-------------|
| `auth_req_id` | string | Opaque identifier for the pending authentication request. Use this to poll and for user approve/deny actions. |
| `expires_in` | integer | Seconds until the `auth_req_id` expires. Fixed at `300` (5 minutes). |
| `interval` | integer | Minimum seconds between polling attempts. Fixed at `5`. |

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_request` | `scope`, `client_id`, or a login hint is missing. |
| 400 | `invalid_request` | No user found for the provided `login_hint`. |
| 401 | `invalid_client` | Client not found, inactive, or secret mismatch. |
| 401 | `unauthorized_client` | Client not authorized for the CIBA grant. |
| 500 | `server_error` | Unexpected internal error. |

---

## POST /api/v1/oauth/ciba/approve

Called by the authenticated end user to approve a pending CIBA request. The user must be signed in and the `auth_req_id` must not have expired.

### Authentication

Requires an authenticated user session:

```
Authorization: Bearer <user_access_token>
```

### Request

#### Headers

| Header | Value |
|--------|-------|
| `Content-Type` | `application/x-www-form-urlencoded` |
| `Authorization` | `Bearer <user_access_token>` |

#### Body

`application/x-www-form-urlencoded`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `auth_req_id` | string | Yes | The `auth_req_id` from the CIBA initiation response. |

### Response

#### Success — 200 OK

```json
{
  "success": true,
  "message": "request approved"
}
```

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | — | `auth_req_id` is missing. Returns `{ "success": false, "message": "auth_req_id is required" }`. |
| 400 | `invalid_grant` | `auth_req_id` not found or has expired. |
| 401 | — | No authenticated user session. Returns `{ "success": false, "message": "authentication required" }`. |
| 500 | `server_error` | Unexpected internal error. |

---

## POST /api/v1/oauth/ciba/deny

Called by the authenticated end user to deny a pending CIBA request. The client will receive `access_denied` on its next poll.

### Authentication

Requires an authenticated user session:

```
Authorization: Bearer <user_access_token>
```

### Request

#### Headers

| Header | Value |
|--------|-------|
| `Content-Type` | `application/x-www-form-urlencoded` |
| `Authorization` | `Bearer <user_access_token>` |

#### Body

`application/x-www-form-urlencoded`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `auth_req_id` | string | Yes | The `auth_req_id` from the CIBA initiation response. |

### Response

#### Success — 200 OK

```json
{
  "success": true,
  "message": "request denied"
}
```

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | — | `auth_req_id` is missing. |
| 400 | `invalid_grant` | `auth_req_id` not found. |
| 401 | — | No authenticated user session. |
| 500 | `server_error` | Unexpected internal error. |

---

## POST /api/v1/oauth/token (CIBA polling)

The client polls this endpoint with `grant_type=urn:openid:params:grant-type:ciba` until it receives a token, the request is denied, or the `auth_req_id` expires.

### Authentication

Client credentials via HTTP Basic auth or POST body.

### Request

#### Headers

| Header | Value |
|--------|-------|
| `Content-Type` | `application/x-www-form-urlencoded` |
| `Authorization` | `Basic <base64(client_id:client_secret)>` (confidential clients) |

#### Body

`application/x-www-form-urlencoded`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `grant_type` | string | Yes | Must be `urn:openid:params:grant-type:ciba` |
| `auth_req_id` | string | Yes | The `auth_req_id` from the initiation response. |
| `client_id` | string | Yes | The client identifier. Required when not using Basic auth. |
| `client_secret` | string | No | Client secret for `client_secret_post` auth. |

### Response

#### Success — 200 OK

Returned when the user has approved the request:

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "openid profile"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `access_token` | string | Signed RS256 JWT access token. |
| `token_type` | string | Always `Bearer`. |
| `expires_in` | integer | Seconds until the access token expires. |
| `scope` | string | Scopes granted. |

#### Polling Error Responses

| Status | `error` | Description | Action |
|--------|---------|-------------|--------|
| 400 | `authorization_pending` | User has not yet acted on the request. | Continue polling at the current interval. |
| 400 | `slow_down` | Polling too frequently. | Increase the polling interval by 5 seconds and continue. |
| 400 | `expired_token` | The `auth_req_id` has expired. | Restart the flow with a new `POST /oauth/ciba`. |
| 403 | `access_denied` | The user denied the request. | Inform the user; do not retry. |

#### Other Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_grant` | `auth_req_id` not found or belongs to a different client. |
| 401 | `invalid_client` | Client authentication failed. |
| 500 | `server_error` | Unexpected internal error. |

### Example

```bash
# Step 1: Initiate the CIBA request
curl -X POST https://auth.example.com/api/v1/oauth/ciba \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "my_client_id:my_client_secret" \
  -d "scope=openid+profile" \
  -d "login_hint=user@example.com" \
  -d "binding_message=Login+to+Acme+Dashboard" \
  -d "client_id=my_client_id"
```

```json
{
  "auth_req_id": "1c266114-a1be-4252-8ad1-04986c5b9ac1",
  "expires_in": 300,
  "interval": 5
}
```

```bash
# Step 2: User receives email notification and approves at https://auth.example.com

# Step 3: Poll every 5 seconds
curl -X POST https://auth.example.com/api/v1/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "my_client_id:my_client_secret" \
  -d "grant_type=urn:openid:params:grant-type:ciba" \
  -d "auth_req_id=1c266114-a1be-4252-8ad1-04986c5b9ac1" \
  -d "client_id=my_client_id"
```

While pending:
```json
{
  "error": "authorization_pending",
  "error_description": "the user has not yet approved the request"
}
```

After user approves:
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "openid profile"
}
```
