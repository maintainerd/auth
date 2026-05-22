# Dynamic Client Registration

Dynamic Client Registration allows applications to register themselves as OAuth 2.0 clients programmatically without manual configuration. The server creates a new client and returns a `client_id` and, for confidential clients, a `client_secret`.

**RFC reference:** [RFC 7591 — OAuth 2.0 Dynamic Client Registration Protocol](https://www.rfc-editor.org/rfc/rfc7591)

---

## Endpoints

| Method | Path | Auth | Port |
|--------|------|------|------|
| POST | `/api/v1/oauth/register` | None (open) | 8081 |

---

## POST /api/v1/oauth/register

Registers a new OAuth 2.0 client. The request body is JSON. The server generates a `client_id` and, unless `token_endpoint_auth_method` is `none`, a `client_secret`.

Clients are registered under the system tenant. Providing `identity_provider_id` associates the client with a specific identity provider.

### Authentication

This endpoint is unauthenticated. Access control should be enforced at the network level if open registration is not desired.

### Request

#### Headers

| Header | Value |
|--------|-------|
| `Content-Type` | `application/json` |

#### Body

`application/json`

| Field | Type | Required | Constraints | Description |
|-------|------|----------|-------------|-------------|
| `client_name` | string | Yes | 1–255 chars | Human-readable name for the client application. |
| `redirect_uris` | array of strings | Yes | 1–10 entries | Redirect URIs for the authorization code flow. |
| `identity_provider_id` | integer | Yes | Positive integer | ID of the identity provider to associate this client with. |
| `grant_types` | array of strings | No | — | Grant types the client will use. Defaults to `["authorization_code"]` when omitted. |
| `response_types` | array of strings | No | — | Response types the client will use. Defaults to `["code"]` when omitted. |
| `scope` | string | No | Max 1024 chars | Space-separated list of scopes the client may request. |
| `token_endpoint_auth_method` | string | No | `client_secret_basic`, `client_secret_post`, or `none` | How the client authenticates at the token endpoint. Defaults to `client_secret_basic`. Set to `none` for public clients. |
| `logo_uri` | string | No | — | URI of the client's logo. |
| `policy_uri` | string | No | — | URI of the client's privacy policy. |
| `tos_uri` | string | No | — | URI of the client's terms of service. |
| `contacts` | array of strings | No | — | Email addresses of people responsible for the client. |

### Response

#### Success — 201 Created

```json
{
  "client_id": "7f3a9b2c1d4e5f6a8b9c0d1e",
  "client_secret": "sEcR3tV4Lu3g3n3r4t3dByS3rv3r...",
  "client_id_issued_at": 1716000000,
  "client_secret_expires_at": 0,
  "client_name": "My Application",
  "redirect_uris": ["https://app.example.com/callback"],
  "grant_types": ["authorization_code"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "client_secret_basic",
  "scope": "openid profile email"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `client_id` | string | Generated unique client identifier. |
| `client_secret` | string | Generated client secret. Present only for confidential clients (`token_endpoint_auth_method` is not `none`). |
| `client_id_issued_at` | integer | Unix timestamp when the client was registered. |
| `client_secret_expires_at` | integer | Unix timestamp when the client secret expires. `0` means it does not expire (per RFC 7591). |
| `client_name` | string | Registered client name. |
| `redirect_uris` | array | Registered redirect URIs. |
| `grant_types` | array | Registered grant types. |
| `response_types` | array | Registered response types. |
| `token_endpoint_auth_method` | string | Registered authentication method. |
| `scope` | string | Registered scopes. Omitted when not provided. |

For public clients (`token_endpoint_auth_method: "none"`), `client_secret` is omitted from the response.

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_request` | `client_name`, `redirect_uris`, or `identity_provider_id` is missing or invalid. |
| 400 | `invalid_request` | `token_endpoint_auth_method` has an unrecognized value. |
| 400 | — | Request body is not valid JSON. Returns `{ "success": false, "message": "invalid JSON body" }`. |
| 500 | `server_error` | Unexpected internal error. |

### Examples

**Confidential client (default):**

```bash
curl -X POST https://auth.example.com/api/v1/oauth/register \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "My Web App",
    "redirect_uris": ["https://app.example.com/callback"],
    "identity_provider_id": 1,
    "grant_types": ["authorization_code", "refresh_token"],
    "scope": "openid profile email offline_access",
    "token_endpoint_auth_method": "client_secret_basic"
  }'
```

```json
{
  "client_id": "7f3a9b2c1d4e5f6a8b9c0d1e",
  "client_secret": "sEcR3tV4Lu3g3n3r4t3dByS3rv3r...",
  "client_id_issued_at": 1716000000,
  "client_secret_expires_at": 0,
  "client_name": "My Web App",
  "redirect_uris": ["https://app.example.com/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "client_secret_basic",
  "scope": "openid profile email offline_access"
}
```

**Public client (SPA / mobile):**

```bash
curl -X POST https://auth.example.com/api/v1/oauth/register \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "My SPA",
    "redirect_uris": ["https://spa.example.com/callback"],
    "identity_provider_id": 1,
    "token_endpoint_auth_method": "none"
  }'
```

```json
{
  "client_id": "a1b2c3d4e5f6a7b8c9d0e1f2",
  "client_id_issued_at": 1716000000,
  "client_secret_expires_at": 0,
  "client_name": "My SPA",
  "redirect_uris": ["https://spa.example.com/callback"],
  "grant_types": ["authorization_code"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "none"
}
```
