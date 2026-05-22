# Email Verification

Two-step flow to verify a user's email address. First, request a one-time passcode (OTP) be sent to the address. Then, submit the OTP to confirm ownership.

A verification email is also sent automatically after registration when an email address is provided. This API covers explicit sends and the verify step.

**Base URL (Internal — Port 8080):** `http://localhost:8080`
**Base URL (Public — Port 8081):** `https://auth.example.com`

---

## Endpoints

| Method | Path | Auth | Port | Description |
|--------|------|------|------|-------------|
| POST | /api/v1/email-verification/send | None | 8080 | Send a verification OTP (client_id optional) |
| POST | /api/v1/email-verification/send | None | 8081 | Send a verification OTP (client_id required) |
| POST | /api/v1/email-verification/verify | None | 8080 / 8081 | Verify the OTP and mark email as confirmed |

---

## POST /api/v1/email-verification/send

Send a verification OTP to the specified email address. The OTP is valid for a limited time. This endpoint is rate-limited per email address.

### Authentication

None required.

### Differences Between Ports

| | Port 8080 (Internal) | Port 8081 (Public) |
|-|----------------------|--------------------|
| `client_id` | Optional | Required |
| `provider_id` | Optional | Required |

### Query Parameters

| Parameter | Required (8080) | Required (8081) | Description |
|-----------|-----------------|-----------------|-------------|
| client_id | No | Yes | Client ID identifying the application |
| provider_id | No | Yes | Provider ID identifying the identity provider |

### Request Body

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| email | string | Yes | Email address to verify (max 255 characters) |

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Verification email sent",
  "data": {
    "message": "Verification email sent",
    "success": true
  }
}
```

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
curl -X POST "http://localhost:8080/api/v1/email-verification/send" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com"
  }'
```

**Public (port 8081):**

```bash
curl -X POST "https://auth.example.com/api/v1/email-verification/send?client_id=my-app&provider_id=default" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com"
  }'
```

---

## POST /api/v1/email-verification/verify

Consume the OTP and mark the email address as verified. The same handler is used on both ports — the OTP is self-contained and does not require `client_id` at verification time. This endpoint is rate-limited per email address.

### Authentication

None required.

### Request Body

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| email | string | Yes | Email address being verified (max 255 characters) |
| otp | string | Yes | Verification code received by email (4–12 characters) |

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Email verified",
  "data": {
    "message": "Email verified",
    "success": true
  }
}
```

#### Error Responses

| Status | Message | Description |
|--------|---------|-------------|
| 400 | Invalid request body | Malformed JSON |
| 400 | Validation error | Missing email or OTP, or OTP outside 4–12 character range |
| 400 | Failed to verify email | Invalid or expired OTP |
| 429 | Too many requests. Please try again later. | Rate limit exceeded per email |

```json
{
  "success": false,
  "message": "Failed to verify email"
}
```

### Example

```bash
curl -X POST "https://auth.example.com/api/v1/email-verification/verify" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "otp": "847291"
  }'
```
