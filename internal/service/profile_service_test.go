package service

import (
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newProfileSvc(profileRepo *mockProfileRepo, userRepo *mockUserRepo) ProfileService {
	return NewProfileService(nil, profileRepo, userRepo)
}

// ---------------------------------------------------------------------------
// ProfileService.CreateOrUpdateProfile – transactional
// ---------------------------------------------------------------------------

func TestProfileService_CreateOrUpdateProfile(t *testing.T) {
	userUUID := uuid.New()
	userID := int64(42)

	t.Run("user not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		})
		_, err := svc.CreateOrUpdateProfile(userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("FindDefaultByUserID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{
			findDefaultByUserIDFn: func(_ int64) (*model.Profile, error) {
				return nil, errors.New("db error")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		_, err := svc.CreateOrUpdateProfile(userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("create new profile → Create error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{
			findDefaultByUserIDFn: func(_ int64) (*model.Profile, error) { return nil, nil },
			createFn: func(_ *model.Profile) (*model.Profile, error) {
				return nil, errors.New("create failed")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		_, err := svc.CreateOrUpdateProfile(userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create failed")
	})

	t.Run("create new profile → UpdateByUUID user error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{
			findDefaultByUserIDFn: func(_ int64) (*model.Profile, error) { return nil, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.User, error) {
				return nil, errors.New("user update failed")
			},
		})
		_, err := svc.CreateOrUpdateProfile(userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user update failed")
	})

	t.Run("create new profile success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewProfileService(db, &mockProfileRepo{
			findDefaultByUserIDFn: func(_ int64) (*model.Profile, error) { return nil, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		res, err := svc.CreateOrUpdateProfile(userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "John", res.FirstName)
		assert.True(t, res.IsDefault)
	})

	t.Run("create with metadata", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		meta := map[string]any{"key": "value"}
		svc := NewProfileService(db, &mockProfileRepo{
			findDefaultByUserIDFn: func(_ int64) (*model.Profile, error) { return nil, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		res, err := svc.CreateOrUpdateProfile(userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, meta)
		require.NoError(t, err)
		assert.Equal(t, "value", res.Metadata["key"])
	})

	t.Run("metadata marshal error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		badMeta := map[string]any{"bad": math.Inf(1)}
		svc := NewProfileService(db, &mockProfileRepo{
			findDefaultByUserIDFn: func(_ int64) (*model.Profile, error) { return nil, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		_, err := svc.CreateOrUpdateProfile(userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, badMeta)
		require.Error(t, err)
	})

	t.Run("update existing → UpdateByUserID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		existing := &model.Profile{ProfileID: 10, ProfileUUID: uuid.New(), UserID: userID, IsDefault: true}
		svc := NewProfileService(db, &mockProfileRepo{
			findDefaultByUserIDFn: func(_ int64) (*model.Profile, error) { return existing, nil },
			updateByUserIDFn: func(_ int64, _ *model.Profile) error {
				return errors.New("update failed")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		_, err := svc.CreateOrUpdateProfile(userUUID, "Jane", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update failed")
	})

	t.Run("update existing success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		existing := &model.Profile{ProfileID: 10, ProfileUUID: uuid.New(), UserID: userID, IsDefault: true}
		svc := NewProfileService(db, &mockProfileRepo{
			findDefaultByUserIDFn: func(_ int64) (*model.Profile, error) { return existing, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		res, err := svc.CreateOrUpdateProfile(userUUID, "Jane", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "Jane", res.FirstName)
	})
}

// ---------------------------------------------------------------------------
// ProfileService.CreateOrUpdateSpecificProfile – transactional
// ---------------------------------------------------------------------------

func TestProfileService_CreateOrUpdateSpecificProfile(t *testing.T) {
	profileUUID := uuid.New()
	userUUID := uuid.New()
	userID := int64(42)

	t.Run("user not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		})
		_, err := svc.CreateOrUpdateSpecificProfile(profileUUID, userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("FindByUUID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) {
				return nil, errors.New("db error")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		_, err := svc.CreateOrUpdateSpecificProfile(profileUUID, userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("new profile → FindByUserID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{
			findByUserIDFn: func(_ int64) (*model.Profile, error) {
				return nil, errors.New("find user prof err")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		_, err := svc.CreateOrUpdateSpecificProfile(profileUUID, userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find user prof err")
	})

	t.Run("new profile first profile → Create error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{
			createFn: func(_ *model.Profile) (*model.Profile, error) {
				return nil, errors.New("create failed")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		_, err := svc.CreateOrUpdateSpecificProfile(profileUUID, userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create failed")
	})

	t.Run("new profile first → UpdateByUUID user error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.User, error) {
				return nil, errors.New("user update failed")
			},
		})
		_, err := svc.CreateOrUpdateSpecificProfile(profileUUID, userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user update failed")
	})

	t.Run("new profile first → success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewProfileService(db, &mockProfileRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		res, err := svc.CreateOrUpdateSpecificProfile(profileUUID, userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "John", res.FirstName)
		assert.True(t, res.IsDefault)
	})

	t.Run("new profile not first → success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewProfileService(db, &mockProfileRepo{
			findByUserIDFn: func(_ int64) (*model.Profile, error) {
				return &model.Profile{ProfileID: 99}, nil // existing profile exists
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		res, err := svc.CreateOrUpdateSpecificProfile(profileUUID, userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		assert.False(t, res.IsDefault)
	})

	t.Run("new profile with metadata → success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		meta := map[string]any{"role": "admin"}
		svc := NewProfileService(db, &mockProfileRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		res, err := svc.CreateOrUpdateSpecificProfile(profileUUID, userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, meta)
		require.NoError(t, err)
		assert.Equal(t, "admin", res.Metadata["role"])
	})

	t.Run("metadata marshal error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		badMeta := map[string]any{"bad": math.Inf(1)}
		svc := NewProfileService(db, &mockProfileRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		_, err := svc.CreateOrUpdateSpecificProfile(profileUUID, userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, badMeta)
		require.Error(t, err)
	})

	t.Run("existing profile belongs to different user → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) {
				return &model.Profile{ProfileID: 10, ProfileUUID: profileUUID, UserID: 999}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		_, err := svc.CreateOrUpdateSpecificProfile(profileUUID, userUUID, "John", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not belong")
	})

	t.Run("update existing → CreateOrUpdate error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		existing := &model.Profile{ProfileID: 10, ProfileUUID: profileUUID, UserID: userID}
		svc := NewProfileService(db, &mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) { return existing, nil },
			createOrUpdateFn: func(_ *model.Profile) (*model.Profile, error) {
				return nil, errors.New("update failed")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		_, err := svc.CreateOrUpdateSpecificProfile(profileUUID, userUUID, "Jane", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update failed")
	})

	t.Run("update existing → success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		existing := &model.Profile{ProfileID: 10, ProfileUUID: profileUUID, UserID: userID}
		svc := NewProfileService(db, &mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) { return existing, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		res, err := svc.CreateOrUpdateSpecificProfile(profileUUID, userUUID, "Jane", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "Jane", res.FirstName)
	})
}

func TestProfileService_GetByUUID(t *testing.T) {
	profileUUID := uuid.New()
	userUUID := uuid.New()
	userID := int64(42)

	t.Run("user not found", func(t *testing.T) {
		svc := newProfileSvc(&mockProfileRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		})
		_, err := svc.GetByUUID(profileUUID, userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("profile not found", func(t *testing.T) {
		svc := newProfileSvc(&mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) { return nil, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID}, nil
			},
		})
		_, err := svc.GetByUUID(profileUUID, userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "profile not found")
	})

	t.Run("profile belongs to different user", func(t *testing.T) {
		svc := newProfileSvc(&mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) {
				return &model.Profile{ProfileUUID: profileUUID, UserID: 999}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID}, nil
			},
		})
		_, err := svc.GetByUUID(profileUUID, userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not belong")
	})

	t.Run("success", func(t *testing.T) {
		svc := newProfileSvc(&mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) {
				return &model.Profile{ProfileUUID: profileUUID, UserID: userID}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID}, nil
			},
		})
		res, err := svc.GetByUUID(profileUUID, userUUID)
		require.NoError(t, err)
		assert.Equal(t, profileUUID, res.ProfileUUID)
	})
}

func TestProfileService_GetByUserUUID(t *testing.T) {
	userUUID := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		svc := newProfileSvc(&mockProfileRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		})
		_, err := svc.GetByUserUUID(userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("profile not found", func(t *testing.T) {
		svc := newProfileSvc(&mockProfileRepo{
			findDefaultByUserIDFn: func(_ int64) (*model.Profile, error) { return nil, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		_, err := svc.GetByUserUUID(userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "profile not found")
	})

	t.Run("success", func(t *testing.T) {
		pid := uuid.New()
		svc := newProfileSvc(&mockProfileRepo{
			findDefaultByUserIDFn: func(_ int64) (*model.Profile, error) {
				return &model.Profile{ProfileUUID: pid, IsDefault: true}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		res, err := svc.GetByUserUUID(userUUID)
		require.NoError(t, err)
		assert.Equal(t, pid, res.ProfileUUID)
	})
}

func TestProfileService_GetAll(t *testing.T) {
	userUUID := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		svc := newProfileSvc(&mockProfileRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, errors.New("db") },
		})
		_, err := svc.GetAll(userUUID, nil, nil, nil, nil, nil, nil, nil, 1, 10, "created_at", "asc")
		require.Error(t, err)
	})

	t.Run("FindAllByUserID error", func(t *testing.T) {
		svc := newProfileSvc(&mockProfileRepo{
			findAllByUserIDFn: func(_ repository.ProfileRepositoryGetFilter) (*repository.PaginationResult[model.Profile], error) {
				return nil, errors.New("repo error")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		_, err := svc.GetAll(userUUID, nil, nil, nil, nil, nil, nil, nil, 1, 10, "created_at", "asc")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo error")
	})

	t.Run("success with results", func(t *testing.T) {
		svc := newProfileSvc(&mockProfileRepo{
			findAllByUserIDFn: func(_ repository.ProfileRepositoryGetFilter) (*repository.PaginationResult[model.Profile], error) {
				return &repository.PaginationResult[model.Profile]{
					Data:  []model.Profile{{ProfileUUID: uuid.New()}},
					Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		res, err := svc.GetAll(userUUID, nil, nil, nil, nil, nil, nil, nil, 1, 10, "created_at", "asc")
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
	})
}

func TestProfileService_DeleteByUUID(t *testing.T) {
	profileUUID := uuid.New()
	userUUID := uuid.New()
	userID := int64(1)

	t.Run("user not found", func(t *testing.T) {
		svc := newProfileSvc(&mockProfileRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		})
		_, err := svc.DeleteByUUID(profileUUID, userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("profile not found", func(t *testing.T) {
		svc := newProfileSvc(&mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) { return nil, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID}, nil
			},
		})
		_, err := svc.DeleteByUUID(profileUUID, userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "profile not found")
	})

	t.Run("profile belongs to different user", func(t *testing.T) {
		svc := newProfileSvc(&mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) {
				return &model.Profile{UserID: 999}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID}, nil
			},
		})
		_, err := svc.DeleteByUUID(profileUUID, userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not belong")
	})

	t.Run("cannot delete default profile", func(t *testing.T) {
		svc := newProfileSvc(&mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) {
				return &model.Profile{UserID: userID, IsDefault: true}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID}, nil
			},
		})
		_, err := svc.DeleteByUUID(profileUUID, userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete default")
	})

	t.Run("DeleteByUUID repo error", func(t *testing.T) {
		svc := newProfileSvc(&mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) {
				return &model.Profile{ProfileUUID: profileUUID, UserID: userID, IsDefault: false}, nil
			},
			deleteByUUIDFn: func(_ any) error { return errors.New("delete failed") },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID}, nil
			},
		})
		_, err := svc.DeleteByUUID(profileUUID, userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})

	t.Run("success", func(t *testing.T) {
		svc := newProfileSvc(&mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) {
				return &model.Profile{ProfileUUID: profileUUID, UserID: userID, IsDefault: false}, nil
			},
			deleteByUUIDFn: func(_ any) error { return nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID}, nil
			},
		})
		res, err := svc.DeleteByUUID(profileUUID, userUUID)
		require.NoError(t, err)
		assert.Equal(t, profileUUID, res.ProfileUUID)
	})
}

func TestProfileService_SetDefaultProfile(t *testing.T) {
	profileUUID := uuid.New()
	userUUID := uuid.New()
	userID := int64(1)

	t.Run("user not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		})
		_, err := svc.SetDefaultProfile(profileUUID, userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("profile not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) { return nil, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID}, nil
			},
		})
		_, err := svc.SetDefaultProfile(profileUUID, userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "profile not found")
	})

	t.Run("FindByUUID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) {
				return nil, errors.New("find error")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID}, nil
			},
		})
		_, err := svc.SetDefaultProfile(profileUUID, userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find error")
	})

	t.Run("profile belongs to different user → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) {
				return &model.Profile{ProfileUUID: profileUUID, UserID: 999}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID}, nil
			},
		})
		_, err := svc.SetDefaultProfile(profileUUID, userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not belong")
	})

	t.Run("UnsetDefaultProfiles error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) {
				return &model.Profile{ProfileUUID: profileUUID, UserID: userID}, nil
			},
			unsetDefaultFn: func(_ int64) error { return errors.New("unset error") },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID}, nil
			},
		})
		_, err := svc.SetDefaultProfile(profileUUID, userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unset error")
	})

	t.Run("CreateOrUpdate error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewProfileService(db, &mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) {
				return &model.Profile{ProfileUUID: profileUUID, UserID: userID}, nil
			},
			createOrUpdateFn: func(_ *model.Profile) (*model.Profile, error) {
				return nil, errors.New("save error")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID}, nil
			},
		})
		_, err := svc.SetDefaultProfile(profileUUID, userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save error")
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewProfileService(db, &mockProfileRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Profile, error) {
				return &model.Profile{ProfileUUID: profileUUID, UserID: userID}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID}, nil
			},
		})
		res, err := svc.SetDefaultProfile(profileUUID, userUUID)
		require.NoError(t, err)
		assert.True(t, res.IsDefault)
	})
}

// ---------------------------------------------------------------------------
// toProfileServiceDataResult
// ---------------------------------------------------------------------------

func TestToProfileServiceDataResult(t *testing.T) {
	t.Run("nil input → nil", func(t *testing.T) {
		assert.Nil(t, toProfileServiceDataResult(nil))
	})

	t.Run("invalid metadata JSON → metadata is nil", func(t *testing.T) {
		p := &model.Profile{
			ProfileUUID: uuid.New(),
			FirstName:   "John",
			Metadata:    []byte("not-json"),
		}
		res := toProfileServiceDataResult(p)
		require.NotNil(t, res)
		assert.Nil(t, res.Metadata)
		assert.Equal(t, "John", res.FirstName)
	})

	t.Run("valid metadata JSON", func(t *testing.T) {
		p := &model.Profile{
			ProfileUUID: uuid.New(),
			FirstName:   "Jane",
			Metadata:    []byte(`{"key":"val"}`),
		}
		res := toProfileServiceDataResult(p)
		require.NotNil(t, res)
		assert.Equal(t, "val", res.Metadata["key"])
	})
}
