package user

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type UserSettingHandler struct {
	userSettingService UserSettingService
}

func NewUserSettingHandler(userSettingService UserSettingService) *UserSettingHandler {
	return &UserSettingHandler{userSettingService}
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
