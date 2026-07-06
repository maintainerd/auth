package user

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type UserConsentHandler struct {
	consentService UserConsentService
	userService    UserService
	userRepo       UserRepository
}

func NewUserConsentHandler(consentService UserConsentService, userService UserService, userRepo UserRepository) *UserConsentHandler {
	return &UserConsentHandler{
		consentService: consentService,
		userService:    userService,
		userRepo:       userRepo,
	}
}

type RecordConsentRequestDTO struct {
	ConsentType   string `json:"consent_type"`
	PolicyVersion string `json:"policy_version"`
}

func (r *RecordConsentRequestDTO) Validate() error {
	if r.ConsentType == "" {
		return errConsentTypeRequired
	}
	if r.PolicyVersion == "" {
		return errPolicyVersionRequired
	}
	switch r.ConsentType {
	case "terms_of_service", "privacy_policy", "data_processing":
	default:
		return errInvalidConsentType
	}
	return nil
}

type UserConsentResponseDTO struct {
	UUID          string `json:"uuid"`
	ConsentType   string `json:"consent_type"`
	PolicyVersion string `json:"policy_version"`
	Accepted      bool   `json:"accepted"`
	IPAddress     string `json:"ip_address,omitempty"`
	UserAgent     string `json:"user_agent,omitempty"`
	CreatedAt     string `json:"created_at"`
}

// RecordConsent records a user's consent to a policy (self-service).
//
// POST /me/consent
func (h *UserConsentHandler) RecordConsent(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req RecordConsentRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		resp.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	clientIP := middleware.ClientIPFromContext(r.Context())
	userAgent := r.UserAgent()

	if err := h.consentService.Record(r.Context(), nil, auth.User.UserID, auth.Tenant.TenantID, req.ConsentType, req.PolicyVersion, clientIP, userAgent); err != nil {
		resp.Error(w, http.StatusInternalServerError, "Failed to record consent")
		return
	}

	resp.Success(w, map[string]string{"status": "recorded"}, "Consent recorded successfully")
}

// GetUserConsents returns all consents for a user (admin).
//
// GET /users/{user_uuid}/consents
func (h *UserConsentHandler) GetUserConsents(w http.ResponseWriter, r *http.Request) {
	userUUID := chi.URLParam(r, "user_uuid")
	if userUUID == "" {
		resp.Error(w, http.StatusBadRequest, "Missing user_uuid")
		return
	}

	u, err := h.userRepo.FindByUUID(userUUID)
	if err != nil || u == nil {
		resp.Error(w, http.StatusNotFound, "User not found")
		return
	}

	consents, err := h.consentService.FindByUserID(r.Context(), u.UserID)
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, "Failed to retrieve consents")
		return
	}

	dtos := make([]UserConsentResponseDTO, 0, len(consents))
	for _, c := range consents {
		dto := UserConsentResponseDTO{
			UUID:          c.UserConsentUUID.String(),
			ConsentType:   c.ConsentType,
			PolicyVersion: c.PolicyVersion,
			Accepted:      c.Accepted,
			CreatedAt:     c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if c.IPAddress != nil {
			dto.IPAddress = *c.IPAddress
		}
		if c.UserAgent != nil {
			dto.UserAgent = *c.UserAgent
		}
		dtos = append(dtos, dto)
	}

	resp.Success(w, dtos, "User consents fetched successfully")
}
