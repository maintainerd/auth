package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
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
			findByUUIDFn: func(_ any, _ ...string) (*model.UserSetting, error) { return nil, nil },
		}, &mockUserRepo{})
		_, err := svc.GetByUUID(id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("success", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.UserSetting, error) {
				return &model.UserSetting{UserSettingUUID: id}, nil
			},
		}, &mockUserRepo{})
		res, err := svc.GetByUUID(id)
		require.NoError(t, err)
		assert.Equal(t, id, res.UserSettingUUID)
	})
}

func TestUserSettingService_GetByUserUUID(t *testing.T) {
	userUUID := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		})
		_, err := svc.GetByUserUUID(userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("setting not found", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*model.UserSetting, error) { return nil, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		_, err := svc.GetByUserUUID(userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("success", func(t *testing.T) {
		sid := uuid.New()
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*model.UserSetting, error) {
				return &model.UserSetting{UserSettingUUID: sid}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		res, err := svc.GetByUserUUID(userUUID)
		require.NoError(t, err)
		assert.Equal(t, sid, res.UserSettingUUID)
	})
}

func TestUserSettingService_DeleteByUUID(t *testing.T) {
	id := uuid.New()

	t.Run("not found", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.UserSetting, error) { return nil, nil },
		}, &mockUserRepo{})
		_, err := svc.DeleteByUUID(id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("success", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.UserSetting, error) {
				return &model.UserSetting{UserSettingUUID: id}, nil
			},
			deleteByUUIDFn: func(_ any) error { return nil },
		}, &mockUserRepo{})
		res, err := svc.DeleteByUUID(id)
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
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		})
		_, err := svc.CreateOrUpdateUserSetting(userUUID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("new setting → create → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		sid := uuid.New()
		svc := NewUserSettingService(db, &mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*model.UserSetting, error) { return nil, nil },
			createFn: func(e *model.UserSetting) (*model.UserSetting, error) {
				e.UserSettingUUID = sid
				return e, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		res, err := svc.CreateOrUpdateUserSetting(userUUID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, sid, res.UserSettingUUID)
	})
}

