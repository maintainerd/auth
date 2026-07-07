package user

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jsonutil"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type ProfileHandler struct {
	profileService ProfileService
	auditLogger    auditlog.ManagementAuditLogger
}

func NewProfileHandler(profileService ProfileService) *ProfileHandler {
	return &ProfileHandler{profileService: profileService}
}

// SetAuditLogger injects the audit logger (called by the wiring layer).
func (h *ProfileHandler) SetAuditLogger(l auditlog.ManagementAuditLogger) { h.auditLogger = l }

func (h *ProfileHandler) logAudit(r *http.Request, tenantID int64, actorUserID *int64, action, resourceType, resourceID string, resourceUUID *uuid.UUID, changes, outcome string) {
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

func (h *ProfileHandler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	var req ProfileRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Parse birthdate string to *time.Time (format already validated by DTO)
	var birthdate *time.Time
	if req.Birthdate != nil && *req.Birthdate != "" {
		parsed, _ := time.Parse("2006-01-02", *req.Birthdate)
		birthdate = &parsed
	}

	user := middleware.AuthFromRequest(r).User
	profile, err := h.profileService.CreateOrUpdateProfile(
		r.Context(),
		user.UserUUID,
		req.FirstName,
		req.MiddleName, req.LastName, req.DisplayName,
		birthdate,
		req.Gender,
		req.Email,
		req.Timezone, req.Language,
		req.ProfileURL,
		req.Metadata,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Save profile failed", err)
		return
	}

	tenantIDCOU := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDCOU = t.TenantID
	}
	var actorUserIDCOU *int64
	if user != nil {
		actorUserIDCOU = &user.UserID
	}
	changesJSONCOU, _ := json.Marshal(map[string]any{"after": profile})
	profileUUIDCOU := profile.ProfileUUID
	h.logAudit(r, tenantIDCOU, actorUserIDCOU, "profile.create_or_update", "profile", profileUUIDCOU.String(), &profileUUIDCOU, string(changesJSONCOU), "success")

	resp.Success(w, toProfileResponseDTO(*profile), "Profile saved successfully")
}

func (h *ProfileHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	var req ProfileRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Parse birthdate string to *time.Time (format already validated by DTO)
	var birthdate *time.Time
	if req.Birthdate != nil && *req.Birthdate != "" {
		parsed, _ := time.Parse("2006-01-02", *req.Birthdate)
		birthdate = &parsed
	}

	user := middleware.AuthFromRequest(r).User

	// Generate new UUID for the profile
	profileUUID := uuid.New()

	profile, err := h.profileService.CreateOrUpdateSpecificProfile(
		r.Context(),
		profileUUID,
		user.UserUUID,
		req.FirstName,
		req.MiddleName, req.LastName, req.DisplayName,
		birthdate,
		req.Gender,
		req.Email,
		req.Timezone, req.Language,
		req.ProfileURL,
		req.Metadata,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Create profile failed", err)
		return
	}

	tenantIDCP := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDCP = t.TenantID
	}
	var actorUserIDCP *int64
	if user != nil {
		actorUserIDCP = &user.UserID
	}
	changesJSONCP, _ := json.Marshal(map[string]any{"after": profile})
	h.logAudit(r, tenantIDCP, actorUserIDCP, "profile.create", "profile", profileUUID.String(), &profileUUID, string(changesJSONCP), "success")

	resp.Created(w, toProfileResponseDTO(*profile), "Profile created successfully")
}

func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Get profile UUID from URL parameter
	profileUUIDStr := chi.URLParam(r, "profile_uuid")
	profileUUID, err := uuid.Parse(profileUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid profile UUID")
		return
	}

	var req ProfileRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Parse birthdate string to *time.Time (format already validated by DTO)
	var birthdate *time.Time
	if req.Birthdate != nil && *req.Birthdate != "" {
		parsed, _ := time.Parse("2006-01-02", *req.Birthdate)
		birthdate = &parsed
	}

	user := middleware.AuthFromRequest(r).User
	profile, err := h.profileService.CreateOrUpdateSpecificProfile(
		r.Context(),
		profileUUID,
		user.UserUUID,
		req.FirstName,
		req.MiddleName, req.LastName, req.DisplayName,
		birthdate,
		req.Gender,
		req.Email,
		req.Timezone, req.Language,
		req.ProfileURL,
		req.Metadata,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Update profile failed", err)
		return
	}

	tenantIDUP := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDUP = t.TenantID
	}
	var actorUserIDUP *int64
	if user != nil {
		actorUserIDUP = &user.UserID
	}
	changesJSONUP, _ := json.Marshal(map[string]any{"update": req, "after": profile})
	h.logAudit(r, tenantIDUP, actorUserIDUP, "profile.update", "profile", profileUUID.String(), &profileUUID, string(changesJSONUP), "success")

	resp.Success(w, toProfileResponseDTO(*profile), "Profile updated successfully")
}

func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	profile, err := h.profileService.GetByUserUUID(r.Context(), user.UserUUID)
	if err != nil || profile == nil {
		resp.Error(w, http.StatusNotFound, "Profile not found")
		return
	}

	resp.Success(w, toProfileResponseDTO(*profile), "Profile retrieved successfully")
}

func (h *ProfileHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	q := r.URL.Query()

	// Build filter DTO
	reqParams := ProfileFilterDTO{
		FirstName:            ptr.PtrOrNil(q.Get("first_name")),
		LastName:             ptr.PtrOrNil(q.Get("last_name")),
		Email:                ptr.PtrOrNil(q.Get("email")),
		PaginationRequestDTO: pagination.ParseQuery(r),
	}

	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Get all profiles
	result, err := h.profileService.GetAll(
		r.Context(),
		user.UserUUID,
		reqParams.FirstName,
		reqParams.LastName,
		reqParams.Email,
		reqParams.Page,
		reqParams.Limit,
		reqParams.SortBy,
		reqParams.SortOrder,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to fetch profiles", err)
		return
	}

	// Map service result to dto
	rows := make([]ProfileResponseDTO, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toProfileResponseDTO(r)
	}

	// Build response data
	response := PaginatedResponseDTO[ProfileResponseDTO]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Profiles fetched successfully")
}

func (h *ProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User

	// First get the profile to get its UUID
	profile, err := h.profileService.GetByUserUUID(r.Context(), user.UserUUID)
	if err != nil || profile == nil {
		resp.Error(w, http.StatusNotFound, "Profile not found")
		return
	}

	// Delete by profile UUID
	deletedProfile, err := h.profileService.DeleteByUUID(r.Context(), profile.ProfileUUID, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Delete profile failed", err)
		return
	}

	tenantIDDel := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDDel = t.TenantID
	}
	var actorUserIDDel *int64
	if user != nil {
		actorUserIDDel = &user.UserID
	}
	changesJSONDel, _ := json.Marshal(map[string]any{"before": map[string]any{"id": deletedProfile.ProfileUUID.String()}})
	delUUID := deletedProfile.ProfileUUID
	h.logAudit(r, tenantIDDel, actorUserIDDel, "profile.delete", "profile", delUUID.String(), &delUUID, string(changesJSONDel), "success")

	resp.Success(w, toProfileResponseDTO(*deletedProfile), "Profile deleted successfully")
}

func (h *ProfileHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User

	// Get profile UUID from URL parameter
	profileUUIDStr := chi.URLParam(r, "profile_uuid")
	profileUUID, err := uuid.Parse(profileUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid profile UUID")
		return
	}

	// Get profile by UUID with ownership verification
	profile, err := h.profileService.GetByUUID(r.Context(), profileUUID, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to fetch profile", err)
		return
	}

	resp.Success(w, toProfileResponseDTO(*profile), "Profile retrieved successfully")
}

func (h *ProfileHandler) DeleteByUUID(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User

	// Get profile UUID from URL parameter
	profileUUIDStr := chi.URLParam(r, "profile_uuid")
	profileUUID, err := uuid.Parse(profileUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid profile UUID")
		return
	}

	// Delete by profile UUID with ownership verification
	deletedProfile, err := h.profileService.DeleteByUUID(r.Context(), profileUUID, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete profile", err)
		return
	}

	tenantIDDBU := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDDBU = t.TenantID
	}
	var actorUserIDDBU *int64
	if user != nil {
		actorUserIDDBU = &user.UserID
	}
	changesJSONDBU, _ := json.Marshal(map[string]any{"before": map[string]any{"id": profileUUID.String()}})
	h.logAudit(r, tenantIDDBU, actorUserIDDBU, "profile.delete", "profile", profileUUID.String(), &profileUUID, string(changesJSONDBU), "success")

	resp.Success(w, toProfileResponseDTO(*deletedProfile), "Profile deleted successfully")
}

// Admin handlers - for managing other users' profiles
func (h *ProfileHandler) AdminGetAllProfiles(w http.ResponseWriter, r *http.Request) {
	// Get user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	q := r.URL.Query()

	// Build filter DTO
	reqParams := ProfileFilterDTO{
		FirstName:            ptr.PtrOrNil(q.Get("first_name")),
		LastName:             ptr.PtrOrNil(q.Get("last_name")),
		Email:                ptr.PtrOrNil(q.Get("email")),
		PaginationRequestDTO: pagination.ParseQuery(r),
	}

	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Get all profiles for specified user
	result, err := h.profileService.GetAll(
		r.Context(),
		userUUID,
		reqParams.FirstName,
		reqParams.LastName,
		reqParams.Email,
		reqParams.Page,
		reqParams.Limit,
		reqParams.SortBy,
		reqParams.SortOrder,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to fetch profiles", err)
		return
	}

	// Map service result to dto
	rows := make([]ProfileResponseDTO, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toProfileResponseDTO(r)
	}

	// Build response data
	response := PaginatedResponseDTO[ProfileResponseDTO]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Profiles fetched successfully")
}

func (h *ProfileHandler) AdminGetProfile(w http.ResponseWriter, r *http.Request) {
	// Get user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Get profile UUID from URL parameter
	profileUUIDStr := chi.URLParam(r, "profile_uuid")
	profileUUID, err := uuid.Parse(profileUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid profile UUID")
		return
	}

	// Get profile by UUID without ownership check (admin access)
	profile, err := h.profileService.GetByUUID(r.Context(), profileUUID, userUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Profile not found", err)
		return
	}

	resp.Success(w, toProfileResponseDTO(*profile), "Profile retrieved successfully")
}

func (h *ProfileHandler) AdminCreateProfile(w http.ResponseWriter, r *http.Request) {
	// Get user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	var req ProfileRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Parse birthdate string to *time.Time (format already validated by DTO)
	var birthdate *time.Time
	if req.Birthdate != nil && *req.Birthdate != "" {
		parsed, _ := time.Parse("2006-01-02", *req.Birthdate)
		birthdate = &parsed
	}

	// Generate new UUID for the profile
	profileUUID := uuid.New()

	profile, err := h.profileService.CreateOrUpdateSpecificProfile(
		r.Context(),
		profileUUID,
		userUUID,
		req.FirstName,
		req.MiddleName, req.LastName, req.DisplayName,
		birthdate,
		req.Gender,
		req.Email,
		req.Timezone, req.Language,
		req.ProfileURL,
		req.Metadata,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Create profile failed", err)
		return
	}

	tenantIDACP := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDACP = t.TenantID
	}
	var actorUserIDACP *int64
	if u := middleware.AuthFromRequest(r).User; u != nil {
		actorUserIDACP = &u.UserID
	}
	changesJSONACP, _ := json.Marshal(map[string]any{"after": profile})
	h.logAudit(r, tenantIDACP, actorUserIDACP, "profile.admin_create", "profile", profileUUID.String(), &profileUUID, string(changesJSONACP), "success")

	resp.Created(w, toProfileResponseDTO(*profile), "Profile created successfully")
}

func (h *ProfileHandler) AdminUpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Get user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Get profile UUID from URL parameter
	profileUUIDStr := chi.URLParam(r, "profile_uuid")
	profileUUID, err := uuid.Parse(profileUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid profile UUID")
		return
	}

	var req ProfileRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Parse birthdate string to *time.Time (format already validated by DTO)
	var birthdate *time.Time
	if req.Birthdate != nil && *req.Birthdate != "" {
		parsed, _ := time.Parse("2006-01-02", *req.Birthdate)
		birthdate = &parsed
	}

	profile, err := h.profileService.CreateOrUpdateSpecificProfile(
		r.Context(),
		profileUUID,
		userUUID,
		req.FirstName,
		req.MiddleName, req.LastName, req.DisplayName,
		birthdate,
		req.Gender,
		req.Email,
		req.Timezone, req.Language,
		req.ProfileURL,
		req.Metadata,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Update profile failed", err)
		return
	}

	tenantIDAUP := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDAUP = t.TenantID
	}
	var actorUserIDAUP *int64
	if u := middleware.AuthFromRequest(r).User; u != nil {
		actorUserIDAUP = &u.UserID
	}
	changesJSONAUP, _ := json.Marshal(map[string]any{"update": req, "after": profile})
	h.logAudit(r, tenantIDAUP, actorUserIDAUP, "profile.admin_update", "profile", profileUUID.String(), &profileUUID, string(changesJSONAUP), "success")

	resp.Success(w, toProfileResponseDTO(*profile), "Profile updated successfully")
}

func (h *ProfileHandler) AdminDeleteProfile(w http.ResponseWriter, r *http.Request) {
	// Get user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Get profile UUID from URL parameter
	profileUUIDStr := chi.URLParam(r, "profile_uuid")
	profileUUID, err := uuid.Parse(profileUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid profile UUID")
		return
	}

	// Delete by profile UUID without strict ownership check (admin access)
	deletedProfile, err := h.profileService.DeleteByUUID(r.Context(), profileUUID, userUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Delete profile failed", err)
		return
	}

	tenantIDADP := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDADP = t.TenantID
	}
	var actorUserIDADP *int64
	if u := middleware.AuthFromRequest(r).User; u != nil {
		actorUserIDADP = &u.UserID
	}
	changesJSONADP, _ := json.Marshal(map[string]any{"before": map[string]any{"id": profileUUID.String()}})
	h.logAudit(r, tenantIDADP, actorUserIDADP, "profile.admin_delete", "profile", profileUUID.String(), &profileUUID, string(changesJSONADP), "success")

	resp.Success(w, toProfileResponseDTO(*deletedProfile), "Profile deleted successfully")
}

// Convert service result to DTO
func toProfileResponseDTO(p ProfileServiceDataResult) ProfileResponseDTO {
	return ProfileResponseDTO{
		ProfileUUID: p.ProfileUUID.String(),
		FirstName:   p.FirstName,
		MiddleName:  p.MiddleName,
		LastName:    p.LastName,
		DisplayName: p.DisplayName,
		Birthdate:   p.Birthdate,
		Gender:      p.Gender,
		Email:       p.Email,
		Timezone:    p.Timezone,
		Language:    p.Language,
		ProfileURL:  p.ProfileURL,
		Metadata:    p.Metadata,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func NewProfileResponseDTO(p *Profile) *ProfileResponseDTO {
	return &ProfileResponseDTO{
		ProfileUUID: p.ProfileUUID.String(),
		FirstName:   p.FirstName,
		MiddleName:  p.MiddleName,
		LastName:    p.LastName,
		DisplayName: p.DisplayName,
		Birthdate:   p.Birthdate,
		Gender:      p.Gender,
		Email:       p.Email,
		Timezone:    p.Timezone,
		Language:    p.Language,
		ProfileURL:  p.ProfileURL,
		Metadata:    jsonutil.JSONToMap(p.Metadata),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
