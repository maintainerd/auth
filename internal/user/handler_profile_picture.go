package user

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// userReadPermission is what the admin console holds to view other people's
// users; it is the marker for "may see an avatar that is not mine".
const userReadPermission = "user:read"

// uploadFormField is the multipart field an avatar arrives in.
const uploadFormField = "file"

// UploadPicture stores an uploaded avatar on the caller's own profile.
//
// The body is bounded by MaxBytesReader BEFORE anything reads it, so an
// oversized upload is refused as it arrives. Checking the size only after
// reading would still require buffering the whole thing first, which is the
// cheap way to exhaust memory on a service with a million users.
func (h *ProfileHandler) UploadPicture(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "No valid authentication found")
		return
	}
	profileUUID, err := uuid.Parse(chi.URLParam(r, "profile_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid profile UUID")
		return
	}

	// +1 KiB of headroom for the multipart envelope, so a file exactly at the
	// limit is not rejected for its boundary and headers.
	r.Body = http.MaxBytesReader(w, r.Body, MaxProfilePictureBytes+1024)
	if err := r.ParseMultipartForm(MaxProfilePictureBytes + 1024); err != nil {
		// Distinguish "too big" from "not a multipart body at all". Reporting
		// every parse failure as 413 sent operators hunting for an oversized
		// image when the real fault was a client that had labelled the request
		// application/json and so never included a multipart boundary — a
		// misdiagnosis that costs far more than the branch.
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			resp.Error(w, http.StatusRequestEntityTooLarge,
				"The image is larger than the "+strconv.Itoa(MaxProfilePictureBytes>>20)+" MiB limit")
			return
		}
		resp.Error(w, http.StatusBadRequest,
			"The upload could not be read as a multipart form. Send the image as multipart/form-data in the \""+uploadFormField+"\" field.")
		return
	}
	defer func() {
		// ParseMultipartForm spills to temp files above its memory budget; without
		// this they survive the request.
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, _, err := r.FormFile(uploadFormField)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "No file was uploaded in the \""+uploadFormField+"\" field")
		return
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, MaxProfilePictureBytes+1))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "The uploaded file could not be read")
		return
	}

	// Decoding establishes the type. The multipart Content-Type and the filename
	// are both attacker-chosen and deliberately ignored.
	picture, err := DecodeProfilePicture(data)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to upload profile picture", err)
		return
	}
	pictureURL, err := h.profileService.SetProfilePicture(r.Context(), profileUUID, auth.User.UserID, picture)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to upload profile picture", err)
		return
	}
	resp.Success(w, map[string]any{"profile_url": pictureURL}, "Profile picture updated")
}

// DeletePicture removes an uploaded avatar from the caller's own profile.
func (h *ProfileHandler) DeletePicture(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "No valid authentication found")
		return
	}
	profileUUID, err := uuid.Parse(chi.URLParam(r, "profile_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid profile UUID")
		return
	}
	if err := h.profileService.ClearProfilePicture(r.Context(), profileUUID, auth.User.UserID); err != nil {
		resp.HandleServiceError(w, r, "Failed to remove profile picture", err)
		return
	}
	resp.Success(w, nil, "Profile picture removed")
}

// GetPicture serves a stored avatar.
//
// A conditional request is answered from the ETag alone, without reading the
// image — an avatar is rendered on every page that shows the user, so the
// common case has to cost a small row read rather than up to 2 MiB.
//
// The response is also deliberately hostile to being interpreted as anything
// but an image: nosniff stops a browser second-guessing the Content-Type, and a
// restrictive CSP neuters any active content that survived the decode check. So
// a crafted upload cannot become script on this origin even if the format
// checks are one day bypassed.
func (h *ProfileHandler) GetPicture(w http.ResponseWriter, r *http.Request) {
	profileUUID, err := uuid.Parse(chi.URLParam(r, "profile_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid profile UUID")
		return
	}

	// The route permission is account:profile:read:self — "your own profile" —
	// but a UUID in the path can name anyone's. Without this the endpoint served
	// any avatar to any authenticated caller, which is not what that permission
	// says. The admin console legitimately renders other people's avatars, so a
	// caller who administers users is allowed through as well.
	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "No valid authentication found")
		return
	}
	if !middleware.UserHasPermission(auth.User, userReadPermission) {
		if err := h.profileService.EnsureProfileOwnedBy(r.Context(), profileUUID, auth.User.UserID); err != nil {
			// Not-found, never forbidden: a caller probing UUIDs must not learn
			// which ones exist.
			resp.Error(w, http.StatusNotFound, "Profile picture not found")
			return
		}
	}

	if inm := strings.TrimSpace(r.Header.Get("If-None-Match")); inm != "" {
		etag, err := h.profileService.GetProfilePictureETag(r.Context(), profileUUID)
		if err == nil && etagMatches(inm, etag) {
			writePictureCacheHeaders(w, etag)
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	picture, err := h.profileService.GetProfilePicture(r.Context(), profileUUID)
	if err != nil {
		var notFound *apperror.NotFoundError
		if errors.As(err, &notFound) {
			resp.Error(w, http.StatusNotFound, "Profile picture not found")
			return
		}
		resp.HandleServiceError(w, r, "Failed to load profile picture", err)
		return
	}

	writePictureCacheHeaders(w, picture.ETag)
	w.Header().Set("Content-Type", picture.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(picture.Data)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(picture.Data)
}

func writePictureCacheHeaders(w http.ResponseWriter, etag string) {
	if etag != "" {
		w.Header().Set("ETag", `"`+etag+`"`)
	}
	// private: an avatar is visible to anyone who can see the profile, but it is
	// still personal data and does not belong in a shared proxy cache.
	// must-revalidate keeps a replaced avatar from lingering: the client asks
	// again, and the ETag makes that ask cheap.
	w.Header().Set("Cache-Control", "private, max-age=300, must-revalidate")
}

// etagMatches compares an If-None-Match header against the stored validator,
// tolerating the quoting and weak prefix a client may send.
func etagMatches(header, etag string) bool {
	if etag == "" {
		return false
	}
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		candidate = strings.TrimPrefix(candidate, "W/")
		candidate = strings.Trim(candidate, `"`)
		if candidate == etag || candidate == "*" {
			return true
		}
	}
	return false
}
