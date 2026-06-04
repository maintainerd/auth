# Services

A **Service** represents an external application or system that integrates with the authentication platform. Services can be public (available to all tenants) or scoped to a specific tenant. Each service tracks how many APIs and policies are associated with it, and can be assigned policies for fine-grained access control.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/services` | List services with filters and pagination |
| GET | `/api/v1/services/{service_uuid}` | Get a single service by UUID |
| POST | `/api/v1/services` | Create a new service |
| PUT | `/api/v1/services/{service_uuid}` | Update a service |
| PUT | `/api/v1/services/{service_uuid}/status` | Update service status only |
| DELETE | `/api/v1/services/{service_uuid}` | Delete a service |
| POST | `/api/v1/services/{service_uuid}/policies/{policy_uuid}` | Assign a policy to a service |
| DELETE | `/api/v1/services/{service_uuid}/policies/{policy_uuid}` | Remove a policy from a service |
| GET | `/api/v1/services/me/policy-bundle` | Fetch the current service principal's IAM policy bundle |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

`GET /services/me/policy-bundle` requires a service-principal access token from
the OAuth `client_credentials` flow. See
[Service-to-Service Authorization](./authorization.md) for bundle caching,
`ETag` handling, and local authorization.

---

## GET /api/v1/services

Returns a paginated list of services for the authenticated tenant. Supports filtering by name, display name, description, version, status, and system flag.

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | integer | Page number (minimum: 1) |
| `limit` | integer | Results per page (minimum: 1) |
| `sort_by` | string | Field to sort by |
| `sort_order` | string | Sort direction: `asc` or `desc` |
| `name` | string | Filter by name (partial match) |
| `display_name` | string | Filter by display name (partial match) |
| `description` | string | Filter by description (partial match) |
| `version` | string | Filter by version |
| `status` | string | Comma-separated status values. Allowed: `active`, `inactive`, `maintenance`, `deprecated` |
| `is_system` | boolean | Filter by system flag (`true` or `false`) |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Services fetched successfully",
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

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/services?page=1&limit=10&status=active" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/services/{service_uuid}

Returns a single service by its UUID.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `service_uuid` | UUID | The UUID of the service |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Service fetched successfully",
  "data": {
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
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid service UUID format |
| 404 | Service not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/services/3fa85f64-5717-4562-b3fc-2c963f66afa6" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/services

Creates a new service for the authenticated tenant. The `is_system` flag is reserved for seeded services and cannot be set via this endpoint.

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique identifier name. 3–50 characters. |
| `display_name` | string | Yes | Human-readable name. 3–100 characters. |
| `description` | string | Yes | Service description. 8–255 characters. |
| `version` | string | Yes | Version string (e.g., `v1`, `2.0`). |
| `status` | string | Yes | Initial status. One of: `active`, `inactive`, `maintenance`, `deprecated`. |

### Response — 201 Created

```json
{
  "success": true,
  "message": "Service created successfully",
  "data": {
    "service_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "name": "billing-service",
    "display_name": "Billing Service",
    "description": "Manages subscriptions and payment processing",
    "version": "v1",
    "status": "active",
    "is_system": false,
    "api_count": 0,
    "policy_count": 0,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or validation failure |
| 409 | Service name already exists |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/services" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "billing-service",
    "display_name": "Billing Service",
    "description": "Manages subscriptions and payment processing",
    "version": "v1",
    "status": "active"
  }'
```

---

## PUT /api/v1/services/{service_uuid}

Updates all fields of an existing service. System services cannot have their `is_system` or `is_default` flags modified.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `service_uuid` | UUID | The UUID of the service to update |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Service name. 3–50 characters. |
| `display_name` | string | Yes | Human-readable name. 3–100 characters. |
| `description` | string | Yes | Service description. 8–255 characters. |
| `version` | string | Yes | Version string. |
| `status` | string | Yes | One of: `active`, `inactive`, `maintenance`, `deprecated`. |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Service updated successfully",
  "data": {
    "service_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "name": "billing-service",
    "display_name": "Billing Service",
    "description": "Manages subscriptions and payment processing",
    "version": "v2",
    "status": "active",
    "is_system": false,
    "api_count": 4,
    "policy_count": 2,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID or request body |
| 404 | Service not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/services/3fa85f64-5717-4562-b3fc-2c963f66afa6" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "billing-service",
    "display_name": "Billing Service",
    "description": "Manages subscriptions and payment processing",
    "version": "v2",
    "status": "active"
  }'
```

---

## PUT /api/v1/services/{service_uuid}/status

Updates the status of a service without modifying other fields.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `service_uuid` | UUID | The UUID of the service |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | New status. One of: `active`, `inactive`, `maintenance`, `deprecated`. |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Service updated successfully",
  "data": {
    "service_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "name": "billing-service",
    "display_name": "Billing Service",
    "description": "Manages subscriptions and payment processing",
    "version": "v1",
    "status": "maintenance",
    "is_system": false,
    "api_count": 4,
    "policy_count": 2,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID or invalid status value |
| 404 | Service not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/services/3fa85f64-5717-4562-b3fc-2c963f66afa6/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "maintenance"}'
```

---

## DELETE /api/v1/services/{service_uuid}

Permanently deletes a service. This also removes associated API relationships and policy assignments. System services cannot be deleted.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `service_uuid` | UUID | The UUID of the service to delete |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Service deleted successfully",
  "data": {
    "service_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "name": "billing-service",
    "display_name": "Billing Service",
    "description": "Manages subscriptions and payment processing",
    "version": "v1",
    "status": "inactive",
    "is_system": false,
    "api_count": 0,
    "policy_count": 0,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID |
| 403 | Cannot delete a system service |
| 404 | Service not found |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/services/3fa85f64-5717-4562-b3fc-2c963f66afa6" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/services/{service_uuid}/policies/{policy_uuid}

Associates a policy with a service for access control. Both the service and policy must belong to the authenticated tenant. Assignment emits `iam.service.policy.assigned` so service-policy bundle consumers can refresh cached authorization data.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `service_uuid` | UUID | The UUID of the service |
| `policy_uuid` | UUID | The UUID of the policy to assign |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Policy assigned to service successfully",
  "data": null
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid service or policy UUID |
| 404 | Service or policy not found |
| 409 | Policy already assigned to this service |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/services/3fa85f64-5717-4562-b3fc-2c963f66afa6/policies/7b1e3c91-8823-4d12-a9f0-3b7c52d1e4f2" \
  -H "Authorization: Bearer <token>"
```

---

## DELETE /api/v1/services/{service_uuid}/policies/{policy_uuid}

Removes the association between a policy and a service. The access control rules defined by the policy no longer apply to the service. Removal emits `iam.service.policy.removed` so service-policy bundle consumers can refresh cached authorization data.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `service_uuid` | UUID | The UUID of the service |
| `policy_uuid` | UUID | The UUID of the policy to remove |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Policy removed from service successfully",
  "data": null
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid service or policy UUID |
| 404 | Service, policy, or assignment not found |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/services/3fa85f64-5717-4562-b3fc-2c963f66afa6/policies/7b1e3c91-8823-4d12-a9f0-3b7c52d1e4f2" \
  -H "Authorization: Bearer <token>"
```
