package user

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type UserConsentHandler struct {
	consentService UserConsentService
	userService    UserService
	userRepo       UserRepository
	auditLogger    auditlog.ManagementAuditLogger
}

func NewUserConsentHandler(consentService UserConsentService, userService UserService, userRepo UserRepository) *UserConsentHandler {
	return &UserConsentHandler{
		consentService: consentService,
		userService:    userService,
		userRepo:       userRepo,
	}
}

// SetAuditLogger injects the audit logger (called by the wiring layer).
func (h *UserConsentHandler) SetAuditLogger(l auditlog.ManagementAuditLogger) { h.auditLogger = l }

func (h *UserConsentHandler) logAudit(r *http.Request, tenantID int64, actorUserID *int64, action, resourceType, resourceID string, resourceUUID *uuid.UUID, changes, outcome string) {
	if h.auditLogger == nil {
		return
	}
	_ = h.auditLogger.Log(r.Context(), auditlog.LogEntry{
		TenantID:     tenantID,
		ActorUserID:  actorUserID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceUUID: resourceUUID,
		Changes:      changes,
		Outcome:      outcome,
	})
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
		resp.ValidationError(w, err)
		return
	}

	clientIP := middleware.ClientIPFromContext(r.Context())
	userAgent := r.UserAgent()

	if err := h.consentService.Record(r.Context(), nil, auth.User.UserID, auth.Tenant.TenantID, req.ConsentType, req.PolicyVersion, clientIP, userAgent); err != nil {
		resp.Error(w, http.StatusInternalServerError, "Failed to record consent")
		return
	}

	tenantIDRC := int64(0)
	if auth.Tenant != nil {
		tenantIDRC = auth.Tenant.TenantID
	}
	actorUserIDRC := &auth.User.UserID
	changesJSONRC, _ := json.Marshal(map[string]any{"after": req})
	h.logAudit(r, tenantIDRC, actorUserIDRC, "consent.record", "user_consent", "", nil, string(changesJSONRC), "success")

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

	if _, err := uuid.Parse(userUUID); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user_uuid format")
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

type WithdrawConsentRequestDTO struct {
	ConsentType string `json:"consent_type"`
}

// WithdrawUserConsent records a withdrawal of a user's consent (admin). The
// original grant is preserved; a withdrawal row is appended (GDPR Art. 7(3)).
//
// POST /users/{user_uuid}/consents/withdraw
func (h *UserConsentHandler) WithdrawUserConsent(w http.ResponseWriter, r *http.Request) {
	userUUID := chi.URLParam(r, "user_uuid")
	if _, err := uuid.Parse(userUUID); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user_uuid format")
		return
	}

	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req WithdrawConsentRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	switch req.ConsentType {
	case "terms_of_service", "privacy_policy", "data_processing":
	default:
		resp.Error(w, http.StatusBadRequest, "Invalid consent_type")
		return
	}

	u, err := h.userRepo.FindByUUID(userUUID, "UserIdentities")
	if err != nil || u == nil {
		resp.Error(w, http.StatusNotFound, "User not found")
		return
	}
	// Tenant isolation: the target user must belong to the requesting tenant.
	if !userHasTenantAccess(u, tenant.TenantID) {
		resp.Error(w, http.StatusNotFound, "User not found")
		return
	}

	clientIP := middleware.ClientIPFromContext(r.Context())
	userAgent := r.UserAgent()
	if err := h.consentService.Withdraw(r.Context(), u.UserID, tenant.TenantID, req.ConsentType, clientIP, userAgent); err != nil {
		resp.Error(w, http.StatusInternalServerError, "Failed to withdraw consent")
		return
	}

	var actorUserID *int64
	if authUser := middleware.AuthFromRequest(r).User; authUser != nil {
		actorUserID = &authUser.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"withdraw": map[string]any{"consent_type": req.ConsentType}})
	userUUIDRef := u.UserUUID
	h.logAudit(r, tenant.TenantID, actorUserID, "consent.withdraw", "user_consent", userUUID, &userUUIDRef, string(changesJSON), "success")

	resp.Success(w, nil, "Consent withdrawn successfully")
}
