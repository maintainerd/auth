# Token Introspection

Token introspection allows authorized resource servers and internal services to query the active state of an access or refresh token. The response follows the standard format defined in RFC 7662.

This endpoint is only accessible on the management port (8080). It is not reachable from the public internet.

**RFC reference:** [RFC 7662 — OAuth 2.0 Token Introspection](https://www.rfc-editor.org/rfc/rfc7662)

---

## Endpoints

| Method | Path | Auth | Port |
|--------|------|------|------|
| POST | `/api/v1/oauth/introspect` | Bearer JWT (user session) | 8080 (management only) |

---

## POST /api/v1/oauth/introspect

Inspects a token and returns its active status along with associated metadata. If the token is invalid, expired, or revoked, the response is `{ "active": false }` — no error is returned.

### Authentication

Requires a valid Bearer JWT in the `Authorization` header. This endpoint is mounted on the management port (8080) and must not be exposed publicly.

```
Authorization: Bearer <access_token>
```

### Request

#### Headers

| Header | Value |
|--------|-------|
| `Content-Type` | `application/x-www-form-urlencoded` |
| `Authorization` | `Bearer <management_access_token>` |

#### Body

`application/x-www-form-urlencoded`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | string | Yes | The access token or refresh token to introspect. |
| `token_type_hint` | string | No | Hint about the token type. One of `access_token` or `refresh_token`. |

### Response

#### Success — 200 OK

When the token is active:

```json
{
  "active": true,
  "scope": "openid profile email",
  "client_id": "abc123clientid",
  "username": "user@example.com",
  "token_type": "Bearer",
  "exp": 1716000000,
  "iat": 1715996400,
  "nbf": 1715996400,
  "sub": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "aud": "https://auth.example.com",
  "iss": "https://auth.example.com",
  "jti": "unique-token-id"
}
```

When the token is inactive (invalid, expired, or revoked):

```json
{
  "active": false
}
```

#### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `active` | boolean | `true` if the token is valid and not expired or revoked. |
| `scope` | string | Space-separated list of scopes granted by the token. Omitted when inactive. |
| `client_id` | string | The client identifier the token was issued to. Omitted when inactive. |
| `username` | string | Human-readable identifier for the resource owner. Omitted when inactive. |
| `token_type` | string | Token type, typically `Bearer`. Omitted when inactive. |
| `exp` | integer | Unix timestamp when the token expires. Omitted when inactive. |
| `iat` | integer | Unix timestamp when the token was issued. Omitted when inactive. |
| `nbf` | integer | Unix timestamp before which the token must not be accepted. Omitted when inactive. |
| `sub` | string | Subject — the user UUID the token was issued for. Omitted when inactive. |
| `aud` | string | Audience the token is intended for. Omitted when inactive. |
| `iss` | string | Issuer identifier. Omitted when inactive. |
| `jti` | string | Unique token identifier. Omitted when inactive. |

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_request` | `token` field is missing or `token_type_hint` has an invalid value. |
| 401 | — | No valid management session JWT provided. Returns `{ "success": false, "message": "..." }`. |

### Example

```bash
curl -X POST https://auth.example.com/api/v1/oauth/introspect \
  -H "Authorization: Bearer <management_access_token>" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...&token_type_hint=access_token"
```
