# Authentication Route Structure

## 🎯 **Unified Route Overview**

The authentication system now uses a **unified route structure** on both servers, with different endpoints for public and admin functionality. This provides a cleaner, more maintainable architecture.

---

## 🌐 **External Server (Port 8080)**

**Purpose**: Public-facing authentication for external applications and users

### **Registration Routes**
```
POST /api/v1/register              → RegisterPublic (no auth required)
POST /api/v1/register/invite       → RegisterPublicInvite (no auth required)
POST /api/v1/register/admin        → RegisterPrivate (first admin only)
POST /api/v1/register/admin/invite → RegisterPrivateInvite (JWT + user:create permission)
```

### **Login Routes**
```
POST /api/v1/login                 → LoginPublic (no auth required, requires client_id)
POST /api/v1/login/admin           → LoginPrivate (no auth required, uses default client)
```

### **Profile Routes**
```
GET  /api/v1/profile               → GetProfile (JWT required)
PUT  /api/v1/profile               → UpdateProfile (JWT required)
```

**Access Control**:
- ✅ **Public Access**: No VPN or firewall restrictions
- ✅ **Rate Limited**: Protected against brute force attacks
- ✅ **Client Validation**: Public routes require valid client_id parameter
- ✅ **Admin Routes**: Admin endpoints available but should be restricted by network

---

## 🔒 **Internal Server (Port 8081)**

**Purpose**: Internal administration and management (VPN/firewall restricted)

### **Authentication Routes** (Same as External)
```
POST /api/v1/register              → RegisterPublic (no auth required)
POST /api/v1/register/invite       → RegisterPublicInvite (no auth required)
POST /api/v1/register/admin        → RegisterPrivate (first admin only)
POST /api/v1/register/admin/invite → RegisterPrivateInvite (JWT + user:create permission)
POST /api/v1/login                 → LoginPublic (no auth required, requires client_id)
POST /api/v1/login/admin           → LoginPrivate (no auth required, uses default client)
```

### **Management Routes**
```
# Organization Management
GET    /api/v1/organizations       → List organizations
POST   /api/v1/organizations       → Create organization
GET    /api/v1/organizations/{id}  → Get organization
PUT    /api/v1/organizations/{id}  → Update organization
DELETE /api/v1/organizations/{id}  → Delete organization

# Auth Container Management
GET    /api/v1/auth-containers     → List auth containers
POST   /api/v1/auth-containers     → Create auth container
GET    /api/v1/auth-containers/{id} → Get auth container
PUT    /api/v1/auth-containers/{id} → Update auth container
DELETE /api/v1/auth-containers/{id} → Delete auth container

# Identity Provider Management
GET    /api/v1/identity-providers  → List identity providers
POST   /api/v1/identity-providers  → Create identity provider
GET    /api/v1/identity-providers/{id} → Get identity provider
PUT    /api/v1/identity-providers/{id} → Update identity provider
DELETE /api/v1/identity-providers/{id} → Delete identity provider

# Auth Client Management
GET    /api/v1/auth-clients        → List auth clients
POST   /api/v1/auth-clients        → Create auth client
GET    /api/v1/auth-clients/{id}   → Get auth client
PUT    /api/v1/auth-clients/{id}   → Update auth client
DELETE /api/v1/auth-clients/{id}   → Delete auth client

# Role Management
GET    /api/v1/roles               → List roles
POST   /api/v1/roles               → Create role
GET    /api/v1/roles/{id}          → Get role
PUT    /api/v1/roles/{id}          → Update role
DELETE /api/v1/roles/{id}          → Delete role

# User Management
GET    /api/v1/users               → List users
POST   /api/v1/users               → Create user
GET    /api/v1/users/{id}          → Get user
PUT    /api/v1/users/{id}          → Update user
DELETE /api/v1/users/{id}          → Delete user

# Profile Management
GET    /api/v1/profile             → Get profile
PUT    /api/v1/profile             → Update profile

# Invite Management
GET    /api/v1/invites             → List invites
POST   /api/v1/invites             → Create invite
GET    /api/v1/invites/{id}        → Get invite
PUT    /api/v1/invites/{id}        → Update invite
DELETE /api/v1/invites/{id}        → Delete invite
```

**Access Control**:
- 🔒 **VPN/Firewall Restricted**: Only accessible from internal network
- 🔒 **JWT Authentication**: Most endpoints require valid JWT tokens
- 🔒 **Permission-Based**: Granular permissions (user:create, user:read, etc.)
- 🔒 **Admin Access**: Full system administration capabilities

---

## 🔧 **Implementation Details**

### **Route Handler Functions**

#### **Unified Register Route**
```go
func RegisterRoute(
    r chi.Router,
    registerHandler *resthandler.RegisterHandler,
    userRepo repository.UserRepository,
    redisClient *redis.Client,
) {
    // Public register (for external users and first admin registration)
    r.Post("/register", registerHandler.RegisterPublic)

    // Public register with invite (for external users)
    r.Post("/register/invite", registerHandler.RegisterPublicInvite)

    // Private register (admin only, first registration)
    r.Post("/register/admin", registerHandler.RegisterPrivate)

    // Private register with invite (requires authentication and user management permissions)
    r.Route("/register/admin/invite", func(r chi.Router) {
        r.Use(middleware.JWTAuthMiddleware)
        r.Use(middleware.UserContextMiddleware(userRepo, redisClient))
        r.With(middleware.PermissionMiddleware([]string{"user:create"})).
            Post("/", registerHandler.RegisterPrivateInvite)
    })
}
```

#### **Unified Login Route**
```go
func LoginRoute(r chi.Router, loginHandler *resthandler.LoginHandler) {
    // Public login (requires client_id parameter)
    r.Post("/login", loginHandler.LoginPublic)

    // Private/admin login (uses default client)
    r.Post("/login/admin", loginHandler.LoginPrivate)
}
```

### **Server Configuration**

#### **External Server (Port 8080)**
```go
r.Route("/api/v1", func(api chi.Router) {
    route.RegisterRoute(api, application.RegisterRestHandler, application.UserRepository, application.RedisClient)
    route.LoginRoute(api, application.LoginRestHandler)
    route.ProfileRoute(api, application.ProfileRestHandler, application.UserRepository, application.RedisClient)
})
```

#### **Internal Server (Port 8081)**
```go
r.Route("/api/v1", func(api chi.Router) {
    // ... management routes ...
    route.RegisterRoute(api, application.RegisterRestHandler, application.UserRepository, application.RedisClient)
    route.LoginRoute(api, application.LoginRestHandler)
    route.ProfileRoute(api, application.ProfileRestHandler, application.UserRepository, application.RedisClient)
    // ... other internal routes ...
})
```

---

## 🎯 **Architecture Benefits**

### **1. ✅ Unified Route Structure**
- **Same routes**: Both servers expose identical endpoints
- **Consistent API**: No confusion about which server to use
- **Simplified maintenance**: Single route definition for both servers

### **2. ✅ Clear Endpoint Separation**
- **Public endpoints**: `/register`, `/register/invite`, `/login`
- **Admin endpoints**: `/register/admin`, `/register/admin/invite`, `/login/admin`
- **Logical grouping**: Easy to understand functionality

### **3. ✅ Flexible Deployment**
- **Single server**: Can run everything on one server if needed
- **Dual server**: Can still separate public/internal for security
- **Load balancing**: Easy to distribute load across multiple instances

### **4. ✅ Simplified Architecture**
- **No route duplication**: Single source of truth for all routes
- **Easier testing**: Test once, works on both servers
- **Better maintainability**: Changes apply to both servers automatically

---

## 📋 **Usage Examples**

### **Public User Registration**
```bash
# Public user registration
curl -X POST "http://auth.company.com/api/v1/register?client_id=mobile-app" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "user@company.com",
    "password": "userpassword"
  }'
```

### **Admin User Registration**
```bash
# Admin user registration (first admin or with invite)
curl -X POST "http://auth.company.com/api/v1/register/admin" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin@company.com",
    "password": "adminpassword"
  }'
```

### **Public User Login**
```bash
# Public users login with client_id
curl -X POST "http://auth.company.com/api/v1/login?client_id=mobile-app" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "user@company.com",
    "password": "userpassword"
  }'
```

### **Admin User Login**
```bash
# Admin users login without client_id (uses default client)
curl -X POST "http://auth.company.com/api/v1/login/admin" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin@company.com",
    "password": "adminpassword"
  }'
```

---

## 🚀 **Deployment Configuration**

### **Load Balancer Configuration**
```nginx
# External server (public access)
upstream external_auth {
    server auth-external-1:8080;
    server auth-external-2:8080;
}

# Internal server (VPN only)
upstream internal_auth {
    server auth-internal-1:8081;
    server auth-internal-2:8081;
}

server {
    listen 443 ssl;
    server_name auth.company.com;
    
    location /api/v1/ {
        proxy_pass http://external_auth;
    }
}

server {
    listen 443 ssl;
    server_name auth-admin.company.com;
    
    # Restrict to VPN network
    allow 10.0.0.0/8;
    deny all;
    
    location /api/v1/ {
        proxy_pass http://internal_auth;
    }
}
```

Your authentication system now has a clean, secure route separation that follows security best practices! 🎉
