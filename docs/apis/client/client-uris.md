# Client URIs

**Client URIs** control the allowed redirect, logout, login, origin, and CORS origin URLs for a client. OAuth2 and OIDC flows validate redirect destinations against these registered URIs — requests with unregistered URIs are rejected.

## URI Types

| Type | Description |
|------|-------------|
| `redirect-uri` | Allowed callback URLs for the authorization code flow |
| `origin-uri` | Allowed application origins |
| `logout-uri` | Allowed post-logout redirect URLs |
| `login-uri` | Allowed login initiation URLs |
| `cors-origin-uri` | Allowed origins for CORS requests from browser clients |

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/clients/{client_uuid}/uris` | List all URIs for a client |
| POST | `/api/v1/clients/{client_uuid}/uris` | Add a URI to a client |
| PUT | `/api/v1/clients/{client_uuid}/uris/{client_uri_uuid}` | Update a URI |
| DELETE | `/api/v1/clients/{client_uuid}/uris/{client_uri_uuid}` | Remove a URI from a client |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/clients/{client_uuid}/uris

Returns all URIs registered for the specified client.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client |

### Response — 200 OK

```json
{
  "success": true,
  "message": "URIs retrieved successfully",
  "data": {
    "uris": [
      {
        "uri_id": "e1f2a3b4-c5d6-7890-abcd-ef0123456789",
        "uri": "https://app.example.com/callback",
        "type": "redirect-uri",
        "created_at": "2024-01-15T10:00:00Z",
        "updated_at": "2024-01-15T10:00:00Z"
      },
      {
        "uri_id": "f2a3b4c5-d6e7-8901-bcde-f01234567890",
        "uri": "https://app.example.com/logout",
        "type": "logout-uri",
        "created_at": "2024-01-15T10:00:00Z",
        "updated_at": "2024-01-15T10:00:00Z"
      }
    ]
  }
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `uris` | array | List of URI objects. Empty array if no URIs are registered. |
| `uris[].uri_id` | UUID | Unique identifier for this URI entry. |
| `uris[].uri` | string | The registered URI value. |
| `uris[].type` | string | URI type. One of: `redirect-uri`, `origin-uri`, `logout-uri`, `login-uri`, `cors-origin-uri`. |
| `uris[].created_at` | timestamp | ISO 8601 creation timestamp. |
| `uris[].updated_at` | timestamp | ISO 8601 last-updated timestamp. |

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid client UUID format |
| 404 | Client not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc/uris" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/clients/{client_uuid}/uris

Adds a new URI to the client's registered URI list.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `uri` | string | Yes | The URI to register. 5–200 characters. |
| `type` | string | Yes | URI type. One of: `redirect-uri`, `origin-uri`, `logout-uri`, `login-uri`, `cors-origin-uri`. |

### Response — 201 Created

```json
{
  "success": true,
  "message": "URI created successfully",
  "data": {
    "uri_id": "e1f2a3b4-c5d6-7890-abcd-ef0123456789",
    "uri": "https://app.example.com/callback",
    "type": "redirect-uri",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID, invalid URI type, or URI too short/long |
| 404 | Client not found |
| 409 | URI already registered for this client |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc/uris" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "uri": "https://app.example.com/callback",
    "type": "redirect-uri"
  }'
```

---

## PUT /api/v1/clients/{client_uuid}/uris/{client_uri_uuid}

Updates the URI value and/or type for an existing URI entry.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client |
| `client_uri_uuid` | UUID | The UUID of the URI entry to update |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `uri` | string | Yes | The new URI value. 5–200 characters. |
| `type` | string | Yes | URI type. One of: `redirect-uri`, `origin-uri`, `logout-uri`, `login-uri`, `cors-origin-uri`. |

### Response — 200 OK

```json
{
  "success": true,
  "message": "URI updated successfully",
  "data": {
    "uri_id": "e1f2a3b4-c5d6-7890-abcd-ef0123456789",
    "uri": "https://app.example.com/auth/callback",
    "type": "redirect-uri",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format or invalid URI type |
| 404 | Client or URI entry not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc/uris/e1f2a3b4-c5d6-7890-abcd-ef0123456789" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "uri": "https://app.example.com/auth/callback",
    "type": "redirect-uri"
  }'
```

---

## DELETE /api/v1/clients/{client_uuid}/uris/{client_uri_uuid}

Removes a URI from the client. After deletion, the OAuth2 flows will no longer accept this URI as a redirect destination.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `client_uuid` | UUID | The UUID of the client |
| `client_uri_uuid` | UUID | The UUID of the URI entry to remove |

### Response — 200 OK

Returns the updated full client object with the URI removed.

```json
{
  "success": true,
  "message": "URI deleted successfully",
  "data": {
    "client_id": "d4e5f6a7-b8c9-0123-def0-123456789abc",
    "name": "my-web-app",
    "display_name": "My Web Application",
    "client_type": "traditional",
    "domain": "app.example.com",
    "uris": [],
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
| 400 | Invalid UUID format |
| 404 | Client or URI entry not found |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/clients/d4e5f6a7-b8c9-0123-def0-123456789abc/uris/e1f2a3b4-c5d6-7890-abcd-ef0123456789" \
  -H "Authorization: Bearer <token>"
```
