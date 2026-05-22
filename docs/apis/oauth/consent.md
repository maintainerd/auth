# Consent Endpoints

When a client requires user consent and the user has not previously approved the requested scopes, the [authorization endpoint](./authorize.md) returns a `consent_challenge` instead of an authorization code. These endpoints allow the frontend to retrieve the details of that challenge and then submit the user's decision.

**Authentication required.** Both endpoints require a valid JWT access token. The challenge is scoped to the authenticated user — it cannot be retrieved or resolved by a different user.

---

## Endpoints

| Method | Path | Auth |
|--------|------|------|
| GET | `/api/v1/oauth/consent/{challenge_id}` | Bearer JWT |
| POST | `/api/v1/oauth/consent` | Bearer JWT |

---

## GET /api/v1/oauth/consent/{challenge_id}

Retrieves the details of a pending consent challenge. Use this to render the consent screen — it returns the client name, the scopes being requested, and when the challenge expires.

### Authentication

Bearer JWT in the `Authorization` header.

### Request

#### Headers

| Name | Required | Description |
|------|----------|-------------|
| `Authorization` | Yes | `Bearer <access_token>` |

#### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `challenge_id` | UUID | Yes | The consent challenge UUID returned by the authorization endpoint. |

### Response

#### Success — 200 OK

```json
{
  "success": true,
  "message": "Consent challenge retrieved",
  "data": {
    "challenge_id": "a3f1b2c4-1234-5678-abcd-ef0123456789",
    "client_name": "My Application",
    "client_uuid": "d1e2f3a4-5678-90ab-cdef-012345678901",
    "scopes": ["openid", "email", "profile"],
    "redirect_uri": "https://app.example.com/callback",
    "expires_at": 1716300600
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `challenge_id` | string | The UUID of this consent challenge. Pass this to the POST endpoint. |
| `client_name` | string | Human-readable display name of the client requesting access. |
| `client_uuid` | string | UUID of the client application. |
| `scopes` | array of strings | The scopes the client is requesting. |
| `redirect_uri` | string | The URI the user will be redirected to after the decision is submitted. |
| `expires_at` | integer | Unix timestamp when this challenge expires. Challenges expire after 10 minutes. |

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | `{ "success": false, "message": "Invalid challenge ID" }` — `challenge_id` is not a valid UUID. |
| 401 | `{ "success": false, "message": "Authentication required" }` — no valid JWT was provided. |
| 403 | The challenge belongs to a different user. |
| 404 | The challenge was not found or has already been used. |
| 422 | The challenge has expired. |

### Example

```bash
curl "https://auth.example.com/api/v1/oauth/consent/a3f1b2c4-1234-5678-abcd-ef0123456789" \
  -H "Authorization: Bearer <access_token>"
```

---

## POST /api/v1/oauth/consent

Submits the user's decision for a pending consent challenge. On approval, a consent grant is persisted and an authorization code is issued. On denial, the user is redirected to the client's redirect URI with an `access_denied` error.

In both cases the response contains a `redirect_uri` — the frontend should redirect the user there.

### Authentication

Bearer JWT in the `Authorization` header.

### Request

#### Headers

| Name | Required | Description |
|------|----------|-------------|
| `Authorization` | Yes | `Bearer <access_token>` |
| `Content-Type` | Yes | `application/json` |

#### Request Body

```json
{
  "challenge_id": "a3f1b2c4-1234-5678-abcd-ef0123456789",
  "approved": true
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `challenge_id` | string (UUID) | Yes | The consent challenge UUID to resolve. |
| `approved` | boolean | Yes | `true` to grant consent, `false` to deny. |

### Response

#### Success — 200 OK (approved)

The authorization code is embedded in the `redirect_uri`. The frontend should redirect the user to this URI.

```json
{
  "success": true,
  "message": "Consent processed",
  "data": {
    "redirect_uri": "https://app.example.com/callback?code=<authorization_code>&state=<state>"
  }
}
```

#### Success — 200 OK (denied)

The redirect URI contains `error=access_denied`.

```json
{
  "success": true,
  "message": "Consent processed",
  "data": {
    "redirect_uri": "https://app.example.com/callback?error=access_denied&error_description=the+resource+owner+denied+the+request&state=<state>"
  }
}
```

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_request` | The challenge was not found, has expired, or `challenge_id` is missing. |
| 403 | `access_denied` | The challenge belongs to a different user. |
| 500 | `server_error` | An unexpected internal error occurred. |

Non-OAuth errors:

| Status | Description |
|--------|-------------|
| 400 | `{ "success": false, "message": "Invalid request body" }` — the body could not be parsed as JSON. |
| 401 | `{ "success": false, "message": "Authentication required" }` — no valid JWT was provided. |
| 422 | Validation error — `challenge_id` is missing or not a valid UUID. |

```json
{ "error": "invalid_request", "error_description": "consent challenge not found or expired" }
```

### Examples

#### Approve consent

```bash
curl -X POST "https://auth.example.com/api/v1/oauth/consent" \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"challenge_id": "a3f1b2c4-1234-5678-abcd-ef0123456789", "approved": true}'
```

#### Deny consent

```bash
curl -X POST "https://auth.example.com/api/v1/oauth/consent" \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"challenge_id": "a3f1b2c4-1234-5678-abcd-ef0123456789", "approved": false}'
```
