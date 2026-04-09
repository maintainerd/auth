package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	resp "github.com/maintainerd/auth/internal/rest/response"
	"github.com/maintainerd/auth/internal/security"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/signedurl"
)

type ResetPasswordHandler struct {
	resetPasswordService service.ResetPasswordService
}

func NewResetPasswordHandler(resetPasswordService service.ResetPasswordService) *ResetPasswordHandler {
	return &ResetPasswordHandler{
		resetPasswordService: resetPasswordService,
	}
}

func (h *ResetPasswordHandler) ResetPasswordPublic(w http.ResponseWriter, r *http.Request) {
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

	// Validate signed URL parameters first (security critical)
	signedParams, err := signedurl.ValidateSignedURL(r.URL.Query())
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_invalid_signature",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid signed URL: " + err.Error(),
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid or expired reset link")
		return
	}

	// Extract validated parameters from signed URL
	clientID := signedParams["client_id"]
	providerID := signedParams["provider_id"]
	urlToken := signedParams["token"]

	if clientID == "" || providerID == "" || urlToken == "" {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_missing_signed_params",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Missing required parameters in signed URL",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid reset link parameters")
		return
	}

	// Parse request body
	var req dto.ResetPasswordRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_invalid_json",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid JSON in request body",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_validation_failure",
			UserID:    urlToken,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request validation failed: " + err.Error(),
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	// Use token from signed URL (more secure) - no fallback needed for public route
	token := urlToken

	// Rate limiting check to prevent abuse
	if err := security.CheckRateLimit(token); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_rate_limited",
			UserID:    token,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Rate limit exceeded for password reset",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
		return
	}

	// Process reset password request
	response, err := h.resetPasswordService.ResetPassword(r.Context(), token, req.NewPassword, &clientID, &providerID)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_service_error",
			UserID:    token,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Service error: " + err.Error(),
			Severity:  "HIGH",
		})
		resp.HandleServiceError(w, r, "Failed to reset password", err)
		return
	}

	// Log successful password reset
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "reset_password_success",
		UserID:    token,
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/reset-password",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "Password reset completed successfully",
		Severity:  "INFO",
	})

	resp.Success(w, response, "Password reset successfully")
}

func (h *ResetPasswordHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
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

	// Always require signed URL validation for reset password
	signedParams, err := signedurl.ValidateSignedURL(r.URL.Query())
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_invalid_signature",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid signed URL: " + err.Error(),
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid or expired reset link")
		return
	}

	// Extract validated parameters from signed URL
	var clientIDPtr, providerIDPtr *string
	if clientID := signedParams["client_id"]; clientID != "" {
		clientIDPtr = &clientID
	}
	if providerID := signedParams["provider_id"]; providerID != "" {
		providerIDPtr = &providerID
	}
	token := signedParams["token"]

	if token == "" {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_missing_token",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Missing token in signed URL",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid reset link")
		return
	}

	// Parse request body
	var req dto.ResetPasswordRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_invalid_json",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid JSON in request body",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_validation_failure",
			UserID:    token,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request validation failed: " + err.Error(),
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	// Token is always from signed URL - no need to validate against request body

	// Rate limiting check to prevent abuse
	if err := security.CheckRateLimit(token); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_rate_limited",
			UserID:    token,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Rate limit exceeded for password reset",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
		return
	}

	// Process reset password request
	response, err := h.resetPasswordService.ResetPassword(r.Context(), token, req.NewPassword, clientIDPtr, providerIDPtr)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_service_error",
			UserID:    token,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Service error: " + err.Error(),
			Severity:  "HIGH",
		})
		resp.HandleServiceError(w, r, "Failed to reset password", err)
		return
	}

	// Log successful password reset
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "reset_password_success",
		UserID:    token,
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/reset-password",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "Password reset completed successfully",
		Severity:  "INFO",
	})

	resp.Success(w, response, "Password reset successfully")
}
