# Permissions

A **Permission** is a named, granular action that can be granted to clients, roles, or API keys. Permissions are scoped to an API and follow the format `resource:action` (e.g., `user:create`, `invoice:read`). They are the leaf-level building blocks of the access control system — policies reference permissions, and clients are granted permissions through their assigned APIs.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/permissions` | List permissions with filters and pagination |
| GET | `/api/v1/permissions/{permission_uuid}` | Get a single permission by UUID |
| POST | `/api/v1/permissions` | Create a new permission |
| PUT | `/api/v1/permissions/{permission_uuid}` | Update a permission |
| PUT | `/api/v1/permissions/{permission_uuid}/status` | Update permission status only |
| DELETE | `/api/v1/permissions/{permission_uuid}` | Delete a permission |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/permissions

Returns a paginated list of permissions for the authenticated tenant. Supports filtering by name, description, associated API, role, client, status, and flags.

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | integer | Page number (minimum: 1) |
| `limit` | integer | Results per page (minimum: 1) |
| `sort_by` | string | Field to sort by |
| `sort_order` | string | Sort direction: `asc` or `desc` |
| `name` | string | Filter by name (partial match) |
| `description` | string | Filter by description (partial match) |
| `api_id` | UUID | Filter by parent API UUID |
| `role_id` | UUID | Filter permissions assigned to this role UUID |
| `client_id` | UUID | Filter permissions assigned to this client UUID |
| `status` | string | Filter by status. One of: `active`, `inactive` |
| `is_default` | boolean | Filter by default flag (`true` or `false`) |
| `is_system` | boolean | Filter by system flag (`true` or `false`) |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Permissions fetched successfully",
  "data": {
    "rows": [
      {
        "permission_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "name": "user:create",
        "description": "Allows creating new users in the system",
        "api": {
          "api_id": "9c1a2b3d-4e5f-6789-abcd-ef0123456789",
          "name": "user-management",
          "display_name": "User Management",
          "description": "Manages user accounts",
          "api_type": "rest",
          "identifier": "user-management",
          "status": "active",
          "is_system": false,
          "created_at": "2024-01-15T10:00:00Z",
          "updated_at": "2024-01-15T10:00:00Z"
        },
        "status": "active",
        "is_default": false,
        "is_system": false,
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

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/permissions?page=1&limit=10&api_id=9c1a2b3d-4e5f-6789-abcd-ef0123456789" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/permissions/{permission_uuid}

Returns a single permission by its UUID, including its parent API.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `permission_uuid` | UUID | The UUID of the permission |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Permission fetched successfully",
  "data": {
    "permission_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "user:create",
    "description": "Allows creating new users in the system",
    "api": {
      "api_id": "9c1a2b3d-4e5f-6789-abcd-ef0123456789",
      "name": "user-management",
      "display_name": "User Management",
      "description": "Manages user accounts",
      "api_type": "rest",
      "identifier": "user-management",
      "status": "active",
      "is_system": false,
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z"
    },
    "status": "active",
    "is_default": false,
    "is_system": false,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid permission UUID format |
| 404 | Permission not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/permissions/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/permissions

Creates a new permission and associates it with the specified API. The `is_default` and `is_system` flags are managed by the system and cannot be set directly.

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Permission name. 3–50 characters. Convention: `resource:action`. |
| `description` | string | Yes | Description of what this permission grants. 8–200 characters. |
| `api_id` | UUID | Yes | UUID of the parent API this permission belongs to. |
| `status` | string | Yes | Initial status. One of: `active`, `inactive`. |

### Response — 201 Created

```json
{
  "success": true,
  "message": "Permission created successfully",
  "data": {
    "permission_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "user:create",
    "description": "Allows creating new users in the system",
    "api": {
      "api_id": "9c1a2b3d-4e5f-6789-abcd-ef0123456789",
      "name": "user-management",
      "display_name": "User Management",
      "description": "Manages user accounts",
      "api_type": "rest",
      "identifier": "user-management",
      "status": "active",
      "is_system": false,
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z"
    },
    "status": "active",
    "is_default": false,
    "is_system": false,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or validation failure |
| 404 | Referenced API not found |
| 409 | Permission name already exists for this API |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/permissions" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user:create",
    "description": "Allows creating new users in the system",
    "api_id": "9c1a2b3d-4e5f-6789-abcd-ef0123456789",
    "status": "active"
  }'
```

---

## PUT /api/v1/permissions/{permission_uuid}

Updates the name, description, and status of an existing permission. The parent API cannot be changed after creation.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `permission_uuid` | UUID | The UUID of the permission to update |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Permission name. 3–50 characters. |
| `description` | string | Yes | Description. 8–200 characters. |
| `status` | string | Yes | One of: `active`, `inactive`. |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Permission updated successfully",
  "data": {
    "permission_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "user:create",
    "description": "Allows creating new user accounts in the system",
    "api": {
      "api_id": "9c1a2b3d-4e5f-6789-abcd-ef0123456789",
      "name": "user-management",
      "display_name": "User Management",
      "description": "Manages user accounts",
      "api_type": "rest",
      "identifier": "user-management",
      "status": "active",
      "is_system": false,
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z"
    },
    "status": "active",
    "is_default": false,
    "is_system": false,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID or request body |
| 404 | Permission not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/permissions/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user:create",
    "description": "Allows creating new user accounts in the system",
    "status": "active"
  }'
```

---

## PUT /api/v1/permissions/{permission_uuid}/status

Updates the status of a permission without modifying other fields.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `permission_uuid` | UUID | The UUID of the permission |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | New status. One of: `active`, `inactive`. |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Permission status updated successfully",
  "data": {
    "permission_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "user:create",
    "description": "Allows creating new users in the system",
    "status": "inactive",
    "is_default": false,
    "is_system": false,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID or invalid status value |
| 404 | Permission not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/permissions/a1b2c3d4-e5f6-7890-abcd-ef1234567890/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "inactive"}'
```

---

## DELETE /api/v1/permissions/{permission_uuid}

Permanently deletes a permission. System and default permissions cannot be deleted.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `permission_uuid` | UUID | The UUID of the permission to delete |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Permission deleted successfully",
  "data": {
    "permission_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "user:create",
    "description": "Allows creating new users in the system",
    "status": "inactive",
    "is_default": false,
    "is_system": false,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID |
| 403 | Cannot delete a system or default permission |
| 404 | Permission not found |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/permissions/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>"
```
