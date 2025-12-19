package resthandler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"

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
		req.MiddleName, req.LastName, req.Suffix, req.DisplayName, req.Bio,
		birthdate,
		req.Gender,
		req.Phone, req.Email, req.Address,
		req.City, req.Country,
		req.Timezone, req.Language,
		req.ProfileURL,
		req.Metadata,
	)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Save profile failed", err.Error())
		return
	}

	util.Success(w, toProfileResponseDto(*profile), "Profile saved successfully")
}

func (h *ProfileHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
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

	// Generate new UUID for the profile
	profileUUID := uuid.New()

	profile, err := h.profileService.CreateOrUpdateSpecificProfile(
		profileUUID,
		user.UserUUID,
		req.FirstName,
		req.MiddleName, req.LastName, req.Suffix, req.DisplayName, req.Bio,
		birthdate,
		req.Gender,
		req.Phone, req.Email, req.Address,
		req.City, req.Country,
		req.Timezone, req.Language,
		req.ProfileURL,
		req.Metadata,
	)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Create profile failed", err.Error())
		return
	}

	util.Created(w, toProfileResponseDto(*profile), "Profile created successfully")
}

func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Get profile UUID from URL parameter
	profileUUIDStr := chi.URLParam(r, "profile_uuid")
	profileUUID, err := uuid.Parse(profileUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid profile UUID", err.Error())
		return
	}

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
	profile, err := h.profileService.CreateOrUpdateSpecificProfile(
		profileUUID,
		user.UserUUID,
		req.FirstName,
		req.MiddleName, req.LastName, req.Suffix, req.DisplayName, req.Bio,
		birthdate,
		req.Gender,
		req.Phone, req.Email, req.Address,
		req.City, req.Country,
		req.Timezone, req.Language,
		req.ProfileURL,
		req.Metadata,
	)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Update profile failed", err.Error())
		return
	}

	util.Success(w, toProfileResponseDto(*profile), "Profile updated successfully")
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

func (h *ProfileHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Build filter DTO
	var isDefault *bool
	if v := q.Get("is_default"); v != "" {
		if v == "true" {
			trueVal := true
			isDefault = &trueVal
		} else if v == "false" {
			falseVal := false
			isDefault = &falseVal
		}
	}

	reqParams := dto.ProfileFilterDto{
		FirstName: util.PtrOrNil(q.Get("first_name")),
		LastName:  util.PtrOrNil(q.Get("last_name")),
		Email:     util.PtrOrNil(q.Get("email")),
		Phone:     util.PtrOrNil(q.Get("phone")),
		City:      util.PtrOrNil(q.Get("city")),
		Country:   util.PtrOrNil(q.Get("country")),
		IsDefault: isDefault,
		PaginationRequestDto: dto.PaginationRequestDto{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
	}

	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Get all profiles
	result, err := h.profileService.GetAll(
		user.UserUUID,
		reqParams.FirstName,
		reqParams.LastName,
		reqParams.Email,
		reqParams.Phone,
		reqParams.City,
		reqParams.Country,
		reqParams.IsDefault,
		reqParams.Page,
		reqParams.Limit,
		reqParams.SortBy,
		reqParams.SortOrder,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch profiles", err.Error())
		return
	}

	// Map service result to dto
	rows := make([]dto.ProfileResponse, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toProfileResponseDto(r)
	}

	// Build response data
	response := dto.PaginatedResponseDto[dto.ProfileResponse]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Profiles fetched successfully")
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
	deletedProfile, err := h.profileService.DeleteByUUID(profile.ProfileUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Delete profile failed", err.Error())
		return
	}

	util.Success(w, toProfileResponseDto(*deletedProfile), "Profile deleted successfully")
}

func (h *ProfileHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Get profile UUID from URL parameter
	profileUUIDStr := chi.URLParam(r, "profile_uuid")
	profileUUID, err := uuid.Parse(profileUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid profile UUID", err.Error())
		return
	}

	// Get profile by UUID with ownership verification
	profile, err := h.profileService.GetByUUID(profileUUID, user.UserUUID)
	if err != nil {
		if err.Error() == "profile does not belong to user" {
			util.Error(w, http.StatusForbidden, "Access denied", "You don't have permission to access this profile")
			return
		}
		util.Error(w, http.StatusNotFound, "Profile not found")
		return
	}

	util.Success(w, toProfileResponseDto(*profile), "Profile retrieved successfully")
}

func (h *ProfileHandler) DeleteByUUID(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Get profile UUID from URL parameter
	profileUUIDStr := chi.URLParam(r, "profile_uuid")
	profileUUID, err := uuid.Parse(profileUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid profile UUID", err.Error())
		return
	}

	// Delete by profile UUID with ownership verification
	deletedProfile, err := h.profileService.DeleteByUUID(profileUUID, user.UserUUID)
	if err != nil {
		if err.Error() == "profile does not belong to user" {
			util.Error(w, http.StatusForbidden, "Access denied", "You don't have permission to delete this profile")
			return
		}
		util.Error(w, http.StatusBadRequest, "Delete profile failed", err.Error())
		return
	}

	util.Success(w, toProfileResponseDto(*deletedProfile), "Profile deleted successfully")
}

// Admin handlers - for managing other users' profiles
func (h *ProfileHandler) AdminGetAllProfiles(w http.ResponseWriter, r *http.Request) {
	// Get user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID", err.Error())
		return
	}

	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Build filter DTO
	var isDefault *bool
	if v := q.Get("is_default"); v != "" {
		if v == "true" {
			trueVal := true
			isDefault = &trueVal
		} else if v == "false" {
			falseVal := false
			isDefault = &falseVal
		}
	}

	reqParams := dto.ProfileFilterDto{
		FirstName: util.PtrOrNil(q.Get("first_name")),
		LastName:  util.PtrOrNil(q.Get("last_name")),
		Email:     util.PtrOrNil(q.Get("email")),
		Phone:     util.PtrOrNil(q.Get("phone")),
		City:      util.PtrOrNil(q.Get("city")),
		Country:   util.PtrOrNil(q.Get("country")),
		IsDefault: isDefault,
		PaginationRequestDto: dto.PaginationRequestDto{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
	}

	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Get all profiles for specified user
	result, err := h.profileService.GetAll(
		userUUID,
		reqParams.FirstName,
		reqParams.LastName,
		reqParams.Email,
		reqParams.Phone,
		reqParams.City,
		reqParams.Country,
		reqParams.IsDefault,
		reqParams.Page,
		reqParams.Limit,
		reqParams.SortBy,
		reqParams.SortOrder,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch profiles", err.Error())
		return
	}

	// Map service result to dto
	rows := make([]dto.ProfileResponse, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toProfileResponseDto(r)
	}

	// Build response data
	response := dto.PaginatedResponseDto[dto.ProfileResponse]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Profiles fetched successfully")
}

func (h *ProfileHandler) AdminGetProfile(w http.ResponseWriter, r *http.Request) {
	// Get user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID", err.Error())
		return
	}

	// Get profile UUID from URL parameter
	profileUUIDStr := chi.URLParam(r, "profile_uuid")
	profileUUID, err := uuid.Parse(profileUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid profile UUID", err.Error())
		return
	}

	// Get profile by UUID without ownership check (admin access)
	profile, err := h.profileService.GetByUUID(profileUUID, userUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Profile not found")
		return
	}

	util.Success(w, toProfileResponseDto(*profile), "Profile retrieved successfully")
}

func (h *ProfileHandler) AdminCreateProfile(w http.ResponseWriter, r *http.Request) {
	// Get user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID", err.Error())
		return
	}

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

	// Generate new UUID for the profile
	profileUUID := uuid.New()

	profile, err := h.profileService.CreateOrUpdateSpecificProfile(
		profileUUID,
		userUUID,
		req.FirstName,
		req.MiddleName, req.LastName, req.Suffix, req.DisplayName, req.Bio,
		birthdate,
		req.Gender,
		req.Phone, req.Email, req.Address,
		req.City, req.Country,
		req.Timezone, req.Language,
		req.ProfileURL,
		req.Metadata,
	)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Create profile failed", err.Error())
		return
	}

	util.Created(w, toProfileResponseDto(*profile), "Profile created successfully")
}

func (h *ProfileHandler) AdminUpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Get user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID", err.Error())
		return
	}

	// Get profile UUID from URL parameter
	profileUUIDStr := chi.URLParam(r, "profile_uuid")
	profileUUID, err := uuid.Parse(profileUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid profile UUID", err.Error())
		return
	}

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

	profile, err := h.profileService.CreateOrUpdateSpecificProfile(
		profileUUID,
		userUUID,
		req.FirstName,
		req.MiddleName, req.LastName, req.Suffix, req.DisplayName, req.Bio,
		birthdate,
		req.Gender,
		req.Phone, req.Email, req.Address,
		req.City, req.Country,
		req.Timezone, req.Language,
		req.ProfileURL,
		req.Metadata,
	)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Update profile failed", err.Error())
		return
	}

	util.Success(w, toProfileResponseDto(*profile), "Profile updated successfully")
}

func (h *ProfileHandler) AdminDeleteProfile(w http.ResponseWriter, r *http.Request) {
	// Get user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID", err.Error())
		return
	}

	// Get profile UUID from URL parameter
	profileUUIDStr := chi.URLParam(r, "profile_uuid")
	profileUUID, err := uuid.Parse(profileUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid profile UUID", err.Error())
		return
	}

	// Delete by profile UUID without strict ownership check (admin access)
	deletedProfile, err := h.profileService.DeleteByUUID(profileUUID, userUUID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Delete profile failed", err.Error())
		return
	}

	util.Success(w, toProfileResponseDto(*deletedProfile), "Profile deleted successfully")
}

func (h *ProfileHandler) SetDefaultProfile(w http.ResponseWriter, r *http.Request) {
	// Get profile UUID from URL parameter
	profileUUIDStr := chi.URLParam(r, "profile_uuid")
	profileUUID, err := uuid.Parse(profileUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid profile UUID", err.Error())
		return
	}

	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Set profile as default with ownership verification
	profile, err := h.profileService.SetDefaultProfile(profileUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Set default profile failed", err.Error())
		return
	}

	util.Success(w, toProfileResponseDto(*profile), "Profile set as default successfully")
}

func (h *ProfileHandler) AdminSetDefaultProfile(w http.ResponseWriter, r *http.Request) {
	// Get user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID", err.Error())
		return
	}

	// Get profile UUID from URL parameter
	profileUUIDStr := chi.URLParam(r, "profile_uuid")
	profileUUID, err := uuid.Parse(profileUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid profile UUID", err.Error())
		return
	}

	// Set profile as default without strict ownership check (admin access)
	profile, err := h.profileService.SetDefaultProfile(profileUUID, userUUID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Set default profile failed", err.Error())
		return
	}

	util.Success(w, toProfileResponseDto(*profile), "Profile set as default successfully")
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
		Bio:         p.Bio,

		// Personal Information
		Birthdate: p.Birthdate,
		Gender:    p.Gender,

		// Contact Information
		Phone:   p.Phone,
		Email:   p.Email,
		Address: p.Address,

		// Location Information
		City:    p.City,
		Country: p.Country,

		// Preference
		Timezone: p.Timezone,
		Language: p.Language,

		// Media & Assets (auth-centric)
		ProfileURL: p.ProfileURL,

		// Profile Flags
		IsDefault: p.IsDefault,

		// Extended data
		Metadata: p.Metadata,

		// System Fields
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
