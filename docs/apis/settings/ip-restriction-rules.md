# IP Restriction Rules

IP restriction rules define allow and deny policies for inbound traffic based on IPv4 address. Each rule targets a single IP address and can be classified as `allow`, `deny`, `whitelist`, or `blacklist`. Rules are tenant-scoped and evaluated at the authentication layer.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/ip-restriction-rules` | List IP restriction rules |
| GET | `/api/v1/ip-restriction-rules/{ip_restriction_rule_uuid}` | Get a single rule |
| POST | `/api/v1/ip-restriction-rules` | Create a rule |
| PUT | `/api/v1/ip-restriction-rules/{ip_restriction_rule_uuid}` | Update a rule |
| DELETE | `/api/v1/ip-restriction-rules/{ip_restriction_rule_uuid}` | Delete a rule |
| PATCH | `/api/v1/ip-restriction-rules/{ip_restriction_rule_uuid}/status` | Update rule status |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/ip-restriction-rules

Returns a paginated list of IP restriction rules for the authenticated tenant, with optional filters.

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `type` | string | Filter by rule type. One of: `allow`, `deny`, `whitelist`, `blacklist` |
| `status` | string | Filter by status. One of: `active`, `inactive` |
| `ip_address` | string | Filter by exact IP address |
| `description` | string | Filter by description (substring match) |
| `page` | integer | Page number (required, minimum 1) |
| `limit` | integer | Results per page (required, minimum 1) |
| `sort_by` | string | Field to sort by, max 50 characters |
| `sort_order` | string | Sort direction. One of: `asc`, `desc` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "IP restriction rules retrieved successfully",
  "data": {
    "rows": [
      {
        "ip_restriction_rule_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "description": "Block known malicious IP",
        "type": "deny",
        "ip_address": "192.0.2.100",
        "status": "active",
        "created_at": "2025-03-01T10:00:00Z",
        "updated_at": "2025-03-01T10:00:00Z"
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

## GET /api/v1/ip-restriction-rules/{ip_restriction_rule_uuid}

Returns a single IP restriction rule by UUID.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ip_restriction_rule_uuid` | string (UUID) | The rule's unique identifier |

### Response — 200 OK

```json
{
  "success": true,
  "message": "IP restriction rule retrieved successfully",
  "data": {
    "ip_restriction_rule_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "description": "Block known malicious IP",
    "type": "deny",
    "ip_address": "192.0.2.100",
    "status": "active",
    "created_at": "2025-03-01T10:00:00Z",
    "updated_at": "2025-03-01T10:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 401 | Tenant not found in context |
| 404 | Rule not found or does not belong to this tenant |
| 500 | Internal server error |

---

## POST /api/v1/ip-restriction-rules

Creates a new IP restriction rule for the authenticated tenant.

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Rule type. One of: `allow`, `deny`, `whitelist`, `blacklist` |
| `ip_address` | string | Yes | IPv4 address to match. Must be a valid IPv4 address, max 50 characters |
| `description` | string | No | Human-readable description of the rule, max 500 characters |
| `status` | string | No | Initial status. One of: `active`, `inactive`. Defaults to `active` |

### Response — 201 Created

```json
{
  "success": true,
  "message": "IP restriction rule created successfully",
  "data": {
    "ip_restriction_rule_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "description": "Block known malicious IP",
    "type": "deny",
    "ip_address": "192.0.2.100",
    "status": "active",
    "created_at": "2025-05-22T12:00:00Z",
    "updated_at": "2025-05-22T12:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Malformed JSON request body |
| 401 | Tenant or user not found in context |
| 422 | Validation error — see response body for field-level details |
| 500 | Internal server error |

### Example

```bash
curl -X POST "http://localhost:8080/api/v1/ip-restriction-rules" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "deny",
    "ip_address": "192.0.2.100",
    "description": "Block known malicious IP"
  }'
```

---

## PUT /api/v1/ip-restriction-rules/{ip_restriction_rule_uuid}

Updates an existing IP restriction rule. All fields are replaced.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ip_restriction_rule_uuid` | string (UUID) | The rule's unique identifier |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Rule type. One of: `allow`, `deny`, `whitelist`, `blacklist` |
| `ip_address` | string | Yes | IPv4 address to match. Must be a valid IPv4 address, max 50 characters |
| `description` | string | No | Human-readable description, max 500 characters |
| `status` | string | No | Status. One of: `active`, `inactive`. Defaults to `active` if omitted |

### Response — 200 OK

```json
{
  "success": true,
  "message": "IP restriction rule updated successfully",
  "data": {
    "ip_restriction_rule_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "description": "Updated description",
    "type": "allow",
    "ip_address": "192.0.2.100",
    "status": "active",
    "created_at": "2025-03-01T10:00:00Z",
    "updated_at": "2025-05-22T12:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format or malformed request body |
| 401 | Tenant or user not found in context |
| 404 | Rule not found or does not belong to this tenant |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/ip-restriction-rules/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "allow",
    "ip_address": "192.0.2.100",
    "description": "Updated description"
  }'
```

---

## DELETE /api/v1/ip-restriction-rules/{ip_restriction_rule_uuid}

Deletes an IP restriction rule. Returns the deleted rule's data.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ip_restriction_rule_uuid` | string (UUID) | The rule's unique identifier |

### Response — 200 OK

```json
{
  "success": true,
  "message": "IP restriction rule deleted successfully",
  "data": {
    "ip_restriction_rule_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "description": "Block known malicious IP",
    "type": "deny",
    "ip_address": "192.0.2.100",
    "status": "inactive",
    "created_at": "2025-03-01T10:00:00Z",
    "updated_at": "2025-05-22T12:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format |
| 401 | Tenant not found in context |
| 404 | Rule not found or does not belong to this tenant |
| 500 | Internal server error |

### Example

```bash
curl -X DELETE "http://localhost:8080/api/v1/ip-restriction-rules/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer <token>"
```

---

## PATCH /api/v1/ip-restriction-rules/{ip_restriction_rule_uuid}/status

Updates only the status of an IP restriction rule.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ip_restriction_rule_uuid` | string (UUID) | The rule's unique identifier |

### Request Body (application/json)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | New status. One of: `active`, `inactive` |

### Response — 200 OK

```json
{
  "success": true,
  "message": "IP restriction rule status updated successfully",
  "data": {
    "ip_restriction_rule_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "description": "Block known malicious IP",
    "type": "deny",
    "ip_address": "192.0.2.100",
    "status": "inactive",
    "created_at": "2025-03-01T10:00:00Z",
    "updated_at": "2025-05-22T12:00:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid UUID format or malformed request body |
| 401 | Tenant or user not found in context |
| 404 | Rule not found or does not belong to this tenant |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PATCH "http://localhost:8080/api/v1/ip-restriction-rules/a1b2c3d4-e5f6-7890-abcd-ef1234567890/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "inactive"}'
```
