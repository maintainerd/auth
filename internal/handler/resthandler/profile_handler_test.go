package resthandler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

// validProfileBody returns a minimal valid ProfileRequestDto body.
func validProfileBody() map[string]any {
	return map[string]any{"first_name": "Alice"}
}

func TestProfileHandler_CreateOrUpdate(t *testing.T) {
	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockProfileService{
			createOrUpdateFn: func(u uuid.UUID, fn string, mn, ln, suf, dn, bio *string, bd *time.Time, gen, ph, em, addr, city, co, tz, lang, purl *string, meta map[string]any) (*service.ProfileServiceDataResult, error) {
				return nil, errors.New("save error")
			},
		}
		r := jsonReq(t, http.MethodPost, "/profiles", validProfileBody())
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewProfileHandler(svc).CreateOrUpdate(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockProfileService{
			createOrUpdateFn: func(u uuid.UUID, fn string, mn, ln, suf, dn, bio *string, bd *time.Time, gen, ph, em, addr, city, co, tz, lang, purl *string, meta map[string]any) (*service.ProfileServiceDataResult, error) {
				return &service.ProfileServiceDataResult{FirstName: fn}, nil
			},
		}
		r := jsonReq(t, http.MethodPost, "/profiles", validProfileBody())
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewProfileHandler(svc).CreateOrUpdate(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestProfileHandler_CreateProfile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockProfileService{
			createOrUpdateSpecificFn: func(pUUID, uUUID uuid.UUID, fn string, mn, ln, suf, dn, bio *string, bd *time.Time, gen, ph, em, addr, city, co, tz, lang, purl *string, meta map[string]any) (*service.ProfileServiceDataResult, error) {
				return &service.ProfileServiceDataResult{FirstName: fn}, nil
			},
		}
		r := jsonReq(t, http.MethodPost, "/profiles", validProfileBody())
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewProfileHandler(svc).CreateProfile(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestProfileHandler_UpdateProfile(t *testing.T) {
	profUUID := uuid.New()

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", validProfileBody())
		r = withTenantAndUser(r)
		r = withChiParam(r, "profile_uuid", "bad")
		w := httptest.NewRecorder()
		NewProfileHandler(&mockProfileService{}).UpdateProfile(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockProfileService{
			createOrUpdateSpecificFn: func(pUUID, uUUID uuid.UUID, fn string, mn, ln, suf, dn, bio *string, bd *time.Time, gen, ph, em, addr, city, co, tz, lang, purl *string, meta map[string]any) (*service.ProfileServiceDataResult, error) {
				return &service.ProfileServiceDataResult{FirstName: fn}, nil
			},
		}
		r := jsonReq(t, http.MethodPut, "/", validProfileBody())
		r = withTenantAndUser(r)
		r = withChiParam(r, "profile_uuid", profUUID.String())
		w := httptest.NewRecorder()
		NewProfileHandler(svc).UpdateProfile(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestProfileHandler_Get(t *testing.T) {
	t.Run("profile not found returns 404", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/profiles", nil)
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewProfileHandler(&mockProfileService{}).Get(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockProfileService{
			getByUserUUIDFn: func(u uuid.UUID) (*service.ProfileServiceDataResult, error) {
				return &service.ProfileServiceDataResult{FirstName: "Alice"}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/profiles", nil)
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewProfileHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestProfileHandler_GetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/profiles?page=1&limit=10", nil)
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewProfileHandler(&mockProfileService{}).GetAll(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockProfileService{
			getAllFn: func(u uuid.UUID, fn, ln, em, ph, city, co *string, isD *bool, pg, lim int, sb, so string) (*service.ProfileServiceListResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodGet, "/profiles?page=1&limit=10", nil)
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewProfileHandler(svc).GetAll(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestProfileHandler_Delete(t *testing.T) {
	t.Run("profile not found returns 404", func(t *testing.T) {
		// mock default GetByUserUUID returns nil, nil → 404
		r := jsonReq(t, http.MethodDelete, "/profiles", nil)
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewProfileHandler(&mockProfileService{}).Delete(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		profUUID := uuid.New()
		svc := &mockProfileService{
			getByUserUUIDFn: func(u uuid.UUID) (*service.ProfileServiceDataResult, error) {
				return &service.ProfileServiceDataResult{ProfileUUID: profUUID}, nil
			},
			deleteByUUIDFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
				return &service.ProfileServiceDataResult{ProfileUUID: pUUID}, nil
			},
		}
		r := jsonReq(t, http.MethodDelete, "/profiles", nil)
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewProfileHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestProfileHandler_GetByUUID(t *testing.T) {
	profUUID := uuid.New()

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "profile_uuid", "bad")
		w := httptest.NewRecorder()
		NewProfileHandler(&mockProfileService{}).GetByUUID(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		svc := &mockProfileService{
			getByUUIDFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
				return nil, errors.New("not found")
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "profile_uuid", profUUID.String())
		w := httptest.NewRecorder()
		NewProfileHandler(svc).GetByUUID(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockProfileService{
			getByUUIDFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
				return &service.ProfileServiceDataResult{ProfileUUID: pUUID}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "profile_uuid", profUUID.String())
		w := httptest.NewRecorder()
		NewProfileHandler(svc).GetByUUID(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestProfileHandler_DeleteByUUID(t *testing.T) {
	profUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockProfileService{
			deleteByUUIDFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
				return &service.ProfileServiceDataResult{ProfileUUID: pUUID}, nil
			},
		}
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "profile_uuid", profUUID.String())
		w := httptest.NewRecorder()
		NewProfileHandler(svc).DeleteByUUID(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestProfileHandler_SetDefaultProfile(t *testing.T) {
	profUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockProfileService{
			setDefaultFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
				return &service.ProfileServiceDataResult{ProfileUUID: pUUID}, nil
			},
		}
		r := jsonReq(t, http.MethodPost, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "profile_uuid", profUUID.String())
		w := httptest.NewRecorder()
		NewProfileHandler(svc).SetDefaultProfile(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "profile_uuid", "bad")
		w := httptest.NewRecorder()
		NewProfileHandler(&mockProfileService{}).SetDefaultProfile(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestProfileHandler_AdminGetAllProfiles(t *testing.T) {
	userUUID := uuid.New()

	t.Run("invalid user uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withChiParam(r, "user_uuid", "bad")
		w := httptest.NewRecorder()
		NewProfileHandler(&mockProfileService{}).AdminGetAllProfiles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil)
		r = withChiParam(r, "user_uuid", userUUID.String())
		w := httptest.NewRecorder()
		NewProfileHandler(&mockProfileService{}).AdminGetAllProfiles(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestProfileHandler_AdminGetProfile(t *testing.T) {
	userUUID := uuid.New()
	profUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockProfileService{
			getByUUIDFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
				return &service.ProfileServiceDataResult{ProfileUUID: pUUID}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withChiParam(r, "user_uuid", userUUID.String())
		r = withChiParam(r, "profile_uuid", profUUID.String())
		w := httptest.NewRecorder()
		NewProfileHandler(svc).AdminGetProfile(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestProfileHandler_AdminCreateProfile(t *testing.T) {
	userUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockProfileService{
			createOrUpdateSpecificFn: func(pUUID, uUUID uuid.UUID, fn string, mn, ln, suf, dn, bio *string, bd *time.Time, gen, ph, em, addr, city, co, tz, lang, purl *string, meta map[string]any) (*service.ProfileServiceDataResult, error) {
				return &service.ProfileServiceDataResult{FirstName: fn}, nil
			},
		}
		r := jsonReq(t, http.MethodPost, "/", validProfileBody())
		r = withChiParam(r, "user_uuid", userUUID.String())
		w := httptest.NewRecorder()
		NewProfileHandler(svc).AdminCreateProfile(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestProfileHandler_AdminUpdateProfile(t *testing.T) {
	userUUID := uuid.New()
	profUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockProfileService{
			createOrUpdateSpecificFn: func(pUUID, uUUID uuid.UUID, fn string, mn, ln, suf, dn, bio *string, bd *time.Time, gen, ph, em, addr, city, co, tz, lang, purl *string, meta map[string]any) (*service.ProfileServiceDataResult, error) {
				return &service.ProfileServiceDataResult{FirstName: fn}, nil
			},
		}
		r := jsonReq(t, http.MethodPut, "/", validProfileBody())
		r = withChiParam(r, "user_uuid", userUUID.String())
		r = withChiParam(r, "profile_uuid", profUUID.String())
		w := httptest.NewRecorder()
		NewProfileHandler(svc).AdminUpdateProfile(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestProfileHandler_AdminDeleteProfile(t *testing.T) {
	userUUID := uuid.New()
	profUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockProfileService{
			deleteByUUIDFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
				return &service.ProfileServiceDataResult{ProfileUUID: pUUID}, nil
			},
		}
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withChiParam(r, "user_uuid", userUUID.String())
		r = withChiParam(r, "profile_uuid", profUUID.String())
		w := httptest.NewRecorder()
		NewProfileHandler(svc).AdminDeleteProfile(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestProfileHandler_AdminSetDefaultProfile(t *testing.T) {
	userUUID := uuid.New()
	profUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockProfileService{
			setDefaultFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
				return &service.ProfileServiceDataResult{ProfileUUID: pUUID}, nil
			},
		}
		r := jsonReq(t, http.MethodPost, "/", nil)
		r = withChiParam(r, "user_uuid", userUUID.String())
		r = withChiParam(r, "profile_uuid", profUUID.String())
		w := httptest.NewRecorder()
		NewProfileHandler(svc).AdminSetDefaultProfile(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
