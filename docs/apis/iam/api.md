# APIs

An **API** represents a specific interface exposed by a Service — for example, a REST endpoint, gRPC service, or WebSocket interface. APIs belong to a Service and are the unit to which Permissions are attached. Each API has a type, an identifier generated from its name, and a status.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/apis` | List APIs with filters and pagination |
| GET | `/api/v1/apis/{api_uuid}` | Get a single API by UUID |
| POST | `/api/v1/apis` | Create a new API |
| PUT | `/api/v1/apis/{api_uuid}` | Update an API |
| PUT | `/api/v1/apis/{api_uuid}/status` | Update API status only |
| DELETE | `/api/v1/apis/{api_uuid}` | Delete an API |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/apis

Returns a paginated list of APIs for the authenticated tenant.

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | integer | Page number (minimum: 1) |
| `limit` | integer | Results per page (minimum: 1) |
| `sort_by` | string | Field to sort by |
| `sort_order` | string | Sort direction: `asc` or `desc` |
| `name` | string | Filter by name (partial match) |
| `display_name` | string | Filter by display name (partial match) |
| `identifier` | string | Filter by identifier (partial match) |
| `api_type` | string | Filter by type. One of: `rest`, `grpc`, `graphql`, `soap`, `webhook`, `websocket`, `rpc` |
| `service_id` | UUID | Filter by parent service UUID |
| `status` | string | Comma-separated status values. Allowed: `active`, `inactive` |
| `is_system` | boolean | Filter by system flag (`true` or `false`) |

### Response — 200 OK

```json
{
  "success": true,
  "message": "APIs fetched successfully",
  "data": {
    "rows": [
      {
        "api_id": "9c1a2b3d-4e5f-6789-abcd-ef0123456789",
        "name": "user-login",
        "display_name": "User Login",
        "description": "Authenticates a user and returns an access token",
        "api_type": "rest",
        "identifier": "user-login",
        "service": {
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
        },
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
curl -X GET "http://localhost:8080/api/v1/apis?page=1&limit=10&service_id=3fa85f64-5717-4562-b3fc-2c963f66afa6" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/apis/{api_uuid}

Returns a single API by its UUID, including its parent service.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_uuid` | UUID | The UUID of the API |

### Response — 200 OK

```json
{
  "success": true,
  "message": "API fetched successfully",
  "data": {
    "api_id": "9c1a2b3d-4e5f-6789-abcd-ef0123456789",
    "name": "user-login",
    "display_name": "User Login",
    "description": "Authenticates a user and returns an access token",
    "api_type": "rest",
    "identifier": "user-login",
    "service": {
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
    },
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
| 400 | Invalid API UUID format |
| 404 | API not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/apis/9c1a2b3d-4e5f-6789-abcd-ef0123456789" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/apis

Creates a new API and associates it with the specified service. The `identifier` field is derived from the API name.

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | API name. 3–50 characters. |
| `display_name` | string | Yes | Human-readable name. 3–50 characters. |
| `description` | string | Yes | API description. 8–200 characters. |
| `api_type` | string | Yes | API protocol type. One of: `rest`, `grpc`, `graphql`, `soap`, `webhook`, `websocket`, `rpc`. |
| `service_id` | UUID | Yes | UUID of the parent service. |
| `status` | string | Yes | Initial status. One of: `active`, `inactive`. |

### Response — 201 Created

```json
{
  "success": true,
  "message": "API created successfully",
  "data": {
    "api_id": "9c1a2b3d-4e5f-6789-abcd-ef0123456789",
    "name": "user-login",
    "display_name": "User Login",
    "description": "Authenticates a user and returns an access token",
    "api_type": "rest",
    "identifier": "user-login",
    "service": {
      "service_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
      "name": "auth-service",
      "display_name": "Authentication Service",
      "description": "Handles user authentication and token issuance",
      "version": "v1",
      "status": "active",
      "is_system": false,
      "api_count": 5,
      "policy_count": 2,
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z"
    },
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
| 400 | Invalid request body or validation failure |
| 404 | Referenced service not found |
| 409 | API name already exists |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/apis" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user-login",
    "display_name": "User Login",
    "description": "Authenticates a user and returns an access token",
    "api_type": "rest",
    "service_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "status": "active"
  }'
```

---

## PUT /api/v1/apis/{api_uuid}

Updates all fields of an existing API.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_uuid` | UUID | The UUID of the API to update |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | API name. 3–50 characters. |
| `display_name` | string | Yes | Human-readable name. 3–50 characters. |
| `description` | string | Yes | API description. 8–200 characters. |
| `api_type` | string | Yes | API protocol type. One of: `rest`, `grpc`, `graphql`, `soap`, `webhook`, `websocket`, `rpc`. |
| `service_id` | UUID | Yes | UUID of the parent service. |
| `status` | string | Yes | One of: `active`, `inactive`. |

### Response — 200 OK

```json
{
  "success": true,
  "message": "API updated successfully",
  "data": {
    "api_id": "9c1a2b3d-4e5f-6789-abcd-ef0123456789",
    "name": "user-login",
    "display_name": "User Login",
    "description": "Authenticates a user and returns an access token",
    "api_type": "rest",
    "identifier": "user-login",
    "service": {
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
    },
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
| 400 | Invalid UUID or request body |
| 404 | API or service not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/apis/9c1a2b3d-4e5f-6789-abcd-ef0123456789" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user-login",
    "display_name": "User Login",
    "description": "Authenticates a user and returns a signed access token",
    "api_type": "rest",
    "service_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "status": "active"
  }'
```

---

## PUT /api/v1/apis/{api_uuid}/status

Updates the status of an API without modifying other fields.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_uuid` | UUID | The UUID of the API |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | New status. One of: `active`, `inactive`. |

### Response — 200 OK

```json
{
  "success": true,
  "message": "API status updated successfully",
  "data": {
    "api_id": "9c1a2b3d-4e5f-6789-abcd-ef0123456789",
    "name": "user-login",
    "display_name": "User Login",
    "description": "Authenticates a user and returns an access token",
    "api_type": "rest",
    "identifier": "user-login",
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
| 404 | API not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/apis/9c1a2b3d-4e5f-6789-abcd-ef0123456789/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "inactive"}'
```

---

## DELETE /api/v1/apis/{api_uuid}

Permanently deletes an API and removes its associated permissions. System APIs cannot be deleted.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_uuid` | UUID | The UUID of the API to delete |

### Response — 200 OK

```json
{
  "success": true,
  "message": "API deleted successfully",
  "data": {
    "api_id": "9c1a2b3d-4e5f-6789-abcd-ef0123456789",
    "name": "user-login",
    "display_name": "User Login",
    "description": "Authenticates a user and returns an access token",
    "api_type": "rest",
    "identifier": "user-login",
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
| 403 | Cannot delete a system API |
| 404 | API not found |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/apis/9c1a2b3d-4e5f-6789-abcd-ef0123456789" \
  -H "Authorization: Bearer <token>"
```
