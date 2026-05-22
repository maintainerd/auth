# Magic Link (Passwordless Login)

Two-step passwordless authentication flow. First, request a sign-in link be emailed to the user. Then, the link carries a token that is exchanged for a session.

**Base URL (Internal — Port 8080):** `http://localhost:8080`
**Base URL (Public — Port 8081):** `https://auth.example.com`

---

## Endpoints

| Method | Path | Auth | Port | Description |
|--------|------|------|------|-------------|
| POST | /api/v1/magic-link/send | None | 8080 | Send a magic link (client_id optional) |
| POST | /api/v1/magic-link/send | None | 8081 | Send a magic link (client_id required) |
| POST | /api/v1/magic-link/verify | None | 8080 / 8081 | Exchange the magic link token for a session |

---

## POST /api/v1/magic-link/send

Send a one-time sign-in link to the specified email address. The link is valid for a limited time. This endpoint is rate-limited per email address.

The internal endpoint (port 8080) uses the auth-facing frontend hostname when constructing the link. The public endpoint (port 8081) uses the account-facing frontend hostname.

### Authentication

None required.

### Differences Between Ports

| | Port 8080 (Internal) | Port 8081 (Public) |
|-|----------------------|--------------------|
| `client_id` | Optional | Required |
| `provider_id` | Optional | Required |
| Link hostname | Auth-facing frontend | Account-facing frontend |

### Query Parameters

| Parameter | Required (8080) | Required (8081) | Description |
|-----------|-----------------|-----------------|-------------|
| client_id | No | Yes | Client ID identifying the application |
| provider_id | No | Yes | Provider ID identifying the identity provider |

### Request Body

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| email | string | Yes | Email address to send the sign-in link to (max 255 characters) |

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Sign-in link sent",
  "data": {
    "message": "Sign-in link sent",
    "success": true
  }
}
```

The response is the same whether or not the email address exists, to prevent user enumeration.

#### Error Responses

| Status | Message | Description |
|--------|---------|-------------|
| 400 | Invalid request body | Malformed JSON |
| 400 | Validation error | Missing or invalid email |
| 400 | Missing required parameters: client_id and provider_id | Query params absent (port 8081 only) |
| 429 | Too many requests. Please try again later. | Rate limit exceeded per email |

```json
{
  "success": false,
  "message": "Too many requests. Please try again later."
}
```

### Examples

**Internal (port 8080):**

```bash
curl -X POST "http://localhost:8080/api/v1/magic-link/send" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com"
  }'
```

**Public (port 8081):**

```bash
curl -X POST "https://auth.example.com/api/v1/magic-link/send?client_id=my-app&provider_id=default" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com"
  }'
```

---

## POST /api/v1/magic-link/verify

Exchange a magic link token for an authenticated session. The token is embedded in the sign-in link that was emailed to the user. `client_id` and `provider_id` must be present as query parameters (they are included in the signed link).

This endpoint is rate-limited by client IP address.

### Authentication

None required.

### Query Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| client_id | Yes | Client ID (carried in the signed link) |
| provider_id | Yes | Provider ID (carried in the signed link) |

### Request Body

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| token | string | Yes | Magic link token from the email (16–256 characters) |

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Signed in",
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
| 400 | Invalid request body | Malformed JSON |
| 400 | Validation error | Token missing or outside 16–256 character range |
| 400 | Missing required parameters: client_id and provider_id | Query params absent |
| 400 | Failed to sign in | Invalid or expired token |
| 429 | Too many requests. Please try again later. | Rate limit exceeded by IP |

```json
{
  "success": false,
  "message": "Failed to sign in"
}
```

### Example

The magic link from the email looks like:

```
https://auth.example.com/api/v1/magic-link/verify
  ?client_id=my-app
  &provider_id=default
```

The front-end extracts the token from the URL hash or path and POSTs it:

```bash
curl -X POST "https://auth.example.com/api/v1/magic-link/verify?client_id=my-app&provider_id=default" \
  -H "Content-Type: application/json" \
  -d '{
    "token": "ml_abc123def456..."
  }'
```
