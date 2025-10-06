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

type RegisterHandler struct {
	registerService service.RegisterService
}

func NewRegisterHandler(registerService service.RegisterService) *RegisterHandler {
	return &RegisterHandler{
		registerService: registerService,
	}
}

func (h *RegisterHandler) Register(w http.ResponseWriter, r *http.Request) {
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
	q := dto.RegisterQueryDto{
		AuthClientID:    util.SanitizeInput(r.URL.Query().Get("auth_client_id")),
		AuthContainerID: util.SanitizeInput(r.URL.Query().Get("auth_container_id")),
	}

	if err := q.Validate(); err != nil {
		util.LogSecurityEvent(util.SecurityEvent{
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
			Endpoint:  "/register",
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
		util.Error(w, http.StatusBadRequest, "Invalid request format")
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

		util.LogSecurityEvent(util.SecurityEvent{
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
		util.ValidationError(w, err)
		return
	}

	// Registration attempt
	tokenResponse, err := h.registerService.Register(
		req.Username, req.Password, q.AuthClientID, q.AuthContainerID,
	)
	if err != nil {
		util.LogSecurityEvent(util.SecurityEvent{
			EventType: "registration_failure",
			UserID:    req.Username,
			ClientID:  q.AuthClientID,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Registration failed",
			Severity:  "MEDIUM",
		})
		util.Error(w, http.StatusInternalServerError, "Registration failed")
		return
	}

	// Log successful registration
	util.LogSecurityEvent(util.SecurityEvent{
		EventType: "registration_success",
		UserID:    req.Username,
		ClientID:  q.AuthClientID,
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
	util.CreatedWithCookies(w, r, tokenResponse, "Registration successful")
}

func (h *RegisterHandler) RegisterInvite(w http.ResponseWriter, r *http.Request) {
	// Validate query parameters
	q := dto.RegisterInviteQueryDto{
		AuthClientID:    r.URL.Query().Get("auth_client_id"),
		AuthContainerID: r.URL.Query().Get("auth_container_id"),
		InviteToken:     r.URL.Query().Get("invite_token"),
		Expires:         r.URL.Query().Get("expires"),
		Sig:             r.URL.Query().Get("sig"),
	}

	if err := q.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Validate body payload
	var req dto.AuthRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Register
	tokenResponse, err := h.registerService.RegisterInvite(
		req.Username,
		req.Password,
		q.AuthClientID,
		q.AuthContainerID,
		q.InviteToken,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Registration failed", err.Error())
		return
	}

	// Response with optional cookie delivery based on X-Token-Delivery header
	util.CreatedWithCookies(w, r, tokenResponse, "Registration successful")
}
