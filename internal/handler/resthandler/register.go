package resthandler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/service"
		resp "github.com/maintainerd/auth/internal/response"
	"github.com/maintainerd/auth/internal/security"
)

type RegisterHandler struct {
	registerService service.RegisterService
}

func NewRegisterHandler(registerService service.RegisterService) *RegisterHandler {
	return &RegisterHandler{
		registerService: registerService,
	}
}

func (h *RegisterHandler) RegisterPublic(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	// Validate query parameters
	q := dto.RegisterQueryDTO{
		ClientID:   r.URL.Query().Get("client_id"),
		ProviderID: r.URL.Query().Get("provider_id"),
	}

	if err := q.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "registration_validation_failure",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
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
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Suspicious user agent detected",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate body payload
	var req dto.RegisterRequestDTO
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
		resp.Error(w, http.StatusBadRequest, "Invalid request format")
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
			err.Error() == "password contains a common weak password" {
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
			Details:   "Request validation failed: " + err.Error(),
			Severity:  severity,
		})
		resp.ValidationError(w, err)
		return
	}

	// Public registration attempt (requires client_id and provider_id)
	tokenResponse, err := h.registerService.RegisterPublic(
		req.Username, req.Fullname, req.Password, req.Email, req.Phone, q.ClientID, q.ProviderID,
	)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "registration_failure",
			UserID:    req.Username,
			ClientID:  q.ClientID,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Registration failed: " + err.Error(),
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusInternalServerError, "Registration failed", err.Error())
		return
	}

	// Log successful registration
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "registration_success",
		UserID:    req.Username,
		ClientID:  q.ClientID,
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/register",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "User successfully registered",
		Severity:  "LOW",
	})

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
	var req dto.RegisterRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
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
			err.Error() == "password contains a common weak password" {
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
			Details:   "Request validation failed: " + err.Error(),
			Severity:  severity,
		})
		resp.ValidationError(w, err)
		return
	}

	// Internal registration attempt (client_id/provider_id optional)
	tokenResponse, err := h.registerService.Register(
		req.Username, req.Fullname, req.Password, req.Email, req.Phone, clientIDPtr, providerIDPtr,
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
			Details:   "Internal registration failed: " + err.Error(),
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusInternalServerError, "Registration failed", err.Error())
		return
	}

	// Log successful registration
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

func (h *RegisterHandler) RegisterInvite(w http.ResponseWriter, r *http.Request) {
	// Get invite token from query parameters
	inviteToken := r.URL.Query().Get("invite_token")
	if inviteToken == "" {
		resp.Error(w, http.StatusBadRequest, "Invite token is required")
		return
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
	var req dto.LoginRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Internal register with invite (client_id/provider_id optional)
	tokenResponse, err := h.registerService.RegisterInvite(
		req.Username,
		req.Password,
		inviteToken,
		clientIDPtr, providerIDPtr,
	)
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, "Registration failed", err.Error())
		return
	}

	// Response with optional cookie delivery based on X-Token-Delivery header
	resp.CreatedWithCookies(w, r, tokenResponse, "Registration successful")
}

func (h *RegisterHandler) RegisterInvitePublic(w http.ResponseWriter, r *http.Request) {
	// Validate query parameters
	q := dto.RegisterInviteQueryDTO{
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
	var req dto.LoginRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Public register with invite (requires client_id and provider_id)
	tokenResponse, err := h.registerService.RegisterInvitePublic(
		req.Username,
		req.Password,
		q.ClientID,
		q.ProviderID,
		q.InviteToken,
	)
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, "Registration failed", err.Error())
		return
	}

	// Response with optional cookie delivery based on X-Token-Delivery header
	resp.CreatedWithCookies(w, r, tokenResponse, "Registration successful")
}
