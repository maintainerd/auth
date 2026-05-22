# Branding

Branding configuration controls the visual identity presented to users across the tenant's authentication flows. This includes logos, color scheme, typography, custom CSS, and legal page URLs. The configuration is stored as a single record per tenant and is created or replaced on every PUT.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/branding` | Get branding configuration |
| PUT | `/api/v1/branding` | Create or update branding configuration |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/branding

Returns the current branding configuration for the authenticated tenant.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Branding retrieved successfully",
  "data": {
    "branding_id": "c4a1b2d3-e5f6-7890-abcd-ef1234567890",
    "company_name": "Acme Corp",
    "logo_url": "https://cdn.example.com/logo.png",
    "favicon_url": "https://cdn.example.com/favicon.ico",
    "primary_color": "#1a73e8",
    "secondary_color": "#ffffff",
    "accent_color": "#f5a623",
    "font_family": "Inter, sans-serif",
    "custom_css": "",
    "support_url": "https://support.example.com",
    "privacy_policy_url": "https://example.com/privacy",
    "terms_of_service_url": "https://example.com/terms",
    "created_at": "2025-01-10T08:00:00Z",
    "updated_at": "2025-04-20T14:30:00Z"
  }
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `branding_id` | string (UUID) | Unique identifier for this branding record |
| `company_name` | string | Company or application name, max 255 characters |
| `logo_url` | string | URL for the main logo image, max 2048 characters |
| `favicon_url` | string | URL for the browser favicon, max 2048 characters |
| `primary_color` | string | Primary brand color (CSS color value), max 20 characters |
| `secondary_color` | string | Secondary brand color (CSS color value), max 20 characters |
| `accent_color` | string | Accent color for highlights and CTAs (CSS color value), max 20 characters |
| `font_family` | string | CSS font-family value for UI text, max 100 characters |
| `custom_css` | string | Raw CSS injected into authentication pages, max 50,000 characters |
| `support_url` | string | Link to the tenant's support page, max 2048 characters |
| `privacy_policy_url` | string | Link to the privacy policy page, max 2048 characters |
| `terms_of_service_url` | string | Link to the terms of service page, max 2048 characters |
| `created_at` | string (ISO 8601) | Creation timestamp |
| `updated_at` | string (ISO 8601) | Last update timestamp |

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 404 | No branding configuration found for this tenant |
| 500 | Internal server error |

---

## PUT /api/v1/branding

Creates or replaces the branding configuration for the authenticated tenant. All fields are optional; omitted fields are cleared.

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `company_name` | string | No | Company or application name, max 255 characters |
| `logo_url` | string | No | URL for the main logo image. Must be a valid URL if provided, max 2048 characters |
| `favicon_url` | string | No | URL for the browser favicon. Must be a valid URL if provided, max 2048 characters |
| `primary_color` | string | No | Primary brand color (CSS color value), max 20 characters |
| `secondary_color` | string | No | Secondary brand color (CSS color value), max 20 characters |
| `accent_color` | string | No | Accent color (CSS color value), max 20 characters |
| `font_family` | string | No | CSS font-family string, max 100 characters |
| `custom_css` | string | No | Raw CSS injected into auth pages, max 50,000 characters |
| `support_url` | string | No | Support page URL. Must be a valid URL if provided, max 2048 characters |
| `privacy_policy_url` | string | No | Privacy policy URL. Must be a valid URL if provided, max 2048 characters |
| `terms_of_service_url` | string | No | Terms of service URL. Must be a valid URL if provided, max 2048 characters |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Branding updated successfully",
  "data": {
    "branding_id": "c4a1b2d3-e5f6-7890-abcd-ef1234567890",
    "company_name": "Acme Corp",
    "logo_url": "https://cdn.example.com/logo-v2.png",
    "favicon_url": "https://cdn.example.com/favicon.ico",
    "primary_color": "#0057b8",
    "secondary_color": "#f0f4ff",
    "accent_color": "#ff6600",
    "font_family": "Roboto, sans-serif",
    "custom_css": ".login-btn { border-radius: 8px; }",
    "support_url": "https://help.example.com",
    "privacy_policy_url": "https://example.com/privacy",
    "terms_of_service_url": "https://example.com/terms",
    "created_at": "2025-01-10T08:00:00Z",
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
curl -X PUT "http://localhost:8080/api/v1/branding" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "company_name": "Acme Corp",
    "logo_url": "https://cdn.example.com/logo.png",
    "primary_color": "#0057b8",
    "font_family": "Inter, sans-serif",
    "privacy_policy_url": "https://example.com/privacy",
    "terms_of_service_url": "https://example.com/terms"
  }'
```
