# Device Authorization Grant

The Device Authorization Grant enables OAuth 2.0 on input-constrained devices (smart TVs, CLIs, IoT devices) that cannot open a browser or handle complex redirects. The device obtains a code, displays a URL and user code to the operator, and polls for a token while the user approves the request on a separate device.

**RFC reference:** [RFC 8628 — OAuth 2.0 Device Authorization Grant](https://www.rfc-editor.org/rfc/rfc8628)

---

## Endpoints

| Method | Path | Auth | Port |
|--------|------|------|------|
| POST | `/api/v1/oauth/device_authorization` | Client credentials | 8081 |
| POST | `/api/v1/oauth/device` | Bearer JWT (end user) | 8081 |
| POST | `/api/v1/oauth/device/deny` | Bearer JWT (end user) | 8081 |
| POST | `/api/v1/oauth/token` | Client credentials | 8081 |

---

## Flow Overview

```
1. Device          → POST /oauth/device_authorization  → get device_code + user_code
2. Device          → Display verification_uri and user_code to the user
3. User (browser)  → Navigate to verification_uri, authenticate, enter user_code
4. User (browser)  → POST /oauth/device              → approve (or /oauth/device/deny to deny)
5. Device          → POST /oauth/token (poll)         → receive access_token when approved
```

The device must begin polling after step 1 using the `interval` returned in the authorization response (5 seconds). The device code expires after 15 minutes.

---

## POST /api/v1/oauth/device_authorization

Initiates a device authorization flow. Returns a `device_code`, a short `user_code` for human entry, and verification URIs.

### Authentication

Client credentials via HTTP Basic auth or POST body. The client must be authorized for the `urn:ietf:params:oauth:grant-type:device_code` grant.

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
| `client_id` | string | Yes | Max 255 chars | The client identifier. Required when not using Basic auth. |
| `scope` | string | No | Max 1024 chars | Space-separated scopes to request. |
| `client_secret` | string | No | — | Client secret for `client_secret_post` auth. |

### Response

#### Success — 200 OK

```json
{
  "device_code": "GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS",
  "user_code": "BCDF-GHJK",
  "verification_uri": "https://auth.example.com/device",
  "verification_uri_complete": "https://auth.example.com/device?user_code=BCDF-GHJK",
  "expires_in": 900,
  "interval": 5
}
```

| Field | Type | Description |
|-------|------|-------------|
| `device_code` | string | Opaque code the device uses to poll for a token. Keep this secret. |
| `user_code` | string | 8-character code (format `XXXX-XXXX`) the user enters at the verification URI. |
| `verification_uri` | string | URL the user visits to approve the request. |
| `verification_uri_complete` | string | Same URL with `user_code` pre-filled as a query parameter. |
| `expires_in` | integer | Seconds until both the `device_code` and `user_code` expire. Fixed at `900` (15 minutes). |
| `interval` | integer | Minimum seconds between polling attempts. Fixed at `5`. |

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_request` | `client_id` is missing. |
| 401 | `invalid_client` | Client not found, inactive, or secret mismatch. |
| 401 | `unauthorized_client` | Client not authorized for the device_code grant. |
| 500 | `server_error` | Unexpected internal error. |

---

## POST /api/v1/oauth/device

Called by the authenticated end user to approve a pending device authorization request. The user must be signed in (valid Bearer JWT).

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

| Field | Type | Required | Constraints | Description |
|-------|------|----------|-------------|-------------|
| `user_code` | string | Yes | 8–9 chars, `XXXX-XXXX` format | The user code shown on the device. |

### Response

#### Success — 200 OK

```json
{
  "success": true,
  "message": "device authorized"
}
```

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_grant` | `user_code` not found, already used, or expired. |
| 401 | — | No authenticated user session. Returns `{ "success": false, "message": "authentication required" }`. |
| 500 | `server_error` | Unexpected internal error. |

---

## POST /api/v1/oauth/device/deny

Called by the authenticated end user to explicitly deny a pending device authorization request. The device will receive `access_denied` on its next poll.

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
| `user_code` | string | Yes | The user code shown on the device. |

### Response

#### Success — 200 OK

```json
{
  "success": true,
  "message": "device authorization denied"
}
```

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_grant` | `user_code` not found or already used. |
| 401 | — | No authenticated user session. |
| 500 | `server_error` | Unexpected internal error. |

---

## POST /api/v1/oauth/token (device_code polling)

The device polls this endpoint with `grant_type=urn:ietf:params:oauth:grant-type:device_code` until it receives a token, an error indicating the request was denied, or the code expires.

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
| `grant_type` | string | Yes | Must be `urn:ietf:params:oauth:grant-type:device_code` |
| `device_code` | string | Yes | The `device_code` from the authorization response. |
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

These errors are expected during polling and must be handled by the device:

| Status | `error` | Description | Action |
|--------|---------|-------------|--------|
| 400 | `authorization_pending` | User has not yet acted on the request. | Continue polling at the current interval. |
| 400 | `slow_down` | Polling too frequently. | Increase the polling interval by 5 seconds and continue. |
| 400 | `expired_token` | The `device_code` has expired. | Restart the flow with a new `POST /oauth/device_authorization`. |
| 403 | `access_denied` | The user denied the request. | Inform the user; do not retry. |

#### Other Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_grant` | `device_code` not found or belongs to a different client. |
| 401 | `invalid_client` | Client authentication failed. |
| 500 | `server_error` | Unexpected internal error. |

### Polling Example

```bash
# Step 1: Request device authorization
curl -X POST https://auth.example.com/api/v1/oauth/device_authorization \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "my_client_id:my_client_secret" \
  -d "client_id=my_client_id&scope=openid+profile"
```

```json
{
  "device_code": "GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS",
  "user_code": "BCDF-GHJK",
  "verification_uri": "https://auth.example.com/device",
  "verification_uri_complete": "https://auth.example.com/device?user_code=BCDF-GHJK",
  "expires_in": 900,
  "interval": 5
}
```

```bash
# Step 2: Display to user:
#   Go to: https://auth.example.com/device
#   Enter code: BCDF-GHJK

# Step 3: Poll every 5 seconds
curl -X POST https://auth.example.com/api/v1/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "my_client_id:my_client_secret" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:device_code" \
  -d "device_code=GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS" \
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
