package authn

import (
	"encoding/json"
	"net/http"
	"time"

	resp "github.com/maintainerd/auth/internal/platform/response"
	"github.com/maintainerd/auth/internal/platform/security"
)

type EmailVerificationHandler struct {
	emailVerificationService EmailVerificationService
}

func NewEmailVerificationHandler(emailVerificationService EmailVerificationService) *EmailVerificationHandler {
	return &EmailVerificationHandler{
		emailVerificationService: emailVerificationService,
	}
}

// SendVerificationEmailPublic handles public resend requests; requires client_id and provider_id.
func (h *EmailVerificationHandler) SendVerificationEmailPublic(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	clientID := r.URL.Query().Get("client_id")
	providerID := r.URL.Query().Get("provider_id")
	if clientID == "" || providerID == "" {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_missing_params",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Missing required client_id or provider_id parameters",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Missing required parameters: client_id and provider_id")
		return
	}

	h.handleSendVerification(w, r, &clientID, &providerID, startTime, sc)
}

// SendVerificationEmail handles internal resend requests; client_id/provider_id are optional.
func (h *EmailVerificationHandler) SendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)

	var clientIDPtr, providerIDPtr *string
	if v := r.URL.Query().Get("client_id"); v != "" {
		clientIDPtr = &v
	}
	if v := r.URL.Query().Get("provider_id"); v != "" {
		providerIDPtr = &v
	}

	h.handleSendVerification(w, r, clientIDPtr, providerIDPtr, startTime, sc)
}

func (h *EmailVerificationHandler) handleSendVerification(
	w http.ResponseWriter,
	r *http.Request,
	clientID, providerID *string,
	startTime time.Time,
	sc securityContext,
) {
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	var req SendEmailVerificationRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_invalid_json",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid request body",
			Severity:  "MEDIUM",
		})
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_validation_failure",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request validation failed",
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	if err := security.CheckRateLimit(req.Email); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_rate_limited",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Rate limit exceeded for email verification send",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
		return
	}

	response, err := h.emailVerificationService.SendVerificationEmail(r.Context(), req.Email, clientID, providerID)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_service_error",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Email verification send failed",
			Severity:  "HIGH",
		})
		resp.HandleServiceError(w, r, "Failed to send verification email", err)
		return
	}

	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "email_verification_request",
		UserID:    req.Email,
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/email-verification/send",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "Email verification send processed",
		Severity:  "INFO",
	})

	resp.Success(w, response, "Verification email sent")
}

// VerifyEmail consumes a verification code. Public + internal share the same handler
// because the OTP is self-contained and doesn't require a client_id at consume time.
func (h *EmailVerificationHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	var req VerifyEmailRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_invalid_json",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/verify",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid request body",
			Severity:  "MEDIUM",
		})
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_validation_failure",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/verify",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request validation failed",
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	if err := security.CheckRateLimit(req.Email); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_rate_limited",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/verify",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Rate limit exceeded for email verification consume",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
		return
	}

	response, err := h.emailVerificationService.VerifyEmail(r.Context(), req.Email, req.OTP)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to verify email", err)
		return
	}

	resp.Success(w, response, "Email verified")
}
