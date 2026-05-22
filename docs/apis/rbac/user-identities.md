# User Identities

User identities represent external authentication providers (such as Google, GitHub, or other OAuth/OIDC providers) linked to a user account. Each identity record captures the provider name, the provider's subject identifier (`sub`), and the associated client.

---

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/users/{user_uuid}/identities` | Bearer JWT | List all identity providers linked to a user |

---

## GET /api/v1/users/{user_uuid}/identities

Returns a paginated list of identity records for the specified user. The user must belong to the authenticated tenant. Supports filtering by provider with in-memory sorting and pagination applied after fetching.

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
| `sort_by` | string | No | Field to sort by: `provider`, `sub`, `created_at`, `updated_at` |
| `sort_order` | string | No | `asc` or `desc` |
| `provider` | string | No | Filter by provider name (partial, case-insensitive match) |

### Response — 200 OK

```json
{
  "success": true,
  "message": "User identities fetched successfully",
  "data": {
    "rows": [
      {
        "user_identity_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "provider": "google",
        "sub": "109876543210987654321",
        "metadata": {},
        "client": {
          "client_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
          "name": "web-app",
          "display_name": "Web Application",
          "client_type": "public",
          "domain": "app.example.com",
          "status": "active",
          "is_default": true,
          "is_system": false,
          "created_at": "2024-01-01T00:00:00Z",
          "updated_at": "2024-01-01T00:00:00Z"
        },
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

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `user_identity_id` | UUID | Unique identifier for this identity record |
| `provider` | string | Identity provider name (e.g., `google`, `github`) |
| `sub` | string | Subject identifier from the identity provider |
| `metadata` | object | Provider-supplied metadata stored at link time |
| `client` | object | The auth client through which this identity was linked. Present only when a client is associated. |
| `created_at` | datetime | When the identity was linked |
| `updated_at` | datetime | When the identity record was last updated |

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid user UUID format |
| 401 | Missing or invalid JWT, or tenant not found in context |
| 404 | User not found or does not belong to tenant |
| 422 | Validation error on query parameters |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890/identities?page=1&limit=10" \
  -H "Authorization: Bearer <token>"
```

#### Filter by provider

```bash
curl -X GET "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890/identities?page=1&limit=10&provider=google" \
  -H "Authorization: Bearer <token>"
```
