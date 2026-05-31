package authn

import (
	"encoding/json"
	"net/http"

	resp "github.com/maintainerd/auth/internal/platform/response"
)

// SMSLoginHandler handles SMS one-time-code login flows.
type SMSLoginHandler struct {
	smsLoginService SMSLoginService
}

func NewSMSLoginHandler(smsLoginService SMSLoginService) *SMSLoginHandler {
	return &SMSLoginHandler{smsLoginService: smsLoginService}
}

// SendOTP sends a one-time SMS code to the given phone number.
//
// POST /sms-login/send
func (h *SMSLoginHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req SMSLoginSendDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	if err := h.smsLoginService.SendOTP(r.Context(), req); err != nil {
		resp.HandleServiceError(w, r, "Failed to send OTP", err)
		return
	}

	resp.Success(w, nil, "If a matching account exists, a verification code has been sent")
}

// VerifyOTP validates the submitted OTP and returns tokens on success.
//
// POST /sms-login/verify
func (h *SMSLoginHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req SMSLoginVerifyDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	tokens, err := h.smsLoginService.VerifyOTP(r.Context(), req)
	if err != nil {
		resp.HandleServiceError(w, r, "OTP verification failed", err)
		return
	}

	resp.Success(w, tokens, "Authenticated successfully")
}
