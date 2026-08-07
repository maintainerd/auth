package user

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image"
	"net/url"
	"strings"

	// Registered for their decoders only: DecodeConfig below establishes what an
	// upload actually IS, rather than believing its Content-Type or extension.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"golang.org/x/image/webp"
	"gorm.io/gorm"
)

// MaxProfilePictureBytes caps an uploaded avatar at 2 MiB. Mirrored by
// chk_profile_pictures_size, so the limit holds for writers that never touch
// this service.
const MaxProfilePictureBytes = 2 << 20

// maxPictureDimension bounds the declared canvas. A header can claim an
// enormous image in very few bytes; nothing needs to render one this large.
const maxPictureDimension = 8192

// allowedPictureTypes are the raster formats an avatar may be stored as.
//
// SVG is deliberately absent. It is an XML document that can carry <script> and
// external references, so serving one from this origin would be stored XSS
// against everyone who views that avatar — and unlike the raster formats there
// is no decode step that can establish it is "just an image".
var allowedPictureTypes = map[string]string{
	"png":  "image/png",
	"jpeg": "image/jpeg",
	"webp": "image/webp",
	"gif":  "image/gif",
}

// DecodeProfilePicture validates an uploaded avatar and reports its real type.
//
// The type is established by DECODING the bytes, never from the multipart
// Content-Type or the filename: both are attacker-chosen, and the value is
// echoed back on the serve path, so trusting either would let an uploader pick
// how a browser interprets their bytes.
func DecodeProfilePicture(data []byte) (*ProfilePicture, error) {
	if len(data) == 0 {
		return nil, apperror.NewValidation("the uploaded file is empty")
	}
	if len(data) > MaxProfilePictureBytes {
		return nil, apperror.NewValidation(fmt.Sprintf(
			"the image is larger than the %d MiB limit", MaxProfilePictureBytes>>20))
	}

	// DecodeConfig reads the header only, so a decompression bomb is refused on
	// its declared dimensions rather than by being expanded first.
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		// WebP is not in the standard library's registry; try it explicitly
		// before concluding the bytes are not an image.
		if wcfg, werr := webp.DecodeConfig(bytes.NewReader(data)); werr == nil {
			cfg, format = wcfg, "webp"
		} else {
			return nil, apperror.NewValidation("the file is not a readable PNG, JPEG, WebP or GIF image")
		}
	}

	contentType, ok := allowedPictureTypes[format]
	if !ok {
		return nil, apperror.NewValidation(fmt.Sprintf(
			"%s images are not supported; use PNG, JPEG, WebP or GIF", format))
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, apperror.NewValidation("the image has no dimensions")
	}
	if cfg.Width > maxPictureDimension || cfg.Height > maxPictureDimension {
		return nil, apperror.NewValidation(fmt.Sprintf(
			"the image is larger than %dx%d pixels", maxPictureDimension, maxPictureDimension))
	}

	// A content hash, so the serve endpoint can answer a repeat view with 304
	// instead of re-reading the image. Content-derived rather than a timestamp
	// means re-uploading the same file does not invalidate a cached copy.
	sum := sha256.Sum256(data)
	return &ProfilePicture{
		Data:        data,
		ContentType: contentType,
		ETag:        base64.RawURLEncoding.EncodeToString(sum[:16]),
	}, nil
}

// ValidateProfileURL bounds an externally hosted avatar URL.
//
// https only and no embedded credentials: the value is rendered as an <img> src
// for anyone who views the profile, so http is mixed content and a user:pass@
// link leaks a credential into markup and referrer headers. Rejecting
// non-http(s) schemes also keeps javascript: and data: out of that attribute.
func ValidateProfileURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if len(value) > 2048 {
		return "", apperror.NewValidation("the image URL is too long")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return "", apperror.NewValidation("the image URL is not a valid absolute URL")
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return "", apperror.NewValidation("the image URL must use https")
	}
	if parsed.User != nil {
		return "", apperror.NewValidation("the image URL must not contain credentials")
	}
	return value, nil
}

// SetProfilePicture stores an uploaded avatar and points the profile's URL at
// it.
//
// profile_url is the durable contract — a profile always simply has a URL — so
// an upload sets it to this service's serve path. When object storage arrives,
// this method writes there and sets the returned URL instead; the profile
// model, the API response and every client are untouched.
func (s *profileService) SetProfilePicture(ctx context.Context, profileUUID uuid.UUID, userID int64, picture *ProfilePicture) (string, error) {
	profile, err := s.requireOwnedProfile(profileUUID, userID)
	if err != nil {
		return "", err
	}
	if s.profilePictureRepo == nil {
		return "", apperror.NewInternal("profile picture storage is not configured", nil)
	}

	servePath := ProfilePictureURL(profileUUID)
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.profilePictureRepo.WithTx(tx).Upsert(&ProfilePictureRecord{
			ProfileID:   profile.ProfileID,
			Data:        picture.Data,
			ContentType: picture.ContentType,
			ETag:        picture.ETag,
		}); err != nil {
			return err
		}
		// Same transaction: a stored image the profile does not point at is
		// invisible, and a URL with no image behind it is a broken avatar.
		return tx.Model(&Profile{}).
			Where("profile_id = ?", profile.ProfileID).
			Update("profile_url", servePath).Error
	})
	if err != nil {
		return "", apperror.NewInternal("could not save the profile picture", err)
	}
	return servePath, nil
}

// ClearProfilePicture removes an uploaded avatar and the URL pointing at it.
func (s *profileService) ClearProfilePicture(ctx context.Context, profileUUID uuid.UUID, userID int64) error {
	profile, err := s.requireOwnedProfile(profileUUID, userID)
	if err != nil {
		return err
	}
	if s.profilePictureRepo == nil {
		return apperror.NewInternal("profile picture storage is not configured", nil)
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.profilePictureRepo.WithTx(tx).Delete(profile.ProfileID); err != nil {
			return err
		}
		// Only clear the URL if it pointed at the image just deleted. A user who
		// set an external link after uploading should keep that link.
		return tx.Model(&Profile{}).
			Where("profile_id = ? AND profile_url = ?", profile.ProfileID, ProfilePictureURL(profileUUID)).
			Update("profile_url", nil).Error
	})
	if err != nil {
		return apperror.NewInternal("could not remove the profile picture", err)
	}
	return nil
}

// EnsureProfileOwnedBy reports an error unless the profile belongs to userID.
//
// Separate from requireOwnedProfile so the read path can ask the question
// without loading a picture it may not be allowed to see.
func (s *profileService) EnsureProfileOwnedBy(ctx context.Context, profileUUID uuid.UUID, userID int64) error {
	_, err := s.requireOwnedProfile(profileUUID, userID)
	return err
}

// GetProfilePicture returns the stored avatar for serving.
func (s *profileService) GetProfilePicture(ctx context.Context, profileUUID uuid.UUID) (*ProfilePicture, error) {
	record, err := s.loadPictureRecord(profileUUID, false)
	if err != nil {
		return nil, err
	}
	return &ProfilePicture{Data: record.Data, ContentType: record.ContentType, ETag: record.ETag}, nil
}

// GetProfilePictureETag returns the validator WITHOUT reading the image, so a
// conditional request costs a small row read instead of up to 2 MiB.
func (s *profileService) GetProfilePictureETag(ctx context.Context, profileUUID uuid.UUID) (string, error) {
	record, err := s.loadPictureRecord(profileUUID, true)
	if err != nil {
		return "", err
	}
	return record.ETag, nil
}

func (s *profileService) loadPictureRecord(profileUUID uuid.UUID, metaOnly bool) (*ProfilePictureRecord, error) {
	if s.profilePictureRepo == nil {
		return nil, apperror.NewNotFound("profile picture")
	}
	profile, err := s.profileRepo.FindByUUID(profileUUID)
	if err != nil {
		return nil, apperror.NewInternal("could not load the profile", err)
	}
	if profile == nil {
		return nil, apperror.NewNotFound("profile")
	}

	var record *ProfilePictureRecord
	if metaOnly {
		record, err = s.profilePictureRepo.FindMetaByProfileID(profile.ProfileID)
	} else {
		record, err = s.profilePictureRepo.FindByProfileID(profile.ProfileID)
	}
	if err != nil {
		return nil, apperror.NewInternal("could not load the profile picture", err)
	}
	if record == nil {
		return nil, apperror.NewNotFound("profile picture")
	}
	return record, nil
}

// requireOwnedProfile loads a profile and refuses one belonging to somebody
// else, so a UUID alone is not authority to change another user's picture.
func (s *profileService) requireOwnedProfile(profileUUID uuid.UUID, userID int64) (*Profile, error) {
	profile, err := s.profileRepo.FindByUUID(profileUUID)
	if err != nil {
		return nil, apperror.NewInternal("could not load the profile", err)
	}
	if profile == nil {
		return nil, apperror.NewNotFound("profile")
	}
	if profile.UserID != userID {
		// Not-found rather than forbidden: a caller probing UUIDs learns nothing
		// about which ones exist.
		return nil, apperror.NewNotFound("profile")
	}
	return profile, nil
}

// ProfilePictureURL is the path an uploaded avatar is served from. Kept in one
// place so the stored profile_url, the router and the API response cannot
// disagree.
func ProfilePictureURL(profileUUID uuid.UUID) string {
	return "/api/v1/profiles/" + profileUUID.String() + "/picture"
}
