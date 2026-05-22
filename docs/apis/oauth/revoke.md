# Token Revocation Endpoint

Revokes an access token or refresh token (RFC 7009). After revocation the token can no longer be used. If the token does not exist or has already expired, the server still returns a success response — revocation is idempotent by design.

---

## Endpoints

| Method | Path | Auth |
|--------|------|------|
| POST | `/api/v1/oauth/revoke` | Client credentials (Basic or body) |

---

## POST /api/v1/oauth/revoke

### Authentication

Confidential clients must authenticate using one of:

- **HTTP Basic Authentication** (`client_secret_basic`): `Authorization: Basic <base64(client_id:client_secret)>`
- **Request body** (`client_secret_post`): Include `client_id` and `client_secret` as form fields.

Public clients (registered with `token_endpoint_auth_method=none`) omit client credentials.

### Request

#### Headers

| Name | Required | Description |
|------|----------|-------------|
| `Content-Type` | Yes | Must be `application/x-www-form-urlencoded`. |
| `Authorization` | Conditional | `Basic <base64(client_id:client_secret)>` for `client_secret_basic` auth. |

#### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | string | Yes | The token to revoke. |
| `token_type_hint` | string | No | Hint about the token type. One of: `access_token`, `refresh_token`. The server will attempt the indicated type first but will try both if the hint is absent or incorrect. |
| `client_id` | string | Conditional | Required when using `client_secret_post` auth or for public clients. |
| `client_secret` | string | Conditional | Required when using `client_secret_post` auth. |

### Response

#### Success — 200 OK

Per RFC 7009 §2.2, a successful revocation always returns HTTP 200, even if the token did not exist or was already expired.

```json
{
  "success": true,
  "message": "Token revoked"
}
```

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_request` | The `token` parameter is missing. |
| 400 | `invalid_request` | `token_type_hint` is not `access_token` or `refresh_token`. |
| 401 | `invalid_client` | Client authentication failed. |
| 500 | `server_error` | An unexpected internal error occurred. |

```json
{ "error": "invalid_request", "error_description": "token is required" }
```

### Examples

#### Revoke a refresh token

```bash
curl -X POST "https://auth.example.com/api/v1/oauth/revoke" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "my-client-id:my-client-secret" \
  -d "token=<refresh_token>" \
  -d "token_type_hint=refresh_token"
```

#### Revoke an access token

```bash
curl -X POST "https://auth.example.com/api/v1/oauth/revoke" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "my-client-id:my-client-secret" \
  -d "token=<access_token>" \
  -d "token_type_hint=access_token"
```
