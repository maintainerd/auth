# Tenant Members API

The Tenant Members API manages user membership within a tenant. Members have a role (`owner` or `member`) that governs their capabilities within the tenant.

> **Port 8080 only.** All endpoints require a valid Bearer JWT and the appropriate permission scope.

---

## Endpoints

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| GET | `/api/v1/tenants/{tenant_uuid}/members` | `tenant:read` | List members of a tenant |
| POST | `/api/v1/tenants/{tenant_uuid}/members` | `tenant:update` | Add a user to a tenant |
| PATCH | `/api/v1/tenants/{tenant_uuid}/members/{tenant_member_uuid}/role` | `tenant:update` | Update a member's role |
| DELETE | `/api/v1/tenants/{tenant_uuid}/members/{tenant_member_uuid}` | `tenant:update` | Remove a member from a tenant |

---

## Tenant Member Object

All member endpoints return objects in this shape:

```json
{
  "tenant_member_id": "018e1b2c-cccc-7f8a-9b0c-1d2e3f4a5b6c",
  "role": "member",
  "user": {
    "user_id": "018e1b2c-aaaa-7f8a-9b0c-1d2e3f4a5b6c",
    "username": "jdoe",
    "fullname": "Jane Doe",
    "email": "jdoe@example.com",
    "phone": "",
    "is_email_verified": true,
    "is_phone_verified": false,
    "is_profile_completed": true,
    "is_account_completed": true,
    "status": "active",
    "metadata": {},
    "created_at": "2026-01-15T10:00:00Z",
    "updated_at": "2026-01-15T10:00:00Z"
  },
  "created_at": "2026-01-15T10:05:00Z",
  "updated_at": "2026-01-15T10:05:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `tenant_member_id` | UUID | Unique identifier for the membership record |
| `role` | string | `owner` or `member` |
| `user` | object | The associated user object (see [User Object](#user-object)) |
| `created_at` | string | ISO 8601 timestamp — when the membership was created |
| `updated_at` | string | ISO 8601 timestamp — when the membership was last modified |

### User Object

| Field | Type | Description |
|-------|------|-------------|
| `user_id` | UUID | Unique identifier for the user |
| `username` | string | The user's login username |
| `fullname` | string | The user's full name |
| `email` | string | Primary email address |
| `phone` | string | Primary phone number |
| `is_email_verified` | boolean | Whether the email address has been verified |
| `is_phone_verified` | boolean | Whether the phone number has been verified |
| `is_profile_completed` | boolean | Whether the user has completed their profile |
| `is_account_completed` | boolean | Whether account setup is fully complete |
| `status` | string | `active`, `inactive`, `pending`, or `suspended` |
| `metadata` | object | Arbitrary JSON metadata |
| `created_at` | string | ISO 8601 timestamp |
| `updated_at` | string | ISO 8601 timestamp |

---

## GET /api/v1/tenants/{tenant_uuid}/members

Returns all members of a tenant.

### Authentication

Bearer JWT with `tenant:read` permission.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `tenant_uuid` | UUID | The tenant's unique identifier |

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page` | integer | Yes | Page number (minimum: 1) |
| `limit` | integer | Yes | Results per page (minimum: 1) |
| `sort_by` | string | No | Field to sort by |
| `sort_order` | string | No | `asc` or `desc` |
| `role` | string | No | Filter by role: `owner` or `member` |

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Members retrieved successfully",
  "data": [
    {
      "tenant_member_id": "018e1b2c-cccc-7f8a-9b0c-1d2e3f4a5b6c",
      "role": "owner",
      "user": { ... },
      "created_at": "2026-01-15T10:05:00Z",
      "updated_at": "2026-01-15T10:05:00Z"
    }
  ]
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid or missing `tenant_uuid` |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions |
| 404 | Tenant not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/tenants/018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c/members?page=1&limit=20" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/tenants/{tenant_uuid}/members

Adds an existing user to a tenant with a specified role.

### Authentication

Bearer JWT with `tenant:update` permission.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `tenant_uuid` | UUID | The tenant's unique identifier |

### Request Body

```json
{
  "user_id": "018e1b2c-aaaa-7f8a-9b0c-1d2e3f4a5b6c",
  "role": "member"
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `user_id` | UUID | Yes | Must be a valid UUID of an existing user |
| `role` | string | Yes | `owner` or `member` |

### Response

#### 201 Created

```json
{
  "success": true,
  "message": "Member added successfully",
  "data": {
    "tenant_member_id": "018e1b2c-cccc-7f8a-9b0c-1d2e3f4a5b6c",
    "role": "member",
    "user": { ... },
    "created_at": "2026-01-15T10:05:00Z",
    "updated_at": "2026-01-15T10:05:00Z"
  }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body, missing fields, or invalid UUID |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions |
| 404 | Tenant or user not found |
| 409 | User is already a member of this tenant |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/tenants/018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c/members" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "018e1b2c-aaaa-7f8a-9b0c-1d2e3f4a5b6c",
    "role": "member"
  }'
```

---

## PATCH /api/v1/tenants/{tenant_uuid}/members/{tenant_member_uuid}/role

Updates the role of an existing tenant member.

### Authentication

Bearer JWT with `tenant:update` permission.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `tenant_uuid` | UUID | The tenant's unique identifier |
| `tenant_member_uuid` | UUID | The membership record's unique identifier |

### Request Body

```json
{
  "role": "owner"
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `role` | string | Yes | `owner` or `member` |

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Member role updated successfully",
  "data": {
    "tenant_member_id": "018e1b2c-cccc-7f8a-9b0c-1d2e3f4a5b6c",
    "role": "owner",
    "user": { ... },
    "created_at": "2026-01-15T10:05:00Z",
    "updated_at": "2026-01-15T10:10:00Z"
  }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID or request body |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions |
| 404 | Membership record not found |

### Example

```bash
curl -X PATCH "http://localhost:8080/api/v1/tenants/018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c/members/018e1b2c-cccc-7f8a-9b0c-1d2e3f4a5b6c/role" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"role": "owner"}'
```

---

## DELETE /api/v1/tenants/{tenant_uuid}/members/{tenant_member_uuid}

Removes a user from a tenant by deleting their membership record.

### Authentication

Bearer JWT with `tenant:update` permission.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `tenant_uuid` | UUID | The tenant's unique identifier |
| `tenant_member_uuid` | UUID | The membership record's unique identifier |

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Member removed successfully",
  "data": null
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid or missing UUID |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions |
| 404 | Membership record not found |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/tenants/018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c/members/018e1b2c-cccc-7f8a-9b0c-1d2e3f4a5b6c" \
  -H "Authorization: Bearer <token>"
```
