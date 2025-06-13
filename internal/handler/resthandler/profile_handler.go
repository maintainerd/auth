package resthandler

import (
	"encoding/json"
	"net/http"

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

func (h *ProfileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.ProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	profile, err := h.profileService.CreateProfile(user.UserID, &req)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Create profile failed", err.Error())
		return
	}

	util.Created(w, dto.NewProfileResponse(profile), "Profile created successfully")
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

func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req dto.ProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	profile, err := h.profileService.UpdateProfile(user.UserID, &req)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Update profile failed", err.Error())
		return
	}

	util.Success(w, dto.NewProfileResponse(profile), "Profile updated successfully")
}

func (h *ProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	if err := h.profileService.DeleteProfile(user.UserID); err != nil {
		util.Error(w, http.StatusBadRequest, "Delete profile failed", err.Error())
		return
	}

	util.Success(w, nil, "Profile deleted successfully")
}
