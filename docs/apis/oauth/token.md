# Token Endpoint

Exchanges credentials for access tokens (RFC 6749 §4.1.3, §4.4, §6). The specific behavior depends on the `grant_type`. All requests must be form-encoded (`application/x-www-form-urlencoded`).

Client authentication is required for confidential clients. Public clients (registered with `token_endpoint_auth_method=none`) omit client credentials.

---

## Endpoints

| Method | Path | Auth |
|--------|------|------|
| POST | `/api/v1/oauth/token` | Client credentials (Basic or body) |

---

## POST /api/v1/oauth/token

### Authentication

Confidential clients authenticate using one of:

- **HTTP Basic Authentication** (`client_secret_basic`): `Authorization: Basic <base64(client_id:client_secret)>`
- **Request body** (`client_secret_post`): Include `client_id` and `client_secret` as form fields.

Public clients (registered with `token_endpoint_auth_method=none`) omit client credentials entirely.

### Request

#### Headers

| Name | Required | Description |
|------|----------|-------------|
| `Content-Type` | Yes | Must be `application/x-www-form-urlencoded`. |
| `Authorization` | Conditional | `Basic <base64(client_id:client_secret)>` for `client_secret_basic` auth. |

#### Request Body

All fields are form-encoded. Fields required per grant type are noted below.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `grant_type` | string | Yes | One of: `authorization_code`, `refresh_token`, `client_credentials`. |
| `code` | string | Conditional | Required for `authorization_code`. The authorization code received from the authorize endpoint. |
| `redirect_uri` | string | Conditional | Required for `authorization_code`. Must match the `redirect_uri` used in the authorization request. |
| `code_verifier` | string | Conditional | Required for `authorization_code`. The PKCE code verifier that corresponds to the `code_challenge` sent in the authorization request. |
| `refresh_token` | string | Conditional | Required for `refresh_token`. The refresh token to exchange. |
| `scope` | string | No | Requested scope. For `refresh_token`, can be used to request a subset of the original scopes. |
| `client_id` | string | Conditional | Required when using `client_secret_post` auth or for public clients. |
| `client_secret` | string | Conditional | Required when using `client_secret_post` auth. |

---

### grant_type=authorization_code

Exchanges a short-lived authorization code (issued by the `/authorize` endpoint) for tokens. Requires PKCE.

#### Example

```bash
curl -X POST "https://auth.example.com/api/v1/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "my-client-id:my-client-secret" \
  -d "grant_type=authorization_code" \
  -d "code=<authorization_code>" \
  -d "redirect_uri=https://app.example.com/callback" \
  -d "code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
```

---

### grant_type=refresh_token

Exchanges a refresh token for a new access token (and optionally a new refresh token). The original refresh token is consumed.

#### Example

```bash
curl -X POST "https://auth.example.com/api/v1/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "my-client-id:my-client-secret" \
  -d "grant_type=refresh_token" \
  -d "refresh_token=<refresh_token>"
```

---

### grant_type=client_credentials

Issues an access token directly to the client, without user involvement. Used for machine-to-machine authentication.

When the OAuth client is linked to an IAM service through `clients.service_id`,
the issued access token represents that service principal. The token subject is
the service name and includes `sub_type=service` plus `svc=<service name>`,
which allows the service to fetch
`GET /api/v1/services/me/policy-bundle` and participate in
[service-to-service authorization](../iam/authorization.md).

#### Example

```bash
curl -X POST "https://auth.example.com/api/v1/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "my-client-id:my-client-secret" \
  -d "grant_type=client_credentials" \
  -d "scope=openid"
```

---

### Response

#### Success — 200 OK

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "id_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "scope": "openid email"
}
```

| Field | Type | Always present | Description |
|-------|------|----------------|-------------|
| `access_token` | string | Yes | JWT access token. |
| `token_type` | string | Yes | Always `Bearer`. |
| `expires_in` | integer | Yes | Lifetime of the access token in seconds. |
| `refresh_token` | string | No | Included when `offline_access` scope was granted. |
| `id_token` | string | No | JWT ID token. Included when `openid` scope was granted. |
| `scope` | string | No | Space-delimited list of granted scopes. |

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_request` | A required parameter is missing or malformed. |
| 400 | `invalid_grant` | The authorization code or refresh token is invalid, expired, revoked, or the `redirect_uri` / `code_verifier` does not match. |
| 400 | `invalid_scope` | The requested scope is invalid or exceeds what was originally granted. |
| 400 | `unsupported_grant_type` | The `grant_type` is not supported. |
| 401 | `invalid_client` | Client authentication failed — unknown client, wrong secret, or unsupported auth method. |
| 401 | `unauthorized_client` | The client is not authorized to use the requested grant type. |
| 500 | `server_error` | An unexpected internal error occurred. |

```json
{ "error": "invalid_grant", "error_description": "authorization code has expired" }
```
