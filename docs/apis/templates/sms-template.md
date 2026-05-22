# SMS Templates

SMS templates define the message content for outbound text messages sent to users (e.g., OTP codes, notifications). Templates are tenant-scoped. System templates (`is_system: true`) are created by the platform and cannot be deleted. The list endpoint omits the `message` field for brevity; use the single-resource endpoint to retrieve full content.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/sms_templates` | List SMS templates |
| GET | `/api/v1/sms_templates/{sms_template_uuid}` | Get a single template |
| POST | `/api/v1/sms_templates` | Create a template |
| PUT | `/api/v1/sms_templates/{sms_template_uuid}` | Update a template |
| DELETE | `/api/v1/sms_templates/{sms_template_uuid}` | Delete a template |
| PATCH | `/api/v1/sms_templates/{sms_template_uuid}/status` | Update template status |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/sms_templates

Returns a paginated list of SMS templates for the authenticated tenant. The `message` field is not included in list results.

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | string | Filter by template name (substring match) |
| `status` | string | Filter by status. One of: `active`, `inactive` |
| `is_default` | boolean | Filter for default templates (`true` or `false`) |
| `is_system` | boolean | Filter for system templates (`true` or `false`) |
| `page` | integer | Page number (required, minimum 1) |
| `limit` | integer | Results per page (required, minimum 1) |
| `sort_by` | string | Field to sort by, max 50 characters |
| `sort_order` | string | Sort direction. One of: `asc`, `desc` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "SMS templates retrieved successfully",
  "data": {
    "rows": [
      {
        "sms_template_id": "e5f6a7b8-c9d0-1234-ef01-234567890123",
        "name": "OTP Verification",
        "description": "One-time password for login",
        "sender_id": "MyApp",
        "status": "active",
        "is_default": true,
        "is_system": true,
        "created_at": "2025-01-01T00:00:00Z",
        "updated_at": "2025-01-01T00:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 20,
    "total_pages": 1
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 422 | Invalid filter parameters |
| 500 | Internal server error |

---

## GET /api/v1/sms_templates/{sms_template_uuid}

Returns a single SMS template with full message content.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `sms_template_uuid` | string (UUID) | The template's unique identifier |

### Response — 200 OK

```json
{
  "success": true,
  "message": "SMS template retrieved successfully",
  "data": {
    "sms_template_id": "e5f6a7b8-c9d0-1234-ef01-234567890123",
    "name": "OTP Verification",
    "description": "One-time password for login",
    "message": "Your {{app_name}} verification code is {{otp_code}}. It expires in {{expiry_minutes}} minutes.",
    "sender_id": "MyApp",
    "status": "active",
    "is_default": true,
    "is_system": true,
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

### Response Fields (detail view)

| Field | Type | Description |
|-------|------|-------------|
| `sms_template_id` | string (UUID) | Unique identifier |
| `name` | string | Template name |
| `description` | string or null | Optional description |
| `message` | string | SMS message body (supports template variables) |
| `sender_id` | string or null | Alphanumeric sender ID override for this template |
| `status` | string | `active` or `inactive` |
| `is_default` | boolean | Whether this is the tenant's default for its event type |
| `is_system` | boolean | Whether this template was created by the platform |
| `created_at` | string (ISO 8601) | Creation timestamp |
| `updated_at` | string (ISO 8601) | Last update timestamp |

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 401 | Tenant not found in context |
| 404 | Template not found or does not belong to this tenant |
| 500 | Internal server error |

---

## POST /api/v1/sms_templates

Creates a new SMS template.

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Template name. 1–100 characters |
| `message` | string | Yes | SMS message body |
| `description` | string | No | Optional description |
| `sender_id` | string | No | Alphanumeric sender ID override, max 20 characters |
| `status` | string | No | Initial status. One of: `active`, `inactive`. Defaults to `active` |

### Response — 201 Created

```json
{
  "success": true,
  "message": "SMS template created successfully",
  "data": {
    "sms_template_id": "f6a7b8c9-d0e1-2345-f012-345678901234",
    "name": "Password Reset OTP",
    "description": "OTP for password reset flow",
    "message": "Your {{app_name}} password reset code is {{otp_code}}.",
    "sender_id": "MyApp",
    "status": "active",
    "is_default": false,
    "is_system": false,
    "created_at": "2025-05-22T12:00:00Z",
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

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/sms_templates" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Password Reset OTP",
    "message": "Your {{app_name}} password reset code is {{otp_code}}.",
    "description": "OTP for password reset flow",
    "sender_id": "MyApp"
  }'
```

---

## PUT /api/v1/sms_templates/{sms_template_uuid}

Updates an existing SMS template.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `sms_template_uuid` | string (UUID) | The template's unique identifier |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Template name. 1–100 characters |
| `message` | string | Yes | SMS message body |
| `description` | string | No | Optional description |
| `sender_id` | string | No | Alphanumeric sender ID override, max 20 characters |
| `status` | string | No | Status. One of: `active`, `inactive`. Defaults to `active` if omitted |

### Response — 200 OK

```json
{
  "success": true,
  "message": "SMS template updated successfully",
  "data": {
    "sms_template_id": "f6a7b8c9-d0e1-2345-f012-345678901234",
    "name": "Password Reset OTP",
    "description": "Updated description",
    "message": "Your {{app_name}} reset code is {{otp_code}}. Expires in {{expiry_minutes}} minutes.",
    "sender_id": "MyApp",
    "status": "active",
    "is_default": false,
    "is_system": false,
    "created_at": "2025-05-22T12:00:00Z",
    "updated_at": "2025-05-22T13:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format or malformed request body |
| 401 | Tenant not found in context |
| 404 | Template not found or does not belong to this tenant |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/sms_templates/f6a7b8c9-d0e1-2345-f012-345678901234" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Password Reset OTP",
    "message": "Your {{app_name}} reset code is {{otp_code}}. Expires in {{expiry_minutes}} minutes."
  }'
```

---

## DELETE /api/v1/sms_templates/{sms_template_uuid}

Deletes an SMS template. Returns the deleted template's data.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `sms_template_uuid` | string (UUID) | The template's unique identifier |

### Response — 200 OK

```json
{
  "success": true,
  "message": "SMS template deleted successfully",
  "data": {
    "sms_template_id": "f6a7b8c9-d0e1-2345-f012-345678901234",
    "name": "Password Reset OTP",
    "description": "OTP for password reset flow",
    "message": "Your {{app_name}} reset code is {{otp_code}}.",
    "sender_id": "MyApp",
    "status": "inactive",
    "is_default": false,
    "is_system": false,
    "created_at": "2025-05-22T12:00:00Z",
    "updated_at": "2025-05-22T14:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 401 | Tenant not found in context |
| 404 | Template not found or does not belong to this tenant |
| 500 | Internal server error |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/sms_templates/f6a7b8c9-d0e1-2345-f012-345678901234" \
  -H "Authorization: Bearer <token>"
```

---

## PATCH /api/v1/sms_templates/{sms_template_uuid}/status

Updates only the status of an SMS template.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `sms_template_uuid` | string (UUID) | The template's unique identifier |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | New status. One of: `active`, `inactive` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "SMS template status updated successfully",
  "data": {
    "sms_template_id": "f6a7b8c9-d0e1-2345-f012-345678901234",
    "name": "Password Reset OTP",
    "description": "OTP for password reset flow",
    "message": "Your {{app_name}} reset code is {{otp_code}}.",
    "sender_id": "MyApp",
    "status": "inactive",
    "is_default": false,
    "is_system": false,
    "created_at": "2025-05-22T12:00:00Z",
    "updated_at": "2025-05-22T15:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format or malformed request body |
| 401 | Tenant not found in context |
| 404 | Template not found or does not belong to this tenant |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PATCH "http://localhost:8080/api/v1/sms_templates/f6a7b8c9-d0e1-2345-f012-345678901234/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "inactive"}'
```
