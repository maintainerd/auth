# Identity Provider API

The Identity Provider API manages the authentication providers configured for each tenant. A tenant can have multiple identity providers — internal (username/password), social (OAuth), or federated (external identity systems).

> **Port 8080 only.** All endpoints are available on the internal management port. Results are scoped to the tenant derived from the authenticated user's context.

---

## Endpoints

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| GET | `/api/v1/identity_providers` | `idp:read` | List identity providers for the current tenant |
| GET | `/api/v1/identity_providers/{identity_provider_uuid}` | `idp:read` | Get a single identity provider |
| POST | `/api/v1/identity_providers` | `idp:create` | Create an identity provider |
| PUT | `/api/v1/identity_providers/{identity_provider_uuid}` | `idp:update` | Update an identity provider |
| PUT | `/api/v1/identity_providers/{identity_provider_uuid}/status` | `idp:update` | Set an identity provider's status |
| DELETE | `/api/v1/identity_providers/{identity_provider_uuid}` | `idp:delete` | Delete an identity provider |

---

## Provider Values

| Value | Description |
|-------|-------------|
| `internal` | Built-in username/password authentication |
| `cognito` | AWS Cognito |
| `auth0` | Auth0 |
| `google` | Google OAuth |
| `facebook` | Facebook OAuth |
| `github` | GitHub OAuth |
| `microsoft` | Microsoft / Azure AD |
| `apple` | Sign in with Apple |
| `linkedin` | LinkedIn OAuth |
| `twitter` | Twitter / X OAuth |

## Provider Type Values

| Value | Description |
|-------|-------------|
| `identity` | Primary identity provider (handles authentication) |
| `social` | Social login provider (linked accounts) |

---

## Identity Provider Objects

### List Response (without `config` and `tenant`)

Returned by `GET /api/v1/identity_providers`.

```json
{
  "identity_provider_id": "018e1b2c-dddd-7f8a-9b0c-1d2e3f4a5b6c",
  "name": "google-social",
  "display_name": "Sign in with Google",
  "provider": "google",
  "provider_type": "social",
  "identifier": "google-social",
  "status": "active",
  "is_default": false,
  "is_system": false,
  "created_at": "2026-01-15T10:00:00Z",
  "updated_at": "2026-01-15T10:00:00Z"
}
```

### Detail Response (with `config` and `tenant`)

Returned by create, get by UUID, update, set status, and delete.

```json
{
  "identity_provider_id": "018e1b2c-dddd-7f8a-9b0c-1d2e3f4a5b6c",
  "name": "google-social",
  "display_name": "Sign in with Google",
  "provider": "google",
  "provider_type": "social",
  "identifier": "google-social",
  "config": {
    "client_id": "your-google-client-id",
    "client_secret": "your-google-client-secret",
    "scopes": ["openid", "email", "profile"]
  },
  "tenant": {
    "tenant_id": "018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c",
    "name": "acme",
    "display_name": "Acme Corp",
    "description": "Main tenant for Acme Corp",
    "identifier": "acme",
    "status": "active",
    "is_public": false,
    "created_at": "2026-01-15T10:00:00Z",
    "updated_at": "2026-01-15T10:00:00Z"
  },
  "status": "active",
  "is_default": false,
  "is_system": false,
  "created_at": "2026-01-15T10:00:00Z",
  "updated_at": "2026-01-15T10:00:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `identity_provider_id` | UUID | Unique identifier |
| `name` | string | Internal name (3–50 characters) |
| `display_name` | string | Human-readable label |
| `provider` | string | Provider key (see [Provider Values](#provider-values)) |
| `provider_type` | string | `identity` or `social` |
| `identifier` | string | System-generated slug |
| `config` | object | Provider-specific configuration (detail responses only) |
| `tenant` | object | The owning tenant (detail responses only) |
| `status` | string | `active` or `inactive` |
| `is_default` | boolean | Whether this is the tenant's default provider |
| `is_system` | boolean | Whether this was created by the system seeder |
| `created_at` | string | ISO 8601 timestamp |
| `updated_at` | string | ISO 8601 timestamp |

---

## GET /api/v1/identity_providers

Returns a paginated list of identity providers scoped to the authenticated user's tenant.

### Authentication

Bearer JWT with `idp:read` permission.

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page` | integer | Yes | Page number (minimum: 1) |
| `limit` | integer | Yes | Results per page (minimum: 1) |
| `sort_by` | string | No | Field to sort by |
| `sort_order` | string | No | `asc` or `desc` |
| `name` | string | No | Filter by name |
| `display_name` | string | No | Filter by display name |
| `identifier` | string | No | Filter by identifier |
| `provider` | string | No | Comma-separated provider values (e.g., `google,github`) |
| `provider_type` | string | No | `identity` or `social` |
| `status` | string | No | Comma-separated status values: `active`, `inactive` |
| `is_default` | boolean | No | Filter by default flag |
| `is_system` | boolean | No | Filter by system flag |

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Identity providers fetched successfully",
  "data": {
    "rows": [
      {
        "identity_provider_id": "018e1b2c-dddd-7f8a-9b0c-1d2e3f4a5b6c",
        "name": "google-social",
        "display_name": "Sign in with Google",
        "provider": "google",
        "provider_type": "social",
        "identifier": "google-social",
        "status": "active",
        "is_default": false,
        "is_system": false,
        "created_at": "2026-01-15T10:00:00Z",
        "updated_at": "2026-01-15T10:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 20,
    "total_pages": 1
  }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid query parameters |
| 401 | Missing or invalid Bearer token, or tenant not in context |
| 403 | Insufficient permissions |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/identity_providers?page=1&limit=20&provider_type=social" \
  -H "Authorization: Bearer <token>"
```

---

## GET /api/v1/identity_providers/{identity_provider_uuid}

Returns a single identity provider with full detail, including `config` and the associated `tenant` object.

### Authentication

Bearer JWT with `idp:read` permission.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `identity_provider_uuid` | UUID | The identity provider's unique identifier |

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Identity provider fetched successfully",
  "data": {
    "identity_provider_id": "018e1b2c-dddd-7f8a-9b0c-1d2e3f4a5b6c",
    "name": "google-social",
    "display_name": "Sign in with Google",
    "provider": "google",
    "provider_type": "social",
    "identifier": "google-social",
    "config": { ... },
    "tenant": { ... },
    "status": "active",
    "is_default": false,
    "is_system": false,
    "created_at": "2026-01-15T10:00:00Z",
    "updated_at": "2026-01-15T10:00:00Z"
  }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions |
| 404 | Identity provider not found |

### Example

```bash
curl -X GET "http://localhost:8080/api/v1/identity_providers/018e1b2c-dddd-7f8a-9b0c-1d2e3f4a5b6c" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/identity_providers

Creates a new identity provider for a tenant.

### Authentication

Bearer JWT with `idp:create` permission.

### Request Body

```json
{
  "name": "google-social",
  "display_name": "Sign in with Google",
  "provider": "google",
  "provider_type": "social",
  "config": {
    "client_id": "your-google-client-id",
    "client_secret": "your-google-client-secret"
  },
  "status": "active",
  "tenant_id": "018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c"
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `name` | string | Yes | 3–50 characters |
| `display_name` | string | Yes | 8–200 characters |
| `provider` | string | Yes | One of the [Provider Values](#provider-values) |
| `provider_type` | string | Yes | `identity` or `social` |
| `config` | object | Yes | Provider-specific JSON configuration |
| `status` | string | Yes | `active` or `inactive` |
| `tenant_id` | UUID | Yes | UUID of the tenant to own this provider |

### Response

#### 201 Created

Returns the created identity provider detail object (including `config` and `tenant`).

```json
{
  "success": true,
  "message": "Identity provider created successfully",
  "data": { ... }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or validation failure |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions |
| 404 | Referenced tenant not found |
| 409 | An identity provider with that name already exists for this tenant |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/identity_providers" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "google-social",
    "display_name": "Sign in with Google",
    "provider": "google",
    "provider_type": "social",
    "config": {
      "client_id": "your-google-client-id",
      "client_secret": "your-google-client-secret"
    },
    "status": "active",
    "tenant_id": "018e1b2c-3d4e-7f8a-9b0c-1d2e3f4a5b6c"
  }'
```

---

## PUT /api/v1/identity_providers/{identity_provider_uuid}

Updates an identity provider. All fields are replaced; omitting a field resets it to its default.

### Authentication

Bearer JWT with `idp:update` permission.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `identity_provider_uuid` | UUID | The identity provider's unique identifier |

### Request Body

```json
{
  "name": "google-social",
  "display_name": "Sign in with Google (Updated)",
  "provider": "google",
  "provider_type": "social",
  "config": {
    "client_id": "your-google-client-id",
    "client_secret": "new-client-secret"
  },
  "status": "active"
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `name` | string | Yes | 3–50 characters |
| `display_name` | string | Yes | 8–200 characters |
| `provider` | string | Yes | One of the [Provider Values](#provider-values) |
| `provider_type` | string | Yes | `identity` or `social` |
| `config` | object | Yes | Provider-specific JSON configuration |
| `status` | string | Yes | `active` or `inactive` |

> `tenant_id` is not accepted on update. The owning tenant cannot be changed after creation.

### Response

#### 200 OK

Returns the updated identity provider detail object.

```json
{
  "success": true,
  "message": "Identity provider updated successfully",
  "data": { ... }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID or request body |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions |
| 404 | Identity provider not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/identity_providers/018e1b2c-dddd-7f8a-9b0c-1d2e3f4a5b6c" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "google-social",
    "display_name": "Sign in with Google (Updated)",
    "provider": "google",
    "provider_type": "social",
    "config": {
      "client_id": "your-google-client-id",
      "client_secret": "new-client-secret"
    },
    "status": "active"
  }'
```

---

## PUT /api/v1/identity_providers/{identity_provider_uuid}/status

Updates only the status of an identity provider.

### Authentication

Bearer JWT with `idp:update` permission.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `identity_provider_uuid` | UUID | The identity provider's unique identifier |

### Request Body

```json
{
  "status": "inactive"
}
```

| Field | Type | Required | Allowed Values |
|-------|------|----------|----------------|
| `status` | string | Yes | `active` or `inactive` |

### Response

#### 200 OK

Returns the updated identity provider detail object.

```json
{
  "success": true,
  "message": "Identity provider status updated successfully",
  "data": { ... }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID or request body |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions |
| 404 | Identity provider not found |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/identity_providers/018e1b2c-dddd-7f8a-9b0c-1d2e3f4a5b6c/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "inactive"}'
```

---

## DELETE /api/v1/identity_providers/{identity_provider_uuid}

Deletes an identity provider. The operation is scoped to the authenticated user's tenant — providers belonging to another tenant cannot be deleted.

### Authentication

Bearer JWT with `idp:delete` permission.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `identity_provider_uuid` | UUID | The identity provider's unique identifier |

### Response

#### 200 OK

Returns the deleted identity provider detail object.

```json
{
  "success": true,
  "message": "Identity provider deleted successfully",
  "data": { ... }
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID |
| 401 | Missing or invalid Bearer token |
| 403 | Insufficient permissions |
| 404 | Identity provider not found or not owned by the current tenant |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/identity_providers/018e1b2c-dddd-7f8a-9b0c-1d2e3f4a5b6c" \
  -H "Authorization: Bearer <token>"
```
