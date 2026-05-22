# User Settings

User settings store per-user preferences including localization, communication consent, privacy choices, social links, and emergency contact information. Each user has at most one settings record; the endpoints use upsert semantics.

These endpoints are available on both the management port (8080) and the auth port (8081). All require a valid Bearer JWT.

---

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/user-settings` | Bearer JWT | Get the authenticated user's settings |
| POST | `/api/v1/user-settings` | Bearer JWT | Create or update the authenticated user's settings |
| DELETE | `/api/v1/user-settings` | Bearer JWT | Delete the authenticated user's settings |

---

## GET /api/v1/user-settings

Returns the authenticated user's current settings record.

### Authentication

Bearer JWT required.

### Response — 200 OK

```json
{
  "success": true,
  "message": "User setting retrieved successfully",
  "data": {
    "user_setting_id": "d4e5f6a7-b8c9-0123-def0-234567890123",
    "timezone": "America/Los_Angeles",
    "preferred_language": "en",
    "locale": "en-US",
    "social_links": {
      "twitter": "https://twitter.com/jdoe",
      "linkedin": "https://linkedin.com/in/jdoe"
    },
    "preferred_contact_method": "email",
    "marketing_email_consent": true,
    "sms_notifications_consent": false,
    "push_notifications_consent": true,
    "profile_visibility": "public",
    "data_processing_consent": true,
    "terms_accepted_at": "2024-01-15T10:00:00Z",
    "privacy_policy_accepted_at": "2024-01-15T10:00:00Z",
    "emergency_contact_name": "Jane Doe",
    "emergency_contact_phone": "+15559876543",
    "emergency_contact_email": "jane@example.com",
    "emergency_contact_relation": "spouse",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `user_setting_id` | string (UUID) | Unique identifier for this settings record |
| `timezone` | string | User's timezone (e.g., `America/Los_Angeles`) |
| `preferred_language` | string | ISO 639-1 language code (e.g., `en`) |
| `locale` | string | Locale string (e.g., `en-US`) |
| `social_links` | object | Map of platform names to profile URLs |
| `preferred_contact_method` | string | `email`, `phone`, or `sms` |
| `marketing_email_consent` | boolean | Whether the user has consented to marketing emails |
| `sms_notifications_consent` | boolean | Whether the user has consented to SMS notifications |
| `push_notifications_consent` | boolean | Whether the user has consented to push notifications |
| `profile_visibility` | string | `public`, `private`, or `friends` |
| `data_processing_consent` | boolean | Whether the user has consented to data processing |
| `terms_accepted_at` | datetime | When the user accepted the terms of service |
| `privacy_policy_accepted_at` | datetime | When the user accepted the privacy policy |
| `emergency_contact_name` | string | Emergency contact full name |
| `emergency_contact_phone` | string | Emergency contact phone number |
| `emergency_contact_email` | string | Emergency contact email address |
| `emergency_contact_relation` | string | Relationship to the emergency contact |
| `created_at` | datetime | When the settings record was created |
| `updated_at` | datetime | When the settings record was last updated |

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Missing or invalid JWT |
| 404 | No settings record found for this user |

### Example

```bash
curl -X GET "https://auth.example.com/api/v1/user-settings" \
  -H "Authorization: Bearer <token>"
```

---

## POST /api/v1/user-settings

Creates or updates the authenticated user's settings record. All fields are optional — only the fields provided are applied. If no record exists, one is created.

### Authentication

Bearer JWT required.

### Request Body

```json
{
  "timezone": "America/Los_Angeles",
  "preferred_language": "en",
  "locale": "en-US",
  "social_links": {
    "twitter": "https://twitter.com/jdoe",
    "linkedin": "https://linkedin.com/in/jdoe"
  },
  "preferred_contact_method": "email",
  "marketing_email_consent": true,
  "sms_notifications_consent": false,
  "push_notifications_consent": true,
  "profile_visibility": "public",
  "data_processing_consent": true,
  "emergency_contact_name": "Jane Doe",
  "emergency_contact_phone": "+15559876543",
  "emergency_contact_email": "jane@example.com",
  "emergency_contact_relation": "spouse"
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `timezone` | string | No | Up to 50 characters (e.g., `America/Los_Angeles`) |
| `preferred_language` | string | No | 2–10 characters (ISO 639-1) |
| `locale` | string | No | 2–10 characters (e.g., `en-US`) |
| `social_links` | object | No | Map of string keys to string URLs |
| `preferred_contact_method` | string | No | `email`, `phone`, or `sms` |
| `marketing_email_consent` | boolean | No | |
| `sms_notifications_consent` | boolean | No | |
| `push_notifications_consent` | boolean | No | |
| `profile_visibility` | string | No | `public`, `private`, or `friends` |
| `data_processing_consent` | boolean | No | |
| `emergency_contact_name` | string | No | Up to 200 characters |
| `emergency_contact_phone` | string | No | Up to 20 characters |
| `emergency_contact_email` | string | No | Valid email, up to 255 characters |
| `emergency_contact_relation` | string | No | Up to 50 characters |

### Response — 200 OK

```json
{
  "success": true,
  "message": "User setting saved successfully",
  "data": {
    "user_setting_id": "d4e5f6a7-b8c9-0123-def0-234567890123",
    "timezone": "America/Los_Angeles",
    "preferred_language": "en",
    "locale": "en-US",
    "social_links": {
      "twitter": "https://twitter.com/jdoe"
    },
    "preferred_contact_method": "email",
    "marketing_email_consent": true,
    "sms_notifications_consent": false,
    "push_notifications_consent": true,
    "profile_visibility": "public",
    "data_processing_consent": true,
    "emergency_contact_name": "Jane Doe",
    "emergency_contact_phone": "+15559876543",
    "emergency_contact_email": "jane@example.com",
    "emergency_contact_relation": "spouse",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Malformed JSON |
| 401 | Missing or invalid JWT |
| 422 | Validation error (e.g., invalid `preferred_contact_method`, invalid email format) |

### Example

```bash
curl -X POST "https://auth.example.com/api/v1/user-settings" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "timezone": "America/Los_Angeles",
    "preferred_language": "en",
    "preferred_contact_method": "email",
    "profile_visibility": "public"
  }'
```

---

## DELETE /api/v1/user-settings

Deletes the authenticated user's settings record.

### Authentication

Bearer JWT required.

### Response — 200 OK

```json
{
  "success": true,
  "message": "User setting deleted successfully",
  "data": {
    "user_setting_id": "d4e5f6a7-b8c9-0123-def0-234567890123",
    "timezone": "America/Los_Angeles",
    "preferred_language": "en",
    "profile_visibility": "public",
    "data_processing_consent": true,
    "marketing_email_consent": true,
    "sms_notifications_consent": false,
    "push_notifications_consent": true,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T09:30:00Z"
  }
}
```

### Error Responses

| Status | Description |
|--------|-------------|
| 401 | Missing or invalid JWT |
| 404 | No settings record found for this user |

### Example

```bash
curl -X DELETE "https://auth.example.com/api/v1/user-settings" \
  -H "Authorization: Bearer <token>"
```
