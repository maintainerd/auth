# Consent Grant Management

Consent grants are the persisted records of which scopes a user has approved for a client. These endpoints let users view and revoke their existing grants.

Revoking a grant does not immediately invalidate active tokens — it means the user will be prompted to re-consent the next time that client initiates an authorization request.

**Authentication required.** Both endpoints require a valid JWT access token.

---

## Endpoints

| Method | Path | Auth |
|--------|------|------|
| GET | `/api/v1/oauth/consent/grants` | Bearer JWT |
| DELETE | `/api/v1/oauth/consent/grants/{grant_uuid}` | Bearer JWT |

---

## GET /api/v1/oauth/consent/grants

Returns all consent grants for the authenticated user. Each grant represents a client that the user has authorized, along with the approved scopes and timestamps.

### Authentication

Bearer JWT in the `Authorization` header.

### Request

#### Headers

| Name | Required | Description |
|------|----------|-------------|
| `Authorization` | Yes | `Bearer <access_token>` |

### Response

#### Success — 200 OK

```json
{
  "success": true,
  "message": "Consent grants retrieved",
  "data": [
    {
      "consent_grant_id": "f1a2b3c4-1234-5678-abcd-ef0123456789",
      "client_name": "My Application",
      "client_uuid": "d1e2f3a4-5678-90ab-cdef-012345678901",
      "scopes": ["openid", "email"],
      "granted_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

An empty array is returned when the user has no active consent grants.

| Field | Type | Description |
|-------|------|-------------|
| `consent_grant_id` | string (UUID) | Unique identifier for this consent grant. Use this UUID to revoke the grant. |
| `client_name` | string | Human-readable display name of the client. |
| `client_uuid` | string | UUID of the client application. |
| `scopes` | array of strings | The scopes the user has approved for this client. |
| `granted_at` | string (ISO 8601) | When the grant was first created. |
| `updated_at` | string (ISO 8601) | When the grant was last updated (e.g., new scopes added). |

#### Error Responses

| Status | Description |
|--------|-------------|
| 401 | `{ "success": false, "message": "Authentication required" }` — no valid JWT was provided. |
| 500 | `{ "success": false, "message": "Failed to retrieve consent grants" }` — internal error. |

### Example

```bash
curl "https://auth.example.com/api/v1/oauth/consent/grants" \
  -H "Authorization: Bearer <access_token>"
```

---

## DELETE /api/v1/oauth/consent/grants/{grant_uuid}

Revokes a specific consent grant. The user will be prompted to re-consent the next time the associated client initiates an authorization request.

Only the owner of the grant can revoke it — the `grant_uuid` must belong to the authenticated user.

### Authentication

Bearer JWT in the `Authorization` header.

### Request

#### Headers

| Name | Required | Description |
|------|----------|-------------|
| `Authorization` | Yes | `Bearer <access_token>` |

#### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `grant_uuid` | UUID | Yes | The `consent_grant_id` of the grant to revoke. Obtain this from `GET /consent/grants`. |

### Response

#### Success — 200 OK

```json
{
  "success": true,
  "message": "Consent grant revoked",
  "data": null
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | `{ "success": false, "message": "Invalid grant UUID" }` — `grant_uuid` is not a valid UUID. |
| 401 | `{ "success": false, "message": "Authentication required" }` — no valid JWT was provided. |
| 404 | `{ "success": false, "message": "Failed to revoke consent grant" }` — the grant was not found or does not belong to the authenticated user. |
| 500 | `{ "success": false, "message": "Failed to revoke consent grant" }` — internal error. |

### Example

```bash
curl -X DELETE "https://auth.example.com/api/v1/oauth/consent/grants/f1a2b3c4-1234-5678-abcd-ef0123456789" \
  -H "Authorization: Bearer <access_token>"
```
