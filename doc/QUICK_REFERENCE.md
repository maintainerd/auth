# 📋 Quick Reference Card

Essential commands and information for Maintainerd Auth service.

## 🚀 **Quick Start (5 Minutes)**

```bash
# 1. Clone and setup
git clone https://github.com/maintainerd/auth.git
cd auth
cp .env.example .env

# 2. Start services
docker-compose up -d

# 3. Initialize system
curl -X POST http://localhost:8080/api/v1/setup \
  -H "Content-Type: application/json" \
  -d '{
    "organization_name": "My Company",
    "admin_email": "admin@company.com", 
    "admin_password": "SecurePassword123!"
  }'
```

## 🔗 **Essential Endpoints**

| Endpoint | Method | Purpose | Auth Required |
|----------|--------|---------|---------------|
| `/api/v1/setup` | POST | Initialize system | ❌ |
| `/api/v1/register` | POST | User registration | ❌ |
| `/api/v1/login` | POST | User authentication | ❌ |
| `/api/v1/logout` | POST | Clear cookies | ❌ |
| `/api/v1/profile` | GET/PUT | User profile | ✅ |

## 🍪 **Token Delivery Options**

### Default (JSON Response)
```bash
curl -X POST "/api/v1/login?auth_client_id=web&tenant_id=1" \
  -H "Content-Type: application/json" \
  -d '{"username":"user@example.com", "password":"password"}'
```

### Cookie Response  
```bash
curl -X POST "/api/v1/login?auth_client_id=web&tenant_id=1" \
  -H "Content-Type: application/json" \
  -H "X-Token-Delivery: cookie" \
  -c cookies.jar \
  -d '{"username":"user@example.com", "password":"password"}'
```

## 🛠️ **Development Commands**

```bash
# Start development environment
docker-compose up -d

# View logs  
docker logs -f m9d-auth-dev

# Hot reload (automatic with Air)
# Just edit files - changes auto-detected

# Rebuild from scratch
docker-compose down
docker-compose build --no-cache
docker-compose up -d

# Reset database (⚠️ deletes all data)
docker-compose down -v
```

## 🔐 **Key Security Settings**

### Cookie Security (Production)
- **HttpOnly**: ✅ Enabled (prevents XSS)
- **Secure**: ✅ HTTPS only  
- **SameSite**: ✅ Strict (prevents CSRF)
- **Path Restrictions**: ✅ Refresh token limited to `/auth/refresh`

### JWT Configuration
- **Algorithm**: RS256 (RSA + SHA-256)
- **Access Token**: 15 minutes expiry
- **ID Token**: 1 hour expiry  
- **Refresh Token**: 7 days expiry

## 📊 **Service URLs (Development)**

| Service | URL | Purpose |
|---------|-----|---------|
| Auth API | http://localhost:8080 | Main authentication API |
| Nginx Proxy | http://localhost:80 | Load balancer/proxy |
| PostgreSQL | localhost:5433 | Database access |
| Redis | localhost:6379 | Cache/sessions |
| RabbitMQ UI | http://localhost:15672 | Message queue admin |

## 🚨 **Common Issues & Solutions**

### Port Already in Use
```bash
# Change ports in docker-compose.yml
ports:
  - "8081:8080"  # Use different host port
```

### Database Connection Failed
```bash
# Wait for PostgreSQL to start
docker logs postgres-db
# Look for: "database system is ready to accept connections"
```

### JWT Key Errors  
```bash
# Regenerate keys
./scripts/generate-jwt-keys.sh
# Copy output to .env file
```

### Cookies Not Working
- Check domain settings in browser DevTools
- Ensure HTTPS in production
- Include `credentials: 'include'` in fetch requests

## 📚 **Documentation Links**

- **[Setup Guide](SETUP.md)** - Complete installation guide
- **[API Routes](ROUTE_STRUCTURE.md)** - All endpoints and examples  
- **[Token Delivery](TOKEN_DELIVERY.md)** - JSON vs Cookie authentication
- **[Security Guide](JWT_SECURITY.md)** - Production security settings
- **[Compliance](COMPLIANCE_STATUS.md)** - SOC2 & ISO27001 status

## 🔍 **Debug Commands**

```bash
# Check service health
curl http://localhost:8080/health

# Test database connection  
docker exec -it postgres-db psql -U devuser -d maintainerd

# Test Redis connection
docker exec -it redis-db redis-cli -a Pass123 ping

# View all containers
docker ps -a

# Check JWT token (decode)
echo "JWT_TOKEN" | cut -d. -f2 | base64 -d | jq .
```

## ⚡ **Performance Tips**

- Use cookie delivery for web applications (better security)
- Use JSON delivery for mobile apps (easier integration)
- Implement token refresh logic for long-lived applications
- Set up Redis for session caching in production
- Use connection pooling for database connections

---

*Keep this reference handy for daily development tasks! 📌*