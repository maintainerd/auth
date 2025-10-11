package resthandler

import (
	"encoding/json"
	"net/http"
	"time"

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

	// Parse birthdate string to *time.Time
	var birthdate *time.Time
	if req.Birthdate != nil && *req.Birthdate != "" {
		parsed, err := time.Parse("2006-01-02", *req.Birthdate)
		if err != nil {
			util.Error(w, http.StatusBadRequest, "Invalid birthdate format, must be YYYY-MM-DD", err.Error())
			return
		}
		birthdate = &parsed
	}

	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	profile, err := h.profileService.CreateOrUpdateProfile(
		user.UserUUID,
		req.FirstName,
		req.MiddleName, req.LastName, req.Suffix, req.DisplayName,
		birthdate,
		req.Gender, req.Bio,
		req.Phone, req.Email,
		req.Address, req.City, req.State, req.Country, req.PostalCode,
		req.Company, req.JobTitle, req.Department, req.Industry, req.WebsiteURL,
		req.AvatarURL, req.CoverURL,
	)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Save profile failed", err.Error())
		return
	}

	util.Success(w, toProfileResponseDto(*profile), "Profile saved successfully")
}

func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	profile, err := h.profileService.GetByUserUUID(user.UserUUID)
	if err != nil || profile == nil {
		util.Error(w, http.StatusNotFound, "Profile not found")
		return
	}

	util.Success(w, toProfileResponseDto(*profile), "Profile retrieved successfully")
}

func (h *ProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// First get the profile to get its UUID
	profile, err := h.profileService.GetByUserUUID(user.UserUUID)
	if err != nil || profile == nil {
		util.Error(w, http.StatusNotFound, "Profile not found")
		return
	}

	// Delete by profile UUID
	deletedProfile, err := h.profileService.DeleteByUUID(profile.ProfileUUID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Delete profile failed", err.Error())
		return
	}

	util.Success(w, toProfileResponseDto(*deletedProfile), "Profile deleted successfully")
}

// Convert service result to DTO
func toProfileResponseDto(p service.ProfileServiceDataResult) dto.ProfileResponse {
	return dto.ProfileResponse{
		ProfileUUID: p.ProfileUUID.String(),

		// Basic Identity Information
		FirstName:   p.FirstName,
		MiddleName:  p.MiddleName,
		LastName:    p.LastName,
		Suffix:      p.Suffix,
		DisplayName: p.DisplayName,

		// Personal Information
		Birthdate: p.Birthdate,
		Gender:    p.Gender,
		Bio:       p.Bio,

		// Contact Information
		Phone: p.Phone,
		Email: p.Email,

		// Address Information
		Address:    p.Address,
		City:       p.City,
		State:      p.State,
		Country:    p.Country,
		PostalCode: p.PostalCode,

		// Professional Information
		Company:    p.Company,
		JobTitle:   p.JobTitle,
		Department: p.Department,
		Industry:   p.Industry,
		WebsiteURL: p.WebsiteURL,

		// Media & Assets
		AvatarURL: p.AvatarURL,
		CoverURL:  p.CoverURL,

		// System Fields
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
