# Login (Public)

Authenticate a user through the public-facing endpoint. The public login endpoint runs on port 8081 and requires `client_id` and `provider_id` as query parameters to identify the application and identity provider.

**Base URL (Public — Port 8081):** `https://auth.example.com`

---

## Endpoints

| Method | Path | Auth | Port | Description |
|--------|------|------|------|-------------|
| POST | /api/v1/login | None | 8081 | Authenticate a user (requires client_id + provider_id) |
| POST | /api/v1/logout | None | 8081 | Invalidate session and clear auth cookies |

---

## POST /api/v1/login

Authenticate a user with username and password scoped to a specific client and provider. Returns an access token, ID token, and optional refresh token.

### Authentication

None required.

### Query Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| client_id | Yes | Client ID identifying the application |
| provider_id | Yes | Provider ID identifying the identity provider |

### Request Body

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| username | string | Yes | Username (max 255 characters) |
| password | string | Yes | Password (max 128 characters) |

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Login successful",
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
| 400 | Invalid request | Suspicious or missing User-Agent |
| 400 | Validation error | Missing or invalid fields |
| 400 | Missing required parameters: client_id and provider_id | Query parameters absent or blank |
| 401 | Authentication failed | Wrong username or password |

```json
{
  "success": false,
  "message": "Authentication failed"
}
```

### Example

```bash
curl -X POST "https://auth.example.com/api/v1/login?client_id=my-app&provider_id=default" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "user@example.com",
    "password": "your-password"
  }'
```

---

## POST /api/v1/logout

Clear authentication cookies and invalidate the current session.

### Authentication

None required.

### Request Body

None.

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Logout successful",
  "data": null
}
```

### Example

```bash
curl -X POST "https://auth.example.com/api/v1/logout"
```
