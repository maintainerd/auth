# Tenant Settings

Tenant settings control operational configuration at the tenant level across four sub-resources: rate limiting, audit logging, maintenance mode, and feature flags. Each sub-resource is stored as an independent JSONB config block and updated in isolation.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/tenant-settings/rate-limit` | Get rate limit configuration |
| PUT | `/api/v1/tenant-settings/rate-limit` | Update rate limit configuration |
| GET | `/api/v1/tenant-settings/audit` | Get audit logging configuration |
| PUT | `/api/v1/tenant-settings/audit` | Update audit logging configuration |
| GET | `/api/v1/tenant-settings/maintenance` | Get maintenance mode configuration |
| PUT | `/api/v1/tenant-settings/maintenance` | Update maintenance mode configuration |
| GET | `/api/v1/tenant-settings/feature-flags` | Get feature flags |
| PUT | `/api/v1/tenant-settings/feature-flags` | Update feature flags |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/tenant-settings/rate-limit

Returns the current rate limit configuration for the authenticated tenant.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Rate limit config retrieved successfully",
  "data": {
    "requests_per_minute": 60,
    "burst_size": 10,
    "login_attempts_per_minute": 5,
    "enabled": true
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 500 | Internal server error |

---

## PUT /api/v1/tenant-settings/rate-limit

Updates rate limit configuration for the authenticated tenant.

### Request Body (application/json)

The request body is a free-form JSON object containing the rate limit fields to update. The object must not be empty.

| Field | Type | Description |
|-------|------|-------------|
| `requests_per_minute` | integer | Maximum API requests per minute per client |
| `burst_size` | integer | Allowed burst above the per-minute limit |
| `login_attempts_per_minute` | integer | Maximum login attempts per minute per IP |
| `enabled` | boolean | Whether rate limiting is active |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Rate limit config updated successfully",
  "data": {
    "requests_per_minute": 120,
    "burst_size": 20,
    "login_attempts_per_minute": 3,
    "enabled": true
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or empty config object |
| 401 | Tenant not found in context |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/tenant-settings/rate-limit" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"requests_per_minute": 120, "login_attempts_per_minute": 3}'
```

---

## GET /api/v1/tenant-settings/audit

Returns the current audit logging configuration for the authenticated tenant.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Audit config retrieved successfully",
  "data": {
    "enabled": true,
    "retention_days": 365,
    "log_successful_logins": true,
    "log_failed_logins": true,
    "log_admin_actions": true
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 500 | Internal server error |

---

## PUT /api/v1/tenant-settings/audit

Updates audit logging configuration for the authenticated tenant.

### Request Body (application/json)

The request body is a free-form JSON object containing the audit configuration fields to update. The object must not be empty.

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | boolean | Whether audit logging is active |
| `retention_days` | integer | Days to retain audit events |
| `log_successful_logins` | boolean | Record successful authentication events |
| `log_failed_logins` | boolean | Record failed authentication events |
| `log_admin_actions` | boolean | Record administrative configuration changes |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Audit config updated successfully",
  "data": {
    "enabled": true,
    "retention_days": 730,
    "log_successful_logins": true,
    "log_failed_logins": true,
    "log_admin_actions": true
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or empty config object |
| 401 | Tenant not found in context |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/tenant-settings/audit" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"enabled": true, "retention_days": 730, "log_admin_actions": true}'
```

---

## GET /api/v1/tenant-settings/maintenance

Returns the current maintenance mode configuration for the authenticated tenant.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Maintenance config retrieved successfully",
  "data": {
    "enabled": false,
    "message": "We are performing scheduled maintenance. Please try again later.",
    "allowed_ips": ["203.0.113.10"],
    "scheduled_start": null,
    "scheduled_end": null
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 500 | Internal server error |

---

## PUT /api/v1/tenant-settings/maintenance

Updates maintenance mode configuration for the authenticated tenant.

### Request Body (application/json)

The request body is a free-form JSON object containing the maintenance configuration fields to update. The object must not be empty.

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | boolean | Whether maintenance mode is currently active |
| `message` | string | Message displayed to users during maintenance |
| `allowed_ips` | array of strings | IP addresses that can bypass maintenance mode |
| `scheduled_start` | string (ISO 8601) | Planned maintenance start time |
| `scheduled_end` | string (ISO 8601) | Planned maintenance end time |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Maintenance config updated successfully",
  "data": {
    "enabled": true,
    "message": "Scheduled maintenance in progress.",
    "allowed_ips": ["203.0.113.10"],
    "scheduled_start": "2025-06-01T02:00:00Z",
    "scheduled_end": "2025-06-01T04:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or empty config object |
| 401 | Tenant not found in context |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/tenant-settings/maintenance" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"enabled": true, "message": "Scheduled maintenance in progress.", "allowed_ips": ["203.0.113.10"]}'
```

---

## GET /api/v1/tenant-settings/feature-flags

Returns the current feature flag configuration for the authenticated tenant.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Feature flags retrieved successfully",
  "data": {
    "magic_link_login": true,
    "social_login": false,
    "passkeys": false,
    "user_impersonation": true
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 500 | Internal server error |

---

## PUT /api/v1/tenant-settings/feature-flags

Updates feature flags for the authenticated tenant.

### Request Body (application/json)

The request body is a free-form JSON object containing the feature flag keys to update. The object must not be empty. Any boolean key-value pair is accepted; the available flags depend on the platform version.

| Field | Type | Description |
|-------|------|-------------|
| `magic_link_login` | boolean | Enable passwordless magic link login |
| `social_login` | boolean | Enable OAuth social login providers |
| `passkeys` | boolean | Enable WebAuthn / passkey authentication |
| `user_impersonation` | boolean | Allow admins to impersonate users |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Feature flags updated successfully",
  "data": {
    "magic_link_login": true,
    "social_login": true,
    "passkeys": false,
    "user_impersonation": true
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or empty config object |
| 401 | Tenant not found in context |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/tenant-settings/feature-flags" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"magic_link_login": true, "social_login": true}'
```
