# User Profiles (Admin)

Admin endpoints for managing profiles belonging to any user within the tenant. These endpoints operate on behalf of a specified user rather than the authenticated caller. For self-service profile management see [Profile](../profile/profile.md).

---

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/users/{user_uuid}/profiles` | Bearer JWT | List all profiles for a user |
| GET | `/api/v1/users/{user_uuid}/profiles/{profile_uuid}` | Bearer JWT | Get a specific profile for a user |
| POST | `/api/v1/users/{user_uuid}/profiles` | Bearer JWT | Create a profile for a user |
| PUT | `/api/v1/users/{user_uuid}/profiles/{profile_uuid}` | Bearer JWT | Update a profile for a user |
| DELETE | `/api/v1/users/{user_uuid}/profiles/{profile_uuid}` | Bearer JWT | Delete a profile for a user |
| PATCH | `/api/v1/users/{user_uuid}/profiles/{profile_uuid}/default` | Bearer JWT | Set a profile as the user's default |

---

## GET /api/v1/users/{user_uuid}/profiles

Returns a paginated list of profiles belonging to the specified user.

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
| `sort_by` | string | No | Field to sort by |
| `sort_order` | string | No | `asc` or `desc` |
| `first_name` | string | No | Filter by first name |
| `last_name` | string | No | Filter by last name |
| `email` | string | No | Filter by email |
| `phone` | string | No | Filter by phone |
| `city` | string | No | Filter by city |
| `country` | string | No | Filter by 2-character ISO country code |
| `is_default` | boolean | No | Filter by default flag |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Profiles fetched successfully",
  "data": {
    "rows": [
      {
        "profile_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
        "first_name": "John",
        "middle_name": "Michael",
        "last_name": "Doe",
        "suffix": null,
        "display_name": "JD",
        "bio": "Software engineer",
        "birthdate": "1990-05-15T00:00:00Z",
        "gender": "male",
        "phone": "+15551234567",
        "email": "john@example.com",
        "address": "123 Main St",
        "city": "San Francisco",
        "country": "US",
        "timezone": "America/Los_Angeles",
        "language": "en",
        "profile_url": "https://example.com/avatar.jpg",
        "is_default": true,
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
| 400 | Invalid user UUID format |
| 401 | Missing or invalid JWT |
| 422 | Validation error on query parameters |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890/profiles?page=1&limit=10" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/users/{user_uuid}/profiles/{profile_uuid}

Returns a specific profile by UUID for the specified user.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_uuid` | UUID | The UUID of the user |
| `profile_uuid` | UUID | The UUID of the profile |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Profile retrieved successfully",
  "data": {
    "profile_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
    "first_name": "John",
    "last_name": "Doe",
    "display_name": "JD",
    "bio": "Software engineer",
    "birthdate": "1990-05-15T00:00:00Z",
    "gender": "male",
    "phone": "+15551234567",
    "email": "john@example.com",
    "city": "San Francisco",
    "country": "US",
    "timezone": "America/Los_Angeles",
    "language": "en",
    "profile_url": "https://example.com/avatar.jpg",
    "is_default": true,
    "metadata": {},
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 401 | Missing or invalid JWT |
| 404 | Profile not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890/profiles/c3d4e5f6-a7b8-9012-cdef-123456789012" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/users/{user_uuid}/profiles

Creates a new profile for the specified user. A new UUID is automatically generated for the profile.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_uuid` | UUID | The UUID of the user |

### Request Body

```json
{
  "first_name": "John",
  "middle_name": "Michael",
  "last_name": "Doe",
  "suffix": "Jr",
  "display_name": "JD",
  "bio": "Software engineer",
  "birthdate": "1990-05-15",
  "gender": "male",
  "phone": "+15551234567",
  "email": "john@example.com",
  "address": "123 Main St",
  "city": "San Francisco",
  "country": "US",
  "timezone": "America/Los_Angeles",
  "language": "en",
  "profile_url": "https://example.com/avatar.jpg",
  "metadata": {}
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `first_name` | string | Yes | 1–100 characters |
| `middle_name` | string | No | Up to 100 characters |
| `last_name` | string | No | Up to 100 characters |
| `suffix` | string | No | Up to 50 characters |
| `display_name` | string | No | Up to 100 characters |
| `bio` | string | No | Up to 1000 characters |
| `birthdate` | string | No | Format: `YYYY-MM-DD` |
| `gender` | string | No | `male`, `female`, `other`, or `prefer_not_to_say` |
| `phone` | string | No | Up to 20 characters |
| `email` | string | No | Valid email address, up to 255 characters |
| `address` | string | No | Up to 500 characters |
| `city` | string | No | Up to 100 characters |
| `country` | string | No | 2-character ISO 3166-1 alpha-2 code (e.g., `US`) |
| `timezone` | string | No | Up to 50 characters (e.g., `America/Los_Angeles`) |
| `language` | string | No | Up to 10 characters (ISO 639-1, e.g., `en`) |
| `profile_url` | string | No | Valid URL, up to 1000 characters |
| `metadata` | object | No | Arbitrary key-value pairs |

### Response — 201 Created

```json
{
  "success": true,
  "message": "Profile created successfully",
  "data": {
    "profile_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
    "first_name": "John",
    "last_name": "Doe",
    "display_name": "JD",
    "bio": "Software engineer",
    "birthdate": "1990-05-15T00:00:00Z",
    "gender": "male",
    "phone": "+15551234567",
    "email": "john@example.com",
    "city": "San Francisco",
    "country": "US",
    "timezone": "America/Los_Angeles",
    "language": "en",
    "profile_url": "https://example.com/avatar.jpg",
    "is_default": false,
    "metadata": {},
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID or malformed JSON |
| 401 | Missing or invalid JWT |
| 422 | Validation error |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890/profiles" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"first_name": "John", "last_name": "Doe", "country": "US"}'
```

---

## PUT /api/v1/users/{user_uuid}/profiles/{profile_uuid}

Updates an existing profile for the specified user using upsert semantics — the profile is created if it does not already exist with that UUID.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_uuid` | UUID | The UUID of the user |
| `profile_uuid` | UUID | The UUID of the profile to update |

### Request Body

Same fields as [POST /api/v1/users/{user_uuid}/profiles](#post-apiv1usersuser_uuidprofiles).

### Response — 200 OK

```json
{
  "success": true,
  "message": "Profile updated successfully",
  "data": {
    "profile_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
    "first_name": "John",
    "last_name": "Doe",
    "city": "New York",
    "country": "US",
    "is_default": true,
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
| 422 | Validation error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890/profiles/c3d4e5f6-a7b8-9012-cdef-123456789012" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"first_name": "John", "last_name": "Doe", "city": "New York", "country": "US"}'
```

---

## DELETE /api/v1/users/{user_uuid}/profiles/{profile_uuid}

Deletes a specific profile for the specified user.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_uuid` | UUID | The UUID of the user |
| `profile_uuid` | UUID | The UUID of the profile to delete |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Profile deleted successfully",
  "data": {
    "profile_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
    "first_name": "John",
    "last_name": "Doe",
    "is_default": false,
    "metadata": {},
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 401 | Missing or invalid JWT |
| 404 | Profile not found |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890/profiles/c3d4e5f6-a7b8-9012-cdef-123456789012" \
  -H "Authorization: Bearer <token>"
```

---

## PATCH /api/v1/users/{user_uuid}/profiles/{profile_uuid}/default

Sets the specified profile as the user's default profile. The previous default is unset automatically.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_uuid` | UUID | The UUID of the user |
| `profile_uuid` | UUID | The UUID of the profile to set as default |

### Request Body

None.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Profile set as default successfully",
  "data": {
    "profile_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
    "first_name": "John",
    "last_name": "Doe",
    "is_default": true,
    "metadata": {},
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 401 | Missing or invalid JWT |
| 404 | Profile not found |

### Example

```bash
curl -X PATCH "http://localhost:8080/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890/profiles/c3d4e5f6-a7b8-9012-cdef-123456789012/default" \
  -H "Authorization: Bearer <token>"
```
