# Profile

Self-service profile endpoints for the authenticated user. Profiles store personal information, contact details, location, and preferences. A user can have multiple profiles; one is marked as the default.

These endpoints are available on both the management port (8080) and the auth port (8081). All require a valid Bearer JWT.

For admin-level profile management across users, see [User Profiles (Admin)](../rbac/user-profiles.md).

---

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/profile` | Bearer JWT | Get the authenticated user's default profile |
| POST | `/api/v1/profile` | Bearer JWT | Create or update the authenticated user's default profile |
| PUT | `/api/v1/profile` | Bearer JWT | Create or update the authenticated user's default profile |
| DELETE | `/api/v1/profile` | Bearer JWT | Delete the authenticated user's default profile |
| GET | `/api/v1/profiles` | Bearer JWT | List all profiles for the authenticated user |
| POST | `/api/v1/profiles` | Bearer JWT | Create a new profile for the authenticated user |
| GET | `/api/v1/profiles/{profile_uuid}` | Bearer JWT | Get a specific profile by UUID |
| PUT | `/api/v1/profiles/{profile_uuid}` | Bearer JWT | Update a specific profile by UUID |
| PATCH | `/api/v1/profiles/{profile_uuid}/set-default` | Bearer JWT | Set a profile as the user's default |
| DELETE | `/api/v1/profiles/{profile_uuid}` | Bearer JWT | Delete a specific profile by UUID |

---

## GET /api/v1/profile

Returns the authenticated user's default profile.

### Authentication

Bearer JWT required.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Profile retrieved successfully",
  "data": {
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
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Missing or invalid JWT |
| 404 | No default profile found for this user |

### Example

```bash
curl -X GET "https://auth.example.com/api/v1/profile" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/profile

Creates or updates the authenticated user's default profile. If a default profile already exists it is overwritten; otherwise a new one is created and marked as default.

### Authentication

Bearer JWT required.

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
| `timezone` | string | No | Up to 50 characters |
| `language` | string | No | Up to 10 characters (ISO 639-1) |
| `profile_url` | string | No | Valid URL, up to 1000 characters |
| `metadata` | object | No | Arbitrary key-value pairs |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Profile saved successfully",
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
    "updated_at": "2024-01-16T09:30:00Z"
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
curl -X POST "https://auth.example.com/api/v1/profile" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"first_name": "John", "last_name": "Doe", "country": "US"}'
```

---

## DELETE /api/v1/profile

Deletes the authenticated user's default profile.

### Authentication

Bearer JWT required.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Profile deleted successfully",
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
| 401 | Missing or invalid JWT |
| 404 | No default profile found |

### Example

```bash
curl -X DELETE "https://auth.example.com/api/v1/profile" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/profiles

Returns a paginated list of all profiles belonging to the authenticated user.

### Authentication

Bearer JWT required.

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
        "last_name": "Doe",
        "display_name": "JD",
        "city": "San Francisco",
        "country": "US",
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
| 401 | Missing or invalid JWT |
| 422 | Validation error on query parameters |

### Example

```bash
curl -X GET "https://auth.example.com/api/v1/profiles?page=1&limit=10" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/profiles

Creates a new profile for the authenticated user. A UUID is automatically generated. This profile is not set as default automatically.

### Authentication

Bearer JWT required.

### Request Body

Same fields as [POST /api/v1/profile](#post-apiv1profile).

### Response — 201 Created

```json
{
  "success": true,
  "message": "Profile created successfully",
  "data": {
    "profile_id": "d4e5f6a7-b8c9-0123-def0-234567890123",
    "first_name": "John",
    "last_name": "Doe",
    "country": "US",
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
| 400 | Malformed JSON |
| 401 | Missing or invalid JWT |
| 422 | Validation error |

### Example

```bash
curl -X POST "https://auth.example.com/api/v1/profiles" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"first_name": "John", "last_name": "Doe", "country": "US"}'
```

---

## GET /api/v1/profiles/{profile_uuid}

Returns a specific profile by UUID. Ownership is verified — the profile must belong to the authenticated user.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
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
    "city": "San Francisco",
    "country": "US",
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
| 400 | Invalid profile UUID format |
| 401 | Missing or invalid JWT |
| 404 | Profile not found or does not belong to user |

### Example

```bash
curl -X GET "https://auth.example.com/api/v1/profiles/c3d4e5f6-a7b8-9012-cdef-123456789012" \
  -H "Authorization: Bearer <token>"
```

---

## PUT /api/v1/profiles/{profile_uuid}

Updates a specific profile by UUID using upsert semantics. Ownership is verified.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `profile_uuid` | UUID | The UUID of the profile to update |

### Request Body

Same fields as [POST /api/v1/profile](#post-apiv1profile).

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
curl -X PUT "https://auth.example.com/api/v1/profiles/c3d4e5f6-a7b8-9012-cdef-123456789012" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"first_name": "John", "last_name": "Doe", "city": "New York", "country": "US"}'
```

---

## PATCH /api/v1/profiles/{profile_uuid}/set-default

Sets the specified profile as the authenticated user's default profile. Any previous default is automatically unset.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
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
| 400 | Invalid profile UUID format |
| 401 | Missing or invalid JWT |
| 404 | Profile not found or does not belong to user |

### Example

```bash
curl -X PATCH "https://auth.example.com/api/v1/profiles/c3d4e5f6-a7b8-9012-cdef-123456789012/set-default" \
  -H "Authorization: Bearer <token>"
```

---

## DELETE /api/v1/profiles/{profile_uuid}

Deletes a specific profile by UUID. Ownership is verified — the profile must belong to the authenticated user.

### Authentication

Bearer JWT required.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
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
| 400 | Invalid profile UUID format |
| 401 | Missing or invalid JWT |
| 404 | Profile not found or does not belong to user |

### Example

```bash
curl -X DELETE "https://auth.example.com/api/v1/profiles/c3d4e5f6-a7b8-9012-cdef-123456789012" \
  -H "Authorization: Bearer <token>"
```
