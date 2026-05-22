# Security Settings

Security settings control authentication policies across seven configurable sub-resources: MFA, password, session, threat detection, account lockout, registration, and token configuration. Each sub-resource is stored and updated independently. All changes are audited with the acting user's ID, IP address, and user agent.

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/security-settings/mfa` | Get MFA configuration |
| PUT | `/api/v1/security-settings/mfa` | Update MFA configuration |
| GET | `/api/v1/security-settings/password` | Get password policy |
| PUT | `/api/v1/security-settings/password` | Update password policy |
| GET | `/api/v1/security-settings/session` | Get session configuration |
| PUT | `/api/v1/security-settings/session` | Update session configuration |
| GET | `/api/v1/security-settings/threat` | Get threat detection configuration |
| PUT | `/api/v1/security-settings/threat` | Update threat detection configuration |
| GET | `/api/v1/security-settings/lockout` | Get account lockout configuration |
| PUT | `/api/v1/security-settings/lockout` | Update account lockout configuration |
| GET | `/api/v1/security-settings/registration` | Get registration configuration |
| PUT | `/api/v1/security-settings/registration` | Update registration configuration |
| GET | `/api/v1/security-settings/token` | Get token configuration |
| PUT | `/api/v1/security-settings/token` | Update token configuration |

All endpoints require: `Authorization: Bearer <token>` — Port 8080

---

## GET /api/v1/security-settings/mfa

Returns the current MFA configuration for the authenticated tenant.

### Response — 200 OK

```json
{
  "success": true,
  "message": "General config retrieved successfully",
  "data": {
    "mfa_enabled": true,
    "mfa_required": false,
    "totp_enabled": true,
    "sms_enabled": true,
    "email_enabled": true
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 500 | Internal server error |

---

## PUT /api/v1/security-settings/mfa

Updates MFA configuration for the authenticated tenant. This operation is audited.

### Request Body (application/json)

The request body is a free-form JSON object containing the MFA configuration fields to update. The object must not be empty.

| Field | Type | Description |
|-------|------|-------------|
| `mfa_enabled` | boolean | Whether MFA is enabled |
| `mfa_required` | boolean | Whether MFA is enforced for all users |
| `totp_enabled` | boolean | Whether TOTP (authenticator app) is allowed |
| `sms_enabled` | boolean | Whether SMS OTP is allowed |
| `email_enabled` | boolean | Whether email OTP is allowed |

### Response — 200 OK

```json
{
  "success": true,
  "message": "General config updated successfully",
  "data": {
    "mfa_enabled": true,
    "mfa_required": true,
    "totp_enabled": true,
    "sms_enabled": false,
    "email_enabled": true
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or empty config object |
| 401 | Tenant or user not found in context |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/security-settings/mfa" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"mfa_enabled": true, "mfa_required": true, "totp_enabled": true}'
```

---

## GET /api/v1/security-settings/password

Returns the current password policy configuration for the authenticated tenant.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Password config retrieved successfully",
  "data": {
    "min_length": 8,
    "max_length": 128,
    "require_uppercase": true,
    "require_lowercase": true,
    "require_numbers": true,
    "require_symbols": false,
    "password_expiry_days": 90,
    "password_history_count": 5
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 500 | Internal server error |

---

## PUT /api/v1/security-settings/password

Updates password policy for the authenticated tenant. This operation is audited.

### Request Body (application/json)

The request body is a free-form JSON object containing the password configuration fields to update. The object must not be empty.

| Field | Type | Description |
|-------|------|-------------|
| `min_length` | integer | Minimum password length |
| `max_length` | integer | Maximum password length |
| `require_uppercase` | boolean | Require at least one uppercase letter |
| `require_lowercase` | boolean | Require at least one lowercase letter |
| `require_numbers` | boolean | Require at least one numeric character |
| `require_symbols` | boolean | Require at least one special character |
| `password_expiry_days` | integer | Days before passwords expire (0 = no expiry) |
| `password_history_count` | integer | Number of previous passwords to prevent reuse |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Password config updated successfully",
  "data": {
    "min_length": 12,
    "max_length": 128,
    "require_uppercase": true,
    "require_lowercase": true,
    "require_numbers": true,
    "require_symbols": true,
    "password_expiry_days": 90,
    "password_history_count": 10
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or empty config object |
| 401 | Tenant or user not found in context |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/security-settings/password" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"min_length": 12, "require_symbols": true, "password_history_count": 10}'
```

---

## GET /api/v1/security-settings/session

Returns the current session management configuration for the authenticated tenant.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Session config retrieved successfully",
  "data": {
    "session_timeout_minutes": 60,
    "idle_timeout_minutes": 15,
    "max_concurrent_sessions": 5,
    "remember_me_days": 30
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 500 | Internal server error |

---

## PUT /api/v1/security-settings/session

Updates session management settings for the authenticated tenant. This operation is audited.

### Request Body (application/json)

The request body is a free-form JSON object containing the session configuration fields to update. The object must not be empty.

| Field | Type | Description |
|-------|------|-------------|
| `session_timeout_minutes` | integer | Absolute session duration before forced re-authentication |
| `idle_timeout_minutes` | integer | Idle time before session expires |
| `max_concurrent_sessions` | integer | Maximum active sessions per user (0 = unlimited) |
| `remember_me_days` | integer | Duration for persistent sessions when "remember me" is selected |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Session config updated successfully",
  "data": {
    "session_timeout_minutes": 120,
    "idle_timeout_minutes": 30,
    "max_concurrent_sessions": 3,
    "remember_me_days": 14
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or empty config object |
| 401 | Tenant or user not found in context |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/security-settings/session" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"session_timeout_minutes": 120, "idle_timeout_minutes": 30, "max_concurrent_sessions": 3}'
```

---

## GET /api/v1/security-settings/threat

Returns the current threat detection configuration for the authenticated tenant.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Threat config retrieved successfully",
  "data": {
    "brute_force_enabled": true,
    "brute_force_threshold": 10,
    "brute_force_window_minutes": 15,
    "suspicious_ip_detection": true,
    "impossible_travel_detection": false
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 500 | Internal server error |

---

## PUT /api/v1/security-settings/threat

Updates threat detection settings for the authenticated tenant. This operation is audited.

### Request Body (application/json)

The request body is a free-form JSON object containing the threat configuration fields to update. The object must not be empty.

| Field | Type | Description |
|-------|------|-------------|
| `brute_force_enabled` | boolean | Enable brute force detection |
| `brute_force_threshold` | integer | Number of failures before triggering brute force action |
| `brute_force_window_minutes` | integer | Time window for counting failures |
| `suspicious_ip_detection` | boolean | Enable detection of suspicious IP addresses |
| `impossible_travel_detection` | boolean | Enable impossible travel detection |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Threat config updated successfully",
  "data": {
    "brute_force_enabled": true,
    "brute_force_threshold": 5,
    "brute_force_window_minutes": 10,
    "suspicious_ip_detection": true,
    "impossible_travel_detection": true
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or empty config object |
| 401 | Tenant or user not found in context |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/security-settings/threat" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"brute_force_enabled": true, "brute_force_threshold": 5, "impossible_travel_detection": true}'
```

---

## GET /api/v1/security-settings/lockout

Returns the current account lockout configuration for the authenticated tenant.

### Response — 200 OK

```json
{
  "success": true,
  "message": "IP config retrieved successfully",
  "data": {
    "lockout_enabled": true,
    "max_failed_attempts": 5,
    "lockout_duration_minutes": 30,
    "progressive_lockout": false
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 500 | Internal server error |

---

## PUT /api/v1/security-settings/lockout

Updates account lockout settings for the authenticated tenant. This operation is audited.

### Request Body (application/json)

The request body is a free-form JSON object containing the lockout configuration fields to update. The object must not be empty.

| Field | Type | Description |
|-------|------|-------------|
| `lockout_enabled` | boolean | Enable account lockout after failed attempts |
| `max_failed_attempts` | integer | Number of failed attempts before lockout |
| `lockout_duration_minutes` | integer | Duration of lockout in minutes |
| `progressive_lockout` | boolean | Increase lockout duration on repeated violations |

### Response — 200 OK

```json
{
  "success": true,
  "message": "IP config updated successfully",
  "data": {
    "lockout_enabled": true,
    "max_failed_attempts": 3,
    "lockout_duration_minutes": 60,
    "progressive_lockout": true
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or empty config object |
| 401 | Tenant or user not found in context |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/security-settings/lockout" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"lockout_enabled": true, "max_failed_attempts": 3, "lockout_duration_minutes": 60}'
```

---

## GET /api/v1/security-settings/registration

Returns the current registration configuration for the authenticated tenant.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Registration config retrieved successfully",
  "data": {
    "registration_enabled": true,
    "email_verification_required": true,
    "invite_only": false,
    "allowed_domains": ["example.com"]
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 500 | Internal server error |

---

## PUT /api/v1/security-settings/registration

Updates registration settings for the authenticated tenant. This operation is audited.

### Request Body (application/json)

The request body is a free-form JSON object containing the registration configuration fields to update. The object must not be empty.

| Field | Type | Description |
|-------|------|-------------|
| `registration_enabled` | boolean | Allow new user self-registration |
| `email_verification_required` | boolean | Require email verification on sign-up |
| `invite_only` | boolean | Restrict registration to invited users only |
| `allowed_domains` | array of strings | Whitelist of email domains permitted to register |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Registration config updated successfully",
  "data": {
    "registration_enabled": true,
    "email_verification_required": true,
    "invite_only": true,
    "allowed_domains": ["corp.example.com"]
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or empty config object |
| 401 | Tenant or user not found in context |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/security-settings/registration" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"invite_only": true, "allowed_domains": ["corp.example.com"]}'
```

---

## GET /api/v1/security-settings/token

Returns the current token configuration for the authenticated tenant.

### Response — 200 OK

```json
{
  "success": true,
  "message": "Token config retrieved successfully",
  "data": {
    "access_token_ttl_seconds": 3600,
    "refresh_token_ttl_seconds": 2592000,
    "refresh_token_rotation": true,
    "jwt_algorithm": "RS256"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Tenant not found in context |
| 500 | Internal server error |

---

## PUT /api/v1/security-settings/token

Updates token settings for the authenticated tenant. This operation is audited.

### Request Body (application/json)

The request body is a free-form JSON object containing the token configuration fields to update. The object must not be empty.

| Field | Type | Description |
|-------|------|-------------|
| `access_token_ttl_seconds` | integer | Access token lifetime in seconds |
| `refresh_token_ttl_seconds` | integer | Refresh token lifetime in seconds |
| `refresh_token_rotation` | boolean | Issue a new refresh token on every refresh |
| `jwt_algorithm` | string | Signing algorithm for JWTs |

### Response — 200 OK

```json
{
  "success": true,
  "message": "Token config updated successfully",
  "data": {
    "access_token_ttl_seconds": 900,
    "refresh_token_ttl_seconds": 604800,
    "refresh_token_rotation": true,
    "jwt_algorithm": "RS256"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or empty config object |
| 401 | Tenant or user not found in context |
| 422 | Validation error |
| 500 | Internal server error |

### Example

```bash
curl -X PUT "http://localhost:8080/api/v1/security-settings/token" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"access_token_ttl_seconds": 900, "refresh_token_rotation": true}'
```
