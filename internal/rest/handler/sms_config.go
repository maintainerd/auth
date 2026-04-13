package handler

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	resp "github.com/maintainerd/auth/internal/rest/response"
	"github.com/maintainerd/auth/internal/service"
)

// SMSConfigHandler handles tenant SMS delivery configuration endpoints.
type SMSConfigHandler struct {
	smsConfigService service.SMSConfigService
}

// NewSMSConfigHandler creates a new SMSConfigHandler.
func NewSMSConfigHandler(smsConfigService service.SMSConfigService) *SMSConfigHandler {
	return &SMSConfigHandler{smsConfigService: smsConfigService}
}

// Get retrieves the SMS config for the authenticated tenant.
//
// GET /sms-config
func (h *SMSConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	result, err := h.smsConfigService.Get(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get SMS config", err)
		return
	}

	resp.Success(w, toSMSConfigResponseDTO(result), "SMS config retrieved successfully")
}

// Update upserts the SMS config for the authenticated tenant.
//
// PUT /sms-config
func (h *SMSConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req dto.SMSConfigUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.smsConfigService.Update(
		r.Context(), tenant.TenantID,
		req.Provider, req.AccountSID, req.AuthToken,
		req.FromNumber, req.SenderID, req.TestMode,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update SMS config", err)
		return
	}

	resp.Success(w, toSMSConfigResponseDTO(result), "SMS config updated successfully")
}

func toSMSConfigResponseDTO(sc *service.SMSConfigServiceDataResult) dto.SMSConfigResponseDTO {
	return dto.SMSConfigResponseDTO{
		SMSConfigID: sc.SMSConfigUUID.String(),
		Provider:    sc.Provider,
		AccountSID:  sc.AccountSID,
		FromNumber:  sc.FromNumber,
		SenderID:    sc.SenderID,
		TestMode:    sc.TestMode,
		Status:      sc.Status,
		CreatedAt:   sc.CreatedAt,
		UpdatedAt:   sc.UpdatedAt,
	}
}
