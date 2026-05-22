# Invite

Send an invitation email to a user to join the current tenant with one or more roles assigned. This is a management endpoint available on port 8080 only.

The caller must be authenticated and must belong to the tenant they are inviting into. Tenant context is resolved by middleware from the JWT. The invited user completes registration via `POST /api/v1/register/invite`.

**Base URL (Internal — Port 8080):** `http://localhost:8080`

---

## Endpoints

| Method | Path | Auth | Port | Description |
|--------|------|------|------|-------------|
| POST | /api/v1/invite/ | Bearer JWT | 8080 | Send an invitation to a user |

---

## POST /api/v1/invite/

Send an invitation email to the specified address. The invite is automatically scoped to the tenant derived from the authenticated user's JWT. The invited user is assigned the specified roles upon registration.

### Authentication

Bearer JWT required.

```
Authorization: Bearer <access_token>
```

The token must belong to a user who is a member of the tenant. Tenant context is resolved by middleware.

### Request Body

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| email | string | Yes | Email address to invite (3–100 characters, valid email format) |
| roles | array of UUID strings | Yes | Role UUIDs to assign to the invited user (1–10 roles) |

### Response

#### 200 OK

```json
{
  "success": true,
  "message": "Invite sent successfully",
  "data": null
}
```

#### Error Responses

| Status | Message | Description |
|--------|---------|-------------|
| 400 | Invalid request payload | Malformed JSON |
| 400 | Validation error | Missing or invalid email or roles |
| 401 | Tenant not found in context | JWT missing or tenant context unresolved |
| 401 | User not found in context | JWT missing or user context unresolved |
| 400 | Failed to send invite | Invalid role UUIDs or other service error |

```json
{
  "success": false,
  "message": "Validation error"
}
```

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/invite/" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJ..." \
  -d '{
    "email": "newuser@example.com",
    "roles": [
      "3fa85f64-5717-4562-b3fc-2c963f66afa6"
    ]
  }'
```
