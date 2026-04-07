package handler

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

// validProfileBody returns a minimal valid ProfileRequestDTO body.
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

// ── CreateOrUpdate ────────────────────────────────────────────────────────────

func TestProfileHandler_CreateOrUpdate_BadJSON(t *testing.T) {
	r := badJSONReq(t, http.MethodPost, "/profiles")
	r = withTenantAndUser(r)
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).CreateOrUpdate(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_CreateOrUpdate_ValidationError(t *testing.T) {
	r := jsonReq(t, http.MethodPost, "/profiles", map[string]any{})
	r = withTenantAndUser(r)
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).CreateOrUpdate(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_CreateOrUpdate_WithBirthdate(t *testing.T) {
	bd := "2000-01-15"
	svc := &mockProfileService{
		createOrUpdateFn: func(u uuid.UUID, fn string, mn, ln, suf, dn, bio *string, bdate *time.Time, gen, ph, em, addr, city, co, tz, lang, purl *string, meta map[string]any) (*service.ProfileServiceDataResult, error) {
			return &service.ProfileServiceDataResult{FirstName: fn}, nil
		},
	}
	body := map[string]any{"first_name": "Alice", "birthdate": bd}
	r := jsonReq(t, http.MethodPost, "/profiles", body)
	r = withTenantAndUser(r)
	w := httptest.NewRecorder()
	NewProfileHandler(svc).CreateOrUpdate(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── CreateProfile ─────────────────────────────────────────────────────────────

func TestProfileHandler_CreateProfile_BadJSON(t *testing.T) {
	r := badJSONReq(t, http.MethodPost, "/profiles")
	r = withTenantAndUser(r)
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).CreateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_CreateProfile_ValidationError(t *testing.T) {
	r := jsonReq(t, http.MethodPost, "/profiles", map[string]any{})
	r = withTenantAndUser(r)
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).CreateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_CreateProfile_WithBirthdate(t *testing.T) {
	bd := "1990-06-15"
	svc := &mockProfileService{
		createOrUpdateSpecificFn: func(pUUID, uUUID uuid.UUID, fn string, mn, ln, suf, dn, bio *string, bdate *time.Time, gen, ph, em, addr, city, co, tz, lang, purl *string, meta map[string]any) (*service.ProfileServiceDataResult, error) {
			return &service.ProfileServiceDataResult{FirstName: fn}, nil
		},
	}
	body := map[string]any{"first_name": "Bob", "birthdate": bd}
	r := jsonReq(t, http.MethodPost, "/profiles", body)
	r = withTenantAndUser(r)
	w := httptest.NewRecorder()
	NewProfileHandler(svc).CreateProfile(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestProfileHandler_CreateProfile_ServiceError(t *testing.T) {
	svc := &mockProfileService{
		createOrUpdateSpecificFn: func(pUUID, uUUID uuid.UUID, fn string, mn, ln, suf, dn, bio *string, bdate *time.Time, gen, ph, em, addr, city, co, tz, lang, purl *string, meta map[string]any) (*service.ProfileServiceDataResult, error) {
			return nil, errors.New("create error")
		},
	}
	r := jsonReq(t, http.MethodPost, "/profiles", validProfileBody())
	r = withTenantAndUser(r)
	w := httptest.NewRecorder()
	NewProfileHandler(svc).CreateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── UpdateProfile ─────────────────────────────────────────────────────────────

func TestProfileHandler_UpdateProfile_BadJSON(t *testing.T) {
	profUUID := uuid.New()
	r := badJSONReq(t, http.MethodPut, "/")
	r = withTenantAndUser(r)
	r = withChiParam(r, "profile_uuid", profUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).UpdateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_UpdateProfile_ValidationError(t *testing.T) {
	profUUID := uuid.New()
	r := jsonReq(t, http.MethodPut, "/", map[string]any{})
	r = withTenantAndUser(r)
	r = withChiParam(r, "profile_uuid", profUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).UpdateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_UpdateProfile_WithBirthdate(t *testing.T) {
	profUUID := uuid.New()
	svc := &mockProfileService{
		createOrUpdateSpecificFn: func(pUUID, uUUID uuid.UUID, fn string, mn, ln, suf, dn, bio *string, bdate *time.Time, gen, ph, em, addr, city, co, tz, lang, purl *string, meta map[string]any) (*service.ProfileServiceDataResult, error) {
			return &service.ProfileServiceDataResult{FirstName: fn}, nil
		},
	}
	body := map[string]any{"first_name": "Carol", "birthdate": "1985-03-20"}
	r := jsonReq(t, http.MethodPut, "/", body)
	r = withTenantAndUser(r)
	r = withChiParam(r, "profile_uuid", profUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(svc).UpdateProfile(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProfileHandler_UpdateProfile_ServiceError(t *testing.T) {
	profUUID := uuid.New()
	svc := &mockProfileService{
		createOrUpdateSpecificFn: func(pUUID, uUUID uuid.UUID, fn string, mn, ln, suf, dn, bio *string, bdate *time.Time, gen, ph, em, addr, city, co, tz, lang, purl *string, meta map[string]any) (*service.ProfileServiceDataResult, error) {
			return nil, errors.New("update error")
		},
	}
	r := jsonReq(t, http.MethodPut, "/", validProfileBody())
	r = withTenantAndUser(r)
	r = withChiParam(r, "profile_uuid", profUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(svc).UpdateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── GetAll ────────────────────────────────────────────────────────────────────

func TestProfileHandler_GetAll_ValidationError(t *testing.T) {
	r := jsonReq(t, http.MethodGet, "/profiles?sort_order=bad", nil)
	r = withTenantAndUser(r)
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).GetAll(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_GetAll_IsDefaultTrue(t *testing.T) {
	svc := &mockProfileService{
		getAllFn: func(u uuid.UUID, fn, ln, em, ph, city, co *string, isD *bool, pg, lim int, sb, so string) (*service.ProfileServiceListResult, error) {
			return &service.ProfileServiceListResult{
				Data: []service.ProfileServiceDataResult{{FirstName: "Alice"}},
			}, nil
		},
	}
	r := jsonReq(t, http.MethodGet, "/profiles?is_default=true&page=1&limit=10", nil)
	r = withTenantAndUser(r)
	w := httptest.NewRecorder()
	NewProfileHandler(svc).GetAll(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProfileHandler_GetAll_IsDefaultFalse(t *testing.T) {
	r := jsonReq(t, http.MethodGet, "/profiles?is_default=false&page=1&limit=10", nil)
	r = withTenantAndUser(r)
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).GetAll(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestProfileHandler_Delete_ServiceError(t *testing.T) {
	profUUID := uuid.New()
	svc := &mockProfileService{
		getByUserUUIDFn: func(u uuid.UUID) (*service.ProfileServiceDataResult, error) {
			return &service.ProfileServiceDataResult{ProfileUUID: profUUID}, nil
		},
		deleteByUUIDFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
			return nil, errors.New("delete error")
		},
	}
	r := jsonReq(t, http.MethodDelete, "/profiles", nil)
	r = withTenantAndUser(r)
	w := httptest.NewRecorder()
	NewProfileHandler(svc).Delete(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── GetByUUID ─────────────────────────────────────────────────────────────────

func TestProfileHandler_GetByUUID_Forbidden(t *testing.T) {
	profUUID := uuid.New()
	svc := &mockProfileService{
		getByUUIDFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
			return nil, errors.New("profile does not belong to user")
		},
	}
	r := jsonReq(t, http.MethodGet, "/", nil)
	r = withTenantAndUser(r)
	r = withChiParam(r, "profile_uuid", profUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(svc).GetByUUID(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ── DeleteByUUID ──────────────────────────────────────────────────────────────

func TestProfileHandler_DeleteByUUID_InvalidUUID(t *testing.T) {
	r := jsonReq(t, http.MethodDelete, "/", nil)
	r = withTenantAndUser(r)
	r = withChiParam(r, "profile_uuid", "bad")
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).DeleteByUUID(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_DeleteByUUID_Forbidden(t *testing.T) {
	profUUID := uuid.New()
	svc := &mockProfileService{
		deleteByUUIDFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
			return nil, errors.New("profile does not belong to user")
		},
	}
	r := jsonReq(t, http.MethodDelete, "/", nil)
	r = withTenantAndUser(r)
	r = withChiParam(r, "profile_uuid", profUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(svc).DeleteByUUID(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestProfileHandler_DeleteByUUID_ServiceError(t *testing.T) {
	profUUID := uuid.New()
	svc := &mockProfileService{
		deleteByUUIDFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
			return nil, errors.New("generic error")
		},
	}
	r := jsonReq(t, http.MethodDelete, "/", nil)
	r = withTenantAndUser(r)
	r = withChiParam(r, "profile_uuid", profUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(svc).DeleteByUUID(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── SetDefaultProfile ─────────────────────────────────────────────────────────

func TestProfileHandler_SetDefaultProfile_ServiceError(t *testing.T) {
	profUUID := uuid.New()
	svc := &mockProfileService{
		setDefaultFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
			return nil, errors.New("set default error")
		},
	}
	r := jsonReq(t, http.MethodPost, "/", nil)
	r = withTenantAndUser(r)
	r = withChiParam(r, "profile_uuid", profUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(svc).SetDefaultProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── AdminGetAllProfiles ───────────────────────────────────────────────────────

func TestProfileHandler_AdminGetAllProfiles_ValidationError(t *testing.T) {
	userUUID := uuid.New()
	r := jsonReq(t, http.MethodGet, "/?sort_order=bad", nil)
	r = withChiParam(r, "user_uuid", userUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).AdminGetAllProfiles(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_AdminGetAllProfiles_ServiceError(t *testing.T) {
	userUUID := uuid.New()
	svc := &mockProfileService{
		getAllFn: func(u uuid.UUID, fn, ln, em, ph, city, co *string, isD *bool, pg, lim int, sb, so string) (*service.ProfileServiceListResult, error) {
			return nil, errors.New("db error")
		},
	}
	r := jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil)
	r = withChiParam(r, "user_uuid", userUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(svc).AdminGetAllProfiles(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestProfileHandler_AdminGetAllProfiles_IsDefaultTrue(t *testing.T) {
	userUUID := uuid.New()
	svc := &mockProfileService{
		getAllFn: func(u uuid.UUID, fn, ln, em, ph, city, co *string, isD *bool, pg, lim int, sb, so string) (*service.ProfileServiceListResult, error) {
			return &service.ProfileServiceListResult{
				Data: []service.ProfileServiceDataResult{{FirstName: "Alice"}},
			}, nil
		},
	}
	r := jsonReq(t, http.MethodGet, "/?is_default=true&page=1&limit=10", nil)
	r = withChiParam(r, "user_uuid", userUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(svc).AdminGetAllProfiles(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProfileHandler_AdminGetAllProfiles_IsDefaultFalse(t *testing.T) {
	userUUID := uuid.New()
	r := jsonReq(t, http.MethodGet, "/?is_default=false&page=1&limit=10", nil)
	r = withChiParam(r, "user_uuid", userUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).AdminGetAllProfiles(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── AdminGetProfile ───────────────────────────────────────────────────────────

func TestProfileHandler_AdminGetProfile_InvalidUserUUID(t *testing.T) {
	r := jsonReq(t, http.MethodGet, "/", nil)
	r = withChiParam(r, "user_uuid", "bad")
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).AdminGetProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_AdminGetProfile_InvalidProfileUUID(t *testing.T) {
	userUUID := uuid.New()
	r := jsonReq(t, http.MethodGet, "/", nil)
	r = withChiParam(r, "user_uuid", userUUID.String())
	r = withChiParam(r, "profile_uuid", "bad")
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).AdminGetProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_AdminGetProfile_NotFound(t *testing.T) {
	userUUID := uuid.New()
	profUUID := uuid.New()
	svc := &mockProfileService{
		getByUUIDFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
			return nil, errors.New("not found")
		},
	}
	r := jsonReq(t, http.MethodGet, "/", nil)
	r = withChiParam(r, "user_uuid", userUUID.String())
	r = withChiParam(r, "profile_uuid", profUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(svc).AdminGetProfile(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── AdminCreateProfile ────────────────────────────────────────────────────────

func TestProfileHandler_AdminCreateProfile_InvalidUserUUID(t *testing.T) {
	r := jsonReq(t, http.MethodPost, "/", validProfileBody())
	r = withChiParam(r, "user_uuid", "bad")
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).AdminCreateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_AdminCreateProfile_BadJSON(t *testing.T) {
	userUUID := uuid.New()
	r := badJSONReq(t, http.MethodPost, "/")
	r = withChiParam(r, "user_uuid", userUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).AdminCreateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_AdminCreateProfile_ValidationError(t *testing.T) {
	userUUID := uuid.New()
	r := jsonReq(t, http.MethodPost, "/", map[string]any{})
	r = withChiParam(r, "user_uuid", userUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).AdminCreateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_AdminCreateProfile_WithBirthdate(t *testing.T) {
	userUUID := uuid.New()
	svc := &mockProfileService{
		createOrUpdateSpecificFn: func(pUUID, uUUID uuid.UUID, fn string, mn, ln, suf, dn, bio *string, bdate *time.Time, gen, ph, em, addr, city, co, tz, lang, purl *string, meta map[string]any) (*service.ProfileServiceDataResult, error) {
			return &service.ProfileServiceDataResult{FirstName: fn}, nil
		},
	}
	body := map[string]any{"first_name": "Dave", "birthdate": "1995-08-10"}
	r := jsonReq(t, http.MethodPost, "/", body)
	r = withChiParam(r, "user_uuid", userUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(svc).AdminCreateProfile(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestProfileHandler_AdminCreateProfile_ServiceError(t *testing.T) {
	userUUID := uuid.New()
	svc := &mockProfileService{
		createOrUpdateSpecificFn: func(pUUID, uUUID uuid.UUID, fn string, mn, ln, suf, dn, bio *string, bdate *time.Time, gen, ph, em, addr, city, co, tz, lang, purl *string, meta map[string]any) (*service.ProfileServiceDataResult, error) {
			return nil, errors.New("create error")
		},
	}
	r := jsonReq(t, http.MethodPost, "/", validProfileBody())
	r = withChiParam(r, "user_uuid", userUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(svc).AdminCreateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── AdminUpdateProfile ────────────────────────────────────────────────────────

func TestProfileHandler_AdminUpdateProfile_InvalidUserUUID(t *testing.T) {
	r := jsonReq(t, http.MethodPut, "/", validProfileBody())
	r = withChiParam(r, "user_uuid", "bad")
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).AdminUpdateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_AdminUpdateProfile_InvalidProfileUUID(t *testing.T) {
	userUUID := uuid.New()
	r := jsonReq(t, http.MethodPut, "/", validProfileBody())
	r = withChiParam(r, "user_uuid", userUUID.String())
	r = withChiParam(r, "profile_uuid", "bad")
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).AdminUpdateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_AdminUpdateProfile_BadJSON(t *testing.T) {
	userUUID := uuid.New()
	profUUID := uuid.New()
	r := badJSONReq(t, http.MethodPut, "/")
	r = withChiParam(r, "user_uuid", userUUID.String())
	r = withChiParam(r, "profile_uuid", profUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).AdminUpdateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_AdminUpdateProfile_ValidationError(t *testing.T) {
	userUUID := uuid.New()
	profUUID := uuid.New()
	r := jsonReq(t, http.MethodPut, "/", map[string]any{})
	r = withChiParam(r, "user_uuid", userUUID.String())
	r = withChiParam(r, "profile_uuid", profUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).AdminUpdateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_AdminUpdateProfile_WithBirthdate(t *testing.T) {
	userUUID := uuid.New()
	profUUID := uuid.New()
	svc := &mockProfileService{
		createOrUpdateSpecificFn: func(pUUID, uUUID uuid.UUID, fn string, mn, ln, suf, dn, bio *string, bdate *time.Time, gen, ph, em, addr, city, co, tz, lang, purl *string, meta map[string]any) (*service.ProfileServiceDataResult, error) {
			return &service.ProfileServiceDataResult{FirstName: fn}, nil
		},
	}
	body := map[string]any{"first_name": "Eve", "birthdate": "1988-11-30"}
	r := jsonReq(t, http.MethodPut, "/", body)
	r = withChiParam(r, "user_uuid", userUUID.String())
	r = withChiParam(r, "profile_uuid", profUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(svc).AdminUpdateProfile(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProfileHandler_AdminUpdateProfile_ServiceError(t *testing.T) {
	userUUID := uuid.New()
	profUUID := uuid.New()
	svc := &mockProfileService{
		createOrUpdateSpecificFn: func(pUUID, uUUID uuid.UUID, fn string, mn, ln, suf, dn, bio *string, bdate *time.Time, gen, ph, em, addr, city, co, tz, lang, purl *string, meta map[string]any) (*service.ProfileServiceDataResult, error) {
			return nil, errors.New("update error")
		},
	}
	r := jsonReq(t, http.MethodPut, "/", validProfileBody())
	r = withChiParam(r, "user_uuid", userUUID.String())
	r = withChiParam(r, "profile_uuid", profUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(svc).AdminUpdateProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── AdminDeleteProfile ────────────────────────────────────────────────────────

func TestProfileHandler_AdminDeleteProfile_InvalidUserUUID(t *testing.T) {
	r := jsonReq(t, http.MethodDelete, "/", nil)
	r = withChiParam(r, "user_uuid", "bad")
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).AdminDeleteProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_AdminDeleteProfile_InvalidProfileUUID(t *testing.T) {
	userUUID := uuid.New()
	r := jsonReq(t, http.MethodDelete, "/", nil)
	r = withChiParam(r, "user_uuid", userUUID.String())
	r = withChiParam(r, "profile_uuid", "bad")
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).AdminDeleteProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_AdminDeleteProfile_ServiceError(t *testing.T) {
	userUUID := uuid.New()
	profUUID := uuid.New()
	svc := &mockProfileService{
		deleteByUUIDFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
			return nil, errors.New("delete error")
		},
	}
	r := jsonReq(t, http.MethodDelete, "/", nil)
	r = withChiParam(r, "user_uuid", userUUID.String())
	r = withChiParam(r, "profile_uuid", profUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(svc).AdminDeleteProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── AdminSetDefaultProfile ────────────────────────────────────────────────────

func TestProfileHandler_AdminSetDefaultProfile_InvalidUserUUID(t *testing.T) {
	r := jsonReq(t, http.MethodPost, "/", nil)
	r = withChiParam(r, "user_uuid", "bad")
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).AdminSetDefaultProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_AdminSetDefaultProfile_InvalidProfileUUID(t *testing.T) {
	userUUID := uuid.New()
	r := jsonReq(t, http.MethodPost, "/", nil)
	r = withChiParam(r, "user_uuid", userUUID.String())
	r = withChiParam(r, "profile_uuid", "bad")
	w := httptest.NewRecorder()
	NewProfileHandler(&mockProfileService{}).AdminSetDefaultProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_AdminSetDefaultProfile_ServiceError(t *testing.T) {
	userUUID := uuid.New()
	profUUID := uuid.New()
	svc := &mockProfileService{
		setDefaultFn: func(pUUID, uUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
			return nil, errors.New("set default error")
		},
	}
	r := jsonReq(t, http.MethodPost, "/", nil)
	r = withChiParam(r, "user_uuid", userUUID.String())
	r = withChiParam(r, "profile_uuid", profUUID.String())
	w := httptest.NewRecorder()
	NewProfileHandler(svc).AdminSetDefaultProfile(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
