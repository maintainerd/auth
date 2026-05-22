# Tenant API

The Tenant API manages the tenants (organizations) within a maintainerd-auth installation. Each tenant is an isolated namespace with its own users, roles, identity providers, and settings.

> **Port 8080 only.** All endpoints in this document are available on the internal management port. They require a valid Bearer JWT and the appropriate permission scope.

---

## Endpoints

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| GET | `/api/v1/tenants` | `tenant:read` | List tenants with filtering and pagination |
| GET | `/api/v1/tenants/{tenant_uuid}` | `tenant:read` | Get a single tenant by UUID |
| POST | `/api/v1/tenants` | `tenant:create` | Create a new tenant |
| PUT | `/api/v1/tenants/{tenant_uuid}` | `tenant:update` | Update a tenant |
| PUT | `/api/v1/tenants/{tenant_uuid}/status` | `tenant:update` | Set a tenant's status |
| PUT | `/api/v1/tenants/{tenant_uuid}/public` | `tenant:update` | Promote a tenant to the active public tenant |
| DELETE | `/api/v1/tenants/{tenant_uuid}` | `tenant:delete` | Delete a tenant |

---

## Tenant Object

All tenant endpoints return objects in this shape:

```json
{
  "tenant_id": "018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c",
  "name": "acme",
  "display_name": "Acme Corp",
  "description": "Main tenant for Acme Corp",
  "identifier": "acme",
  "status": "active",
  "is_public": false,
  "is_system": false,
  "metadata": {},
  "created_at": "2026-01-15T10:00:00Z",
  "updated_at": "2026-01-15T10:00:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `tenant_id` | UUID | Unique identifier for the tenant |
| `name` | string | URL-safe slug (lowercase letters, numbers, hyphens) |
| `display_name` | string | Human-readable name |
| `description` | string | Optional description |
| `identifier` | string | System-generated identifier derived from `name` |
| `status` | string | `active`, `inactive`, `pending`, or `suspended` |
| `is_public` | boolean | Whether this is the currently active public tenant |
| `is_system` | boolean | Whether this is the protected system tenant |
| `metadata` | object | Arbitrary JSON metadata |
| `created_at` | string | ISO 8601 timestamp |
| `updated_at` | string | ISO 8601 timestamp |

---

## GET /api/v1/tenants

Returns a paginated list of tenants. Supports filtering by multiple criteria.

### Authentication

Bearer JWT with `tenant:read` permission.

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page` | integer | Yes | Page number (minimum: 1) |
| `limit` | integer | Yes | Results per page (minimum: 1) |
| `sort_by` | string | No | Field to sort by (max 50 characters) |
| `sort_order` | string | No | `asc` or `desc` |
| `name` | string | No | Filter by name (partial match) |
| `display_name` | string | No | Filter by display name |
| `description` | string | No | Filter by description |
| `identifier` | string | No | Filter by identifier |
| `status` | string | No | Comma-separated status values: `active`, `inactive`, `pending`, `suspended` |
| `is_public` | boolean | No | Filter by public flag |
| `is_system` | boolean | No | Filter by system flag |

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Tenants fetched successfully",
  "data": {
    "rows": [
      {
        "tenant_id": "018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c",
        "name": "acme",
        "display_name": "Acme Corp",
        "description": "Main tenant for Acme Corp",
        "identifier": "acme",
        "status": "active",
        "is_public": false,
        "is_system": false,
        "metadata": {},
        "created_at": "2026-01-15T10:00:00Z",
        "updated_at": "2026-01-15T10:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 20,
    "total_pages": 1
  }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid query parameters |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/tenants?page=1&limit=20&status=active" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/tenants/{tenant_uuid}

Returns a single tenant by its UUID.

### Authentication

Bearer JWT with `tenant:read` permission.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `tenant_uuid` | UUID | The tenant's unique identifier |

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Tenant fetched successfully",
  "data": {
    "tenant_id": "018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c",
    "name": "acme",
    "display_name": "Acme Corp",
    "description": "Main tenant for Acme Corp",
    "identifier": "acme",
    "status": "active",
    "is_public": false,
    "is_system": false,
    "metadata": {},
    "created_at": "2026-01-15T10:00:00Z",
    "updated_at": "2026-01-15T10:00:00Z"
  }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions |
| 404 | Tenant not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/tenants/018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/tenants

Creates a new tenant.

### Authentication

Bearer JWT with `tenant:create` permission.

### Request Body

```json
{
  "name": "acme",
  "display_name": "Acme Corp",
  "description": "Main tenant for Acme Corp",
  "status": "active",
  "is_public": false
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `name` | string | Yes | 3–50 characters; lowercase letters, numbers, hyphens only |
| `display_name` | string | No | No explicit validation; used for display only |
| `description` | string | Yes | 8–200 characters |
| `status` | string | Yes | `active`, `inactive`, `pending`, or `suspended` |
| `is_public` | boolean | Yes | `true` or `false` |

### Response

#### 201 Created

```json
{
  "success": true,
  "message": "Tenant created successfully",
  "data": {
    "tenant_id": "018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c",
    "name": "acme",
    "display_name": "Acme Corp",
    "description": "Main tenant for Acme Corp",
    "identifier": "acme",
    "status": "active",
    "is_public": false,
    "is_system": false,
    "metadata": {},
    "created_at": "2026-01-15T10:00:00Z",
    "updated_at": "2026-01-15T10:00:00Z"
  }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or validation failure |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions |
| 409 | A tenant with that name already exists |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/tenants" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "acme",
    "display_name": "Acme Corp",
    "description": "Main tenant for Acme Corp",
    "status": "active",
    "is_public": false
  }'
```

---

## PUT /api/v1/tenants/{tenant_uuid}

Updates a tenant's name, display name, description, status, and public flag. The authenticated user must be a member of the tenant being updated.

### Authentication

Bearer JWT with `tenant:update` permission. The authenticated user must be a member of the target tenant.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `tenant_uuid` | UUID | The tenant's unique identifier |

### Request Body

```json
{
  "name": "acme-updated",
  "display_name": "Acme Corp (Updated)",
  "description": "Updated description for Acme Corp",
  "status": "active",
  "is_public": false
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `name` | string | Yes | 3–50 characters; lowercase letters, numbers, hyphens only |
| `display_name` | string | No | No explicit validation |
| `description` | string | Yes | 8–200 characters |
| `status` | string | Yes | `active`, `inactive`, `pending`, or `suspended` |
| `is_public` | boolean | Yes | `true` or `false` |

### Response

#### 200 OK

Returns the updated tenant object (same shape as the Tenant Object above).

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID or request body |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions or not a tenant member |
| 404 | Tenant not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/tenants/018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "acme",
    "display_name": "Acme Corp",
    "description": "Updated description",
    "status": "active",
    "is_public": false
  }'
```

---

## PUT /api/v1/tenants/{tenant_uuid}/status

Updates only the status of a tenant.

### Authentication

Bearer JWT with `tenant:update` permission.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `tenant_uuid` | UUID | The tenant's unique identifier |

### Request Body

```json
{
  "status": "inactive"
}
```

| Field | Type | Required | Allowed Values |
|-------|------|----------|----------------|
| `status` | string | Yes | `active`, `inactive`, `pending`, `suspended` |

### Response

#### 200 OK

Returns the updated tenant object.

```json
{
  "success": true,
  "message": "Tenant status updated successfully",
  "data": { ... }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID or request body |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions |
| 404 | Tenant not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/tenants/018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "inactive"}'
```

---

## PUT /api/v1/tenants/{tenant_uuid}/public

Sets the specified tenant as the active public tenant. Only one tenant can be public at a time; this operation clears the `is_public` flag on any previously public tenant and sets it on the target.

### Authentication

Bearer JWT with `tenant:update` permission.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `tenant_uuid` | UUID | The tenant to set as public |

### Request Body

No body required.

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Tenant public updated successfully",
  "data": { ... }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions |
| 404 | Tenant not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/tenants/018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c/public" \
  -H "Authorization: Bearer <token>"
```

---

## DELETE /api/v1/tenants/{tenant_uuid}

Deletes a tenant. The authenticated user must be a member of the tenant. System tenants (`is_system: true`) cannot be deleted.

### Authentication

Bearer JWT with `tenant:delete` permission. The authenticated user must be a member of the target tenant.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `tenant_uuid` | UUID | The tenant's unique identifier |

### Response

#### 200 OK

Returns the deleted tenant object.

```json
{
  "success": true,
  "message": "Tenant deleted successfully",
  "data": { ... }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions, not a tenant member, or attempting to delete a system tenant |
| 404 | Tenant not found |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/tenants/018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c" \
  -H "Authorization: Bearer <token>"
```
