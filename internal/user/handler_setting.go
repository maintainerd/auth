package user

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type UserSettingHandler struct {
	userSettingService UserSettingService
	auditLogger        auditlog.ManagementAuditLogger
}

func NewUserSettingHandler(userSettingService UserSettingService) *UserSettingHandler {
	return &UserSettingHandler{userSettingService: userSettingService}
}

// SetAuditLogger injects the audit logger (called by the wiring layer).
func (h *UserSettingHandler) SetAuditLogger(l auditlog.ManagementAuditLogger) { h.auditLogger = l }

func (h *UserSettingHandler) logAudit(r *http.Request, tenantID int64, actorUserID *int64, action, resourceType, resourceID string, resourceUUID *uuid.UUID, changes, outcome string) {
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

func (h *UserSettingHandler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	var req UserSettingRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	user := middleware.AuthFromRequest(r).User
	userSetting, err := h.userSettingService.CreateOrUpdateUserSetting(
		r.Context(),
		user.UserUUID,
		req.Timezone, req.PreferredLanguage, req.Locale,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Save user setting failed", err)
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
	changesJSONCOU, _ := json.Marshal(map[string]any{"after": userSetting})
	settingUUIDCOU := userSetting.UserSettingUUID
	h.logAudit(r, tenantIDCOU, actorUserIDCOU, "user_setting.create_or_update", "user_setting", settingUUIDCOU.String(), &settingUUIDCOU, string(changesJSONCOU), "success")

	resp.Success(w, toUserSettingResponseDTO(*userSetting), "User setting saved successfully")
}

func (h *UserSettingHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	userSetting, err := h.userSettingService.GetByUserUUID(r.Context(), user.UserUUID)
	if err != nil || userSetting == nil {
		resp.Error(w, http.StatusNotFound, "User setting not found")
		return
	}

	resp.Success(w, toUserSettingResponseDTO(*userSetting), "User setting retrieved successfully")
}

func (h *UserSettingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User

	// First get the user setting to get its UUID
	userSetting, err := h.userSettingService.GetByUserUUID(r.Context(), user.UserUUID)
	if err != nil || userSetting == nil {
		resp.Error(w, http.StatusNotFound, "User setting not found")
		return
	}

	// Delete by user setting UUID
	deletedUserSetting, err := h.userSettingService.DeleteByUUID(r.Context(), userSetting.UserSettingUUID, user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Delete user setting failed", err)
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
	changesJSONDel, _ := json.Marshal(map[string]any{"before": map[string]any{"id": deletedUserSetting.UserSettingUUID.String()}})
	settingUUIDDel := deletedUserSetting.UserSettingUUID
	h.logAudit(r, tenantIDDel, actorUserIDDel, "user_setting.delete", "user_setting", settingUUIDDel.String(), &settingUUIDDel, string(changesJSONDel), "success")

	resp.Success(w, toUserSettingResponseDTO(*deletedUserSetting), "User setting deleted successfully")
}

// Convert service result to DTO
func toUserSettingResponseDTO(us UserSettingServiceDataResult) UserSettingResponseDTO {
	return UserSettingResponseDTO{
		UserSettingUUID:   us.UserSettingUUID.String(),
		Timezone:          us.Timezone,
		PreferredLanguage: us.PreferredLanguage,
		Locale:            us.Locale,
		CreatedAt:         us.CreatedAt,
		UpdatedAt:         us.UpdatedAt,
	}
}

func NewUserSettingResponseDTO(us *UserSetting) *UserSettingResponseDTO {
	hydrateUserSettingTransients(us)
	return &UserSettingResponseDTO{
		UserSettingUUID:   us.UserSettingUUID.String(),
		Timezone:          us.Timezone,
		PreferredLanguage: us.PreferredLanguage,
		Locale:            us.Locale,
		CreatedAt:         us.CreatedAt,
		UpdatedAt:         us.UpdatedAt,
	}
}
