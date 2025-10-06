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

	// Validate query parameters
	q := dto.LoginQueryDto{
		AuthClientID:    util.SanitizeInput(r.URL.Query().Get("auth_client_id")),
		AuthContainerID: util.SanitizeInput(r.URL.Query().Get("auth_container_id")),
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
	var req dto.AuthRequestDto
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

	// Sanitize input data
	req.Username = util.SanitizeInput(req.Username)
	req.Password = util.SanitizeInput(req.Password)

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

	// Login attempt
	tokenResponse, err := h.loginService.Login(
		req.Username, req.Password, q.AuthClientID, q.AuthContainerID,
	)
	if err != nil {
		util.LogSecurityEvent(util.SecurityEvent{
			EventType: "login_failure",
			UserID:    req.Username,
			ClientID:  q.AuthClientID,
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
		ClientID:  q.AuthClientID,
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
