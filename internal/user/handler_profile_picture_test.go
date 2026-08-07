package user

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pictureRequest(t *testing.T, profileUUID uuid.UUID, body *bytes.Buffer, contentType string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/"+profileUUID.String()+"/picture", body)
	req.Header.Set("Content-Type", contentType)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("profile_uuid", profileUUID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = middleware.WithAuthContextValue(ctx, &authctx.AuthContext{
		User: &authctx.AuthUser{UserID: 7, UserUUID: uuid.New()},
	})
	return req.WithContext(ctx)
}

func multipartPNG(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", "avatar.png")
	require.NoError(t, err)
	_, err = part.Write(encodePNG(t, 32, 32))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return &buf, w.FormDataContentType()
}

func TestUploadPicture(t *testing.T) {
	profileUUID := uuid.New()

	t.Run("accepts a well-formed multipart upload", func(t *testing.T) {
		svc := &mockProfileService{
			setProfilePictureFn: func(u uuid.UUID, userID int64, p *ProfilePicture) (string, error) {
				assert.Equal(t, profileUUID, u)
				assert.Equal(t, int64(7), userID)
				assert.Equal(t, "image/png", p.ContentType)
				return ProfilePictureURL(u), nil
			},
		}
		body, contentType := multipartPNG(t)
		rec := httptest.NewRecorder()

		NewProfileHandler(svc).UploadPicture(rec, pictureRequest(t, profileUUID, body, contentType))

		assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Body.String(), "/picture")
	})

	// REGRESSION. The axios instance defaults every request to
	// application/json, and axios only computes a multipart boundary when that
	// header is absent — so uploads went out as JSON-labelled multipart bodies
	// the server could not parse.
	//
	// The handler then reported EVERY parse failure as 413, so the symptom was
	// "the image is larger than the 2 MiB limit" for a 40 KB file. This asserts
	// the honest status, because the misleading one cost real debugging time.
	t.Run("a body labelled application/json is a bad request, not too large", func(t *testing.T) {
		body, _ := multipartPNG(t)
		rec := httptest.NewRecorder()

		NewProfileHandler(&mockProfileService{}).UploadPicture(rec, pictureRequest(t, profileUUID, body, "application/json"))

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.NotContains(t, rec.Body.String(), "larger than",
			"a malformed body must not be reported as an oversized image")
		assert.Contains(t, strings.ToLower(rec.Body.String()), "multipart")
	})

	t.Run("an oversized body is reported as too large", func(t *testing.T) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		part, err := w.CreateFormFile("file", "big.png")
		require.NoError(t, err)
		_, err = part.Write(make([]byte, MaxProfilePictureBytes+4096))
		require.NoError(t, err)
		require.NoError(t, w.Close())

		rec := httptest.NewRecorder()
		NewProfileHandler(&mockProfileService{}).UploadPicture(rec, pictureRequest(t, profileUUID, &buf, w.FormDataContentType()))

		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	})

	t.Run("a multipart body with no file field is refused", func(t *testing.T) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		require.NoError(t, w.WriteField("notafile", "x"))
		require.NoError(t, w.Close())

		rec := httptest.NewRecorder()
		NewProfileHandler(&mockProfileService{}).UploadPicture(rec, pictureRequest(t, profileUUID, &buf, w.FormDataContentType()))

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// The route permission is account:profile:read:self, but a UUID in the path can
// name anyone's profile. Without a per-resource check the endpoint served every
// avatar to every authenticated caller — not what that permission says.
func TestGetPictureAccessScope(t *testing.T) {
	profileUUID := uuid.New()

	getRequest := func(user *authctx.AuthUser) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+profileUUID.String()+"/picture", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("profile_uuid", profileUUID.String())
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		ctx = middleware.WithAuthContextValue(ctx, &authctx.AuthContext{User: user})
		return req.WithContext(ctx)
	}

	// Someone else's profile, and the caller does not administer users.
	notOwner := &mockProfileService{
		ensureProfileOwnedByFn: func(uuid.UUID, int64) error { return apperror.NewNotFound("profile") },
		getProfilePictureFn: func(uuid.UUID) (*ProfilePicture, error) {
			t.Fatal("the picture must not be read for a caller who may not see it")
			return nil, nil
		},
	}

	t.Run("a stranger cannot fetch someone else's avatar", func(t *testing.T) {
		rec := httptest.NewRecorder()
		NewProfileHandler(notOwner).GetPicture(rec, getRequest(&authctx.AuthUser{UserID: 9}))
		// Not-found, not forbidden: probing UUIDs must not reveal which exist.
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	// The admin console legitimately renders other people's avatars.
	t.Run("a user administrator may fetch any avatar", func(t *testing.T) {
		served := &mockProfileService{
			ensureProfileOwnedByFn: func(uuid.UUID, int64) error { return apperror.NewNotFound("profile") },
			getProfilePictureFn: func(uuid.UUID) (*ProfilePicture, error) {
				return &ProfilePicture{Data: []byte{1, 2, 3}, ContentType: "image/png", ETag: "abc"}, nil
			},
		}
		rec := httptest.NewRecorder()
		NewProfileHandler(served).GetPicture(rec, getRequest(&authctx.AuthUser{
			UserID: 9,
			// Permissions hang off roles, which is how the middleware resolves them.
			Roles: []authctx.AuthRole{{
				Name:        "user-admin",
				Permissions: []authctx.AuthPermission{{Name: "user:read"}},
			}},
		}))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	})

	t.Run("the owner may fetch their own avatar", func(t *testing.T) {
		owned := &mockProfileService{
			ensureProfileOwnedByFn: func(uuid.UUID, int64) error { return nil },
			getProfilePictureFn: func(uuid.UUID) (*ProfilePicture, error) {
				return &ProfilePicture{Data: []byte{1}, ContentType: "image/png", ETag: "abc"}, nil
			},
		}
		rec := httptest.NewRecorder()
		NewProfileHandler(owned).GetPicture(rec, getRequest(&authctx.AuthUser{UserID: 7}))
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
