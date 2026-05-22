# API Keys

API keys provide machine-to-machine authentication for tenant resources. Each key is scoped to a tenant, optionally rate-limited, and can carry a structured configuration object. The plain key value is only returned once at creation time — store it securely.

---

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/api_keys` | Bearer JWT | List all API keys for the tenant |
| GET | `/api/v1/api_keys/{api_key_uuid}` | Bearer JWT | Get a specific API key by UUID |
| POST | `/api/v1/api_keys` | Bearer JWT | Create a new API key |
| PUT | `/api/v1/api_keys/{api_key_uuid}` | Bearer JWT | Update an API key |
| PUT | `/api/v1/api_keys/{api_key_uuid}/status` | Bearer JWT | Update API key status |
| DELETE | `/api/v1/api_keys/{api_key_uuid}` | Bearer JWT | Delete an API key |
| GET | `/api/v1/api_keys/{api_key_uuid}/config` | Bearer JWT | Get the configuration for an API key |

---

## GET /api/v1/api_keys

Returns a paginated list of API keys for the authenticated tenant. Results default to page 1, limit 10, sorted by `created_at` descending.

### Authentication

Bearer JWT required.

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page` | integer | No | Page number. Default: `1` |
| `limit` | integer | No | Results per page. Default: `10` |
| `sort_by` | string | No | Field to sort by. Default: `created_at` |
| `sort_order` | string | No | `asc` or `desc`. Default: `desc` |
| `name` | string | No | Filter by name |
| `description` | string | No | Filter by description |
| `status` | string | No | Filter by status: `active` or `inactive` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "API keys fetched successfully",
  "data": {
    "rows": [
      {
        "api_key_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "name": "production-key",
        "description": "Key for production service",
        "key_prefix": "mk_live_",
        "expires_at": "2025-01-15T10:00:00Z",
        "rate_limit": 1000,
        "status": "active",
        "created_at": "2024-01-15T10:00:00Z",
        "updated_at": "2024-01-15T10:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 10,
    "total_pages": 1
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Missing or invalid JWT, or tenant not found in context |
| 422 | Validation error on query parameters |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/api_keys?page=1&limit=10&status=active" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/api_keys/{api_key_uuid}

Returns a single API key by UUID. The service validates that the key belongs to the tenant.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_key_uuid` | UUID | The UUID of the API key |

### Response — 200 OK

```json
{
  "success": true,
  "message": "API key fetched successfully",
  "data": {
    "api_key_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "production-key",
    "description": "Key for production service",
    "key_prefix": "mk_live_",
    "expires_at": "2025-01-15T10:00:00Z",
    "rate_limit": 1000,
    "status": "active",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid API key UUID format |
| 401 | Missing or invalid JWT |
| 404 | API key not found or does not belong to tenant |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/api_keys/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/api_keys

Creates a new API key for the tenant. The response includes the full plain-text key in the `key` field — this value is shown **only once** and cannot be retrieved again.

### Authentication

Bearer JWT required.

### Request Body

```json
{
  "name": "production-key",
  "description": "Key for production service",
  "config": {
    "allowed_ips": ["203.0.113.0/24"]
  },
  "expires_at": "2025-01-15T10:00:00Z",
  "rate_limit": 1000,
  "status": "active"
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | 1–100 characters |
| `description` | string | No | Up to 500 characters |
| `config` | object | No | Arbitrary JSON configuration for this key |
| `expires_at` | datetime | No | ISO 8601 expiration datetime. No expiry if omitted. |
| `rate_limit` | integer | No | Maximum requests per period. Minimum: 1. |
| `status` | string | No | `active` or `inactive`. Default: `active`. |

### Response — 201 Created

```json
{
  "success": true,
  "message": "API key created successfully",
  "data": {
    "api_key_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "production-key",
    "description": "Key for production service",
    "key_prefix": "mk_live_",
    "key": "mk_live_aB3cD4eF5gH6iJ7kL8mN9oP0",
    "expires_at": "2025-01-15T10:00:00Z",
    "rate_limit": 1000,
    "status": "active",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

The `key` field contains the full API key value. It is only present in the creation response. Store it securely — it will not be returned again.

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Malformed JSON |
| 401 | Missing or invalid JWT |
| 422 | Validation error (e.g., name too long, invalid status, rate_limit less than 1) |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/api_keys" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "production-key",
    "description": "Key for production service",
    "rate_limit": 1000,
    "status": "active"
  }'
```

---

## PUT /api/v1/api_keys/{api_key_uuid}

Updates an existing API key. The service validates tenant ownership.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_key_uuid` | UUID | The UUID of the API key to update |

### Request Body

```json
{
  "name": "updated-key",
  "description": "Updated description",
  "config": {},
  "expires_at": "2026-01-15T10:00:00Z",
  "rate_limit": 2000,
  "status": "active"
}
```

### Request Fields

All fields are optional. Only the fields provided are updated.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | No | 1–100 characters |
| `description` | string | No | Up to 500 characters |
| `config` | object | No | Replacement JSON configuration |
| `expires_at` | datetime | No | New expiration datetime |
| `rate_limit` | integer | No | Minimum: 1 |
| `status` | string | No | `active` or `inactive` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "API key updated successfully",
  "data": {
    "api_key_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "updated-key",
    "description": "Updated description",
    "key_prefix": "mk_live_",
    "expires_at": "2026-01-15T10:00:00Z",
    "rate_limit": 2000,
    "status": "active",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID or malformed JSON |
| 401 | Missing or invalid JWT |
| 404 | API key not found or does not belong to tenant |
| 422 | Validation error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/api_keys/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "updated-key", "rate_limit": 2000}'
```

---

## PUT /api/v1/api_keys/{api_key_uuid}/status

Updates only the status of an API key. Use this to quickly enable or disable a key without a full update.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_key_uuid` | UUID | The UUID of the API key |

### Request Body

```json
{
  "status": "inactive"
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | `active` or `inactive` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "API key status updated successfully",
  "data": {
    "api_key_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "production-key",
    "description": "Key for production service",
    "key_prefix": "mk_live_",
    "expires_at": "2025-01-15T10:00:00Z",
    "rate_limit": 1000,
    "status": "inactive",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID or malformed JSON |
| 401 | Missing or invalid JWT |
| 404 | API key not found or does not belong to tenant |
| 422 | Validation error (invalid status value) |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/api_keys/a1b2c3d4-e5f6-7890-abcd-ef1234567890/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "inactive"}'
```

---

## DELETE /api/v1/api_keys/{api_key_uuid}

Deletes an API key. The service validates tenant ownership.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_key_uuid` | UUID | The UUID of the API key to delete |

### Response — 200 OK

```json
{
  "success": true,
  "message": "API key deleted successfully",
  "data": {
    "api_key_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "production-key",
    "description": "Key for production service",
    "key_prefix": "mk_live_",
    "status": "inactive",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid API key UUID format |
| 401 | Missing or invalid JWT |
| 404 | API key not found or does not belong to tenant |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/api_keys/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/api_keys/{api_key_uuid}/config

Returns the raw configuration object stored on the API key. The service validates tenant ownership.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_key_uuid` | UUID | The UUID of the API key |

### Response — 200 OK

```json
{
  "success": true,
  "message": "API key config fetched successfully",
  "data": {
    "allowed_ips": ["203.0.113.0/24"],
    "custom_field": "custom_value"
  }
}
```

The `data` field contains the raw config object as stored — its structure is defined by the application creating or updating the key.

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid API key UUID format |
| 401 | Missing or invalid JWT |
| 404 | API key not found or does not belong to tenant |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/api_keys/a1b2c3d4-e5f6-7890-abcd-ef1234567890/config" \
  -H "Authorization: Bearer <token>"
```
