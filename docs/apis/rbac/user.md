# Users

User management endpoints for creating, updating, and querying users within a tenant. All endpoints are tenant-scoped — the authenticated tenant context is validated by middleware and enforced at the service layer.

---

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/users` | Bearer JWT | List all users for the tenant |
| GET | `/api/v1/users/{user_uuid}` | Bearer JWT | Get a specific user by UUID |
| POST | `/api/v1/users` | Bearer JWT | Create a new user |
| PUT | `/api/v1/users/{user_uuid}` | Bearer JWT | Update a user |
| PATCH | `/api/v1/users/{user_uuid}/status` | Bearer JWT | Update user status |
| PATCH | `/api/v1/users/{user_uuid}/verify-email` | Bearer JWT | Mark user's email as verified |
| PATCH | `/api/v1/users/{user_uuid}/verify-phone` | Bearer JWT | Mark user's phone as verified |
| PATCH | `/api/v1/users/{user_uuid}/complete-account` | Bearer JWT | Mark user's account as completed |
| DELETE | `/api/v1/users/{user_uuid}` | Bearer JWT | Delete a user |

---

## GET /api/v1/users

Returns a paginated list of users belonging to the authenticated tenant. Supports filtering by username, email, phone, status, role, user pool, and client.

### Authentication

Bearer JWT required.

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page` | integer | Yes | Page number (minimum: 1) |
| `limit` | integer | Yes | Results per page (minimum: 1) |
| `sort_by` | string | No | Field to sort by |
| `sort_order` | string | No | `asc` or `desc` |
| `username` | string | No | Filter by username (partial match) |
| `email` | string | No | Filter by email (partial match) |
| `phone` | string | No | Filter by phone (partial match) |
| `status` | string | No | Filter by status: `active` or `inactive` |
| `role_id` | UUID | No | Filter users who have this role |
| `user_pool_id` | UUID | No | Filter by user pool UUID |
| `client_id` | UUID | No | Filter by client UUID |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Users fetched successfully",
  "data": {
    "rows": [
      {
        "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "username": "jdoe",
        "fullname": "John Doe",
        "email": "john@example.com",
        "phone": "+15551234567",
        "is_email_verified": true,
        "is_phone_verified": false,
        "is_profile_completed": true,
        "is_account_completed": true,
        "status": "active",
        "metadata": {},
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
| 401 | Missing or invalid JWT, or tenant not found in context |
| 422 | Validation error on query parameters |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/users?page=1&limit=10&status=active" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/users/{user_uuid}

Returns a single user by UUID. The service validates that the user belongs to the tenant.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_uuid` | UUID | The UUID of the user |

### Response — 200 OK

```json
{
  "success": true,
  "message": "User fetched successfully",
  "data": {
    "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "username": "jdoe",
    "fullname": "John Doe",
    "email": "john@example.com",
    "phone": "+15551234567",
    "is_email_verified": true,
    "is_phone_verified": false,
    "is_profile_completed": true,
    "is_account_completed": true,
    "status": "active",
    "metadata": {},
    "tenant": {
      "tenant_id": "f1e2d3c4-b5a6-7890-abcd-ef1234567890",
      "name": "Acme Corp",
      "description": "Acme Corp tenant",
      "identifier": "acme",
      "status": "active",
      "is_public": false,
      "is_system": false,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    },
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid user UUID format |
| 401 | Missing or invalid JWT |
| 404 | User not found or does not belong to tenant |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/users

Creates a new user within the authenticated tenant.

### Authentication

Bearer JWT required.

### Request Body

```json
{
  "username": "jdoe",
  "fullname": "John Doe",
  "email": "john@example.com",
  "phone": "+15551234567",
  "password": "securepassword123",
  "status": "active",
  "metadata": {}
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | Yes | 3–50 characters |
| `fullname` | string | Yes | 1–255 characters |
| `email` | string | No | Valid email address |
| `phone` | string | No | 10–20 characters |
| `password` | string | Yes | 8–100 characters |
| `status` | string | Yes | `active`, `inactive`, `pending`, or `suspended` |
| `metadata` | object | No | Arbitrary JSON metadata |

### Response — 201 Created

```json
{
  "success": true,
  "message": "User created successfully",
  "data": {
    "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "username": "jdoe",
    "fullname": "John Doe",
    "email": "john@example.com",
    "phone": "+15551234567",
    "is_email_verified": false,
    "is_phone_verified": false,
    "is_profile_completed": false,
    "is_account_completed": false,
    "status": "active",
    "metadata": {},
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Malformed JSON |
| 401 | Missing or invalid JWT |
| 422 | Validation error |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/users" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "jdoe",
    "fullname": "John Doe",
    "email": "john@example.com",
    "password": "securepassword123",
    "status": "active"
  }'
```

---

## PUT /api/v1/users/{user_uuid}

Updates an existing user's information. The service validates tenant ownership.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_uuid` | UUID | The UUID of the user to update |

### Request Body

```json
{
  "username": "jdoe2",
  "fullname": "John Doe II",
  "email": "john2@example.com",
  "phone": "+15559876543",
  "status": "active",
  "metadata": {}
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | Yes | 3–50 characters |
| `fullname` | string | Yes | 1–255 characters |
| `email` | string | No | Valid email address |
| `phone` | string | No | 10–20 characters |
| `status` | string | Yes | `active`, `inactive`, `pending`, or `suspended` |
| `metadata` | object | No | Arbitrary JSON metadata |

### Response — 200 OK

```json
{
  "success": true,
  "message": "User updated successfully",
  "data": {
    "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "username": "jdoe2",
    "fullname": "John Doe II",
    "email": "john2@example.com",
    "phone": "+15559876543",
    "is_email_verified": true,
    "is_phone_verified": false,
    "is_profile_completed": true,
    "is_account_completed": true,
    "status": "active",
    "metadata": {},
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID or malformed JSON |
| 401 | Missing or invalid JWT |
| 404 | User not found or does not belong to tenant |
| 422 | Validation error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"username": "jdoe2", "fullname": "John Doe II", "status": "active"}'
```

---

## PATCH /api/v1/users/{user_uuid}/status

Updates only the status of a user. A convenience endpoint for status-only changes.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_uuid` | UUID | The UUID of the user |

### Request Body

```json
{
  "status": "suspended"
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | `active`, `inactive`, `pending`, or `suspended` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "User status updated successfully",
  "data": {
    "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "username": "jdoe",
    "fullname": "John Doe",
    "email": "john@example.com",
    "phone": "+15551234567",
    "is_email_verified": true,
    "is_phone_verified": false,
    "is_profile_completed": true,
    "is_account_completed": true,
    "status": "suspended",
    "metadata": {},
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID or malformed JSON |
| 401 | Missing or invalid JWT |
| 404 | User not found or does not belong to tenant |
| 422 | Validation error |

### Example

```bash
curl -X PATCH "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "suspended"}'
```

---

## PATCH /api/v1/users/{user_uuid}/verify-email

Marks a user's email address as verified. The service validates tenant ownership.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_uuid` | UUID | The UUID of the user |

### Request Body

None.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Email verified and account completed successfully",
  "data": {
    "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "username": "jdoe",
    "fullname": "John Doe",
    "email": "john@example.com",
    "phone": "+15551234567",
    "is_email_verified": true,
    "is_phone_verified": false,
    "is_profile_completed": true,
    "is_account_completed": true,
    "status": "active",
    "metadata": {},
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid user UUID format |
| 401 | Missing or invalid JWT |
| 404 | User not found or does not belong to tenant |

### Example

```bash
curl -X PATCH "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890/verify-email" \
  -H "Authorization: Bearer <token>"
```

---

## PATCH /api/v1/users/{user_uuid}/verify-phone

Marks a user's phone number as verified.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_uuid` | UUID | The UUID of the user |

### Request Body

None.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Phone verified successfully",
  "data": {
    "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "username": "jdoe",
    "fullname": "John Doe",
    "email": "john@example.com",
    "phone": "+15551234567",
    "is_email_verified": true,
    "is_phone_verified": true,
    "is_profile_completed": true,
    "is_account_completed": true,
    "status": "active",
    "metadata": {},
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid user UUID format |
| 401 | Missing or invalid JWT |
| 404 | User not found or does not belong to tenant |

### Example

```bash
curl -X PATCH "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890/verify-phone" \
  -H "Authorization: Bearer <token>"
```

---

## PATCH /api/v1/users/{user_uuid}/complete-account

Manually marks a user's account as completed. Typically called after all required profile information and verifications are in place.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_uuid` | UUID | The UUID of the user |

### Request Body

None.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Account marked as completed successfully",
  "data": {
    "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "username": "jdoe",
    "fullname": "John Doe",
    "email": "john@example.com",
    "phone": "+15551234567",
    "is_email_verified": true,
    "is_phone_verified": true,
    "is_profile_completed": true,
    "is_account_completed": true,
    "status": "active",
    "metadata": {},
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid user UUID format |
| 401 | Missing or invalid JWT |
| 404 | User not found or does not belong to tenant |

### Example

```bash
curl -X PATCH "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890/complete-account" \
  -H "Authorization: Bearer <token>"
```

---

## DELETE /api/v1/users/{user_uuid}

Deletes a user from the tenant. The service validates tenant ownership.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_uuid` | UUID | The UUID of the user to delete |

### Response — 200 OK

```json
{
  "success": true,
  "message": "User deleted successfully",
  "data": {
    "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "username": "jdoe",
    "fullname": "John Doe",
    "email": "john@example.com",
    "phone": "+15551234567",
    "is_email_verified": true,
    "is_phone_verified": false,
    "is_profile_completed": true,
    "is_account_completed": true,
    "status": "active",
    "metadata": {},
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid user UUID format |
| 401 | Missing or invalid JWT |
| 404 | User not found or does not belong to tenant |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>"
```
