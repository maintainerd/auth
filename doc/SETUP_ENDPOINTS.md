# Setup Endpoints Documentation

## Overview

The setup endpoints are **one-time only** endpoints used to initialize the authentication system. These endpoints are only active when the system has not been set up yet.

## Setup Flow

1. **Check Setup Status** - Verify what has been set up
2. **Create Organization** - Initialize organization and run all seeders
3. **Create Admin** - Create the first admin user in the default auth container

## Endpoints

### 1. Get Setup Status

**Endpoint:** `GET /api/v1/setup/status`

**Description:** Check the current setup status of the system.

**Response:**
```json
{
  "success": true,
  "message": "Setup status retrieved successfully",
  "data": {
    "is_organization_setup": false,
    "is_admin_setup": false,
    "is_setup_complete": false
  }
}
```

### 2. Create Organization

**Endpoint:** `POST /api/v1/setup/create_organization`

**Description:** Creates the initial organization and runs all seeders. This can only be done once.

**Request Body:**
```json
{
  "name": "My Organization",
  "description": "Organization description (optional)",
  "email": "admin@myorg.com (optional)",
  "phone": "+1234567890 (optional)"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Organization created successfully",
  "data": {
    "message": "Organization created successfully",
    "organization": {
      "organization_uuid": "123e4567-e89b-12d3-a456-426614174000",
      "name": "My Organization",
      "description": "Organization description",
      "email": "admin@myorg.com",
      "phone": "+1234567890",
      "is_active": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  }
}
```

**What happens when you call this endpoint:**
- Creates the organization directly with your provided details
- Runs all other seeders (service, API, permissions, auth container, identity providers, auth clients, roles, etc.)
- Creates the complete authentication infrastructure

### 3. Create Admin

**Endpoint:** `POST /api/v1/setup/create_admin`

**Description:** Creates the first admin user in the default auth container. This can only be done once and requires the organization to be created first.

**Request Body:**
```json
{
  "username": "admin",
  "password": "securepassword123",
  "email": "admin@myorg.com"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Admin user created successfully",
  "data": {
    "message": "Admin user created successfully",
    "user": {
      "user_uuid": "123e4567-e89b-12d3-a456-426614174001",
      "username": "admin",
      "email": "admin@myorg.com",
      "is_email_verified": true,
      "is_active": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  }
}
```

**What happens when you call this endpoint:**
- Creates admin user in the default auth container
- Assigns super-admin role to the user
- Creates user identity record
- User can immediately login using universal login endpoint

## Important Notes

### One-Time Only Setup

- **Organization creation** can only be done once. If an organization already exists, the endpoint will return an error.
- **Admin creation** can only be done once. If a super-admin user already exists, the endpoint will return an error.

### No Authentication Required

These setup endpoints do **not require authentication** since they are used to bootstrap the system.

### Automatic Auth Container Selection

The admin user is automatically created in the **default auth container**. You don't need to provide `auth_client_id` or `auth_container_id` - the system automatically uses the default values.

### After Setup

Once setup is complete:
1. Use the universal authentication endpoints for login/registration
2. The admin user can access all management endpoints
3. The setup endpoints will reject further attempts

## Error Responses

### Organization Already Exists
```json
{
  "success": false,
  "message": "Failed to create organization",
  "error": "organization already exists: setup can only be run once"
}
```

### Admin Already Exists
```json
{
  "success": false,
  "message": "Failed to create admin",
  "error": "admin user already exists: setup can only be run once"
}
```

### Organization Required First
```json
{
  "success": false,
  "message": "Failed to create admin",
  "error": "organization must be created first"
}
```

## Example Setup Flow

```bash
# 1. Check setup status
curl -X GET http://localhost:8080/api/v1/setup/status

# 2. Create organization
curl -X POST http://localhost:8080/api/v1/setup/create_organization \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Company",
    "description": "My company description",
    "email": "admin@mycompany.com"
  }'

# 3. Create admin user
curl -X POST http://localhost:8080/api/v1/setup/create_admin \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "securepassword123",
    "email": "admin@mycompany.com"
  }'

# 4. Now you can login with the admin user
curl -X POST http://localhost:8080/api/v1/login?auth_client_id=default&auth_container_id=1 \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "securepassword123"
  }'
```
