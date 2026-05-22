# User Roles

Manage role assignments for a specific user. Assigning a role grants the user all permissions associated with that role within the tenant.

---

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/users/{user_uuid}/roles` | Bearer JWT | List all roles assigned to a user |
| POST | `/api/v1/users/{user_uuid}/roles` | Bearer JWT | Assign roles to a user |
| DELETE | `/api/v1/users/{user_uuid}/roles/{role_uuid}` | Bearer JWT | Remove a role from a user |

---

## GET /api/v1/users/{user_uuid}/roles

Returns a paginated list of roles assigned to the specified user. The user must belong to the authenticated tenant. Supports filtering by name, description, and status with in-memory sorting and pagination applied after fetching.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_uuid` | UUID | The UUID of the user |

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page` | integer | Yes | Page number (minimum: 1) |
| `limit` | integer | Yes | Results per page (minimum: 1) |
| `sort_by` | string | No | Field to sort by: `name`, `description`, `status`, `created_at`, `updated_at` |
| `sort_order` | string | No | `asc` or `desc` |
| `name` | string | No | Filter by role name (partial, case-insensitive match) |
| `description` | string | No | Filter by role description (partial, case-insensitive match) |
| `status` | string | No | Filter by status: `active` or `inactive` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "User roles fetched successfully",
  "data": {
    "rows": [
      {
        "role_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "name": "admin",
        "description": "Full administrative access",
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

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid user UUID format |
| 401 | Missing or invalid JWT, or tenant not found in context |
| 404 | User not found or does not belong to tenant |
| 422 | Validation error on query parameters |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890/roles?page=1&limit=10" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/users/{user_uuid}/roles

Assigns one or more roles to the specified user. The user must belong to the authenticated tenant.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_uuid` | UUID | The UUID of the user |

### Request Body

```json
{
  "role_ids": [
    "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "b2c3d4e5-f6a7-8901-bcde-f12345678901"
  ]
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `role_ids` | array of UUIDs | Yes | 1–10 role UUIDs to assign |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Roles assigned to user successfully",
  "data": {
    "user_id": "f1e2d3c4-b5a6-7890-abcd-ef1234567890",
    "username": "jdoe",
    "fullname": "John Doe",
    "email": "john@example.com",
    "phone": "+15551234567",
    "is_email_verified": true,
    "is_phone_verified": false,
    "is_profile_completed": true,
    "is_account_completed": true,
    "status": "active",
    "metadata": {},
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid user UUID or malformed JSON |
| 401 | Missing or invalid JWT |
| 404 | User not found or does not belong to tenant |
| 422 | Validation error (e.g., empty role list, more than 10 roles) |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/users/f1e2d3c4-b5a6-7890-abcd-ef1234567890/roles" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"role_ids": ["a1b2c3d4-e5f6-7890-abcd-ef1234567890"]}'
```

---

## DELETE /api/v1/users/{user_uuid}/roles/{role_uuid}

Removes a role from the specified user. Revoking a role removes all permissions that role granted to the user.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_uuid` | UUID | The UUID of the user |
| `role_uuid` | UUID | The UUID of the role to remove |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Role removed from user successfully",
  "data": {
    "user_id": "f1e2d3c4-b5a6-7890-abcd-ef1234567890",
    "username": "jdoe",
    "fullname": "John Doe",
    "email": "john@example.com",
    "phone": "+15551234567",
    "is_email_verified": true,
    "is_phone_verified": false,
    "is_profile_completed": true,
    "is_account_completed": true,
    "status": "active",
    "metadata": {},
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid user UUID or role UUID format |
| 401 | Missing or invalid JWT |
| 404 | User or role not found, or user does not belong to tenant |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/users/f1e2d3c4-b5a6-7890-abcd-ef1234567890/roles/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>"
```
