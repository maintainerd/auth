# Webhook Endpoints

Webhook endpoints receive HTTP POST callbacks when auth events occur in the tenant. Each endpoint subscribes to a list of event types and is called with a signed request body when a matching event fires. Configuration includes a secret for HMAC signature verification, retry count, and timeout.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/webhook-endpoints` | List webhook endpoints |
| GET | `/api/v1/webhook-endpoints/{webhook_endpoint_uuid}` | Get a single endpoint |
| POST | `/api/v1/webhook-endpoints` | Create an endpoint |
| PUT | `/api/v1/webhook-endpoints/{webhook_endpoint_uuid}` | Update an endpoint |
| DELETE | `/api/v1/webhook-endpoints/{webhook_endpoint_uuid}` | Delete an endpoint |
| PATCH | `/api/v1/webhook-endpoints/{webhook_endpoint_uuid}/status` | Update endpoint status |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/webhook-endpoints

Returns a paginated list of webhook endpoints for the authenticated tenant.

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | string | Filter by status. One of: `active`, `inactive` |
| `page` | integer | Page number (required, minimum 1) |
| `limit` | integer | Results per page (required, minimum 1) |
| `sort_by` | string | Field to sort by, max 50 characters |
| `sort_order` | string | Sort direction. One of: `asc`, `desc` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Webhook endpoints retrieved successfully",
  "data": {
    "rows": [
      {
        "webhook_endpoint_id": "c9d0e1f2-a3b4-5678-c012-56789abcdef0",
        "url": "https://app.example.com/hooks/auth",
        "events": ["authn_login_success", "authn_login_fail", "user_created"],
        "max_retries": 3,
        "timeout_seconds": 30,
        "status": "active",
        "description": "Main application webhook",
        "last_triggered_at": "2025-05-21T18:42:00Z",
        "created_at": "2025-03-10T09:00:00Z",
        "updated_at": "2025-05-01T11:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 20,
    "total_pages": 1
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 422 | Invalid filter parameters |
| 500 | Internal server error |

---

## GET /api/v1/webhook-endpoints/{webhook_endpoint_uuid}

Returns a single webhook endpoint by UUID.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `webhook_endpoint_uuid` | string (UUID) | The endpoint's unique identifier |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Webhook endpoint retrieved successfully",
  "data": {
    "webhook_endpoint_id": "c9d0e1f2-a3b4-5678-c012-56789abcdef0",
    "url": "https://app.example.com/hooks/auth",
    "events": ["authn_login_success", "authn_login_fail", "user_created"],
    "max_retries": 3,
    "timeout_seconds": 30,
    "status": "active",
    "description": "Main application webhook",
    "last_triggered_at": "2025-05-21T18:42:00Z",
    "created_at": "2025-03-10T09:00:00Z",
    "updated_at": "2025-05-01T11:00:00Z"
  }
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `webhook_endpoint_id` | string (UUID) | Unique identifier |
| `url` | string | HTTPS URL that receives event callbacks |
| `events` | array of strings | Subscribed event types |
| `max_retries` | integer | Maximum retry attempts on delivery failure (0–10) |
| `timeout_seconds` | integer | Request timeout in seconds (1–120) |
| `status` | string | `active` or `inactive` |
| `description` | string | Human-readable description |
| `last_triggered_at` | string (ISO 8601) or null | Timestamp of the most recent delivery attempt |
| `created_at` | string (ISO 8601) | Creation timestamp |
| `updated_at` | string (ISO 8601) | Last update timestamp |

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 401 | Tenant not found in context |
| 404 | Endpoint not found or does not belong to this tenant |
| 500 | Internal server error |

---

## POST /api/v1/webhook-endpoints

Creates a new webhook endpoint.

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | Destination URL. Must be a valid URL |
| `events` | array of strings | Yes | List of event types to subscribe to. Each event name 1–100 characters |
| `secret` | string | No | Signing secret for HMAC signature verification (write-only) |
| `max_retries` | integer | No | Retry count on failure. Range: 0–10 |
| `timeout_seconds` | integer | No | Request timeout. Range: 1–120 seconds |
| `description` | string | No | Human-readable description, max 500 characters |
| `status` | string | No | Initial status. One of: `active`, `inactive`. Defaults to `active` |

### Response — 201 Created

```json
{
  "success": true,
  "message": "Webhook endpoint created successfully",
  "data": {
    "webhook_endpoint_id": "c9d0e1f2-a3b4-5678-c012-56789abcdef0",
    "url": "https://app.example.com/hooks/auth",
    "events": ["authn_login_success", "authn_login_fail"],
    "max_retries": 3,
    "timeout_seconds": 30,
    "status": "active",
    "description": "Main application webhook",
    "last_triggered_at": null,
    "created_at": "2025-05-22T12:00:00Z",
    "updated_at": "2025-05-22T12:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Malformed JSON request body |
| 401 | Tenant not found in context |
| 422 | Validation error — see response body for field-level details |
| 500 | Internal server error |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/webhook-endpoints" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://app.example.com/hooks/auth",
    "events": ["authn_login_success", "authn_login_fail", "user_created"],
    "secret": "your-signing-secret",
    "max_retries": 3,
    "timeout_seconds": 30,
    "description": "Main application webhook"
  }'
```

---

## PUT /api/v1/webhook-endpoints/{webhook_endpoint_uuid}

Updates an existing webhook endpoint.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `webhook_endpoint_uuid` | string (UUID) | The endpoint's unique identifier |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | Destination URL. Must be a valid URL |
| `events` | array of strings | Yes | Subscribed event types. Each event name 1–100 characters |
| `secret` | string | No | Updated signing secret (write-only) |
| `max_retries` | integer | No | Retry count on failure. Range: 0–10 |
| `timeout_seconds` | integer | No | Request timeout. Range: 1–120 seconds |
| `description` | string | No | Human-readable description, max 500 characters |
| `status` | string | No | Status. One of: `active`, `inactive`. Defaults to `active` if omitted |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Webhook endpoint updated successfully",
  "data": {
    "webhook_endpoint_id": "c9d0e1f2-a3b4-5678-c012-56789abcdef0",
    "url": "https://app.example.com/hooks/auth-v2",
    "events": ["authn_login_success", "authn_login_fail", "user_created", "user_updated"],
    "max_retries": 5,
    "timeout_seconds": 60,
    "status": "active",
    "description": "Updated webhook",
    "last_triggered_at": "2025-05-21T18:42:00Z",
    "created_at": "2025-03-10T09:00:00Z",
    "updated_at": "2025-05-22T12:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format or malformed request body |
| 401 | Tenant not found in context |
| 404 | Endpoint not found or does not belong to this tenant |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/webhook-endpoints/c9d0e1f2-a3b4-5678-c012-56789abcdef0" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://app.example.com/hooks/auth-v2",
    "events": ["authn_login_success", "authn_login_fail", "user_created"],
    "max_retries": 5,
    "timeout_seconds": 60
  }'
```

---

## DELETE /api/v1/webhook-endpoints/{webhook_endpoint_uuid}

Deletes a webhook endpoint. Returns the deleted endpoint's data.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `webhook_endpoint_uuid` | string (UUID) | The endpoint's unique identifier |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Webhook endpoint deleted successfully",
  "data": {
    "webhook_endpoint_id": "c9d0e1f2-a3b4-5678-c012-56789abcdef0",
    "url": "https://app.example.com/hooks/auth",
    "events": ["authn_login_success"],
    "max_retries": 3,
    "timeout_seconds": 30,
    "status": "inactive",
    "description": "Main application webhook",
    "last_triggered_at": null,
    "created_at": "2025-03-10T09:00:00Z",
    "updated_at": "2025-05-22T12:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 401 | Tenant not found in context |
| 404 | Endpoint not found or does not belong to this tenant |
| 500 | Internal server error |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/webhook-endpoints/c9d0e1f2-a3b4-5678-c012-56789abcdef0" \
  -H "Authorization: Bearer <token>"
```

---

## PATCH /api/v1/webhook-endpoints/{webhook_endpoint_uuid}/status

Updates only the status of a webhook endpoint.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `webhook_endpoint_uuid` | string (UUID) | The endpoint's unique identifier |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | New status. One of: `active`, `inactive` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Webhook endpoint status updated successfully",
  "data": {
    "webhook_endpoint_id": "c9d0e1f2-a3b4-5678-c012-56789abcdef0",
    "url": "https://app.example.com/hooks/auth",
    "events": ["authn_login_success", "authn_login_fail"],
    "max_retries": 3,
    "timeout_seconds": 30,
    "status": "inactive",
    "description": "Main application webhook",
    "last_triggered_at": "2025-05-21T18:42:00Z",
    "created_at": "2025-03-10T09:00:00Z",
    "updated_at": "2025-05-22T12:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format or malformed request body |
| 401 | Tenant not found in context |
| 404 | Endpoint not found or does not belong to this tenant |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PATCH "http://localhost:8080/api/v1/webhook-endpoints/c9d0e1f2-a3b4-5678-c012-56789abcdef0/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "inactive"}'
```

---

## Event Types

The `events` array accepts any string event type. The following standard event types are emitted by the platform:

### AUTHN — Authentication Events

| Event Type | Description |
|------------|-------------|
| `authn_login_success` | Successful user login |
| `authn_login_fail` | Failed login attempt |
| `authn_login_fail_max` | Maximum failed login attempts reached |
| `authn_login_lock` | Account locked after too many failures |
| `authn_login_successafterfail` | Successful login following previous failures |
| `authn_password_change` | Password changed successfully |
| `authn_password_change_fail` | Password change attempt failed |
| `authn_token_created` | Authentication token issued |
| `authn_token_revoked` | Token explicitly revoked |
| `authn_token_reuse` | Reuse of a consumed token detected |
| `authn_token_delete` | Token deleted |
| `authn_impossible_travel` | Login from geographically improbable location |
| `authn_oauth_authorize` | OAuth authorization request |
| `authn_oauth_consent` | OAuth consent granted |
| `authn_oauth_consent_deny` | OAuth consent denied |
| `authn_oauth_token_exchange` | OAuth token exchange |
| `authn_oauth_token_refresh` | OAuth token refresh |
| `authn_oauth_token_revoke` | OAuth token revoked |
| `authn_oauth_client_auth` | OAuth client authentication success |
| `authn_oauth_client_auth_fail` | OAuth client authentication failure |

### AUTHZ — Authorization Events

| Event Type | Description |
|------------|-------------|
| `authz_fail` | Authorization check failed |
| `authz_change` | Authorization policy changed |
| `authz_admin` | Administrative authorization action |

### SESSION — Session Events

| Event Type | Description |
|------------|-------------|
| `session_created` | New session established |
| `session_renewed` | Session renewed or refreshed |
| `session_expired` | Session expired naturally |
| `session_use_after_expire` | Attempt to use an expired session |

### USER — User Lifecycle Events

| Event Type | Description |
|------------|-------------|
| `user_created` | New user account created |
| `user_updated` | User profile or attributes updated |
| `user_archived` | User account archived |
| `user_deleted` | User account deleted |

### SYSTEM — System Events

| Event Type | Description |
|------------|-------------|
| `sys_startup` | Service started |
| `sys_shutdown` | Service stopped |
| `sys_crash` | Unexpected service crash |
