# Email Configuration

Email configuration controls the outbound email delivery provider for the authenticated tenant. This includes SMTP server details, API credentials for managed providers, sender identity, and test mode. The configuration is stored as a single record per tenant and is created or replaced on every PUT.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/email-config` | Get email delivery configuration |
| PUT | `/api/v1/email-config` | Create or update email delivery configuration |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/email-config

Returns the current email configuration for the authenticated tenant. The `password` field is never returned in responses.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Email config retrieved successfully",
  "data": {
    "email_config_id": "3f6e4b2a-9c12-4d8e-b3f1-0a5c7d9e1234",
    "provider": "smtp",
    "host": "smtp.example.com",
    "port": 587,
    "username": "noreply@example.com",
    "from_address": "noreply@example.com",
    "from_name": "Example App",
    "reply_to": "support@example.com",
    "encryption": "tls",
    "test_mode": false,
    "status": "active",
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-05-01T08:00:00Z"
  }
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `email_config_id` | string (UUID) | Unique identifier for this configuration |
| `provider` | string | Delivery provider (`smtp`, `ses`, `sendgrid`, `mailgun`, `postmark`, `resend`) |
| `host` | string | SMTP server hostname (SMTP provider only) |
| `port` | integer | SMTP server port (SMTP provider only) |
| `username` | string | SMTP or provider username |
| `from_address` | string | Sender email address |
| `from_name` | string | Display name shown in the From header |
| `reply_to` | string | Reply-to email address |
| `encryption` | string | SMTP encryption method (`tls`, `ssl`, `none`) |
| `test_mode` | boolean | When `true`, emails are captured but not delivered |
| `status` | string | Configuration status (`active`, `inactive`) |
| `created_at` | string (ISO 8601) | Creation timestamp |
| `updated_at` | string (ISO 8601) | Last update timestamp |

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 404 | No email configuration found for this tenant |
| 500 | Internal server error |

---

## PUT /api/v1/email-config

Creates or replaces the email configuration for the authenticated tenant. All fields shown below are evaluated on every request.

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `provider` | string | Yes | Delivery provider. One of: `smtp`, `ses`, `sendgrid`, `mailgun`, `postmark`, `resend` |
| `from_address` | string | Yes | Sender email address. Must be a valid email, max 255 characters |
| `from_name` | string | No | Display name in the From header, max 255 characters |
| `reply_to` | string | No | Reply-to address. Must be a valid email if provided, max 255 characters |
| `host` | string | No | SMTP hostname. Required when `provider` is `smtp`, max 255 characters |
| `port` | integer | No | SMTP port (1–65535). Required when `provider` is `smtp` |
| `username` | string | No | SMTP or provider username, max 255 characters |
| `password` | string | No | SMTP password or provider API key (write-only, never returned) |
| `encryption` | string | No | SMTP encryption. One of: `tls`, `ssl`, `none` |
| `test_mode` | boolean | No | Set to `true` to capture emails without delivering them |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Email config updated successfully",
  "data": {
    "email_config_id": "3f6e4b2a-9c12-4d8e-b3f1-0a5c7d9e1234",
    "provider": "smtp",
    "host": "smtp.example.com",
    "port": 587,
    "username": "noreply@example.com",
    "from_address": "noreply@example.com",
    "from_name": "Example App",
    "reply_to": "support@example.com",
    "encryption": "tls",
    "test_mode": false,
    "status": "active",
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-05-22T12:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Malformed JSON request body |
| 401 | Tenant not found in context |
| 422 | Validation error — see response body for field-level details |
| 500 | Internal server error |

### Example — SMTP

```bash
curl -X PUT "http://localhost:8080/api/v1/email-config" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "smtp",
    "host": "smtp.example.com",
    "port": 587,
    "username": "noreply@example.com",
    "password": "secret",
    "from_address": "noreply@example.com",
    "from_name": "Example App",
    "encryption": "tls"
  }'
```

### Example — SendGrid

```bash
curl -X PUT "http://localhost:8080/api/v1/email-config" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "sendgrid",
    "password": "SG.your-api-key",
    "from_address": "noreply@example.com",
    "from_name": "Example App"
  }'
```
