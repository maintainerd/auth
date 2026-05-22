# Roles

Roles group permissions and are assigned to users within a tenant. All role endpoints are tenant-scoped — the authenticated tenant context is validated by middleware and enforced at the service layer.

---

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/roles` | Bearer JWT | List all roles for the tenant |
| GET | `/api/v1/roles/{role_uuid}` | Bearer JWT | Get a specific role by UUID |
| POST | `/api/v1/roles` | Bearer JWT | Create a new role |
| PUT | `/api/v1/roles/{role_uuid}` | Bearer JWT | Update a role |
| PUT | `/api/v1/roles/{role_uuid}/status` | Bearer JWT | Update role status |
| DELETE | `/api/v1/roles/{role_uuid}` | Bearer JWT | Delete a role |
| GET | `/api/v1/roles/{role_uuid}/permissions` | Bearer JWT | List permissions on a role |
| POST | `/api/v1/roles/{role_uuid}/permissions` | Bearer JWT | Add permissions to a role |
| DELETE | `/api/v1/roles/{role_uuid}/permissions/{permission_uuid}` | Bearer JWT | Remove a permission from a role |

---

## GET /api/v1/roles

Returns a paginated list of roles belonging to the authenticated tenant.

### Authentication

Bearer JWT required.

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page` | integer | Yes | Page number (minimum: 1) |
| `limit` | integer | Yes | Results per page (minimum: 1) |
| `sort_by` | string | No | Field to sort by |
| `sort_order` | string | No | `asc` or `desc` |
| `name` | string | No | Filter by role name (partial match) |
| `description` | string | No | Filter by description (partial match) |
| `is_default` | boolean | No | Filter by default flag |
| `is_system` | boolean | No | Filter by system flag |
| `status` | string | No | Filter by status: `active` or `inactive` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Roles fetched successfully",
  "data": {
    "rows": [
      {
        "role_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "name": "admin",
        "description": "Full administrative access",
        "is_default": false,
        "is_system": false,
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
curl -X GET "http://localhost:8080/api/v1/roles?page=1&limit=10&status=active" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/roles/{role_uuid}

Returns a single role by its UUID. The service validates that the role belongs to the tenant.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `role_uuid` | UUID | The UUID of the role |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Role fetched successfully",
  "data": {
    "role_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "admin",
    "description": "Full administrative access",
    "is_default": false,
    "is_system": false,
    "status": "active",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid role UUID format |
| 401 | Missing or invalid JWT |
| 404 | Role not found or does not belong to tenant |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/roles/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/roles

Creates a new role associated with the authenticated tenant.

### Authentication

Bearer JWT required.

### Request Body

```json
{
  "name": "editor",
  "description": "Can read and write content",
  "status": "active"
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Role name. 3–20 characters. |
| `description` | string | Yes | Role description. 8–100 characters. |
| `status` | string | Yes | `active` or `inactive` |

### Response — 201 Created

```json
{
  "success": true,
  "message": "Role created successfully",
  "data": {
    "role_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "editor",
    "description": "Can read and write content",
    "is_default": false,
    "is_system": false,
    "status": "active",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Malformed JSON |
| 401 | Missing or invalid JWT |
| 422 | Validation error (e.g., name too short, missing description) |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/roles" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "editor", "description": "Can read and write content", "status": "active"}'
```

---

## PUT /api/v1/roles/{role_uuid}

Updates an existing role. The service validates that the role belongs to the tenant.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `role_uuid` | UUID | The UUID of the role to update |

### Request Body

```json
{
  "name": "senior-editor",
  "description": "Can manage all content and publish",
  "status": "active"
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Role name. 3–20 characters. |
| `description` | string | Yes | Role description. 8–100 characters. |
| `status` | string | Yes | `active` or `inactive` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Role updated successfully",
  "data": {
    "role_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "senior-editor",
    "description": "Can manage all content and publish",
    "is_default": false,
    "is_system": false,
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
| 404 | Role not found or does not belong to tenant |
| 422 | Validation error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/roles/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "senior-editor", "description": "Can manage all content and publish", "status": "active"}'
```

---

## PUT /api/v1/roles/{role_uuid}/status

Updates the status of a role without changing other fields.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `role_uuid` | UUID | The UUID of the role |

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
  "message": "Role updated successfully",
  "data": {
    "role_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "editor",
    "description": "Can read and write content",
    "is_default": false,
    "is_system": false,
    "status": "inactive",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID, malformed JSON, or invalid status value |
| 401 | Missing or invalid JWT |
| 404 | Role not found or does not belong to tenant |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/roles/a1b2c3d4-e5f6-7890-abcd-ef1234567890/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "inactive"}'
```

---

## DELETE /api/v1/roles/{role_uuid}

Soft-deletes a role. The service validates tenant ownership before deletion.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `role_uuid` | UUID | The UUID of the role to delete |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Role deleted successfully",
  "data": {
    "role_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "editor",
    "description": "Can read and write content",
    "is_default": false,
    "is_system": false,
    "status": "inactive",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid role UUID format |
| 401 | Missing or invalid JWT |
| 404 | Role not found or does not belong to tenant |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/roles/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/roles/{role_uuid}/permissions

Returns a paginated list of permissions assigned to a role. Includes API details for each permission when available.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `role_uuid` | UUID | The UUID of the role |

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page` | integer | Yes | Page number |
| `limit` | integer | Yes | Results per page |
| `sort_by` | string | No | Field to sort by |
| `sort_order` | string | No | `asc` or `desc` |
| `status` | string | No | Filter by status: `active` or `inactive` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Role permissions fetched successfully",
  "data": {
    "rows": [
      {
        "permission_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "name": "users:read",
        "description": "Read user records",
        "is_default": false,
        "is_system": false,
        "status": "active",
        "api": {
          "api_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
          "name": "users-api",
          "display_name": "Users API",
          "description": "Core user management API",
          "api_type": "rest",
          "identifier": "users",
          "status": "active",
          "created_at": "2024-01-10T08:00:00Z",
          "updated_at": "2024-01-10T08:00:00Z"
        },
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
| 400 | Invalid role UUID format |
| 401 | Missing or invalid JWT |
| 404 | Role not found or does not belong to tenant |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/roles/a1b2c3d4-e5f6-7890-abcd-ef1234567890/permissions?page=1&limit=10" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/roles/{role_uuid}/permissions

Adds one or more permissions to a role.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `role_uuid` | UUID | The UUID of the role |

### Request Body

```json
{
  "permissions": [
    "b2c3d4e5-f6a7-8901-bcde-f12345678901",
    "c3d4e5f6-a7b8-9012-cdef-123456789012"
  ]
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `permissions` | array of UUIDs | Yes | One or more permission UUIDs to add |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Permissions added to role successfully",
  "data": {
    "role_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "editor",
    "description": "Can read and write content",
    "is_default": false,
    "is_system": false,
    "status": "active",
    "permissions": [
      {
        "permission_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "name": "users:read",
        "description": "Read user records",
        "is_default": false,
        "is_system": false,
        "status": "active",
        "created_at": "2024-01-15T10:00:00Z",
        "updated_at": "2024-01-15T10:00:00Z"
      }
    ],
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid role UUID or malformed JSON |
| 401 | Missing or invalid JWT |
| 404 | Role not found or does not belong to tenant |
| 422 | Validation error (e.g., invalid permission UUID in array) |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/roles/a1b2c3d4-e5f6-7890-abcd-ef1234567890/permissions" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"permissions": ["b2c3d4e5-f6a7-8901-bcde-f12345678901"]}'
```

---

## DELETE /api/v1/roles/{role_uuid}/permissions/{permission_uuid}

Removes a specific permission from a role.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `role_uuid` | UUID | The UUID of the role |
| `permission_uuid` | UUID | The UUID of the permission to remove |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Permission removed from role successfully",
  "data": {
    "role_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "editor",
    "description": "Can read and write content",
    "is_default": false,
    "is_system": false,
    "status": "active",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid role UUID or permission UUID format |
| 401 | Missing or invalid JWT |
| 404 | Role or permission not found, or role does not belong to tenant |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/roles/a1b2c3d4-e5f6-7890-abcd-ef1234567890/permissions/b2c3d4e5-f6a7-8901-bcde-f12345678901" \
  -H "Authorization: Bearer <token>"
```
