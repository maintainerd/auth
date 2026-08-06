package authn

import (
	"encoding/json"
	"net/http"

	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// SMSLoginHandler handles SMS one-time-code login flows.
type SMSLoginHandler struct {
	smsLoginService SMSLoginService
}

func NewSMSLoginHandler(smsLoginService SMSLoginService) *SMSLoginHandler {
	return &SMSLoginHandler{smsLoginService: smsLoginService}
}

// SendOTPPublic sends a one-time SMS code to the given phone number (public surface).
//
// POST /sms-login/send?client_id=xxx
func (h *SMSLoginHandler) SendOTPPublic(w http.ResponseWriter, r *http.Request) {
	clientID, tenantID, ok := authenticationContextQuery(r)
	if !ok {
		resp.Error(w, http.StatusBadRequest, "Public SMS login requires client_id and does not accept tenant_id")
		return
	}

	var req SMSLoginSendDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	if err := h.smsLoginService.SendOTP(r.Context(), req.Phone, clientID, tenantID); err != nil {
		resp.HandleServiceError(w, r, "Failed to send OTP", err)
		return
	}

	resp.Success(w, nil, "If a matching account exists, a verification code has been sent")
}

// VerifyOTPPublic validates the submitted OTP and returns tokens on success (public surface).
//
// POST /sms-login/verify?client_id=xxx
func (h *SMSLoginHandler) VerifyOTPPublic(w http.ResponseWriter, r *http.Request) {
	clientID, tenantID, ok := authenticationContextQuery(r)
	if !ok {
		resp.Error(w, http.StatusBadRequest, "Public SMS OTP verification requires client_id and does not accept tenant_id")
		return
	}

	var req SMSLoginVerifyDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	tokens, err := h.smsLoginService.VerifyOTP(r.Context(), req.Phone, req.OTP, clientID, tenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "OTP verification failed", err)
		return
	}

	resp.Success(w, tokens, "Authenticated successfully")
}
