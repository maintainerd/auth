package notifier

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

// EmailConfigHandler handles tenant email delivery configuration endpoints.
type EmailConfigHandler struct {
	emailConfigService EmailConfigService
}

// NewEmailConfigHandler creates a new EmailConfigHandler.
func NewEmailConfigHandler(emailConfigService EmailConfigService) *EmailConfigHandler {
	return &EmailConfigHandler{emailConfigService: emailConfigService}
}

// Get retrieves the email config for the authenticated tenant.
//
// GET /email-config
func (h *EmailConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	result, err := h.emailConfigService.Get(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get email config", err)
		return
	}

	resp.Success(w, toEmailConfigResponseDTO(result), "Email config retrieved successfully")
}

// Update upserts the email config for the authenticated tenant.
//
// PUT /email-config
func (h *EmailConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req EmailConfigUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.emailConfigService.Update(
		r.Context(), tenant.TenantID,
		req.Provider, req.Host, req.Port,
		req.Username, req.Password,
		req.FromAddress, req.FromName, req.ReplyTo,
		req.Encryption, req.LogoURL, req.TestMode,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update email config", err)
		return
	}

	resp.Success(w, toEmailConfigResponseDTO(result), "Email config updated successfully")
}

func toEmailConfigResponseDTO(ec *EmailConfigServiceDataResult) EmailConfigResponseDTO {
	return EmailConfigResponseDTO{
		EmailConfigID: ec.EmailConfigUUID.String(),
		Provider:      ec.Provider,
		Host:          ec.Host,
		Port:          ec.Port,
		Username:      ec.Username,
		FromAddress:   ec.FromAddress,
		FromName:      ec.FromName,
		ReplyTo:       ec.ReplyTo,
		Encryption:    ec.Encryption,
		LogoURL:       ec.LogoURL,
		TestMode:      ec.TestMode,
		Status:        ec.Status,
		CreatedAt:     ec.CreatedAt,
		UpdatedAt:     ec.UpdatedAt,
	}
}
