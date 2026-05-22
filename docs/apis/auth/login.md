# Login (Internal)

Authenticate a user and receive tokens. The internal login endpoint runs on port 8080 and does not require `client_id` or `provider_id` — those parameters are optional and can be supplied to scope the session to a specific client/provider.

**Base URL (Internal — Port 8080):** `http://localhost:8080`

---

## Endpoints

| Method | Path | Auth | Port | Description |
|--------|------|------|------|-------------|
| POST | /api/v1/login | None | 8080 | Authenticate a user |
| POST | /api/v1/logout | None | 8080 | Invalidate session and clear auth cookies |

---

## POST /api/v1/login

Authenticate a user with username and password. Returns an access token, ID token, and optional refresh token. Token delivery can be controlled with the `X-Token-Delivery` request header.

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
| 400 | Validation error | Missing or invalid fields |
| 401 | Authentication failed | Wrong username or password |

```json
{
  "success": false,
  "message": "Authentication failed"
}
```

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/login" \
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
curl -X POST "http://localhost:8080/api/v1/logout"
```
