package user

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUserSettingSvc(settingRepo *mockUserSettingRepo, userRepo *mockUserRepo) UserSettingService {
	return NewUserSettingService(nil, settingRepo, userRepo)
}

func TestUserSettingService_GetByUUID(t *testing.T) {
	id := uuid.New()

	t.Run("not found", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserSetting, error) { return nil, nil },
		}, &mockUserRepo{})
		_, err := svc.GetByUUID(context.Background(), id, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("success", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserSetting, error) {
				return &UserSetting{UserSettingUUID: id, UserID: 1}, nil
			},
		}, &mockUserRepo{})
		res, err := svc.GetByUUID(context.Background(), id, 1)
		require.NoError(t, err)
		assert.Equal(t, id, res.UserSettingUUID)
	})
}

func TestUserSettingService_GetByUserUUID(t *testing.T) {
	userUUID := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		})
		_, err := svc.GetByUserUUID(context.Background(), userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("setting not found", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*UserSetting, error) { return nil, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: 1}, nil
			},
		})
		_, err := svc.GetByUserUUID(context.Background(), userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("success", func(t *testing.T) {
		sid := uuid.New()
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*UserSetting, error) {
				return &UserSetting{UserSettingUUID: sid, UserID: 1}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: 1}, nil
			},
		})
		res, err := svc.GetByUserUUID(context.Background(), userUUID)
		require.NoError(t, err)
		assert.Equal(t, sid, res.UserSettingUUID)
	})
}

func TestUserSettingService_DeleteByUUID(t *testing.T) {
	id := uuid.New()

	t.Run("not found", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserSetting, error) { return nil, nil },
		}, &mockUserRepo{})
		_, err := svc.DeleteByUUID(context.Background(), id, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("delete error", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserSetting, error) {
				return &UserSetting{UserSettingUUID: id, UserID: 1}, nil
			},
			deleteByUUIDFn: func(_ any) error { return errors.New("delete failed") },
		}, &mockUserRepo{})
		_, err := svc.DeleteByUUID(context.Background(), id, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})

	t.Run("success", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserSetting, error) {
				return &UserSetting{UserSettingUUID: id, UserID: 1}, nil
			},
			deleteByUUIDFn: func(_ any) error { return nil },
		}, &mockUserRepo{})
		res, err := svc.DeleteByUUID(context.Background(), id, 1)
		require.NoError(t, err)
		assert.Equal(t, id, res.UserSettingUUID)
	})
}

func TestUserSettingService_CreateOrUpdateUserSetting(t *testing.T) {
	userUUID := uuid.New()

	t.Run("user not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewUserSettingService(db, &mockUserSettingRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		})
		_, err := svc.CreateOrUpdateUserSetting(context.Background(), userUUID, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("new setting → create → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		sid := uuid.New()
		svc := NewUserSettingService(db, &mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*UserSetting, error) { return nil, nil },
			createFn: func(e *UserSetting) (*UserSetting, error) {
				e.UserSettingUUID = sid
				return e, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: 1}, nil
			},
		})
		res, err := svc.CreateOrUpdateUserSetting(context.Background(), userUUID, nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, sid, res.UserSettingUUID)
	})

	t.Run("FindByUserID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewUserSettingService(db, &mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*UserSetting, error) {
				return nil, errors.New("db error")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: 1}, nil
			},
		})
		_, err := svc.CreateOrUpdateUserSetting(context.Background(), userUUID, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("new setting → Create error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewUserSettingService(db, &mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*UserSetting, error) { return nil, nil },
			createFn: func(_ *UserSetting) (*UserSetting, error) {
				return nil, errors.New("create failed")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: 1}, nil
			},
		})
		_, err := svc.CreateOrUpdateUserSetting(context.Background(), userUUID, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create failed")
	})

	t.Run("update existing → success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		sid := uuid.New()
		tz := "America/New_York"
		svc := NewUserSettingService(db, &mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*UserSetting, error) {
				return &UserSetting{UserSettingID: 10, UserSettingUUID: sid, UserID: 1}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: 1}, nil
			},
		})
		res, err := svc.CreateOrUpdateUserSetting(context.Background(), userUUID, &tz, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, sid, res.UserSettingUUID)
		assert.Equal(t, &tz, res.Timezone)
	})

	t.Run("update existing → UpdateByUserID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewUserSettingService(db, &mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*UserSetting, error) {
				return &UserSetting{UserSettingID: 10, UserSettingUUID: uuid.New(), UserID: 1}, nil
			},
			updateByUserIDFn: func(_ int64, _ *UserSetting) error {
				return errors.New("update failed")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: 1}, nil
			},
		})
		_, err := svc.CreateOrUpdateUserSetting(context.Background(), userUUID, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update failed")
	})

	t.Run("preferredLanguage fallback when locale is nil", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		tz := "UTC"
		lang := "en"
		svc := NewUserSettingService(db, &mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*UserSetting, error) { return nil, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: 1}, nil
			},
		})
		res, err := svc.CreateOrUpdateUserSetting(context.Background(), userUUID, &tz, &lang, nil)
		require.NoError(t, err)
		assert.Equal(t, &lang, res.Locale)
		assert.Equal(t, &lang, res.PreferredLanguage)
	})

	t.Run("create with timezone, language and locale", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		tz := "UTC"
		lang := "en"
		locale := "en-US"
		svc := NewUserSettingService(db, &mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*UserSetting, error) { return nil, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: 1}, nil
			},
		})
		res, err := svc.CreateOrUpdateUserSetting(context.Background(), userUUID, &tz, &lang, &locale)
		require.NoError(t, err)
		assert.Equal(t, &tz, res.Timezone)
		assert.Equal(t, locale, *res.PreferredLanguage)
		assert.Equal(t, &locale, res.Locale)
	})
}

// ---------------------------------------------------------------------------
// toUserSettingServiceDataResult
// ---------------------------------------------------------------------------

func TestToUserSettingServiceDataResult(t *testing.T) {
	t.Run("nil input → nil", func(t *testing.T) {
		assert.Nil(t, toUserSettingServiceDataResult(nil))
	})

	t.Run("full fields", func(t *testing.T) {
		sid := uuid.New()
		tz := "UTC"
		us := &UserSetting{
			UserSettingUUID: sid,
			Timezone:        &tz,
		}
		res := toUserSettingServiceDataResult(us)
		require.NotNil(t, res)
		assert.Equal(t, sid, res.UserSettingUUID)
		assert.Equal(t, &tz, res.Timezone)
	})
}
