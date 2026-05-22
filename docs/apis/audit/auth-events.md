# Auth Events

Auth events are immutable audit records that capture security-relevant activity across the tenant. Events are append-only and aligned with the OWASP Application Logging Vocabulary. Each event carries a category, event type, severity, result, and optional metadata. These endpoints provide read-only access for audit review and monitoring.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/auth-events` | List auth events with filters and pagination |
| GET | `/api/v1/auth-events/count` | Count events by event type |
| GET | `/api/v1/auth-events/{auth_event_uuid}` | Get a single auth event |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/auth-events

Returns a paginated list of auth events for the authenticated tenant. Events are ordered by `created_at` descending by default.

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `category` | string | Filter by event category. One of: `AUTHN`, `AUTHZ`, `SESSION`, `USER`, `SYSTEM` |
| `event_type` | string | Filter by event type (exact match), max 60 characters |
| `severity` | string | Filter by severity. One of: `INFO`, `WARN`, `CRITICAL` |
| `result` | string | Filter by outcome. One of: `success`, `failure` |
| `date_from` | string (ISO 8601) | Return events on or after this timestamp |
| `date_to` | string (ISO 8601) | Return events on or before this timestamp |
| `page` | integer | Page number (required, minimum 1) |
| `limit` | integer | Results per page (required, minimum 1, maximum 100) |
| `sort_by` | string | Field to sort by, max 50 characters |
| `sort_order` | string | Sort direction. One of: `asc`, `desc` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Auth events retrieved successfully",
  "data": {
    "rows": [
      {
        "auth_event_id": "d0e1f2a3-b4c5-6789-d012-6789abcdef01",
        "tenant_id": 42,
        "actor_user_id": 101,
        "target_user_id": null,
        "ip_address": "203.0.113.55",
        "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
        "category": "AUTHN",
        "event_type": "authn_login_success",
        "severity": "INFO",
        "result": "success",
        "description": "User authenticated successfully",
        "error_reason": null,
        "trace_id": "a1b2c3d4e5f6a7b8",
        "metadata": {
          "mfa_used": true,
          "device_fingerprint": "abc123"
        },
        "created_at": "2025-05-22T10:30:00Z"
      }
    ],
    "total": 1482,
    "page": 1,
    "limit": 20,
    "total_pages": 75
  }
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `auth_event_id` | string (UUID) | Unique event identifier |
| `tenant_id` | integer | Tenant that owns this event |
| `actor_user_id` | integer or null | ID of the user who performed the action |
| `target_user_id` | integer or null | ID of the user affected by the action |
| `ip_address` | string | Client IP address (up to 45 characters, supports IPv6) |
| `user_agent` | string or null | Client user agent string |
| `category` | string | Event category: `AUTHN`, `AUTHZ`, `SESSION`, `USER`, or `SYSTEM` |
| `event_type` | string | Specific event type identifier |
| `severity` | string | Severity level: `INFO`, `WARN`, or `CRITICAL` |
| `result` | string | Outcome: `success` or `failure` |
| `description` | string or null | Human-readable event description |
| `error_reason` | string or null | Error detail for failure events, max 255 characters |
| `trace_id` | string or null | Distributed trace identifier, max 32 characters |
| `metadata` | object or null | Event-specific contextual data |
| `created_at` | string (ISO 8601) | Event timestamp (immutable) |

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 422 | Invalid filter parameters |
| 500 | Internal server error |

### Example — Recent failures

```bash
curl "http://localhost:8080/api/v1/auth-events?result=failure&category=AUTHN&page=1&limit=50&sort_order=desc" \
  -H "Authorization: Bearer <token>"
```

### Example — Events within a time range

```bash
curl "http://localhost:8080/api/v1/auth-events?date_from=2025-05-01T00:00:00Z&date_to=2025-05-22T23:59:59Z&page=1&limit=100" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/auth-events/count

Returns the total count of auth events matching a specific event type for the authenticated tenant.

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `event_type` | string | Yes | The event type to count |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Auth event count retrieved successfully",
  "data": {
    "count": 347
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Missing `event_type` query parameter |
| 401 | Tenant not found in context |
| 500 | Internal server error |

### Example

```bash
curl "http://localhost:8080/api/v1/auth-events/count?event_type=authn_login_fail" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/auth-events/{auth_event_uuid}

Returns a single auth event by UUID. The event must belong to the authenticated tenant.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `auth_event_uuid` | string (UUID) | The event's unique identifier |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Auth event retrieved successfully",
  "data": {
    "auth_event_id": "d0e1f2a3-b4c5-6789-d012-6789abcdef01",
    "tenant_id": 42,
    "actor_user_id": 101,
    "target_user_id": null,
    "ip_address": "203.0.113.55",
    "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
    "category": "AUTHN",
    "event_type": "authn_login_success",
    "severity": "INFO",
    "result": "success",
    "description": "User authenticated successfully",
    "error_reason": null,
    "trace_id": "a1b2c3d4e5f6a7b8",
    "metadata": {
      "mfa_used": true
    },
    "created_at": "2025-05-22T10:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 401 | Tenant not found in context |
| 404 | Event not found or does not belong to this tenant |
| 500 | Internal server error |

### Example

```bash
curl "http://localhost:8080/api/v1/auth-events/d0e1f2a3-b4c5-6789-d012-6789abcdef01" \
  -H "Authorization: Bearer <token>"
```

---

## Event Reference

### Categories

| Value | Description |
|-------|-------------|
| `AUTHN` | Authentication — login, logout, token, MFA, OAuth |
| `AUTHZ` | Authorization — permission checks, privilege changes |
| `SESSION` | Session lifecycle — creation, renewal, expiry |
| `USER` | User account lifecycle — create, update, archive, delete |
| `SYSTEM` | Platform-level events — startup, shutdown, crash |

### Severity Levels

| Value | Description |
|-------|-------------|
| `INFO` | Normal operation, no action required |
| `WARN` | Anomalous or suspicious activity worth monitoring |
| `CRITICAL` | High-impact security event requiring immediate review |

### Result Values

| Value | Description |
|-------|-------------|
| `success` | The action completed as expected |
| `failure` | The action was denied, blocked, or encountered an error |

### Event Types — AUTHN

| Event Type | Severity | Description |
|------------|----------|-------------|
| `authn_login_success` | INFO | User authenticated successfully |
| `authn_login_fail` | WARN | Authentication attempt failed |
| `authn_login_fail_max` | WARN | Maximum failed attempts reached |
| `authn_login_lock` | WARN | Account locked after repeated failures |
| `authn_login_successafterfail` | INFO | Successful login following previous failures |
| `authn_password_change` | INFO | Password changed |
| `authn_password_change_fail` | WARN | Password change attempt failed |
| `authn_token_created` | INFO | Access or refresh token issued |
| `authn_token_revoked` | INFO | Token explicitly revoked |
| `authn_token_reuse` | CRITICAL | Reuse of a previously consumed token detected |
| `authn_token_delete` | INFO | Token deleted |
| `authn_impossible_travel` | CRITICAL | Login from geographically improbable location |
| `authn_oauth_authorize` | INFO | OAuth authorization request initiated |
| `authn_oauth_consent` | INFO | OAuth consent granted |
| `authn_oauth_consent_deny` | INFO | OAuth consent denied by user |
| `authn_oauth_token_exchange` | INFO | OAuth authorization code exchanged for tokens |
| `authn_oauth_token_refresh` | INFO | OAuth access token refreshed |
| `authn_oauth_token_revoke` | INFO | OAuth token revoked |
| `authn_oauth_client_auth` | INFO | OAuth client credentials authenticated |
| `authn_oauth_client_auth_fail` | WARN | OAuth client credentials authentication failed |

### Event Types — AUTHZ

| Event Type | Severity | Description |
|------------|----------|-------------|
| `authz_fail` | WARN | Authorization check denied |
| `authz_change` | INFO | Authorization policy modified |
| `authz_admin` | INFO | Administrative authorization action performed |

### Event Types — SESSION

| Event Type | Severity | Description |
|------------|----------|-------------|
| `session_created` | INFO | New session established |
| `session_renewed` | INFO | Session extended or refreshed |
| `session_expired` | INFO | Session expired naturally |
| `session_use_after_expire` | WARN | Attempt to use an expired session |

### Event Types — USER

| Event Type | Severity | Description |
|------------|----------|-------------|
| `user_created` | INFO | New user account created |
| `user_updated` | INFO | User profile or attributes updated |
| `user_archived` | INFO | User account archived |
| `user_deleted` | INFO | User account permanently deleted |

### Event Types — SYSTEM

| Event Type | Severity | Description |
|------------|----------|-------------|
| `sys_startup` | INFO | Service started |
| `sys_shutdown` | INFO | Service gracefully stopped |
| `sys_crash` | CRITICAL | Unexpected service termination |
