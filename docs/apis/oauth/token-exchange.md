# Token Exchange

Token Exchange allows a client to obtain a new token by presenting an existing token (the subject token). Common use cases include service-to-service impersonation, delegation, and cross-service token propagation.

**RFC reference:** [RFC 8693 — OAuth 2.0 Token Exchange](https://www.rfc-editor.org/rfc/rfc8693)

---

## Endpoints

| Method | Path | Auth | Port |
|--------|------|------|------|
| POST | `/api/v1/oauth/token` | Client credentials | 8081 |

---

## POST /api/v1/oauth/token

Submit `grant_type=urn:ietf:params:oauth:grant-type:token-exchange` to exchange an existing token for a new one. The server validates the subject token, verifies the client, and issues a new token of the requested type.

### Authentication

Client credentials via HTTP Basic auth or POST body. The `client_id` field is always required.

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
| `grant_type` | string | Yes | Must be `urn:ietf:params:oauth:grant-type:token-exchange` |
| `subject_token` | string | Yes | The token to exchange. |
| `subject_token_type` | string | Yes | Type URI of `subject_token`. See supported values below. |
| `client_id` | string | Yes | The client identifier. Required when not using Basic auth. |
| `client_secret` | string | No | Client secret for `client_secret_post` auth. |
| `requested_token_type` | string | No | Desired type of the issued token. Defaults to `access_token` when omitted. See supported values below. |
| `actor_token` | string | No | Token representing the acting party (delegation scenarios). |
| `actor_token_type` | string | No | Type URI of `actor_token`. Required when `actor_token` is provided. |
| `audience` | string | No | Intended audience for the issued token. |
| `resource` | string | No | URI of the target resource. |
| `scope` | string | No | Space-separated scopes for the issued token. Max 1024 chars. |

#### Supported Token Type URIs

| URI | Description |
|-----|-------------|
| `urn:ietf:params:oauth:token-type:access_token` | OAuth 2.0 access token |
| `urn:ietf:params:oauth:token-type:refresh_token` | OAuth 2.0 refresh token |
| `urn:ietf:params:oauth:token-type:id_token` | OpenID Connect ID token |
| `urn:ietf:params:oauth:token-type:jwt` | JSON Web Token |

### Response

#### Success — 200 OK

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "openid profile",
  "refresh_token": "def50200..."
}
```

| Field | Type | Description |
|-------|------|-------------|
| `access_token` | string | The newly issued token. |
| `issued_token_type` | string | Type URI of the issued token (RFC 8693 §2.2). |
| `token_type` | string | Always `Bearer` for access tokens. |
| `expires_in` | integer | Seconds until the issued token expires. |
| `scope` | string | Scopes granted. Omitted when not applicable. |
| `refresh_token` | string | Issued refresh token, if applicable. Omitted when not issued. |

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_request` | `subject_token` or `subject_token_type` is missing. |
| 400 | `invalid_request` | `subject_token_type` or `requested_token_type` is not a recognized URI. |
| 400 | `invalid_request` | `client_id` is missing. |
| 400 | `invalid_grant` | `subject_token` is invalid, expired, or does not match the expected type. |
| 401 | `invalid_client` | Client not found, inactive, or secret mismatch. |
| 500 | `server_error` | Unexpected internal error. |

### Example

```bash
curl -X POST https://auth.example.com/api/v1/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "service_client_id:service_client_secret" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "client_id=service_client_id" \
  -d "audience=https://api.example.com" \
  -d "scope=read+write"
```

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read write"
}
```
