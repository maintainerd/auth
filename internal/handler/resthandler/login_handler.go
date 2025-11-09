package resthandler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

type LoginHandler struct {
	loginService service.LoginService
}

func NewLoginHandler(loginService service.LoginService) *LoginHandler {
	return &LoginHandler{
		loginService: loginService,
	}
}

func (h *LoginHandler) LoginPublic(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Extract security context
	clientIP := r.Context().Value(middleware.ClientIPKey)
	userAgent := r.Context().Value(middleware.UserAgentKey)
	requestID := r.Context().Value(middleware.RequestIDKey)

	// Convert context values to strings safely
	clientIPStr := ""
	userAgentStr := ""
	requestIDStr := ""

	if clientIP != nil {
		clientIPStr = clientIP.(string)
	}
	if userAgent != nil {
		userAgentStr = userAgent.(string)
	}
	if requestID != nil {
		requestIDStr = requestID.(string)
	}

	// Validate query parameters
	q := dto.LoginQueryDto{
		ClientID:   r.URL.Query().Get("client_id"),
		ProviderID: r.URL.Query().Get("provider_id"),
	}

	if err := q.Validate(); err != nil {
		util.LogSecurityEvent(util.SecurityEvent{
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
		util.ValidationError(w, err)
		return
	}

	// Validate User-Agent for suspicious patterns
	if !util.ValidateUserAgent(userAgentStr) {
		util.LogSecurityEvent(util.SecurityEvent{
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
		util.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate body payload
	var req dto.LoginRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.LogSecurityEvent(util.SecurityEvent{
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
		util.Error(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate using DTO convention (includes sanitization)
	if err := req.Validate(); err != nil {
		util.LogSecurityEvent(util.SecurityEvent{
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
		util.ValidationError(w, err)
		return
	}

	// Public login attempt (requires client_id and provider_id)
	tokenResponse, err := h.loginService.LoginPublic(
		req.Username, req.Password, q.ClientID, q.ProviderID,
	)
	if err != nil {
		util.LogSecurityEvent(util.SecurityEvent{
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
		util.Error(w, http.StatusUnauthorized, "Authentication failed")
		return
	}

	// Log successful login
	util.LogSecurityEvent(util.SecurityEvent{
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
	util.SuccessWithCookies(w, r, tokenResponse, "Login successful")
}

func (h *LoginHandler) Logout(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Extract security context
	clientIP := r.Context().Value(middleware.ClientIPKey)
	userAgent := r.Context().Value(middleware.UserAgentKey)
	requestID := r.Context().Value(middleware.RequestIDKey)

	// Convert context values to strings safely
	clientIPStr := ""
	userAgentStr := ""
	requestIDStr := ""

	if clientIP != nil {
		clientIPStr = clientIP.(string)
	}
	if userAgent != nil {
		userAgentStr = userAgent.(string)
	}
	if requestID != nil {
		requestIDStr = requestID.(string)
	}

	// Log logout event
	util.LogSecurityEvent(util.SecurityEvent{
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
	util.ClearAuthCookies(w)

	// Return success response
	util.Success(w, nil, "Logout successful")
}

func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Extract security context
	clientIP := r.Context().Value(middleware.ClientIPKey)
	userAgent := r.Context().Value(middleware.UserAgentKey)
	requestID := r.Context().Value(middleware.RequestIDKey)

	// Convert context values to strings safely
	clientIPStr := ""
	userAgentStr := ""
	requestIDStr := ""

	if clientIP != nil {
		clientIPStr = clientIP.(string)
	}
	if userAgent != nil {
		userAgentStr = userAgent.(string)
	}
	if requestID != nil {
		requestIDStr = requestID.(string)
	}

	// Parse optional query parameters (client_id and provider_id)
	var clientIDPtr, providerIDPtr *string
	if clientID := r.URL.Query().Get("client_id"); clientID != "" {
		clientIDPtr = &clientID
	}
	if providerID := r.URL.Query().Get("provider_id"); providerID != "" {
		providerIDPtr = &providerID
	}

	// Validate body payload
	var req dto.LoginRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Validate using DTO convention (includes sanitization)
	if err := req.Validate(); err != nil {
		util.LogSecurityEvent(util.SecurityEvent{
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
		util.ValidationError(w, err)
		return
	}

	// Internal login attempt (client_id/provider_id optional)
	tokenResponse, err := h.loginService.Login(
		req.Username, req.Password, clientIDPtr, providerIDPtr,
	)
	if err != nil {
		util.LogSecurityEvent(util.SecurityEvent{
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
		util.Error(w, http.StatusUnauthorized, "Authentication failed")
		return
	}

	// Log successful login
	util.LogSecurityEvent(util.SecurityEvent{
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
	util.SuccessWithCookies(w, r, tokenResponse, "Login successful")
}
