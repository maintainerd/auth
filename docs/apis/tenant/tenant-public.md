# Tenant Public API

The Tenant Public API provides unauthenticated tenant discovery endpoints used by the identity app (login UI) to look up tenant configuration before a user authenticates. These endpoints expose only non-sensitive tenant information.

> **Port 8081.** These endpoints are served on the public-facing port and do not require authentication.

---

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/tenant` | None | Get the default (system) tenant |
| GET | `/api/v1/tenant/{identifier}` | None | Get a tenant by its identifier |

---

## Tenant Object

Both endpoints return a tenant object in this shape:

```json
{
  "tenant_id": "018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c",
  "name": "acme",
  "display_name": "Acme Corp",
  "description": "Main tenant for Acme Corp",
  "identifier": "acme",
  "status": "active",
  "is_public": true,
  "is_system": false,
  "metadata": {},
  "created_at": "2026-01-15T10:00:00Z",
  "updated_at": "2026-01-15T10:00:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `tenant_id` | UUID | Unique identifier for the tenant |
| `name` | string | URL-safe slug |
| `display_name` | string | Human-readable name |
| `description` | string | Optional description |
| `identifier` | string | System-generated identifier |
| `status` | string | `active`, `inactive`, `pending`, or `suspended` |
| `is_public` | boolean | Whether this is the active public tenant |
| `is_system` | boolean | Whether this is the protected system tenant |
| `metadata` | object | Arbitrary JSON metadata |
| `created_at` | string | ISO 8601 timestamp |
| `updated_at` | string | ISO 8601 timestamp |

---

## GET /api/v1/tenant

Returns the system (default) tenant. This is the root tenant created during initial setup.

### Authentication

None.

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "System tenant fetched successfully",
  "data": {
    "tenant_id": "018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c",
    "name": "my-org",
    "display_name": "My Organization",
    "description": "Primary organization tenant",
    "identifier": "my-org",
    "status": "active",
    "is_public": true,
    "is_system": true,
    "metadata": {
      "application_logo_url": "https://example.com/logo.png",
      "language": "en",
      "timezone": "America/New_York"
    },
    "created_at": "2026-01-15T10:00:00Z",
    "updated_at": "2026-01-15T10:00:00Z"
  }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 404 | System tenant not found (setup not complete) |
| 500 | Internal server error |

### Example

```bash
curl -X GET "https://auth.example.com/api/v1/tenant"
```

---

## GET /api/v1/tenant/{identifier}

Returns a tenant by its identifier. The identifier is the URL-safe slug assigned to the tenant, matching the `identifier` field in the tenant object.

### Authentication

None.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `identifier` | string | The tenant's identifier (e.g., `acme`) |

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Tenant fetched successfully",
  "data": {
    "tenant_id": "018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c",
    "name": "acme",
    "display_name": "Acme Corp",
    "description": "Main tenant for Acme Corp",
    "identifier": "acme",
    "status": "active",
    "is_public": false,
    "is_system": false,
    "metadata": {},
    "created_at": "2026-01-15T10:00:00Z",
    "updated_at": "2026-01-15T10:00:00Z"
  }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Identifier is empty |
| 404 | No tenant found with that identifier |

### Example

```bash
curl -X GET "https://auth.example.com/api/v1/tenant/acme"
```
