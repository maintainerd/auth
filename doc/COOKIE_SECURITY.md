# Cookie Security Configuration Examples

This document provides examples of how to configure cookie security settings for different environments.

## 🔧 **Environment-Specific Configuration**

### **Development Environment**

For local development, you may need to adjust cookie settings to work over HTTP:

```go
// config/cookie_config.go
package config

import (
    "os"
    "strconv"
)

type CookieConfig struct {
    Secure   bool
    SameSite http.SameSite
    Domain   string
}

func GetCookieConfig() *CookieConfig {
    isProduction := os.Getenv("ENV") == "production"
    
    return &CookieConfig{
        Secure:   isProduction, // Only secure in production
        SameSite: getSameSitePolicy(isProduction),
        Domain:   os.Getenv("COOKIE_DOMAIN"), // e.g., ".company.com"
    }
}

func getSameSitePolicy(isProduction bool) http.SameSite {
    if isProduction {
        return http.SameSiteStrictMode
    }
    // More lenient for development
    return http.SameSiteLaxMode
}
```

### **Enhanced Response Utility with Environment Configuration**

You can enhance the response utility to use environment-specific settings:

```go
// internal/util/response_util.go - Enhanced Version

// setAuthCookies sets authentication tokens with environment-specific security
func setAuthCookies(w http.ResponseWriter, authResponse *AuthTokenResponse) {
    config := config.GetCookieConfig()
    
    // Set access token cookie
    accessTokenCookie := &http.Cookie{
        Name:     "access_token",
        Value:    authResponse.AccessToken,
        Path:     "/",
        MaxAge:   int(authResponse.ExpiresIn),
        HttpOnly: true,
        Secure:   config.Secure,
        SameSite: config.SameSite,
        Domain:   config.Domain,
    }
    http.SetCookie(w, accessTokenCookie)
    
    // Set ID token cookie
    if authResponse.IDToken != "" {
        idTokenCookie := &http.Cookie{
            Name:     "id_token",
            Value:    authResponse.IDToken,
            Path:     "/",
            MaxAge:   3600,
            HttpOnly: true,
            Secure:   config.Secure,
            SameSite: config.SameSite,
            Domain:   config.Domain,
        }
        http.SetCookie(w, idTokenCookie)
    }
    
    // Set refresh token cookie with enhanced security
    if authResponse.RefreshToken != "" {
        refreshTokenCookie := &http.Cookie{
            Name:     "refresh_token",
            Value:    authResponse.RefreshToken,
            Path:     "/auth/refresh", // Restricted path
            MaxAge:   7 * 24 * 60 * 60, // 7 days
            HttpOnly: true,
            Secure:   config.Secure,
            SameSite: config.SameSite,
            Domain:   config.Domain,
        }
        http.SetCookie(w, refreshTokenCookie)
    }
}
```

## 🌍 **Environment Variables**

### **Development (.env.local)**
```bash
ENV=development
COOKIE_DOMAIN=localhost
COOKIE_SECURE=false
```

### **Production (.env.production)**
```bash
ENV=production
COOKIE_DOMAIN=.company.com
COOKIE_SECURE=true
```

## 🔒 **Advanced Security Headers**

For enhanced security, consider adding these security headers:

```go
// Enhanced security middleware
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // CSRF Protection
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        
        // HTTPS Enforcement (production only)
        if os.Getenv("ENV") == "production" {
            w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        }
        
        // Content Security Policy
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        
        next.ServeHTTP(w, r)
    })
}
```

## 🔧 **CORS Configuration for Cookie-Based Auth**

When using cookies with cross-origin requests:

```go
// CORS configuration for cookie-based authentication
func ConfigureCORS() *cors.Cors {
    return cors.New(cors.Options{
        AllowedOrigins: []string{
            "https://app.company.com",
            "https://admin.company.com",
        },
        AllowedMethods: []string{
            http.MethodGet,
            http.MethodPost,
            http.MethodPut,
            http.MethodDelete,
            http.MethodOptions,
        },
        AllowedHeaders: []string{
            "Accept",
            "Authorization",
            "Content-Type",
            "X-CSRF-Token",
            "X-Token-Delivery", // Our custom header
        },
        ExposedHeaders: []string{
            "Link",
        },
        AllowCredentials: true, // Required for cookies
        MaxAge:          300,
    })
}
```

## 🛡️ **CSRF Protection with Cookies**

When using cookies, implement CSRF protection:

```go
// CSRF protection middleware
func CSRFProtectionMiddleware(next http.Handler) http.Handler {
    return csrf.Protect(
        []byte("your-csrf-key-here"),
        csrf.Secure(os.Getenv("ENV") == "production"),
        csrf.HttpOnly(true),
        csrf.SameSite(csrf.SameSiteStrictMode),
    )(next)
}
```

## 📱 **Mobile App Configuration**

For mobile applications using body-based tokens:

```javascript
// Mobile app token storage
class AuthManager {
    static async login(username, password) {
        const response = await fetch('/api/v1/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                // Note: No X-Token-Delivery header means body response
            },
            body: JSON.stringify({ username, password })
        });
        
        const result = await response.json();
        
        if (result.success) {
            // Store tokens securely (Keychain/Keystore)
            await SecureStorage.setItem('access_token', result.data.access_token);
            await SecureStorage.setItem('refresh_token', result.data.refresh_token);
        }
        
        return result;
    }
    
    static async makeAuthenticatedRequest(url, options = {}) {
        const token = await SecureStorage.getItem('access_token');
        
        return fetch(url, {
            ...options,
            headers: {
                ...options.headers,
                'Authorization': `Bearer ${token}`,
            },
        });
    }
}
```

## 🔍 **Monitoring and Logging**

Log authentication events for security monitoring:

```go
// Security event logging
type SecurityLogger struct {
    logger *logrus.Logger
}

func (s *SecurityLogger) LogAuthEvent(eventType, userID, clientIP, userAgent string) {
    s.logger.WithFields(logrus.Fields{
        "event_type": eventType,
        "user_id":    userID,
        "client_ip":  clientIP,
        "user_agent": userAgent,
        "timestamp":  time.Now().UTC(),
    }).Info("Authentication event")
}

// Usage in handlers
func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
    // ... existing login logic ...
    
    if loginSuccessful {
        h.securityLogger.LogAuthEvent("login_success", user.ID, getClientIP(r), r.UserAgent())
    }
}
```