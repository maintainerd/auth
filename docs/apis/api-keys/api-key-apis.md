# API Key — APIs and Permissions

An API key can be scoped to one or more APIs and, within each API, to specific permissions. This enables fine-grained access control: the key only grants access to the APIs and permissions explicitly assigned to it.

---

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/api_keys/{api_key_uuid}/apis` | Bearer JWT | List APIs assigned to an API key |
| POST | `/api/v1/api_keys/{api_key_uuid}/apis` | Bearer JWT | Add APIs to an API key |
| DELETE | `/api/v1/api_keys/{api_key_uuid}/apis/{api_uuid}` | Bearer JWT | Remove an API from an API key |
| GET | `/api/v1/api_keys/{api_key_uuid}/apis/{api_uuid}/permissions` | Bearer JWT | List permissions for an API on an API key |
| POST | `/api/v1/api_keys/{api_key_uuid}/apis/{api_uuid}/permissions` | Bearer JWT | Add permissions for an API on an API key |
| DELETE | `/api/v1/api_keys/{api_key_uuid}/apis/{api_uuid}/permissions/{permission_uuid}` | Bearer JWT | Remove a permission for an API from an API key |

---

## GET /api/v1/api_keys/{api_key_uuid}/apis

Returns a paginated list of APIs assigned to the API key.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_key_uuid` | UUID | The UUID of the API key |

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page` | integer | No | Page number. Default: `1` |
| `limit` | integer | No | Results per page. Default: `10` |
| `sort_by` | string | No | Field to sort by |
| `sort_order` | string | No | `asc` or `desc` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "API key APIs retrieved successfully",
  "data": {
    "rows": [
      {
        "api_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "name": "users-api",
        "display_name": "Users API",
        "description": "Core user management API",
        "api_type": "rest",
        "identifier": "users",
        "status": "active",
        "is_system": false,
        "created_at": "2024-01-10T08:00:00Z",
        "updated_at": "2024-01-10T08:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 10,
    "total_pages": 1
  }
}
```

### Response Fields (per row)

| Field | Type | Description |
|-------|------|-------------|
| `api_id` | UUID | Unique identifier for the API |
| `name` | string | Internal name |
| `display_name` | string | Human-readable name |
| `description` | string | Description of the API |
| `api_type` | string | Type of API (e.g., `rest`) |
| `identifier` | string | Machine-readable identifier string |
| `status` | string | `active` or `inactive` |
| `is_system` | boolean | Whether this is a system-managed API |
| `created_at` | datetime | |
| `updated_at` | datetime | |

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid API key UUID format |
| 401 | Missing or invalid JWT |
| 422 | Validation error on query parameters |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/api_keys/a1b2c3d4-e5f6-7890-abcd-ef1234567890/apis?page=1&limit=10" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/api_keys/{api_key_uuid}/apis

Adds one or more APIs to the API key.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_key_uuid` | UUID | The UUID of the API key |

### Request Body

```json
{
  "api_uuids": [
    "b2c3d4e5-f6a7-8901-bcde-f12345678901",
    "c3d4e5f6-a7b8-9012-cdef-123456789012"
  ]
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `api_uuids` | array of UUIDs | Yes | One or more API UUIDs to assign |

### Response — 200 OK

```json
{
  "success": true,
  "message": "APIs added to API key successfully",
  "data": null
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid API key UUID or malformed JSON |
| 401 | Missing or invalid JWT |
| 422 | Validation error (e.g., invalid UUID in array) |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/api_keys/a1b2c3d4-e5f6-7890-abcd-ef1234567890/apis" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"api_uuids": ["b2c3d4e5-f6a7-8901-bcde-f12345678901"]}'
```

---

## DELETE /api/v1/api_keys/{api_key_uuid}/apis/{api_uuid}

Removes an API from the API key. Any permissions associated with this API on this key are also removed.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_key_uuid` | UUID | The UUID of the API key |
| `api_uuid` | UUID | The UUID of the API to remove |

### Response — 200 OK

```json
{
  "success": true,
  "message": "API removed from API key successfully",
  "data": null
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 401 | Missing or invalid JWT |
| 404 | API key or API assignment not found |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/api_keys/a1b2c3d4-e5f6-7890-abcd-ef1234567890/apis/b2c3d4e5-f6a7-8901-bcde-f12345678901" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/api_keys/{api_key_uuid}/apis/{api_uuid}/permissions

Returns all permissions granted to the API key for a specific API.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_key_uuid` | UUID | The UUID of the API key |
| `api_uuid` | UUID | The UUID of the API |

### Response — 200 OK

```json
{
  "success": true,
  "message": "API key API permissions retrieved successfully",
  "data": {
    "permissions": [
      {
        "permission_id": "d4e5f6a7-b8c9-0123-def0-234567890123",
        "name": "users:read",
        "description": "Read user records",
        "is_default": false,
        "is_system": false,
        "status": "active",
        "created_at": "2024-01-10T08:00:00Z",
        "updated_at": "2024-01-10T08:00:00Z"
      }
    ]
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 401 | Missing or invalid JWT |
| 404 | API key or API not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/api_keys/a1b2c3d4-e5f6-7890-abcd-ef1234567890/apis/b2c3d4e5-f6a7-8901-bcde-f12345678901/permissions" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/api_keys/{api_key_uuid}/apis/{api_uuid}/permissions

Adds one or more permissions for a specific API to the API key.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_key_uuid` | UUID | The UUID of the API key |
| `api_uuid` | UUID | The UUID of the API |

### Request Body

```json
{
  "permission_uuids": [
    "d4e5f6a7-b8c9-0123-def0-234567890123",
    "e5f6a7b8-c9d0-1234-ef01-345678901234"
  ]
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `permission_uuids` | array of UUIDs | Yes | One or more permission UUIDs to grant |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Permissions added to API key API successfully",
  "data": null
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format or malformed JSON |
| 401 | Missing or invalid JWT |
| 422 | Validation error (e.g., invalid UUID in array) |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/api_keys/a1b2c3d4-e5f6-7890-abcd-ef1234567890/apis/b2c3d4e5-f6a7-8901-bcde-f12345678901/permissions" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"permission_uuids": ["d4e5f6a7-b8c9-0123-def0-234567890123"]}'
```

---

## DELETE /api/v1/api_keys/{api_key_uuid}/apis/{api_uuid}/permissions/{permission_uuid}

Removes a specific permission for an API from the API key.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_key_uuid` | UUID | The UUID of the API key |
| `api_uuid` | UUID | The UUID of the API |
| `permission_uuid` | UUID | The UUID of the permission to remove |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Permission removed from API key API successfully",
  "data": null
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 401 | Missing or invalid JWT |
| 404 | Permission assignment not found |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/api_keys/a1b2c3d4-e5f6-7890-abcd-ef1234567890/apis/b2c3d4e5-f6a7-8901-bcde-f12345678901/permissions/d4e5f6a7-b8c9-0123-def0-234567890123" \
  -H "Authorization: Bearer <token>"
```
