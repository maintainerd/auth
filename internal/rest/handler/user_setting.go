package handler

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/service"
	resp "github.com/maintainerd/auth/internal/rest/response"
)

type UserSettingHandler struct {
	userSettingService service.UserSettingService
}

func NewUserSettingHandler(userSettingService service.UserSettingService) *UserSettingHandler {
	return &UserSettingHandler{userSettingService}
}

func (h *UserSettingHandler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	var req dto.UserSettingRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Convert SocialLinks from map[string]string to map[string]any
	var socialLinks map[string]any
	if req.SocialLinks != nil {
		socialLinks = make(map[string]any)
		for k, v := range req.SocialLinks {
			socialLinks[k] = v
		}
	}

	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	userSetting, err := h.userSettingService.CreateOrUpdateUserSetting(
		user.UserUUID,
		req.Timezone, req.PreferredLanguage, req.Locale,
		socialLinks,
		req.PreferredContactMethod,
		req.MarketingEmailConsent, req.SMSNotificationsConsent, req.PushNotificationsConsent,
		req.ProfileVisibility,
		req.DataProcessingConsent,
		nil, nil, // termsAcceptedAt, privacyPolicyAcceptedAt - not in DTO
		req.EmergencyContactName, req.EmergencyContactPhone, req.EmergencyContactEmail, req.EmergencyContactRelation,
	)
	if err != nil {
		resp.HandleServiceError(w, "Save user setting failed", err)
		return
	}

	resp.Success(w, toUserSettingResponseDTO(*userSetting), "User setting saved successfully")
}

func (h *UserSettingHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	userSetting, err := h.userSettingService.GetByUserUUID(user.UserUUID)
	if err != nil || userSetting == nil {
		resp.Error(w, http.StatusNotFound, "User setting not found")
		return
	}

	resp.Success(w, toUserSettingResponseDTO(*userSetting), "User setting retrieved successfully")
}

func (h *UserSettingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// First get the user setting to get its UUID
	userSetting, err := h.userSettingService.GetByUserUUID(user.UserUUID)
	if err != nil || userSetting == nil {
		resp.Error(w, http.StatusNotFound, "User setting not found")
		return
	}

	// Delete by user setting UUID
	deletedUserSetting, err := h.userSettingService.DeleteByUUID(userSetting.UserSettingUUID)
	if err != nil {
		resp.HandleServiceError(w, "Delete user setting failed", err)
		return
	}

	resp.Success(w, toUserSettingResponseDTO(*deletedUserSetting), "User setting deleted successfully")
}

// Convert service result to DTO
func toUserSettingResponseDTO(us service.UserSettingServiceDataResult) dto.UserSettingResponseDTO {
	// Convert GORM JSON to map for social links
	var socialLinks map[string]any
	if len(us.SocialLinks) > 0 {
		if err := json.Unmarshal(us.SocialLinks, &socialLinks); err != nil {
			socialLinks = nil
		}
	}

	return dto.UserSettingResponseDTO{
		UserSettingUUID: us.UserSettingUUID.String(),

		// Internationalization
		Timezone:          us.Timezone,
		PreferredLanguage: us.PreferredLanguage,
		Locale:            us.Locale,

		// Social Media & External Links
		SocialLinks: socialLinks,

		// Communication Preferences
		PreferredContactMethod:   us.PreferredContactMethod,
		MarketingEmailConsent:    us.MarketingEmailConsent,
		SMSNotificationsConsent:  us.SMSNotificationsConsent,
		PushNotificationsConsent: us.PushNotificationsConsent,

		// Privacy & Compliance
		ProfileVisibility:       us.ProfileVisibility,
		DataProcessingConsent:   us.DataProcessingConsent,
		TermsAcceptedAt:         us.TermsAcceptedAt,
		PrivacyPolicyAcceptedAt: us.PrivacyPolicyAcceptedAt,

		// Emergency Contact
		EmergencyContactName:     us.EmergencyContactName,
		EmergencyContactPhone:    us.EmergencyContactPhone,
		EmergencyContactEmail:    us.EmergencyContactEmail,
		EmergencyContactRelation: us.EmergencyContactRelation,

		// System Fields
		CreatedAt: us.CreatedAt,
		UpdatedAt: us.UpdatedAt,
	}
}
