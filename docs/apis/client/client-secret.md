# Client Credentials and Configuration

These two read-only endpoints expose sensitive data that is intentionally separated from the main client object. Access to each is controlled by distinct permissions (`client:secret:read` and `client:config:read`).

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/clients/{client_uuid}/secret` | Retrieve the client ID and secret |
| GET | `/api/v1/clients/{client_uuid}/config` | Retrieve the full OAuth2 configuration |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/clients/{client_uuid}/secret

Returns the OAuth2 `client_id` (the public identifier used in token requests) and the `client_secret` (used only by confidential clients). For public clients such as `spa` and `mobile`, `client_secret` will be `null`.

> Store the client secret securely. It is not recoverable if lost — a new client must be created.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Auth client secret fetched successfully",
  "data": {
    "client_id": "8f2a1b3c4d5e6f7a",
    "client_secret": "cs_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  }
}
```

For public clients (`spa`, `mobile`) the secret field is `null`:

```json
{
  "success": true,
  "message": "Auth client secret fetched successfully",
  "data": {
    "client_id": "8f2a1b3c4d5e6f7a",
    "client_secret": null
  }
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `client_id` | string | The OAuth2 client identifier used in token requests and authorization URLs. |
| `client_secret` | string \| null | The client secret for confidential clients. `null` for public clients (`spa`, `mobile`). |

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid client UUID format |
| 401 | Unauthorized — missing or invalid token |
| 403 | Forbidden — token lacks `client:secret:read` permission |
| 404 | Client not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc/secret" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/clients/{client_uuid}/config

Returns the complete OAuth2/OIDC configuration for the client. This includes the `config` JSON block (token endpoint auth method, grant types, response types, TTLs, consent settings) along with the full client record as stored in the system.

This endpoint is useful for initializing OAuth2 library configurations and for debugging client setup.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client |

### Response — 200 OK

The `data` field contains the raw client configuration object as stored internally. The exact structure reflects the `config` JSON column merged with OAuth2 fields from the client record.

```json
{
  "success": true,
  "message": "Auth client config fetched successfully",
  "data": {
    "client_id": "8f2a1b3c4d5e6f7a",
    "client_type": "traditional",
    "domain": "app.example.com",
    "token_endpoint_auth_method": "client_secret_basic",
    "grant_types": ["authorization_code", "refresh_token"],
    "response_types": ["code"],
    "access_token_ttl": 3600,
    "refresh_token_ttl": 86400,
    "require_consent": true,
    "redirect_uris": [
      "https://app.example.com/callback"
    ],
    "logout_uris": [
      "https://app.example.com/logout"
    ]
  }
}
```

### Token Endpoint Auth Methods

| Value | Description |
|-------|-------------|
| `client_secret_basic` | Client authenticates using HTTP Basic auth with `client_id:client_secret`. Default for confidential clients. |
| `client_secret_post` | Client sends credentials in the POST body. |
| `none` | No authentication. Used for public clients (`spa`, `mobile`). |

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid client UUID format |
| 401 | Unauthorized — missing or invalid token |
| 403 | Forbidden — token lacks `client:config:read` permission |
| 404 | Client not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc/config" \
  -H "Authorization: Bearer <token>"
```
