package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	resp "github.com/maintainerd/auth/internal/response"
	"github.com/maintainerd/auth/internal/security"
	"github.com/maintainerd/auth/internal/service"
)

type ForgotPasswordHandler struct {
	forgotPasswordService service.ForgotPasswordService
}

func NewForgotPasswordHandler(forgotPasswordService service.ForgotPasswordService) *ForgotPasswordHandler {
	return &ForgotPasswordHandler{
		forgotPasswordService: forgotPasswordService,
	}
}

func (h *ForgotPasswordHandler) ForgotPasswordPublic(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	// Validate query parameters (required for public route)
	clientID := r.URL.Query().Get("client_id")
	providerID := r.URL.Query().Get("provider_id")

	if clientID == "" || providerID == "" {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "forgot_password_missing_params",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/forgot-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Missing required client_id or provider_id parameters",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Missing required parameters: client_id and provider_id")
		return
	}

	// Parse request body
	var req dto.ForgotPasswordRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "forgot_password_invalid_json",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/forgot-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid JSON in request body",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "forgot_password_validation_failure",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/forgot-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request validation failed: " + err.Error(),
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	// Rate limiting check to prevent abuse
	if err := security.CheckRateLimit(req.Email); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "forgot_password_rate_limited",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/forgot-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Rate limit exceeded for forgot password",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
		return
	}

	// Process forgot password request (external - use ACCOUNT_HOSTNAME)
	response, err := h.forgotPasswordService.SendPasswordResetEmail(req.Email, &clientID, &providerID, false)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "forgot_password_service_error",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/forgot-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Service error: " + err.Error(),
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusInternalServerError, "Failed to process password reset request")
		return
	}

	// Log successful request (don't log whether email exists for security)
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "forgot_password_request",
		UserID:    req.Email,
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/forgot-password",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "Password reset request processed",
		Severity:  "INFO",
	})

	resp.Success(w, response, "Password reset email sent")
}

func (h *ForgotPasswordHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
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

	// Parse request body
	var req dto.ForgotPasswordRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "forgot_password_invalid_json",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/forgot-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid JSON in request body",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "forgot_password_validation_failure",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/forgot-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request validation failed: " + err.Error(),
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	// Rate limiting check to prevent abuse
	if err := security.CheckRateLimit(req.Email); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "forgot_password_rate_limited",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/forgot-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Rate limit exceeded for forgot password",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
		return
	}

	// Process forgot password request (internal - use AUTH_HOSTNAME)
	response, err := h.forgotPasswordService.SendPasswordResetEmail(req.Email, clientIDPtr, providerIDPtr, true)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "forgot_password_service_error",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/forgot-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Service error: " + err.Error(),
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusInternalServerError, "Failed to process password reset request")
		return
	}

	// Log successful request (don't log whether email exists for security)
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "forgot_password_request",
		UserID:    req.Email,
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/forgot-password",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "Password reset request processed",
		Severity:  "INFO",
	})

	resp.Success(w, response, "Password reset email sent")
}
