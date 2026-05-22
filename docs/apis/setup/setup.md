# Setup API

The Setup API provides the first-run bootstrap endpoints used to initialize a new maintainerd-auth installation. These endpoints must be called in sequence during initial deployment: create the tenant, create the admin user, then create the admin's profile.

> **Port 8080 only.** These endpoints are available exclusively on the internal management port and are never exposed publicly.

> **No authentication required.** All setup endpoints are unauthenticated. Once setup is complete, the endpoints reject further calls.

---

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/setup/status` | None | Check which setup steps have been completed |
| POST | `/api/v1/setup/create_tenant` | None | Create the initial tenant |
| POST | `/api/v1/setup/create_admin` | None | Create the initial admin user |
| POST | `/api/v1/setup/create_profile` | None | Create the admin user's profile |

---

## GET /api/v1/setup/status

Returns the completion state of each setup step. Poll this endpoint to determine which step to execute next.

### Authentication

None.

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Setup status retrieved successfully",
  "data": {
    "is_tenant_setup": false,
    "is_admin_setup": false,
    "is_profile_setup": false,
    "is_setup_complete": false
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `is_tenant_setup` | boolean | Whether the initial tenant has been created |
| `is_admin_setup` | boolean | Whether the initial admin user has been created |
| `is_profile_setup` | boolean | Whether the admin profile has been created |
| `is_setup_complete` | boolean | `true` when all three steps are done |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/setup/status"
```

---

## POST /api/v1/setup/create_tenant

Creates the initial (system) tenant. Also seeds default roles, clients, identity providers, and other required data. This must be the first setup step.

### Authentication

None.

### Request Body

```json
{
  "name": "my-org",
  "display_name": "My Organization",
  "description": "Primary organization tenant",
  "metadata": {
    "application_logo_url": "https://example.com/logo.png",
    "favicon_url": "https://example.com/favicon.ico",
    "language": "en",
    "timezone": "America/New_York",
    "date_format": "MM/DD/YYYY",
    "time_format": "12h",
    "privacy_policy_url": "https://example.com/privacy",
    "term_of_service_url": "https://example.com/terms"
  }
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `name` | string | Yes | 2–100 characters; letters, numbers, spaces, `-`, `_`, `.` |
| `display_name` | string | Yes | 2–100 characters |
| `description` | string | No | Max 500 characters |
| `metadata` | object | No | See metadata fields below |

**Metadata fields** (all optional):

| Field | Type | Constraints |
|-------|------|-------------|
| `application_logo_url` | string | Valid URL, max 500 characters |
| `favicon_url` | string | Valid URL, max 500 characters |
| `language` | string | Format: `en` or `en-US`, 2–10 characters |
| `timezone` | string | Max 50 characters |
| `date_format` | string | Max 20 characters |
| `time_format` | string | Max 20 characters |
| `privacy_policy_url` | string | Valid URL, max 500 characters |
| `term_of_service_url` | string | Valid URL, max 500 characters |

### Response

#### 201 Created

Returns the created tenant object.

```json
{
  "success": true,
  "message": "Tenant created successfully",
  "data": {
    "tenant_id": "018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c",
    "name": "my-org",
    "display_name": "My Organization",
    "description": "Primary organization tenant",
    "identifier": "my-org",
    "status": "active",
    "is_public": false,
    "is_system": true,
    "metadata": {},
    "created_at": "2026-01-15T10:00:00Z",
    "updated_at": "2026-01-15T10:00:00Z"
  }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or validation failure |
| 409 | Tenant already exists (setup step already completed) |
| 500 | Internal server error |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/setup/create_tenant" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-org",
    "display_name": "My Organization",
    "description": "Primary organization tenant"
  }'
```

---

## POST /api/v1/setup/create_admin

Creates the initial admin user. Requires the tenant setup step to be complete first.

### Authentication

None.

### Request Body

```json
{
  "username": "admin",
  "fullname": "System Administrator",
  "email": "admin@example.com",
  "password": "securepassword123"
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `username` | string | Yes | 3–50 characters; letters, numbers, `_`, `-`, `.`, `@` |
| `fullname` | string | Yes | 1–255 characters |
| `email` | string | Yes | Valid email, max 100 characters |
| `password` | string | Yes | 8–100 characters |

### Response

#### 201 Created

Returns the created admin user object. A `token_response` may be included if the system auto-issues a session token on admin creation.

```json
{
  "success": true,
  "message": "Admin user created successfully",
  "data": {
    "user_id": "018e1b2c-aaaa-7f8a-9b0c-1d2e3f4a5b6c",
    "username": "admin",
    "fullname": "System Administrator",
    "email": "admin@example.com",
    "phone": "",
    "is_email_verified": false,
    "is_phone_verified": false,
    "is_profile_completed": false,
    "is_account_completed": false,
    "status": "active",
    "metadata": {},
    "created_at": "2026-01-15T10:01:00Z",
    "updated_at": "2026-01-15T10:01:00Z"
  }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or validation failure |
| 409 | Admin user already exists (setup step already completed) |
| 422 | Tenant setup step not yet completed |
| 500 | Internal server error |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/setup/create_admin" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "fullname": "System Administrator",
    "email": "admin@example.com",
    "password": "securepassword123"
  }'
```

---

## POST /api/v1/setup/create_profile

Creates the profile for the initial admin user. Requires both the tenant and admin setup steps to be complete.

### Authentication

None.

### Request Body

```json
{
  "first_name": "System",
  "last_name": "Administrator",
  "middle_name": null,
  "suffix": null,
  "display_name": "System Admin",
  "birthdate": "1990-01-25",
  "gender": "prefer_not_to_say",
  "bio": "Platform administrator",
  "phone": "+12025551234",
  "email": "admin@example.com",
  "address": "123 Main St",
  "city": "New York",
  "country": "US",
  "timezone": "America/New_York",
  "language": "en",
  "profile_url": "https://example.com/avatar.png",
  "metadata": {}
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `first_name` | string | Yes | 1–100 characters |
| `middle_name` | string | No | Max 100 characters |
| `last_name` | string | No | Max 100 characters |
| `suffix` | string | No | Max 50 characters |
| `display_name` | string | No | Max 100 characters |
| `birthdate` | string | No | Format: `YYYY-MM-DD` |
| `gender` | string | No | `male`, `female`, `other`, `prefer_not_to_say` |
| `bio` | string | No | Max 1000 characters |
| `phone` | string | No | Max 20 characters |
| `email` | string | No | Valid email, max 255 characters |
| `address` | string | No | Max 500 characters |
| `city` | string | No | Max 100 characters |
| `country` | string | No | ISO 3166-1 alpha-2 code (e.g., `US`, `PH`, `CA`) |
| `timezone` | string | No | Max 50 characters |
| `language` | string | No | Max 10 characters |
| `profile_url` | string | No | Valid URL, max 1000 characters |
| `metadata` | object | No | Arbitrary key-value pairs |

### Response

#### 201 Created

Returns the created profile object.

```json
{
  "success": true,
  "message": "Profile created successfully",
  "data": {
    "profile_id": "018e1b2c-bbbb-7f8a-9b0c-1d2e3f4a5b6c",
    "first_name": "System",
    "last_name": "Administrator",
    "display_name": "System Admin",
    "birthdate": null,
    "gender": "prefer_not_to_say",
    "bio": "Platform administrator",
    "phone": "+12025551234",
    "email": "admin@example.com",
    "address": "123 Main St",
    "city": "New York",
    "country": "US",
    "timezone": "America/New_York",
    "language": "en",
    "profile_url": "https://example.com/avatar.png",
    "is_default": true,
    "metadata": {},
    "created_at": "2026-01-15T10:02:00Z",
    "updated_at": "2026-01-15T10:02:00Z"
  }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or validation failure |
| 409 | Profile already exists (setup step already completed) |
| 422 | Tenant or admin setup step not yet completed |
| 500 | Internal server error |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/setup/create_profile" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "System",
    "last_name": "Administrator",
    "country": "US",
    "timezone": "America/New_York",
    "language": "en"
  }'
```
