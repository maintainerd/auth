# Forgot Password

Initiate a password reset by sending a reset link to the user's email address. Available on both ports with different `client_id`/`provider_id` requirements.

**Base URL (Internal — Port 8080):** `http://localhost:8080`
**Base URL (Public — Port 8081):** `https://auth.example.com`

---

## Endpoints

| Method | Path | Auth | Port | Description |
|--------|------|------|------|-------------|
| POST | /api/v1/forgot-password | None | 8080 | Request a password reset link (client_id optional) |
| POST | /api/v1/forgot-password | None | 8081 | Request a password reset link (client_id required) |

---

## POST /api/v1/forgot-password

Send a password reset email to the specified address. The email contains a signed link that the user follows to set a new password.

The response is deliberately generic — the same success message is returned regardless of whether the email exists. This prevents email enumeration.

### Authentication

None required.

### Differences Between Ports

| | Port 8080 (Internal) | Port 8081 (Public) |
|-|----------------------|--------------------|
| `client_id` | Optional | Required |
| `provider_id` | Optional | Required |
| User-Agent validation | No | Yes |

### Query Parameters

| Parameter | Required (8080) | Required (8081) | Description |
|-----------|-----------------|-----------------|-------------|
| client_id | No | Yes | Client ID identifying the application |
| provider_id | No | Yes | Provider ID identifying the identity provider |

### Request Body

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| email | string | Yes | Email address of the account to reset |

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Check your email",
  "data": {
    "message": "Check your email",
    "success": true
  }
}
```

#### Error Responses

| Status | Message | Description |
|--------|---------|-------------|
| 400 | Invalid request body | Malformed JSON |
| 400 | Validation error | Missing or invalid email |
| 400 | Invalid request | Suspicious User-Agent (port 8081 only) |
| 400 | Missing required parameters: client_id and provider_id | Query params absent (port 8081 only) |
| 429 | Too many requests. Please try again later. | Rate limit exceeded |

```json
{
  "success": false,
  "message": "Too many requests. Please try again later."
}
```

### Examples

**Internal (port 8080):**

```bash
curl -X POST "http://localhost:8080/api/v1/forgot-password" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com"
  }'
```

**Public (port 8081):**

```bash
curl -X POST "https://auth.example.com/api/v1/forgot-password?client_id=my-app&provider_id=default" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com"
  }'
```
