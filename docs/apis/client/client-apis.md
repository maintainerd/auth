# Client APIs and Permissions

Clients are granted access to resources by associating APIs (and specific permissions within those APIs) with them. This two-level model allows precise control over what a client can do: first assign an API, then optionally restrict which permissions within that API the client may exercise.

## How it works

1. **Assign an API** to the client — this gives the client the ability to request tokens scoped to that API.
2. **Assign permissions** within each API — this restricts which specific permissions the client can claim. If no permissions are explicitly assigned, the client inherits all active permissions from the API.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/clients/{client_uuid}/apis` | List APIs assigned to a client |
| POST | `/api/v1/clients/{client_uuid}/apis` | Assign one or more APIs to a client |
| DELETE | `/api/v1/clients/{client_uuid}/apis/{api_uuid}` | Remove an API from a client |
| GET | `/api/v1/clients/{client_uuid}/apis/{api_uuid}/permissions` | List permissions for a specific client API |
| POST | `/api/v1/clients/{client_uuid}/apis/{api_uuid}/permissions` | Assign permissions to a client API |
| DELETE | `/api/v1/clients/{client_uuid}/apis/{api_uuid}/permissions/{permission_uuid}` | Remove a permission from a client API |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/clients/{client_uuid}/apis

Returns all APIs assigned to the client, with their associated permissions.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Auth client APIs retrieved successfully",
  "data": {
    "apis": [
      {
        "client_api_id": "b1c2d3e4-f5a6-7890-bcde-f01234567890",
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
        "permissions": [
          {
            "permission_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
            "name": "user:read",
            "description": "Allows reading user records",
            "status": "active",
            "is_default": false,
            "is_system": false,
            "created_at": "2024-01-15T10:00:00Z",
            "updated_at": "2024-01-15T10:00:00Z"
          }
        ],
        "created_at": "2024-01-15T10:00:00Z"
      }
    ]
  }
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `apis` | array | List of API assignments. |
| `apis[].client_api_id` | UUID | Unique identifier of the client-API assignment. |
| `apis[].api` | object | The assigned API object. |
| `apis[].permissions` | array | Permissions granted for this API on this client. Empty if none are explicitly set. |
| `apis[].created_at` | timestamp | When the API was assigned to the client. |

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid client UUID format |
| 404 | Client not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc/apis" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/clients/{client_uuid}/apis

Assigns one or more APIs to a client in a single request. APIs already assigned to the client are ignored (no error is returned for duplicates).

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `api_uuids` | array of UUIDs | Yes | One or more API UUIDs to assign to the client. |

### Response — 200 OK

```json
{
  "success": true,
  "message": "APIs added to auth client successfully",
  "data": {
    "message": "APIs added to auth client successfully"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format or empty `api_uuids` array |
| 404 | Client or one of the referenced APIs not found |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc/apis" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "api_uuids": [
      "9c1a2b3d-4e5f-6789-abcd-ef0123456789",
      "7d8e9f0a-1b2c-3456-789a-bcdef0123456"
    ]
  }'
```

---

## DELETE /api/v1/clients/{client_uuid}/apis/{api_uuid}

Removes an API from a client. All permissions associated with this API assignment are also removed.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client |
| `api_uuid` | UUID | The UUID of the API to remove |

### Response — 200 OK

```json
{
  "success": true,
  "message": "API removed from auth client successfully",
  "data": {
    "message": "API removed from auth client successfully"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 404 | Client, API, or assignment not found |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc/apis/9c1a2b3d-4e5f-6789-abcd-ef0123456789" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/clients/{client_uuid}/apis/{api_uuid}/permissions

Returns the permissions explicitly granted to the client for the specified API.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client |
| `api_uuid` | UUID | The UUID of the API |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Auth client API permissions retrieved successfully",
  "data": {
    "permissions": [
      {
        "permission_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "name": "user:read",
        "description": "Allows reading user records",
        "status": "active",
        "is_default": false,
        "is_system": false,
        "created_at": "2024-01-15T10:00:00Z",
        "updated_at": "2024-01-15T10:00:00Z"
      }
    ]
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 404 | Client, API, or assignment not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc/apis/9c1a2b3d-4e5f-6789-abcd-ef0123456789/permissions" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/clients/{client_uuid}/apis/{api_uuid}/permissions

Grants one or more permissions to a client for a specific API. The API must already be assigned to the client. Permissions already granted are ignored.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client |
| `api_uuid` | UUID | The UUID of the API |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `permission_uuids` | array of UUIDs | Yes | One or more permission UUIDs to grant. All permissions must belong to the specified API. |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Permissions added to auth client API successfully",
  "data": {
    "message": "Permissions added to auth client API successfully"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format or empty `permission_uuids` array |
| 404 | Client, API, assignment, or one of the referenced permissions not found |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc/apis/9c1a2b3d-4e5f-6789-abcd-ef0123456789/permissions" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "permission_uuids": [
      "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "b2c3d4e5-f6a7-8901-bcde-f01234567890"
    ]
  }'
```

---

## DELETE /api/v1/clients/{client_uuid}/apis/{api_uuid}/permissions/{permission_uuid}

Removes a single permission from a client's API assignment. The permission is no longer claimable by this client for this API.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client |
| `api_uuid` | UUID | The UUID of the API |
| `permission_uuid` | UUID | The UUID of the permission to remove |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Permission removed from auth client API successfully",
  "data": {
    "message": "Permission removed from auth client API successfully"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 404 | Client, API, permission, or assignment not found |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc/apis/9c1a2b3d-4e5f-6789-abcd-ef0123456789/permissions/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>"
```
