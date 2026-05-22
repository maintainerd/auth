# Login Templates

Login templates control the visual presentation of authentication pages for the tenant. Each template uses a named style (`modern`, `classic`, `minimal`, `corporate`, `creative`, or `custom`) and supports an optional metadata object for style-specific configuration. System templates (`is_system: true`) are created by the platform and cannot be deleted. The list endpoint omits `metadata` for brevity.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/login_templates` | List login templates |
| GET | `/api/v1/login_templates/{login_template_uuid}` | Get a single template |
| POST | `/api/v1/login_templates` | Create a template |
| PUT | `/api/v1/login_templates/{login_template_uuid}` | Update a template |
| DELETE | `/api/v1/login_templates/{login_template_uuid}` | Delete a template |
| PATCH | `/api/v1/login_templates/{login_template_uuid}/status` | Update template status |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/login_templates

Returns a paginated list of login templates for the authenticated tenant. The `metadata` object is not included in list results.

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | string | Filter by template name (substring match) |
| `template` | string | Filter by template style. One of: `modern`, `classic`, `minimal`, `corporate`, `creative`, `custom` |
| `status` | string | Filter by status. One of: `active`, `inactive` (comma-separated for multiple) |
| `is_default` | boolean | Filter for default templates (`true` or `false`) |
| `is_system` | boolean | Filter for system templates (`true` or `false`) |
| `page` | integer | Page number (required, minimum 1, defaults to 1) |
| `limit` | integer | Results per page (required, minimum 1, defaults to 10) |
| `sort_by` | string | Field to sort by, max 50 characters |
| `sort_order` | string | Sort direction. One of: `asc`, `desc` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Login templates retrieved successfully",
  "data": {
    "rows": [
      {
        "login_template_id": "a7b8c9d0-e1f2-3456-a012-3456789abcde",
        "name": "Default Login Page",
        "description": "Standard login page with modern styling",
        "template": "modern",
        "status": "active",
        "is_default": true,
        "is_system": true,
        "created_at": "2025-01-01T00:00:00Z",
        "updated_at": "2025-01-01T00:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 10,
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

## GET /api/v1/login_templates/{login_template_uuid}

Returns a single login template with full `metadata`.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `login_template_uuid` | string (UUID) | The template's unique identifier |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Login template retrieved successfully",
  "data": {
    "login_template_id": "a7b8c9d0-e1f2-3456-a012-3456789abcde",
    "name": "Default Login Page",
    "description": "Standard login page with modern styling",
    "template": "modern",
    "status": "active",
    "metadata": {
      "background_image": "https://cdn.example.com/bg.jpg",
      "show_logo": true,
      "card_style": "elevated"
    },
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
| `login_template_id` | string (UUID) | Unique identifier |
| `name` | string | Template name |
| `description` | string or null | Optional description |
| `template` | string | Template style: `modern`, `classic`, `minimal`, `corporate`, `creative`, or `custom` |
| `status` | string | `active` or `inactive` |
| `metadata` | object | Style-specific configuration key-value pairs |
| `is_default` | boolean | Whether this is the active default template for the tenant |
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

## POST /api/v1/login_templates

Creates a new login template.

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Template name. 1–100 characters |
| `template` | string | Yes | Template style. One of: `modern`, `classic`, `minimal`, `corporate`, `creative`, `custom` |
| `description` | string | No | Optional description |
| `metadata` | object | No | Style-specific configuration key-value pairs. Defaults to `{}` |
| `status` | string | No | Initial status. One of: `active`, `inactive`. Defaults to `active` |

### Response — 201 Created

```json
{
  "success": true,
  "message": "Login template created successfully",
  "data": {
    "login_template_id": "b8c9d0e1-f2a3-4567-b012-456789abcdef",
    "name": "Corporate Login",
    "description": "Branded login for corporate users",
    "template": "corporate",
    "status": "active",
    "metadata": {
      "show_company_logo": true
    },
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
curl -X POST "http://localhost:8080/api/v1/login_templates" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Corporate Login",
    "template": "corporate",
    "description": "Branded login for corporate users",
    "metadata": {"show_company_logo": true}
  }'
```

---

## PUT /api/v1/login_templates/{login_template_uuid}

Updates an existing login template.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `login_template_uuid` | string (UUID) | The template's unique identifier |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Template name. 1–100 characters |
| `template` | string | Yes | Template style. One of: `modern`, `classic`, `minimal`, `corporate`, `creative`, `custom` |
| `description` | string | No | Optional description |
| `metadata` | object | No | Style-specific configuration key-value pairs. Defaults to `{}` if omitted |
| `status` | string | No | Status. One of: `active`, `inactive`. Defaults to `active` if omitted |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Login template updated successfully",
  "data": {
    "login_template_id": "b8c9d0e1-f2a3-4567-b012-456789abcdef",
    "name": "Corporate Login",
    "description": "Updated description",
    "template": "corporate",
    "status": "active",
    "metadata": {
      "show_company_logo": true,
      "background_color": "#001f5b"
    },
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
curl -X PUT "http://localhost:8080/api/v1/login_templates/b8c9d0e1-f2a3-4567-b012-456789abcdef" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Corporate Login",
    "template": "corporate",
    "metadata": {"show_company_logo": true, "background_color": "#001f5b"}
  }'
```

---

## DELETE /api/v1/login_templates/{login_template_uuid}

Deletes a login template. Returns the deleted template's data.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `login_template_uuid` | string (UUID) | The template's unique identifier |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Login template deleted successfully",
  "data": {
    "login_template_id": "b8c9d0e1-f2a3-4567-b012-456789abcdef",
    "name": "Corporate Login",
    "description": "Branded login for corporate users",
    "template": "corporate",
    "status": "inactive",
    "metadata": {},
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
curl -X DELETE "http://localhost:8080/api/v1/login_templates/b8c9d0e1-f2a3-4567-b012-456789abcdef" \
  -H "Authorization: Bearer <token>"
```

---

## PATCH /api/v1/login_templates/{login_template_uuid}/status

Updates only the status of a login template.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `login_template_uuid` | string (UUID) | The template's unique identifier |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | New status. One of: `active`, `inactive` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Login template status updated successfully",
  "data": {
    "login_template_id": "b8c9d0e1-f2a3-4567-b012-456789abcdef",
    "name": "Corporate Login",
    "description": "Branded login for corporate users",
    "template": "corporate",
    "status": "inactive",
    "metadata": {},
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
curl -X PATCH "http://localhost:8080/api/v1/login_templates/b8c9d0e1-f2a3-4567-b012-456789abcdef/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "inactive"}'
```
