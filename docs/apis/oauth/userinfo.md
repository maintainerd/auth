# UserInfo Endpoint

Returns claims about the authenticated user (OpenID Connect Core §5.3). The claims returned depend on the scopes granted in the access token used to call this endpoint.

**Authentication required.** This endpoint requires a valid JWT access token.

---

## Endpoints

| Method | Path | Auth |
|--------|------|------|
| GET | `/api/v1/oauth/userinfo` | Bearer JWT |

---

## GET /api/v1/oauth/userinfo

### Authentication

Bearer JWT access token in the `Authorization` header. The token must have been issued with at least the `openid` scope.

### Request

#### Headers

| Name | Required | Description |
|------|----------|-------------|
| `Authorization` | Yes | `Bearer <access_token>` |

### Response

#### Success — 200 OK

Responses are served with `Cache-Control: no-store`.

```json
{
  "sub": "a1b2c3d4-0000-1111-2222-333344445555",
  "email": "user@example.com",
  "email_verified": true,
  "phone_number": "+15550001234",
  "phone_number_verified": false,
  "name": "Jane Smith",
  "picture": "https://cdn.example.com/avatars/janesmith.jpg",
  "updated_at": 1716300000
}
```

| Field | Type | Description |
|-------|------|-------------|
| `sub` | string | The user's UUID. Stable, unique identifier for the user within this authorization server. |
| `email` | string | The user's email address. Omitted if not available. |
| `email_verified` | boolean | Whether the email address has been verified. Omitted if `false`. |
| `phone_number` | string | The user's phone number. Omitted if not available. |
| `phone_number_verified` | boolean | Whether the phone number has been verified. Omitted if `false`. |
| `name` | string | The user's full name. Omitted if not available. |
| `picture` | string | URL of the user's profile picture. Omitted if no profile picture is set. |
| `updated_at` | integer | Unix timestamp of when the user's profile was last updated. Omitted if not available. |

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 401 | `invalid_token` | The access token is missing, invalid, or has expired. |

```json
{ "error": "invalid_token", "error_description": "the access token is invalid or has expired" }
```

### Example

```bash
curl "https://auth.example.com/api/v1/oauth/userinfo" \
  -H "Authorization: Bearer <access_token>"
```
