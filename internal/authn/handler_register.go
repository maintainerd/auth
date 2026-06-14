package authn

import (
	"encoding/json"
	"net/http"
	"time"

	resp "github.com/maintainerd/auth/internal/platform/response"
	"github.com/maintainerd/auth/internal/platform/security"
)

type RegisterHandler struct {
	registerService RegisterService
}

func NewRegisterHandler(
	registerService RegisterService,
) *RegisterHandler {
	return &RegisterHandler{
		registerService: registerService,
	}
}

func (h *RegisterHandler) RegisterPublic(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	// Parse optional query parameters (client_id and provider_id)
	var clientIDPtr, providerIDPtr *string
	var clientIDStr string
	if clientID := r.URL.Query().Get("client_id"); clientID != "" {
		clientIDPtr = &clientID
		clientIDStr = clientID
	}
	if providerID := r.URL.Query().Get("provider_id"); providerID != "" {
		providerIDPtr = &providerID
	}

	// Validate User-Agent for suspicious patterns
	if !security.ValidateUserAgent(userAgentStr) {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "suspicious_user_agent",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Suspicious user agent detected",
			Severity:  "HIGH",
		})
		resp.BadRequest(w)
		return
	}

	// Validate body payload
	var req RegisterRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "registration_malformed_request",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Malformed JSON request body",
			Severity:  "MEDIUM",
		})
		resp.BadRequestBody(w)
		return
	}

	// Validate using DTO convention for registration (includes sanitization and password strength)
	if err := req.ValidateForRegistration(); err != nil {
		// Determine event type based on error
		eventType := "registration_validation_failure"
		severity := "MEDIUM"
		if err.Error() == "password is too weak" ||
			err.Error() == "password must contain at least one uppercase letter" ||
			err.Error() == "password must contain at least one lowercase letter" ||
			err.Error() == "password must contain at least one digit" ||
			err.Error() == "password must contain at least one special character" ||
			err.Error() == "password contains a common weak password" ||
			err.Error() == "password is a common weak password" {
			eventType = "registration_weak_password"
		}

		security.LogSecurityEvent(security.SecurityEvent{
			EventType: eventType,
			UserID:    req.Username,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request validation failed",
			Severity:  severity,
		})
		resp.ValidationError(w, err)
		return
	}

	// Public registration attempt (client_id/provider_id optional)
	ctx := contextWithRegistrationCaptchaToken(r.Context(), req.CaptchaToken)
	tokenResponse, err := h.registerService.RegisterPublic(
		ctx, req.Username, req.Fullname, req.Password, req.Email, req.Phone, clientIDPtr, providerIDPtr,
	)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "registration_failure",
			UserID:    req.Username,
			ClientID:  clientIDStr,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Registration failed",
			Severity:  "MEDIUM",
		})
		resp.HandleServiceError(w, r, "Registration failed", err)
		return
	}

	// authevent.Log successful registration
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "registration_success",
		UserID:    req.Username,
		ClientID:  clientIDStr,
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/register",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "User successfully registered",
		Severity:  "LOW",
	})

	// The verification email (when the tenant policy requires it) is sent by the
	// registration service inside its transaction — see RegisterService.RegisterPublic.
	// Sending it here as well would issue a second OTP and revoke the first.

	// Response with optional cookie delivery based on X-Token-Delivery header
	resp.CreatedWithCookies(w, r, tokenResponse, "Registration successful")
}

func (h *RegisterHandler) Register(w http.ResponseWriter, r *http.Request) {
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
	var req RegisterRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	// Validate using DTO convention for registration (includes sanitization and password strength)
	if err := req.ValidateForRegistration(); err != nil {
		// Determine event type based on error
		eventType := "registration_validation_failure"
		severity := "MEDIUM"
		if err.Error() == "password is too weak" ||
			err.Error() == "password must contain at least one uppercase letter" ||
			err.Error() == "password must contain at least one lowercase letter" ||
			err.Error() == "password must contain at least one digit" ||
			err.Error() == "password must contain at least one special character" ||
			err.Error() == "password contains a common weak password" ||
			err.Error() == "password is a common weak password" {
			eventType = "registration_weak_password"
		}

		security.LogSecurityEvent(security.SecurityEvent{
			EventType: eventType,
			UserID:    req.Username,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request validation failed",
			Severity:  severity,
		})
		resp.ValidationError(w, err)
		return
	}

	// Internal registration attempt (client_id/provider_id optional)
	ctx := contextWithRegistrationCaptchaToken(r.Context(), req.CaptchaToken)
	tokenResponse, err := h.registerService.Register(
		ctx, req.Username, req.Fullname, req.Password, req.Email, req.Phone, clientIDPtr, providerIDPtr,
	)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "registration_failure",
			UserID:    req.Username,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Internal registration failed",
			Severity:  "MEDIUM",
		})
		resp.HandleServiceError(w, r, "Registration failed", err)
		return
	}

	// authevent.Log successful registration
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "registration_success",
		UserID:    req.Username,
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/register",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "User successfully registered via internal endpoint",
		Severity:  "LOW",
	})

	// Response with optional cookie delivery based on X-Token-Delivery header
	resp.CreatedWithCookies(w, r, tokenResponse, "Registration successful")
}

func (h *RegisterHandler) RegisterInvitePublic(w http.ResponseWriter, r *http.Request) {
	// Validate query parameters
	q := RegisterInviteQueryDTO{
		ClientID:    r.URL.Query().Get("client_id"),
		ProviderID:  r.URL.Query().Get("provider_id"),
		InviteToken: r.URL.Query().Get("invite_token"),
		Expires:     r.URL.Query().Get("expires"),
		Sig:         r.URL.Query().Get("sig"),
	}

	if err := q.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Validate body payload
	var req LoginRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Public register with invite (requires client_id and provider_id)
	tokenResponse, err := h.registerService.RegisterInvitePublic(
		r.Context(),
		req.Username,
		req.Password,
		q.ClientID,
		q.ProviderID,
		q.InviteToken,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Registration failed", err)
		return
	}

	// Response with optional cookie delivery based on X-Token-Delivery header
	resp.CreatedWithCookies(w, r, tokenResponse, "Registration successful")
}
