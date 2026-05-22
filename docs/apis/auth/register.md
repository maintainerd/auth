# Register (Internal)

Create a new user account through the internal endpoint on port 8080. `client_id` and `provider_id` are optional. Supports both direct registration and invite-based registration.

When a new user registers with an email address, a verification email is automatically sent in the background (failures do not block registration).

**Base URL (Internal — Port 8080):** `http://localhost:8080`

---

## Endpoints

| Method | Path | Auth | Port | Description |
|--------|------|------|------|-------------|
| POST | /api/v1/register | None | 8080 | Register a new user |
| POST | /api/v1/register/invite | None | 8080 | Complete registration via invite token |

---

## POST /api/v1/register

Create a new user account. Password must meet strength requirements (uppercase, lowercase, digit, special character, minimum 8 characters).

### Authentication

None required.

### Query Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| client_id | No | Client ID to scope the session |
| provider_id | No | Provider ID to scope the session |

### Request Body

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| username | string | Yes | Username (max 255 characters) |
| fullname | string | Yes | Full name (max 255 characters) |
| password | string | Yes | Password (8–128 characters, must include uppercase, lowercase, digit, and special character) |
| email | string | No | Email address |
| phone | string | No | Phone number |

### Response

#### 201 Created

```json
{
  "success": true,
  "message": "Registration successful",
  "data": {
    "access_token": "eyJ...",
    "id_token": "eyJ...",
    "refresh_token": "eyJ...",
    "expires_in": 900,
    "token_type": "Bearer",
    "issued_at": 1716000000
  }
}
```

`refresh_token` is omitted when not issued.

#### Error Responses

| Status | Message | Description |
|--------|---------|-------------|
| 400 | Invalid request format | Malformed JSON body |
| 400 | Validation error | Missing or invalid fields |
| 400 | Validation error | Password too weak or does not meet strength requirements |
| 409 | Registration failed | Username or email already exists |

```json
{
  "success": false,
  "message": "Registration failed"
}
```

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "jdoe",
    "fullname": "Jane Doe",
    "password": "Secure@Pass1",
    "email": "jane@example.com"
  }'
```

---

## POST /api/v1/register/invite

Complete registration using an invite token. The invite token is passed as a query parameter. The request body uses the same shape as a standard login (`username` + `password`) rather than the full registration form.

### Authentication

None required.

### Query Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| invite_token | Yes | Invite token received via email |
| client_id | No | Client ID to scope the session |
| provider_id | No | Provider ID to scope the session |

### Request Body

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| username | string | Yes | Username to claim (max 255 characters) |
| password | string | Yes | Password to set (max 128 characters) |

### Response

#### 201 Created

```json
{
  "success": true,
  "message": "Registration successful",
  "data": {
    "access_token": "eyJ...",
    "id_token": "eyJ...",
    "refresh_token": "eyJ...",
    "expires_in": 900,
    "token_type": "Bearer",
    "issued_at": 1716000000
  }
}
```

#### Error Responses

| Status | Message | Description |
|--------|---------|-------------|
| 400 | Invite token is required | `invite_token` query parameter missing |
| 400 | Invalid request | Malformed JSON body |
| 400 | Validation error | Missing or invalid fields |
| 400 | Registration failed | Invalid or expired invite token |

```json
{
  "success": false,
  "message": "Registration failed"
}
```

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/register/invite?invite_token=abc123token" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "jdoe",
    "password": "Secure@Pass1"
  }'
```
