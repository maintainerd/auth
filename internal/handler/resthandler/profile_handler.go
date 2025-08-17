package resthandler

import (
	"encoding/json"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

type ProfileHandler struct {
	profileService service.ProfileService
}

func NewProfileHandler(profileService service.ProfileService) *ProfileHandler {
	return &ProfileHandler{profileService}
}

func (h *ProfileHandler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	var req dto.ProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		if ve, ok := err.(validation.Errors); ok {
			util.Error(w, http.StatusBadRequest, "Validation failed", ve)
			return
		}
		util.Error(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	profile, err := h.profileService.CreateOrUpdateProfile(user.UserID, &req)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Save profile failed", err.Error())
		return
	}

	util.Success(w, dto.NewProfileResponse(profile), "Profile saved successfully")
}

func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	profile, err := h.profileService.GetProfileByUserID(user.UserID)
	if err != nil || profile == nil {
		util.Error(w, http.StatusNotFound, "Profile not found")
		return
	}

	util.Success(w, dto.NewProfileResponse(profile), "Profile retrieved successfully")
}

func (h *ProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	if err := h.profileService.DeleteProfile(user.UserID); err != nil {
		util.Error(w, http.StatusBadRequest, "Delete profile failed", err.Error())
		return
	}

	util.Success(w, nil, "Profile deleted successfully")
}
