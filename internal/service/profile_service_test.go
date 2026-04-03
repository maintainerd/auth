package service

import (
	"errors"
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
