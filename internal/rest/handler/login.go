package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/service"
		"github.com/maintainerd/auth/internal/cookie"
	resp "github.com/maintainerd/auth/internal/rest/response"
	"github.com/maintainerd/auth/internal/security"
)

type LoginHandler struct {
	loginService service.LoginService
}

func NewLoginHandler(loginService service.LoginService) *LoginHandler {
	return &LoginHandler{
		loginService: loginService,
	}
}

// securityContext holds the values extracted from the request context that are
// set by SecurityContextMiddleware. It is used for audit logging in every handler.
type securityContext struct {
	clientIP  string
	userAgent string
	requestID string
}

// extractSecurityContext reads the security-related values that
// SecurityContextMiddleware stores in the request context.
func extractSecurityContext(r *http.Request) securityContext {
	strVal := func(key middleware.SecurityContextKey) string {
		if v := r.Context().Value(key); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	return securityContext{
		clientIP:  strVal(middleware.ClientIPKey),
		userAgent: strVal(middleware.UserAgentKey),
		requestID: strVal(middleware.RequestIDKey),
	}
}

func (h *LoginHandler) LoginPublic(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	// Validate query parameters
	q := dto.LoginQueryDTO{
		ClientID:   r.URL.Query().Get("client_id"),
		ProviderID: r.URL.Query().Get("provider_id"),
	}

	if err := q.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_validation_failure",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/login",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Query parameter validation failed",
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	// Validate User-Agent for suspicious patterns
	if !security.ValidateUserAgent(userAgentStr) {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "suspicious_user_agent",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/login",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Suspicious user agent detected",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate body payload
	var req dto.LoginRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_malformed_request",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/login",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Malformed JSON request body",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate using DTO convention (includes sanitization)
	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_validation_failure",
			UserID:    req.Username,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/login",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request body validation failed",
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	// Public login attempt (requires client_id and provider_id)
	tokenResponse, err := h.loginService.LoginPublic(
		req.Username, req.Password, q.ClientID, q.ProviderID,
	)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_failure",
			UserID:    req.Username,
			ClientID:  q.ClientID,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/login",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Authentication failed",
			Severity:  "MEDIUM",
		})
		resp.HandleServiceError(w, "Authentication failed", err)
		return
	}

	// Log successful login
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "login_success",
		UserID:    req.Username,
		ClientID:  q.ClientID,
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/login",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "User successfully authenticated",
		Severity:  "LOW",
	})

	// Response with optional cookie delivery based on X-Token-Delivery header
	resp.SuccessWithCookies(w, r, tokenResponse, "Login successful")
}

func (h *LoginHandler) Logout(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	// Log logout event
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "logout",
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/logout",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "User logout initiated",
		Severity:  "LOW",
	})

	// Clear authentication cookies if they exist
	cookie.ClearAuthCookies(w)

	// Return success response
	resp.Success(w, nil, "Logout successful")
}

func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	// Parse optional query parameters (client_id and provider_id)
	var clientIDPtr, providerIDPtr *string
	if clientID := r.URL.Query().Get("client_id"); clientID != "" {
		clientIDPtr = &clientID
	}
	if providerID := r.URL.Query().Get("provider_id"); providerID != "" {
		providerIDPtr = &providerID
	}

	// Validate body payload
	var req dto.LoginRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Validate using DTO convention (includes sanitization)
	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_validation_failure",
			UserID:    req.Username,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/login",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request body validation failed",
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	// Internal login attempt (client_id/provider_id optional)
	tokenResponse, err := h.loginService.Login(
		req.Username, req.Password, clientIDPtr, providerIDPtr,
	)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_failure",
			UserID:    req.Username,
			ClientID:  "internal",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/login",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Internal authentication failed",
			Severity:  "MEDIUM",
		})
		resp.HandleServiceError(w, "Authentication failed", err)
		return
	}

	// Log successful login
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "login_success",
		UserID:    req.Username,
		ClientID:  "internal",
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/login",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "User successfully authenticated via internal endpoint",
		Severity:  "LOW",
	})

	// Response with optional cookie delivery based on X-Token-Delivery header
	resp.SuccessWithCookies(w, r, tokenResponse, "Login successful")
}
