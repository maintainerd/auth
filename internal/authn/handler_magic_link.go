package authn

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	resp "github.com/maintainerd/auth/internal/platform/response"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/service"
)

type MagicLinkHandler struct {
	magicLinkService service.MagicLinkService
}

func NewMagicLinkHandler(magicLinkService service.MagicLinkService) *MagicLinkHandler {
	return &MagicLinkHandler{
		magicLinkService: magicLinkService,
	}
}

// SendMagicLinkPublic handles public magic-link requests; requires client_id and provider_id.
// Mounted on the public surface (port 8081); uses the account-facing frontend hostname.
func (h *MagicLinkHandler) SendMagicLinkPublic(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	clientID := r.URL.Query().Get("client_id")
	providerID := r.URL.Query().Get("provider_id")
	if clientID == "" || providerID == "" {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "magic_link_missing_params",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Missing required client_id or provider_id parameters",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Missing required parameters: client_id and provider_id")
		return
	}

	h.handleSendMagicLink(w, r, &clientID, &providerID, false, startTime, sc)
}

// SendMagicLink handles internal magic-link requests; client_id/provider_id are optional.
// Mounted on the management surface (port 8080); uses the auth-facing frontend hostname.
func (h *MagicLinkHandler) SendMagicLink(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)

	var clientIDPtr, providerIDPtr *string
	if v := r.URL.Query().Get("client_id"); v != "" {
		clientIDPtr = &v
	}
	if v := r.URL.Query().Get("provider_id"); v != "" {
		providerIDPtr = &v
	}

	h.handleSendMagicLink(w, r, clientIDPtr, providerIDPtr, true, startTime, sc)
}

func (h *MagicLinkHandler) handleSendMagicLink(
	w http.ResponseWriter,
	r *http.Request,
	clientID, providerID *string,
	isInternal bool,
	startTime time.Time,
	sc securityContext,
) {
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	var req dto.SendMagicLinkRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "magic_link_invalid_json",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid JSON in request body",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "magic_link_validation_failure",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/send",
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
			EventType: "magic_link_rate_limited",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Rate limit exceeded for magic link send",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
		return
	}

	response, err := h.magicLinkService.SendMagicLink(r.Context(), req.Email, clientID, providerID, isInternal)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to send sign-in link", err)
		return
	}

	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "magic_link_request",
		UserID:    req.Email,
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/magic-link/send",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "Magic link send processed",
		Severity:  "INFO",
	})

	resp.Success(w, response, "Sign-in link sent")
}

// VerifyMagicLink consumes a magic-link token and exchanges it for a session.
// Requires client_id and provider_id query parameters (carried by the signed URL
// in the email). Issues a standard LoginResponseDTO on success.
func (h *MagicLinkHandler) VerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	clientID := r.URL.Query().Get("client_id")
	providerID := r.URL.Query().Get("provider_id")
	if clientID == "" || providerID == "" {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "magic_link_missing_params",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/verify",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Missing required client_id or provider_id parameters",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Missing required parameters: client_id and provider_id")
		return
	}

	var req dto.VerifyMagicLinkRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "magic_link_invalid_json",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/verify",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid JSON in request body",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "magic_link_validation_failure",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/verify",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request validation failed",
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	// Rate limit by client IP — the token itself is a secret so we can't safely
	// expose it as the limiter key, and there's no email at this point.
	if err := security.CheckRateLimit(clientIPStr); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "magic_link_rate_limited",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/verify",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Rate limit exceeded for magic link verify",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
		return
	}

	response, err := h.magicLinkService.LoginWithMagicLink(r.Context(), req.Token, clientID, providerID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to sign in", err)
		return
	}

	resp.Success(w, response, "Signed in")
}
