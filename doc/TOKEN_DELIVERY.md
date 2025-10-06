# Token Delivery Options

The authentication system supports flexible token delivery methods, allowing frontend applications to choose between receiving tokens in the response body or as secure HTTP-only cookies.

## 🎯 **Overview**

When authenticating (login or registration), frontend clients can specify their preferred token delivery method using the `X-Token-Delivery` header:

- **`body`** (default): Tokens are returned in the JSON response body
- **`cookie`**: Tokens are set as secure HTTP-only cookies

## 📋 **Usage**

### **Body Response (Default)**

```bash
curl -X POST "http://auth.company.com/api/v1/login?auth_client_id=web-app&auth_container_id=1" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "user@company.com",
    "password": "userpassword"
  }'
```

**Response:**
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "access_token": "eyJhbGciOiJSUzI1NiIs...",
    "id_token": "eyJhbGciOiJSUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJSUzI1NiIs...",
    "expires_in": 3600,
    "token_type": "Bearer",
    "issued_at": 1699123456
  }
}
```

### **Cookie Response**

```bash
curl -X POST "http://auth.company.com/api/v1/login?auth_client_id=web-app&auth_container_id=1" \
  -H "Content-Type: application/json" \
  -H "X-Token-Delivery: cookie" \
  -d '{
    "username": "user@company.com",
    "password": "userpassword"
  }'
```

**Response:**
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "expires_in": 3600,
    "token_type": "Bearer",
    "issued_at": 1699123456
  }
}
```

**Cookies Set:**
- `access_token` - HTTP-only, Secure, SameSite=Strict, expires in 3600 seconds
- `id_token` - HTTP-only, Secure, SameSite=Strict, expires in 1 hour
- `refresh_token` - HTTP-only, Secure, SameSite=Strict, path="/auth/refresh", expires in 7 days

## 🔒 **Security Features**

### **Cookie Security Attributes**
All authentication cookies are set with maximum security:

- **`HttpOnly`**: Prevents JavaScript access, mitigating XSS attacks
- **`Secure`**: Only transmitted over HTTPS in production
- **`SameSite=Strict`**: Prevents CSRF attacks
- **Path restrictions**: Refresh token limited to `/auth/refresh` endpoint

### **Token Isolation**
- **Access Token**: Short-lived (1 hour), used for API access
- **ID Token**: Contains user profile information, expires in 1 hour
- **Refresh Token**: Long-lived (7 days), restricted to refresh endpoint only

## 📚 **Implementation Examples**

### **JavaScript/Fetch API**

```javascript
// Login with cookie delivery
const response = await fetch('/api/v1/login?auth_client_id=web-app&auth_container_id=1', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-Token-Delivery': 'cookie'
  },
  credentials: 'include', // Important: Include cookies in requests
  body: JSON.stringify({
    username: 'user@company.com',
    password: 'userpassword'
  })
});

const result = await response.json();
console.log('Login successful:', result.success);
// Tokens are now stored in secure cookies automatically
```

### **React/Axios Example**

```javascript
import axios from 'axios';

// Configure axios to include cookies
axios.defaults.withCredentials = true;

const loginWithCookies = async (username, password) => {
  try {
    const response = await axios.post('/api/v1/login', {
      username,
      password
    }, {
      params: {
        auth_client_id: 'web-app',
        auth_container_id: '1'
      },
      headers: {
        'X-Token-Delivery': 'cookie'
      }
    });
    
    return response.data;
  } catch (error) {
    throw error.response.data;
  }
};
```

### **Registration with Cookie Delivery**

```javascript
const register = async (username, password) => {
  const response = await fetch('/api/v1/register?auth_client_id=web-app&auth_container_id=1', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Token-Delivery': 'cookie'
    },
    credentials: 'include',
    body: JSON.stringify({ username, password })
  });
  
  return response.json();
};
```

## 🔄 **Token Refresh with Cookies**

When using cookies, the refresh token is automatically included in requests to the refresh endpoint:

```javascript
const refreshToken = async () => {
  const response = await fetch('/auth/refresh', {
    method: 'POST',
    credentials: 'include', // Includes refresh_token cookie
    headers: {
      'X-Token-Delivery': 'cookie'
    }
  });
  
  // New tokens are automatically set as cookies
  return response.json();
};
```

## 🚪 **Logout with Cookie Clearing**

A logout endpoint is available that automatically clears all authentication cookies:

**Endpoint**: `POST /api/v1/logout`

```bash
curl -X POST "http://auth.company.com/api/v1/logout" \
  -H "Content-Type: application/json" \
  --cookie "access_token=eyJhbG...; id_token=eyJhbG...; refresh_token=eyJhbG..."
```

**JavaScript Example:**
```javascript
const logout = async () => {
  const response = await fetch('/api/v1/logout', {
    method: 'POST',
    credentials: 'include' // Include cookies in request
  });
  
  // Server automatically clears all auth cookies
  const result = await response.json();
  console.log(result.message); // "Logout successful"
  
  return result;
};
```

**Response:**
```json
{
  "success": true,
  "message": "Logout successful"
}
```

The logout endpoint automatically clears all authentication cookies regardless of how they were originally set.

## ⚠️ **Troubleshooting**

### **Common Issues**

#### **Cookies Not Being Set**
- **Check Domain**: Ensure your frontend and API are on the same domain or properly configured for cross-origin cookies
- **HTTPS Required**: In production, cookies require HTTPS (`Secure` flag is enabled)
- **Browser Settings**: Check if browser blocks third-party cookies

#### **Authentication Failing**
- **Missing Header**: Ensure `X-Token-Delivery: cookie` header is sent for cookie-based auth
- **Cookie Path**: Verify cookies are sent to the correct paths (especially refresh tokens)
- **Expiration**: Check if tokens have expired

#### **CORS Issues with Cookies**
```javascript
// Ensure credentials are included in requests
fetch('/api/v1/login', {
  credentials: 'include', // This is crucial for cookies
  headers: {
    'X-Token-Delivery': 'cookie'
  }
});
```

#### **Development Environment**
- **Localhost Issues**: Some browsers have special handling for localhost cookies
- **Port Differences**: Ensure frontend and backend ports are properly configured
- **Mixed Content**: Don't mix HTTP/HTTPS in development

### **Debug Tips**

#### **Check Cookie Storage**
```javascript
// In browser console
console.log(document.cookie);
// Look for access_token, id_token, refresh_token
```

#### **Verify Headers**
```bash
# Check response headers
curl -v -X POST "http://localhost:8080/api/v1/login" \
  -H "X-Token-Delivery: cookie" \
  -H "Content-Type: application/json" \
  -d '{"username":"test@example.com", "password":"password"}'
```

#### **Test Token Validity**
```bash
# Decode JWT to check expiration
echo "eyJhbGciOiJSUzI1NiIs..." | cut -d. -f2 | base64 -d | jq .
```

## ⚡ **Best Practices**

### **When to Use Cookie Delivery**
- ✅ Web applications (SPAs, traditional web apps)
- ✅ Same-origin requests
- ✅ When you want automatic token management
- ✅ Enhanced XSS protection requirements

### **When to Use Body Delivery**
- ✅ Mobile applications
- ✅ Cross-origin API access
- ✅ When you need custom token storage logic
- ✅ API integrations and machine-to-machine communication

### **Security Considerations**
1. **Always use HTTPS** in production for cookie security
2. **Configure CORS properly** when using cookies with cross-origin requests
3. **Implement CSRF protection** when using cookies (though SameSite=Strict helps)
4. **Set proper cookie domain** for multi-subdomain applications

## 🛠️ **Development vs Production**

### **Development**
- Cookies work over HTTP
- `Secure` flag is automatically adjusted for local development

### **Production**
- Cookies require HTTPS
- All security flags are enforced
- Proper domain configuration required

## 📖 **Error Handling**

Both delivery methods use the same error response format:

```json
{
  "success": false,
  "error": "Login failed",
  "details": "Invalid credentials"
}
```

The only difference is that with cookie delivery, no tokens are set on authentication failure.