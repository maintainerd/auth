# SMS Configuration

SMS configuration controls the outbound SMS delivery provider for the authenticated tenant. This includes provider selection, account credentials, sender identity, and test mode. The configuration is stored as a single record per tenant and is created or replaced on every PUT. The `auth_token` field is write-only and never returned in responses.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/sms-config` | Get SMS delivery configuration |
| PUT | `/api/v1/sms-config` | Create or update SMS delivery configuration |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/sms-config

Returns the current SMS configuration for the authenticated tenant.

### Response — 200 OK

```json
{
  "success": true,
  "message": "SMS config retrieved successfully",
  "data": {
    "sms_config_id": "7a2f1d4c-3b8e-4a9f-c2d5-6e0b1f3a5678",
    "provider": "twilio",
    "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "from_number": "+15551234567",
    "sender_id": null,
    "test_mode": false,
    "status": "active",
    "created_at": "2025-02-10T09:00:00Z",
    "updated_at": "2025-05-01T11:00:00Z"
  }
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `sms_config_id` | string (UUID) | Unique identifier for this configuration |
| `provider` | string | SMS provider (`twilio`, `sns`, `vonage`, `messagebird`) |
| `account_sid` | string | Provider account SID or account identifier |
| `from_number` | string | Sender phone number in E.164 format |
| `sender_id` | string | Alphanumeric sender ID (where supported by provider) |
| `test_mode` | boolean | When `true`, SMS messages are captured but not sent |
| `status` | string | Configuration status (`active`, `inactive`) |
| `created_at` | string (ISO 8601) | Creation timestamp |
| `updated_at` | string (ISO 8601) | Last update timestamp |

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 404 | No SMS configuration found for this tenant |
| 500 | Internal server error |

---

## PUT /api/v1/sms-config

Creates or replaces the SMS configuration for the authenticated tenant.

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `provider` | string | Yes | SMS provider. One of: `twilio`, `sns`, `vonage`, `messagebird` |
| `account_sid` | string | No | Provider account SID or account identifier, max 255 characters |
| `auth_token` | string | No | Provider auth token or secret key (write-only, never returned) |
| `from_number` | string | No | Sender phone number in E.164 format, max 50 characters |
| `sender_id` | string | No | Alphanumeric sender ID (where supported by provider), max 50 characters |
| `test_mode` | boolean | No | Set to `true` to capture messages without sending them |

### Response — 200 OK

```json
{
  "success": true,
  "message": "SMS config updated successfully",
  "data": {
    "sms_config_id": "7a2f1d4c-3b8e-4a9f-c2d5-6e0b1f3a5678",
    "provider": "twilio",
    "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "from_number": "+15551234567",
    "sender_id": null,
    "test_mode": false,
    "status": "active",
    "created_at": "2025-02-10T09:00:00Z",
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

### Example — Twilio

```bash
curl -X PUT "http://localhost:8080/api/v1/sms-config" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "twilio",
    "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "auth_token": "your_auth_token",
    "from_number": "+15551234567"
  }'
```

### Example — Vonage with Sender ID

```bash
curl -X PUT "http://localhost:8080/api/v1/sms-config" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "vonage",
    "account_sid": "your_api_key",
    "auth_token": "your_api_secret",
    "sender_id": "MyApp"
  }'
```
