# Clients

A **Client** represents an OAuth2/OIDC application registered with the authentication platform. Clients are the entities that request tokens on behalf of users or themselves. Each client has a type that determines its OAuth flow, a domain, a JSON configuration block, and an associated identity provider.

## Client Types

| Type | Description |
|------|-------------|
| `traditional` | Server-side web applications using authorization code flow |
| `spa` | Single-page applications (public clients, no secret) |
| `mobile` | Native mobile applications (public clients) |
| `m2m` | Machine-to-machine clients using client credentials flow |

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/clients` | List clients with filters and pagination |
| GET | `/api/v1/clients/{client_uuid}` | Get a single client by UUID |
| POST | `/api/v1/clients` | Create a new client |
| PUT | `/api/v1/clients/{client_uuid}` | Update a client |
| PUT | `/api/v1/clients/{client_uuid}/status` | Toggle client status |
| DELETE | `/api/v1/clients/{client_uuid}` | Delete a client |
| GET | `/api/v1/clients/{client_uuid}/secret` | Get client credentials (ID and secret) |
| GET | `/api/v1/clients/{client_uuid}/config` | Get client OAuth configuration |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/clients

Returns a paginated list of clients for the authenticated tenant.

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | integer | Page number (minimum: 1) |
| `limit` | integer | Results per page (minimum: 1) |
| `sort_by` | string | Field to sort by |
| `sort_order` | string | Sort direction: `asc` or `desc` |
| `name` | string | Filter by name (partial match) |
| `display_name` | string | Filter by display name (partial match) |
| `client_type` | string | Comma-separated client types. One or more of: `traditional`, `spa`, `mobile`, `m2m` |
| `identity_provider_id` | UUID | Filter by identity provider UUID |
| `status` | string | Comma-separated status values. Allowed: `active`, `inactive` |
| `is_default` | boolean | Filter by default flag (`true` or `false`) |
| `is_system` | boolean | Filter by system flag (`true` or `false`) |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Auth clients fetched successfully",
  "data": {
    "rows": [
      {
        "client_id": "d4e5f6a7-b8c9-0123-def0-123456789abc",
        "name": "my-web-app",
        "display_name": "My Web Application",
        "client_type": "traditional",
        "domain": "app.example.com",
        "uris": [
          {
            "uri_id": "e1f2a3b4-c5d6-7890-abcd-ef0123456789",
            "uri": "https://app.example.com/callback",
            "type": "redirect-uri",
            "created_at": "2024-01-15T10:00:00Z",
            "updated_at": "2024-01-15T10:00:00Z"
          }
        ],
        "identity_provider": {
          "identity_provider_id": "f1a2b3c4-d5e6-7890-abcd-ef0123456789",
          "name": "default-idp",
          "display_name": "Default Identity Provider",
          "provider": "internal",
          "provider_type": "identity",
          "identifier": "default-idp",
          "status": "active",
          "is_default": true,
          "is_system": true,
          "created_at": "2024-01-01T00:00:00Z",
          "updated_at": "2024-01-01T00:00:00Z"
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
curl -X GET "http://localhost:8080/api/v1/clients?page=1&limit=10&client_type=traditional,spa" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/clients/{client_uuid}

Returns a single client by its UUID, including associated URIs, identity provider, and permissions.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Auth client fetched successfully",
  "data": {
    "client_id": "d4e5f6a7-b8c9-0123-def0-123456789abc",
    "name": "my-web-app",
    "display_name": "My Web Application",
    "client_type": "traditional",
    "domain": "app.example.com",
    "uris": [],
    "identity_provider": {
      "identity_provider_id": "f1a2b3c4-d5e6-7890-abcd-ef0123456789",
      "name": "default-idp",
      "display_name": "Default Identity Provider",
      "provider": "internal",
      "provider_type": "identity",
      "identifier": "default-idp",
      "status": "active",
      "is_default": true,
      "is_system": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
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
| 400 | Invalid client UUID format |
| 404 | Client not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/clients

Creates a new client. A `config` JSON object is required and holds OAuth2/OIDC configuration specific to the client type. The `identity_provider_id` links the client to an existing identity provider.

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Client name. 3–50 characters. |
| `display_name` | string | Yes | Human-readable name. 8–200 characters. |
| `client_type` | string | Yes | One of: `traditional`, `spa`, `mobile`, `m2m`. |
| `domain` | string | Yes | Application domain. 3–100 characters. |
| `config` | object | Yes | OAuth2 configuration JSON for the client. |
| `status` | string | Yes | Initial status. One of: `active`, `inactive`. |
| `identity_provider_id` | UUID | Yes | UUID of the identity provider to associate. |

### Response — 201 Created

```json
{
  "success": true,
  "message": "Auth client created successfully",
  "data": {
    "client_id": "d4e5f6a7-b8c9-0123-def0-123456789abc",
    "name": "my-web-app",
    "display_name": "My Web Application",
    "client_type": "traditional",
    "domain": "app.example.com",
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
| 404 | Referenced identity provider not found |
| 409 | Client name already exists |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/clients" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-web-app",
    "display_name": "My Web Application",
    "client_type": "traditional",
    "domain": "app.example.com",
    "config": {
      "token_endpoint_auth_method": "client_secret_basic",
      "grant_types": ["authorization_code", "refresh_token"],
      "response_types": ["code"]
    },
    "status": "active",
    "identity_provider_id": "f1a2b3c4-d5e6-7890-abcd-ef0123456789"
  }'
```

---

## PUT /api/v1/clients/{client_uuid}

Updates all fields of an existing client. The identity provider cannot be changed after creation.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client to update |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Client name. 3–50 characters. |
| `display_name` | string | Yes | Human-readable name. 8–200 characters. |
| `client_type` | string | Yes | One of: `traditional`, `spa`, `mobile`, `m2m`. |
| `domain` | string | Yes | Application domain. 3–100 characters. |
| `config` | object | Yes | OAuth2 configuration JSON. |
| `status` | string | Yes | One of: `active`, `inactive`. |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Auth client updated successfully",
  "data": {
    "client_id": "d4e5f6a7-b8c9-0123-def0-123456789abc",
    "name": "my-web-app",
    "display_name": "My Web Application (Updated)",
    "client_type": "traditional",
    "domain": "app.example.com",
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
| 404 | Client not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-web-app",
    "display_name": "My Web Application (Updated)",
    "client_type": "traditional",
    "domain": "app.example.com",
    "config": {
      "token_endpoint_auth_method": "client_secret_basic",
      "grant_types": ["authorization_code", "refresh_token"],
      "response_types": ["code"]
    },
    "status": "active"
  }'
```

---

## PUT /api/v1/clients/{client_uuid}/status

Toggles the client status between `active` and `inactive`. No request body is required — the current status is read and toggled automatically.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Auth client status updated successfully",
  "data": {
    "client_id": "d4e5f6a7-b8c9-0123-def0-123456789abc",
    "name": "my-web-app",
    "display_name": "My Web Application",
    "client_type": "traditional",
    "domain": "app.example.com",
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
| 404 | Client not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc/status" \
  -H "Authorization: Bearer <token>"
```

---

## DELETE /api/v1/clients/{client_uuid}

Permanently deletes a client and all associated URIs, API assignments, and permissions. System clients cannot be deleted.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client to delete |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Auth client deleted successfully",
  "data": {
    "client_id": "d4e5f6a7-b8c9-0123-def0-123456789abc",
    "name": "my-web-app",
    "display_name": "My Web Application",
    "client_type": "traditional",
    "domain": "app.example.com",
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
| 403 | Cannot delete a system client |
| 404 | Client not found |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc" \
  -H "Authorization: Bearer <token>"
```
