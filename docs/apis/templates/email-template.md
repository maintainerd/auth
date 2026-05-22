# Email Templates

Email templates define the HTML and plain-text content for transactional emails sent to users (e.g., verification codes, password resets, invitations). Templates are tenant-scoped. System templates (`is_system: true`) are created by the platform and cannot be deleted. Default templates (`is_default: true`) are promoted by the system and cannot be set via the create endpoint — only the system sets that flag.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/email_templates` | List email templates |
| GET | `/api/v1/email_templates/{email_template_uuid}` | Get a single template |
| POST | `/api/v1/email_templates` | Create a template |
| PUT | `/api/v1/email_templates/{email_template_uuid}` | Update a template |
| DELETE | `/api/v1/email_templates/{email_template_uuid}` | Delete a template |
| PATCH | `/api/v1/email_templates/{email_template_uuid}/status` | Update template status |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/email_templates

Returns a paginated list of email templates for the authenticated tenant. The list response omits `body_html` and `body_plain` for brevity.

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
  "message": "Email templates retrieved successfully",
  "data": {
    "rows": [
      {
        "email_template_id": "b3c4d5e6-f7a8-9012-bcde-f12345678901",
        "name": "Email Verification",
        "subject": "Verify your email address",
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

## GET /api/v1/email_templates/{email_template_uuid}

Returns a single email template with full body content.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `email_template_uuid` | string (UUID) | The template's unique identifier |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Email template retrieved successfully",
  "data": {
    "email_template_id": "b3c4d5e6-f7a8-9012-bcde-f12345678901",
    "name": "Email Verification",
    "subject": "Verify your email address",
    "body_html": "<html><body><p>Click <a href='{{verification_url}}'>here</a> to verify.</p></body></html>",
    "body_plain": "Click the following link to verify your email: {{verification_url}}",
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
| `email_template_id` | string (UUID) | Unique identifier |
| `name` | string | Template name |
| `subject` | string | Email subject line |
| `body_html` | string | HTML body content |
| `body_plain` | string or null | Plain-text fallback body |
| `status` | string | `active` or `inactive` |
| `is_default` | boolean | Whether this is the tenant's default template for its event type |
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

## POST /api/v1/email_templates

Creates a new email template. `is_default` is always set to `false` on creation.

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Template name. 1–100 characters |
| `subject` | string | Yes | Email subject line. 1–255 characters |
| `body_html` | string | Yes | HTML body content |
| `body_plain` | string | No | Plain-text fallback body |
| `status` | string | No | Initial status. One of: `active`, `inactive`. Defaults to `active` |

### Response — 201 Created

```json
{
  "success": true,
  "message": "Email template created successfully",
  "data": {
    "email_template_id": "d4e5f6a7-b8c9-0123-def0-123456789012",
    "name": "Welcome Email",
    "subject": "Welcome to our platform!",
    "body_html": "<html><body><p>Welcome, {{first_name}}!</p></body></html>",
    "body_plain": "Welcome, {{first_name}}!",
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
curl -X POST "http://localhost:8080/api/v1/email_templates" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Welcome Email",
    "subject": "Welcome to our platform!",
    "body_html": "<html><body><p>Welcome, {{first_name}}!</p></body></html>",
    "body_plain": "Welcome, {{first_name}}!"
  }'
```

---

## PUT /api/v1/email_templates/{email_template_uuid}

Updates an existing email template.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `email_template_uuid` | string (UUID) | The template's unique identifier |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Template name. 1–100 characters |
| `subject` | string | Yes | Email subject line. 1–255 characters |
| `body_html` | string | Yes | HTML body content |
| `body_plain` | string | No | Plain-text fallback body |
| `status` | string | No | Status. One of: `active`, `inactive`. Defaults to `active` if omitted |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Email template updated successfully",
  "data": {
    "email_template_id": "d4e5f6a7-b8c9-0123-def0-123456789012",
    "name": "Welcome Email",
    "subject": "Welcome to our platform!",
    "body_html": "<html><body><p>Hi {{first_name}}, welcome!</p></body></html>",
    "body_plain": "Hi {{first_name}}, welcome!",
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
curl -X PUT "http://localhost:8080/api/v1/email_templates/d4e5f6a7-b8c9-0123-def0-123456789012" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Welcome Email",
    "subject": "Welcome to our platform!",
    "body_html": "<html><body><p>Hi {{first_name}}, welcome!</p></body></html>"
  }'
```

---

## DELETE /api/v1/email_templates/{email_template_uuid}

Deletes an email template. Returns the deleted template's data.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `email_template_uuid` | string (UUID) | The template's unique identifier |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Email template deleted successfully",
  "data": {
    "email_template_id": "d4e5f6a7-b8c9-0123-def0-123456789012",
    "name": "Welcome Email",
    "subject": "Welcome to our platform!",
    "body_html": "<html><body><p>Hi {{first_name}}, welcome!</p></body></html>",
    "body_plain": null,
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
curl -X DELETE "http://localhost:8080/api/v1/email_templates/d4e5f6a7-b8c9-0123-def0-123456789012" \
  -H "Authorization: Bearer <token>"
```

---

## PATCH /api/v1/email_templates/{email_template_uuid}/status

Updates only the status of an email template.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `email_template_uuid` | string (UUID) | The template's unique identifier |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | New status. One of: `active`, `inactive` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Email template status updated successfully",
  "data": {
    "email_template_id": "d4e5f6a7-b8c9-0123-def0-123456789012",
    "name": "Welcome Email",
    "subject": "Welcome to our platform!",
    "body_html": "<html><body><p>Hi {{first_name}}, welcome!</p></body></html>",
    "body_plain": null,
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
curl -X PATCH "http://localhost:8080/api/v1/email_templates/d4e5f6a7-b8c9-0123-def0-123456789012/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "inactive"}'
```
