# Signup Flows

Signup flows define the registration experience for users within a tenant. Each flow is associated with an auth client and carries a configuration object that controls the signup process. Roles assigned to a flow are automatically granted to users who complete registration through that flow.

---

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/signup_flows` | Bearer JWT | List all signup flows for the tenant |
| GET | `/api/v1/signup_flows/{signup_flow_uuid}` | Bearer JWT | Get a specific signup flow by UUID |
| POST | `/api/v1/signup_flows` | Bearer JWT | Create a new signup flow |
| PUT | `/api/v1/signup_flows/{signup_flow_uuid}` | Bearer JWT | Update a signup flow |
| PATCH | `/api/v1/signup_flows/{signup_flow_uuid}/status` | Bearer JWT | Update signup flow status |
| DELETE | `/api/v1/signup_flows/{signup_flow_uuid}` | Bearer JWT | Delete a signup flow |
| GET | `/api/v1/signup_flows/{signup_flow_uuid}/roles` | Bearer JWT | List roles assigned to a signup flow |
| POST | `/api/v1/signup_flows/{signup_flow_uuid}/roles` | Bearer JWT | Assign roles to a signup flow |
| DELETE | `/api/v1/signup_flows/{signup_flow_uuid}/roles/{role_uuid}` | Bearer JWT | Remove a role from a signup flow |

---

## GET /api/v1/signup_flows

Returns a paginated list of signup flows belonging to the authenticated tenant.

### Authentication

Bearer JWT required.

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page` | integer | Yes | Page number (minimum: 1) |
| `limit` | integer | Yes | Results per page (minimum: 1) |
| `sort_by` | string | No | Field to sort by |
| `sort_order` | string | No | `asc` or `desc` |
| `name` | string | No | Filter by name (partial match) |
| `identifier` | string | No | Filter by identifier (partial match) |
| `status` | string | No | Filter by status: `active` or `inactive` |
| `client_id` | UUID | No | Filter by associated auth client UUID |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Signup flows retrieved successfully",
  "data": {
    "rows": [
      {
        "signup_flow_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "name": "Default Signup",
        "description": "Standard user registration flow",
        "identifier": "default-signup",
        "config": {
          "require_email_verification": true,
          "allow_social_login": false
        },
        "status": "active",
        "client_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
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
curl -X GET "http://localhost:8080/api/v1/signup_flows?page=1&limit=10&status=active" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/signup_flows/{signup_flow_uuid}

Returns a single signup flow by UUID. The service validates that the flow belongs to the tenant.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `signup_flow_uuid` | UUID | The UUID of the signup flow |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Signup flow retrieved successfully",
  "data": {
    "signup_flow_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "Default Signup",
    "description": "Standard user registration flow",
    "identifier": "default-signup",
    "config": {
      "require_email_verification": true,
      "allow_social_login": false
    },
    "status": "active",
    "client_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid signup flow UUID format |
| 401 | Missing or invalid JWT |
| 404 | Signup flow not found or does not belong to tenant |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/signup_flows/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/signup_flows

Creates a new signup flow for the tenant.

### Authentication

Bearer JWT required.

### Request Body

```json
{
  "name": "Default Signup",
  "description": "Standard user registration flow",
  "config": {
    "require_email_verification": true,
    "allow_social_login": false
  },
  "status": "active",
  "client_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901"
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | 1–100 characters |
| `description` | string | Yes | Non-empty description |
| `config` | object | No | Arbitrary JSON configuration for this flow |
| `status` | string | No | `active` or `inactive`. Default: `active`. |
| `client_id` | UUID | Yes | UUID of the auth client this flow is associated with |

### Response — 201 Created

```json
{
  "success": true,
  "message": "Signup flow created successfully",
  "data": {
    "signup_flow_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "Default Signup",
    "description": "Standard user registration flow",
    "identifier": "default-signup",
    "config": {
      "require_email_verification": true,
      "allow_social_login": false
    },
    "status": "active",
    "client_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
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
| 422 | Validation error (e.g., missing name, invalid `client_id`, invalid status) |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/signup_flows" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Default Signup",
    "description": "Standard user registration flow",
    "client_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
    "status": "active"
  }'
```

---

## PUT /api/v1/signup_flows/{signup_flow_uuid}

Updates an existing signup flow. The service validates tenant ownership.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `signup_flow_uuid` | UUID | The UUID of the signup flow to update |

### Request Body

```json
{
  "name": "Updated Signup",
  "description": "Updated registration flow",
  "config": {
    "require_email_verification": false
  },
  "status": "active"
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | 1–100 characters |
| `description` | string | Yes | Non-empty description |
| `config` | object | No | Replacement JSON configuration |
| `status` | string | No | `active` or `inactive`. Default: `active`. |

Note: `client_id` cannot be changed on update.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Signup flow updated successfully",
  "data": {
    "signup_flow_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "Updated Signup",
    "description": "Updated registration flow",
    "identifier": "updated-signup",
    "config": {
      "require_email_verification": false
    },
    "status": "active",
    "client_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
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
| 404 | Signup flow not found or does not belong to tenant |
| 422 | Validation error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/signup_flows/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "Updated Signup", "description": "Updated registration flow"}'
```

---

## PATCH /api/v1/signup_flows/{signup_flow_uuid}/status

Updates only the status of a signup flow.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `signup_flow_uuid` | UUID | The UUID of the signup flow |

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
  "message": "Signup flow status updated successfully",
  "data": {
    "signup_flow_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "Default Signup",
    "description": "Standard user registration flow",
    "identifier": "default-signup",
    "config": {},
    "status": "inactive",
    "client_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
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
| 404 | Signup flow not found or does not belong to tenant |
| 422 | Validation error (invalid status value) |

### Example

```bash
curl -X PATCH "http://localhost:8080/api/v1/signup_flows/a1b2c3d4-e5f6-7890-abcd-ef1234567890/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "inactive"}'
```

---

## DELETE /api/v1/signup_flows/{signup_flow_uuid}

Deletes a signup flow and its associated role assignments. The service validates tenant ownership.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `signup_flow_uuid` | UUID | The UUID of the signup flow to delete |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Signup flow deleted successfully",
  "data": {
    "signup_flow_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "Default Signup",
    "description": "Standard user registration flow",
    "identifier": "default-signup",
    "config": {},
    "status": "inactive",
    "client_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid signup flow UUID format |
| 401 | Missing or invalid JWT |
| 404 | Signup flow not found or does not belong to tenant |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/signup_flows/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/signup_flows/{signup_flow_uuid}/roles

Returns a paginated list of roles assigned to the signup flow. Users who complete registration through this flow are automatically granted these roles.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `signup_flow_uuid` | UUID | The UUID of the signup flow |

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page` | integer | Yes | Page number (minimum: 1) |
| `limit` | integer | Yes | Results per page (minimum: 1) |
| `sort_by` | string | No | Field to sort by |
| `sort_order` | string | No | `asc` or `desc` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Roles retrieved successfully",
  "data": {
    "rows": [
      {
        "role_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
        "name": "member",
        "description": "Default member role",
        "is_default": true,
        "is_system": false,
        "status": "active",
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

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid signup flow UUID format |
| 401 | Missing or invalid JWT |
| 404 | Signup flow not found or does not belong to tenant |
| 422 | Validation error on query parameters |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/signup_flows/a1b2c3d4-e5f6-7890-abcd-ef1234567890/roles?page=1&limit=10" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/signup_flows/{signup_flow_uuid}/roles

Assigns one or more roles to the signup flow. Users who register through this flow will automatically receive the assigned roles.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `signup_flow_uuid` | UUID | The UUID of the signup flow |

### Request Body

```json
{
  "role_uuids": [
    "c3d4e5f6-a7b8-9012-cdef-123456789012",
    "d4e5f6a7-b8c9-0123-def0-234567890123"
  ]
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `role_uuids` | array of UUID strings | Yes | One or more role UUIDs to assign |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Roles assigned successfully",
  "data": [
    {
      "role_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
      "name": "member",
      "description": "Default member role",
      "is_default": true,
      "is_system": false,
      "status": "active",
      "created_at": "2024-01-10T08:00:00Z",
      "updated_at": "2024-01-10T08:00:00Z"
    }
  ]
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID or malformed JSON |
| 401 | Missing or invalid JWT |
| 404 | Signup flow not found or does not belong to tenant |
| 422 | Validation error (e.g., invalid UUID in array, empty list) |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/signup_flows/a1b2c3d4-e5f6-7890-abcd-ef1234567890/roles" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"role_uuids": ["c3d4e5f6-a7b8-9012-cdef-123456789012"]}'
```

---

## DELETE /api/v1/signup_flows/{signup_flow_uuid}/roles/{role_uuid}

Removes a role from the signup flow. Future registrations through this flow will no longer receive this role. Existing users who already registered and received the role are not affected.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `signup_flow_uuid` | UUID | The UUID of the signup flow |
| `role_uuid` | UUID | The UUID of the role to remove |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Role removed successfully",
  "data": null
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 401 | Missing or invalid JWT |
| 404 | Signup flow or role assignment not found, or does not belong to tenant |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/signup_flows/a1b2c3d4-e5f6-7890-abcd-ef1234567890/roles/c3d4e5f6-a7b8-9012-cdef-123456789012" \
  -H "Authorization: Bearer <token>"
```
