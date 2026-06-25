package authn

import (
	"encoding/json"
	"net/http"
	"time"

	resp "github.com/maintainerd/auth/internal/platform/response"
	"github.com/maintainerd/auth/internal/platform/security"
)

type ForgotPasswordHandler struct {
	forgotPasswordService ForgotPasswordService
}

func NewForgotPasswordHandler(forgotPasswordService ForgotPasswordService) *ForgotPasswordHandler {
	return &ForgotPasswordHandler{
		forgotPasswordService: forgotPasswordService,
	}
}

func (h *ForgotPasswordHandler) ForgotPasswordPublic(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	// Validate query parameters (required for public route)
	clientID, tenantID, ok := authenticationContextQuery(r)
	if !ok {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "forgot_password_missing_params",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/forgot-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Missing or ambiguous authentication context",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Public password recovery requires client_id and does not accept tenant_id")
		return
	}

	// Parse request body
	var req ForgotPasswordRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "forgot_password_invalid_json",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/forgot-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid request body",
			Severity:  "MEDIUM",
		})
		resp.BadRequestBody(w)
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
			Details:   "Request validation failed",
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

	// Process forgot password request (external - use APP_FRONTEND_IDENTITY_HOSTNAME)
	response, err := h.forgotPasswordService.SendPasswordResetEmail(r.Context(), req.Email, clientID, tenantID, false)
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
			Details:   "Password reset request failed",
			Severity:  "HIGH",
		})
		resp.HandleServiceError(w, r, "Failed to process password reset request", err)
		return
	}

	// authevent.Log successful request (don't log whether email exists for security)
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

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" || r.URL.Query().Get("client_id") != "" {
		resp.Error(w, http.StatusBadRequest, "Internal password recovery requires tenant_id and does not accept client_id")
		return
	}

	// Parse request body
	var req ForgotPasswordRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "forgot_password_invalid_json",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/forgot-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid request body",
			Severity:  "MEDIUM",
		})
		resp.BadRequestBody(w)
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
			Details:   "Request validation failed",
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

	// Process forgot password request (internal - use APP_FRONTEND_CONSOLE_HOSTNAME)
	response, err := h.forgotPasswordService.SendPasswordResetEmail(r.Context(), req.Email, nil, &tenantID, true)
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
			Details:   "Password reset request failed",
			Severity:  "HIGH",
		})
		resp.HandleServiceError(w, r, "Failed to process password reset request", err)
		return
	}

	// authevent.Log successful request (don't log whether email exists for security)
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
