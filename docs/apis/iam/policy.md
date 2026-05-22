# Policies

A **Policy** is a JSON document that defines access control rules through a set of statements. Each statement specifies an effect (`allow` or `deny`), a list of actions, and a list of resources. Policies are assigned to Services to grant or restrict access.

## Policy Document Format

A policy document must follow this structure:

```json
{
  "version": "v1",
  "statement": [
    {
      "effect": "allow",
      "action": ["user:create", "user:read"],
      "resource": ["auth:*"]
    },
    {
      "effect": "deny",
      "action": ["user:delete"],
      "resource": ["auth:user-management"]
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `version` | string | Document schema version. Currently `v1`. |
| `statement` | array | One or more statement objects. At least one is required. |
| `statement[].effect` | string | `allow` or `deny`. |
| `statement[].action` | array | Permission identifiers in `resource:action` format. Use `resource:*` for all actions on a resource. |
| `statement[].resource` | array | Service and API identifiers in `service:api` format. Use `service:*` for all APIs under a service. |

Action and resource values are not validated against existing permissions or services at write time. Invalid values result in no access being granted.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/policies` | List policies with filters and pagination |
| GET | `/api/v1/policies/{policy_uuid}` | Get a single policy by UUID (includes document) |
| POST | `/api/v1/policies` | Create a new policy |
| PUT | `/api/v1/policies/{policy_uuid}` | Update a policy |
| PUT | `/api/v1/policies/{policy_uuid}/status` | Update policy status only |
| DELETE | `/api/v1/policies/{policy_uuid}` | Delete a policy |
| GET | `/api/v1/policies/{policy_uuid}/services` | List services that use this policy |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/policies

Returns a paginated list of policies. The list response omits the `document` field for efficiency; use `GET /policies/{policy_uuid}` to retrieve the full document.

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | integer | Page number (minimum: 1, default: 1) |
| `limit` | integer | Results per page (1–100, default: 10) |
| `sort_by` | string | Field to sort by |
| `sort_order` | string | Sort direction: `asc` or `desc` |
| `name` | string | Filter by name (partial match) |
| `description` | string | Filter by description (partial match) |
| `version` | string | Filter by version |
| `status` | string | Comma-separated status values. Allowed: `active`, `inactive` |
| `is_system` | boolean | Filter by system flag (`true` or `false`) |
| `service_id` | UUID | Filter policies assigned to this service UUID |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Policies retrieved successfully",
  "data": {
    "rows": [
      {
        "policy_id": "7b1e3c91-8823-4d12-a9f0-3b7c52d1e4f2",
        "name": "admin:full-access",
        "description": "Grants full access to all administrative resources",
        "version": "v1",
        "status": "active",
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
curl -X GET "http://localhost:8080/api/v1/policies?page=1&limit=10&status=active" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/policies/{policy_uuid}

Returns a single policy including its full `document` field.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `policy_uuid` | UUID | The UUID of the policy |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Policy retrieved successfully",
  "data": {
    "policy_id": "7b1e3c91-8823-4d12-a9f0-3b7c52d1e4f2",
    "name": "admin:full-access",
    "description": "Grants full access to all administrative resources",
    "document": {
      "version": "v1",
      "statement": [
        {
          "effect": "allow",
          "action": ["user:*", "role:*"],
          "resource": ["auth:*"]
        }
      ]
    },
    "version": "v1",
    "status": "active",
    "is_system": false,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid policy UUID format |
| 404 | Policy not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/policies/7b1e3c91-8823-4d12-a9f0-3b7c52d1e4f2" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/policies

Creates a new policy. The `name` field must use only lowercase letters, numbers, underscores, colons, slashes, and hyphens.

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Policy name. 3–150 characters. Allowed characters: `a-z`, `0-9`, `_`, `:`, `/`, `\`, `-`. |
| `description` | string | No | Optional description. Up to 500 characters. |
| `document` | object | Yes | Policy document. Must contain `version` and at least one `statement`. |
| `version` | string | Yes | Version label. 1–20 characters (e.g., `v1`). |
| `status` | string | Yes | Initial status. One of: `active`, `inactive`. |

### Response — 201 Created

```json
{
  "success": true,
  "message": "Policy created successfully",
  "data": {
    "policy_id": "7b1e3c91-8823-4d12-a9f0-3b7c52d1e4f2",
    "name": "admin:full-access",
    "description": "Grants full access to all administrative resources",
    "document": {
      "version": "v1",
      "statement": [
        {
          "effect": "allow",
          "action": ["user:*", "role:*"],
          "resource": ["auth:*"]
        }
      ]
    },
    "version": "v1",
    "status": "active",
    "is_system": false,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body, invalid document structure, or name contains invalid characters |
| 409 | Policy name already exists |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/policies" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "admin:full-access",
    "description": "Grants full access to all administrative resources",
    "document": {
      "version": "v1",
      "statement": [
        {
          "effect": "allow",
          "action": ["user:*", "role:*"],
          "resource": ["auth:*"]
        }
      ]
    },
    "version": "v1",
    "status": "active"
  }'
```

---

## PUT /api/v1/policies/{policy_uuid}

Updates all fields of an existing policy, including replacing the document.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `policy_uuid` | UUID | The UUID of the policy to update |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Policy name. 3–150 characters. |
| `description` | string | No | Optional description. Up to 500 characters. |
| `document` | object | Yes | Replacement policy document. |
| `version` | string | Yes | Version label. 1–20 characters. |
| `status` | string | Yes | One of: `active`, `inactive`. |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Policy updated successfully",
  "data": {
    "policy_id": "7b1e3c91-8823-4d12-a9f0-3b7c52d1e4f2",
    "name": "admin:full-access",
    "description": "Grants full access to all administrative resources",
    "document": {
      "version": "v1",
      "statement": [
        {
          "effect": "allow",
          "action": ["user:*", "role:*", "client:*"],
          "resource": ["auth:*"]
        }
      ]
    },
    "version": "v2",
    "status": "active",
    "is_system": false,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID, invalid document structure, or validation failure |
| 404 | Policy not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/policies/7b1e3c91-8823-4d12-a9f0-3b7c52d1e4f2" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "admin:full-access",
    "description": "Grants full access to all administrative resources",
    "document": {
      "version": "v1",
      "statement": [
        {
          "effect": "allow",
          "action": ["user:*", "role:*", "client:*"],
          "resource": ["auth:*"]
        }
      ]
    },
    "version": "v2",
    "status": "active"
  }'
```

---

## PUT /api/v1/policies/{policy_uuid}/status

Updates the status of a policy without modifying other fields.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `policy_uuid` | UUID | The UUID of the policy |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | New status. One of: `active`, `inactive`. |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Policy status updated successfully",
  "data": {
    "policy_id": "7b1e3c91-8823-4d12-a9f0-3b7c52d1e4f2",
    "name": "admin:full-access",
    "description": "Grants full access to all administrative resources",
    "document": {
      "version": "v1",
      "statement": [
        {
          "effect": "allow",
          "action": ["user:*"],
          "resource": ["auth:*"]
        }
      ]
    },
    "version": "v1",
    "status": "inactive",
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
| 404 | Policy not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/policies/7b1e3c91-8823-4d12-a9f0-3b7c52d1e4f2/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "inactive"}'
```

---

## DELETE /api/v1/policies/{policy_uuid}

Permanently deletes a policy. This also removes the policy from any services it is assigned to. System policies cannot be deleted.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `policy_uuid` | UUID | The UUID of the policy to delete |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Policy deleted successfully",
  "data": {
    "policy_id": "7b1e3c91-8823-4d12-a9f0-3b7c52d1e4f2",
    "name": "admin:full-access",
    "description": "Grants full access to all administrative resources",
    "document": {
      "version": "v1",
      "statement": [
        {
          "effect": "allow",
          "action": ["user:*"],
          "resource": ["auth:*"]
        }
      ]
    },
    "version": "v1",
    "status": "inactive",
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
| 403 | Cannot delete a system policy |
| 404 | Policy not found |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/policies/7b1e3c91-8823-4d12-a9f0-3b7c52d1e4f2" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/policies/{policy_uuid}/services

Returns a paginated list of services that have this policy assigned.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `policy_uuid` | UUID | The UUID of the policy |

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | integer | Page number (minimum: 1, default: 1) |
| `limit` | integer | Results per page (minimum: 1, default: 10) |
| `sort_by` | string | Field to sort by |
| `sort_order` | string | Sort direction: `asc` or `desc` |
| `name` | string | Filter by service name (partial match, up to 150 characters) |
| `display_name` | string | Filter by display name (partial match, up to 150 characters) |
| `description` | string | Filter by description (partial match, up to 500 characters) |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Services retrieved successfully",
  "data": {
    "rows": [
      {
        "service_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
        "name": "auth-service",
        "display_name": "Authentication Service",
        "description": "Handles user authentication and token issuance",
        "version": "v1",
        "status": "active",
        "is_system": false,
        "api_count": 4,
        "policy_count": 2,
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
| 400 | Invalid policy UUID |
| 404 | Policy not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/policies/7b1e3c91-8823-4d12-a9f0-3b7c52d1e4f2/services?page=1&limit=10" \
  -H "Authorization: Bearer <token>"
```
